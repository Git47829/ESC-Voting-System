package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"crud-db-api/server"
)

// ---------------------------------------------------------------------------
// StartContest
// ---------------------------------------------------------------------------

func TestStartContest_ValidToken_Returns201(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	rows := sqlmock.NewRows([]string{"ID"}).AddRow(1).AddRow(2).AddRow(3)
	mock.ExpectQuery("SELECT ID FROM Song").WillReturnRows(rows)
	mock.ExpectExec("UPDATE Contest_Run SET IsActive = FALSE").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO Contest_Run").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.StartContest(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestStartContest_ValidToken_ResponseShape(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	rows := sqlmock.NewRows([]string{"ID"}).AddRow(1).AddRow(2).AddRow(3)
	mock.ExpectQuery("SELECT ID FROM Song").WillReturnRows(rows)
	mock.ExpectExec("UPDATE Contest_Run").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO Contest_Run").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.StartContest(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	if _, ok := body["songCount"]; !ok {
		t.Error("expected 'songCount' in response")
	}
	if _, ok := body["order"]; !ok {
		t.Error("expected 'order' in response")
	}
}

func TestStartContest_SongOrderIsPermutation(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	rows := sqlmock.NewRows([]string{"ID"}).AddRow(1).AddRow(2).AddRow(3)
	mock.ExpectQuery("SELECT ID FROM Song").WillReturnRows(rows)
	mock.ExpectExec("UPDATE Contest_Run").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO Contest_Run").WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.StartContest(rr, req)

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	orderRaw, ok := body["order"].([]any)
	if !ok {
		t.Fatal("expected 'order' to be an array")
	}

	got := make([]int, len(orderRaw))
	for i, v := range orderRaw {
		got[i] = int(v.(float64))
	}
	sort.Ints(got)
	expected := []int{1, 2, 3}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("order is not a permutation of [1,2,3]: got %v", got)
			break
		}
	}
}

func TestStartContest_NoSongs_Returns422(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID FROM Song").
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.StartContest(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestStartContest_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, badTokenURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(server.StartContest)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestStartContest_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID FROM Song").WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/startContest/"), nil)
	rr := httptest.NewRecorder()
	server.StartContest(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AdvanceContest
// ---------------------------------------------------------------------------

func TestAdvanceContest_ActiveContest_Returns200(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}).
			AddRow(1, `[3,1,2]`, 0))
	mock.ExpectExec("UPDATE Contest_Run SET CurrentIndex").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/advanceContest/"), nil)
	rr := httptest.NewRecorder()
	server.AdvanceContest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)

	if body["finished"] != false {
		t.Errorf("expected finished=false, got %v", body["finished"])
	}
	if body["currentIndex"].(float64) != 1 {
		t.Errorf("expected currentIndex=1, got %v", body["currentIndex"])
	}
	if body["songId"].(float64) != 1 {
		t.Errorf("expected songId=1 (ids[1]), got %v", body["songId"])
	}
}

func TestAdvanceContest_LastSong_ReturnsFinished(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	// currentIndex=2, len(ids)=3 → nextIndex=3 >= 3 → finished
	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}).
			AddRow(1, `[1,2,3]`, 2))
	mock.ExpectExec("UPDATE Contest_Run SET IsActive = FALSE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/advanceContest/"), nil)
	rr := httptest.NewRecorder()
	server.AdvanceContest(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	if body["finished"] != true {
		t.Errorf("expected finished=true, got %v", body["finished"])
	}
}

func TestAdvanceContest_NoActiveContest_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}))

	req := httptest.NewRequest(http.MethodPost, adminURL("/admin/advanceContest/"), nil)
	rr := httptest.NewRecorder()
	server.AdvanceContest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAdvanceContest_InvalidToken_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, badTokenURL("/admin/advanceContest/"), nil)
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(server.AdvanceContest)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetCurrentSong
// ---------------------------------------------------------------------------

func TestGetCurrentSong_ActiveContest_Returns200WithAllFields(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}).
			AddRow(1, `[5,3,1]`, 1))

	songCols := []string{
		"s.ID", "s.Name", "s.PublikumsPunkte", "s.JuryPunkte", "s.GesamtPunkte",
		"COALESCE(s.YoutubeURL,'')",
		"l.ID", "l.Name",
		"k.ID", "k.Vorname", "k.Name", "k.Typ",
		"vs.isOpen",
	}
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows(songCols).
			AddRow(3, "Northern Lights", 80, 60, 140, "https://youtube.com/embed/test",
				"SE", "Sweden", 2, "Alice", "Lindgren", "duo", true))

	req := httptest.NewRequest(http.MethodGet, "/contest/current/", nil)
	rr := httptest.NewRecorder()
	server.GetCurrentSong(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	payload, ok := body["payload"].(map[string]any)
	if !ok {
		t.Fatal("expected payload to be an object")
	}

	for _, key := range []string{
		"runId", "currentIndex", "totalSongs", "contestActive", "currentSong",
	} {
		if _, ok := payload[key]; !ok {
			t.Errorf("expected key %q in payload", key)
		}
	}

	if payload["contestActive"] != true {
		t.Errorf("expected contestActive=true, got %v", payload["contestActive"])
	}

	currentSong, ok := payload["currentSong"].(map[string]any)
	if !ok {
		t.Fatal("expected currentSong to be an object")
	}

	for _, key := range []string{
		"songId", "songName", "youtubeUrl",
		"countryId", "countryName",
		"artistId", "artistFirstName", "artistLastName", "artistType",
		"publicVotes", "juryVotes", "totalVotes", "votingIsOpen",
	} {
		if _, ok := currentSong[key]; !ok {
			t.Errorf("expected key %q in currentSong", key)
		}
	}

	// currentIndex=1, ids=[5,3,1] → current song ID = 3
	if currentSong["songId"].(float64) != 3 {
		t.Errorf("expected songId=3, got %v", currentSong["songId"])
	}
}

func TestGetCurrentSong_NoActiveContest_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}))

	req := httptest.NewRequest(http.MethodGet, "/contest/current/", nil)
	rr := httptest.NewRecorder()
	server.GetCurrentSong(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetCurrentSong_ContestEnded_Returns410(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	// currentIndex=3 >= len([1,2,3])=3 → contest ended
	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}).
			AddRow(1, `[1,2,3]`, 3))

	req := httptest.NewRequest(http.MethodGet, "/contest/current/", nil)
	rr := httptest.NewRecorder()
	server.GetCurrentSong(rr, req)

	if rr.Code != http.StatusGone {
		t.Errorf("expected 410, got %d", rr.Code)
	}
}

func TestGetCurrentSong_SongNotFoundInDB_Returns404(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "SongOrder", "CurrentIndex"}).
			AddRow(1, `[99]`, 0))

	// Song query returns no rows.
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{
			"s.ID", "s.Name", "s.PublikumsPunkte", "s.JuryPunkte", "s.GesamtPunkte",
			"COALESCE(s.YoutubeURL,'')", "l.ID", "l.Name",
			"k.ID", "k.Vorname", "k.Name", "k.Typ", "vs.isOpen",
		}))

	req := httptest.NewRequest(http.MethodGet, "/contest/current/", nil)
	rr := httptest.NewRecorder()
	server.GetCurrentSong(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetCurrentSong_DBError_Returns500(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	mock.ExpectQuery("SELECT ID, SongOrder, CurrentIndex FROM Contest_Run").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/contest/current/", nil)
	rr := httptest.NewRecorder()
	server.GetCurrentSong(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
