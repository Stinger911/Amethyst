package writepath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/index"
)

func TestWriteAndIndex_WritesAndReindexesSynchronously(t *testing.T) {
	root := t.TempDir()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hash, err := WriteAndIndex(db, root, nil, "Note.md", []byte("# Hello\n"))
	if err != nil {
		t.Fatalf("WriteAndIndex: %v", err)
	}
	if hash == "" {
		t.Error("hash is empty")
	}

	onDisk, err := os.ReadFile(filepath.Join(root, "Note.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(onDisk) != "# Hello\n" {
		t.Errorf("on-disk content = %q, want %q", onDisk, "# Hello\n")
	}

	var body string
	if err := db.QueryRow(`SELECT body FROM notes WHERE path = 'Note.md'`).Scan(&body); err != nil {
		t.Fatalf("query body (expected synchronous reindex): %v", err)
	}
	if body != "# Hello\n" {
		t.Errorf("indexed body = %q, want %q", body, "# Hello\n")
	}
}

func TestWriteAndIndex_NilWatcherIsFine(t *testing.T) {
	root := t.TempDir()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := WriteAndIndex(db, root, nil, "Note.md", []byte("text")); err != nil {
		t.Fatalf("WriteAndIndex with nil watcher: %v", err)
	}
}
