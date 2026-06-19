// This file implements GET /api/graph from plan_amethyst-web-ui §4: the
// server hands over nodes/edges only, the force-directed layout and
// interactivity are client-side (the one deliberate exception to
// "rendering happens on Go").
package api

import (
	"log"
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
)

// GraphNode is one note in the link graph.
type GraphNode struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// GraphEdge is one resolved wiki-link/embed between two notes.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// GraphResponse is the JSON body of GET /api/graph.
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphHandler serves GET /api/graph: every note as a node (including
// notes with no links, so isolated notes still show up), plus an edge for
// each distinct note-to-note link. Links to non-notes (attachments) or to
// unresolved targets are omitted since they have no graph node to connect to.
func GraphHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		graph, err := loadGraph(db)
		if err != nil {
			log.Printf("load graph: %v", err)
			http.Error(w, "load graph failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, graph)
	}
}

func loadGraph(db *index.DB) (*GraphResponse, error) {
	nodeRows, err := db.Query(`SELECT path, title FROM notes ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer nodeRows.Close()

	nodes := []GraphNode{}
	for nodeRows.Next() {
		var n GraphNode
		if err := nodeRows.Scan(&n.Path, &n.Title); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	if err := nodeRows.Err(); err != nil {
		return nil, err
	}

	edgeRows, err := db.Query(`
		SELECT DISTINCT links.source_path, links.target_path
		FROM links
		JOIN notes ON notes.path = links.target_path
		ORDER BY links.source_path, links.target_path`)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()

	edges := []GraphEdge{}
	for edgeRows.Next() {
		var e GraphEdge
		if err := edgeRows.Scan(&e.Source, &e.Target); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, err
	}

	return &GraphResponse{Nodes: nodes, Edges: edges}, nil
}
