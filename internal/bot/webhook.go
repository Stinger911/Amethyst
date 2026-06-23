// This file implements TELEGRAM_MODE=webhook from
// plan_amethyst-telegram-bot §1/§5 step 6: a purely additive alternative
// transport to the default long-polling Run loop — same HandleUpdate
// dispatch, just fed by an HTTP POST from Telegram instead of
// GetUpdatesChan. Lowest priority by design: most self-hosted users have
// no public HTTPS endpoint to receive a webhook on at all.
package bot

import (
	"encoding/json"
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SetWebhook registers webhookURL with Telegram so it POSTs Updates there
// instead of waiting for GetUpdates polling. If secretToken is non-empty,
// Telegram echoes it back as the X-Telegram-Bot-Api-Secret-Token header on
// every webhook POST, which WebhookHandler then verifies — see plan §1.
func SetWebhook(api *tgbotapi.BotAPI, webhookURL, secretToken string) error {
	params := tgbotapi.Params{"url": webhookURL}
	if secretToken != "" {
		params["secret_token"] = secretToken
	}
	return makeRequestOK(api, "setWebhook", params)
}

// RemoveWebhook clears any registered webhook. Telegram refuses
// GetUpdates while a webhook is active, so polling mode must call this
// before starting (harmless, idempotent, if no webhook was ever set).
func RemoveWebhook(api *tgbotapi.BotAPI) error {
	return makeRequestOK(api, "deleteWebhook", tgbotapi.Params{})
}

func makeRequestOK(api *tgbotapi.BotAPI, endpoint string, params tgbotapi.Params) error {
	resp, err := api.MakeRequest(endpoint, params)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("%s: %s", endpoint, resp.Description)
	}
	return nil
}

// WebhookHandler serves the HTTP endpoint Telegram POSTs Updates to in
// webhook mode (registered at POST /api/telegram/webhook, see
// internal/api/server.go) — the same HandleUpdate dispatch as polling,
// just a different transport. Not behind RequireAuth: Telegram's servers
// don't carry our session cookie, so the secret token is this endpoint's
// only defense against forged requests.
func (b *Bot) WebhookHandler(secretToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if secretToken != "" && r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secretToken {
			http.Error(w, "invalid secret token", http.StatusUnauthorized)
			return
		}

		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "invalid update", http.StatusBadRequest)
			return
		}

		if update.Message != nil && update.Message.Chat != nil {
			b.HandleUpdate(IncomingMessage{
				ChatID: update.Message.Chat.ID,
				Text:   update.Message.Text,
			})
		}
		w.WriteHeader(http.StatusOK)
	}
}
