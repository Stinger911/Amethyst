package api

import (
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
)

// TelegramConfig holds the Login Widget settings. The zero value disables
// it: TelegramCallbackHandler responds 503 until both fields are set.
type TelegramConfig struct {
	BotToken    string
	OwnerChatID string
	// BotUsername is the bot's @handle (no leading @), used only by the
	// frontend to render the Login Widget's data-telegram-login attribute.
	// It carries no auth weight itself — VerifyTelegramWidgetData still
	// checks the signature against BotToken.
	BotUsername string
}

// NewServer wires up the Go REST surface from plan_amethyst-web-ui.
// Routes are added incrementally as each is implemented.
func NewServer(db *index.DB, telegram TelegramConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", RequireAuth(db, SearchHandler(db)))
	mux.HandleFunc("GET /api/notes", RequireAuth(db, NotesListHandler(db)))
	mux.HandleFunc("GET /api/notes/{path...}", RequireAuth(db, NoteHandler(db)))
	mux.HandleFunc("GET /api/graph", RequireAuth(db, GraphHandler(db)))
	mux.HandleFunc("GET /api/auth/config", AuthConfigHandler(telegram))
	mux.HandleFunc("POST /api/auth/login", LoginHandler(db))
	mux.HandleFunc("POST /api/auth/logout", LogoutHandler(db))
	mux.HandleFunc("GET /api/auth/telegram/callback", TelegramCallbackHandler(db, telegram.BotToken, telegram.OwnerChatID))
	return mux
}
