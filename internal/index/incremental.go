package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/Stinger911/Amethyst/internal/vault"
)

// ReindexFile (re-)indexes a single vault-relative path, for use by a
// file watcher reacting to one stabilized change instead of rescanning
// the whole vault. If the path no longer exists, it delegates to
// RemoveFile rather than erroring.
func ReindexFile(db *DB, vaultRoot, relPath string) error {
	fullPath := path.Join(vaultRoot, relPath)

	info, err := os.Stat(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		return RemoveFile(db, relPath)
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", relPath, err)
	}
	if info.IsDir() {
		return nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", relPath, err)
	}
	hash := sha256.Sum256(content)
	kind := vault.ClassifyKind(relPath)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO files(path, kind, mtime, size, content_hash) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET kind=excluded.kind, mtime=excluded.mtime, size=excluded.size, content_hash=excluded.content_hash`,
		relPath, string(kind), info.ModTime().Unix(), info.Size(), hex.EncodeToString(hash[:]),
	); err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	if kind != vault.KindNote {
		// The path may have previously been a note (e.g. renamed away from
		// .md) — drop any stale note-derived rows for it either way.
		if err := deleteIndexedNote(tx, relPath); err != nil {
			return err
		}
		return tx.Commit()
	}

	paths, err := allFilePaths(tx)
	if err != nil {
		return fmt.Errorf("list paths: %w", err)
	}
	resolver := newTargetResolver(paths)

	if _, _, err := indexNoteTx(tx, resolver, relPath, content); err != nil {
		return fmt.Errorf("index note %s: %w", relPath, err)
	}

	return tx.Commit()
}

// RemoveFile drops relPath from the index, along with anything indexed
// under it as a path prefix (covers a removed directory, since the
// watcher may only learn that the directory itself is gone, not which
// files were inside it).
func RemoveFile(db *DB, relPath string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := removePathAndPrefix(tx, relPath); err != nil {
		return err
	}
	return tx.Commit()
}

func removePathAndPrefix(tx *sql.Tx, relPath string) error {
	prefix := escapeLike(relPath) + "/%"
	exact := relPath

	rows, err := tx.Query(
		`SELECT id, path FROM notes WHERE path = ? OR path LIKE ? ESCAPE '\'`,
		exact, prefix,
	)
	if err != nil {
		return fmt.Errorf("find notes under %s: %w", relPath, err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var p string
		if err := rows.Scan(&id, &p); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM notes_fts WHERE rowid = ?`, id); err != nil {
			return err
		}
	}

	for _, stmt := range []string{
		`DELETE FROM notes WHERE path = ? OR path LIKE ? ESCAPE '\'`,
		`DELETE FROM links WHERE source_path = ? OR source_path LIKE ? ESCAPE '\'`,
		`DELETE FROM tags WHERE path = ? OR path LIKE ? ESCAPE '\'`,
		`DELETE FROM files WHERE path = ? OR path LIKE ? ESCAPE '\'`,
	} {
		if _, err := tx.Exec(stmt, exact, prefix); err != nil {
			return fmt.Errorf("exec %q: %w", stmt, err)
		}
	}
	return nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

func allFilePaths(tx *sql.Tx) ([]string, error) {
	rows, err := tx.Query(`SELECT path FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// deleteIndexedNote removes any existing notes/notes_fts/links/tags rows
// for exactPath, without touching its files row. Safe to call even if no
// such rows exist yet (first-time index of a path).
func deleteIndexedNote(tx *sql.Tx, exactPath string) error {
	var id int64
	err := tx.QueryRow(`SELECT id FROM notes WHERE path = ?`, exactPath).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// nothing indexed yet, nothing to delete
	case err != nil:
		return err
	default:
		if _, err := tx.Exec(`DELETE FROM notes_fts WHERE rowid = ?`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM links WHERE source_path = ?`, exactPath); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE path = ?`, exactPath); err != nil {
		return err
	}
	return nil
}

// indexNoteTx (re-)inserts a single note's rows, replacing any rows that
// already exist for relPath. Shared by ColdScan (where the delete is a
// no-op, since its tables were just cleared) and ReindexFile (where it's
// load-bearing).
func indexNoteTx(tx *sql.Tx, resolver *targetResolver, relPath string, content []byte) (links, tags int, err error) {
	if err := deleteIndexedNote(tx, relPath); err != nil {
		return 0, 0, err
	}

	note, err := vault.ParseNote(relPath, content)
	if err != nil {
		return 0, 0, err
	}

	frontmatterJSON, err := json.Marshal(note.Frontmatter)
	if err != nil {
		return 0, 0, err
	}

	res, err := tx.Exec(
		`INSERT INTO notes(path, title, body, frontmatter_json) VALUES (?, ?, ?, ?)`,
		note.Path, note.Title, note.Body, string(frontmatterJSON),
	)
	if err != nil {
		return 0, 0, err
	}
	noteID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO notes_fts(rowid, title, body) VALUES (?, ?, ?)`,
		noteID, note.Title, note.Body,
	); err != nil {
		return 0, 0, err
	}

	for _, link := range note.Links {
		var targetPath any
		if resolved := resolver.resolve(link.TargetRaw); resolved != "" {
			targetPath = resolved
		}
		if _, err := tx.Exec(
			`INSERT INTO links(source_path, target_raw, target_path, kind) VALUES (?, ?, ?, ?)`,
			note.Path, link.TargetRaw, targetPath, string(link.Kind),
		); err != nil {
			return 0, 0, err
		}
		links++
	}

	for _, tag := range note.Tags {
		if _, err := tx.Exec(`INSERT INTO tags(path, tag) VALUES (?, ?)`, note.Path, tag); err != nil {
			return 0, 0, err
		}
		tags++
	}

	return links, tags, nil
}
