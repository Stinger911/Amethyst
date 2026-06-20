package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Stinger911/Amethyst/internal/auth"
)

// signTelegramWidgetData mirrors what Telegram's servers do when signing a
// Login Widget callback, independent of the auth package's own
// implementation of the same algorithm.
func signTelegramWidgetData(botToken string, values url.Values) {
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

func telegramCallbackRequest(botToken, ownerID string) *http.Request {
	values := url.Values{
		"id":         {ownerID},
		"first_name": {"Andrey"},
		"auth_date":  {strconv.FormatInt(time.Now().Unix(), 10)},
	}
	signTelegramWidgetData(botToken, values)
	return httptest.NewRequest(http.MethodGet, "/api/auth/telegram/callback?"+values.Encode(), nil)
}

func TestTelegramCallback_NotConfiguredIsServiceUnavailable(t *testing.T) {
	db := openMiddlewareTestDB(t)
	req := telegramCallbackRequest("bot-token", "12345")
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestTelegramCallback_ValidOwnerSetsSessionCookie(t *testing.T) {
	db := openMiddlewareTestDB(t)
	req := telegramCallbackRequest("bot-token", "12345")
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{BotToken: "bot-token", OwnerChatID: "12345"}).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s, want 302", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want %q", loc, "/")
	}

	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie set on successful telegram login")
	}
	ok, err := auth.ValidateSession(db, sessionCookie.Value)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if !ok {
		t.Error("the cookie's session should validate against the DB")
	}
}

func TestTelegramCallback_WrongOwnerIdIsRejected(t *testing.T) {
	db := openMiddlewareTestDB(t)
	req := telegramCallbackRequest("bot-token", "99999")
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{BotToken: "bot-token", OwnerChatID: "12345"}).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 redirect to /login", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("no session cookie should be set for a non-owner id")
	}
}

func TestTelegramCallback_TamperedSignatureIsRejected(t *testing.T) {
	db := openMiddlewareTestDB(t)
	req := telegramCallbackRequest("bot-token", "12345")
	q := req.URL.Query()
	q.Set("first_name", "Mallory")
	req.URL.RawQuery = q.Encode()

	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{BotToken: "bot-token", OwnerChatID: "12345"}).ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 redirect to /login", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("no session cookie should be set for a tampered payload")
	}
}
