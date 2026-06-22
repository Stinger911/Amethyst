package settings

import (
	"path/filepath"
	"testing"

	"github.com/Stinger911/Amethyst/internal/index"
)

func openTestDB(t *testing.T) *index.DB {
	t.Helper()
	db, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestGetCaptureMode_DefaultsToInbox(t *testing.T) {
	db := openTestDB(t)
	mode, err := GetCaptureMode(db)
	if err != nil {
		t.Fatalf("GetCaptureMode: %v", err)
	}
	if mode != CaptureModeInbox {
		t.Errorf("mode = %q, want default %q", mode, CaptureModeInbox)
	}
}

func TestSetCaptureMode_PersistsAndOverwrites(t *testing.T) {
	db := openTestDB(t)

	if err := SetCaptureMode(db, CaptureModeDaily); err != nil {
		t.Fatalf("SetCaptureMode: %v", err)
	}
	if mode, err := GetCaptureMode(db); err != nil || mode != CaptureModeDaily {
		t.Fatalf("GetCaptureMode = (%q, %v), want (%q, nil)", mode, err, CaptureModeDaily)
	}

	if err := SetCaptureMode(db, CaptureModeInbox); err != nil {
		t.Fatalf("SetCaptureMode (overwrite): %v", err)
	}
	if mode, err := GetCaptureMode(db); err != nil || mode != CaptureModeInbox {
		t.Fatalf("GetCaptureMode = (%q, %v), want (%q, nil)", mode, err, CaptureModeInbox)
	}
}

func TestSetCaptureMode_RejectsUnknownMode(t *testing.T) {
	db := openTestDB(t)
	if err := SetCaptureMode(db, "bogus"); err == nil {
		t.Error("SetCaptureMode(bogus) = nil error, want one")
	}
}
