package vault

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestWalk_ClassifiesAndSkipsDotfiles(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "Notes", "page.md"), "# Page\n")
	writeFile(t, filepath.Join(root, "Board.canvas"), "{}")
	writeFile(t, filepath.Join(root, "Attachments", "photo.png"), "binary")
	writeFile(t, filepath.Join(root, ".obsidian", "config.json"), "{}")
	writeFile(t, filepath.Join(root, ".trash", "deleted.md"), "gone")

	entries, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	got := map[string]FileKind{}
	for _, e := range entries {
		got[e.Path] = e.Kind
	}

	want := map[string]FileKind{
		"Notes/page.md":         KindNote,
		"Board.canvas":          KindCanvas,
		"Attachments/photo.png": KindAttachment,
	}
	if len(got) != len(want) {
		paths := make([]string, 0, len(got))
		for p := range got {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		t.Fatalf("entries = %v, want keys %v", paths, want)
	}
	for path, kind := range want {
		if got[path] != kind {
			t.Errorf("Kind(%q) = %q, want %q", path, got[path], kind)
		}
	}
}

func TestWalk_ReportsSizeAndModTime(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "note.md"), "hello world")

	entries, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Size != int64(len("hello world")) {
		t.Errorf("Size = %d, want %d", entries[0].Size, len("hello world"))
	}
	if entries[0].ModTime.IsZero() {
		t.Error("ModTime is zero, want a real timestamp")
	}
}

func TestWalk_EmptyVault(t *testing.T) {
	root := t.TempDir()
	entries, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want empty", entries)
	}
}
