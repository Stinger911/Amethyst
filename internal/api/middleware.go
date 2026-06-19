// This file implements the auth gate from plan_amethyst-web-ui step 4
// ("Auth — /login, гейт всех остальных маршрутов"): every content route
// requires a valid session, established via POST /api/auth/login.
package api

import (
	"net/http"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

// RequireAuth wraps next so it only runs for requests carrying a valid
// session cookie; anything else gets 401 without touching the index.
func RequireAuth(db *index.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		ok, err := auth.ValidateSession(db, c.Value)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
