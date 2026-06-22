package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Stinger911/Amethyst/internal/index"
)

func newTestWatcher(t *testing.T, root string) (*Watcher, chan Event) {
	t.Helper()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	w, err := New(root, db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	w.DebounceWindow = 20 * time.Millisecond
	w.StabilityWindow = 20 * time.Millisecond
	w.StabilityPoll = 5 * time.Millisecond

	events := make(chan Event, 64)
	w.OnEvent = func(ev Event) { events <- ev }
	t.Cleanup(func() { w.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go w.Start(ctx)

	// Give the initial addRecursive a moment to register OS watches
	// before the test starts mutating the filesystem.
	time.Sleep(20 * time.Millisecond)

	return w, events
}

func awaitEvent(t *testing.T, events chan Event, wantPath string) Event {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Path == wantPath {
				return ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event on %q", wantPath)
		}
	}
}

func TestWatcher_IndexesNewFile(t *testing.T) {
	root := t.TempDir()
	_, events := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("Hello watcher.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ev := awaitEvent(t, events, "note.md")
	if ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}
}

func TestWatcher_ReindexesOnModify(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("Original.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	w, events := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("Updated content.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if ev := awaitEvent(t, events, "note.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}

	var body string
	if err := w.DB.QueryRow(`SELECT body FROM notes WHERE path = 'note.md'`).Scan(&body); err != nil {
		t.Fatalf("query body: %v", err)
	}
	if body != "Updated content.\n" {
		t.Errorf("body = %q, want %q", body, "Updated content.\n")
	}
}

func TestWatcher_RemovesDeletedFile(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "note.md")
	if err := os.WriteFile(full, []byte("Temporary.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	w, events := newTestWatcher(t, root)

	if err := os.Remove(full); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if ev := awaitEvent(t, events, "note.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}

	var count int
	if err := w.DB.QueryRow(`SELECT count(*) FROM files WHERE path = 'note.md'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("files count = %d, want 0 after delete", count)
	}
}

func TestWatcher_AtomicSaveCollapsesToSingleReindex(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "note.md")
	if err := os.WriteFile(full, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	w, events := newTestWatcher(t, root)

	// Simulate write-to-temp + rename, the pattern Obsidian and most
	// editors use for atomic saves.
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, []byte("v2 final\n"), 0o644); err != nil {
		t.Fatalf("WriteFile tmp: %v", err)
	}
	if err := os.Rename(tmp, full); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	if ev := awaitEvent(t, events, "note.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}

	var body string
	if err := w.DB.QueryRow(`SELECT body FROM notes WHERE path = 'note.md'`).Scan(&body); err != nil {
		t.Fatalf("query body: %v", err)
	}
	if body != "v2 final\n" {
		t.Errorf("body = %q, want %q", body, "v2 final\n")
	}
}

func TestWatcher_NewSubdirectoryWithFilesGetsIndexed(t *testing.T) {
	root := t.TempDir()
	_, events := newTestWatcher(t, root)

	sub := filepath.Join(root, "Folder")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "child.md"), []byte("Child note.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if ev := awaitEvent(t, events, "Folder/child.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}
}

func TestWatcher_SuppressSkipsOneEventThenResumes(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "note.md")
	if err := os.WriteFile(full, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	w, events := newTestWatcher(t, root)
	// addRecursive schedules every pre-existing file at startup, so drain
	// that initial reindex before testing suppression of our own write —
	// otherwise the two could race.
	awaitEvent(t, events, "note.md")

	w.Suppress("note.md")
	if err := os.WriteFile(full, []byte("v2 (written by API handler)\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	select {
	case ev := <-events:
		t.Fatalf("expected suppressed write to produce no event, got %+v", ev)
	case <-time.After(200 * time.Millisecond):
		// expected: nothing arrived
	}

	// Suppress is consumed by the first process() call, so a subsequent,
	// genuinely external change must still be picked up normally.
	if err := os.WriteFile(full, []byte("v3 (external)\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if ev := awaitEvent(t, events, "note.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}

	var body string
	if err := w.DB.QueryRow(`SELECT body FROM notes WHERE path = 'note.md'`).Scan(&body); err != nil {
		t.Fatalf("query body: %v", err)
	}
	if body != "v3 (external)\n" {
		t.Errorf("body = %q, want %q", body, "v3 (external)\n")
	}
}

func TestWatcher_SkipsDotDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w, events := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, ".obsidian", "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Also make a real, watched change so we have a positive signal that
	// processing is happening at all (otherwise a silent no-op for the
	// dotfile would be indistinguishable from "test never ran").
	if err := os.WriteFile(filepath.Join(root, "real.md"), []byte("Real note.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if ev := awaitEvent(t, events, "real.md"); ev.Err != nil {
		t.Fatalf("process error: %v", ev.Err)
	}

	var count int
	if err := w.DB.QueryRow(`SELECT count(*) FROM files WHERE path LIKE '.obsidian%'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("files count under .obsidian = %d, want 0", count)
	}
}
