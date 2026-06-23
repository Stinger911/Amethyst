package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/notify"
)

func TestWS_RequiresAuth(t *testing.T) {
	db := openAuthTestDB(t)
	hub := notify.NewHub()
	server := httptest.NewServer(NewServer(db, TelegramConfig{}, WriteConfig{}, hub, nil, nil))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws"

	_, resp, err := websocket.Dial(ctx, url, nil)
	if err == nil {
		t.Fatal("Dial succeeded without a session cookie, want a rejection")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWS_AuthenticatedClientReceivesBroadcast(t *testing.T) {
	db := openAuthTestDB(t)
	hub := notify.NewHub()
	server := httptest.NewServer(NewServer(db, TelegramConfig{}, WriteConfig{}, hub, nil, nil))
	defer server.Close()

	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws"

	header := http.Header{}
	header.Set("Cookie", auth.SessionCookieName+"="+token)
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	time.Sleep(20 * time.Millisecond)
	hub.Broadcast("Note.md")

	var ev notify.Event
	if err := wsjson.Read(ctx, conn, &ev); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Path != "Note.md" {
		t.Errorf("Path = %q, want %q", ev.Path, "Note.md")
	}
}

func TestWS_RouteOmittedWhenHubIsNil(t *testing.T) {
	db := openAuthTestDB(t)
	server := httptest.NewServer(NewServer(db, TelegramConfig{}, WriteConfig{}, nil, nil, nil))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws"

	_, resp, err := websocket.Dial(ctx, url, nil)
	if err == nil {
		t.Fatal("Dial succeeded against a server with no hub, want a rejection")
	}
	if resp != nil && resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
