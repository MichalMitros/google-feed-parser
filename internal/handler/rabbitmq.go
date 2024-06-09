package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MichalMitros/google-feed-parser/internal/platform/rabbitmq"
	"github.com/MichalMitros/google-feed-parser/pkg/v1/commander"
	"github.com/rs/zerolog"
)

// Parser parses feed files from feed url.
type Parser interface {
	Parse(ctx context.Context, shopURL string) error
}

// RMQHandler handles RMQ messages.
type RMQHandler struct {
	rmq    *rabbitmq.RabbitMQ
	parser Parser
	logger *zerolog.Logger
}

// NewHandler returns new RMQHandler.
func NewHandler(rmq *rabbitmq.RabbitMQ, parser Parser, logger *zerolog.Logger) *RMQHandler {
	return &RMQHandler{
		rmq:    rmq,
		parser: parser,
		logger: logger,
	}
}

// Start starts consuming and handling parsing commands from RMQ.
func (h *RMQHandler) Start(ctx context.Context, queue string) error {
	errorsChan, err := h.rmq.Consume(ctx, queue, func(ctx context.Context, message []byte) error {
		cmd, err := decodeMessage(message)
		if err != nil {
			return err
		}

		h.logger.Debug().
			Str("shopUrl", cmd.ShopURL).
			Msg("parsing started")

		err = h.parser.Parse(ctx, cmd.ShopURL)
		if err != nil {
			return fmt.Errorf("parsing failed: %w", err)
		}

		h.logger.Debug().
			Str("shopUrl", cmd.ShopURL).
			Msg("parsing finished")

		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		for err := range errorsChan {
			h.logger.Error().
				Err(err).
				Msg("can't handle message")
		}
	}()

	return nil
}

func decodeMessage(msg []byte) (*commander.ParseCommand, error) {
	var cmd commander.ParseCommand
	err := json.Unmarshal(msg, &cmd)
	if err != nil {
		return nil, fmt.Errorf("can't decode parse command: %w", err)
	}

	return &cmd, err
}
