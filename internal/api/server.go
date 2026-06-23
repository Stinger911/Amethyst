package api

import (
	"net/http"
	"strings"

	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/notify"
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
// hub, if non-nil, serves GET /api/ws (plan_amethyst-web-ui §5): pushes a
// {"path": "..."} event to every connected client whenever the watcher
// reindexes a genuinely external change. Pass nil to omit the route
// (tests that don't exercise it).
//
// webhook, if non-nil, serves POST /api/telegram/webhook for
// TELEGRAM_MODE=webhook (plan_amethyst-telegram-bot §1/§5 step 6) — not
// behind RequireAuth, since Telegram's servers don't carry our session
// cookie; the handler itself (internal/bot's WebhookHandler) checks the
// secret token header instead.
func NewServer(db *index.DB, telegram TelegramConfig, write WriteConfig, hub *notify.Hub, webhook http.HandlerFunc, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", RequireAuth(db, SearchHandler(db)))
	mux.HandleFunc("GET /api/notes", RequireAuth(db, NotesListHandler(db)))
	mux.HandleFunc("GET /api/notes/{path...}", RequireAuth(db, NoteHandler(db, write.VaultRoot)))
	mux.HandleFunc("PUT /api/notes/{path...}", RequireAuth(db, SaveNoteHandler(db, write)))
	mux.HandleFunc("GET /api/graph", RequireAuth(db, GraphHandler(db)))
	mux.HandleFunc("GET /api/settings", RequireAuth(db, GetSettingsHandler(db)))
	mux.HandleFunc("PUT /api/settings", RequireAuth(db, SaveSettingsHandler(db)))
	mux.HandleFunc("GET /api/auth/config", AuthConfigHandler(db, telegram))
	mux.HandleFunc("POST /api/auth/login", LoginHandler(db))
	mux.HandleFunc("POST /api/auth/logout", LogoutHandler(db))
	mux.HandleFunc("GET /api/auth/telegram/callback", TelegramCallbackHandler(db, telegram.BotToken, telegram.OwnerChatID))
	mux.HandleFunc("POST /api/telegram/pair", RequireAuth(db, PairHandler(db, telegram.BotUsername)))
	if hub != nil {
		mux.HandleFunc("GET /api/ws", RequireAuth(db, hub.Handler()))
	}
	if webhook != nil {
		mux.HandleFunc("POST /api/telegram/webhook", webhook)
	}
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
