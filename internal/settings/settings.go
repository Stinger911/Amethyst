// Package settings stores user-configurable runtime settings — currently
// just the Telegram bot's capture target (plan_amethyst-telegram-bot §4) —
// in the settings table created by internal/index.
package settings

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Stinger911/Amethyst/internal/index"
)

const (
	// CaptureModeInbox writes each captured Telegram message to its own
	// file under Inbox/. It's the default: it needs no per-vault
	// configuration and works for any vault layout.
	CaptureModeInbox = "inbox"
	// CaptureModeDaily appends captured messages to today's daily note
	// instead, resolving only {{date}} — no Templater JS execution.
	CaptureModeDaily = "daily"

	keyTelegramCaptureMode = "telegram_capture_mode"
)

// GetCaptureMode returns the configured Telegram-bot capture target,
// defaulting to CaptureModeInbox when nothing has been set yet.
func GetCaptureMode(db *index.DB) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, keyTelegramCaptureMode).Scan(&value)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return CaptureModeInbox, nil
	case err != nil:
		return "", err
	default:
		return value, nil
	}
}

// SetCaptureMode persists mode, which must be CaptureModeInbox or
// CaptureModeDaily.
func SetCaptureMode(db *index.DB, mode string) error {
	if mode != CaptureModeInbox && mode != CaptureModeDaily {
		return fmt.Errorf("invalid capture mode %q", mode)
	}
	_, err := db.Exec(
		`INSERT INTO settings(key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		keyTelegramCaptureMode, mode,
	)
	return err
}
