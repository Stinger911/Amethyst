package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

func openAuthTestDB(t *testing.T) *index.DB {
	t.Helper()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func postJSON(t *testing.T, db *index.DB, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil).ServeHTTP(rec, req)
	return rec
}

func TestLogin_CorrectPasswordSetsSessionCookie(t *testing.T) {
	db := openAuthTestDB(t)
	if err := auth.EnsureCredential(db, "s3cret", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}

	rec := postJSON(t, db, "/api/auth/login", loginRequest{Password: "s3cret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie set on successful login")
	}
	if sessionCookie.Value == "" {
		t.Error("session cookie value is empty")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}

	ok, err := auth.ValidateSession(db, sessionCookie.Value)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if !ok {
		t.Error("the cookie's session should validate against the DB")
	}
}

func TestLogin_WrongPasswordIsUnauthorized(t *testing.T) {
	db := openAuthTestDB(t)
	if err := auth.EnsureCredential(db, "s3cret", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}

	rec := postJSON(t, db, "/api/auth/login", loginRequest{Password: "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("no cookie should be set on a failed login")
	}
}

func TestLogin_NoCredentialConfiguredYet(t *testing.T) {
	db := openAuthTestDB(t)
	rec := postJSON(t, db, "/api/auth/login", loginRequest{Password: "anything"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestLogin_EmptyPasswordIsBadRequest(t *testing.T) {
	db := openAuthTestDB(t)
	rec := postJSON(t, db, "/api/auth/login", loginRequest{Password: ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAuthConfig_TelegramNotConfigured(t *testing.T) {
	db := openAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp authConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TelegramBotUsername != "" {
		t.Errorf("telegramBotUsername = %q, want empty when telegram login is unconfigured", resp.TelegramBotUsername)
	}
}

func TestAuthConfig_TelegramConfigured(t *testing.T) {
	db := openAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{
		BotToken:    "bot-token",
		OwnerChatID: "12345",
		BotUsername: "AmethystBot",
	}, WriteConfig{}, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp authConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TelegramBotUsername != "AmethystBot" {
		t.Errorf("telegramBotUsername = %q, want %q", resp.TelegramBotUsername, "AmethystBot")
	}
}

func TestLogout_RevokesSessionAndClearsCookie(t *testing.T) {
	db := openAuthTestDB(t)
	if err := auth.EnsureCredential(db, "s3cret", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}

	loginRec := postJSON(t, db, "/api/auth/login", loginRequest{Password: "s3cret"})
	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("login did not set a session cookie")
	}

	logoutRec := postJSON(t, db, "/api/auth/logout", nil, sessionCookie)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, body = %s", logoutRec.Code, logoutRec.Body.String())
	}

	ok, err := auth.ValidateSession(db, sessionCookie.Value)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if ok {
		t.Error("session should be invalid after logout")
	}
}
