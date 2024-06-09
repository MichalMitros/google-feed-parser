package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/MichalMitros/google-feed-parser/cmd/parser/config"
	"github.com/MichalMitros/google-feed-parser/internal/decoder"
	"github.com/MichalMitros/google-feed-parser/internal/fetcher"
	"github.com/MichalMitros/google-feed-parser/internal/handler"
	"github.com/MichalMitros/google-feed-parser/internal/parser"
	"github.com/MichalMitros/google-feed-parser/internal/platform/rabbitmq"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage"
	"github.com/caarlos0/env/v6"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	// UserAgent is user agent header value used when fetching feed file.
	UserAgent = "google-feed-parser/0.0.1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	var cfg config.Config
	if err := env.Parse(&cfg); err != nil {
		logger.Fatal().
			Err(err).
			Msg("can't parse env variables")
	}

	amqpConnection, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("can't open RabbitMQ connection")
	}

	conn, err := rabbitmq.NewRabbitMQ(amqpConnection, cfg.RabbitMQ.Exchange)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("can't open RabbitMQ connection")
	}

	pgDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("can't open Postgres connection")
	}

	par := parser.NewParser(
		fetcher.NewFetcher(&http.Client{Timeout: cfg.HTTPTimeout}, UserAgent),
		&decoder.Decoder{},
		storage.NewPostgres(pgDB),
		cfg.BatchSize,
	)

	han := handler.NewHandler(conn, par, &logger)

	// start consuming and handling messages
	err = han.Start(ctx, cfg.RabbitMQ.Queue)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("can't start consuming")
	}

	logger.Info().Msg("feed parser up and running")

	// handle graceful shutdown and context cancellation
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-termChan:
		cancel()
	case <-ctx.Done():
	}

	logger.Info().Msg("graceful shutdown start")

	// wait for consumer to finish
	<-conn.Done()

	// close connections
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := pgDB.Close(); err != nil {
			logger.Fatal().
				Err(err).
				Msg("can't close Postgres connection")
		}
	}()

	go func() {
		defer wg.Done()
		if err := amqpConnection.Close(); err != nil {
			logger.Fatal().
				Err(err).
				Msg("can't close RabbitMQ connection")
		}
	}()

	wg.Wait()

	logger.Info().Msg("graceful shutdown successful")
}
