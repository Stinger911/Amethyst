package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"

	"github.com/Stinger911/Amethyst/internal/vault"
)

// Stats summarizes a completed ColdScan, for logging.
type Stats struct {
	Files int
	Notes int
	Links int
	Tags  int
}

// ColdScan walks vaultRoot from scratch and replaces the index contents
// with what it finds. It's idempotent: safe to call again on an existing
// index (e.g. a manual rescan), not just on first run.
func ColdScan(db *DB, vaultRoot string) (Stats, error) {
	entries, err := vault.Walk(vaultRoot)
	if err != nil {
		return Stats{}, fmt.Errorf("walk vault: %w", err)
	}

	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	resolver := newTargetResolver(paths)

	tx, err := db.Begin()
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()

	for _, table := range []string{"tags", "links", "notes_fts", "notes", "files"} {
		if _, err := tx.Exec(`DELETE FROM ` + table); err != nil {
			return Stats{}, fmt.Errorf("clear %s: %w", table, err)
		}
	}

	var stats Stats
	for _, entry := range entries {
		fullPath := path.Join(vaultRoot, entry.Path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return Stats{}, fmt.Errorf("read %s: %w", entry.Path, err)
		}
		hash := sha256.Sum256(content)

		if _, err := tx.Exec(
			`INSERT INTO files(path, kind, mtime, size, content_hash) VALUES (?, ?, ?, ?, ?)`,
			entry.Path, string(entry.Kind), entry.ModTime.Unix(), entry.Size, hex.EncodeToString(hash[:]),
		); err != nil {
			return Stats{}, fmt.Errorf("insert file %s: %w", entry.Path, err)
		}
		stats.Files++

		if entry.Kind != vault.KindNote {
			continue
		}

		links, tags, err := indexNoteTx(tx, resolver, entry.Path, content)
		if err != nil {
			return Stats{}, fmt.Errorf("index note %s: %w", entry.Path, err)
		}
		stats.Notes++
		stats.Links += links
		stats.Tags += tags
	}

	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}
	return stats, nil
}
