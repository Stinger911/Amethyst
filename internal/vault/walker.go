package vault

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// FileKind classifies a file found while walking the vault, matching the
// `files.kind` column in plan_amethyst-storage-index.
type FileKind string

const (
	KindNote       FileKind = "note"
	KindCanvas     FileKind = "canvas"
	KindAttachment FileKind = "attachment"
)

// FileEntry describes one file in the vault, enough to populate the
// `files` table without reading its content.
type FileEntry struct {
	Path    string // vault-relative, forward-slash separated
	Kind    FileKind
	ModTime time.Time
	Size    int64
}

// Walk traverses root and returns one FileEntry per file. Dotfiles and
// dotdirs (.obsidian, .trash, .git, ...) are skipped entirely — per
// plan_amethyst-storage-index, `.obsidian/` config is read on demand,
// never indexed as content.
func Walk(root string) ([]FileEntry, error) {
	var entries []FileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		entries = append(entries, FileEntry{
			Path:    filepath.ToSlash(rel),
			Kind:    ClassifyKind(rel),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ClassifyKind derives a FileKind from a path's extension alone, the same
// rule Walk uses for each entry. Exported so callers reindexing a single
// path (e.g. an incremental file-watcher update) don't have to re-walk.
func ClassifyKind(path string) FileKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		return KindNote
	case ".canvas":
		return KindCanvas
	default:
		return KindAttachment
	}
}
