// Package bot implements the Telegram bot from plan_amethyst-telegram-bot:
// capture (plain text -> Inbox or daily note), /search and /note. The
// owner's chat ID is env-configured for now (TELEGRAM_OWNER_CHAT_ID, same
// as the Login Widget in internal/auth/telegram.go) — the dynamic
// /start <token> pairing flow described in the plan is deferred until
// there's a Settings-page button to generate a token from.
package bot

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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
	DB          *index.DB
	VaultRoot   string
	Watcher     *watch.Watcher
	OwnerChatID int64 // 0 means "not configured" — see plan §2.
	Sender      Sender
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
func (b *Bot) HandleUpdate(msg IncomingMessage) {
	if b.OwnerChatID == 0 || msg.ChatID != b.OwnerChatID {
		return
	}

	text := strings.TrimSpace(msg.Text)
	switch {
	case text == "":
		return
	case matchesCommand(text, "/search"):
		b.handleSearch(msg.ChatID, strings.TrimSpace(stripCommand(text, "/search")))
	case matchesCommand(text, "/note"):
		b.handleNote(msg.ChatID, strings.TrimSpace(stripCommand(text, "/note")))
	case matchesCommand(text, "/start"):
		// Pairing is deferred (see package doc) — per plan §4, no
		// argument or an invalid token is silently ignored.
		return
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
