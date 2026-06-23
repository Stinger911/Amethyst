// This file implements POST /api/telegram/pair from
// plan_amethyst-telegram-bot §3: an already-logged-in user generates a
// one-time pairing token, sends "/start <token>" to the bot, and the bot
// persists that chat as the Telegram owner (internal/auth's
// telegram_pairing.go) — an alternative to setting TELEGRAM_OWNER_CHAT_ID
// by hand.
package api

import (
	"log"
	"net/http"
	"time"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

// PairResponse is the JSON body of POST /api/telegram/pair.
type PairResponse struct {
	Token string `json:"token"`
	// BotUsername lets the frontend offer a t.me/<username>?start=<token>
	// deep link in addition to the literal "/start <token>" command; empty
	// if the bot username isn't configured.
	BotUsername string `json:"botUsername"`
	ExpiresAt   string `json:"expiresAt"`
}

// PairHandler serves POST /api/telegram/pair.
func PairHandler(db *index.DB, botUsername string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, expiresAt, err := auth.NewPairingToken(db)
		if err != nil {
			log.Printf("create pairing token: %v", err)
			http.Error(w, "create pairing token failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, PairResponse{
			Token:       token,
			BotUsername: botUsername,
			ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
		})
	}
}
