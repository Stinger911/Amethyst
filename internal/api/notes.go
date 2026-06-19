// This file implements GET /api/notes and GET /api/notes/{path...} from
// plan_amethyst-web-ui §2 (Phase 1: list + single-note view with rendered
// HTML, frontmatter, tags and backlinks).
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/render"
)

// NoteSummary is one entry in the GET /api/notes list.
type NoteSummary struct {
	Path  string   `json:"path"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// NotesListResponse is the JSON body of GET /api/notes.
type NotesListResponse struct {
	Notes []NoteSummary `json:"notes"`
}

// Backlink is one note linking to the note being viewed.
type Backlink struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// NoteDetail is the JSON body of GET /api/notes/{path...}.
type NoteDetail struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	HTML        string         `json:"html"`
	Frontmatter map[string]any `json:"frontmatter"`
	Tags        []string       `json:"tags"`
	Backlinks   []Backlink     `json:"backlinks"`
}

// NotesListHandler serves GET /api/notes: every note's path, title and tags.
func NotesListHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notes, err := listNotes(db)
		if err != nil {
			log.Printf("list notes: %v", err)
			http.Error(w, "list notes failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, NotesListResponse{Notes: notes})
	}
}

// NoteHandler serves GET /api/notes/{path...}: rendered HTML, frontmatter,
// tags and backlinks for one note.
func NoteHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		if path == "" {
			http.Error(w, "missing note path", http.StatusBadRequest)
			return
		}

		detail, err := loadNote(db, path)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("load note %q: %v", path, err)
			http.Error(w, "load note failed", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, detail)
	}
}

func listNotes(db *index.DB) ([]NoteSummary, error) {
	rows, err := db.Query(`SELECT path, title FROM notes ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []NoteSummary{}
	for rows.Next() {
		var n NoteSummary
		if err := rows.Scan(&n.Path, &n.Title); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagsByPath, err := allTagsByPath(db)
	if err != nil {
		return nil, err
	}
	for i := range notes {
		notes[i].Tags = tagsByPath[notes[i].Path]
	}
	return notes, nil
}

func loadNote(db *index.DB, path string) (*NoteDetail, error) {
	var title, body, frontmatterJSON string
	err := db.QueryRow(
		`SELECT title, body, frontmatter_json FROM notes WHERE path = ?`, path,
	).Scan(&title, &body, &frontmatterJSON)
	if err != nil {
		return nil, err
	}

	var frontmatter map[string]any
	if err := json.Unmarshal([]byte(frontmatterJSON), &frontmatter); err != nil {
		return nil, err
	}

	resolve, err := linkResolverFor(db, path)
	if err != nil {
		return nil, err
	}
	html, err := render.Render(body, resolve)
	if err != nil {
		return nil, err
	}

	tags, err := tagsFor(db, path)
	if err != nil {
		return nil, err
	}

	backlinks, err := backlinksFor(db, path)
	if err != nil {
		return nil, err
	}

	return &NoteDetail{
		Path:        path,
		Title:       title,
		HTML:        html,
		Frontmatter: frontmatter,
		Tags:        tags,
		Backlinks:   backlinks,
	}, nil
}

// linkResolverFor builds a render.Resolver from this note's links rows,
// which the indexer already resolved against the full vault path list —
// rendering reuses that instead of re-resolving targets itself.
func linkResolverFor(db *index.DB, sourcePath string) (render.Resolver, error) {
	rows, err := db.Query(
		`SELECT target_raw, target_path FROM links WHERE source_path = ? AND target_path IS NOT NULL`,
		sourcePath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := map[string]string{}
	for rows.Next() {
		var raw, target string
		if err := rows.Scan(&raw, &target); err != nil {
			return nil, err
		}
		targets[raw] = target
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return func(raw string) (string, bool) {
		target, ok := targets[raw]
		return target, ok
	}, nil
}

func tagsFor(db *index.DB, path string) ([]string, error) {
	rows, err := db.Query(`SELECT tag FROM tags WHERE path = ? ORDER BY tag`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func allTagsByPath(db *index.DB) (map[string][]string, error) {
	rows, err := db.Query(`SELECT path, tag FROM tags ORDER BY path, tag`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var path, tag string
		if err := rows.Scan(&path, &tag); err != nil {
			return nil, err
		}
		result[path] = append(result[path], tag)
	}
	return result, rows.Err()
}

func backlinksFor(db *index.DB, path string) ([]Backlink, error) {
	rows, err := db.Query(`
		SELECT DISTINCT notes.path, notes.title
		FROM links
		JOIN notes ON notes.path = links.source_path
		WHERE links.target_path = ?
		ORDER BY notes.title`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	backlinks := []Backlink{}
	for rows.Next() {
		var b Backlink
		if err := rows.Scan(&b.Path, &b.Title); err != nil {
			return nil, err
		}
		backlinks = append(backlinks, b)
	}
	return backlinks, rows.Err()
}
