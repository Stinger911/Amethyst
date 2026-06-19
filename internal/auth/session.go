package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/Stinger911/Amethyst/internal/index"
)

// SessionCookieName is the cookie carrying the session token.
const SessionCookieName = "amethyst_session"

// SessionLifetime is long-lived on purpose: this is a single-user,
// self-hosted system, not a multi-tenant service — there's no one else to
// protect the session from re-using, so it favors low login friction.
const SessionLifetime = 30 * 24 * time.Hour

// NewSession creates a session and returns the raw token to hand the
// client as a cookie. Only its hash is ever persisted, so a leak of the
// SQLite file alone doesn't let an attacker replay sessions.
func NewSession(db *index.DB) (token string, expiresAt time.Time, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expiresAt = time.Now().Add(SessionLifetime)

	if _, err := db.Exec(
		`INSERT INTO sessions(token_hash, expires_at) VALUES (?, ?)`,
		hashToken(token), expiresAt.Unix(),
	); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// ValidateSession reports whether token is a live, unexpired session.
// An expired session is deleted as a side effect instead of needing a
// separate cleanup job.
func ValidateSession(db *index.DB, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	hash := hashToken(token)

	var expiresAt int64
	err := db.QueryRow(`SELECT expires_at FROM sessions WHERE token_hash = ?`, hash).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if time.Now().Unix() >= expiresAt {
		_, err := db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hash)
		return false, err
	}
	return true, nil
}

// RevokeSession deletes one session (logout). A no-op if it doesn't exist.
func RevokeSession(db *index.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
