# Amethyst

A self-hosted, single-user, Obsidian-compatible knowledge system with a web UI and a Telegram bot.

Amethyst is not a replacement for Obsidian — it's a remote control for it. It opens your existing real Obsidian vault (plain `.md` files, frontmatter, wiki-links, embeds, block refs — as-is, no migration) and gives you access to it from a browser and from Telegram, while you keep editing the same vault in desktop Obsidian. Changes sync both ways.

**Status:** early development, not usable yet.

## Why not just use Blinko / Notion / etc.?

Most self-hosted note tools with a web + Telegram interface replace your notes with their own storage format. Amethyst doesn't — it's designed to extend an existing Obsidian vault, not migrate you away from one.

## Planned stack

- Backend: Go
- Frontend: React
- Storage: the vault stays plain markdown files on disk; a SQLite (FTS5) index is a disposable, rebuildable cache for search/links/tags — never the source of truth
- Auth: Telegram login (primary) + password fallback
- Optional add-on: local AI search via an Ollama sidecar

## Install

Not ready yet — there's no released MVP, so treat the steps below as a build-from-source dev/preview setup, not a stable install path.

### Docker (recommended once running this for real)

```sh
cp .env.example .env   # then edit VAULT_HOST_PATH, ADMIN_PASSWORD, etc.
docker compose up --build
```

This builds the frontend, embeds it into the Go binary, and serves both the API and the UI from one container on `:8080`.

### From source

Requires Go 1.26+ and Node 22+.

```sh
make build   # builds web/ with vite, then the amethyst binary (bin/amethyst)
VAULT_PATH=/path/to/your/vault ADMIN_PASSWORD=change-me ./bin/amethyst
```

Other Makefile targets: `make test` (Go + frontend tests), `make lint` (`go vet` + eslint), `make run` (build + run), `make docker-build`.

During day-to-day frontend development, run `cd web && npm run dev` instead — its Vite dev server proxies `/api` to a separately-running `go run ./cmd/amethyst` on `:8080`, so you get hot reload instead of rebuilding the embedded binary on every change.

## License

[Business Source License 1.1](./LICENSE-BSL.txt). Personal, educational, and internal non-commercial self-hosted use is free and unrestricted. Commercial use (including offering Amethyst, or a derivative, to third parties on a hosted/managed basis) requires a commercial license — contact office@lab18.net. Four years after each version is published, that version automatically converts to Apache License 2.0.
