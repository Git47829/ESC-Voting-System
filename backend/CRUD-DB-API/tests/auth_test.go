package server_test

import (
	"testing"

	"crud-db-api/server"
)

func mustHash(t *testing.T, pw string) string {
	t.Helper()
	h, err := server.HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// ---------------------------------------------------------------------------
// CheckAccessAdmin
// ---------------------------------------------------------------------------

func TestCheckAccessAdmin_CorrectPassword(t *testing.T) {
	t.Setenv("adminPassword", mustHash(t, "secret123"))
	t.Setenv("adminMail", "admin@example.com")
	ok, msg := server.CheckAccessAdmin("secret123", "admin@example.com")
	if !ok {
		t.Errorf("expected true, got false (msg: %s)", msg)
	}
}

func TestCheckAccessAdmin_WrongPassword(t *testing.T) {
	t.Setenv("adminPassword", mustHash(t, "secret123"))
	t.Setenv("adminMail", "admin@example.com")
	ok, _ := server.CheckAccessAdmin("wrong", "admin@example.com")
	if ok {
		t.Error("expected false for wrong password")
	}
}

func TestCheckAccessAdmin_EmptyToken(t *testing.T) {
	t.Setenv("adminPassword", mustHash(t, "secret123"))
	t.Setenv("adminMail", "admin@example.com")
	ok, msg := server.CheckAccessAdmin("", "admin@example.com")
	if ok {
		t.Error("expected false for empty token")
	}
	if msg == "" {
		t.Error("expected non-empty message for empty token")
	}
}

func TestCheckAccessAdmin_EmptyEmail(t *testing.T) {
	t.Setenv("adminPassword", mustHash(t, "secret123"))
	t.Setenv("adminMail", "admin@example.com")
	ok, msg := server.CheckAccessAdmin("secret123", "")
	if ok {
		t.Error("expected false for empty email")
	}
	if msg == "" {
		t.Error("expected non-empty message for empty email")
	}
}

func TestCheckAccessAdmin_WrongEmail(t *testing.T) {
	t.Setenv("adminPassword", mustHash(t, "secret123"))
	t.Setenv("adminMail", "admin@example.com")
	ok, _ := server.CheckAccessAdmin("secret123", "other@example.com")
	if ok {
		t.Error("expected false for wrong email")
	}
}

func TestCheckAccessAdmin_EmptyEnvVar_AlwaysFails(t *testing.T) {
	t.Setenv("adminPassword", "")
	t.Setenv("adminMail", "admin@example.com")
	ok, _ := server.CheckAccessAdmin("anything", "admin@example.com")
	if ok {
		t.Error("expected false when adminPassword env var is empty")
	}
}

// ---------------------------------------------------------------------------
// CheckAccessJury
// ---------------------------------------------------------------------------

func TestCheckAccessJury_Password1Accepted(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("jp1", "jury1@example.com")
	if !ok {
		t.Error("juryPassword1 should be accepted")
	}
}

func TestCheckAccessJury_Password2Accepted(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("jp2", "jury2@example.com")
	if !ok {
		t.Error("juryPassword2 should be accepted")
	}
}

func TestCheckAccessJury_Password3Accepted(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("jp3", "jury3@example.com")
	if !ok {
		t.Error("juryPassword3 should be accepted")
	}
}

func TestCheckAccessJury_WrongPassword(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("notajurytoken", "jury1@example.com")
	if ok {
		t.Error("wrong token should be rejected")
	}
}

func TestCheckAccessJury_WrongEmail(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("jp1", "unknown@example.com")
	if ok {
		t.Error("unknown email should be rejected")
	}
}

func TestCheckAccessJury_MismatchedEmailAndPassword(t *testing.T) {
	t.Setenv("juryMail1", "jury1@example.com")
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	t.Setenv("juryMail2", "jury2@example.com")
	t.Setenv("juryPassword2", mustHash(t, "jp2"))
	t.Setenv("juryMail3", "jury3@example.com")
	t.Setenv("juryPassword3", mustHash(t, "jp3"))
	ok, _ := server.CheckAccessJury("jp2", "jury1@example.com")
	if ok {
		t.Error("mismatched email/password pair should be rejected")
	}
}

func TestCheckAccessJury_EmptyToken(t *testing.T) {
	ok, _ := server.CheckAccessJury("", "jury1@example.com")
	if ok {
		t.Error("empty token should be rejected")
	}
}

func TestCheckAccessJury_EmptyEmail(t *testing.T) {
	t.Setenv("juryPassword1", mustHash(t, "jp1"))
	ok, _ := server.CheckAccessJury("jp1", "")
	if ok {
		t.Error("empty email should be rejected")
	}
}

// ---------------------------------------------------------------------------
// CheckPhoneNum
// ---------------------------------------------------------------------------

func TestCheckPhoneNum_ValidGermanNumber_ReturnsDE(t *testing.T) {
	region, err := server.CheckPhoneNum("+4915123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if region != "DE" {
		t.Errorf("expected DE, got %q", region)
	}
}

func TestCheckPhoneNum_ValidFrenchNumber_ReturnsFR(t *testing.T) {
	region, err := server.CheckPhoneNum("+33612345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if region != "FR" {
		t.Errorf("expected FR, got %q", region)
	}
}

func TestCheckPhoneNum_InvalidNumber_ReturnsError(t *testing.T) {
	region, err := server.CheckPhoneNum("notaphone")
	if err == nil {
		t.Fatal("expected error for invalid phone number")
	}
	if region != "" {
		t.Errorf("expected empty region for invalid number, got %q", region)
	}
}

func TestCheckPhoneNum_EmptyString_ReturnsError(t *testing.T) {
	region, err := server.CheckPhoneNum("")
	if err == nil {
		t.Fatal("expected error for empty phone input")
	}
	if region != "" {
		t.Errorf("expected empty region for empty input, got %q", region)
	}
}

// ---------------------------------------------------------------------------
// HashPassword / CheckPassword
// ---------------------------------------------------------------------------

func TestHashPassword_Deterministic(t *testing.T) {
	h, err := server.HashPassword("mysecret")
	if err != nil {
		t.Fatal(err)
	}
	if !server.CheckPassword("mysecret", h) {
		t.Error("hashed password should verify correctly")
	}
}

func TestHashPassword_DifferentInputs_DifferentOutputs(t *testing.T) {
	h1, _ := server.HashPassword("secret1")
	h2, _ := server.HashPassword("secret2")
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestCheckPassword_Match(t *testing.T) {
	hash, _ := server.HashPassword("password")
	if !server.CheckPassword("password", hash) {
		t.Error("identical strings should match")
	}
}

func TestCheckPassword_Mismatch(t *testing.T) {
	hash, _ := server.HashPassword("password")
	if server.CheckPassword("different", hash) {
		t.Error("different strings should not match")
	}
}
