package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"crud-db-api/server"
)

func juryVoteURL(songID, points string) string {
	return "/jury/vote?songID=" + songID + "&points=" + points
}

// ---------------------------------------------------------------------------
// Valid jury votes (table-driven)
// ---------------------------------------------------------------------------

func TestJuryVote_ValidPoints_TableDriven(t *testing.T) {
	validPoints := []string{"1", "2", "3", "4", "5", "6", "7", "8", "10", "12"}

	for _, pts := range validPoints {
		t.Run("points="+pts, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer mockDB.Close()
			server.DB = mockDB
			mock.MatchExpectationsInOrder(false)

			mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
				WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
			mock.ExpectExec("UPDATE Song").
				WillReturnResult(sqlmock.NewResult(0, 1))

			req := httptest.NewRequest(http.MethodPost, juryVoteURL("1", pts), nil)
			rr := httptest.NewRecorder()
			server.JuryVote(rr, req)

			if rr.Code != http.StatusAccepted {
				t.Errorf("points=%s: expected 202, got %d — body: %s", pts, rr.Code, rr.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Invalid jury points (table-driven)
// ---------------------------------------------------------------------------

func TestJuryVote_InvalidPoints_TableDriven(t *testing.T) {
	invalidPoints := []string{"0", "9", "11", "13", "100", "-1"}

	for _, pts := range invalidPoints {
		t.Run("points="+pts, func(t *testing.T) {
			mockDB, _, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer mockDB.Close()
			server.DB = mockDB

			req := httptest.NewRequest(http.MethodPost, juryVoteURL("1", pts), nil)
			rr := httptest.NewRecorder()
			server.JuryVote(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("points=%s: expected 400, got %d", pts, rr.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parsing errors
// ---------------------------------------------------------------------------

func TestJuryVote_InvalidSongID_Returns422(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := httptest.NewRequest(http.MethodPost,
		"/jury/vote?songID=notanumber&points=8", nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestJuryVote_InvalidPointsNotInteger_Returns422(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	req := httptest.NewRequest(http.MethodPost,
		"/jury/vote?songID=1&points=notanumber", nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Auth failure
// ---------------------------------------------------------------------------

func TestJuryVote_WrongToken_Returns403(t *testing.T) {
	req := juryAuthRequest(http.MethodPost, "/jury/vote?songID=1&points=8", "wrongtoken", "jury1@test.com")
	rr := httptest.NewRecorder()
	server.RequireJury(http.HandlerFunc(server.JuryVote)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestJuryVote_MissingEmailHeader_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jury/vote?songID=1&points=8", nil)
	req.Header.Set("Authorization", "Bearer jury1")
	rr := httptest.NewRecorder()
	server.RequireJury(http.HandlerFunc(server.JuryVote)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Voting closed
// ---------------------------------------------------------------------------

func TestJuryVote_VotingClosed_Returns425(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(false))

	req := httptest.NewRequest(http.MethodPost, juryVoteURL("1", "8"), nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	if rr.Code != http.StatusTooEarly {
		t.Errorf("expected 425, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Song not found
// ---------------------------------------------------------------------------

func TestJuryVote_SongNotFound_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	// UPDATE affects 0 rows → song not found.
	mock.ExpectExec("UPDATE Song").
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodPost, juryVoteURL("999", "8"), nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// DB error
// ---------------------------------------------------------------------------

func TestJuryVote_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, juryVoteURL("1", "8"), nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// All three jury tokens are accepted
// ---------------------------------------------------------------------------

func TestJuryVote_AnyJuryTokenAuthorizes(t *testing.T) {
	type juryCred struct {
		token string
		email string
	}
	tokens := []juryCred{
		{token: "jury1", email: "jury1@test.com"},
		{token: "jury2", email: "jury2@test.com"},
		{token: "jury3", email: "jury3@test.com"},
	}

	for _, tok := range tokens {
		t.Run("token="+tok.token, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer mockDB.Close()
			server.DB = mockDB
			mock.MatchExpectationsInOrder(false)

			mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
				WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
			mock.ExpectExec("UPDATE Song").
				WillReturnResult(sqlmock.NewResult(0, 1))

			req := juryAuthRequest(http.MethodPost, "/jury/vote?songID=1&points=8", tok.token, tok.email)
			rr := httptest.NewRecorder()
			server.RequireJury(http.HandlerFunc(server.JuryVote)).ServeHTTP(rr, req)

			if rr.Code != http.StatusAccepted {
				t.Errorf("token=%s: expected 202, got %d — body: %s", tok.token, rr.Code, rr.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Response shape
// ---------------------------------------------------------------------------

func TestJuryVote_SuccessResponseShape(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT isOpen FROM Voting_Status").
		WillReturnRows(sqlmock.NewRows([]string{"isOpen"}).AddRow(true))
	mock.ExpectExec("UPDATE Song").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, juryVoteURL("1", "12"), nil)
	rr := httptest.NewRecorder()
	server.JuryVote(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	if _, ok := body["payload"]; !ok {
		t.Error("expected 'payload' key in response")
	}
}
