package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")

	if err := WriteAtomic(target, []byte("hello")); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestWriteAtomic_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := WriteAtomic(target, []byte("new")); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
}

func TestWriteAtomic_NoTempFileLeftBehind(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")

	if err := WriteAtomic(target, []byte("hello")); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "note.md" {
		t.Errorf("dir entries = %v, want only note.md", entries)
	}
}
