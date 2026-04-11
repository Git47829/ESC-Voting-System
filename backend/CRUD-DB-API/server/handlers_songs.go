package server

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

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

	var countries []Countrys
	for rows.Next() {
		var c Countrys
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
		return
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

	var countries []Countrys
	for rows.Next() {
		var c Countrys
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

func HTTPGetSongs(w http.ResponseWriter, r *http.Request) {
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

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var componentID sql.NullInt64
		var componentFirstName sql.NullString
		var componentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublicPoints, &c.JuryPoints, &c.TotalPoints, &c.CountryID,
			&c.CountryName, &c.CountryPOT, &c.ArtistID, &c.ArtistFirstName, &c.ArtistName, &c.ArtistType, &componentID,
			&componentFirstName, &componentName, &c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {

			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if componentID.Valid {
			component := Composer{
				ID:      int(componentID.Int64),
				firstName: componentFirstName.String,
				name:    componentName.String,
			}
			c.Composer = append(c.Composer, component)
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
		return
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

	var song []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var componentID sql.NullInt64
		var componentFirstName sql.NullString
		var componentName sql.NullString
		if err := rows.Scan(&c.SongID, &c.SongName, &c.PublicPoints, &c.JuryPoints, &c.TotalPoints, &c.CountryID, &c.CountryName, &c.CountryPOT,
			&c.ArtistID, &c.ArtistFirstName, &c.ArtistName, &c.ArtistType, &componentID, &componentFirstName, &componentName,
			&c.VotingID, &c.VotingIsOpen, &c.VotingLastChange); err != nil {

			Logger.ErrorContext(ctx, "failed to scan row", slog.Any("error", err))
			continue
		}
		if componentID.Valid {
			component := Composer{
				ID:      int(componentID.Int64),
				firstName: componentFirstName.String,
				name:    componentName.String,
			}
			c.Composer = append(c.Composer, component)
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
