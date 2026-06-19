// Package auth implements the password-fallback login from
// plan_amethyst-mvp Фаза 2: a single admin credential (bcrypt hash, one
// row by design — this is a single-user system) and opaque session
// tokens. The Telegram Login Widget, the primary auth method, is a
// separate, not-yet-built piece.
package auth

import (
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/Stinger911/Amethyst/internal/index"
)

// ErrNoCredential means no admin password has ever been set.
var ErrNoCredential = errors.New("no admin credential configured")

// EnsureCredential seeds or resets the admin password hash. Per
// plan_amethyst-mvp ("восстановление пароля через переменную окружения"):
// normal restarts leave an existing credential untouched even if password
// is non-empty; reset must be explicitly requested. The very first run,
// with no credential yet, requires a non-empty password.
func EnsureCredential(db *index.DB, password string, reset bool) error {
	exists, err := hasCredential(db)
	if err != nil {
		return err
	}
	if exists && !reset {
		return nil
	}
	if password == "" {
		if exists {
			return fmt.Errorf("ADMIN_PASSWORD_RESET is set but ADMIN_PASSWORD is empty")
		}
		return fmt.Errorf("no admin credential exists yet; set ADMIN_PASSWORD on first run")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = db.Exec(
		`INSERT INTO auth(id, password_hash) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET password_hash = excluded.password_hash`,
		string(hash),
	)
	return err
}

func hasCredential(db *index.DB) (bool, error) {
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM auth`).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// VerifyPassword reports whether password matches the stored admin hash.
func VerifyPassword(db *index.DB, password string) (bool, error) {
	var hash string
	err := db.QueryRow(`SELECT password_hash FROM auth WHERE id = 1`).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNoCredential
	}
	if err != nil {
		return false, err
	}

	switch err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, err
	}
}
