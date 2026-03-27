package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/nyaruka/phonenumbers"
)

type Client struct {
	limiter *rate.Limiter
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

var clients = make(map[string]*Client)

var (
	rateLimitConfigs = map[string]RateLimitConfig{
		"GET /health":          {RequestsPerSecond: 100, BurstSize: 100},
		"GET /votes/":          {RequestsPerSecond: 10, BurstSize: 20},
		"GET /countries/":      {RequestsPerSecond: 10, BurstSize: 20},
		"GET /songs/":          {RequestsPerSecond: 10, BurstSize: 20},
		"GET /songByID/{ID}":   {RequestsPerSecond: 10, BurstSize: 20},
		"GET /contest/current": {RequestsPerSecond: 10, BurstSize: 20},

		"POST /vote/":             {RequestsPerSecond: 1, BurstSize: 1},
		"POST /jury/vote":         {RequestsPerSecond: 5, BurstSize: 5},
		"GET /jury/authenticate":  {RequestsPerSecond: 1, BurstSize: 1},
		"GET /admin/authenticate": {RequestsPerSecond: 1, BurstSize: 1},

		"POST /admin/open/":          {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/close":          {RequestsPerSecond: 2, BurstSize: 2},
		"POST /admin/addCountry":     {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addSong":        {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addArtist":      {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/addInterpret":   {RequestsPerSecond: 5, BurstSize: 5},
		"POST /admin/startContest":   {RequestsPerSecond: 5, BurstSize: 5},
		"POST (admin/advanceContest": {RequestsPerSecond: 5, BurstSize: 5},
		"DELETE /admin/deleteVotes":  {RequestsPerSecond: 1, BurstSize: 1},

		"GET /metrics/": {RequestsPerSecond: 10000, BurstSize: 10000},
	}
)

var (
	usedTokens = make(map[string]bool)
	mu         sync.Mutex
)

const totalVotePoints = 20

type CookieVoteState struct {
	VotesRemaining int            `json:"votes_remaining"`
	VotesCast      map[string]int `json:"votes_cast"`
}

var SignedCookieSecret []byte

// User AdminPassword als signing Secret
func InitCookieSecret() {
	if pw := os.Getenv("adminPassword"); pw != "" {
		sum := sha256.Sum256([]byte("cookie-secret:" + pw))
		SignedCookieSecret = sum[:]
		return
	}
	SignedCookieSecret = make([]byte, 32)
	if _, err := rand.Read(SignedCookieSecret); err != nil {
		panic("cannot generate cookie secret: " + err.Error())
	}
}

func EncodeCookieValue(state CookieVoteState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, SignedCookieSecret)
	mac.Write(data)
	sig := mac.Sum(nil)
	payload := append(data, '.')
	payload = append(payload, []byte(hex.EncodeToString(sig))...)
	return hex.EncodeToString(payload), nil
}

func getCLientLimiter(ip string, endpoint string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	key := ip + "::" + endpoint
	if client, exists := clients[key]; exists {
		return client.limiter
	}

	config, exists := rateLimitConfigs[endpoint]
	if !exists {
		config = RateLimitConfig{RequestsPerSecond: 10, BurstSize: 20}
	}

	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.BurstSize)
	clients[key] = &Client{limiter: limiter}
	return limiter
}

func RateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		clientIP := r.RemoteAddr

		limiter := getCLientLimiter(clientIP, endpoint)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})

			Logger.WarnContext(r.Context(), "rate limit exeeded",
				slog.String("ip", clientIP),
				slog.String("endpoint", endpoint),
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CheckPhoneNum(num string) (string, error) {

	parsed, err := phonenumbers.Parse(num, "")
	if err != nil {
		return "", nil
	}

	if !phonenumbers.IsValidNumber(parsed) {
		return "", nil
	}

	numRegion := phonenumbers.GetRegionCodeForNumber(parsed)

	return numRegion, nil
}

func HashPassword(password string) (string, error) {
	sum, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sum), nil
}

func CheckPassword(password, storedToken string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedToken), []byte(password))
	return err == nil
}

func CheckAccessAdmin(input string) (bool, string) {

	adminPassword := os.Getenv("adminPassword")

	if input == "" {
		return false, "Token has to be provided"
	}
	if CheckPassword(input, adminPassword) {
		return true, "Authorized"
	}

	return false, "Wrong Token provided"
}

