package parser_test

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/MichalMitros/google-feed-parser/internal/parser"
	"github.com/MichalMitros/google-feed-parser/internal/parser/mocks"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models/modelstesting"
	"github.com/go-faker/faker/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// reusable test data
var (
	batchSize = uint(2) // will affect tests results when changed
	shopURL   = faker.Word()
	version   = rand.Int63()
	loc       = func() *time.Location {
		loc, err := time.LoadLocation("Etc/UTC")
		if err != nil {
			panic(err)
		}
		return loc
	}()
	createdAt = time.Date(2020, time.April, 1, 1, 1, 1, 0, loc)
	now       = time.Date(2022, time.April, 1, 1, 1, 1, 0, loc)
	results   = []models.ParsingResult{ // will affect tests results when changed
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Error: assert.AnError},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Error: assert.AnError},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
		{Product: modelstesting.FakeProduct(func(p *models.Product) { p.Version = version })},
	}
	runID                          = rand.Int()
	shopID                         = rand.Int()
	errShouldContainAssertErrorMsg = "should return error containing assert.AnError"
)

func TestUnitParse(t *testing.T) {
	run := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		ProductsVersion: version,
	}

	// non-failed results in batches of 2
	toUpdate := [][]models.Product{
		{results[0].Product, results[1].Product},
		{results[3].Product, results[4].Product},
		{results[6].Product, results[7].Product},
		{results[8].Product},
	}

	wantNewProducts := 4
	wantUpdatedProducts := 3
	wantDeletedProducts := rand.Int31()
	wantFailedProducts := 2
	wantRun := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		FinishedAt:      &now,
		IsSuccess:       lo.ToPtr(true),
		CreatedProducts: lo.ToPtr(int32(wantNewProducts)),
		UpdatedProducts: lo.ToPtr(int32(wantUpdatedProducts)),
		DeletedProducts: lo.ToPtr(wantDeletedProducts),
		FailedProducts:  lo.ToPtr(int32(wantFailedProducts)),
		ProductsVersion: version,
	}

	fetcher := mocks.NewFetcher(t)
	decoder := mocks.NewDecoder(t)
	storage := mocks.NewStorage(t)

	mockStorageStartRun(storage, shopURL, run, nil)
	mockFetcher(fetcher, shopURL, nil)
	mockDecoder(decoder, results, nil)
	for ix := range toUpdate {
		// first products is always new, second (if exists) is updated
		mockStorageUpdateProducts(storage, toUpdate[ix], run.ShopID, 1, int32(len(toUpdate[ix])-1), nil)
	}
	mockStorageDeleteOldProducts(storage, run.ShopID, version, batchSize, wantDeletedProducts, nil)
	mockStorageFinishRun(storage, wantRun, nil)

	par := parser.NewParser(
		fetcher,
		decoder,
		storage,
		batchSize,
		parser.WithClock(fakeClock{timestamp: version, now: &now}),
	)

	err := par.Parse(context.TODO(), shopURL)

	require.NoError(t, err, "shouldn't return any error")
}

