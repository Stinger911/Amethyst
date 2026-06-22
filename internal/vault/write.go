package vault

import (
	"os"
	"path/filepath"
)

// WriteAtomic writes content to fullPath via a temp file in the same
// directory followed by a rename, so a reader (or this vault's own file
// watcher) never observes a partially written file — the same pattern
// Obsidian itself uses for saves (see research_amethyst-filewatcher-sync §3).
func WriteAtomic(fullPath string, content []byte) error {
	dir := filepath.Dir(fullPath)
	tmp, err := os.CreateTemp(dir, ".amethyst-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, fullPath)
}
