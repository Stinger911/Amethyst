// This file verifies the Telegram Login Widget callback
// (https://core.telegram.org/widgets/login#checking-authorization) for
// plan_amethyst-mvp Фаза 2. Owner identity is env-configured
// (TELEGRAM_OWNER_CHAT_ID) for now, the same way ADMIN_PASSWORD is — the
// pairing flow that would let the bot set it dynamically needs the bot
// itself, which is plan_amethyst-telegram-bot Фаза 4, not yet built.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TelegramWidgetMaxAge bounds how old a callback's auth_date may be,
// capping the replay window if a callback URL ever leaks (e.g. via logs).
const TelegramWidgetMaxAge = 1 * time.Hour

// ErrTelegramHashInvalid means the callback's signature doesn't match what
// HMAC-SHA256 over the bot token produces — it didn't genuinely come from
// Telegram for this bot.
var ErrTelegramHashInvalid = errors.New("telegram auth: hash mismatch")

// ErrTelegramAuthStale means auth_date is missing, malformed, or older
// than TelegramWidgetMaxAge.
var ErrTelegramAuthStale = errors.New("telegram auth: auth_date too old")

// VerifyTelegramWidgetData checks the signature Telegram attaches to a
// Login Widget callback's query parameters.
func VerifyTelegramWidgetData(botToken string, values url.Values) error {
	hash := values.Get("hash")
	if hash == "" {
		return ErrTelegramHashInvalid
	}

	pairs := make([]string, 0, len(values))
	for k, v := range values {
		if k == "hash" || len(v) == 0 {
			continue
		}
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	computed := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(computed), []byte(hash)) {
		return ErrTelegramHashInvalid
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return ErrTelegramAuthStale
	}
	if time.Since(time.Unix(authDate, 0)) > TelegramWidgetMaxAge {
		return ErrTelegramAuthStale
	}
	return nil
}
