package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

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
		return
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

	phoneHash := HashPhoneNumber(phone)

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

	tx, txErr := DB.BeginTx(ctx, nil)
	if txErr != nil {
		Logger.ErrorContext(ctx, "failed to begin transaction", slog.Any("error", txErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}
	defer tx.Rollback()

	upsertQuery := `
		INSERT INTO Phone_Nums (Phone_Number, votes_remaining, votes_cast)
		VALUES (?, ?, JSON_OBJECT())
		ON DUPLICATE KEY UPDATE Phone_Number = Phone_Number`

	if _, upsertErr := tx.ExecContext(ctx, upsertQuery, phoneHash, totalVotePoints); upsertErr != nil {
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

	deductResult, deductErr := tx.ExecContext(ctx, deductQuery,
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
		tx.QueryRowContext(ctx,
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

	const weightPublicVote = 1
	songResult, songErr := tx.ExecContext(ctx,
		`UPDATE Song SET PublikumsPunkte = PublikumsPunkte + ? WHERE ID = ?`,
		points*weightPublicVote, songID,
	)
	if songErr != nil {
		Logger.ErrorContext(ctx, "failed to update song votes", slog.Any("error", songErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	songUpdatedRows, _ := songResult.RowsAffected()
	if songUpdatedRows == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Song not found"})
		return
	}

	if commitErr := tx.Commit(); commitErr != nil {
		Logger.ErrorContext(ctx, "failed to commit vote transaction", slog.Any("error", commitErr))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record vote"})
		return
	}

	var (
		remaining int
		votesCast = make(map[string]int)
		cookieVal string
	)

	var votesCastJSON sql.NullString
	err = DB.QueryRowContext(ctx, `SELECT votes_remaining, votes_cast FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash).Scan(&remaining, &votesCastJSON)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to read back vote state", slog.Any("error", err))
	} else {
		if votesCastJSON.Valid && votesCastJSON.String != "" {
			if err := json.Unmarshal([]byte(votesCastJSON.String), &votesCast); err != nil {
				Logger.ErrorContext(ctx, "failed to unmarshal votes_cast", slog.Any("error", err))
			}
		}

		state := CookieVoteState{
			VotesRemaining: remaining,
			VotesCast:      votesCast,
		}

		var encErr error
		cookieVal, encErr = EncodeCookieValue(state)
		if encErr != nil {
			Logger.ErrorContext(ctx, "failed to encode vote cookie", slog.Any("error", encErr))
		}
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
