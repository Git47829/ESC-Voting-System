package server

import "time"

// Countrys represents a country entry in the ESC voting system.
type Countrys struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Pot  *int   `json:"pot"`
}

// Composer represents a song composer/lyricist.
type Composer struct {
	ID        int `json:"id"`
	firstName string
	name      string
}

// CompleteESCEntryWithComposers is the full song entry returned by song queries.
type CompleteESCEntryWithComposers struct {
	SongID       int    `json:"songId"`
	SongName     string `json:"songName"`
	PublicPoints int    `json:"publicVotes"`
	JuryPoints   int    `json:"juryVotes"`
	TotalPoints  int    `json:"totalVotes"`

	CountryID   string `json:"countryId"`
	CountryName string `json:"countryName"`
	CountryPOT  *int   `json:"countryPot,omitempty"`

	ArtistID        int    `json:"artistId"`
	ArtistFirstName string `json:"artistFirstName"`
	ArtistName      string `json:"artistLastName"`
	ArtistType      string `json:"artistType"`

	Composer []Composer `json:"composers"`

	VotingID         int    `json:"votingId"`
	VotingIsOpen     bool   `json:"votingIsOpen"`
	VotingLastChange string `json:"votingLastChange"`
}

// CookieVoteState holds the vote state encoded in the browser cookie.
type CookieVoteState struct {
	VotesRemaining int            `json:"votes_remaining"`
	VotesCast      map[string]int `json:"votes_cast"`
}

// SongVote is used by GetVotes to return vote totals per song.
type SongVote struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Country         string `json:"country"`
	PublikumsPunkte int    `json:"publicVotes"`
	JuryPunkte      int    `json:"juryVotes"`
	GesamtPunkte    int    `json:"totalVotes"`
}

// ContestSong holds full song details returned during an active contest run.
type ContestSong struct {
	SongID          int
	SongName        string
	YoutubeURL      string
	CountryID       string
	CountryName     string
	ArtistID        int
	ArtistFirstName string
	ArtistLastName  string
	ArtistType      string
	PublicVotes     int
	JuryVotes       int
	TotalVotes      int
	VotingIsOpen    bool
}

// PendingVerification holds a 2FA code awaiting confirmation.
type PendingVerification struct {
	Code      string
	Role      string
	CreatedAt time.Time
}

// AuthLoginRequest is the JSON body for the /auth/login endpoint.
type AuthLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// AuthVerifyRequest is the JSON body for the /auth/verify endpoint.
type AuthVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
