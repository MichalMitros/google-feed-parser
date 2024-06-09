package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MichalMitros/google-feed-parser/cmd/parser/config"
	"github.com/samber/lo"

	"github.com/MichalMitros/google-feed-parser/e2e/helpers"
	"github.com/MichalMitros/google-feed-parser/internal/decoder"
	"github.com/MichalMitros/google-feed-parser/internal/fetcher"
	"github.com/MichalMitros/google-feed-parser/internal/handler"
	"github.com/MichalMitros/google-feed-parser/internal/parser"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/MichalMitros/google-feed-parser/internal/platform/rabbitmq"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage/storagetesting"
	"github.com/MichalMitros/google-feed-parser/pkg/v1/commander"
	"github.com/caarlos0/env/v6"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	userAgent = "gfp-e2e-test/0.0.1"
	exchange  = "gfp-e2e"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

type E2ETestSuite struct {
	suite.Suite
	cfg        *config.Config
	connection *amqp.Connection
	channel    *amqp.Channel
	db         *sql.DB
}

func (s *E2ETestSuite) SetupSuite() {
	var err error

	var cfg config.Config
	if err = env.Parse(&cfg); err != nil {
		s.Require().FailNow("can't parse env variables", err)
	}
	s.cfg = &cfg

	if s.connection, err = amqp.Dial(cfg.RabbitMQ.URL); err != nil {
		s.Require().FailNow("can't open RabbitMQ connection", err)
	}

	if s.channel, err = s.connection.Channel(); err != nil {
		s.Require().FailNow("can't open RabbitMQ channel", err)
	}

	helpers.DeclareRMQExchange(s.T(), s.channel, exchange)

	if s.db, err = sql.Open("postgres", cfg.DatabaseURL); err != nil {
		s.Require().FailNow("can't open Postgres connection", err)
	}
}

func (s *E2ETestSuite) TearDownSuite() {
	storagetesting.CleanupData(s.T(), s.db)
	if err := s.db.Close(); err != nil {
		s.FailNow("can't close Postgres connection", err)
	}

	if err := s.channel.Close(); err != nil {
		s.FailNow("can't close RabbitMQ channel", err)
	}

	if err := s.connection.Close(); err != nil {
		s.FailNow("can't close RabbitMQ connection", err)
	}
}

