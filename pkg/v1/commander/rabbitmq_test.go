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

func TestUniRabbitMQSenderSend(t *testing.T) {
	shopURL := faker.Word()
	body := []byte(fmt.Sprintf(`{"shopUrl":"%s"}`, shopURL))
	routingKey := faker.Word()

	tests := map[string]struct {
		publisherError error
		wantErr        error
	}{
		"ok": {},
		"publisher error": {
			publisherError: assert.AnError,
			wantErr:        assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			publisher := mocks.NewRabbitMQPublisher(t)
			publisher.On("Publish", mock.Anything, routingKey, body).Return(tt.publisherError)

			sender := commander.NewRabbitMQSender(publisher, routingKey)
			err := sender.Send(context.TODO(), body)

			require.ErrorIs(t, err, tt.wantErr, "should return correct error")
		})
	}
}
