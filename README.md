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

Not ready yet. Once an MVP is available, this section will have a `docker-compose` quick start.

## License

[Business Source License 1.1](./LICENSE-BSL.txt). Personal, educational, and internal non-commercial self-hosted use is free and unrestricted. Commercial use (including offering Amethyst, or a derivative, to third parties on a hosted/managed basis) requires a commercial license — contact office@lab18.net. Four years after each version is published, that version automatically converts to Apache License 2.0.
