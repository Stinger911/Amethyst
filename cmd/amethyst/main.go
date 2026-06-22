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
	"github.com/Stinger911/Amethyst/internal/watch"
	"github.com/Stinger911/Amethyst/internal/webui"
)

type config struct {
	VaultPath          string
	IndexPath          string
	ListenAddr         string
	AdminPassword      string
	AdminPasswordReset bool
	TelegramBotToken   string
	TelegramOwnerID    string
	TelegramBotName    string
}

func loadConfig() config {
	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		log.Fatal("VAULT_PATH is required (path to the Obsidian vault to open)")
	}
	return config{
		VaultPath:          vaultPath,
		IndexPath:          getenvDefault("INDEX_PATH", "data/index.db"),
		ListenAddr:         getenvDefault("LISTEN_ADDR", ":8080"),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		AdminPasswordReset: os.Getenv("ADMIN_PASSWORD_RESET") == "true",
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramOwnerID:    os.Getenv("TELEGRAM_OWNER_CHAT_ID"),
		TelegramBotName:    os.Getenv("TELEGRAM_BOT_USERNAME"),
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startTelegramBot wires up internal/bot if TELEGRAM_BOT_TOKEN is set,
// running its polling loop in its own goroutine tracked by wg. Owner
// identity is the same env var the Login Widget callback uses
// (TELEGRAM_OWNER_CHAT_ID) — see internal/bot's package doc for why the
// dynamic pairing flow isn't wired up yet.
func startTelegramBot(ctx context.Context, wg *sync.WaitGroup, cfg config, db *index.DB, w *watch.Watcher) {
	if cfg.TelegramBotToken == "" {
		return
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
		log.Printf("telegram bot: TELEGRAM_OWNER_CHAT_ID not set, bot will ignore all messages until it is")
	}

	tgBot := &bot.Bot{
		DB:          db,
		VaultRoot:   cfg.VaultPath,
		Watcher:     w,
		OwnerChatID: ownerChatID,
		Sender:      bot.NewTelegramSender(botAPI),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tgBot.Run(ctx, botAPI); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("telegram bot stopped: %v", err)
		}
	}()
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

	w, err := watch.New(cfg.VaultPath, db)
	if err != nil {
		log.Fatalf("start watcher: %v", err)
	}
	defer w.Close()
	w.OnEvent = func(ev watch.Event) {
		if ev.Err != nil {
			log.Printf("watch %s: %v", ev.Path, ev.Err)
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

	startTelegramBot(ctx, &wg, cfg, db, w)

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
		}, staticHandler),
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