func TestUnitParseStorageError(t *testing.T) {
	t.Run("start run error", func(t *testing.T) {
		run := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			ProductsVersion: version,
		}

		fetcher := mocks.NewFetcher(t)
		decoder := mocks.NewDecoder(t)
		storage := mocks.NewStorage(t)

		mockStorageStartRun(storage, shopURL, run, assert.AnError)

		par := parser.NewParser(
			fetcher,
			decoder,
			storage,
			batchSize,
			parser.WithClock(fakeClock{timestamp: version, now: &now}),
		)

		err := par.Parse(context.TODO(), shopURL)

		require.ErrorContains(t, err,
			"can't start parsing",
			"should return error about failed parsing start",
		)
		require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
	})

	t.Run("update products error", func(t *testing.T) {
		run := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			ProductsVersion: version,
		}

		toUpdate := [][]models.Product{
			{results[0].Product, results[1].Product},
			{results[3].Product, results[4].Product},
		}

		wantNewProducts := 1
		wantUpdatedProducts := 1
		wantFailedProducts := 2
		wantRun := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			FinishedAt:      &now,
			IsSuccess:       lo.ToPtr(false),
			StatusMessage:   lo.ToPtr("can't update products: assert.AnError general error for testing"),
			CreatedProducts: lo.ToPtr(int32(wantNewProducts)),
			UpdatedProducts: lo.ToPtr(int32(wantUpdatedProducts)),
			FailedProducts:  lo.ToPtr(int32(wantFailedProducts)),
			ProductsVersion: version,
		}

		fetcher := mocks.NewFetcher(t)
		decoder := mocks.NewDecoder(t)
		storage := mocks.NewStorage(t)

		mockStorageStartRun(storage, shopURL, run, nil)
		mockFetcher(fetcher, shopURL, nil)
		mockDecoder(decoder, results[:6], nil)
		mockStorageUpdateProducts(storage, toUpdate[0], run.ShopID, 1, 1, nil)
		mockStorageUpdateProducts(storage, toUpdate[1], run.ShopID, 0, 0, assert.AnError)
		mockStorageFinishRun(storage, wantRun, nil)

		par := parser.NewParser(
			fetcher,
			decoder,
			storage,
			batchSize,
			parser.WithClock(fakeClock{timestamp: version, now: &now}),
		)

		err := par.Parse(context.TODO(), shopURL)

		require.ErrorContains(t, err,
			"can't update products",
			"should return error about failed products updating",
		)
		require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
	})

	t.Run("delete old products error", func(t *testing.T) {
		run := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			ProductsVersion: version,
		}

		// non-failed results in batches of 2
		toUpdate := [][]models.Product{
			{results[0].Product, results[1].Product},
			{results[3].Product, results[4].Product},
			{results[6].Product, results[7].Product},
			{results[8].Product},
		}

		wantNewProducts := 4
		wantUpdatedProducts := 3
		wantDeletedProducts := rand.Int31()
		wantFailedProducts := 2
		wantRun := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			FinishedAt:      &now,
			IsSuccess:       lo.ToPtr(false),
			StatusMessage:   lo.ToPtr("can't delete outdated products: assert.AnError general error for testing"),
			CreatedProducts: lo.ToPtr(int32(wantNewProducts)),
			UpdatedProducts: lo.ToPtr(int32(wantUpdatedProducts)),
			DeletedProducts: lo.ToPtr(wantDeletedProducts),
			FailedProducts:  lo.ToPtr(int32(wantFailedProducts)),
			ProductsVersion: version,
		}

		fetcher := mocks.NewFetcher(t)
		decoder := mocks.NewDecoder(t)
		storage := mocks.NewStorage(t)

		mockStorageStartRun(storage, shopURL, run, nil)
		mockFetcher(fetcher, shopURL, nil)
		mockDecoder(decoder, results, nil)
		for ix := range toUpdate {
			// first products is always new, second (if exists) is updated
			mockStorageUpdateProducts(storage, toUpdate[ix], run.ShopID, 1, int32(len(toUpdate[ix])-1), nil)
		}
		mockStorageDeleteOldProducts(storage, run.ShopID, version, batchSize, wantDeletedProducts, assert.AnError)
		mockStorageFinishRun(storage, wantRun, nil)

		par := parser.NewParser(
			fetcher,
			decoder,
			storage,
			batchSize,
			parser.WithClock(fakeClock{timestamp: version, now: &now}),
		)

		err := par.Parse(context.TODO(), shopURL)

		require.ErrorContains(t, err,
			"can't delete outdated products",
			"should return error about failed deleting old products",
		)
		require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
	})

	t.Run("finish run error", func(t *testing.T) {
		run := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			ProductsVersion: version,
		}

		wantRun := &models.Run{
			ID:              runID,
			ShopID:          shopID,
			CreatedAt:       createdAt,
			FinishedAt:      &now,
			IsSuccess:       lo.ToPtr(false),
			StatusMessage:   lo.ToPtr("can't fetch feed file: assert.AnError general error for testing"),
			ProductsVersion: version,
		}

		fetcher := mocks.NewFetcher(t)
		decoder := mocks.NewDecoder(t)
		storage := mocks.NewStorage(t)

		mockStorageStartRun(storage, shopURL, run, nil)
		mockFetcher(fetcher, shopURL, assert.AnError)
		mockStorageFinishRun(storage, wantRun, assert.AnError)

		par := parser.NewParser(
			fetcher,
			decoder,
			storage,
			batchSize,
			parser.WithClock(fakeClock{timestamp: version, now: &now}),
		)

		err := par.Parse(context.TODO(), shopURL)

		require.ErrorContains(t, err, "can't finish failed parsing", "should return error about failed run finishing")
		require.ErrorContains(t, err, "can't fetch feed file", "should return error about failed fetching")
		require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
	})
}

