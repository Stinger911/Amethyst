package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/index"
)

func seedGraphIndex(t *testing.T) *index.DB {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	write("Hub.md", "Links to [[Leaf]], an embedded [[Pic.png]] and a [[Nowhere]] that doesn't exist.\n")
	write("Leaf.md", "A leaf note, links nowhere.\n")
	write("Island.md", "Nobody links to me and I link to nobody.\n")
	write("Pic.png", "not really a png, just needs to exist as a file")

	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := index.ColdScan(db, root); err != nil {
		t.Fatalf("ColdScan: %v", err)
	}
	return db
}

func TestGraph_IncludesIsolatedNotesAsNodes(t *testing.T) {
	db := seedGraphIndex(t)
	rec := doGet(t, db, "", "/api/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp GraphResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, rec.Body.String())
	}

	paths := map[string]bool{}
	for _, n := range resp.Nodes {
		paths[n.Path] = true
	}
	for _, want := range []string{"Hub.md", "Leaf.md", "Island.md"} {
		if !paths[want] {
			t.Errorf("Nodes = %+v, missing %q", resp.Nodes, want)
		}
	}
	if paths["Pic.png"] {
		t.Errorf("Nodes = %+v, attachments should not be graph nodes", resp.Nodes)
	}
}

func TestGraph_EdgesOnlyConnectNoteToNoteLinks(t *testing.T) {
	db := seedGraphIndex(t)
	rec := doGet(t, db, "", "/api/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp GraphResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v, body = %s", err, rec.Body.String())
	}

	if len(resp.Edges) != 1 {
		t.Fatalf("Edges = %+v, want exactly 1 (Hub.md -> Leaf.md)", resp.Edges)
	}
	if resp.Edges[0].Source != "Hub.md" || resp.Edges[0].Target != "Leaf.md" {
		t.Errorf("Edges[0] = %+v, want Hub.md -> Leaf.md", resp.Edges[0])
	}
}
