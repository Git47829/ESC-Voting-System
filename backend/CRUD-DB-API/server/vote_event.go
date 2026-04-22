package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// VoteEvent is the JSON payload published to RabbitMQ when a vote is cast.
type VoteEvent struct {
	CountryVotedFor     string `json:"country_voted_for"`
	CountryVotedForName string `json:"country_voted_for_name"`
	VoteCount           int    `json:"vote_count"`
	VoterCountry        string `json:"voter_country"`
	VoterCountryName    string `json:"voter_country_name"`
	Timestamp           int64  `json:"timestamp"`
	SongID              int    `json:"song_id"`
	SongName            string `json:"song_name"`
}

// NotifyVote publishes a vote event to the RabbitMQ fanout exchange.
func NotifyVote(songID int, voterCountry string, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		id, totalVotes                   int
		songName, countryID, countryName string
		voterCountryName                 string
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		query := `
			SELECT
				s.ID,
				s.Name,
				l.ID as country_id,
				l.Name as country_name,
				s.PublikumsPunkte + s.JuryPunkte as total_votes
			FROM Song s
			JOIN Land l ON s.Land_ID = l.ID
			WHERE s.ID = ?
		`

		err := db.QueryRowContext(gctx, query, songID).
			Scan(&id, &songName, &countryID, &countryName, &totalVotes)
		if err != nil {
			Logger.ErrorContext(gctx, "failed to get vote details for notification",
				slog.Any("error", err),
				slog.Int("song_id", songID),
			)
		}
		return err
	})

	g.Go(func() error {
		if voterCountry == "JURY" {
			voterCountryName = "Jury"
			return nil
		}
		if voterCountry == "" || voterCountry == "SYSTEM" {
			voterCountryName = "Unknown"
			return nil
		}
		err := db.QueryRowContext(ctx, "SELECT Name FROM Land WHERE ID = ?", voterCountry).Scan(&voterCountryName)
		if err != nil {
			voterCountryName = voterCountry
			Logger.Debug("could not get voter country name",
				slog.String("country_id", voterCountry),
				slog.Any("error", err),
			)
			return nil
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		Logger.ErrorContext(ctx, "NotifyVote: aborting broadcast due to DB error",
			slog.Any("error", err),
			slog.Int("song_id", songID),
		)
		return
	}

	vote := VoteEvent{
		CountryVotedFor:     countryID,
		CountryVotedForName: countryName,
		VoteCount:           totalVotes,
		VoterCountry:        voterCountry,
		VoterCountryName:    voterCountryName,
		Timestamp:           time.Now().Unix(),
		SongID:              songID,
		SongName:            songName,
	}

	if err := PublishVote(vote); err != nil {
		Logger.Error("failed to publish vote to RabbitMQ",
			slog.Any("error", err),
			slog.Int("song_id", songID),
		)
		return
	}

	Logger.Info("vote published to RabbitMQ",
		slog.Int("song_id", songID),
		slog.String("country_voted_for", countryID),
		slog.Int("total_votes", totalVotes),
		slog.String("voter_country", voterCountry),
	)
}

// GetSongsWithPublicVotes returns all songs with their public vote counts.
func GetSongsWithPublicVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rows, err := DB.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.ID, l.Name, s.PublikumsPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
	`)
	if err != nil {
		Logger.ErrorContext(ctx, "GetSongsWithPublicVotes: query failed", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve songs"})
		return
	}
	defer rows.Close()

	type SongVoteData struct {
		SongID      int    `json:"songId"`
		SongName    string `json:"songName"`
		CountryID   string `json:"countryId"`
		CountryName string `json:"countryName"`
		PublicVotes int    `json:"publicVotes"`
	}

	var songs []SongVoteData
	for rows.Next() {
		var s SongVoteData
		if err := rows.Scan(&s.SongID, &s.SongName, &s.CountryID, &s.CountryName, &s.PublicVotes); err != nil {
			Logger.ErrorContext(ctx, "GetSongsWithPublicVotes: scan failed", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process songs"})
			return
		}
		songs = append(songs, s)
	}
	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "GetSongsWithPublicVotes: rows error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process songs"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Success",
		"payload": songs,
	})
}
