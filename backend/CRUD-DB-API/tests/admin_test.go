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

// adminReq builds a request with the admin token and email in the headers.
func adminReq(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer test-admin-pw")
	req.Header.Set("X-Email", "test-admin@test.com")
	return req
}

// badTokenReq builds a request with an invalid token in the Authorization header.
func badTokenReq(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer wrongtoken")
	return req
}

// ---------------------------------------------------------------------------
// AdminLogin
// ---------------------------------------------------------------------------

func TestAdminAuthenticate_CorrectToken_Returns202(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := adminReq(http.MethodGet, "/admin/authenticate")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeAdminLogin)).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestAdminAuthenticate_WrongToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := badTokenReq(http.MethodGet, "/admin/authenticate")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeAdminLogin)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestAdminAuthenticate_NoToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/authenticate", nil)
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeAdminLogin)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// OpenVote (no auth middleware on this route)
// ---------------------------------------------------------------------------

func TestOpenVote_Returns202(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Voting_Status SET isOpen = true").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/admin/open/", nil)
	rr := httptest.NewRecorder()
	h.ServeOpenVote(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestOpenVote_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Voting_Status").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, "/admin/open/", nil)
	rr := httptest.NewRecorder()
	h.ServeOpenVote(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestOpenVote_ZeroRowsAffected_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Voting_Status").WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodPost, "/admin/open/", nil)
	rr := httptest.NewRecorder()
	h.ServeOpenVote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Voting_Status SET isOpen = false").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/admin/close", nil)
	rr := httptest.NewRecorder()
	h.ServeCloseVote(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestCloseVote_InvalidToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := badTokenReq(http.MethodPost, "/admin/close")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeCloseVote)).ServeHTTP(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Song SET PublikumsPunkte").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE Phone_Nums SET votes_remaining").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/admin/deleteVotes/", nil)
	rr := httptest.NewRecorder()
	h.ServeDeleteVotes(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("UPDATE Song").WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/admin/deleteVotes/", nil)
	rr := httptest.NewRecorder()
	h.ServeDeleteVotes(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteVotes_InvalidToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := badTokenReq(http.MethodDelete, "/admin/deleteVotes/")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeDeleteVotes)).ServeHTTP(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("INSERT INTO Land").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addCountry/?ID=DE&Name=Germany&Pot=5", nil)
	rr := httptest.NewRecorder()
	h.ServeAddCountry(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddCountry_InvalidPot_Returns422(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addCountry/?ID=DE&Name=Germany&Pot=notanumber", nil)
	rr := httptest.NewRecorder()
	h.ServeAddCountry(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestAddCountry_InvalidToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := badTokenReq(http.MethodPost, "/admin/addCountry/?ID=DE&Name=Germany&Pot=5")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeAddCountry)).ServeHTTP(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("INSERT INTO Song").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?KuenstlerID=1&SongName=TestSong&CountryID=DE", nil)
	rr := httptest.NewRecorder()
	h.ServeAddSong(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddSong_InvalidKuenstlerID_Returns422(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?KuenstlerID=notanumber&SongName=TestSong&CountryID=DE", nil)
	rr := httptest.NewRecorder()
	h.ServeAddSong(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestAddSong_WithYoutubeURL_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("INSERT INTO Song").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addSong/?KuenstlerID=1&SongName=TestSong&CountryID=DE&YoutubeURL=https://youtube.com/watch?v=test", nil)
	rr := httptest.NewRecorder()
	h.ServeAddSong(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("INSERT INTO Kuenstler").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addArtist/?FirstName=Max&LastName=Mueller&Type=solo&CountryID=DE", nil)
	rr := httptest.NewRecorder()
	h.ServeAddArtist(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddArtist_InvalidCountryID_Returns422(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addArtist/?FirstName=Max&LastName=Mueller&Type=solo&CountryID=DEU", nil)
	rr := httptest.NewRecorder()
	h.ServeAddArtist(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestAddArtist_InvalidToken_Returns403(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := badTokenReq(http.MethodPost, "/admin/addArtist/?FirstName=Max&LastName=Mueller&Type=solo&CountryID=DE")
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(h.ServeAddArtist)).ServeHTTP(rr, req)

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
	h := newTestHandlers(t, mockDB)

	mock.ExpectExec("INSERT INTO Komponist").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost,
		"/admin/addInterpret/?ID=1&Name=Wagner&Vorname=Hans", nil)
	rr := httptest.NewRecorder()
	h.ServeAddInterpret(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestAddInterpret_InvalidID_Returns400(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost,
		"/admin/addInterpret/?ID=notanumber&Name=Wagner&Vorname=Hans", nil)
	rr := httptest.NewRecorder()
	h.ServeAddInterpret(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// JuryLogin (auth is handled by RequireJury middleware via Bearer header)
// ---------------------------------------------------------------------------

func TestJuryAuthenticate_CorrectToken_Returns202(t *testing.T) {
	h, err := server.HashPassword("jury1")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("juryPassword1", h)
	hndlr := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/jury/authenticate", nil)
	req.Header.Set("Authorization", "Bearer jury1")
	req.Header.Set("X-Email", "jury1@test.com")
	rr := httptest.NewRecorder()
	server.RequireJury(http.HandlerFunc(hndlr.ServeJuryLogin)).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}

func TestJuryAuthenticate_WrongToken_Returns403(t *testing.T) {
	hndlr := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/jury/authenticate", nil)
	req.Header.Set("Authorization", "Bearer notjurytoken")
	rr := httptest.NewRecorder()
	server.RequireJury(http.HandlerFunc(hndlr.ServeJuryLogin)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestJuryAuthenticate_NoToken_Returns403(t *testing.T) {
	hndlr := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/jury/authenticate", nil)
	rr := httptest.NewRecorder()
	server.RequireJury(http.HandlerFunc(hndlr.ServeJuryLogin)).ServeHTTP(rr, req)

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
