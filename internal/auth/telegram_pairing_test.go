package auth

import "testing"

func TestPairing_RedeemValidTokenSetsOwner(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewPairingToken(db)
	if err != nil {
		t.Fatalf("NewPairingToken: %v", err)
	}

	ok, err := RedeemPairingToken(db, token, "12345")
	if err != nil {
		t.Fatalf("RedeemPairingToken: %v", err)
	}
	if !ok {
		t.Fatal("RedeemPairingToken = false, want true for a freshly created token")
	}

	owner, err := GetTelegramOwnerChatID(db)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "12345" {
		t.Errorf("owner = %q, want %q", owner, "12345")
	}
}

func TestPairing_TokenIsSingleUse(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewPairingToken(db)
	if err != nil {
		t.Fatalf("NewPairingToken: %v", err)
	}

	if ok, err := RedeemPairingToken(db, token, "12345"); err != nil || !ok {
		t.Fatalf("first redeem = (%v, %v), want (true, nil)", ok, err)
	}
	if ok, err := RedeemPairingToken(db, token, "99999"); err != nil || ok {
		t.Fatalf("second redeem = (%v, %v), want (false, nil)", ok, err)
	}

	owner, err := GetTelegramOwnerChatID(db)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "12345" {
		t.Errorf("owner = %q, want it unchanged at %q after a replay attempt", owner, "12345")
	}
}

func TestPairing_UnknownTokenIsRejected(t *testing.T) {
	db := openTestDB(t)
	ok, err := RedeemPairingToken(db, "not-a-real-token", "12345")
	if err != nil {
		t.Fatalf("RedeemPairingToken: %v", err)
	}
	if ok {
		t.Error("RedeemPairingToken = true, want false for an unknown token")
	}
}

func TestPairing_EmptyTokenIsRejected(t *testing.T) {
	db := openTestDB(t)
	ok, err := RedeemPairingToken(db, "", "12345")
	if err != nil {
		t.Fatalf("RedeemPairingToken: %v", err)
	}
	if ok {
		t.Error("RedeemPairingToken = true, want false for an empty token")
	}
}

func TestPairing_ExpiredTokenIsRejectedAndCleaned(t *testing.T) {
	db := openTestDB(t)
	token, _, err := NewPairingToken(db)
	if err != nil {
		t.Fatalf("NewPairingToken: %v", err)
	}
	if _, err := db.Exec(`UPDATE telegram_pairing_tokens SET expires_at = 0 WHERE token_hash = ?`, hashToken(token)); err != nil {
		t.Fatalf("force-expire: %v", err)
	}

	ok, err := RedeemPairingToken(db, token, "12345")
	if err != nil {
		t.Fatalf("RedeemPairingToken: %v", err)
	}
	if ok {
		t.Error("RedeemPairingToken = true, want false for an expired token")
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM telegram_pairing_tokens`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("telegram_pairing_tokens count = %d, want 0 (expired token cleaned up)", count)
	}
}

func TestPairing_GetOwnerChatIDDefaultsToEmpty(t *testing.T) {
	db := openTestDB(t)
	owner, err := GetTelegramOwnerChatID(db)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "" {
		t.Errorf("owner = %q, want empty before any pairing", owner)
	}
}

func TestPairing_SetOwnerChatIDOverwrites(t *testing.T) {
	db := openTestDB(t)
	if err := SetTelegramOwnerChatID(db, "111"); err != nil {
		t.Fatalf("SetTelegramOwnerChatID: %v", err)
	}
	if err := SetTelegramOwnerChatID(db, "222"); err != nil {
		t.Fatalf("SetTelegramOwnerChatID (overwrite): %v", err)
	}

	owner, err := GetTelegramOwnerChatID(db)
	if err != nil {
		t.Fatalf("GetTelegramOwnerChatID: %v", err)
	}
	if owner != "222" {
		t.Errorf("owner = %q, want %q after overwrite", owner, "222")
	}
}
