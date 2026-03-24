package server_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"crud-db-api/server"
)

func TestEncodeCookieValue_ProducesHexString(t *testing.T) {
	state := server.CookieVoteState{VotesRemaining: 15, VotesCast: map[string]int{"1": 5}}
	result, err := server.EncodeCookieValue(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// Must be valid hex
	if _, decErr := hex.DecodeString(result); decErr != nil {
		t.Errorf("result is not valid hex: %v", decErr)
	}
}

func TestEncodeCookieValue_SignatureIsHMACSHA256(t *testing.T) {
	// Ensure a known cookie secret.
	t.Setenv("adminPassword", "test-cookie-pw")
	server.InitCookieSecret()

	state := server.CookieVoteState{VotesRemaining: 10, VotesCast: map[string]int{"2": 10}}
	result, err := server.EncodeCookieValue(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, _ := hex.DecodeString(result)

	// The format is: hex(jsonBytes + "." + hexSig)
	// Find the dot separator from the decoded bytes.
	dotIdx := bytes.LastIndexByte(raw, '.')
	if dotIdx < 0 {
		t.Fatal("no '.' separator found in decoded cookie")
	}

	jsonPart := raw[:dotIdx]
	sigPartHex := string(raw[dotIdx+1:])

	// Recompute expected HMAC.
	mac := hmac.New(sha256.New, server.SignedCookieSecret)
	mac.Write(jsonPart)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if sigPartHex != expectedSig {
		t.Errorf("HMAC signature mismatch: got %q, want %q", sigPartHex, expectedSig)
	}
}

func TestEncodeCookieValue_PayloadIsValidJSON(t *testing.T) {
	state := server.CookieVoteState{VotesRemaining: 8, VotesCast: map[string]int{"3": 12}}
	result, err := server.EncodeCookieValue(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, _ := hex.DecodeString(result)
	dotIdx := bytes.LastIndexByte(raw, '.')
	if dotIdx < 0 {
		t.Fatal("no '.' separator found in decoded cookie")
	}
	jsonPart := raw[:dotIdx]

	var decoded server.CookieVoteState
	if err := json.Unmarshal(jsonPart, &decoded); err != nil {
		t.Errorf("JSON part is not valid JSON: %v", err)
	}
	if decoded.VotesRemaining != 8 {
		t.Errorf("expected VotesRemaining=8, got %d", decoded.VotesRemaining)
	}
}

func TestEncodeCookieValue_DifferentStates_DifferentValues(t *testing.T) {
	s1 := server.CookieVoteState{VotesRemaining: 10, VotesCast: map[string]int{"1": 5}}
	s2 := server.CookieVoteState{VotesRemaining: 15, VotesCast: map[string]int{"1": 5}}
	r1, _ := server.EncodeCookieValue(s1)
	r2, _ := server.EncodeCookieValue(s2)
	if r1 == r2 {
		t.Error("different states should produce different cookie values")
	}
}

func TestInitCookieSecret_DerivedFromAdminPassword(t *testing.T) {
	t.Setenv("adminPassword", "known-password")
	server.InitCookieSecret()

	sum := sha256.Sum256([]byte("cookie-secret:known-password"))
	expected := sum[:]

	if !bytes.Equal(server.SignedCookieSecret, expected) {
		t.Error("cookie secret should be sha256(cookie-secret:<adminPassword>)")
	}
}

func TestInitCookieSecret_RandomWhenNoPassword(t *testing.T) {
	t.Setenv("adminPassword", "")
	server.InitCookieSecret()
	defer func() {
		// Always restore so subsequent tests see the standard password.
		t.Setenv("adminPassword", "test-admin-pw")
		server.InitCookieSecret()
	}()

	if len(server.SignedCookieSecret) != 32 {
		t.Errorf("expected 32-byte secret, got %d bytes", len(server.SignedCookieSecret))
	}

	allZero := true
	for _, b := range server.SignedCookieSecret {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("random cookie secret should not be all zeros")
	}
}
