package config

import "time"

// Config holds application configuration.
type Config struct {
	DatabaseURL string        `env:"DATABASE_URL"`
	BatchSize   uint          `env:"BATCH_SIZE" envDefault:"50"`
	HTTPTimeout time.Duration `env:"HTTP_TIMEOUT" envDefault:"10s"`

	RabbitMQ RabbitMQ
}

// RabbitMQ holds RabbitMQ configuration.
type RabbitMQ struct {
	URL      string `env:"RABBITMQ_URL"`
	Exchange string `env:"RABBITMQ_EXCHANGE" envDefault:"gfp-ex"`
	Queue    string `env:"RABBITMQ_QUEUE" envDefault:"google-feed-parser.commands"`
}
