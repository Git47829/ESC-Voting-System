package server

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

func AddCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	ID := query.Get("ID")
	Name := query.Get("Name")
	POTstr := query.Get("Pot")
	POT, err := strconv.Atoi(POTstr)

	if len(ID) != 2 {
		Logger.ErrorContext(ctx, "Invalid Input, CountryID must be 2 Characters as length", slog.String("Input", ID))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID must be 2 Characters of length",
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
		"payload": result,
	})
}

func AddSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	Name := query.Get("SongName")
	if Name == "" {
		Name = query.Get("Name")
	}

	country := query.Get("CountryID")
	if country == "" {
		country = query.Get("Land")
	}

	artistIDStr := query.Get("KuenstlerID")
	if artistIDStr == "" {
		artistIDStr = query.Get("ID")
	}
	artistID, err := strconv.Atoi(artistIDStr)
	if err != nil {
		Logger.ErrorContext(ctx, "Invalid KuenstlerID Value", slog.Any("error", err), slog.String("KuenstlerID", artistIDStr))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "KuenstlerID must be an Integer",
		})
		return
	}

	youtubeURL := query.Get("YoutubeURL")

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID can only be two charakters in length", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID can only be two characters in length",
		})
		return
	}

	var result sql.Result
	if youtubeURL != "" {
		result, err = DB.ExecContext(ctx,
			`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES (?, ?, ?, 0, 0, ?)`,
			Name, country, artistID, youtubeURL)
	} else {
		result, err = DB.ExecContext(ctx,
			`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte) VALUES (?, ?, ?, 0, 0)`,
			Name, country, artistID)
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
		"payload": result,
	})
}

func AddArtist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	Name := query.Get("Name")
	if Name == "" {
		Name = query.Get("LastName")
	}
	vorName := query.Get("vorName")
	if vorName == "" {
		vorName = query.Get("FirstName")
	}
	typ := query.Get("typ")
	if typ == "" {
		typ = query.Get("Type")
	}
	if typ == "" {
		typ = "solo"
	}
	country := query.Get("CountryID")
	if country == "" {
		country = query.Get("Land")
	}
	dbQuery := `INSERT INTO Kuenstler (Vorname, Name, Typ, Land_ID) VALUES (?, ?, ?, ?)`

	if len(country) != 2 {
		Logger.ErrorContext(ctx, "CountryID can only be two charakters in length", slog.String("CountryID: ", country))
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "CountryID can only be two characters in length",
		})
		return
	}

	result, err := DB.ExecContext(ctx, dbQuery, vorName, Name, typ, country)
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
		"payload": result,
	})
}

func AddInterpret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
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
		"payload": result,
	})
}
