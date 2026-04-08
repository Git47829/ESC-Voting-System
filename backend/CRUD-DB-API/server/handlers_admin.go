package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func OpenVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	dbQuery := `UPDATE Voting_Status SET isOpen = true, lastChange = ?`


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
			"payload": "The Vote has been opened",
		})
}

func CloseVote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	dbQuery := `UPDATE Voting_Status SET isOpen = false, lastChange = ?`

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
			"payload": "The Vote has been closed",
		})
}

func DeleteVotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()

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
		"payload": "All votes have been deleted and vote budgets reset",
	})
}
