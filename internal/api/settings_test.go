package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/auth"
	"github.com/Stinger911/Amethyst/internal/index"
)

func openSettingsTestDB(t *testing.T) *index.DB {
	t.Helper()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func authedRequest(t *testing.T, db *index.DB, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := auth.NewSession(db)
	if err != nil {
		t.Fatalf("auth.NewSession: %v", err)
	}
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	NewServer(db, TelegramConfig{}, WriteConfig{}, nil, nil, nil).ServeHTTP(rec, req)
	return rec
}

func TestSettings_DefaultsToInbox(t *testing.T) {
	db := openSettingsTestDB(t)
	rec := authedRequest(t, db, http.MethodGet, "/api/settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp SettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.CaptureMode != "inbox" {
		t.Errorf("CaptureMode = %q, want %q", resp.CaptureMode, "inbox")
	}
}

func TestSettings_SaveThenGetRoundTrips(t *testing.T) {
	db := openSettingsTestDB(t)

	body, _ := json.Marshal(SettingsResponse{CaptureMode: "daily"})
	putRec := authedRequest(t, db, http.MethodPut, "/api/settings", body)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	getRec := authedRequest(t, db, http.MethodGet, "/api/settings", nil)
	var resp SettingsResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.CaptureMode != "daily" {
		t.Errorf("CaptureMode = %q, want %q", resp.CaptureMode, "daily")
	}
}

func TestSettings_SaveRejectsUnknownMode(t *testing.T) {
	db := openSettingsTestDB(t)
	body, _ := json.Marshal(SettingsResponse{CaptureMode: "bogus"})
	rec := authedRequest(t, db, http.MethodPut, "/api/settings", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
