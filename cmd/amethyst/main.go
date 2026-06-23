// Command amethyst runs the core server: it cold-scans a vault into the
// SQLite index, keeps the index up to date via a file watcher, and serves
// the REST API over HTTP. See plan_amethyst-mvp Фазы 0/1.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/Stinger911/Amethyst/internal/api"
	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/bot"
	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/notify"
	"github.com/Stinger911/Amethyst/internal/watch"
	"github.com/Stinger911/Amethyst/internal/webui"
)

type config struct {
	VaultPath             string
	IndexPath             string
	ListenAddr            string
	AdminPassword         string
	AdminPasswordReset    bool
	TelegramBotToken      string
	TelegramOwnerID       string
	TelegramBotName       string
	TelegramMode          string
	TelegramWebhookURL    string
	TelegramWebhookSecret string
}

func loadConfig() config {
	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		log.Fatal("VAULT_PATH is required (path to the Obsidian vault to open)")
	}
	return config{
		VaultPath:             vaultPath,
		IndexPath:             getenvDefault("INDEX_PATH", "data/index.db"),
		ListenAddr:            getenvDefault("LISTEN_ADDR", ":8080"),
		AdminPassword:         os.Getenv("ADMIN_PASSWORD"),
		AdminPasswordReset:    os.Getenv("ADMIN_PASSWORD_RESET") == "true",
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramOwnerID:       os.Getenv("TELEGRAM_OWNER_CHAT_ID"),
		TelegramBotName:       os.Getenv("TELEGRAM_BOT_USERNAME"),
		TelegramMode:          getenvDefault("TELEGRAM_MODE", "polling"),
		TelegramWebhookURL:    os.Getenv("TELEGRAM_WEBHOOK_URL"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startTelegramBot wires up internal/bot if TELEGRAM_BOT_TOKEN is set.
// Owner identity is env-configured (TELEGRAM_OWNER_CHAT_ID) if set,
// otherwise the bot resolves it dynamically via /start <token> pairing
// (see internal/bot's currentOwnerChatID and internal/auth's
// telegram_pairing.go) — no extra wiring needed here for that fallback,
// since both read straight from db.
//
// TELEGRAM_MODE (default "polling") picks the transport: "polling" runs
// Bot.Run in its own goroutine tracked by wg; "webhook" instead registers
// the webhook URL with Telegram and returns an http.HandlerFunc the
// caller must wire up as POST /api/telegram/webhook (see
// plan_amethyst-telegram-bot §1/§5 step 6) — additive and lowest priority,
// most self-hosted deploys have no public HTTPS endpoint to use it on.
func startTelegramBot(ctx context.Context, wg *sync.WaitGroup, cfg config, db *index.DB, w *watch.Watcher) http.HandlerFunc {
	if cfg.TelegramBotToken == "" {
		return nil
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("telegram bot: %v", err)
	}

	var ownerChatID int64
	if cfg.TelegramOwnerID != "" {
		ownerChatID, err = strconv.ParseInt(cfg.TelegramOwnerID, 10, 64)
		if err != nil {
			log.Fatalf("TELEGRAM_OWNER_CHAT_ID: %v", err)
		}
	}
	if ownerChatID == 0 {
		log.Printf("telegram bot: TELEGRAM_OWNER_CHAT_ID not set, bot will ignore all messages until paired (see /settings)")
	}

	tgBot := &bot.Bot{
		DB:          db,
		VaultRoot:   cfg.VaultPath,
		Watcher:     w,
		OwnerChatID: ownerChatID,
		Sender:      bot.NewTelegramSender(botAPI),
	}

	if cfg.TelegramMode == "webhook" {
		if cfg.TelegramWebhookURL == "" {
			log.Fatal("TELEGRAM_MODE=webhook requires TELEGRAM_WEBHOOK_URL")
		}
		if err := bot.SetWebhook(botAPI, cfg.TelegramWebhookURL, cfg.TelegramWebhookSecret); err != nil {
			log.Fatalf("telegram set webhook: %v", err)
		}
		log.Printf("telegram bot: webhook mode, registered %s", cfg.TelegramWebhookURL)
		return tgBot.WebhookHandler(cfg.TelegramWebhookSecret)
	}

	// Telegram refuses GetUpdates while a webhook is still registered, so
	// clear one left over from a previous run in webhook mode.
	if err := bot.RemoveWebhook(botAPI); err != nil {
		log.Printf("telegram bot: remove webhook: %v", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tgBot.Run(ctx, botAPI); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("telegram bot stopped: %v", err)
		}
	}()
	return nil
}

func main() {
	cfg := loadConfig()

	// The index lives outside the vault on purpose (plan_amethyst-storage-index
	// §2) — it's a disposable cache, not vault content.
	if err := os.MkdirAll(filepath.Dir(cfg.IndexPath), 0o755); err != nil {
		log.Fatalf("create index dir: %v", err)
	}

	db, err := index.Open(cfg.IndexPath)
	if err != nil {
		log.Fatalf("open index: %v", err)
	}
	defer db.Close()

	// Password fallback is the only working login method until the
	// Telegram Login Widget exists, so an admin credential is mandatory
	// from the very first run (plan_amethyst-mvp Фаза 2).
	if err := auth.EnsureCredential(db, cfg.AdminPassword, cfg.AdminPasswordReset); err != nil {
		log.Fatalf("admin credential: %v", err)
	}

	log.Printf("cold scan: %s", cfg.VaultPath)
	stats, err := index.ColdScan(db, cfg.VaultPath)
	if err != nil {
		log.Fatalf("cold scan: %v", err)
	}
	log.Printf("indexed %d files (%d notes, %d links, %d tags)", stats.Files, stats.Notes, stats.Links, stats.Tags)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hub := notify.NewHub()

	w, err := watch.New(cfg.VaultPath, db)
	if err != nil {
		log.Fatalf("start watcher: %v", err)
	}
	defer w.Close()
	w.OnEvent = func(ev watch.Event) {
		if ev.Err != nil {
			log.Printf("watch %s: %v", ev.Path, ev.Err)
			return
		}
		// Suppressed (self-write) events never reach OnEvent at all (see
		// Watcher.consumeSuppressed) — anything that does is a genuinely
		// external change, exactly what plan_amethyst-web-ui §5 wants to
		// push to connected browser tabs.
		if ev.Path != "" {
			hub.Broadcast(ev.Path)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("watcher stopped: %v", err)
		}
	}()

	webhookHandler := startTelegramBot(ctx, &wg, cfg, db, w)

	staticHandler, err := webui.Handler()
	if err != nil {
		log.Fatalf("embedded frontend: %v", err)
	}
	if staticHandler == nil {
		log.Printf("frontend not built into this binary; serving API only (run `make build-frontend` to include it)")
	}

	server := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: api.NewServer(db, api.TelegramConfig{
			BotToken:    cfg.TelegramBotToken,
			OwnerChatID: cfg.TelegramOwnerID,
			BotUsername: cfg.TelegramBotName,
		}, api.WriteConfig{
			VaultRoot: cfg.VaultPath,
			Watcher:   w,
		}, hub, webhookHandler, staticHandler),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}

	wg.Wait()
}
