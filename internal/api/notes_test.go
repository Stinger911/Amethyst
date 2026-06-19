package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

func seedNotesIndex(t *testing.T) *index.DB {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	write("Hub.md", "---\ntags: [intro]\n---\n\nSee [[Leaf]] and [[Nowhere]].\n")
	write("Leaf.md", "# Leaf\n\nA leaf note with a #standalone tag.\n")

	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := index.ColdScan(db, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}
	return db
}

// doGet performs an authenticated GET: every content route is gated on a
// session (see RequireAuth), and that gate is not what these tests exercise.
func doGet(t *testing.T, db *index.DB, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db).ServeHTTP(rec, req)
	return rec
}

func TestNotesList_ReturnsAllNotesWithTags(t *testing.T) {
	db := seedNotesIndex(t)
	rec := doGet(t, db, "/api/notes")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp NotesListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, rec.Body.String())
	}
	if len(resp.Notes) != 2 {
		t.Fatalf("Notes = %+v, want 2", resp.Notes)
	}

	byPath := map[string]NoteSummary{}
	for _, n := range resp.Notes {
		byPath[n.Path] = n
	}
	if got := byPath["Hub.md"].Tags; len(got) != 1 || got[0] != "intro" {
		t.Errorf("Hub.md tags = %v, want [intro]", got)
	}
	if got := byPath["Leaf.md"].Tags; len(got) != 1 || got[0] != "standalone" {
		t.Errorf("Leaf.md tags = %v, want [standalone]", got)
	}
}

func TestNote_RendersHTMLWithResolvedAndMissingLinks(t *testing.T) {
	db := seedNotesIndex(t)
	rec := doGet(t, db, "/api/notes/Hub.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var detail NoteDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, rec.Body.String())
	}

	if detail.Title != "Hub" {
		t.Errorf("Title = %q, want %q", detail.Title, "Hub")
	}
	if !strings.Contains(detail.HTML, `href="/note/Leaf.md"`) {
		t.Errorf("HTML = %q, want resolved link to Leaf.md", detail.HTML)
	}
	if !strings.Contains(detail.HTML, "wikilink-missing") {
		t.Errorf("HTML = %q, want a missing-link span for [[Nowhere]]", detail.HTML)
	}
	if len(detail.Tags) != 1 || detail.Tags[0] != "intro" {
		t.Errorf("Tags = %v, want [intro]", detail.Tags)
	}
	if detail.Frontmatter["tags"] == nil {
		t.Errorf("Frontmatter = %v, want a tags key", detail.Frontmatter)
	}
}

func TestNote_BacklinksListsLinkingNotes(t *testing.T) {
	db := seedNotesIndex(t)
	rec := doGet(t, db, "/api/notes/Leaf.md")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var detail NoteDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, rec.Body.String())
	}
	if len(detail.Backlinks) != 1 || detail.Backlinks[0].Path != "Hub.md" {
		t.Errorf("Backlinks = %+v, want [Hub.md]", detail.Backlinks)
	}
}

func TestNote_UnknownPathReturns404(t *testing.T) {
	db := seedNotesIndex(t)
	rec := doGet(t, db, "/api/notes/DoesNotExist.md")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
