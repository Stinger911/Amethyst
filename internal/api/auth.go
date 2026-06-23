// This file implements POST /api/auth/login (and the paired /logout) from
// plan_amethyst-mvp Фаза 2 — the password-fallback auth method. The
// Telegram Login Widget (the primary method) is separate and not yet built.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

type loginRequest struct {
	Password string `json:"password"`
}

type authConfigResponse struct {
	TelegramBotUsername string `json:"telegramBotUsername"`
}

// AuthConfigHandler serves GET /api/auth/config: tells the (unauthenticated)
// login page whether to render the Telegram Login Widget, and under which
// bot username. Empty string means Telegram login isn't configured. The
// owner can come from env (TELEGRAM_OWNER_CHAT_ID) or from the dynamic
// pairing flow (see internal/auth's telegram_pairing.go) — env wins if set.
func AuthConfigHandler(db *index.DB, telegram TelegramConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := ""
		if telegram.BotToken != "" {
			owner, err := resolveOwnerChatID(db, telegram.OwnerChatID)
			if err == nil && owner != "" {
				username = telegram.BotUsername
			}
		}
		writeJSON(w, http.StatusOK, authConfigResponse{TelegramBotUsername: username})
	}
}

// LoginHandler serves POST /api/auth/login: verifies the password against
// the stored admin hash and, on success, sets a session cookie.
func LoginHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Password == "" {
			http.Error(w, "password is required", http.StatusBadRequest)
			return
		}

		ok, err := auth.VerifyPassword(db, req.Password)
		switch {
		case errors.Is(err, auth.ErrNoCredential):
			http.Error(w, "no admin password configured", http.StatusServiceUnavailable)
			return
		case err != nil:
			log.Printf("verify password: %v", err)
			http.Error(w, "login failed", http.StatusInternalServerError)
			return
		case !ok:
			http.Error(w, "invalid password", http.StatusUnauthorized)
			return
		}

		token, expiresAt, err := auth.NewSession(db)
		if err != nil {
			log.Printf("create session: %v", err)
			http.Error(w, "login failed", http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, r, token, expiresAt)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// LogoutHandler serves POST /api/auth/logout: revokes the session named by
// the request's cookie, if any, and clears it client-side.
func LogoutHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(auth.SessionCookieName); err == nil {
			if err := auth.RevokeSession(db, c.Value); err != nil {
				log.Printf("revoke session: %v", err)
			}
		}
		clearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// TelegramCallbackHandler serves GET /api/auth/telegram/callback: Telegram
// redirects the browser here after the user approves the Login Widget,
// with the user's data signed in the query string. The allowed Telegram
// user id is env-configured (TELEGRAM_OWNER_CHAT_ID, the same way
// ADMIN_PASSWORD seeds the password fallback) if set, otherwise whichever
// chat last completed the dynamic pairing flow (internal/auth's
// telegram_pairing.go).
func TelegramCallbackHandler(db *index.DB, botToken, envOwnerChatID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if botToken == "" {
			http.Error(w, "telegram login is not configured", http.StatusServiceUnavailable)
			return
		}
		ownerChatID, err := resolveOwnerChatID(db, envOwnerChatID)
		if err != nil {
			log.Printf("resolve telegram owner: %v", err)
			http.Error(w, "login failed", http.StatusInternalServerError)
			return
		}
		if ownerChatID == "" {
			http.Error(w, "telegram login is not configured", http.StatusServiceUnavailable)
			return
		}

		query := r.URL.Query()
		if err := auth.VerifyTelegramWidgetData(botToken, query); err != nil {
			http.Redirect(w, r, "/login?error=telegram", http.StatusFound)
			return
		}
		if query.Get("id") != ownerChatID {
			http.Redirect(w, r, "/login?error=telegram", http.StatusFound)
			return
		}

		token, expiresAt, err := auth.NewSession(db)
		if err != nil {
			log.Printf("create session (telegram): %v", err)
			http.Error(w, "login failed", http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, r, token, expiresAt)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// resolveOwnerChatID returns envOwnerChatID if set, otherwise whichever
// chat ID dynamic pairing has persisted (empty if neither).
func resolveOwnerChatID(db *index.DB, envOwnerChatID string) (string, error) {
	if envOwnerChatID != "" {
		return envOwnerChatID, nil
	}
	return auth.GetTelegramOwnerChatID(db)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
