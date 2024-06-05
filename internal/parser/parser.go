package parser

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

//go:generate mockery --name Fetcher --filename fetcher.go
//go:generate mockery --name Decoder --filename decoder.go
//go:generate mockery --name Storage --filename storage.go

// Fetcher fetches feed file.
type Fetcher interface {
	FetchFile(context.Context, string) (io.ReadCloser, error)
}

// Decoder decodes xml feed file into parsing results.
type Decoder interface {
	Decode(context.Context, io.Reader, chan<- models.ParsingResult) error
}

// Clock provides times.
type Clock interface {
	// Timestamp returns UTC unix timestamp.
	Timestamp() int64
	// Now returns current UTC time.
	Now() *time.Time
}

// Storage is products and runs storage.
type Storage interface {
	// StartRun creates new run if there is no run for provided shop running.
	StartRun(ctx context.Context, shopURL string, version int64) (run *models.Run, err error)
	// FinishRun finishes provided run and updates its statistics.
	FinishRun(ctx context.Context, run *models.Run) error
	// UpdateProducts creates new products and updates existing products and their shippings.
	// Returns number of created and updated products.
	UpdateProducts(
		ctx context.Context,
		products []models.Product,
		shopID int,
	) (newProducts int32, updatedProducts int32, err error)
	// DeleteOldProducts deletes from storage all not-deleted products with version lower than provided for provided shop.
	// Returns number of deleted products.
	DeleteOldProducts(
		ctx context.Context,
		shopID int,
		version int64,
		batchSize uint,
	) (deletedProducts int32, err error)
}

// Option is custom configuration of Parser.
type Option func(p *Parser)

// Parser fetches, decodes and parses feed files.
type Parser struct {
	fetcher   Fetcher
	decoder   Decoder
	storage   Storage
	batchSize uint
	clock     Clock
}

// NewParser returns new Parser.
func NewParser(fetcher Fetcher, decoder Decoder, storage Storage, batchSize uint, ops ...Option) *Parser {
	par := &Parser{
		fetcher:   fetcher,
		decoder:   decoder,
		storage:   storage,
		batchSize: batchSize,
		clock:     systemClock{},
	}

	for _, op := range ops {
		op(par)
	}

	return par
}

// Parser parses feed files from shopURL.
func (p Parser) Parse(ctx context.Context, shopURL string) error {
	version := p.clock.Timestamp()

	// insert new run in storage.
	run, err := p.storage.StartRun(ctx, shopURL, version)
	if err != nil {
		return fmt.Errorf("can't start parsing: %w", err)
	}

	// fetch feed file.
	xmlFile, err := p.fetcher.FetchFile(ctx, shopURL)
	if err != nil {
		return p.finishParsing(ctx, run, fmt.Errorf("can't fetch feed file: %w", err))
	}
	defer xmlFile.Close()

	// parse products.
	createdProducts, updatedProducts, failedProducts, err := p.parseProducts(ctx, version, run.ShopID, xmlFile)

	run.CreatedProducts = &createdProducts
	run.UpdatedProducts = &updatedProducts
	run.FailedProducts = &failedProducts

	if err != nil {
		return p.finishParsing(ctx, run, err)
	}

	// delete outdated products.
	deletedProducts, err := p.storage.DeleteOldProducts(ctx, run.ShopID, version, p.batchSize)
	run.DeletedProducts = &deletedProducts

	if err != nil {
		return p.finishParsing(ctx, run, fmt.Errorf("can't delete outdated products: %w", err))
	}

	return p.finishParsing(ctx, run, nil)
}

func (p Parser) parseProducts(
	ctx context.Context,
	version int64,
	shopID int,
	xmlFile io.ReadCloser,
) (int32, int32, int32, error) {
	parsingResults := make(chan models.ParsingResult)
	filteredProducts := make(chan []models.Product)
	failedProducts := int32(0)
	createdProducts := int32(0)
	updatedProducts := int32(0)

	errGroup, egCtx := errgroup.WithContext(ctx)

	// decode feed file.
	errGroup.Go(func() error {
		defer close(parsingResults)
		if err := p.decoder.Decode(egCtx, xmlFile, parsingResults); err != nil {
			return fmt.Errorf("can't decode feed file: %w", err)
		}
		return nil
	})

	// filter decoding results.
	errGroup.Go(func() error {
		defer close(filteredProducts)

		failed, err := p.filterProducts(egCtx, parsingResults, filteredProducts)
		if err != nil {
			return fmt.Errorf("can't filter products: %w", err)
		}
		_ = atomic.AddInt32(&failedProducts, int32(failed))

		return nil
	})

	// update products.
	errGroup.Go(func() error {
		created, updated, err := p.updateProducts(egCtx, shopID, version, filteredProducts)
		_ = atomic.AddInt32(&createdProducts, created)
		_ = atomic.AddInt32(&updatedProducts, updated)

		if err != nil {
			return fmt.Errorf("can't update products: %w", err)
		}

		return nil
	})

	err := errGroup.Wait()

	return createdProducts, updatedProducts, failedProducts, err
}

func (p Parser) filterProducts(
	ctx context.Context,
	input <-chan models.ParsingResult,
	output chan []models.Product,
) (int, error) {
	failedProducts := 0
	batch := make([]models.Product, 0, p.batchSize)

	for result := range input {
		if result.Error != nil {
			failedProducts++
			continue
		}

		batch = append(batch, result.Product)
		if len(batch) == int(p.batchSize) {
			select {
			case <-ctx.Done():
				return failedProducts, ctx.Err()
			case output <- batch:
			}
			batch = make([]models.Product, 0, p.batchSize)
		}
	}

	if len(batch) > 0 {
		select {
		case <-ctx.Done():
			return failedProducts, ctx.Err()
		case output <- batch:
		}
	}

	return failedProducts, nil
}

func (p Parser) updateProducts(
	ctx context.Context,
	shopID int,
	version int64,
	input chan []models.Product,
) (int32, int32, error) {
	createdProducts := int32(0)
	updatedProducts := int32(0)

	for batch := range input {
		lo.ForEach(batch, func(_ models.Product, ix int) { batch[ix].Version = version })
		created, updated, err := p.storage.UpdateProducts(ctx, batch, shopID)
		if err != nil {
			return createdProducts, updatedProducts, err
		}
		createdProducts += created
		updatedProducts += updated
	}

	return createdProducts, updatedProducts, nil
}

func (p Parser) finishParsing(ctx context.Context, run *models.Run, status error) error {
	if status != nil {
		run.StatusMessage = lo.ToPtr(status.Error())
	}
	run.IsSuccess = lo.ToPtr(status == nil)
	run.FinishedAt = p.clock.Now()

	err := p.storage.FinishRun(ctx, run)
	if err != nil && status == nil {
		return fmt.Errorf("can't finish parsing: %w", err)
	}

	if err != nil && status != nil {
		return fmt.Errorf("can't finish failed parsing: %w (fail reason: %w)", err, status)
	}

	return status
}

// WithClock sets Parser's custom Clock.
func WithClock(c Clock) Option {
	return func(p *Parser) {
		p.clock = c
	}
}
