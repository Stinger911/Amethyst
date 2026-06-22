// Package watch reacts to filesystem changes under a vault root and keeps
// the SQLite index (internal/index) incrementally up to date, instead of
// requiring a full ColdScan after every edit. Design follows
// research_amethyst-filewatcher-sync: debounce raw events, wait for the
// file size to stabilize (the fsnotify analog of chokidar's
// awaitWriteFinish, since editors — including Obsidian itself — save via
// write-to-temp + rename), then re-derive ground truth with a stat.
package watch

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/Stinger911/Amethyst/internal/index"
)

// Event reports the outcome of processing one stabilized filesystem
// change, for logging or (later) broadcasting over WebSocket.
type Event struct {
	Path string // vault-relative; empty for watcher-level errors
	Err  error
}

// Watcher watches Root recursively and reindexes one file at a time as
// changes settle. Zero-value duration fields are filled with sensible
// defaults by New.
type Watcher struct {
	Root string
	DB   *index.DB

	// DebounceWindow coalesces a burst of raw events on the same path
	// (e.g. an atomic save looks like unlink+create) into one check.
	DebounceWindow time.Duration
	// StabilityWindow is how long a file's size must stay unchanged
	// before it's considered done being written.
	StabilityWindow time.Duration
	// StabilityPoll is the interval used while polling size during
	// StabilityWindow.
	StabilityPoll time.Duration

	// OnEvent, if set, is called after every processed change.
	OnEvent func(Event)

	fsw *fsnotify.Watcher

	mu         sync.Mutex
	timers     map[string]*time.Timer
	wg         sync.WaitGroup
	indexMu    sync.Mutex           // serializes index writes; SQLite allows one writer at a time
	suppressed map[string]time.Time // relPath -> expiry, see Suppress
}

// suppressTTL bounds how long a Suppress call holds, in case the expected
// fsnotify event never arrives (e.g. a filesystem that doesn't report
// self-writes) — without it a missed event would permanently block
// reindexing of later, genuinely external changes to the same path.
const suppressTTL = 5 * time.Second

// Suppress marks relPath so the next process() call for it is skipped: the
// caller (the PUT /api/notes save path, see internal/api/notes_write.go)
// has already written the file and reindexed it itself, so redoing that
// from the watcher's own fsnotify event would just be redundant work — see
// research_amethyst-filewatcher-sync §6 step 5 ("self-suppression").
func (w *Watcher) Suppress(relPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.suppressed == nil {
		w.suppressed = make(map[string]time.Time)
	}
	w.suppressed[relPath] = time.Now().Add(suppressTTL)
}

// consumeSuppressed reports whether relPath was suppressed (and clears the
// flag either way, so a stale/expired entry doesn't linger).
func (w *Watcher) consumeSuppressed(relPath string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	expiry, ok := w.suppressed[relPath]
	if !ok {
		return false
	}
	delete(w.suppressed, relPath)
	return time.Now().Before(expiry)
}

// New creates a Watcher for the vault at root, backed by db. Call Start
// to begin watching, and Close when done.
func New(root string, db *index.DB) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		Root:            root,
		DB:              db,
		DebounceWindow:  400 * time.Millisecond,
		StabilityWindow: 300 * time.Millisecond,
		StabilityPoll:   100 * time.Millisecond,
		fsw:             fsw,
		timers:          make(map[string]*time.Timer),
	}, nil
}

// Close releases the underlying OS watch handles.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}

// Start adds watches for every non-dot directory under Root and processes
// events until ctx is cancelled. It blocks until then.
func (w *Watcher) Start(ctx context.Context) error {
	if err := w.addRecursive(w.Root); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			for _, t := range w.timers {
				t.Stop()
			}
			w.mu.Unlock()
			w.wg.Wait()
			return ctx.Err()

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(ev)

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			w.report(Event{Err: err})
		}
	}
}

// addRecursive registers a watch on dir and every non-dot subdirectory,
// and schedules any pre-existing files it finds (relevant when a whole
// folder appears at once, e.g. moved in from outside the vault).
func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != dir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return w.fsw.Add(p)
		}
		w.schedule(p)
		return nil
	})
}

func (w *Watcher) handleEvent(ev fsnotify.Event) {
	if ev.Op == fsnotify.Chmod {
		return // permissions-only change never affects indexed content
	}
	if strings.HasPrefix(filepath.Base(ev.Name), ".") {
		return
	}

	if ev.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			if err := w.addRecursive(ev.Name); err != nil {
				w.report(Event{Path: ev.Name, Err: err})
			}
			return
		}
	}

	w.schedule(ev.Name)
}

// schedule (re)starts the debounce timer for fullPath. Repeated events on
// the same path before the timer fires just push it back, so a burst
// collapses into a single process() call.
func (w *Watcher) schedule(fullPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if t, ok := w.timers[fullPath]; ok {
		t.Stop()
	} else {
		w.wg.Add(1)
	}
	w.timers[fullPath] = time.AfterFunc(w.DebounceWindow, func() {
		w.mu.Lock()
		delete(w.timers, fullPath)
		w.mu.Unlock()
		defer w.wg.Done()
		w.process(fullPath)
	})
}

func (w *Watcher) process(fullPath string) {
	w.awaitStable(fullPath)

	relPath, err := filepath.Rel(w.Root, fullPath)
	if err != nil {
		w.report(Event{Path: fullPath, Err: err})
		return
	}
	relPath = filepath.ToSlash(relPath)

	if w.consumeSuppressed(relPath) {
		return
	}

	w.indexMu.Lock()
	defer w.indexMu.Unlock()

	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		w.report(Event{Path: relPath, Err: index.RemoveFile(w.DB, relPath)})
		return
	}
	w.report(Event{Path: relPath, Err: index.ReindexFile(w.DB, w.Root, relPath)})
}

// awaitStable blocks until fullPath's size stops changing for
// StabilityWindow, or until it disappears (atomic-save replace, or a
// genuine delete — either way process() will re-stat to find out which).
func (w *Watcher) awaitStable(fullPath string) {
	var lastSize int64 = -1
	stableSince := time.Now()

	for {
		info, err := os.Stat(fullPath)
		if err != nil {
			return
		}
		if info.Size() != lastSize {
			lastSize = info.Size()
			stableSince = time.Now()
		}
		if time.Since(stableSince) >= w.StabilityWindow {
			return
		}
		time.Sleep(w.StabilityPoll)
	}
}

func (w *Watcher) report(ev Event) {
	if w.OnEvent != nil {
		w.OnEvent(ev)
	}
}
