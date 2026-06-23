package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
)

func TestPair_RequiresAuth(t *testing.T) {
	db := openAuthTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/pair", nil)
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{BotUsername: "AmethystBot"}, WriteConfig{}, nil, nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without a session", rec.Code)
	}
}

func TestPair_ReturnsRedeemableToken(t *testing.T) {
	db := openAuthTestDB(t)
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/telegram/pair", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{BotUsername: "AmethystBot"}, WriteConfig{}, nil, nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp PairResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token == "" {
		t.Error("Token is empty")
	}
	if resp.BotUsername != "AmethystBot" {
		t.Errorf("BotUsername = %q, want %q", resp.BotUsername, "AmethystBot")
	}
	if resp.ExpiresAt == "" {
		t.Error("ExpiresAt is empty")
	}

	ok, err := auth.RedeemPairingToken(db, resp.Token, "12345")
	if err != nil {
		t.Fatalf("RedeemPairingToken: %v", err)
	}
	if !ok {
		t.Error("the token returned by /api/telegram/pair did not redeem successfully")
	}
}
