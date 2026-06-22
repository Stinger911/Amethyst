package index

import (
	"errors"
	"strings"
)

// SearchResult is one FTS5 match, ranked best-first by bm25.
type SearchResult struct {
	Path    string
	Title   string
	Snippet string
}

// Search runs matchQuery (build one with ToFTSQuery) against notes_fts and
// returns up to limit results, best match first. snippetOpen/snippetClose
// wrap the matched terms within Snippet — callers that don't need
// highlighting (e.g. a plain-text Telegram reply) can pass "", "".
func Search(db *DB, matchQuery string, limit int, snippetOpen, snippetClose string) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT notes.path, notes.title,
		       snippet(notes_fts, 1, ?, ?, '…', 12)
		FROM notes_fts
		JOIN notes ON notes.id = notes_fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, snippetOpen, snippetClose, matchQuery, limit)
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

// ToFTSQuery turns free-text user input into an FTS5 MATCH expression
// that can't blow up on the query-language metacharacters FTS5 gives
// special meaning (AND/OR/NOT, column filters, NEAR, unbalanced quotes,
// hyphens, colons, ...). Each whitespace-separated term is quoted as a
// literal phrase and implicitly AND-ed together — plain multi-word
// search, not the full FTS5 query language.
func ToFTSQuery(q string) (string, error) {
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
