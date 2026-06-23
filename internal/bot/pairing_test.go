package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
)

func TestHandleStart_ValidTokenPairsAndReplies(t *testing.T) {
	b, sender, _ := newTestBot(t, 0) // no env owner configured yet
	token, _, err := auth.NewPairingToken(b.DB)
	if err != nil {
		t.Fatalf("NewPairingToken: %v", err)
	}

	b.HandleUpdate(IncomingMessage{ChatID: 777, Text: "/start " + token})

	owner, err := auth.GetTelegramOwnerChatID(b.DB)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "777" {
		t.Errorf("owner = %q, want %q", owner, "777")
	}

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "linked") {
		t.Errorf("sent = %+v, want a confirmation reply", msgs)
	}
}

func TestHandleStart_InvalidTokenIsIgnoredSilently(t *testing.T) {
	b, sender, _ := newTestBot(t, 0)

	b.HandleUpdate(IncomingMessage{ChatID: 777, Text: "/start not-a-real-token"})

	if len(sender.messages()) != 0 {
		t.Errorf("sent = %+v, want no reply for an invalid token", sender.messages())
	}
	owner, err := auth.GetTelegramOwnerChatID(b.DB)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "" {
		t.Errorf("owner = %q, want still unset", owner)
	}
}

func TestHandleStart_BypassesTheOwnerGate(t *testing.T) {
	// A bot with an existing owner (42) must still let a DIFFERENT chat
	// redeem a freshly generated token — /start's security is the token,
	// not the sender's chat ID (plan §3).
	b, _, _ := newTestBot(t, 42)
	token, _, err := auth.NewPairingToken(b.DB)
	if err != nil {
		t.Fatalf("NewPairingToken: %v", err)
	}

	b.HandleUpdate(IncomingMessage{ChatID: 999, Text: "/start " + token})

	owner, err := auth.GetTelegramOwnerChatID(b.DB)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "999" {
		t.Errorf("owner = %q, want %q (re-pairing should succeed)", owner, "999")
	}
}

func TestCurrentOwnerChatID_FallsBackToDynamicPairing(t *testing.T) {
	b, _, _ := newTestBot(t, 0) // no env owner
	if got := b.currentOwnerChatID(); got != 0 {
		t.Fatalf("currentOwnerChatID = %d, want 0 before pairing", got)
	}

	if err := auth.SetTelegramOwnerChatID(b.DB, "555"); err != nil {
		t.Fatalf("SetTelegramOwnerChatID: %v", err)
	}
	if got := b.currentOwnerChatID(); got != 555 {
		t.Errorf("currentOwnerChatID = %d, want 555 after pairing", got)
	}
}

func TestCurrentOwnerChatID_EnvTakesPrecedenceOverPairing(t *testing.T) {
	b, _, _ := newTestBot(t, 42) // env owner set
	if err := auth.SetTelegramOwnerChatID(b.DB, "555"); err != nil {
		t.Fatalf("SetTelegramOwnerChatID: %v", err)
	}

	if got := b.currentOwnerChatID(); got != 42 {
		t.Errorf("currentOwnerChatID = %d, want 42 (env should win over pairing)", got)
	}
}

func TestHandleUpdate_DynamicallyPairedOwnerCanUseCommands(t *testing.T) {
	b, sender, root := newTestBot(t, 0)
	if err := auth.SetTelegramOwnerChatID(b.DB, "777"); err != nil {
		t.Fatalf("SetTelegramOwnerChatID: %v", err)
	}

	b.HandleUpdate(IncomingMessage{ChatID: 777, Text: "Captured note"})

	entries, err := os.ReadDir(filepath.Join(root, "Inbox"))
	if err != nil {
		t.Fatalf("read Inbox dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Inbox entries = %d, want 1", len(entries))
	}
	if len(sender.messages()) != 1 {
		t.Errorf("sent = %+v, want 1 reply", sender.messages())
	}
}
