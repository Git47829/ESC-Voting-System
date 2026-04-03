package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"
)

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
