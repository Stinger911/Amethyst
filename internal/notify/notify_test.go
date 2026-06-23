package notify

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestHub_BroadcastReachesConnectedClient(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	// Give the server a moment to register the client before broadcasting,
	// since registration happens asynchronously in the hub's run loop.
	time.Sleep(20 * time.Millisecond)
	hub.Broadcast("Note.md")

	var ev Event
	if err := wsjson.Read(ctx, conn, &ev); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if ev.Path != "Note.md" {
		t.Errorf("Path = %q, want %q", ev.Path, "Note.md")
	}
}

func TestHub_MultipleClientsAllReceiveBroadcast(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(server.URL, "http")

	conn1, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial 1: %v", err)
	}
	defer conn1.CloseNow()
	conn2, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Dial 2: %v", err)
	}
	defer conn2.CloseNow()

	time.Sleep(20 * time.Millisecond)
	hub.Broadcast("Shared.md")

	for i, conn := range []*websocket.Conn{conn1, conn2} {
		var ev Event
		if err := wsjson.Read(ctx, conn, &ev); err != nil {
			t.Fatalf("Read client %d: %v", i, err)
		}
		if ev.Path != "Shared.md" {
			t.Errorf("client %d: Path = %q, want %q", i, ev.Path, "Shared.md")
		}
	}
}

func TestHub_BroadcastWithNoClientsDoesNotBlock(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Broadcast("Nobody.md")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcast blocked with no connected clients")
	}
}
