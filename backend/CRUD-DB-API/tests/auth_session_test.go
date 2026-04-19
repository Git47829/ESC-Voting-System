package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"crud-db-api/server"
)

func cookieByName(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func loginRecorder(t *testing.T, email, password, role string) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(`{"email":"` + email + `","password":"` + password + `","role":"` + role + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.AuthLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	return rr
}

func TestAuthLogin_SetsHttpOnlyCookies(t *testing.T) {
	rr := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")

	access := cookieByName(rr, "esc_access_token")
	refresh := cookieByName(rr, "esc_refresh_token")
	if access == nil || refresh == nil {
		t.Fatal("expected access and refresh cookies")
	}
	if !access.HttpOnly || !refresh.HttpOnly {
		t.Fatal("expected auth cookies to be HttpOnly")
	}
	if access.SameSite != http.SameSiteLaxMode || refresh.SameSite != http.SameSiteLaxMode {
		t.Fatal("expected auth cookies to use SameSite=Lax")
	}
	if access.Path != "/" || refresh.Path != "/" {
		t.Fatal("expected auth cookies to have Path=/")
	}
	if access.Secure || refresh.Secure {
		t.Fatal("expected auth cookies to be non-secure in test setup")
	}
}

func TestAuthVerify_AcceptsValidAccessCookie(t *testing.T) {
	rrLogin := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")
	access := cookieByName(rrLogin, "esc_access_token")
	if access == nil {
		t.Fatal("missing access token cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	req.AddCookie(access)
	rr := httptest.NewRecorder()
	server.RequireAuth(http.HandlerFunc(server.AuthVerify)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	json.NewDecoder(rr.Body).Decode(&body)
	if body["authenticated"] != true {
		t.Fatalf("expected authenticated=true, got %v", body["authenticated"])
	}
}

func TestAuthRefresh_ReusesRefreshTokenSession(t *testing.T) {
	rrLogin := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")
	oldAccess := cookieByName(rrLogin, "esc_access_token")
	oldRefresh := cookieByName(rrLogin, "esc_refresh_token")

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(oldRefresh)
	rr := httptest.NewRecorder()
	server.AuthRefresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	newAccess := cookieByName(rr, "esc_access_token")
	newRefresh := cookieByName(rr, "esc_refresh_token")
	if newAccess == nil {
		t.Fatal("expected refreshed access cookie")
	}
	if newAccess.Value == oldAccess.Value {
		t.Fatal("expected a new access token")
	}
	if newRefresh != nil {
		t.Fatal("did not expect refresh cookie rotation")
	}

	// Same refresh token remains valid until explicit logout.
	reqAgain := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	reqAgain.AddCookie(oldRefresh)
	rrAgain := httptest.NewRecorder()
	server.AuthRefresh(rrAgain, reqAgain)
	if rrAgain.Code != http.StatusOK {
		t.Fatalf("expected repeated refresh with same token to succeed, got %d body=%s", rrAgain.Code, rrAgain.Body.String())
	}
}

func TestAuthRefresh_MissingCookieReturnsForbidden(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	rr := httptest.NewRecorder()
	server.AuthRefresh(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthRefresh_RejectsAccessTokenAsRefreshToken(t *testing.T) {
	rrLogin := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")
	access := cookieByName(rrLogin, "esc_access_token")
	if access == nil {
		t.Fatal("missing access token cookie")
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "esc_refresh_token", Value: access.Value})
	rr := httptest.NewRecorder()
	server.AuthRefresh(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthRefresh_RejectsTamperedRefreshToken(t *testing.T) {
	rrLogin := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")
	refresh := cookieByName(rrLogin, "esc_refresh_token")
	if refresh == nil {
		t.Fatal("missing refresh token cookie")
	}
	if len(refresh.Value) < 10 {
		t.Fatal("refresh token too short to tamper")
	}
	// Flip a character in the middle of the token to avoid base64 padding-bit
	// coincidences that can occur when only the last character is changed.
	mid := len(refresh.Value) / 2
	flipChar := byte('A')
	if refresh.Value[mid] == 'A' {
		flipChar = 'B'
	}
	tampered := refresh.Value[:mid] + string(flipChar) + refresh.Value[mid+1:]

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "esc_refresh_token", Value: tampered})
	rr := httptest.NewRecorder()
	server.AuthRefresh(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthLogout_RevokesRefreshSession(t *testing.T) {
	rrLogin := loginRecorder(t, "test-admin@test.com", "test-admin-pw", "admin")
	refresh := cookieByName(rrLogin, "esc_refresh_token")

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(refresh)
	logoutRR := httptest.NewRecorder()
	server.AuthLogout(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("expected logout 200, got %d", logoutRR.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	refreshReq.AddCookie(refresh)
	refreshRR := httptest.NewRecorder()
	server.AuthRefresh(refreshRR, refreshReq)
	if refreshRR.Code != http.StatusForbidden {
		t.Fatalf("expected refresh after logout to fail with 403, got %d", refreshRR.Code)
	}
}

func TestAuthLogout_WithoutRefreshCookieStillClearsCookies(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()
	server.AuthLogout(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	access := cookieByName(rr, "esc_access_token")
	refresh := cookieByName(rr, "esc_refresh_token")
	if access == nil || refresh == nil {
		t.Fatal("expected both auth cookies to be cleared")
	}
	if access.MaxAge >= 0 || refresh.MaxAge >= 0 {
		t.Fatal("expected logout cookies to be expired")
	}
	if !access.Expires.Equal(time.Unix(0, 0)) || !refresh.Expires.Equal(time.Unix(0, 0)) {
		t.Fatal("expected logout cookies to expire at unix epoch")
	}
}

func TestRequireAdmin_RejectsJuryToken(t *testing.T) {
	rrLogin := loginRecorder(t, "jury1@test.com", "jury1", "jury")
	juryAccess := cookieByName(rrLogin, "esc_access_token")

	req := httptest.NewRequest(http.MethodGet, "/admin/authenticate", nil)
	req.AddCookie(juryAccess)
	rr := httptest.NewRecorder()
	server.RequireAdmin(http.HandlerFunc(server.AdminLogin)).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestGetResults_AggregatesAndRanks(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	server.DB = mockDB

	rows := sqlmock.NewRows([]string{"ID", "Name", "CountryName", "CountryID", "PublikumsPunkte", "JuryPunkte"}).
		AddRow(1, "Song A", "Germany", "DE", 100, 4).
		AddRow(2, "Song B", "France", "FR", 50, 12).
		AddRow(3, "Song C", "Spain", "ES", 0, 8)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/results", nil)
	rr := httptest.NewRecorder()
	server.GetResults(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(payload))
	}
	if payload[0]["id"].(float64) != 2 || payload[0]["rank"].(float64) != 1 {
		t.Fatalf("expected song 2 at rank 1, got %+v", payload[0])
	}
	if payload[0]["totalPts"].(float64) != 42 {
		t.Fatalf("expected song 2 totalPts=42, got %v", payload[0]["totalPts"])
	}
	if payload[2]["escPublicPts"].(float64) != 0 {
		t.Fatalf("expected zero public ESC points for zero votes, got %v", payload[2]["escPublicPts"])
	}
}
