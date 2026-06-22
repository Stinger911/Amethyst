package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

func seedIndex(t *testing.T) *index.DB {
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
	write("Apple.md", "# Apple\n\nApple pie is a dessert made with apples and pastry.\n")
	write("Banana.md", "# Banana\n\nBanana bread is a dessert made with bananas.\n")
	write("Unrelated.md", "# Unrelated\n\nNothing to do with fruit.\n")

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

func doSearch(t *testing.T, db *index.DB, query string) SearchResponse {
	t.Helper()
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/search?"+query, nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v, body = %s", err, rec.Body.String())
	}
	return resp
}

func TestSearch_MatchesSingleWord(t *testing.T) {
	db := seedIndex(t)
	resp := doSearch(t, db, "q=banana")

	if len(resp.Results) != 1 {
		t.Fatalf("Results = %+v, want 1 match", resp.Results)
	}
	if resp.Results[0].Path != "Banana.md" {
		t.Errorf("Path = %q, want %q", resp.Results[0].Path, "Banana.md")
	}
	if resp.Results[0].Snippet == "" {
		t.Error("Snippet is empty")
	}
}

func TestSearch_MultiWordIsImplicitAnd(t *testing.T) {
	db := seedIndex(t)

	resp := doSearch(t, db, "q=dessert+apples")
	if len(resp.Results) != 1 || resp.Results[0].Path != "Apple.md" {
		t.Fatalf("Results = %+v, want only Apple.md (matches both terms)", resp.Results)
	}

	resp = doSearch(t, db, "q=dessert+fruit")
	if len(resp.Results) != 0 {
		t.Fatalf("Results = %+v, want none (no note has both terms)", resp.Results)
	}
}

func TestSearch_NoMatchReturnsEmptyResults(t *testing.T) {
	db := seedIndex(t)
	resp := doSearch(t, db, "q=nonexistentterm")
	if resp.Results == nil {
		t.Error("Results = nil, want empty slice (so JSON is [] not null)")
	}
	if len(resp.Results) != 0 {
		t.Errorf("Results = %+v, want none", resp.Results)
	}
}

func TestSearch_EmptyQueryReturnsEmptyResults(t *testing.T) {
	db := seedIndex(t)
	resp := doSearch(t, db, "q=")
	if len(resp.Results) != 0 {
		t.Errorf("Results = %+v, want none for empty query", resp.Results)
	}
}

func TestSearch_LimitIsRespected(t *testing.T) {
	db := seedIndex(t)
	resp := doSearch(t, db, "q=dessert&limit=1")
	if len(resp.Results) != 1 {
		t.Fatalf("Results = %+v, want exactly 1 (limit=1)", resp.Results)
	}
}

func TestSearch_SpecialCharactersDontBreakQuery(t *testing.T) {
	db := seedIndex(t)
	for _, q := range []string{`q=banana"bread`, `q=AND+OR+NOT`, `q=col:value`, `q=foo-bar`} {
		resp := doSearch(t, db, q)
		_ = resp // just asserting no 500 / no panic; doSearch already fails the test on non-200
	}
}
