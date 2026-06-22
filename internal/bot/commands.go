// This file implements the /search and /note commands from
// plan_amethyst-telegram-bot §4.
package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Stinger911/Amethyst/internal/index"
)

const searchResultLimit = 5

func (b *Bot) handleSearch(chatID int64, query string) {
	if query == "" {
		b.reply(chatID, "Usage: /search <query>")
		return
	}

	matchQuery, err := index.ToFTSQuery(query)
	if err != nil {
		b.reply(chatID, "Usage: /search <query>")
		return
	}

	results, err := index.Search(b.DB, matchQuery, searchResultLimit, "", "")
	if err != nil {
		log.Printf("telegram /search %q: %v", query, err)
		b.reply(chatID, "Search failed.")
		return
	}
	if len(results) == 0 {
		b.reply(chatID, "No results.")
		return
	}

	var sb strings.Builder
	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		fmt.Fprintf(&sb, "%s (%s)\n%s", r.Title, r.Path, r.Snippet)
	}
	b.reply(chatID, sb.String())
}

// telegramMessageBudget leaves headroom under Telegram's 4096-character
// message limit for the title line and truncation notice this prepends.
const telegramMessageBudget = 3800

func (b *Bot) handleNote(chatID int64, titleQuery string) {
	if titleQuery == "" {
		b.reply(chatID, "Usage: /note <title>")
		return
	}

	title, body, err := findNoteByTitle(b.DB, titleQuery)
	if errors.Is(err, sql.ErrNoRows) {
		b.reply(chatID, fmt.Sprintf("No note found matching %q.", titleQuery))
		return
	}
	if err != nil {
		log.Printf("telegram /note %q: %v", titleQuery, err)
		b.reply(chatID, "Lookup failed.")
		return
	}

	if len(body) > telegramMessageBudget {
		body = body[:telegramMessageBudget] + "\n\n… (truncated — open in the web UI for the full note)"
	}
	b.reply(chatID, fmt.Sprintf("%s\n\n%s", title, body))
}

// findNoteByTitle tries an exact (case-insensitive) title match first, then
// falls back to a substring match, returning the first hit by path for a
// deterministic result when several notes share a partial title.
func findNoteByTitle(db *index.DB, query string) (title, body string, err error) {
	err = db.QueryRow(
		`SELECT title, body FROM notes WHERE title = ? COLLATE NOCASE ORDER BY path LIMIT 1`,
		query,
	).Scan(&title, &body)
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
		return title, body, err
	}

	err = db.QueryRow(
		`SELECT title, body FROM notes WHERE title LIKE ? ESCAPE '\' COLLATE NOCASE ORDER BY path LIMIT 1`,
		"%"+escapeLikePattern(query)+"%",
	).Scan(&title, &body)
	return title, body, err
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
