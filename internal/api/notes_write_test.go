package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

// doPut performs an authenticated PUT with a JSON body, exercising the
// same RequireAuth gate as doGet.
func doPut(t *testing.T, db *index.DB, vaultRoot, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{VaultRoot: vaultRoot}, nil, nil, nil).ServeHTTP(rec, req)
	return rec
}

func TestSaveNote_SuccessOverwritesAndReindexes(t *testing.T) {
	db, root := seedNotesIndex(t)

	getRec := doGet(t, db, root, "/api/notes/Leaf.md")
	var before NoteDetail
	if err := json.Unmarshal(getRec.Body.Bytes(), &before); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	putRec := doPut(t, db, root, "/api/notes/Leaf.md", SaveNoteRequest{
		Content:  "# Leaf\n\nEdited via the web editor.\n",
		BaseHash: before.Hash,
	})
	if putRec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	var resp SaveNoteResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Hash == before.Hash {
		t.Errorf("Hash unchanged after edit, want a new hash")
	}

	onDisk, err := os.ReadFile(filepath.Join(root, "Leaf.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(onDisk) != "# Leaf\n\nEdited via the web editor.\n" {
		t.Errorf("on-disk content = %q, want edited text", onDisk)
	}

	// Synchronous reindex means a GET right after the PUT already reflects
	// the edit, without waiting on the watcher's debounce window.
	afterRec := doGet(t, db, root, "/api/notes/Leaf.md")
	var after NoteDetail
	if err := json.Unmarshal(afterRec.Body.Bytes(), &after); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if after.Hash != resp.Hash {
		t.Errorf("GET hash = %q, want PUT response hash %q", after.Hash, resp.Hash)
	}
}

func TestSaveNote_StaleBaseHashWritesConflictCopy(t *testing.T) {
	db, root := seedNotesIndex(t)

	putRec := doPut(t, db, root, "/api/notes/Leaf.md", SaveNoteRequest{
		Content:  "My conflicting edit.\n",
		BaseHash: "not-the-real-hash",
	})
	if putRec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	var resp ConflictResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != "conflict" {
		t.Errorf("Error = %q, want %q", resp.Error, "conflict")
	}

	original, err := os.ReadFile(filepath.Join(root, "Leaf.md"))
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(original) == "My conflicting edit.\n" {
		t.Errorf("original file was overwritten, want it left untouched")
	}

	conflictContent, err := os.ReadFile(filepath.Join(root, resp.ConflictPath))
	if err != nil {
		t.Fatalf("read conflict copy %q: %v", resp.ConflictPath, err)
	}
	if string(conflictContent) != "My conflicting edit.\n" {
		t.Errorf("conflict copy content = %q, want %q", conflictContent, "My conflicting edit.\n")
	}
}

func TestSaveNote_CreatesNewNoteInExistingFolder(t *testing.T) {
	db, root := seedNotesIndex(t)

	putRec := doPut(t, db, root, "/api/notes/New%20Note.md", SaveNoteRequest{
		Content:  "# Brand new\n",
		BaseHash: "",
	})
	if putRec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	onDisk, err := os.ReadFile(filepath.Join(root, "New Note.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(onDisk) != "# Brand new\n" {
		t.Errorf("on-disk content = %q, want %q", onDisk, "# Brand new\n")
	}

	getRec := doGet(t, db, root, "/api/notes/New%20Note.md")
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET after create: status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
}

func TestSaveNote_DeletedSinceLoadIsAConflict(t *testing.T) {
	db, root := seedNotesIndex(t)
	if err := os.Remove(filepath.Join(root, "Leaf.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	putRec := doPut(t, db, root, "/api/notes/Leaf.md", SaveNoteRequest{
		Content:  "I didn't know it was deleted.\n",
		BaseHash: "some-hash-the-client-had",
	})
	if putRec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", putRec.Code, putRec.Body.String())
	}
}

func TestSaveNote_RejectsPathTraversal(t *testing.T) {
	db, root := seedNotesIndex(t)

	rec := doPut(t, db, root, "/api/notes/..%2F..%2Fescaped.md", SaveNoteRequest{Content: "x"})
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s, want 400 or 404 for an escaping path", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escaped.md")); err == nil {
		t.Errorf("path traversal wrote a file outside the vault root")
	}
}

func TestSaveNote_RejectsNonMarkdownPath(t *testing.T) {
	db, root := seedNotesIndex(t)

	rec := doPut(t, db, root, "/api/notes/Leaf.txt", SaveNoteRequest{Content: "not markdown"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400 for a non-.md path", rec.Code, rec.Body.String())
	}
}
