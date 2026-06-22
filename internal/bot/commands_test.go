package bot

import (
	"strings"
	"testing"

	"github.com/Stinger911/Amethyst/internal/index"
)

func seedSearchableNotes(t *testing.T, b *Bot, root string) {
	t.Helper()
	writeVaultFile(t, root, "Apple.md", "# Apple\n\nApple pie is a dessert made with apples.\n")
	writeVaultFile(t, root, "Banana.md", "# Banana\n\nBanana bread is a dessert made with bananas.\n")
	if _, err := index.ColdScan(b.DB, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}
}

func TestHandleSearch_NoQueryShowsUsage(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/search"})

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "Usage") {
		t.Errorf("sent = %+v, want a usage message", msgs)
	}
}

func TestHandleSearch_ReturnsMatchingNote(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/search banana"})

	msgs := sender.messages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %+v, want 1 reply", msgs)
	}
	if !strings.Contains(msgs[0].Text, "Banana") || !strings.Contains(msgs[0].Text, "Banana.md") {
		t.Errorf("reply = %q, want it to mention Banana / Banana.md", msgs[0].Text)
	}
	if strings.Contains(msgs[0].Text, "Apple") {
		t.Errorf("reply = %q, want it to NOT mention Apple", msgs[0].Text)
	}
}

func TestHandleSearch_NoMatchesSaysSo(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/search nonexistentterm"})

	msgs := sender.messages()
	if len(msgs) != 1 || msgs[0].Text != "No results." {
		t.Errorf("sent = %+v, want a single \"No results.\" reply", msgs)
	}
}

func TestHandleSearch_SupportsBotUsernameSuffix(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/search@AmethystBot banana"})

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "Banana.md") {
		t.Errorf("sent = %+v, want a Banana.md match", msgs)
	}
}

func TestHandleNote_NoTitleShowsUsage(t *testing.T) {
	b, sender, _ := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/note"})

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "Usage") {
		t.Errorf("sent = %+v, want a usage message", msgs)
	}
}

func TestHandleNote_ExactTitleMatch(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/note Banana"})

	msgs := sender.messages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %+v, want 1 reply", msgs)
	}
	if !strings.HasPrefix(msgs[0].Text, "Banana") || !strings.Contains(msgs[0].Text, "Banana bread") {
		t.Errorf("reply = %q, want the Banana note's body", msgs[0].Text)
	}
}

func TestHandleNote_PartialTitleMatch(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/note Ban"})

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "Banana bread") {
		t.Errorf("sent = %+v, want the Banana note via partial match", msgs)
	}
}

func TestHandleNote_NoMatchSaysSo(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	seedSearchableNotes(t, b, root)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/note Nonexistent"})

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Text, "No note found") {
		t.Errorf("sent = %+v, want a \"No note found\" reply", msgs)
	}
}

func TestHandleNote_TruncatesLongBody(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	longBody := strings.Repeat("word ", 1000)
	writeVaultFile(t, root, "Long.md", "# Long\n\n"+longBody)
	if _, err := index.ColdScan(b.DB, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "/note Long"})

	msgs := sender.messages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %+v, want 1 reply", msgs)
	}
	if len(msgs[0].Text) >= len(longBody) {
		t.Errorf("reply length = %d, want it truncated below the full body length %d", len(msgs[0].Text), len(longBody))
	}
	if !strings.Contains(msgs[0].Text, "truncated") {
		t.Errorf("reply = %q, want a truncation notice", msgs[0].Text)
	}
}