func (s *E2ETestSuite) TestFeedParsing() {
	ctx, cancel := context.WithCancel(context.Background())

	// Prepare test RMQ queue
	queue := fmt.Sprintf("gfp-e2e-test-%d", rand.Int63n(100000))
	routingKey := fmt.Sprintf("gfp.cmd.e2e.%d", rand.Int63n(100000))
	helpers.DeclareRMQQueue(s.T(), s.channel, queue, exchange, routingKey)

	// Prepare test data
	products := helpers.GenerateTestData(s.T(), 45)
	firstFeedProducts := products[:25] // first 25 products
	// last 35 product, so finally all products should be inserted and first 10 should be marked as deleted
	secondFeedProducts := products[10:]
	firstFeedFile := helpers.ProductsToXML(s.T(), helpers.ToDecoderProduct(s.T(), firstFeedProducts))
	secondFeedFile := helpers.ProductsToXML(s.T(), helpers.ToDecoderProduct(s.T(), secondFeedProducts))

	// Mock http server
	httpSrv, setFeedFile := helpers.PrepareMockedHTTPServer(s.T(), [][]byte{firstFeedFile, secondFeedFile}, http.StatusOK)
	setFeedFile(0)
	shopURL := fmt.Sprintf("%s/%d.xml", httpSrv.URL, rand.Intn(100000))

	// Prepare parser
	par := parser.NewParser(
		fetcher.NewFetcher(httpSrv.Client(), userAgent),
		&decoder.Decoder{},
		storage.NewPostgres(s.db),
		s.cfg.BatchSize,
	)

	// Prepare RMQ client and commander
	rmq, err := rabbitmq.NewRabbitMQ(s.connection, exchange)
	if err != nil {
		s.Require().FailNow("can't create RabbitMQ client", err)
	}
	publisher := commander.NewParseCommander(commander.NewRabbitMQSender(rmq, routingKey))

	// Prepare test logger
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	// Prepare and run handler
	han := handler.NewHandler(rmq, par, &logger)
	handlerErr := han.Start(ctx, queue)
	s.Require().NoError(handlerErr, "handler shouldn't return any error")

	// Send parse command
	if err := publisher.SendParseCommand(ctx, shopURL); err != nil {
		s.Require().FailNow("can't publish parse command", err)
	}

	// Wait for feed processing to be finished
	firstRun := helpers.WaitForRunToBeFinished(s.T(), s.db, shopURL)

	dbProducts := helpers.GetProducts(s.T(), s.db, shopURL)

	s.Equal(int32(len(firstFeedProducts)), *firstRun.CreatedProducts, "should return correct number of created products")
	s.Equal(int32(0), *firstRun.UpdatedProducts, "should return correct number of updated products")
	s.Equal(int32(0), *firstRun.DeletedProducts, "should return correct number of deleted products")
	s.Equal(int32(0), *firstRun.FailedProducts, "should return correct number of failed products")
	assertProducts(s.T(), firstFeedProducts, firstRun.ProductsVersion, dbProducts)

	// Second iteration
	setFeedFile(1)

	// Send parse command
	if err := publisher.SendParseCommand(ctx, shopURL); err != nil {
		s.Require().FailNow("can't publish parse command", err)
	}

	// Wait for feed processing to be finished
	secondRun := helpers.WaitForRunToBeFinished(s.T(), s.db, shopURL)

	// Cancel context to stop consumer
	cancel()

	// Check results
	logs := strings.Split(buf.String(), "\n")
	logs = lo.Filter(logs, func(log string, _ int) bool { return strings.TrimSpace(log) != "" })

	dbProducts = helpers.GetProducts(s.T(), s.db, shopURL)

	s.Equal(int32(20), *secondRun.CreatedProducts, "should return correct number of created products")
	s.Equal(int32(15), *secondRun.UpdatedProducts, "should return correct number of updated products")
	s.Equal(int32(10), *secondRun.DeletedProducts, "should return correct number of deleted products")
	s.Equal(int32(0), *secondRun.FailedProducts, "should return correct number of failed products")
	assertLogsMessages(s.T(), []string{"parsing started", "parsing finished", "parsing started", "parsing finished"}, logs)
	assertDeletedProducts(s.T(), products[:10], firstRun.ProductsVersion, dbProducts[:10])
	assertProducts(s.T(), products[10:], secondRun.ProductsVersion, dbProducts[10:])
}

// assertLogsMessages is helper function which unmarshals log json and asserts message.
func assertLogsMessages(t *testing.T, expected []string, actual []string) {
	t.Helper()

	require.Len(t, actual, len(expected), "incorrect number of logs")

	for ix, exp := range expected {
		var log struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(actual[ix]), &log); err != nil {
			require.FailNow(t, "can't unmarshal json log", err)
		}

		assert.Equalf(t, exp, log.Message, "log at index %d is incorrect", ix)
	}
}

// assertDeletedProducts is helper function for comparing deleted products.
func assertDeletedProducts(t *testing.T, expected []models.Product, expectedVersion int64, actual []models.Product) {
	t.Helper()

	require.Len(t, actual, len(expected), "incorrect number of products")

	lastDeletedAt := lo.ToPtr(time.Now())
	lo.ForEach(expected, func(_ models.Product, i int) {
		if actual[i].DeletedAt == nil {
			expected[i].DeletedAt = lastDeletedAt
			return
		}
		lastDeletedAt = actual[i].DeletedAt
		expected[i].DeletedAt = actual[i].DeletedAt
	})

	assertProducts(t, expected, expectedVersion, actual)
}

// assertProducts is helper function for comparing products.
func assertProducts(t *testing.T, expected []models.Product, expectedVersion int64, actual []models.Product) {
	t.Helper()

	require.Len(t, actual, len(expected), "incorrect number of products")

	lo.ForEach(actual, func(_ models.Product, ix int) { actual[ix].CreatedAt = time.Time{} })
	lo.ForEach(expected, func(_ models.Product, ix int) { expected[ix].Version = expectedVersion })

	for ix, exp := range expected {
		assert.Equalf(t, exp, actual[ix], "product at index %d has incorrect value", ix)
	}
}
