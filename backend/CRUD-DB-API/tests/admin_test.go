package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"crud-db-api/server"
)

// adminURL builds a URL with the test admin token.
func adminURL(path string) string {
	return path + "?Token=test-admin-pw"
}

func badTokenURL(path string) string {
	return path + "?Token=wrongtoken"
}

// ---------------------------------------------------------------------------
// AdminLogin
// ---------------------------------------------------------------------------

func TestAdminAuthenticate_CorrectToken_Returns202(t *testing.T) {
	os.Setenv("adminPassword", "test-admin-pw")

	req := httptest.NewRequest(http.MethodGet, "/admin/authenticate?Token=test-admin-pw", nil)
	rr := httptest.NewRecorder()
	server.AdminLogin(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestAdminAuthenticate_WrongToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, badTokenURL("/admin/authenticate"), nil)
	rr := httptest.NewRecorder()
	server.AdminLogin(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestAdminAuthenticate_NoToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/authenticate", nil)
	rr := httptest.NewRecorder()
	server.AdminLogin(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// OpenVote
// ---------------------------------------------------------------------------

func TestOpenVote_ValidToken_Returns202(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Voting_Status SET isOpen = true").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/open/"), nil)
	rr := httptest.NewRecorder()
	server.OpenVote(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestOpenVote_ValidToken_DBError_Returns502(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Voting_Status").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/open/"), nil)
	rr := httptest.NewRecorder()
	server.OpenVote(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rr.Code)
	}
}

func TestOpenVote_ValidToken_ZeroRowsAffected_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Voting_Status").WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/open/"), nil)
	rr := httptest.NewRecorder()
	server.OpenVote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestOpenVote_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, badTokenURL("/admin/open/"), nil)
	rr := httptest.NewRecorder()
	server.OpenVote(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// CloseVote
// ---------------------------------------------------------------------------

func TestCloseVote_ValidToken_Returns202(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Voting_Status SET isOpen = false").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/close"), nil)
	rr := httptest.NewRecorder()
	server.CloseVote(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestCloseVote_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, badTokenURL("/admin/close"), nil)
	rr := httptest.NewRecorder()
	server.CloseVote(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// DeleteVotes
// ---------------------------------------------------------------------------

func TestDeleteVotes_ValidToken_Returns202(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Song SET PublikumsPunkte").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE Phone_Nums SET votes_remaining").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, adminURL("/admin/deleteVotes/"), nil)
	rr := httptest.NewRecorder()
	server.DeleteVotes(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteVotes_ValidToken_NoVotes_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("UPDATE Song").WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, adminURL("/admin/deleteVotes/"), nil)
	rr := httptest.NewRecorder()
	server.DeleteVotes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteVotes_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, badTokenURL("/admin/deleteVotes/"), nil)
	rr := httptest.NewRecorder()
	server.DeleteVotes(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AddCountry
// ---------------------------------------------------------------------------

func TestAddCountry_ValidToken_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("INSERT INTO Land").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addCountry/?Token=test-admin-pw&ID=DE&Name=Germany&Pot=5", nil)
	rr := httptest.NewRecorder()
	server.AddCountry(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddCountry_InvalidPot_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addCountry/?Token=test-admin-pw&ID=DE&Name=Germany&Pot=notanumber", nil)
	rr := httptest.NewRecorder()
	server.AddCountry(rr, req)

	// Pot parsing happens BEFORE auth — so 400 regardless of token.
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAddCountry_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addCountry/?Token=wrong&ID=DE&Name=Germany&Pot=5", nil)
	rr := httptest.NewRecorder()
	server.AddCountry(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AddSong
// ---------------------------------------------------------------------------

func TestAddSong_ValidToken_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("INSERT INTO Song").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?Token=test-admin-pw&ID=1&Name=TestSong&Land=DE", nil)
	rr := httptest.NewRecorder()
	server.AddSong(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddSong_InvalidID_Returns406(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?Token=test-admin-pw&ID=notanumber&Name=TestSong&Land=DE", nil)
	rr := httptest.NewRecorder()
	server.AddSong(rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Errorf("expected 406, got %d", rr.Code)
	}
}

func TestAddSong_WithYoutubeURL_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("INSERT INTO Song").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?Token=test-admin-pw&ID=1&Name=TestSong&Land=DE&YoutubeURL=https://youtube.com/watch?v=test", nil)
	rr := httptest.NewRecorder()
	server.AddSong(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// AddArtist
// ---------------------------------------------------------------------------

func TestAddArtist_ValidToken_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("INSERT INTO Kuenstler").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addArtist/?Token=test-admin-pw&ID=1&Name=Mueller&vorName=Max&typ=solo&Land=DE", nil)
	rr := httptest.NewRecorder()
	server.AddArtist(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddArtist_InvalidID_Returns406(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addArtist/?Token=test-admin-pw&ID=notanumber&Name=Mueller&vorName=Max&typ=solo&Land=DE", nil)
	rr := httptest.NewRecorder()
	server.AddArtist(rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Errorf("expected 406, got %d", rr.Code)
	}
}

func TestAddArtist_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addArtist/?Token=wrong&ID=1&Name=Mueller&vorName=Max&typ=solo&Land=DE", nil)
	rr := httptest.NewRecorder()
	server.AddArtist(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AddInterpret
// ---------------------------------------------------------------------------

func TestAddInterpret_ValidToken_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectExec("INSERT INTO Komponist").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addInterpret/?Token=test-admin-pw&ID=1&Name=Wagner&Vorname=Hans", nil)
	rr := httptest.NewRecorder()
	server.AddInterpret(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddInterpret_InvalidID_Returns400(t *testing.T) {
	// Note: AddInterpret returns 400 (not 406) for invalid ID — different from AddArtist.
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addInterpret/?Token=test-admin-pw&ID=notanumber&Name=Wagner&Vorname=Hans", nil)
	rr := httptest.NewRecorder()
	server.AddInterpret(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// JuryLogin
// ---------------------------------------------------------------------------

func TestJuryAuthenticate_CorrectToken_Returns202(t *testing.T) {
	os.Setenv("juryPassword1", "jury1")
	req := httptest.NewRequest(http.MethodGet, "/jury/authenticate?Token=jury1", nil)
	rr := httptest.NewRecorder()
	server.JuryLogin(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestJuryAuthenticate_WrongToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/jury/authenticate?Token=notjurytoken", nil)
	rr := httptest.NewRecorder()
	server.JuryLogin(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}
