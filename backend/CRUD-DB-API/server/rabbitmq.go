package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "crud-db-api/proto"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	amqpConn    *amqp.Connection
	amqpChan    *amqp.Channel
	amqpReady   chan struct{} = make(chan struct{})
)

const (
	VoteFanoutExchange = "votes.fanout"
	EmailQueue         = "email.send"
)

func InitRabbitMQ() error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}

	var err error
	amqpConn, err = amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial failed: %w", err)
	}

	amqpChan, err = amqpConn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel failed: %w", err)
	}

	// Declare fanout exchange for vote broadcasting
	err = amqpChan.ExchangeDeclare(
		VoteFanoutExchange,
		"fanout",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("exchange declare failed: %w", err)
	}

	// Declare email queue
	_, err = amqpChan.QueueDeclare(
		EmailQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare failed: %w", err)
	}

	Logger.Info("RabbitMQ connection established", slog.String("url", url))
	close(amqpReady)
	return nil
}

func rabbitAvailable() bool {
	return amqpChan != nil
}

// PublishVote publishes a vote to the fanout exchange for all consumers.
func PublishVote(vote *pb.Vote) error {
	if !rabbitAvailable() {
		Logger.Warn("RabbitMQ not available, vote not published")
		return nil
	}
	body, err := json.Marshal(map[string]interface{}{
		"country_voted_for":      vote.CountryVotedFor,
		"country_voted_for_name": vote.CountryVotedForName,
		"vote_count":             vote.VoteCount,
		"voter_country":          vote.VoterCountry,
		"voter_country_name":     vote.VoterCountryName,
		"timestamp":              vote.Timestamp,
		"song_id":                vote.SongId,
		"song_name":              vote.SongName,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return amqpChan.PublishWithContext(ctx,
		VoteFanoutExchange,
		"",    // routing key (ignored by fanout)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// EmailMessage represents an email job for RabbitMQ.
type EmailMessage struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

// PublishEmailJob publishes an email job to the email queue.
func PublishEmailJob(email, token string) error {
	if !rabbitAvailable() {
		Logger.Warn("RabbitMQ not available, email job not published")
		return nil
	}
	body, err := json.Marshal(EmailMessage{Email: email, Token: token})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return amqpChan.PublishWithContext(ctx,
		"",         // default exchange
		EmailQueue, // routing key = queue name
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// ConsumeVotes creates an exclusive queue bound to the votes fanout exchange
// and returns a channel of Vote messages. Used by gRPC StreamVotes.
func ConsumeVotes() (<-chan amqp.Delivery, string, error) {
	<-amqpReady

	q, err := amqpChan.QueueDeclare(
		"",    // auto-generated name
		false, // durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, "", err
	}

	err = amqpChan.QueueBind(
		q.Name,
		"",
		VoteFanoutExchange,
		false,
		nil,
	)
	if err != nil {
		return nil, q.Name, err
	}

	msgs, err := amqpChan.Consume(
		q.Name,
		"",    // consumer tag
		true,  // auto-ack
		true,  // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	return msgs, q.Name, err
}

func CloseRabbitMQ() {
	if amqpChan != nil {
		amqpChan.Close()
	}
	if amqpConn != nil {
		amqpConn.Close()
	}
}
