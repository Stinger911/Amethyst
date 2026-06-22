package bot

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Stinger911/Amethyst/internal/index"
)

type sentMessage struct {
	ChatID int64
	Text   string
}

type fakeSender struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (f *fakeSender) Send(chatID int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentMessage{ChatID: chatID, Text: text})
	return nil
}

func (f *fakeSender) messages() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentMessage(nil), f.sent...)
}

func newTestBot(t *testing.T, ownerChatID int64) (*Bot, *fakeSender, string) {
	t.Helper()
	root := t.TempDir()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	sender := &fakeSender{}
	b := &Bot{
		DB:          db,
		VaultRoot:   root,
		OwnerChatID: ownerChatID,
		Sender:      sender,
	}
	return b, sender, root
}

func writeVaultFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestHandleUpdate_IgnoresNonOwnerChat(t *testing.T) {
	b, sender, root := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 99, Text: "Buy milk"})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply to a non-owner chat", sender.messages())
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Errorf("vault root has %d entries, want 0 (no capture for a non-owner)", len(entries))
	}
}

func TestHandleUpdate_IgnoresEverythingWhenOwnerNotConfigured(t *testing.T) {
	b, sender, _ := newTestBot(t, 0)

	b.HandleUpdate(IncomingMessage{ChatID: 12345, Text: "Buy milk"})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply when no owner is configured", sender.messages())
	}
}

func TestHandleUpdate_EmptyTextIsIgnored(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "   "})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply for blank text", sender.messages())
	}
}

func TestHandleUpdate_UnknownSlashCommandIsIgnoredSilently(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/bogus"})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply for an unrecognized command", sender.messages())
	}
}

func TestHandleUpdate_StartWithoutTokenIsIgnoredSilently(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/start"})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply for /start (pairing not implemented yet)", sender.messages())
	}
}

func TestMatchesCommand(t *testing.T) {
	cases := []struct {
		text, cmd string
		want      bool
	}{
		{"/search foo", "/search", true},
		{"/search", "/search", true},
		{"/search@AmethystBot foo", "/search", true},
		{"/searching foo", "/search", false},
		{"plain text", "/search", false},
	}
	for _, c := range cases {
		if got := matchesCommand(c.text, c.cmd); got != c.want {
			t.Errorf("matchesCommand(%q, %q) = %v, want %v", c.text, c.cmd, got, c.want)
		}
	}
}

func TestStripCommand(t *testing.T) {
	cases := []struct {
		text, cmd, want string
	}{
		{"/search foo bar", "/search", " foo bar"},
		{"/search", "/search", ""},
		{"/search@AmethystBot foo", "/search", "foo"},
		{"/search@AmethystBot", "/search", ""},
	}
	for _, c := range cases {
		if got := stripCommand(c.text, c.cmd); got != c.want {
			t.Errorf("stripCommand(%q, %q) = %q, want %q", c.text, c.cmd, got, c.want)
		}
	}
}
