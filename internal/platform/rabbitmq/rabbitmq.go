package rabbitmq

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// HandlerFunc is function which handles messages.
type HandlerFunc func(ctx context.Context, message []byte) error

// RabbitMQ consumes and publishes amqp messages.
type RabbitMQ struct {
	channel   *amqp.Channel
	exchange  string
	isRunning chan struct{}
}

// NewRabbitMQ returns new RabbitMQ.
func NewRabbitMQ(connection *amqp.Connection, exchange string) (*RabbitMQ, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("can't open channel: %w", err)
	}
	mq := RabbitMQ{
		channel:  channel,
		exchange: exchange,
	}

	return &mq, nil
}

// Publish publishes message to routing key.
func (mq *RabbitMQ) Publish(ctx context.Context, routingKey string, message []byte) error {
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        message,
	}

	return mq.channel.PublishWithContext(
		ctx,
		mq.exchange,
		routingKey,
		false,
		false,
		msg,
	)
}

// Consume consumes messages from queue and passes deliveries to provided handler function.
// It returns channel with errors from handler function and consuming process.
// Function works asynchronously, it consumes messages in background as long as context is not closed.
func (mq *RabbitMQ) Consume(ctx context.Context, queue string, handler HandlerFunc) (<-chan error, error) {
	consumerID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("can't create consumer ID: %w", err)
	}

	deliveries, err := mq.channel.Consume(
		queue,
		consumerID.String(),
		false, // auto acknowledge
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("can't start consuming: %w", err)
	}

	consumingErrors := make(chan error)
	mq.isRunning = make(chan struct{})
	go func() {
		defer close(mq.isRunning)
		mq.consumeMessages(ctx, deliveries, consumingErrors, handler)
	}()

	return consumingErrors, nil
}

func (mq *RabbitMQ) consumeMessages(
	ctx context.Context,
	deliveries <-chan amqp.Delivery,
	consumingErrors chan error,
	handler HandlerFunc,
) {
	for delivery := range deliveries {
		err := handler(ctx, delivery.Body)
		if err != nil {
			_ = pushError(ctx, err, consumingErrors)
			if err := mq.nackMessage(ctx, &delivery, consumingErrors); err != nil {
				return
			}
			continue
		}
		if err := mq.ackMessage(ctx, &delivery, consumingErrors); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (mq *RabbitMQ) ackMessage(
	ctx context.Context,
	delivery *amqp.Delivery,
	consumingErrors chan error,
) error {
	if err := delivery.Ack(false); err != nil {
		if pushErr := pushError(ctx, fmt.Errorf("can't ack message: %w", err), consumingErrors); pushErr != nil {
			return pushErr
		}
	}
	return nil
}

func (mq *RabbitMQ) nackMessage(
	ctx context.Context,
	delivery *amqp.Delivery,
	consumingErrors chan error,
) error {
	if err := delivery.Nack(false, false); err != nil {
		if pushErr := pushError(ctx, fmt.Errorf("can't nack message: %w", err), consumingErrors); pushErr != nil {
			return pushErr
		}
	}
	return nil
}

// Done returns channel which will be closed when consuming will be finished.
func (mq *RabbitMQ) Done() chan struct{} {
	return mq.isRunning
}

func pushError(ctx context.Context, err error, errChan chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case errChan <- err:
	}
	return nil
}
