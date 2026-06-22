package api

import (
	"net/http"
	"strings"

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
//
// write provides the vault root and watcher that GET (raw+hash) and PUT
// (save) need for the write path (plan_amethyst-mvp Фаза 3); its zero
// value (empty VaultRoot, nil Watcher) is fine for read-only tests that
// never exercise those fields.
//
// static, if non-nil, serves the built frontend (see internal/webui) for
// every path not claimed by an /api/ route, so a single binary can serve
// both the API and the UI. Pass nil to run API-only (tests, or local dev
// against a separate Vite dev server).
func NewServer(db *index.DB, telegram TelegramConfig, write WriteConfig, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", RequireAuth(db, SearchHandler(db)))
	mux.HandleFunc("GET /api/notes", RequireAuth(db, NotesListHandler(db)))
	mux.HandleFunc("GET /api/notes/{path...}", RequireAuth(db, NoteHandler(db, write.VaultRoot)))
	mux.HandleFunc("PUT /api/notes/{path...}", RequireAuth(db, SaveNoteHandler(db, write)))
	mux.HandleFunc("GET /api/graph", RequireAuth(db, GraphHandler(db)))
	mux.HandleFunc("GET /api/auth/config", AuthConfigHandler(telegram))
	mux.HandleFunc("POST /api/auth/login", LoginHandler(db))
	mux.HandleFunc("POST /api/auth/logout", LogoutHandler(db))
	mux.HandleFunc("GET /api/auth/telegram/callback", TelegramCallbackHandler(db, telegram.BotToken, telegram.OwnerChatID))
	if static != nil {
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			static.ServeHTTP(w, r)
		}))
	}
	return mux
}
