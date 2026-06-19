package auth

import "testing"

func TestSession_NewSessionValidates(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewSession(db)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	ok, err := ValidateSession(db, token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if !ok {
		t.Error("ValidateSession = false, want true for a freshly created session")
	}
}

func TestSession_UnknownTokenIsInvalid(t *testing.T) {
	db := openTestDB(t)
	ok, err := ValidateSession(db, "not-a-real-token")
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if ok {
		t.Error("ValidateSession = true, want false for an unknown token")
	}
}

func TestSession_EmptyTokenIsInvalid(t *testing.T) {
	db := openTestDB(t)
	ok, err := ValidateSession(db, "")
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if ok {
		t.Error("ValidateSession(\"\") = true, want false")
	}
}

func TestSession_ExpiredTokenIsInvalidAndCleaned(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewSession(db)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	// Force expiry directly, bypassing the public API's fixed lifetime.
	if _, err := db.Exec(`UPDATE sessions SET expires_at = 0 WHERE token_hash = ?`, hashToken(token)); err != nil {
		t.Fatalf("force-expire: %v", err)
	}

	ok, err := ValidateSession(db, token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if ok {
		t.Error("ValidateSession = true, want false for an expired session")
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sessions WHERE token_hash = ?`, hashToken(token)).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Error("expired session row should have been deleted as a side effect")
	}
}

func TestSession_RevokeInvalidatesIt(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewSession(db)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := RevokeSession(db, token); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	ok, err := ValidateSession(db, token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if ok {
		t.Error("ValidateSession = true, want false after revoke")
	}
}
