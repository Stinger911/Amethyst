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
	"testing"
	"time"
)

// signTelegramWidgetData reproduces what Telegram's servers do when
// signing a Login Widget callback, so tests can build valid payloads
// without depending on VerifyTelegramWidgetData's own implementation.
func signTelegramWidgetData(t *testing.T, botToken string, values url.Values) {
	t.Helper()
	pairs := make([]string, 0, len(values))
	for k, v := range values {
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
}

func validTelegramValues(t *testing.T, botToken string) url.Values {
	t.Helper()
	values := url.Values{
		"id":         {"12345"},
		"first_name": {"Andrey"},
		"username":   {"stinger"},
		"auth_date":  {strconv.FormatInt(time.Now().Unix(), 10)},
	}
	signTelegramWidgetData(t, botToken, values)
	return values
}

func TestVerifyTelegramWidgetData_ValidSignaturePasses(t *testing.T) {
	values := validTelegramValues(t, "bot-token")
	if err := VerifyTelegramWidgetData("bot-token", values); err != nil {
		t.Fatalf("VerifyTelegramWidgetData: %v", err)
	}
}

func TestVerifyTelegramWidgetData_WrongBotTokenFails(t *testing.T) {
	values := validTelegramValues(t, "bot-token")
	err := VerifyTelegramWidgetData("a-different-token", values)
	if !errors.Is(err, ErrTelegramHashInvalid) {
		t.Fatalf("err = %v, want ErrTelegramHashInvalid", err)
	}
}

func TestVerifyTelegramWidgetData_TamperedFieldFails(t *testing.T) {
	values := validTelegramValues(t, "bot-token")
	values.Set("id", "99999") // tampered after signing
	err := VerifyTelegramWidgetData("bot-token", values)
	if !errors.Is(err, ErrTelegramHashInvalid) {
		t.Fatalf("err = %v, want ErrTelegramHashInvalid", err)
	}
}

func TestVerifyTelegramWidgetData_MissingHashFails(t *testing.T) {
	values := url.Values{"id": {"12345"}}
	err := VerifyTelegramWidgetData("bot-token", values)
	if !errors.Is(err, ErrTelegramHashInvalid) {
		t.Fatalf("err = %v, want ErrTelegramHashInvalid", err)
	}
}

func TestVerifyTelegramWidgetData_StaleAuthDateFails(t *testing.T) {
	values := url.Values{
		"id":        {"12345"},
		"auth_date": {strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10)},
	}
	signTelegramWidgetData(t, "bot-token", values)
	err := VerifyTelegramWidgetData("bot-token", values)
	if !errors.Is(err, ErrTelegramAuthStale) {
		t.Fatalf("err = %v, want ErrTelegramAuthStale", err)
	}
}
