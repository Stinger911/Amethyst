package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// telegramSender is the real Sender, used outside of tests.
type telegramSender struct {
	api *tgbotapi.BotAPI
}

// NewTelegramSender wraps api as a Sender for Bot.Sender.
func NewTelegramSender(api *tgbotapi.BotAPI) Sender {
	return &telegramSender{api: api}
}

func (s *telegramSender) Send(chatID int64, text string) error {
	_, err := s.api.Send(tgbotapi.NewMessage(chatID, text))
	return err
}