func CheckAccessJury(input string) (bool, string) {

	juryPassword1 := os.Getenv("juryPassword1")
	juryPassword2 := os.Getenv("juryPassword2")
	juryPassword3 := os.Getenv("juryPassword3")

	TestToken := []string{juryPassword1, juryPassword2, juryPassword3}

	results := make(chan bool, len(TestToken))

	var wg sync.WaitGroup

	for _, token := range TestToken {
		wg.Add(1)
		t := token
		go func() {
			defer wg.Done()
			results <- CheckPassword(input, t)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	authorized := false

	for matched := range results {
		if matched {
			authorized = true
		}
	}

	if authorized {
		return true, "Authorized"
	}
	return false, "Wrong Token Provided"
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Success",
		"status":  "healthy",
	})

}
func GetVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT
			s.ID,
			s.Name,
			l.Name as Country,
			s.PublikumsPunkte,
			s.JuryPunkte,
			s.GesamtPunkte
		FROM Song s
		JOIN Land l on s.Land_ID = l.ID
		ORDER BY s.GesamtPunkte DESC
	`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to retrieve Votes",
		})
		return
	}
	defer rows.Close()

	type SongVote struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		Country         string `json:"country"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`
	}

	var votes []SongVote
	for rows.Next() {
		var v SongVote
		if err := rows.Scan(&v.ID, &v.Name, &v.Country, &v.PublikumsPunkte, &v.JuryPunkte, &v.GesamtPunkte); err != nil {
			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		votes = append(votes, v)
	}

	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process votes",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": votes,
	})
}

func Vote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	const cookieName = "vote_state"

	ownCountry := r.URL.Query().Get("ownCountry")
	phone := r.URL.Query().Get("phoneNum")

	if len(ownCountry) != 2 {
		Logger.ErrorContext(ctx, "Country ID can only be two Characters in Length", slog.String("CountryID:", ownCountry))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Country code can only be two Charakters in length",
		})
	}

	phoneCountry, phoneErr := CheckPhoneNum(phone)
	if phoneErr != nil {
		Logger.WarnContext(ctx, "phone number failed validation", slog.String("phone", phone), slog.Any("error", phoneErr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": phoneErr.Error()})
		return
	}
	if phoneCountry == "" {
		Logger.WarnContext(ctx, "invalid phone number provided", slog.String("phone", phone))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid phone number"})
		return
	}

	phoneHash, _ := HashPassword(phone)

	rawID := r.URL.Query().Get("songID")
	songID, err := strconv.Atoi(rawID)
	if err != nil {
		Logger.ErrorContext(ctx, "songID must be an integer", slog.Any("error", err), slog.String("songID", rawID))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid songID value - must be an integer"})
		return
	}

	rawPoints := r.URL.Query().Get("points")
	points, err := strconv.Atoi(rawPoints)
	if err != nil || points < 1 || points > totalVotePoints {
		Logger.WarnContext(ctx, "invalid points value", slog.String("points", rawPoints))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("points must be an integer between 1 and %d", totalVotePoints),
		})
		return
	}

	var (
		votingIsOpen bool
		songLandID   string
		songFound    bool
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := DB.QueryRowContext(gctx, `SELECT isOpen FROM Voting_Status`).Scan(&votingIsOpen)
		if err != nil {
			Logger.ErrorContext(gctx, "failed to query voting status", slog.Any("error", err))
		}
		return err
	})

	g.Go(func() error {
		err := DB.QueryRowContext(gctx, `SELECT Land_ID FROM Song WHERE ID = ?`, songID).Scan(&songLandID)
		if err == sql.ErrNoRows {
			songFound = false
			return nil
		}
		if err != nil {
			Logger.ErrorContext(gctx, "error querying song", slog.Any("error", err), slog.Int("songID", songID))
			return err
		}
		songFound = true
		return nil
	})

	if err := g.Wait(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "error querying DB",
		})
		return
	}

	if !votingIsOpen {
		w.WriteHeader(http.StatusTooEarly)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "voting has not started yet, please try again in a few minutes",
		})
		return
	}

	if !songFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Song not found",
		})
		return
	}

	if songLandID == phoneCountry {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "cannot vote for your own country",
		})
		return
	}

	upsertQuery := `
		INSERT INTO Phone_Nums (Phone_Number, votes_remaining, votes_cast)
		VALUES (?, ?, JSON_OBJECT())
		ON DUPLICATE KEY UPDATE Phone_Number = Phone_Number`

	if _, upsertErr := DB.ExecContext(ctx, upsertQuery, phoneHash, totalVotePoints); upsertErr != nil {
		Logger.ErrorContext(ctx, "failed to upsert phone number", slog.Any("error", upsertErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	deductQuery := `
		UPDATE Phone_Nums
		SET
			votes_remaining = votes_remaining - ?,
			votes_cast = JSON_SET(
				COALESCE(votes_cast, JSON_OBJECT()),
				CONCAT('$."', ?, '"'),
				COALESCE(JSON_EXTRACT(votes_cast, CONCAT('$."', ?, '"')), 0) + ?
			)
		WHERE Phone_Number = ? AND votes_remaining >= ?`

	deductResult, deductErr := DB.ExecContext(ctx, deductQuery,
		points,    // subtract
		rawID,     // JSON key (song ID as string)
		rawID,     // JSON key for existing value lookup
		points,    // add to existing tally for this song
		phoneHash, // WHERE clause
		points,    // votes_remaining >= points guard
	)

	if deductErr != nil {
		Logger.ErrorContext(ctx, "failed to deduct vote points", slog.Any("error", deductErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	rowsAffected, _ := deductResult.RowsAffected()
	if rowsAffected == 0 {
		var remaining int
		DB.QueryRowContext(ctx,
			`SELECT votes_remaining FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash,
		).Scan(&remaining)

		Logger.WarnContext(ctx, "insufficient vote points",
			slog.String("phone_hash", phoneHash),
			slog.Int("requested", points),
			slog.Int("remaining", remaining),
		)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error":           fmt.Sprintf("not enough vote points remaining (have %d, requested %d)", remaining, points),
			"votes_remaining": remaining,
		})
		return
	}

	var (
		remaining   int
		votesCast   = make(map[string]int)
		songUpdated bool
		cookieVal   string
	)

	g, cctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var votesCastJSON sql.NullString
		err := DB.QueryRowContext(cctx, `SELECT votes_remaining, votes_cast FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash).Scan(&remaining, &votesCastJSON)
		if err != nil {
			Logger.ErrorContext(cctx, "failed to read back vote state", slog.Any("error", err))
			return err
		}

		if votesCastJSON.Valid && votesCastJSON.String != "" {
			if err := json.Unmarshal([]byte(votesCastJSON.String), &votesCast); err != nil {
				Logger.ErrorContext(cctx, "failed to umarshal votes_cast", slog.Any("error", err))
			}
		}

		state := CookieVoteState{
			VotesRemaining: remaining,
			VotesCast:      votesCast,
		}

		var encErr error
		cookieVal, encErr = EncodeCookieValue(state)
		if encErr != nil {
			Logger.ErrorContext(cctx, "failed to encode vote cookie", slog.Any("error", encErr))
		}
		return nil
	})

	g.Go(func() error {
		const weightPublicVote = 1
		result, err := DB.ExecContext(cctx,
			`UPDATE Song SET PublikumsPunkte = PublikumsPunkte + ? WHERE ID = ?`,
			points*weightPublicVote, songID,
		)
		if err != nil {
			Logger.ErrorContext(gctx, "failed to update song votes", slog.Any("error", err))
			return err
		}
		updatedRows, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(gctx, "failed to get affected rows", slog.Any("error", err))
			return err
		}
		songUpdated = updatedRows > 0
		return nil
	})

	if err := g.Wait(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	if !songUpdated {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	if cookieVal != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    cookieVal,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})
	}

	NotifyVote(songID, ownCountry, DB)

	Logger.InfoContext(ctx, "vote cast",
		slog.String("phone_hash", phoneHash),
		slog.Int("song_id", songID),
		slog.Int("points_spent", points),
		slog.Int("votes_remaining", remaining),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "Vote cast successfully",
		"voted_for":       songID,
		"points_spent":    points,
		"votes_remaining": remaining,
		"votes_cast":      votesCast,
	})
}

func GetCountries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `SELECT ID, Name, POT FROM Land ORDER BY Name ASC`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query countries", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to retrieve countries",
		})
		return
	}
	defer rows.Close()

	type Country struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Pot  *int   `json:"pot"`
	}

	var countries []Country
	for rows.Next() {
		var c Country
		var potValue sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &potValue); err != nil {
			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if potValue.Valid {
			pot := int(potValue.Int64)
			c.Pot = &pot
		}
		countries = append(countries, c)
	}

	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process countries",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func GetCountryByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("NAME")

	if len(idStr) != 2 {
		Logger.ErrorContext(ctx, "Country code can only be two Charakters in length", slog.String("CountryID", idStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Country must be two Characters in length",
		})
	}

	query := `SELECT ID, Name, POT FROM Land WHERE ID = ?`

	rows, err := DB.QueryContext(ctx, query, idStr)

	if err != nil {
		Logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error":       "Could not retrieve Country with ID",
			"requestedID": idStr,
		})
		return
	}
	defer rows.Close()

	type Country struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Pot  *int   `json:"pot"`
	}

	var countries []Country
	for rows.Next() {
		var c Country
		var potValue sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &potValue); err != nil {
			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if potValue.Valid {
			pot := int(potValue.Int64)
			c.Pot = &pot
		}
		countries = append(countries, c)
	}

	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process countries",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": countries,
	})
}

func HttpGetSongs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := `SELECT
    s.ID AS Song_ID,
    s.Name AS Song_Name,
    s.PublikumsPunkte,
    s.JuryPunkte,
    s.GesamtPunkte,
    l.ID AS Land_ID,
    l.Name AS Land_Name,
    l.POT AS Land_POT,
    k.ID AS Kuenstler_ID,
    k.Vorname AS Kuenstler_Vorname,
    k.Name AS Kuenstler_Name,
    k.Typ AS Kuenstler_Typ,
    komp.ID AS Komponist_ID,
    komp.Vorname AS Komponist_Vorname,
    komp.Name AS Komponist_Name,
    vs.VotingID,
    vs.isOpen AS Voting_IsOpen,
    vs.lastChange AS Voting_LastChange
    FROM Song s
    INNER JOIN Land l ON s.Land_ID = l.ID
    INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
    LEFT JOIN Song_Komponist sk ON s.ID = sk.Song_ID
    LEFT JOIN Komponist komp ON sk.Komponist_ID = komp.ID
    LEFT JOIN Voting_Status vs ON vs.VotingID = 1  -- Assuming VotingID 1 is the current contest
    ORDER BY s.ID, komp.ID;
`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		Logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "Error querying Database",
		})
		return
	}
	defer rows.Close()

	type Komponist struct {
		ID      int `json:"id"`
		vorname string
		name    string
	}

	type CompleteESCEntryWithComposers struct {
		SongID          int    `json:"songId"`
		SongName        string `json:"songName"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`

		LandID   string `json:"countryId"`
		LandName string `json:"countryName"`
		LandPOT  *int   `json:"countryPot,omitempty"`

		KuenstlerID      int    `json:"artistId"`
		KuenstlerVorname string `json:"artistFirstName"`
		KuenstlerName    string `json:"artistLastName"`
		KuenstlerTyp     string `json:"artistType"`

		Komponisten []Komponist `json:"composers"`

		VotingID         int    `json:"votingId"`
		VotingIsOpen     bool   `json:"votingIsOpen"`
		VotingLastChange string `json:"votingLastChange"`
	}

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var komponentID sql.NullInt64
		var komponentVorname sql.NullString
		var komponentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID,
			&c.LandName, &c.LandPOT, &c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &komponentID,
			&komponentVorname, &komponentName, &c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {

			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if komponentID.Valid {
			komponent := Komponist{
				ID:      int(komponentID.Int64),
				vorname: komponentVorname.String,
				name:    komponentName.String,
			}
			c.Komponisten = append(c.Komponisten, komponent)
		}
		song = append(song, c)
	}

	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process Song",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": song,
	})
}

func GetSongByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("ID")
	ID, err := strconv.Atoi(idStr)
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid ID Value", slog.Any("error", err), slog.String("ID", idStr))
	}

	query := `SELECT
    s.ID AS Song_ID,
    s.Name AS Song_Name,
    s.PublikumsPunkte,
    s.JuryPunkte,
    s.GesamtPunkte,
    l.ID AS Land_ID,
    l.Name AS Land_Name,
    l.POT AS Land_POT,
    k.ID AS Kuenstler_ID,
    k.Vorname AS Kuenstler_Vorname,
    k.Name AS Kuenstler_Name,
    k.Typ AS Kuenstler_Typ,
    komp.ID AS Komponist_ID,
    komp.Vorname AS Komponist_Vorname,
    komp.Name AS Komponist_Name,
    vs.VotingID,
    vs.isOpen AS Voting_IsOpen,
    vs.lastChange AS Voting_LastChange
    FROM Song s
    INNER JOIN Land l ON s.Land_ID = l.ID
    INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
    LEFT JOIN Song_Komponist sk ON s.ID = sk.Song_ID
    LEFT JOIN Komponist komp ON sk.Komponist_ID = komp.ID
    LEFT JOIN Voting_Status vs ON vs.VotingID = 1
    WHERE s.ID = ?
    ORDER BY s.ID, komp.ID;
`

	rows, err := DB.QueryContext(ctx, query, ID)
	if err != nil {
		Logger.ErrorContext(ctx, "Error Querying Database", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "Faules querying Database",
		})
		return
	}
	defer rows.Close()

	type Komponist struct {
		ID      int `json:"id"`
		vorname string
		name    string
	}

	type CompleteESCEntryWithComposers struct {
		SongID          int    `json:"songId"`
		SongName        string `json:"songName"`
		PublikumsPunkte int    `json:"publicVotes"`
		JuryPunkte      int    `json:"juryVotes"`
		GesamtPunkte    int    `json:"totalVotes"`

		LandID   string `json:"countryId"`
		LandName string `json:"countryName"`
		LandPOT  *int   `json:"countryPot,omitempty"`

		KuenstlerID      int    `json:"artistId"`
		KuenstlerVorname string `json:"artistFirstName"`
		KuenstlerName    string `json:"artistLastName"`
		KuenstlerTyp     string `json:"artistType"`

		Komponisten []Komponist `json:"composers"`

		VotingID         int    `json:"votingId"`
		VotingIsOpen     bool   `json:"votingIsOpen"`
		VotingLastChange string `json:"votingLastChange"`
	}

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var komponentID sql.NullInt64
		var komponentVorname sql.NullString
		var komponentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublikumsPunkte, &c.JuryPunkte, &c.GesamtPunkte, &c.LandID, &c.LandName, &c.LandPOT,
			&c.KuenstlerID, &c.KuenstlerVorname, &c.KuenstlerName, &c.KuenstlerTyp, &komponentID, &komponentVorname, &komponentName,
			&c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {

			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if komponentID.Valid {
			komponent := Komponist{
				ID:      int(komponentID.Int64),
				vorname: komponentVorname.String,
				name:    komponentName.String,
			}
			c.Komponisten = append(c.Komponisten, komponent)
		}
		song = append(song, c)
	}

	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "rows iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to process Song",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": song,
	})
}

func OpenVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")
	dbQuery := `UPDATE Voting_Status SET isOpen = true, lastChange = ?`

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {

		changeTime := time.Now()

		result, err := DB.ExecContext(ctx, dbQuery, changeTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			Logger.ErrorContext(ctx, "Could not Query Rows", slog.Any("error", err))
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error querying rows",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to open Vote", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error opening the Votes",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "The Vote has been opened",
		})
	}

	if autorized != true {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
}

func CloseVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")
	dbQuery := `UPDATE Voting_Status SET isOpen = false, lastChange = ?`

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {

		changeTime := time.Now()

		result, err := DB.ExecContext(ctx, dbQuery, changeTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			Logger.ErrorContext(ctx, "Could not Query Rows", slog.Any("error", err))
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error querying rows",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to close Votes", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error Closing Votes",
			})
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": "The Vote has been closed",
		})
		return
	}

	if autorized != true {

		Logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func DeleteVotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	query := r.URL.Query()
	token := query.Get("Token")

	autorized, message := CheckAccessAdmin(token)

	if autorized != true {
		Logger.Warn("Invalid Login Attempt", slog.String("token", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": message})
		return
	}

	result, err := DB.ExecContext(ctx, `UPDATE Song SET PublikumsPunkte = 0, JuryPunkte = 0`)
	if err != nil {
		Logger.ErrorContext(ctx, "Error resetting song votes", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete Votes"})
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		Logger.ErrorContext(ctx, "failed to get rows affected after vote reset", slog.Any("error", err))
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No votes found to delete"})
		return
	}

	if _, phoneErr := DB.ExecContext(ctx,
		`UPDATE Phone_Nums SET votes_remaining = ?, votes_cast = JSON_OBJECT()`, totalVotePoints,
	); phoneErr != nil {
		Logger.ErrorContext(ctx, "failed to reset phone vote budgets", slog.Any("error", phoneErr))
	}

	mu.Lock()
	usedTokens = make(map[string]bool)
	mu.Unlock()

	Logger.InfoContext(ctx, "all votes deleted and vote budgets reset")

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"message": message,
		"payload": "All votes have been deleted and vote budgets reset",
	})
}

func AddCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	ID := query.Get("ID")
	Name := query.Get("Name")
	POTstr := query.Get("Pot")
	POT, err := strconv.Atoi(POTstr)

	if len(ID) != 2 {
		Logger.ErrorContext(ctx, "Invalid Input, CountryID must be 2 Characters as length", slog.String("Input", ID))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID must be 2 Charakters of length",
		})
		return
	}

	if err != nil {
		Logger.ErrorContext(ctx, "Invalid POT value", slog.Any("error", err), slog.String("POT", POTstr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Pot Value must be Integer",
		})
		return
	}
	dbQuery := `INSERT INTO Land (ID, Name, POT)
				VALUES (?, ?, ?)`

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {

		result, err := DB.ExecContext(ctx, dbQuery, ID, Name, POT)
		if err != nil {
			Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		Logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func AddSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid ID Value", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	country := query.Get("Land")
	youtubeURL := query.Get("YoutubeURL")

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID can only be two charakters in length", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID can only be two characters in length",
		})
	}

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {
		var result sql.Result
		if youtubeURL != "" {
			result, err = DB.ExecContext(ctx,
				`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES (?, ?, ?, 0, 0, ?)`,
				Name, country, ID, youtubeURL)
		} else {
			result, err = DB.ExecContext(ctx,
				`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte) VALUES (?, ?, ?, 0, 0)`,
				Name, country, ID)
		}

		if err != nil {
			Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		Logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}

}

func AddArtist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		Logger.ErrorContext(ctx, "Error Parsing ID", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	vorName := query.Get("vorName")
	typ := query.Get("typ")
	country := query.Get("Land")
	dbQuery := `INSERT INTO Kuenstler (ID, Vorname, Name, Typ, Land_ID) VALUES (?, ?, ?, ?, ?)`

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID can only be two charakters in length", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID can only be two characters in length",
		})
	}

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {

		result, err := DB.ExecContext(ctx, dbQuery, ID, Name, vorName, typ, country)

		if err != nil {
			Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		Logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func AddInterpret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	IDstr := query.Get("ID")
	ID, err := strconv.Atoi(IDstr)
	if err != nil {
		Logger.ErrorContext(ctx, "Error Parsing ID", slog.Any("error", err), slog.String("ID", IDstr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ID must be an Integer",
		})
		return
	}
	Name := query.Get("Name")
	vorName := query.Get("Vorname")
	dbQuery := `INSERT INTO Komponist (ID, Vorname, Name) VALUES (?,?,?)`

	autorized, message := CheckAccessAdmin(token)

	if autorized == true {

		result, err := DB.ExecContext(ctx, dbQuery, ID, Name, vorName)

		if err != nil {
			Logger.ErrorContext(ctx, "Failed to Insert Data", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "error inserting into DB",
			})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			Logger.ErrorContext(ctx, "failed to verify insertions", slog.Any("error", err))
		}
		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Error verifying Query",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message": message,
			"payload": result,
		})
	}

	if autorized != true {

		Logger.Warn("Invalid Login Attempt")
		slog.String("token", token)

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
	}
}

func JuryVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")
	points := query.Get("points")
	songIDStr := query.Get("songID")

	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		Logger.ErrorContext(ctx, "invalid songID value", slog.Any("error", err), slog.String("songID", songIDStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid songID value - must be an Integer",
		})
		return
	}

	parsedPoints, err := strconv.Atoi(points)
	if err != nil {
		Logger.ErrorContext(ctx, "invalid points value", slog.Any("error", err), slog.String("points", points))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid points value - must be an Integer",
		})
		return
	}

	validJuryPoints := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 10: true, 12: true}
	if !validJuryPoints[parsedPoints] {
		Logger.WarnContext(ctx, "invalid jury points value", slog.Int("points", parsedPoints))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid points value - must be one of: 1, 2, 3, 4, 5, 6, 7, 8, 10, 12",
		})
		return
	}

	var (
		authorized bool
		authMsg    string
		isOpen     bool
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		authorized, authMsg = CheckAccessJury(token)
		return nil
	})

	g.Go(func() error {
		err := DB.QueryRowContext(gctx, `SELECT isOpen FROM Voting_Status`).Scan(&isOpen)
		if err != nil {
			Logger.ErrorContext(gctx, "failed to query voting status", slog.Any("error", err))
		}
		return err
	})

	if err := g.Wait(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Error querying DB",
		})
		return
	}

	if !authorized {
		Logger.WarnContext(ctx, "invalid jury login attempt", slog.String("token", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": authMsg,
		})
		return
	}

	if !isOpen {
		w.WriteHeader(http.StatusTooEarly)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "voting has not started yet, please try again in a few minutes",
		})
		return
	}

	const juryWeight int = 1
	totalPoints := juryWeight * parsedPoints

	dbQuery := `UPDATE Song
			 SET JuryPunkte = JuryPunkte + ?
			 WHERE ID = ?`

	result, err := DB.ExecContext(ctx, dbQuery, totalPoints, songID)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to update vote", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to record vote",
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		Logger.ErrorContext(ctx, "failed to get affected rows", slog.Any("error", err))
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Song not found",
		})
		return
	}

	NotifyVote(songID, "JURY", DB)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"message": authMsg,
		"payload": "Vote Successfully Cast",
	})
}

func AdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")

	authenticated, message := CheckAccessAdmin(token)

	if authenticated == true {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
	if authenticated == false {
		Logger.WarnContext(ctx, "Invalid Login Atempt", slog.Any("token:", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
		return
	}
}

func JuryLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	query := r.URL.Query()
	token := query.Get("Token")

	authenticated, message := CheckAccessJury(token)

	if authenticated == true {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": message,
		})
		return
	}
	if authenticated == false {
		Logger.WarnContext(ctx, "Invalid Login Atempt", slog.Any("token:", token))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": message,
		})
		return
	}
}

func StartContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	authenticated, msg := CheckAccessAdmin(token)
	if !authenticated {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	rows, err := DB.QueryContext(ctx, "SELECT ID FROM Song ORDER BY ID")
	if err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to query songs", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query songs"})
		return
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "startContest: rows error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read songs"})
		return
	}

	if len(ids) == 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "No songs in database"})
		return
	}

	// Fisher-Yates shuffle
	for i := len(ids) - 1; i > 0; i-- {
		b := make([]byte, 4)
		rand.Read(b)
		j := int(b[0]) % (i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}

	orderJSON, err := json.Marshal(ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode song order"})
		return
	}

	if _, err := DB.ExecContext(ctx, "UPDATE Contest_Run SET IsActive = FALSE WHERE IsActive = TRUE"); err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to deactivate old runs", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to deactivate previous contest"})
		return
	}

	if _, err := DB.ExecContext(ctx,
		"INSERT INTO Contest_Run (SongOrder, CurrentIndex, IsActive) VALUES (?, 0, TRUE)",
		string(orderJSON),
	); err != nil {
		Logger.ErrorContext(ctx, "startContest: failed to insert contest run", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start contest"})
		return
	}

	Logger.InfoContext(ctx, "contest started", slog.Int("songCount", len(ids)))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "Contest started",
		"songCount": len(ids),
		"order":     ids,
	})
}

func AdvanceContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	token := query.Get("Token")
	authenticated, msg := CheckAccessAdmin(token)
	if !authenticated {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	var (
		runID        int
		orderJSON    string
		currentIndex int
	)
	err := DB.QueryRowContext(ctx,
		"SELECT ID, SongOrder, CurrentIndex FROM Contest_Run WHERE IsActive = TRUE ORDER BY ID DESC LIMIT 1",
	).Scan(&runID, &orderJSON, &currentIndex)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No active contest"})
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query contest"})
		return
	}

	var ids []int
	if err := json.Unmarshal([]byte(orderJSON), &ids); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse song order"})
		return
	}

	nextIndex := currentIndex + 1
	if nextIndex >= len(ids) {
		DB.ExecContext(ctx, "UPDATE Contest_Run SET IsActive = FALSE WHERE ID = ?", runID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"message":  "Contest finished",
			"finished": true,
		})
		return
	}

	if _, err := DB.ExecContext(ctx,
		"UPDATE Contest_Run SET CurrentIndex = ? WHERE ID = ?", nextIndex, runID,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to advance contest"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":      "Advanced to next song",
		"currentIndex": nextIndex,
		"songId":       ids[nextIndex],
		"finished":     false,
	})
}

func GetCurrentSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var (
		runID        int
		orderJSON    string
		currentIndex int
	)
	err := DB.QueryRowContext(ctx,
		"SELECT ID, SongOrder, CurrentIndex FROM Contest_Run WHERE IsActive = TRUE ORDER BY ID DESC LIMIT 1",
	).Scan(&runID, &orderJSON, &currentIndex)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No active contest"})
		return
	}
	if err != nil {
		Logger.ErrorContext(ctx, "getCurrentSong: db error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query contest"})
		return
	}

	var ids []int
	if err := json.Unmarshal([]byte(orderJSON), &ids); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse song order"})
		return
	}

	if currentIndex >= len(ids) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "Contest has ended"})
		return
	}

	currentSongID := ids[currentIndex]

	songQuery := `SELECT
		s.ID, s.Name, s.PublikumsPunkte, s.JuryPunkte, s.GesamtPunkte,
		COALESCE(s.YoutubeURL, ''),
		l.ID, l.Name,
		k.ID, k.Vorname, k.Name, k.Typ,
		vs.isOpen
		FROM Song s
		INNER JOIN Land l ON s.Land_ID = l.ID
		INNER JOIN Kuenstler k ON s.Kuenstler_ID = k.ID
		LEFT JOIN Voting_Status vs ON vs.VotingID = 1
		WHERE s.ID = ?`

	var (
		songID, artistID                            int
		songName, countryID, countryName            string
		artistFirstName, artistLastName, artistType string
		publicVotes, juryVotes, totalVotes          int
		youtubeURL                                  string
		votingIsOpen                                bool
	)
	err = DB.QueryRowContext(ctx, songQuery, currentSongID).Scan(
		&songID, &songName, &publicVotes, &juryVotes, &totalVotes,
		&youtubeURL,
		&countryID, &countryName,
		&artistID, &artistFirstName, &artistLastName, &artistType,
		&votingIsOpen,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}
	if err != nil {
		Logger.ErrorContext(ctx, "getCurrentSong: song query error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch song"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": map[string]any{
			"runId":           runID,
			"currentIndex":    currentIndex,
			"totalSongs":      len(ids),
			"songId":          songID,
			"songName":        songName,
			"youtubeUrl":      youtubeURL,
			"countryId":       countryID,
			"countryName":     countryName,
			"artistId":        artistID,
			"artistFirstName": artistFirstName,
			"artistLastName":  artistLastName,
			"artistType":      artistType,
			"publicVotes":     publicVotes,
			"juryVotes":       juryVotes,
			"totalVotes":      totalVotes,
			"votingIsOpen":    votingIsOpen,
		},
	})
}

func Run() {
	InitCookieSecret()

	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	Logger = slog.New(baseHandler)
	slog.SetDefault(Logger)

	var (
		tp *sdktrace.TracerProvider
		lp *sdklog.LoggerProvider
	)

	{
		g, _ := errgroup.WithContext(context.Background())

		g.Go(func() error {
			var err error
			tp, err = initTracer()
			if err != nil {
				log.Printf("Warning: Failed to initialize tracer: %v. Continuing without tracing", err)
			}
			return nil
		})

		g.Go(func() error {
			var err error
			lp, err = initLogProvider()
			if err != nil {
				log.Printf("Warning: Failed to initalize log provider: %v. Logs will not be forwarded via OTLP", err)
			}
			return nil
		})

		g.Wait()
	}

	if tp != nil {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	if lp != nil {
		defer func() {
			if err := lp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down log provider: %v", err)
			}
		}()
	}

	Logger = slog.New(newOtelSlogHandler(baseHandler))
	slog.SetDefault(Logger)

	Tracer = otel.Tracer("esc-voting-crud-api")

	router := http.NewServeMux()
	router.HandleFunc("GET /health", GetHealth)
	router.HandleFunc("GET /votes/", GetVotes)
	router.HandleFunc("POST /vote/", Vote)
	router.HandleFunc("GET /countries/", GetCountries)
	router.HandleFunc("GET /countryByName/{NAME}", GetCountryByName)
	router.HandleFunc("GET /songs/", HttpGetSongs)
	router.HandleFunc("GET /songByID/{ID}", GetSongByID)
	router.HandleFunc("POST /admin/open", OpenVote)
	router.HandleFunc("POST /admin/close", CloseVote)
	router.HandleFunc("DELETE /admin/deleteVotes/", DeleteVotes)
	router.HandleFunc("POST /admin/addCountry/", AddCountry)
	router.HandleFunc("POST /admin/addSong/", AddSong)
	router.HandleFunc("POST /admin/addArtist/", AddArtist)
	router.HandleFunc("POST /admin/addInterpret/", AddInterpret)
	router.HandleFunc("POST /jury/vote/", JuryVote)
	router.HandleFunc("GET /admin/authenticate", AdminLogin)
	router.HandleFunc("GET /jury/authenticate", JuryLogin)
	router.HandleFunc("POST /admin/startContest", StartContest)
	router.HandleFunc("POST /admin/advanceContest", AdvanceContest)
	router.HandleFunc("GET /contest/current", GetCurrentSong)

	router.Handle("GET /metrics/", promhttp.Handler())

	dbReadinessMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			select {
			case <-dbReady:
				next.ServeHTTP(w, r)
			case <-r.Context().Done():
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "service is starting up, please retry",
				})
			}
		})
	}

	handler := dbReadinessMiddleware(RateLimitingMiddleware(ObservabilityMiddleware(router)))

	srv := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Logger.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	go func() {
		conn, dbErr := connectToDatabase(loadLocalConfig())

		if dbErr != nil {
			Logger.Error("could not connect to Database after all retries",
				slog.Any("error", dbErr),
			)
			os.Exit(1)
		}

		DB = conn
		close(dbReady)

		Logger.Info("database connection established - service fully ready")

		voteService, err := StartGRPCServer(DB, "50051")
		if err != nil {
			Logger.Error("Failed to start gRPC server", slog.String("error", err.Error()))
			os.Exit(1)
		}
		SetGlobalVoteServer(voteService)
		Logger.Info("gRPC vote streaming service initialized")
	}()

	log.Println("Listening and Serving on Port 8000")
	Logger.Info("server starting",
		slog.Int("port", 8000),
		slog.String("service", "esc-voting-crud-api"),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	Logger.Info("shutdown singal recieved, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown %v", err)
	}

	Logger.Info("closing database")
	if DB != nil {
		DB.Close()
	}

	log.Println("server exited cleanly")

}
