package server

import (
	"context"
	"database/sql"
	"time"
)

type VoteRepository interface {
	GetVotingStatus(ctx context.Context) (isOpen bool, err error)
	GetSongCountry(ctx context.Context, songID int) (countryID string, err error)
	BeginTx(ctx context.Context) (*sql.Tx, error)
	UpsertPhoneVoter(ctx context.Context, tx *sql.Tx, phoneHash string, initialBudget int) error
	DeductVotePoints(ctx context.Context, tx *sql.Tx, phoneHash, songIDStr string, points int) (rowsAffected int64, err error)
	GetPhoneVotesRemainingTx(ctx context.Context, tx *sql.Tx, phoneHash string) (int, error)
	UpdateSongPublicVotes(ctx context.Context, tx *sql.Tx, songID, points int) (rowsAffected int64, err error)
	GetPhoneVoteState(ctx context.Context, phoneHash string) (remaining int, castJSON string, err error)
	GetAllVotes(ctx context.Context) ([]SongVote, error)
	SetVotingOpen(ctx context.Context, open bool) (rowsAffected int64, err error)
	ResetAllVotes(ctx context.Context) (rowsAffected int64, err error)
	ResetPhoneBudgets(ctx context.Context, budget int) error
	UpdateSongJuryVotes(ctx context.Context, songID, points int) (rowsAffected int64, err error)
}

type SongRepository interface {
	GetAllCountries(ctx context.Context) ([]Countrys, error)
	GetCountriesByID(ctx context.Context, id string) ([]Countrys, error)
	GetAllSongs(ctx context.Context) ([]CompleteESCEntryWithComposers, error)
	GetSongsByID(ctx context.Context, id int) ([]CompleteESCEntryWithComposers, error)
	GetSongIDs(ctx context.Context) ([]int, error)
	DeactivateContestRuns(ctx context.Context) error
	InsertContestRun(ctx context.Context, orderJSON string) error
	GetCurrentContestRun(ctx context.Context) (runID int, orderJSON string, currentIndex int, err error)
	AdvanceContestRun(ctx context.Context, runID, nextIndex int) error
	FinishContestRun(ctx context.Context, runID int) error
	GetContestSong(ctx context.Context, songID int) (ContestSong, error)
	InsertCountry(ctx context.Context, id, name string, pot int) (rowsAffected int64, err error)
	InsertSong(ctx context.Context, name, countryID string, artistID int, youtubeURL string) (rowsAffected int64, err error)
	InsertArtist(ctx context.Context, firstName, name, typ, countryID string) (rowsAffected int64, err error)
	InsertInterpret(ctx context.Context, id int, name, vorname string) (rowsAffected int64, err error)
}


type mysqlVoteRepository struct{ db *sql.DB }

func newMySQLVoteRepository(db *sql.DB) VoteRepository {
	return &mysqlVoteRepository{db: db}
}

func (r *mysqlVoteRepository) GetVotingStatus(ctx context.Context) (bool, error) {
	var isOpen bool
	err := r.db.QueryRowContext(ctx, `SELECT isOpen FROM Voting_Status`).Scan(&isOpen)
	return isOpen, err
}

func (r *mysqlVoteRepository) GetSongCountry(ctx context.Context, songID int) (string, error) {
	var landID string
	err := r.db.QueryRowContext(ctx, `SELECT Land_ID FROM Song WHERE ID = ?`, songID).Scan(&landID)
	return landID, err
}

func (r *mysqlVoteRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *mysqlVoteRepository) UpsertPhoneVoter(ctx context.Context, tx *sql.Tx, phoneHash string, initialBudget int) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO Phone_Nums (Phone_Number, votes_remaining, votes_cast)
		 VALUES (?, ?, JSON_OBJECT())
		 ON DUPLICATE KEY UPDATE Phone_Number = Phone_Number`,
		phoneHash, initialBudget,
	)
	return err
}

func (r *mysqlVoteRepository) DeductVotePoints(ctx context.Context, tx *sql.Tx, phoneHash, songIDStr string, points int) (int64, error) {
	result, err := tx.ExecContext(ctx,
		`UPDATE Phone_Nums
		 SET
		   votes_remaining = votes_remaining - ?,
		   votes_cast = JSON_SET(
		     COALESCE(votes_cast, JSON_OBJECT()),
		     CONCAT('$."', ?, '"'),
		     COALESCE(JSON_EXTRACT(votes_cast, CONCAT('$."', ?, '"')), 0) + ?
		   )
		 WHERE Phone_Number = ? AND votes_remaining >= ?`,
		points, songIDStr, songIDStr, points, phoneHash, points,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlVoteRepository) GetPhoneVotesRemainingTx(ctx context.Context, tx *sql.Tx, phoneHash string) (int, error) {
	var remaining int
	err := tx.QueryRowContext(ctx,
		`SELECT votes_remaining FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash,
	).Scan(&remaining)
	return remaining, err
}

