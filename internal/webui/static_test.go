package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func builtFS() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html":         {Data: []byte("<html>shell</html>")},
		"dist/assets/app.js":      {Data: []byte("console.log('app')")},
		"dist/notesTree.test.txt": {Data: []byte("not a route")},
	}
}

func TestSpaHandler_NotBuilt(t *testing.T) {
	h, err := spaHandler(fstest.MapFS{"dist/placeholder.txt": {Data: []byte("nope")}})
	if err != nil {
		t.Fatalf("spaHandler: %v", err)
	}
	if h != nil {
		t.Fatal("expected nil handler when index.html is missing")
	}
}

func TestSpaHandler_ServesRealFile(t *testing.T) {
	h, err := spaHandler(builtFS())
	if err != nil {
		t.Fatalf("spaHandler: %v", err)
	}
	if h == nil {
		t.Fatal("expected a non-nil handler once index.html exists")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "console.log('app')" {
		t.Fatalf("body = %q, want the real asset content", got)
	}
}

func TestSpaHandler_FallsBackToIndexForClientRoutes(t *testing.T) {
	h, err := spaHandler(builtFS())
	if err != nil {
		t.Fatalf("spaHandler: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notes/Some%20Note.md", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "<html>shell</html>" {
		t.Fatalf("body = %q, want the index.html shell", got)
	}
}
