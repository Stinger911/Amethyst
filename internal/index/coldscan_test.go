package index

import (
	"os"
	"path/filepath"
	"testing"
)

func writeVaultFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "index.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpen_CreatesFreshSchema(t *testing.T) {
	db := openTestDB(t)
	var version int
	if err := db.QueryRow(`SELECT schema_version FROM meta`).Scan(&version); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != SchemaVersion {
		t.Errorf("schema_version = %d, want %d", version, SchemaVersion)
	}
}

func TestOpen_RebuildsOnVersionMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO files(path, kind, mtime, size, content_hash) VALUES ('stale.md','note',0,0,'x')`); err != nil {
		t.Fatalf("seed stale row: %v", err)
	}
	if _, err := db.Exec(`UPDATE meta SET schema_version = -1`); err != nil {
		t.Fatalf("force mismatch: %v", err)
	}
	db.Close()

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer db2.Close()

	var count int
	if err := db2.QueryRow(`SELECT count(*) FROM files`).Scan(&count); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if count != 0 {
		t.Errorf("files count = %d, want 0 after rebuild", count)
	}
}

func TestColdScan_PopulatesTables(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "Notes/A.md", "---\ntags: [project]\n---\nLinks to [[B]] and embeds ![[photo.png]].\n")
	writeVaultFile(t, root, "Notes/B.md", "Body of B with #inline-tag.\n")
	writeVaultFile(t, root, "Attachments/photo.png", "binary")
	writeVaultFile(t, root, "Board.canvas", "{}")

	db := openTestDB(t)
	stats, err := ColdScan(db, root)
	if err != nil {
		t.Fatalf("ColdScan: %v", err)
	}

	if stats.Files != 4 {
		t.Errorf("Files = %d, want 4", stats.Files)
	}
	if stats.Notes != 2 {
		t.Errorf("Notes = %d, want 2", stats.Notes)
	}
	if stats.Links != 2 {
		t.Errorf("Links = %d, want 2", stats.Links)
	}
	if stats.Tags != 2 {
		t.Errorf("Tags = %d, want 2", stats.Tags)
	}

	var targetPath string
	if err := db.QueryRow(
		`SELECT target_path FROM links WHERE source_path = 'Notes/A.md' AND target_raw = 'B'`,
	).Scan(&targetPath); err != nil {
		t.Fatalf("query link target: %v", err)
	}
	if targetPath != "Notes/B.md" {
		t.Errorf("resolved target_path = %q, want %q", targetPath, "Notes/B.md")
	}

	var embedTarget string
	if err := db.QueryRow(
		`SELECT target_path FROM links WHERE source_path = 'Notes/A.md' AND target_raw = 'photo.png'`,
	).Scan(&embedTarget); err != nil {
		t.Fatalf("query embed target: %v", err)
	}
	if embedTarget != "Attachments/photo.png" {
		t.Errorf("resolved embed target_path = %q, want %q", embedTarget, "Attachments/photo.png")
	}

	var matched int
	if err := db.QueryRow(
		`SELECT count(*) FROM notes_fts WHERE notes_fts MATCH 'Links'`,
	).Scan(&matched); err != nil {
		t.Fatalf("fts query: %v", err)
	}
	if matched != 1 {
		t.Errorf("fts match count = %d, want 1", matched)
	}
}

func TestColdScan_IsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "note.md", "Hello.\n")

	db := openTestDB(t)
	if _, err := ColdScan(db, root); err != nil {
		t.Fatalf("first ColdScan: %v", err)
	}
	stats, err := ColdScan(db, root)
	if err != nil {
		t.Fatalf("second ColdScan: %v", err)
	}
	if stats.Files != 1 || stats.Notes != 1 {
		t.Errorf("stats after rescan = %+v, want 1 file / 1 note (no duplicates)", stats)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM files`).Scan(&count); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if count != 1 {
		t.Errorf("files row count = %d, want 1", count)
	}
}
