// Package bot implements the Telegram bot from plan_amethyst-telegram-bot:
// capture (plain text -> Inbox or daily note), /search, /note, and
// /start <token> pairing. The owner's chat ID is env-configured
// (TELEGRAM_OWNER_CHAT_ID) if set — same as the Login Widget in
// internal/auth/telegram.go — otherwise whichever chat last completed the
// /start <token> pairing flow (internal/auth's telegram_pairing.go), kicked
// off from the Settings page's "Link Telegram" button.
package bot

import (
	"context"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/watch"
)

// IncomingMessage is the subset of a Telegram update HandleUpdate acts on,
// decoupled from tgbotapi's types so the dispatch logic is testable
// without a real bot connection.
type IncomingMessage struct {
	ChatID int64
	Text   string
}

// Sender delivers a reply. The real implementation (NewTelegramSender)
// wraps a *tgbotapi.BotAPI; tests can substitute a fake.
type Sender interface {
	Send(chatID int64, text string) error
}

// Bot holds everything the command/capture handlers need.
type Bot struct {
	DB        *index.DB
	VaultRoot string
	Watcher   *watch.Watcher
	// OwnerChatID is the env-configured owner (TELEGRAM_OWNER_CHAT_ID). 0
	// means "not set via env" — currentOwnerChatID then falls back to
	// whichever chat last completed /start <token> pairing, persisted in DB.
	OwnerChatID int64
	Sender      Sender
}

// currentOwnerChatID resolves the live owner: env wins if set, otherwise
// the dynamically-paired owner (0 if neither is set yet).
func (b *Bot) currentOwnerChatID() int64 {
	if b.OwnerChatID != 0 {
		return b.OwnerChatID
	}
	chatIDStr, err := auth.GetTelegramOwnerChatID(b.DB)
	if err != nil || chatIDStr == "" {
		return 0
	}
	id, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// Run polls api for updates and dispatches them until ctx is cancelled.
func (b *Bot) Run(ctx context.Context, api *tgbotapi.BotAPI) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			api.StopReceivingUpdates()
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil || update.Message.Chat == nil {
				continue
			}
			b.HandleUpdate(IncomingMessage{
				ChatID: update.Message.Chat.ID,
				Text:   update.Message.Text,
			})
		}
	}
}

// HandleUpdate applies the auth gate (plan §2: silently ignore anyone but
// the configured owner) and dispatches commands vs. plain-text capture.
// /start is checked before the gate: its own security is the pairing
// token, not the sender's chat ID — that's how an unpaired bot acquires
// its first owner at all (plan §3).
func (b *Bot) HandleUpdate(msg IncomingMessage) {
	text := strings.TrimSpace(msg.Text)

	if matchesCommand(text, "/start") {
		b.handleStart(msg.ChatID, strings.TrimSpace(stripCommand(text, "/start")))
		return
	}

	owner := b.currentOwnerChatID()
	if owner == 0 || msg.ChatID != owner {
		return
	}

	switch {
	case text == "":
		return
	case matchesCommand(text, "/search"):
		b.handleSearch(msg.ChatID, strings.TrimSpace(stripCommand(text, "/search")))
	case matchesCommand(text, "/note"):
		b.handleNote(msg.ChatID, strings.TrimSpace(stripCommand(text, "/note")))
	case strings.HasPrefix(text, "/"):
		return
	default:
		b.handleCapture(msg.ChatID, text)
	}
}

// matchesCommand reports whether text starts with cmd, also accepting the
// "/cmd@BotName" form Telegram uses when a bot is added to a group.
func matchesCommand(text, cmd string) bool {
	if !strings.HasPrefix(text, cmd) {
		return false
	}
	rest := text[len(cmd):]
	return rest == "" || rest[0] == ' ' || rest[0] == '@'
}

func stripCommand(text, cmd string) string {
	rest := text[len(cmd):]
	if !strings.HasPrefix(rest, "@") {
		return rest
	}
	i := strings.IndexByte(rest, ' ')
	if i == -1 {
		return ""
	}
	return rest[i+1:]
}

func (b *Bot) reply(chatID int64, text string) {
	if b.Sender == nil {
		return
	}
	if err := b.Sender.Send(chatID, text); err != nil {
		log.Printf("telegram reply: %v", err)
	}
}
