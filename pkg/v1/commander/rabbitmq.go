package commander

import "context"

//go:generate mockery --name RabbitMQPublisher --filename rabbitmqpublisher.go

// RabbitMQPublisher is RabbitMQ messages publisher.
type RabbitMQPublisher interface {
	Publish(context.Context, string, []byte) error
}

// RabbitMQSender sends RMQ messages to routing key.
type RabbitMQSender struct {
	publisher     RabbitMQPublisher
	cmdRoutingKey string
}

// NewRabbitMQSender returns new RabbitMQSender using provided publisher for sending messages to provided routing key.
func NewRabbitMQSender(publisher RabbitMQPublisher, cmdRoutingKey string) RabbitMQSender {
	return RabbitMQSender{
		publisher:     publisher,
		cmdRoutingKey: cmdRoutingKey,
	}
}

// Send sends message to RabbitMQSender's routing key.
func (s RabbitMQSender) Send(ctx context.Context, msg []byte) error {
	return s.publisher.Publish(ctx, s.cmdRoutingKey, msg)
}
