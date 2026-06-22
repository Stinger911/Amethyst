// Package writepath holds the write-then-reindex sequence shared by every
// caller that writes into the vault on the server's own initiative: the
// web editor's PUT /api/notes/{path...} (internal/api/notes_write.go) and
// the Telegram bot's capture flow (internal/bot). Both need the same
// atomic write + watcher self-suppression + synchronous reindex, so it
// lives here once instead of being duplicated per caller.
package writepath

import (
	"crypto/sha256"
	"encoding/hex"
	"path"

	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/vault"
	"github.com/Stinger911/Amethyst/internal/watch"
)

// WriteAndIndex atomically writes content to relPath (under vaultRoot),
// then reindexes it synchronously so callers see their own write
// reflected immediately, without waiting out the watcher's debounce and
// stability windows. If watcher is non-nil, its own pass over this path is
// suppressed (Watcher.Suppress) since this call already did the reindex.
// relPath's parent directory must already exist.
func WriteAndIndex(db *index.DB, vaultRoot string, watcher *watch.Watcher, relPath string, content []byte) (hash string, err error) {
	fullPath := path.Join(vaultRoot, relPath)
	if watcher != nil {
		watcher.Suppress(relPath)
	}
	if err := vault.WriteAtomic(fullPath, content); err != nil {
		return "", err
	}
	if err := index.ReindexFile(db, vaultRoot, relPath); err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
