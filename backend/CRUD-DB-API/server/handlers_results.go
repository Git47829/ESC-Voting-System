package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
)

type resultEntry struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	CountryID    string `json:"countryId"`
	ESCPublicPts int    `json:"escPublicPts"`
	JuryPts      int    `json:"juryPts"`
	TotalPts     int    `json:"totalPts"`
	Rank         int    `json:"rank"`
}

func escPointsForRank(rank int) int {
	if rank >= 0 && rank < len(escPointTable) {
		return escPointTable[rank]
	}
	return 0
}

func GetResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rows, err := DB.QueryContext(ctx, `
SELECT
s.ID,
s.Name,
l.Name,
l.ID,
s.PublikumsPunkte,
s.JuryPunkte
FROM Song s
JOIN Land l ON s.Land_ID = l.ID
`)
	if err != nil {
		Logger.ErrorContext(ctx, "failed to query results", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to query results"})
		return
	}
	defer rows.Close()

	entries := make([]resultEntry, 0)
	publicVoteMap := make(map[int]int)
	for rows.Next() {
		var entry resultEntry
		var rawPublicVotes int
		if err := rows.Scan(&entry.ID, &entry.Name, &entry.Country, &entry.CountryID, &rawPublicVotes, &entry.JuryPts); err != nil {
			Logger.ErrorContext(ctx, "failed to scan result row", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process results"})
			return
		}
		publicVoteMap[entry.ID] = rawPublicVotes
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		Logger.ErrorContext(ctx, "result row iteration error", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process results"})
		return
	}

	juryScale := getAuthConfig().juryScale

	sort.SliceStable(entries, func(i, j int) bool {
		leftVotes := publicVoteMap[entries[i].ID]
		rightVotes := publicVoteMap[entries[j].ID]
		if leftVotes != rightVotes {
			return leftVotes > rightVotes
		}
		return entries[i].ID < entries[j].ID
	})
	for i := range entries {
		if publicVoteMap[entries[i].ID] > 0 {
			entries[i].ESCPublicPts = escPointsForRank(i) * juryScale
		}
		entries[i].TotalPts = entries[i].ESCPublicPts + entries[i].JuryPts
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].TotalPts != entries[j].TotalPts {
			return entries[i].TotalPts > entries[j].TotalPts
		}
		return entries[i].ID < entries[j].ID
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entries)
}
