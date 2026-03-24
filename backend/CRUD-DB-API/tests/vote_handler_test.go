package server_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"crud-db-api/server"
)

// setupVoteMock prepares a sqlmock DB with the common successful-vote query
// sequence. Since two goroutines run concurrently, MatchExpectationsInOrder is
// disabled.
func setupVoteSuccessMock(t *testing.T, songLandID string, remaining int) (*sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.MatchExpectationsInOrder(false)
	server.DB = mockDB

	// Concurrent errgroup queries (order undefined):
	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	mock.ExpectQuery("SELECT Land_ID FROM Song WHERE ID").
		WillReturnRows(sqlmock.NewRows([]string{"Land_ID"}).AddRow(songLandID))

	// Sequential: upsert phone, deduct points
	mock.ExpectExec("INSERT INTO Phone_Nums").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE Phone_Nums").WillReturnResult(sqlmock.NewResult(1, 1))

	// Second concurrent errgroup (readback + song update):
	mock.ExpectQuery("SELECT votes_remaining").
		WillReturnRows(sqlmock.NewRows([]string{"votes_remaining", "votes_cast"}).
			AddRow(remaining, `{"1":5}`))
	mock.ExpectExec("UPDATE Song SET PublikumsPunkte").
		WillReturnResult(sqlmock.NewResult(1, 1))

	return &mock, func() { mockDB.Close() }
}

func voteRequest(phone, songID, points string) *http.Request {
	q := url.Values{}
	q.Set("phoneNum", phone)
	q.Set("songID", songID)
	q.Set("points", points)
	q.Set("ownCountry", "SE")
	return httptest.NewRequest(http.MethodPost, "/vote/?"+q.Encode(), nil)
}

// ---------------------------------------------------------------------------
// Successful vote
// ---------------------------------------------------------------------------

func TestVote_ValidRequest_Returns201(t *testing.T) {
	_, cleanup := setupVoteSuccessMock(t, "FR", 15)
	defer cleanup()

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestVote_ValidRequest_ResponseShape(t *testing.T) {
	_, cleanup := setupVoteSuccessMock(t, "FR", 15)
	defer cleanup()

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	for _, key := range []string{"message", "voted_for", "points_spent", "votes_remaining", "votes_cast"} {
		if _, ok := body[key]; !ok {
			t.Errorf("expected key %q in response body", key)
		}
	}
}

func TestVote_ValidRequest_SetsCookieWithHMAC(t *testing.T) {
	_, cleanup := setupVoteSuccessMock(t, "FR", 15)
	defer cleanup()

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	var voteCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "vote_state" {
			voteCookie = c
		}
	}
	if voteCookie == nil {
		t.Fatal("expected vote_state cookie to be set")
	}

	raw, err := hex.DecodeString(voteCookie.Value)
	if err != nil {
		t.Fatalf("cookie value is not valid hex: %v", err)
	}

	dotIdx := bytes.LastIndexByte(raw, '.')
	if dotIdx < 0 {
		t.Fatal("no '.' in decoded cookie")
	}
	jsonPart := raw[:dotIdx]
	sigHex := string(raw[dotIdx+1:])

	mac := hmac.New(sha256.New, server.SignedCookieSecret)
	mac.Write(jsonPart)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if sigHex != expectedSig {
		t.Error("cookie HMAC signature is invalid")
	}
}

// ---------------------------------------------------------------------------
// Phone validation errors
// ---------------------------------------------------------------------------

func TestVote_InvalidPhoneNumber_Returns406(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := voteRequest("notaphone", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Errorf("expected 406, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Voting closed
// ---------------------------------------------------------------------------

func TestVote_VotingClosed_Returns425(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(false))
	mock.ExpectQuery("SELECT Land_ID FROM Song WHERE ID").
		WillReturnRows(sqlmock.NewRows([]string{"Land_ID"}).AddRow("FR"))

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusTooEarly {
		t.Errorf("expected 425, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Song not found
// ---------------------------------------------------------------------------

func TestVote_SongNotFound_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	mock.ExpectQuery("SELECT Land_ID FROM Song WHERE ID").
		WillReturnRows(sqlmock.NewRows([]string{"Land_ID"})) // 0 rows → ErrNoRows

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Vote for own country
// ---------------------------------------------------------------------------

func TestVote_VoteForOwnCountry_Returns403(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	// Phone is German (+49), song belongs to "DE".
	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	mock.ExpectQuery("SELECT Land_ID FROM Song WHERE ID").
		WillReturnRows(sqlmock.NewRows([]string{"Land_ID"}).AddRow("DE"))

	req := voteRequest("+4915123456789", "1", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error message in body")
	}
}

// ---------------------------------------------------------------------------
// Invalid songID / points
// ---------------------------------------------------------------------------

func TestVote_InvalidSongID_Returns406(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := voteRequest("+4915123456789", "notanumber", "5")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Errorf("expected 406, got %d", rr.Code)
	}
}

func TestVote_PointsZero_Returns404(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := voteRequest("+4915123456789", "1", "0")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestVote_PointsTooHigh_Returns404(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := voteRequest("+4915123456789", "1", "21")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestVote_PointsNotInteger_Returns404(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := voteRequest("+4915123456789", "1", "abc")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Insufficient vote budget
// ---------------------------------------------------------------------------

func TestVote_InsufficientBudget_Returns403(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	mock.ExpectQuery("SELECT Land_ID FROM Song WHERE ID").
		WillReturnRows(sqlmock.NewRows([]string{"Land_ID"}).AddRow("FR"))

	mock.ExpectExec("INSERT INTO Phone_Nums").WillReturnResult(sqlmock.NewResult(1, 1))

	// Deduct returns 0 rows affected (insufficient budget).
	mock.ExpectExec("UPDATE Phone_Nums").WillReturnResult(sqlmock.NewResult(0, 0))

	// Handler queries remaining votes to report back.
	mock.ExpectQuery("SELECT votes_remaining FROM Phone_Nums").
		WillReturnRows(sqlmock.NewRows([]string{"votes_remaining"}).AddRow(2))

	req := voteRequest("+4915123456789", "1", "10")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	if _, ok := body["votes_remaining"]; !ok {
		t.Error("expected votes_remaining in error response")
	}
}

// ---------------------------------------------------------------------------
// Full 20-point budget vote
// ---------------------------------------------------------------------------

func TestVote_FullBudget20Points_Returns201(t *testing.T) {
	_, cleanup := setupVoteSuccessMock(t, "FR", 0)
	defer cleanup()

	req := voteRequest("+4915123456789", "1", "20")
	rr := httptest.NewRecorder()
	server.Vote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
