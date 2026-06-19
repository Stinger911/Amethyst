package api

import (
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
)

// NewServer wires up the Go REST surface from plan_amethyst-web-ui.
// Routes are added incrementally as each is implemented.
func NewServer(db *index.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", RequireAuth(db, SearchHandler(db)))
	mux.HandleFunc("GET /api/notes", RequireAuth(db, NotesListHandler(db)))
	mux.HandleFunc("GET /api/notes/{path...}", RequireAuth(db, NoteHandler(db)))
	mux.HandleFunc("GET /api/graph", RequireAuth(db, GraphHandler(db)))
	mux.HandleFunc("POST /api/auth/login", LoginHandler(db))
	mux.HandleFunc("POST /api/auth/logout", LogoutHandler(db))
	return mux
}
