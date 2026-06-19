package auth

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

func TestEnsureCredential_FirstRunRequiresPassword(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "", false); err == nil {
		t.Fatal("EnsureCredential with empty password on first run, want error")
	}
}

func TestEnsureCredential_FirstRunSeedsPassword(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "s3cret", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}
	ok, err := VerifyPassword(db, "s3cret")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Error("VerifyPassword = false, want true for the seeded password")
	}
}

func TestEnsureCredential_SecondRunIgnoresPasswordWithoutReset(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "original", false); err != nil {
		t.Fatalf("EnsureCredential (seed): %v", err)
	}
	if err := EnsureCredential(db, "different", false); err != nil {
		t.Fatalf("EnsureCredential (no reset): %v", err)
	}

	ok, _ := VerifyPassword(db, "original")
	if !ok {
		t.Error("VerifyPassword(original) = false, want true: a plain restart must not change the password")
	}
	ok, _ = VerifyPassword(db, "different")
	if ok {
		t.Error("VerifyPassword(different) = true, want false: a plain restart must not adopt a new password")
	}
}

func TestEnsureCredential_ResetReplacesPassword(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "original", false); err != nil {
		t.Fatalf("EnsureCredential (seed): %v", err)
	}
	if err := EnsureCredential(db, "newpassword", true); err != nil {
		t.Fatalf("EnsureCredential (reset): %v", err)
	}

	ok, _ := VerifyPassword(db, "newpassword")
	if !ok {
		t.Error("VerifyPassword(newpassword) = false, want true after reset")
	}
	ok, _ = VerifyPassword(db, "original")
	if ok {
		t.Error("VerifyPassword(original) = true, want false after reset")
	}
}

func TestEnsureCredential_ResetWithoutPasswordErrors(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "original", false); err != nil {
		t.Fatalf("EnsureCredential (seed): %v", err)
	}
	if err := EnsureCredential(db, "", true); err == nil {
		t.Fatal("EnsureCredential(reset=true, password=\"\"), want error")
	}
}

func TestVerifyPassword_NoCredentialYet(t *testing.T) {
	db := openTestDB(t)
	_, err := VerifyPassword(db, "anything")
	if err != ErrNoCredential {
		t.Fatalf("err = %v, want ErrNoCredential", err)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureCredential(db, "correct", false); err != nil {
		t.Fatalf("EnsureCredential: %v", err)
	}
	ok, err := VerifyPassword(db, "wrong")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Error("VerifyPassword(wrong) = true, want false")
	}
}
