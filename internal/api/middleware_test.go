package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

func openMiddlewareTestDB(t *testing.T) *index.DB {
	t.Helper()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRequireAuth_NoCookieIsUnauthorized(t *testing.T) {
	db := openMiddlewareTestDB(t)
	for _, path := range []string{"/api/notes", "/api/search?q=x", "/api/graph"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		NewServer(db).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", path, rec.Code)
		}
	}
}

func TestRequireAuth_InvalidCookieIsUnauthorized(t *testing.T) {
	db := openMiddlewareTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "bogus"})
	rec := httptest.NewRecorder()
	NewServer(db).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuth_ValidSessionIsAllowedThrough(t *testing.T) {
	db := openMiddlewareTestDB(t)
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 with a valid session", rec.Code, rec.Body.String())
	}
}

func TestRequireAuth_LoginAndLogoutRemainUngated(t *testing.T) {
	db := openMiddlewareTestDB(t)
	if err := auth.EnsureCredential(db, "s3cret", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}

	rec := postJSON(t, db, "/api/auth/login", loginRequest{Password: "s3cret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login without a cookie: status = %d, want 200 (login is not gated)", rec.Code)
	}

	logoutRec := postJSON(t, db, "/api/auth/logout", nil)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout without a session: status = %d, want 200 (logout is a no-op, not gated)", logoutRec.Code)
	}
}
