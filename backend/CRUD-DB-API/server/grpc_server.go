package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"time"

	pb "crud-db-api/proto"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type voteServer struct {
	pb.UnimplementedVoteServiceServer
	db *sql.DB
}

func newVoteServer(database *sql.DB) *voteServer {
	return &voteServer{db: database}
}

func (s *voteServer) StreamVotes(req *pb.VoteStreamRequest, stream pb.VoteService_StreamVotesServer) error {
	ctx := stream.Context()
	Logger.Info("new client subscribed to vote stream", slog.Bool("include_historical", req.IncludeHistorical))

	if req.IncludeHistorical {
		if err := s.sendCurrentVotes(ctx, stream); err != nil {
			Logger.ErrorContext(ctx, "failed to send current votes", slog.Any("error", err))
			return err
		}
	}

	// Consume from RabbitMQ fanout exchange — each StreamVotes call gets its own queue
	msgs, queueName, err := ConsumeVotes()
	if err != nil {
		Logger.ErrorContext(ctx, "failed to consume votes from RabbitMQ", slog.Any("error", err))
		return err
	}
	Logger.Info("streaming votes via RabbitMQ", slog.String("queue", queueName))

	for {
		select {
		case <-ctx.Done():
			Logger.Info("client context cancelled")
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				Logger.Info("RabbitMQ channel closed")
				return nil
			}

			var voteData map[string]any
			if err := json.Unmarshal(msg.Body, &voteData); err != nil {
				Logger.Error("failed to unmarshal vote from RabbitMQ", slog.Any("error", err))
				continue
			}

			vote := &pb.Vote{
				CountryVotedFor:     voteData["country_voted_for"].(string),
				CountryVotedForName: voteData["country_voted_for_name"].(string),
				VoteCount:           int32(voteData["vote_count"].(float64)),
				VoterCountry:        voteData["voter_country"].(string),
				VoterCountryName:    voteData["voter_country_name"].(string),
				Timestamp:           int64(voteData["timestamp"].(float64)),
				SongId:              int32(voteData["song_id"].(float64)),
				SongName:            voteData["song_name"].(string),
			}

			if err := stream.Send(vote); err != nil {
				Logger.ErrorContext(ctx, "failed to send vote", slog.Any("error", err))
				return err
			}
		}
	}
}

func (s *voteServer) sendCurrentVotes(ctx context.Context, stream pb.VoteService_StreamVotesServer) error {
	Logger.Info("querying database for current votes")

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
		Logger.ErrorContext(ctx, "failed to query current votes", slog.Any("error", err))
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
			Logger.ErrorContext(ctx, "failed to scan vote row", slog.Any("error", err))
			continue
		}

		vote := &pb.Vote{
			CountryVotedFor:     countryID,
			CountryVotedForName: countryName,
			VoteCount:           int32(totalVotes),
			Timestamp:           time.Now().Unix(),
			SongId:              int32(songID),
			SongName:            songName,
		}

		if err := stream.Send(vote); err != nil {
			Logger.ErrorContext(ctx, "failed to send current vote", slog.Any("error", err))
			return err
		}
		voteCount++
	}

	Logger.Info("sent all current votes", slog.Int("count", voteCount))
	return rows.Err()
}

func (s *voteServer) GetSongsWithVotes(ctx context.Context, req *pb.GetSongsRequest) (*pb.GetSongsResponse, error) {
	Logger.InfoContext(ctx, "GetSongsWithVotes called")

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.ID, s.Name, l.ID, l.Name, s.PublikumsPunkte
		FROM Song s
		JOIN Land l ON s.Land_ID = l.ID
	`)
	if err != nil {
		Logger.ErrorContext(ctx, "GetSongsWithVotes: query failed", slog.Any("error", err))
		return nil, err
	}
	defer rows.Close()

	var songs []*pb.SongVoteData
	for rows.Next() {
		var s pb.SongVoteData
		if err := rows.Scan(&s.SongId, &s.SongName, &s.CountryId, &s.CountryName, &s.PublicVotes); err != nil {
			Logger.ErrorContext(ctx, "GetSongsWithVotes: scan failed", slog.Any("error", err))
			return nil, err
		}
		songs = append(songs, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	Logger.InfoContext(ctx, "GetSongsWithVotes complete", slog.Int("count", len(songs)))
	return &pb.GetSongsResponse{Songs: songs}, nil
}

func StartGRPCServer(database *sql.DB, port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	voteService := newVoteServer(database)
	pb.RegisterVoteServiceServer(grpcServer, voteService)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on port %s", port)
	Logger.Info("gRPC vote stream server starting", slog.String("port", port))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			Logger.Error("gRPC server error", slog.Any("error", err))
		}
	}()

	return nil
}

// NotifyVote publishes a vote to RabbitMQ fanout exchange.
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
