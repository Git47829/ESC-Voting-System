package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

func StartContest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")


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