func (r *mysqlVoteRepository) UpdateSongPublicVotes(ctx context.Context, tx *sql.Tx, songID, points int) (int64, error) {
	result, err := tx.ExecContext(ctx,
		`UPDATE Song SET PublikumsPunkte = PublikumsPunkte + ? WHERE ID = ?`,
		points, songID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlVoteRepository) GetPhoneVoteState(ctx context.Context, phoneHash string) (int, string, error) {
	var remaining int
	var castJSON sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT votes_remaining, votes_cast FROM Phone_Nums WHERE Phone_Number = ?`, phoneHash,
	).Scan(&remaining, &castJSON)
	if err != nil {
		return 0, "", err
	}
	if castJSON.Valid {
		return remaining, castJSON.String, nil
	}
	return remaining, "", nil
}

func (r *mysqlVoteRepository) GetAllVotes(ctx context.Context) ([]SongVote, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.Name as Country,
		       s.PublikumsPunkte, s.JuryPunkte, s.GesamtPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
		ORDER BY s.GesamtPunkte DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []SongVote
	for rows.Next() {
		var v SongVote
		if err := rows.Scan(&v.ID, &v.Name, &v.Country, &v.PublikumsPunkte, &v.JuryPunkte, &v.GesamtPunkte); err != nil {
			continue
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

func (r *mysqlVoteRepository) SetVotingOpen(ctx context.Context, open bool) (int64, error) {
	var query string
	if open {
		query = `UPDATE Voting_Status SET isOpen = true, lastChange = ?`
	} else {
		query = `UPDATE Voting_Status SET isOpen = false, lastChange = ?`
	}
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlVoteRepository) ResetAllVotes(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE Song SET PublikumsPunkte = 0, JuryPunkte = 0`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlVoteRepository) ResetPhoneBudgets(ctx context.Context, budget int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE Phone_Nums SET votes_remaining = ?, votes_cast = JSON_OBJECT()`, budget,
	)
	return err
}

func (r *mysqlVoteRepository) UpdateSongJuryVotes(ctx context.Context, songID, points int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE Song SET JuryPunkte = JuryPunkte + ? WHERE ID = ?`, points, songID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type mysqlSongRepository struct{ db *sql.DB }

func newMySQLSongRepository(db *sql.DB) SongRepository {
	return &mysqlSongRepository{db: db}
}

func (r *mysqlSongRepository) GetAllCountries(ctx context.Context) ([]Countrys, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ID, Name, POT FROM Land ORDER BY Name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCountries(rows)
}

func (r *mysqlSongRepository) GetCountriesByID(ctx context.Context, id string) ([]Countrys, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ID, Name, POT FROM Land WHERE ID = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCountries(rows)
}

func scanCountries(rows *sql.Rows) ([]Countrys, error) {
	var countries []Countrys
	for rows.Next() {
		var c Countrys
		var potValue sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &potValue); err != nil {
			continue
		}
		if potValue.Valid {
			pot := int(potValue.Int64)
			c.Pot = &pot
		}
		countries = append(countries, c)
	}
	return countries, rows.Err()
}

const songQuery = `SELECT
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
    ORDER BY s.ID, komp.ID`

func (r *mysqlSongRepository) GetAllSongs(ctx context.Context) ([]CompleteESCEntryWithComposers, error) {
	rows, err := r.db.QueryContext(ctx, songQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

func (r *mysqlSongRepository) GetSongsByID(ctx context.Context, id int) ([]CompleteESCEntryWithComposers, error) {
	q := songQuery[:len(songQuery)-len("    ORDER BY s.ID, komp.ID")] +
		"    WHERE s.ID = ?\n    ORDER BY s.ID, komp.ID"
	rows, err := r.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

func scanSongs(rows *sql.Rows) ([]CompleteESCEntryWithComposers, error) {
	var songs []CompleteESCEntryWithComposers
	for rows.Next() {
		var c CompleteESCEntryWithComposers
		var componentID sql.NullInt64
		var componentFirstName, componentName sql.NullString
		if err := rows.Scan(
			&c.SongID, &c.SongName, &c.PublicPoints, &c.JuryPoints, &c.TotalPoints,
			&c.CountryID, &c.CountryName, &c.CountryPOT,
			&c.ArtistID, &c.ArtistFirstName, &c.ArtistName, &c.ArtistType,
			&componentID, &componentFirstName, &componentName,
			&c.VotingID, &c.VotingIsOpen, &c.VotingLastChange,
		); err != nil {
			continue
		}
		if componentID.Valid {
			c.Composer = append(c.Composer, Composer{
				ID:        int(componentID.Int64),
				firstName: componentFirstName.String,
				name:      componentName.String,
			})
		}
		songs = append(songs, c)
	}
	return songs, rows.Err()
}

func (r *mysqlSongRepository) GetSongIDs(ctx context.Context) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ID FROM Song ORDER BY ID`)
	if err != nil {
		return nil, err
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
	return ids, rows.Err()
}

func (r *mysqlSongRepository) DeactivateContestRuns(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `UPDATE Contest_Run SET IsActive = FALSE WHERE IsActive = TRUE`)
	return err
}

func (r *mysqlSongRepository) InsertContestRun(ctx context.Context, orderJSON string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO Contest_Run (SongOrder, CurrentIndex, IsActive) VALUES (?, 0, TRUE)`,
		orderJSON,
	)
	return err
}

func (r *mysqlSongRepository) GetCurrentContestRun(ctx context.Context) (int, string, int, error) {
	var runID, currentIndex int
	var orderJSON string
	err := r.db.QueryRowContext(ctx,
		`SELECT ID, SongOrder, CurrentIndex FROM Contest_Run WHERE IsActive = TRUE ORDER BY ID DESC LIMIT 1`,
	).Scan(&runID, &orderJSON, &currentIndex)
	return runID, orderJSON, currentIndex, err
}

func (r *mysqlSongRepository) AdvanceContestRun(ctx context.Context, runID, nextIndex int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE Contest_Run SET CurrentIndex = ? WHERE ID = ?`, nextIndex, runID,
	)
	return err
}

func (r *mysqlSongRepository) FinishContestRun(ctx context.Context, runID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE Contest_Run SET IsActive = FALSE WHERE ID = ?`, runID,
	)
	return err
}

func (r *mysqlSongRepository) GetContestSong(ctx context.Context, songID int) (ContestSong, error) {
	q := `SELECT
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

	var cs ContestSong
	err := r.db.QueryRowContext(ctx, q, songID).Scan(
		&cs.SongID, &cs.SongName, &cs.PublicVotes, &cs.JuryVotes, &cs.TotalVotes,
		&cs.YoutubeURL,
		&cs.CountryID, &cs.CountryName,
		&cs.ArtistID, &cs.ArtistFirstName, &cs.ArtistLastName, &cs.ArtistType,
		&cs.VotingIsOpen,
	)
	return cs, err
}

func (r *mysqlSongRepository) InsertCountry(ctx context.Context, id, name string, pot int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO Land (ID, Name, POT) VALUES (?, ?, ?)`, id, name, pot,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlSongRepository) InsertSong(ctx context.Context, name, countryID string, artistID int, youtubeURL string) (int64, error) {
	var result sql.Result
	var err error
	if youtubeURL != "" {
		result, err = r.db.ExecContext(ctx,
			`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES (?, ?, ?, 0, 0, ?)`,
			name, countryID, artistID, youtubeURL,
		)
	} else {
		result, err = r.db.ExecContext(ctx,
			`INSERT INTO Song (Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte) VALUES (?, ?, ?, 0, 0)`,
			name, countryID, artistID,
		)
	}
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlSongRepository) InsertArtist(ctx context.Context, firstName, name, typ, countryID string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO Kuenstler (Vorname, Name, Typ, Land_ID) VALUES (?, ?, ?, ?)`,
		firstName, name, typ, countryID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *mysqlSongRepository) InsertInterpret(ctx context.Context, id int, name, vorname string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO Komponist (ID, Vorname, Name) VALUES (?, ?, ?)`, id, vorname, name,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
