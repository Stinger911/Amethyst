// This file implements PUT /api/notes/{path...} from plan_amethyst-mvp
// Фаза 3 (write path) and plan_amethyst-web-ui §6 step 5 (editor save).
// Saves are atomic (temp file + rename, see vault.WriteAtomic) and use
// optimistic concurrency: the client echoes back the hash it loaded
// (NoteDetail.Hash); if the file on disk has since changed, the save is
// redirected into a Syncthing-style conflict copy instead of silently
// overwriting someone else's edit (research_amethyst-filewatcher-sync §5).
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/vault"
	"github.com/Stinger911/Amethyst/internal/watch"
)

// WriteConfig holds what the write-path handlers need beyond the index:
// the vault root, for reading/writing real files, and the watcher, so a
// save can suppress the redundant reindex its own write would otherwise
// trigger (see Watcher.Suppress).
type WriteConfig struct {
	VaultRoot string
	Watcher   *watch.Watcher
}

// SaveNoteRequest is the JSON body of PUT /api/notes/{path...}.
type SaveNoteRequest struct {
	Content string `json:"content"`
	// BaseHash is the NoteDetail.Hash the client last loaded. Empty means
	// "I believe this note doesn't exist yet" (creating a new note).
	BaseHash string `json:"baseHash"`
}

// SaveNoteResponse is the JSON body of a successful save.
type SaveNoteResponse struct {
	Hash string `json:"hash"`
}

// ConflictResponse is the JSON body of a 409: the client's content was
// preserved at ConflictPath instead of overwriting a newer on-disk version.
type ConflictResponse struct {
	Error        string `json:"error"`
	ConflictPath string `json:"conflictPath"`
}

// SaveNoteHandler serves PUT /api/notes/{path...}.
func SaveNoteHandler(db *index.DB, write WriteConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := r.PathValue("path")
		if err := validateNotePath(relPath); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req SaveNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		currentHash, exists, err := currentFileHash(write.VaultRoot, relPath)
		if err != nil {
			log.Printf("save note %q: stat current: %v", relPath, err)
			http.Error(w, "save note failed", http.StatusInternalServerError)
			return
		}

		// A mismatch covers both "changed since I loaded it" and "deleted
		// since I loaded it" (exists=false but the client thought it did).
		if (exists && currentHash != req.BaseHash) || (!exists && req.BaseHash != "") {
			cPath := conflictPath(relPath, time.Now())
			if _, err := writeAndIndex(db, write, cPath, []byte(req.Content)); err != nil {
				log.Printf("save note %q: write conflict copy: %v", relPath, err)
				http.Error(w, "save note failed", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusConflict, ConflictResponse{Error: "conflict", ConflictPath: cPath})
			return
		}

		hash, err := writeAndIndex(db, write, relPath, []byte(req.Content))
		if err != nil {
			log.Printf("save note %q: %v", relPath, err)
			http.Error(w, "save note failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, SaveNoteResponse{Hash: hash})
	}
}

// validateNotePath rejects anything that isn't a plain, vault-relative
// .md path: no escaping the vault root, no absolute paths.
func validateNotePath(relPath string) error {
	if relPath == "" {
		return errors.New("missing note path")
	}
	if !strings.HasSuffix(relPath, ".md") {
		return errors.New("only .md notes can be saved")
	}
	if strings.HasPrefix(relPath, "/") {
		return errors.New("invalid note path")
	}
	for _, seg := range strings.Split(relPath, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return errors.New("invalid note path")
		}
	}
	return nil
}

// conflictPath builds a Syncthing-style conflict copy name, e.g.
// "Folder/Note.md" -> "Folder/Note.sync-conflict-20260622-153045-web.md",
// per research_amethyst-filewatcher-sync §5.
func conflictPath(original string, at time.Time) string {
	ext := path.Ext(original)
	base := strings.TrimSuffix(original, ext)
	return fmt.Sprintf("%s.sync-conflict-%s-web%s", base, at.UTC().Format("20060102-150405"), ext)
}

// currentFileHash reports relPath's current sha256, and whether it exists
// at all (a missing file is not an error here — it just means "new note").
func currentFileHash(vaultRoot, relPath string) (hash string, exists bool, err error) {
	content, err := os.ReadFile(path.Join(vaultRoot, relPath))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), true, nil
}

// writeAndIndex atomically writes content to relPath, then reindexes it
// synchronously so the response's hash and the index agree immediately —
// the caller doesn't have to wait out the watcher's debounce/stability
// windows to see its own write reflected. The watcher is told to skip its
// own pass over this path (Watcher.Suppress) since this call already did
// the reindex.
func writeAndIndex(db *index.DB, write WriteConfig, relPath string, content []byte) (hash string, err error) {
	fullPath := path.Join(write.VaultRoot, relPath)
	if write.Watcher != nil {
		write.Watcher.Suppress(relPath)
	}
	if err := vault.WriteAtomic(fullPath, content); err != nil {
		return "", err
	}
	if err := index.ReindexFile(db, write.VaultRoot, relPath); err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
