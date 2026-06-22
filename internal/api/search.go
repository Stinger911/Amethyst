// Package api exposes the Go-rendered HTTP surface described in
// plan_amethyst-web-ui (REST endpoints + later a WebSocket). This file
// implements GET /api/search.
package api

import (
	"encoding/json"
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
// The query-building and SQL live in internal/index (index.Search), shared
// with the Telegram bot's /search command.
func SearchHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, http.StatusOK, SearchResponse{Query: q, Results: []SearchResult{}})
			return
		}

		limit := parseLimit(r.URL.Query().Get("limit"))

		matchQuery, err := index.ToFTSQuery(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		results, err := index.Search(db, matchQuery, limit, "<mark>", "</mark>")
		if err != nil {
			log.Printf("search %q: %v", q, err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, SearchResponse{Query: q, Results: toAPIResults(results)})
	}
}

func toAPIResults(results []index.SearchResult) []SearchResult {
	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{Path: r.Path, Title: r.Title, Snippet: r.Snippet}
	}
	return out
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
