package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Stinger911/Amethyst/internal/settings"
)

func TestHandleCapture_DefaultsToInbox(t *testing.T) {
	b, sender, root := newTestBot(t, 42)

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "Buy milk"})

	entries, err := os.ReadDir(filepath.Join(root, "Inbox"))
	if err != nil {
		t.Fatalf("read Inbox dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Inbox has %d entries, want 1", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(root, "Inbox", entries[0].Name()))
	if err != nil {
		t.Fatalf("read captured file: %v", err)
	}
	if string(content) != "Buy milk\n" {
		t.Errorf("content = %q, want %q", content, "Buy milk\n")
	}

	msgs := sender.messages()
	if len(msgs) != 1 || !strings.HasPrefix(msgs[0].Text, "Saved to Inbox/") {
		t.Errorf("sent = %+v, want one \"Saved to Inbox/...\" reply", msgs)
	}
}

func TestHandleCapture_TwoMessagesInTheSameSecondGetSeparateFiles(t *testing.T) {
	b, _, root := newTestBot(t, 42)

	// Both captures race to the same once-per-second timestamp filename;
	// captureToInbox must disambiguate rather than letting one overwrite
	// the other.
	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "First"})
	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "Second"})

	entries, err := os.ReadDir(filepath.Join(root, "Inbox"))
	if err != nil {
		t.Fatalf("read Inbox dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Inbox has %d entries, want 2: %v", len(entries), entries)
	}

	var contents []string
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(root, "Inbox", e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		contents = append(contents, string(b))
	}
	if !(strings.Contains(contents[0]+contents[1], "First") && strings.Contains(contents[0]+contents[1], "Second")) {
		t.Errorf("contents = %v, want both First and Second preserved", contents)
	}
}

func TestHandleCapture_DailyModeAppendsToTodaysNote(t *testing.T) {
	b, sender, root := newTestBot(t, 42)
	if err := settings.SetCaptureMode(b.DB, settings.CaptureModeDaily); err != nil {
		t.Fatalf("SetCaptureMode: %v", err)
	}

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "First capture"})
	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "Second capture"})

	today := time.Now().Format("2006-01-02")
	content, err := os.ReadFile(filepath.Join(root, "Daily", today+".md"))
	if err != nil {
		t.Fatalf("read daily note: %v", err)
	}
	if !strings.Contains(string(content), "First capture") || !strings.Contains(string(content), "Second capture") {
		t.Errorf("daily note = %q, want both captures appended", content)
	}
	if !strings.HasPrefix(string(content), "# "+today) {
		t.Errorf("daily note = %q, want it to start with a heading for %s", content, today)
	}

	msgs := sender.messages()
	if len(msgs) != 2 {
		t.Fatalf("sent = %+v, want 2 replies", msgs)
	}
	wantPath := "Saved to Daily/" + today + ".md"
	if msgs[0].Text != wantPath || msgs[1].Text != wantPath {
		t.Errorf("sent = %+v, want both replies to say %q", msgs, wantPath)
	}
}

func TestHandleCapture_DailyModePreservesExistingContent(t *testing.T) {
	b, _, root := newTestBot(t, 42)
	if err := settings.SetCaptureMode(b.DB, settings.CaptureModeDaily); err != nil {
		t.Fatalf("SetCaptureMode: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	writeVaultFile(t, root, "Daily/"+today+".md", "# "+today+"\n\nManually written note from Obsidian.\n\n")

	b.HandleUpdate(IncomingMessage{ChatID: 42, Text: "Captured via Telegram"})

	content, err := os.ReadFile(filepath.Join(root, "Daily", today+".md"))
	if err != nil {
		t.Fatalf("read daily note: %v", err)
	}
	if !strings.Contains(string(content), "Manually written note from Obsidian.") {
		t.Errorf("daily note = %q, want pre-existing content preserved", content)
	}
	if !strings.Contains(string(content), "Captured via Telegram") {
		t.Errorf("daily note = %q, want the new capture appended", content)
	}
}
