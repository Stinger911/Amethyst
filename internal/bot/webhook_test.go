package bot

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// fakeTelegramAPI serves just enough of the real Bot API for
// NewBotAPIWithAPIEndpoint (which calls getMe to validate the token) and
// for SetWebhook/RemoveWebhook to succeed, without ever talking to the
// real api.telegram.org.
func fakeTelegramAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"TestBot"}}`))
		case strings.HasSuffix(r.URL.Path, "/setWebhook"), strings.HasSuffix(r.URL.Path, "/deleteWebhook"):
			w.Write([]byte(`{"ok":true,"result":true}`))
		default:
			w.Write([]byte(`{"ok":false,"description":"unhandled endpoint in test fake"}`))
		}
	}))
}

func newFakeBotAPI(t *testing.T, server *httptest.Server) *tgbotapi.BotAPI {
	t.Helper()
	api, err := tgbotapi.NewBotAPIWithAPIEndpoint("test-token", server.URL+"/bot%s/%s")
	if err != nil {
		t.Fatalf("NewBotAPIWithAPIEndpoint: %v", err)
	}
	return api
}

func TestSetWebhook_SendsURLAndSecretToken(t *testing.T) {
	var capturedForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test"}}`))
		case strings.HasSuffix(r.URL.Path, "/setWebhook"):
			r.ParseForm()
			capturedForm = r.Form
			w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	defer server.Close()

	api := newFakeBotAPI(t, server)
	if err := SetWebhook(api, "https://example.com/api/telegram/webhook", "shh-secret"); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}

	if got := capturedForm.Get("url"); got != "https://example.com/api/telegram/webhook" {
		t.Errorf("url = %q, want the webhook URL", got)
	}
	if got := capturedForm.Get("secret_token"); got != "shh-secret" {
		t.Errorf("secret_token = %q, want %q", got, "shh-secret")
	}
}

func TestSetWebhook_OmitsSecretTokenWhenEmpty(t *testing.T) {
	var capturedForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test"}}`))
		case strings.HasSuffix(r.URL.Path, "/setWebhook"):
			r.ParseForm()
			capturedForm = r.Form
			w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	defer server.Close()

	api := newFakeBotAPI(t, server)
	if err := SetWebhook(api, "https://example.com/hook", ""); err != nil {
		t.Fatalf("SetWebhook: %v", err)
	}

	if capturedForm.Has("secret_token") {
		t.Errorf("secret_token present = %v, want omitted when empty", capturedForm.Get("secret_token"))
	}
}

func TestRemoveWebhook_Succeeds(t *testing.T) {
	server := fakeTelegramAPI(t)
	defer server.Close()

	api := newFakeBotAPI(t, server)
	if err := RemoveWebhook(api); err != nil {
		t.Fatalf("RemoveWebhook: %v", err)
	}
}

func TestWebhookHandler_RejectsWrongSecret(t *testing.T) {
	b, _, _ := newTestBot(t, 42)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(`{}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	rec := httptest.NewRecorder()

	b.WebhookHandler("correct-secret")(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestWebhookHandler_RejectsMissingSecret(t *testing.T) {
	b, _, _ := newTestBot(t, 42)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	b.WebhookHandler("correct-secret")(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestWebhookHandler_NoSecretConfiguredAcceptsAnyRequest(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)
	body := `{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":42,"type":"private"},"text":"Captured via webhook"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	b.WebhookHandler("")(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200", rec.Code, rec.Body.String())
	}
	if len(sender.messages()) != 1 {
		t.Errorf("sent = %+v, want 1 reply from the capture flow", sender.messages())
	}
}

func TestWebhookHandler_DispatchesToHandleUpdate(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)
	body := `{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":42,"type":"private"},"text":"/search banana"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "s3cret")
	rec := httptest.NewRecorder()

	b.WebhookHandler("s3cret")(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	msgs := sender.messages()
	if len(msgs) != 1 || msgs[0].Text != "No results." {
		t.Errorf("sent = %+v, want a /search dispatch reply", msgs)
	}
}

func TestWebhookHandler_InvalidBodyIsBadRequest(t *testing.T) {
	b, _, _ := newTestBot(t, 42)
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	b.WebhookHandler("")(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
