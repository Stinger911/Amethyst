// This file implements the dynamic Telegram pairing flow from
// plan_amethyst-telegram-bot §3: an already-logged-in user generates a
// one-time "/start <token>" command on the Settings page, sends it to the
// bot, and the bot persists that chat as the Telegram owner — an
// alternative to setting TELEGRAM_OWNER_CHAT_ID by hand. Env still wins if
// set (see internal/api/auth.go's TelegramCallbackHandler and
// internal/bot's currentOwnerChatID) — this is only the fallback path.
package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/Stinger911/Amethyst/internal/index"
)

// PairingTokenTTL bounds how long a generated pairing token stays valid —
// short on purpose, since it's shown once and meant to be used immediately.
const PairingTokenTTL = 10 * time.Minute

// NewPairingToken generates a one-time pairing token and returns the raw
// value to show the user once; only its hash is persisted.
func NewPairingToken(db *index.DB) (token string, expiresAt time.Time, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expiresAt = time.Now().Add(PairingTokenTTL)

	if _, err := db.Exec(
		`INSERT INTO telegram_pairing_tokens(token_hash, expires_at) VALUES (?, ?)`,
		hashToken(token), expiresAt.Unix(),
	); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// RedeemPairingToken validates token — deleting it either way, since
// pairing tokens are single-use — and, if it was valid and unexpired,
// persists chatID as the Telegram owner. It reports whether pairing
// succeeded; an invalid or expired token is not an error, just a "no".
func RedeemPairingToken(db *index.DB, token, chatID string) (bool, error) {
	if token == "" {
		return false, nil
	}
	hash := hashToken(token)

	var expiresAt int64
	err := db.QueryRow(`SELECT expires_at FROM telegram_pairing_tokens WHERE token_hash = ?`, hash).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := db.Exec(`DELETE FROM telegram_pairing_tokens WHERE token_hash = ?`, hash); err != nil {
		return false, err
	}
	if time.Now().Unix() >= expiresAt {
		return false, nil
	}

	if err := SetTelegramOwnerChatID(db, chatID); err != nil {
		return false, err
	}
	return true, nil
}

// SetTelegramOwnerChatID persists chatID as the dynamically-paired
// Telegram owner.
func SetTelegramOwnerChatID(db *index.DB, chatID string) error {
	_, err := db.Exec(
		`INSERT INTO telegram_owner(id, chat_id) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET chat_id = excluded.chat_id`,
		chatID,
	)
	return err
}

// GetTelegramOwnerChatID returns the dynamically-paired owner chat ID, or
// "" if pairing hasn't happened (yet).
func GetTelegramOwnerChatID(db *index.DB) (string, error) {
	var chatID string
	err := db.QueryRow(`SELECT chat_id FROM telegram_owner WHERE id = 1`).Scan(&chatID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return chatID, nil
}
