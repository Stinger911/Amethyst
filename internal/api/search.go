// Package api exposes the Go-rendered HTTP surface described in
// plan_amethyst-web-ui (REST endpoints + later a WebSocket). This file
// implements GET /api/search.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Stinger911/Amethyst/internal/index"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 100
)

// SearchResult is one match, ranked best-first by FTS5's bm25 score.
type SearchResult struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// SearchResponse is the JSON body of GET /api/search.
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// SearchHandler serves GET /api/search?q=...&limit=... against notes_fts.
func SearchHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, http.StatusOK, SearchResponse{Query: q, Results: []SearchResult{}})
			return
		}

		limit := parseLimit(r.URL.Query().Get("limit"))

		matchQuery, err := toFTSQuery(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		results, err := runSearch(db, matchQuery, limit)
		if err != nil {
			log.Printf("search %q: %v", q, err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, SearchResponse{Query: q, Results: results})
	}
}

func runSearch(db *index.DB, matchQuery string, limit int) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT notes.path, notes.title,
		       snippet(notes_fts, 1, '<mark>', '</mark>', '…', 12)
		FROM notes_fts
		JOIN notes ON notes.id = notes_fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, matchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var res SearchResult
		if err := rows.Scan(&res.Path, &res.Title, &res.Snippet); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

func parseLimit(raw string) int {
	if raw == "" {
		return defaultSearchLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultSearchLimit
	}
	if n > maxSearchLimit {
		return maxSearchLimit
	}
	return n
}

// toFTSQuery turns free-text user input into an FTS5 MATCH expression
// that can't blow up on the query-language metacharacters FTS5 gives
// special meaning (AND/OR/NOT, column filters, NEAR, unbalanced quotes,
// hyphens, colons, ...). Each whitespace-separated term is quoted as a
// literal phrase and implicitly AND-ed together — plain multi-word
// search, not the full FTS5 query language.
func toFTSQuery(q string) (string, error) {
	fields := strings.Fields(q)
	if len(fields) == 0 {
		return "", errors.New("empty query")
	}
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
	}
	return strings.Join(quoted, " "), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
