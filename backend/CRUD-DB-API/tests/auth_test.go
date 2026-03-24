package server_test

import (
	"testing"

	"crud-db-api/server"
)

// ---------------------------------------------------------------------------
// CheckAccessAdmin
// ---------------------------------------------------------------------------

func TestCheckAccessAdmin_CorrectPassword(t *testing.T) {
	t.Setenv("adminPassword", "secret123")
	ok, msg := server.CheckAccessAdmin("secret123")
	if !ok {
		t.Errorf("expected true, got false (msg: %s)", msg)
	}
}

func TestCheckAccessAdmin_WrongPassword(t *testing.T) {
	t.Setenv("adminPassword", "secret123")
	ok, _ := server.CheckAccessAdmin("wrong")
	if ok {
		t.Error("expected false for wrong password")
	}
}

func TestCheckAccessAdmin_EmptyToken(t *testing.T) {
	t.Setenv("adminPassword", "secret123")
	ok, msg := server.CheckAccessAdmin("")
	if ok {
		t.Error("expected false for empty token")
	}
	if msg == "" {
		t.Error("expected non-empty message for empty token")
	}
}

func TestCheckAccessAdmin_EmptyEnvVar_AlwaysFails(t *testing.T) {
	t.Setenv("adminPassword", "")
	ok, _ := server.CheckAccessAdmin("anything")
	if ok {
		t.Error("expected false when adminPassword env var is empty")
	}
}

// ---------------------------------------------------------------------------
// CheckAccessJury
// ---------------------------------------------------------------------------

func TestCheckAccessJury_Password1Accepted(t *testing.T) {
	t.Setenv("juryPassword1", "jp1")
	t.Setenv("juryPassword2", "jp2")
	t.Setenv("juryPassword3", "jp3")
	ok, _ := server.CheckAccessJury("jp1")
	if !ok {
		t.Error("juryPassword1 should be accepted")
	}
}

func TestCheckAccessJury_Password2Accepted(t *testing.T) {
	t.Setenv("juryPassword1", "jp1")
	t.Setenv("juryPassword2", "jp2")
	t.Setenv("juryPassword3", "jp3")
	ok, _ := server.CheckAccessJury("jp2")
	if !ok {
		t.Error("juryPassword2 should be accepted")
	}
}

func TestCheckAccessJury_Password3Accepted(t *testing.T) {
	t.Setenv("juryPassword1", "jp1")
	t.Setenv("juryPassword2", "jp2")
	t.Setenv("juryPassword3", "jp3")
	ok, _ := server.CheckAccessJury("jp3")
	if !ok {
		t.Error("juryPassword3 should be accepted")
	}
}

func TestCheckAccessJury_WrongPassword(t *testing.T) {
	t.Setenv("juryPassword1", "jp1")
	t.Setenv("juryPassword2", "jp2")
	t.Setenv("juryPassword3", "jp3")
	ok, _ := server.CheckAccessJury("notajurytoken")
	if ok {
		t.Error("wrong token should be rejected")
	}
}

func TestCheckAccessJury_EmptyToken(t *testing.T) {
	ok, _ := server.CheckAccessJury("")
	if ok {
		t.Error("empty token should be rejected")
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

func TestCheckPhoneNum_InvalidNumber_ReturnsEmpty(t *testing.T) {
	region, err := server.CheckPhoneNum("notaphone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if region != "" {
		t.Errorf("expected empty region for invalid number, got %q", region)
	}
}

func TestCheckPhoneNum_EmptyString_ReturnsEmpty(t *testing.T) {
	region, err := server.CheckPhoneNum("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if region != "" {
		t.Errorf("expected empty region for empty input, got %q", region)
	}
}

// ---------------------------------------------------------------------------
// HashPassword / CheckPassword
// ---------------------------------------------------------------------------

func TestHashPassword_Deterministic(t *testing.T) {
	h1, _ := server.HashPassword("mysecret")
	h2, _ := server.HashPassword("mysecret")
	if h1 != h2 {
		t.Error("HashPassword should be deterministic")
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
	if !server.CheckPassword("password", "password") {
		t.Error("identical strings should match")
	}
}

func TestCheckPassword_Mismatch(t *testing.T) {
	if server.CheckPassword("password", "different") {
		t.Error("different strings should not match")
	}
}
