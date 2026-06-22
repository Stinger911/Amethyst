// This file implements the capture flow from plan_amethyst-telegram-bot
// §4: a plain-text message becomes either its own note under Inbox/, or a
// new paragraph appended to today's daily note, depending on
// settings.GetCaptureMode (configurable on the web UI's Settings page).
package bot

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/Stinger911/Amethyst/internal/settings"
	"github.com/Stinger911/Amethyst/internal/writepath"
)

func (b *Bot) handleCapture(chatID int64, text string) {
	mode, err := settings.GetCaptureMode(b.DB)
	if err != nil {
		log.Printf("capture: get mode: %v", err)
		mode = settings.CaptureModeInbox
	}

	var relPath string
	if mode == settings.CaptureModeDaily {
		relPath, err = b.captureToDaily(text)
	} else {
		relPath, err = b.captureToInbox(text)
	}
	if err != nil {
		log.Printf("capture: %v", err)
		b.reply(chatID, "Failed to save your message.")
		return
	}
	b.reply(chatID, fmt.Sprintf("Saved to %s", relPath))
}

// captureToInbox writes text as its own new note under Inbox/, named by
// timestamp so concurrent captures don't collide. Two messages landing in
// the same second get a "-2", "-3", ... suffix instead of one silently
// overwriting the other.
func (b *Bot) captureToInbox(text string) (string, error) {
	base := time.Now().UTC().Format("20060102-150405")
	relPath := path.Join("Inbox", base+".md")
	for n := 2; b.fileExists(relPath); n++ {
		relPath = path.Join("Inbox", fmt.Sprintf("%s-%d.md", base, n))
	}
	if err := b.ensureParentDir(relPath); err != nil {
		return "", err
	}
	if _, err := writepath.WriteAndIndex(b.DB, b.VaultRoot, b.Watcher, relPath, []byte(text+"\n")); err != nil {
		return "", err
	}
	return relPath, nil
}

func (b *Bot) fileExists(relPath string) bool {
	_, err := os.Stat(path.Join(b.VaultRoot, relPath))
	return err == nil
}

// captureToDaily appends text to today's daily note (Daily/<date>.md),
// creating it with a single "# <date>" heading if it doesn't exist yet.
// Only {{date}} is resolved — no Templater JS, per plan §4.
func (b *Bot) captureToDaily(text string) (string, error) {
	today := time.Now().Format("2006-01-02")
	relPath := path.Join("Daily", today+".md")
	if err := b.ensureParentDir(relPath); err != nil {
		return "", err
	}

	existing, err := os.ReadFile(path.Join(b.VaultRoot, relPath))
	if errors.Is(err, os.ErrNotExist) {
		existing = []byte("# " + today + "\n\n")
	} else if err != nil {
		return "", err
	}

	updated := append(existing, []byte(text+"\n\n")...)
	if _, err := writepath.WriteAndIndex(b.DB, b.VaultRoot, b.Watcher, relPath, updated); err != nil {
		return "", err
	}
	return relPath, nil
}

func (b *Bot) ensureParentDir(relPath string) error {
	return os.MkdirAll(filepath.Dir(filepath.Join(b.VaultRoot, relPath)), 0o755)
}
