package commander_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MichalMitros/google-feed-parser/pkg/v1/commander"
	"github.com/MichalMitros/google-feed-parser/pkg/v1/commander/mocks"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUniSendParseCommand(t *testing.T) {
	shopURL := faker.Word()
	body := []byte(fmt.Sprintf(`{"shopUrl":"%s"}`, shopURL))

	tests := map[string]struct {
		senderError error
		wantErr     error
	}{
		"ok": {},
		"sender error": {
			senderError: assert.AnError,
			wantErr:     assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sender := mocks.NewSender(t)
			sender.On("Send", mock.Anything, body).Return(tt.senderError)

			cmndr := commander.NewParseCommander(sender)
			err := cmndr.SendParseCommand(context.TODO(), shopURL)

			require.ErrorIs(t, err, tt.wantErr, "should return correct error")
		})
	}
}
