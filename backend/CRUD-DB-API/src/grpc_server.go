package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"sync"
	"time"

	pb "main/proto"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type voteServer struct {
	pb.UnimplementedVoteServiceServer
	db          *sql.DB
	subscribers []chan *pb.Vote
	mu          sync.RWMutex
}

func newVoteServer(database *sql.DB) *voteServer {
	return &voteServer{
		db:          database,
		subscribers: make([]chan *pb.Vote, 0),
	}
}

func (s *voteServer) StreamVotes(req *pb.VoteStreamRequest, stream pb.VoteService_StreamVotesServer) error {
	ctx := stream.Context()
	logger.Info("new client subscribed to vote stream", slog.Bool("include_historical", req.IncludeHistorical))

	voteChan := make(chan *pb.Vote, 100)

	s.mu.Lock()
	s.subscribers = append(s.subscribers, voteChan)
	subscriberCount := len(s.subscribers)
	s.mu.Unlock()

	logger.Info("subscriber registered", slog.Int("total_subscribers", subscriberCount))

	defer func() {
		s.mu.Lock()
		for i, ch := range s.subscribers {
			if ch == voteChan {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				break
			}
		}
		remainingSubscribers := len(s.subscribers)
		s.mu.Unlock()
		close(voteChan)
		logger.Info("client disconnected from vote stream", slog.Int("remaining_subscribers", remainingSubscribers))
	}()

	if req.IncludeHistorical {
		if err := s.sendCurrentVotes(ctx, stream); err != nil {
			logger.ErrorContext(ctx, "failed to send current votes", slog.Any("error", err))
			return err
		}
	}

	logger.Info("streaming new votes in real-time")
	for {
		select {
		case <-ctx.Done():
			logger.Info("client context cancelled")
			return ctx.Err()
		case vote := <-voteChan:
			if err := stream.Send(vote); err != nil {
				logger.ErrorContext(ctx, "failed to send vote", slog.Any("error", err))
				return err
			}
			logger.Debug("vote sent to subscriber",
				slog.Int("song_id", int(vote.SongId)),
				slog.String("country", vote.CountryVotedFor),
			)
		}
	}
}

func (s *voteServer) sendCurrentVotes(ctx context.Context, stream pb.VoteService_StreamVotesServer) error {
	logger.Info("querying database for current votes")

	query := `
		SELECT
			s.ID,
			s.Name,
			l.ID as country_id,
			l.Name as country_name,
			s.PublikumsPunkte,
			s.JuryPunkte,
			s.GesamtPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
		ORDER BY s.GesamtPunkte DESC, s.ID
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query current votes", slog.Any("error", err))
		return err
	}
	defer rows.Close()

	voteCount := 0
	for rows.Next() {
		var (
			songID      int
			songName    string
			countryID   string
			countryName string
			publicVotes int
			juryVotes   int
			totalVotes  int
		)

		if err := rows.Scan(&songID, &songName, &countryID, &countryName, &publicVotes, &juryVotes, &totalVotes); err != nil {
			logger.ErrorContext(ctx, "failed to scan vote row", slog.Any("error", err))
			continue
		}

		vote := &pb.Vote{
			CountryVotedFor:     countryID,
			CountryVotedForName: countryName,
			VoteCount:           int32(totalVotes),
			VoterCountry:        "INITIAL",
			VoterCountryName:    "Initial Query",
			Timestamp:           time.Now().Unix(),
			SongId:              int32(songID),
			SongName:            songName,
		}

		if err := stream.Send(vote); err != nil {
			logger.ErrorContext(ctx, "failed to send current vote", slog.Any("error", err))
			return err
		}
		voteCount++
	}

	logger.Info("sent all current votes", slog.Int("count", voteCount))
	return rows.Err()
}

func (s *voteServer) BroadcastVote(vote *pb.Vote) {
	s.mu.RLock()
	subscriberCount := len(s.subscribers)
	subscribers := make([]chan *pb.Vote, len(s.subscribers))
	copy(subscribers, s.subscribers)
	s.mu.RUnlock()

	if subscriberCount == 0 {
		return
	}

	logger.Info("broadcasting vote to subscribers",
		slog.Int("subscriber_count", subscriberCount),
		slog.Int("song_id", int(vote.SongId)),
		slog.String("country", vote.CountryVotedFor),
	)

	for i, ch := range subscribers {
		select {
		case ch <- vote:

		default:

			logger.Warn("subscriber channel full, skipping vote broadcast", slog.Int("subscriber_index", i))
		}
	}
}

func (s *voteServer) GetSongsWithVotes(ctx context.Context, req *pb.GetSongsRequest) (*pb.GetSongsResponse, error) {
	logger.InfoContext(ctx, "GetSongsWithVotes called")

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.ID, l.Name, s.PublikumsPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
	`)
	if err != nil {
		logger.ErrorContext(ctx, "GetSongsWithVotes: query failed", slog.Any("error", err))
		return nil, err
	}
	defer rows.Close()

	var songs []*pb.SongVoteData
	for rows.Next() {
		var s pb.SongVoteData
		if err := rows.Scan(&s.SongId, &s.SongName, &s.CountryId, &s.CountryName, &s.PublicVotes); err != nil {
			logger.ErrorContext(ctx, "GetSongsWithVotes: scan failed", slog.Any("error", err))
			return nil, err
		}
		songs = append(songs, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "GetSongsWithVotes complete", slog.Int("count", len(songs)))
	return &pb.GetSongsResponse{Songs: songs}, nil
}

func StartGRPCServer(database *sql.DB, port string) (*voteServer, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	voteService := newVoteServer(database)
	pb.RegisterVoteServiceServer(grpcServer, voteService)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on port %s", port)
	logger.Info("gRPC vote stream server starting", slog.String("port", port))

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server error", slog.Any("error", err))
		}
	}()

	return voteService, nil
}

var globalVoteServer *voteServer

func SetGlobalVoteServer(vs *voteServer) {
	globalVoteServer = vs
}

func NotifyVote(songID int, voterCountry string, db *sql.DB) {
	if globalVoteServer == nil {
		logger.Warn("vote server not initialized, skipping notification")
		return
	}

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
			logger.ErrorContext(gctx, "failed to get vote details for notification",
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
			logger.Debug("could not get voter country name",
				slog.String("country_id", voterCountry),
				slog.Any("error", err),
			)
			return nil
		}
		return nil

	})

	if err := g.Wait(); err != nil {
		logger.ErrorContext(ctx, "NotifyVote: aborting broadcast due to DB error",
			slog.Any("error", err),
			slog.Int("song_id", songID),
		)
		return
	}

	vote := &pb.Vote{
		CountryVotedFor:     countryID,
		CountryVotedForName: countryName,
		VoteCount:           int32(totalVotes),
		VoterCountry:        voterCountry,
		VoterCountryName:    voterCountryName,
		Timestamp:           time.Now().Unix(),
		SongId:              int32(songID),
		SongName:            songName,
	}

	globalVoteServer.BroadcastVote(vote)

	logger.Info("vote notification complete",
		slog.Int("song_id", songID),
		slog.String("country_voted_for", countryID),
		slog.Int("total_votes", totalVotes),
		slog.String("voter_country", voterCountry),
	)
}
