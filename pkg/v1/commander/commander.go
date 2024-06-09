package commander

import (
	"context"
	"encoding/json"
	"fmt"
)

//go:generate mockery --name Sender --filename sender.go

// Sender sends messages.
type Sender interface {
	Send(context.Context, []byte) error
}

// ParseCommander sends parse commands.
type ParseCommander struct {
	sender Sender
}

// NewParseCommander returns new ParseCommander using provided sender for sending messages.
func NewParseCommander(sender Sender) ParseCommander {
	return ParseCommander{
		sender: sender,
	}
}

// SendParseCommand sends parse command with provided shopURL.
func (c ParseCommander) SendParseCommand(ctx context.Context, shopURL string) error {
	cmd := ParseCommand{
		ShopURL: shopURL,
	}

	cmdMsg, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("can't marshal parse command: %w", err)
	}

	return c.sender.Send(ctx, cmdMsg)
}
