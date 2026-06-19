package api

import (
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
)

// NewServer wires up the Go REST surface from plan_amethyst-web-ui.
// Routes are added incrementally as each is implemented.
func NewServer(db *index.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", SearchHandler(db))
	mux.HandleFunc("GET /api/notes", NotesListHandler(db))
	mux.HandleFunc("GET /api/notes/{path...}", NoteHandler(db))
	return mux
}
