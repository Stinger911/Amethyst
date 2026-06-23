package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramWebhookRoute_OmittedWhenHandlerIsNil(t *testing.T) {
	db := openAuthTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil, nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when no webhook handler is wired up", rec.Code)
	}
}

func TestTelegramWebhookRoute_DispatchesToHandler(t *testing.T) {
	db := openAuthTestDB(t)
	var called bool
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil, handler, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Error("the registered webhook handler was never invoked")
	}
}

// Not behind RequireAuth: Telegram's servers never carry our session
// cookie, so an unauthenticated POST must still reach the handler.
func TestTelegramWebhookRoute_NotGatedByAuth(t *testing.T) {
	db := openAuthTestDB(t)
	var called bool
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil, handler, nil).ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatal("webhook route should not require a session cookie")
	}
	if !called {
		t.Error("the registered webhook handler was never invoked")
	}
}
