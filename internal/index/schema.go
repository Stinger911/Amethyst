// Package index stores a disposable SQLite cache of a vault's files, notes,
// links and tags. The vault on disk is always the source of truth — this
// package only ever derives data from it and is safe to delete and rebuild.
package index

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// SchemaVersion is bumped whenever the table layout below changes. Open
// rebuilds the database from scratch on any mismatch instead of migrating —
// see plan_amethyst-storage-index §5: rebuilding a single-user vault's index
// is cheap, so a migration system isn't worth the engineering cost.
const SchemaVersion = 1

const schemaDDL = `
CREATE TABLE files (
	path         TEXT PRIMARY KEY,
	kind         TEXT NOT NULL,
	mtime        INTEGER NOT NULL,
	size         INTEGER NOT NULL,
	content_hash TEXT NOT NULL
);

CREATE TABLE notes (
	id               INTEGER PRIMARY KEY,
	path             TEXT NOT NULL UNIQUE REFERENCES files(path) ON DELETE CASCADE,
	title            TEXT NOT NULL,
	body             TEXT NOT NULL,
	frontmatter_json TEXT NOT NULL DEFAULT '{}'
);

CREATE VIRTUAL TABLE notes_fts USING fts5(
	title, body,
	content='notes', content_rowid='id'
);

CREATE TABLE links (
	source_path TEXT NOT NULL,
	target_raw  TEXT NOT NULL,
	target_path TEXT,
	kind        TEXT NOT NULL
);
CREATE INDEX links_source_idx ON links(source_path);
CREATE INDEX links_target_idx ON links(target_path);

CREATE TABLE tags (
	path TEXT NOT NULL,
	tag  TEXT NOT NULL
);
CREATE INDEX tags_tag_idx ON tags(tag);

CREATE TABLE meta (
	schema_version INTEGER NOT NULL
);
`

// authSchemaDDL holds the credential/session tables. These live outside
// schemaDDL/SchemaVersion on purpose: unlike the note index, they aren't
// derived from the vault and have no other source of truth, so an
// unrelated content-schema bump must never silently drop the admin's
// password or log everyone out. Open creates them once, idempotently, and
// rebuildSchema never touches them.
const authSchemaDDL = `
CREATE TABLE IF NOT EXISTS auth (
	id            INTEGER PRIMARY KEY CHECK (id = 1),
	password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	expires_at INTEGER NOT NULL
);
`

// settingsSchemaDDL holds user-configurable settings (e.g. the Telegram
// bot's capture mode, see internal/settings) — like auth, this is genuine
// persistent state, not a derived cache, so it's created idempotently here
// rather than living under SchemaVersion/rebuildSchema.
const settingsSchemaDDL = `
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// DB wraps a SQLite connection holding the index for a single vault.
type DB struct {
	*sql.DB
}

// Open opens (creating if necessary) the SQLite index at path and ensures
// its schema matches SchemaVersion, rebuilding from scratch if not.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite allows only one writer at a time; a busy_timeout makes
	// concurrent writers (e.g. several file-watcher events settling at
	// once) block and retry instead of failing immediately with SQLITE_BUSY.
	if _, err := conn.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	db := &DB{conn}
	if err := db.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(authSchemaDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create auth schema: %w", err)
	}
	if _, err := db.Exec(settingsSchemaDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create settings schema: %w", err)
	}
	return db, nil
}

func (db *DB) ensureSchema() error {
	version, err := db.currentSchemaVersion()
	if err != nil {
		return err
	}
	if version == SchemaVersion {
		return nil
	}
	return db.rebuildSchema()
}

// currentSchemaVersion returns 0 if the meta table doesn't exist yet
// (fresh database) rather than treating that as an error.
func (db *DB) currentSchemaVersion() (int, error) {
	var version int
	err := db.QueryRow(`SELECT schema_version FROM meta LIMIT 1`).Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil && strings.Contains(err.Error(), "no such table"):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("read schema_version: %w", err)
	default:
		return version, nil
	}
}

func (db *DB) rebuildSchema() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"files", "notes", "notes_fts", "links", "tags", "meta"} {
		if _, err := tx.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			return fmt.Errorf("drop %s: %w", table, err)
		}
	}
	if _, err := tx.Exec(schemaDDL); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO meta(schema_version) VALUES (?)`, SchemaVersion); err != nil {
		return fmt.Errorf("set schema_version: %w", err)
	}
	return tx.Commit()
}
