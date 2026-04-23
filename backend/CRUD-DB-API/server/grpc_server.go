package server

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"sync"
	"time"

	pb "crud-db-api/proto"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type VoteNotifier interface {
	NotifyVote(ctx context.Context, songID int, voterCountry string) error
}

type NoopVoteNotifier struct{}

func (NoopVoteNotifier) NotifyVote(_ context.Context, _ int, _ string) error { return nil }


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
	Logger.Info("new client subscribed to vote stream", slog.Bool("include_historical", req.IncludeHistorical))

	voteChan := make(chan *pb.Vote, 100)

	s.mu.Lock()
	s.subscribers = append(s.subscribers, voteChan)
	subscriberCount := len(s.subscribers)
	s.mu.Unlock()

	Logger.Info("subscriber registered", slog.Int("total_subscribers", subscriberCount))

	defer func() {
		s.mu.Lock()
		for i, ch := range s.subscribers {
			if ch == voteChan {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				break
			}
		}
		remaining := len(s.subscribers)
		s.mu.Unlock()
		close(voteChan)
		Logger.Info("client disconnected from vote stream", slog.Int("remaining_subscribers", remaining))
	}()

	if req.IncludeHistorical {
		if err := s.sendCurrentVotes(ctx, stream); err != nil {
			Logger.ErrorContext(ctx, "failed to send current votes", slog.Any("error", err))
			return err
		}
	}

	Logger.Info("streaming new votes in real-time")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case vote := <-voteChan:
			if err := stream.Send(vote); err != nil {
				Logger.ErrorContext(ctx, "failed to send vote", slog.Any("error", err))
				return err
			}
		}
	}
}

func (s *voteServer) sendCurrentVotes(ctx context.Context, stream pb.VoteService_StreamVotesServer) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.ID as country_id, l.Name as country_name,
		       s.PublikumsPunkte, s.JuryPunkte, s.GesamtPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
		ORDER BY s.GesamtPunkte DESC, s.ID`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			songID, publicVotes, juryVotes, totalVotes int
			songName, countryID, countryName           string
		)
		if err := rows.Scan(&songID, &songName, &countryID, &countryName, &publicVotes, &juryVotes, &totalVotes); err != nil {
			continue
		}
		if err := stream.Send(&pb.Vote{
			CountryVotedFor:     countryID,
			CountryVotedForName: countryName,
			VoteCount:           int32(totalVotes),
			Timestamp:           time.Now().Unix(),
			SongId:              int32(songID),
			SongName:            songName,
		}); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *voteServer) BroadcastVote(vote *pb.Vote) {
	s.mu.RLock()
	subscribers := make([]chan *pb.Vote, len(s.subscribers))
	copy(subscribers, s.subscribers)
	s.mu.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	for i, ch := range subscribers {
		select {
		case ch <- vote:
		default:
			Logger.Warn("subscriber channel full, skipping", slog.Int("subscriber_index", i))
		}
	}
}

func (s *voteServer) GetSongsWithVotes(ctx context.Context, req *pb.GetSongsRequest) (*pb.GetSongsResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.ID, l.Name, s.PublikumsPunkte
		FROM Song s JOIN Land l ON s.Land_ID = l.ID`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*pb.SongVoteData
	for rows.Next() {
		var sv pb.SongVoteData
		if err := rows.Scan(&sv.SongId, &sv.SongName, &sv.CountryId, &sv.CountryName, &sv.PublicVotes); err != nil {
			return nil, err
		}
		songs = append(songs, &sv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &pb.GetSongsResponse{Songs: songs}, nil
}

func (s *voteServer) NotifyVote(ctx context.Context, songID int, voterCountry string) error {
	notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		id, totalVotes                   int
		songName, countryID, countryName string
		voterCountryName                 string
	)

	g, gctx := errgroup.WithContext(notifyCtx)

	g.Go(func() error {
		return s.db.QueryRowContext(gctx, `
			SELECT s.ID, s.Name, l.ID as country_id, l.Name as country_name,
			       s.PublikumsPunkte + s.JuryPunkte as total_votes
			FROM Song s JOIN Land l ON s.Land_ID = l.ID
			WHERE s.ID = ?`, songID,
		).Scan(&id, &songName, &countryID, &countryName, &totalVotes)
	})

	g.Go(func() error {
		switch voterCountry {
		case "JURY":
			voterCountryName = "Jury"
		case "", "SYSTEM":
			voterCountryName = "Unknown"
		default:
			if err := s.db.QueryRowContext(notifyCtx,
				`SELECT Name FROM Land WHERE ID = ?`, voterCountry,
			).Scan(&voterCountryName); err != nil {
				voterCountryName = voterCountry
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		Logger.Warn("NotifyVote: aborting broadcast", slog.Any("error", err), slog.Int("song_id", songID))
		return err
	}

	s.BroadcastVote(&pb.Vote{
		CountryVotedFor:     countryID,
		CountryVotedForName: countryName,
		VoteCount:           int32(totalVotes),
		VoterCountry:        voterCountry,
		VoterCountryName:    voterCountryName,
		Timestamp:           time.Now().Unix(),
		SongId:              int32(songID),
		SongName:            songName,
	})
	return nil
}

func StartGRPCServer(database *sql.DB, port string) (VoteNotifier, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	vs := newVoteServer(database)
	pb.RegisterVoteServiceServer(grpcServer, vs)
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on port %s", port)
	Logger.Info("gRPC vote stream server starting", slog.String("port", port))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			Logger.Error("gRPC server error", slog.Any("error", err))
		}
	}()

	return vs, nil
}
