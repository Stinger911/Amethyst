// This file implements /start <token> from plan_amethyst-telegram-bot §3:
// redeeming a one-time token generated on the Settings page (POST
// /api/telegram/pair) persists this chat as the Telegram owner.
package bot

import (
	"log"
	"strconv"

	"github.com/Stinger911/Amethyst/internal/auth"
)

func (b *Bot) handleStart(chatID int64, token string) {
	if token == "" {
		// No argument: nothing to pair, stay silent per plan §4 — an
		// unconfigured bot shouldn't reveal anything to a random sender.
		return
	}

	ok, err := auth.RedeemPairingToken(b.DB, token, strconv.FormatInt(chatID, 10))
	if err != nil {
		log.Printf("telegram pairing: %v", err)
		return
	}
	if !ok {
		// Invalid or expired token — ignore silently per plan §4.
		return
	}

	b.reply(chatID, "Telegram linked. You can now use capture, /search, /note, and Telegram login from this chat.")
}