func TestUnitParseFetcherError(t *testing.T) {
	run := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		ProductsVersion: version,
	}

	wantRun := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		FinishedAt:      &now,
		IsSuccess:       lo.ToPtr(false),
		StatusMessage:   lo.ToPtr("can't fetch feed file: assert.AnError general error for testing"),
		ProductsVersion: version,
	}

	fetcher := mocks.NewFetcher(t)
	decoder := mocks.NewDecoder(t)
	storage := mocks.NewStorage(t)

	mockStorageStartRun(storage, shopURL, run, nil)
	mockFetcher(fetcher, shopURL, assert.AnError)
	mockStorageFinishRun(storage, wantRun, nil)

	par := parser.NewParser(
		fetcher,
		decoder,
		storage,
		batchSize,
		parser.WithClock(fakeClock{timestamp: version, now: &now}),
	)

	err := par.Parse(context.TODO(), shopURL)

	require.ErrorContains(t, err, "can't fetch feed file", "should return error about failed fetching")
	require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
}

func TestUnitParseDecoderError(t *testing.T) {
	run := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		ProductsVersion: version,
	}

	toUpdate := []models.Product{results[0].Product, results[1].Product}

	wantNewProducts := 1
	wantUpdatedProducts := 1
	wantFailedProducts := 0
	wantRun := &models.Run{
		ID:              runID,
		ShopID:          shopID,
		CreatedAt:       createdAt,
		FinishedAt:      &now,
		IsSuccess:       lo.ToPtr(false),
		StatusMessage:   lo.ToPtr("can't decode feed file: assert.AnError general error for testing"),
		CreatedProducts: lo.ToPtr(int32(wantNewProducts)),
		UpdatedProducts: lo.ToPtr(int32(wantUpdatedProducts)),
		FailedProducts:  lo.ToPtr(int32(wantFailedProducts)),
		ProductsVersion: version,
	}

	fetcher := mocks.NewFetcher(t)
	decoder := mocks.NewDecoder(t)
	storage := mocks.NewStorage(t)

	mockStorageStartRun(storage, shopURL, run, nil)
	mockFetcher(fetcher, shopURL, nil)
	mockDecoder(decoder, results[:2], assert.AnError)
	mockStorageUpdateProducts(storage, toUpdate, run.ShopID, 1, 1, nil)
	mockStorageFinishRun(storage, wantRun, nil)

	par := parser.NewParser(
		fetcher,
		decoder,
		storage,
		batchSize,
		parser.WithClock(fakeClock{timestamp: version, now: &now}),
	)

	err := par.Parse(context.TODO(), shopURL)

	require.ErrorContains(t, err, "can't decode feed file", "should return error about failed decoding")
	require.ErrorIs(t, err, assert.AnError, errShouldContainAssertErrorMsg)
}

func mockStorageStartRun(storage *mocks.Storage, shopURL string, run *models.Run, err error) {
	storage.On("StartRun", mock.Anything, shopURL, mock.AnythingOfType("int64")).Return(run, err)
}

func mockStorageFinishRun(storage *mocks.Storage, run *models.Run, err error) {
	storage.On("FinishRun", mock.Anything, run).Return(err)
}

func mockStorageUpdateProducts(
	storage *mocks.Storage,
	products []models.Product,
	shopID int, newProducts,
	updatedProducts int32,
	err error,
) {
	storage.On("UpdateProducts", mock.Anything, products, shopID).Return(newProducts, updatedProducts, err)
}

func mockStorageDeleteOldProducts(
	storage *mocks.Storage,
	shopID int,
	version int64,
	batchSize uint,
	deletedProducts int32,
	err error,
) {
	storage.On("DeleteOldProducts", mock.Anything, shopID, version, batchSize).Return(deletedProducts, err)
}

func mockDecoder(decoder *mocks.Decoder, results []models.ParsingResult, err error) {
	decoder.On("Decode", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		output := args.Get(2).(chan<- models.ParsingResult)
		ctx := args.Get(0).(context.Context)
		for ix := range results {
			select {
			case <-ctx.Done():
				return
			case output <- results[ix]:
			}
		}
	}).Return(err)
}

func mockFetcher(fetcher *mocks.Fetcher, shopURL string, err error) {
	var reader io.ReadCloser
	if err == nil {
		reader = io.NopCloser(strings.NewReader(""))
	}
	fetcher.On("FetchFile", mock.Anything, shopURL).Return(reader, err)
}

type fakeClock struct {
	timestamp int64
	now       *time.Time
}

func (c fakeClock) Timestamp() int64 {
	return c.timestamp
}

func (c fakeClock) Now() *time.Time {
	return c.now
}
