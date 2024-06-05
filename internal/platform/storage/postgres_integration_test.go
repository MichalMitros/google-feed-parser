package storage_test

import (
	"context"
	"database/sql"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MichalMitros/google-feed-parser/internal/platform"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models/modelstesting"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage"
	pgmodels "github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/model"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage/storagetesting"
	"github.com/go-faker/faker/v4"
	_ "github.com/lib/pq"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var loc = func() *time.Location {
	loc, err := time.LoadLocation("Etc/UTC")
	if err != nil {
		panic(err)
	}
	return loc
}()

func TestPostgresIntegration(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

type PostgresTestSuite struct {
	suite.Suite
	DB *sql.DB
}

func (s *PostgresTestSuite) SetupSuite() {
	s.DB = storagetesting.Open(s.T())
	storagetesting.CleanupData(s.T(), s.DB)
}

func (s *PostgresTestSuite) TearDownSuite() {
	storagetesting.CleanupData(s.T(), s.DB)
	if err := s.DB.Close(); err != nil {
		s.FailNow("close DB", err)
	}
}

func (s *PostgresTestSuite) TestIntegrationStartRun() {
	storagetesting.CleanupData(s.T(), s.DB)
	shopURL := faker.Word()
	version := rand.Int63()

	tests := map[string]struct {
		storedShop *pgmodels.Shop
		storedRuns []pgmodels.Run
		wantRun    *models.Run
		wantErr    error
	}{
		"new shop": {
			wantRun: &models.Run{
				ProductsVersion: version,
			},
		},
		"first run": {
			storedShop: &pgmodels.Shop{
				ID:  123,
				URL: shopURL,
			},
			wantRun: &models.Run{
				ShopID:          123,
				ProductsVersion: version,
			},
		},
		"after successful run": {
			storedShop: &pgmodels.Shop{
				ID:  123,
				URL: shopURL,
			},
			storedRuns: []pgmodels.Run{
				{
					ShopID:          123,
					ProductsVersion: version - 1,
					Success:         lo.ToPtr(true),
					FinishedAt:      lo.ToPtr(time.Now()),
				},
			},
			wantRun: &models.Run{
				ShopID:          123,
				ProductsVersion: version,
			},
		},
		"after failed run": {
			storedShop: &pgmodels.Shop{
				ID:  123,
				URL: shopURL,
			},
			storedRuns: []pgmodels.Run{
				{
					ShopID:          123,
					ProductsVersion: version - 1,
					Success:         lo.ToPtr(false),
					FinishedAt:      lo.ToPtr(time.Now()),
				},
			},
			wantRun: &models.Run{
				ShopID:          123,
				ProductsVersion: version,
			},
		},
		"already running error": {
			storedShop: &pgmodels.Shop{
				ID:  123,
				URL: shopURL,
			},
			storedRuns: []pgmodels.Run{
				{
					ShopID:          123,
					ProductsVersion: version - 1,
				},
			},
			wantErr: platform.ErrAlreadyRunning,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			defer storagetesting.CleanupData(s.T(), s.DB)

			if tt.storedShop != nil {
				storagetesting.InsertShops(s.T(), s.DB, *tt.storedShop)
			}

			if len(tt.storedRuns) > 0 {
				storagetesting.InsertRuns(s.T(), s.DB, tt.storedRuns...)
			}

			post := storage.NewPostgres(s.DB)

			run, err := post.StartRun(context.TODO(), shopURL, version)

			if tt.wantErr == nil {
				s.Require().NoError(err, "shouldn't return any error")
				assertRun(s.T(), tt.wantRun, run)
			} else {
				s.Require().ErrorIs(err, tt.wantErr, "should return correct error")
			}
		})
	}
}

func (s *PostgresTestSuite) TestIntegrationFinishRun() {
	storagetesting.CleanupData(s.T(), s.DB)
	version := rand.Int63()
	createdAt := time.Date(2024, time.April, 1, 1, 1, 1, 0, loc)
	finishedAt := time.Date(2024, time.April, 1, 2, 1, 1, 0, loc)
	shopID := 1

	runsState := []pgmodels.Run{
		{
			ID:              1,
			ShopID:          int32(shopID),
			CreatedAt:       createdAt,
			ProductsVersion: version,
		},
		{
			ID:              2,
			ShopID:          int32(shopID),
			CreatedAt:       createdAt,
			ProductsVersion: rand.Int63(),
			CreatedProducts: lo.ToPtr(rand.Int31()),
			UpdatedProducts: lo.ToPtr(rand.Int31()),
			DeletedProducts: lo.ToPtr(rand.Int31()),
			Success:         lo.ToPtr(true),
		},
		{
			ID:              3,
			ShopID:          int32(shopID),
			CreatedAt:       createdAt,
			ProductsVersion: rand.Int63(),
			CreatedProducts: lo.ToPtr(rand.Int31()),
			UpdatedProducts: lo.ToPtr(rand.Int31()),
			DeletedProducts: lo.ToPtr(rand.Int31()),
			Success:         lo.ToPtr(false),
		},
	}

	createdProducts := rand.Int31()
	updatedProducts := rand.Int31()
	deletedProducts := rand.Int31()

	tests := map[string]struct {
		run           models.Run
		storedRuns    []pgmodels.Run
		wantRunsState []pgmodels.Run
		wantErr       bool
	}{
		"single run": {
			run: models.Run{
				ID:              1,
				ShopID:          shopID,
				CreatedAt:       createdAt,
				ProductsVersion: version,
				IsSuccess:       lo.ToPtr(true),
				FinishedAt:      &finishedAt,
				CreatedProducts: &createdProducts,
				UpdatedProducts: &updatedProducts,
				DeletedProducts: &deletedProducts,
			},
			storedRuns: runsState[0:1],
			wantRunsState: []pgmodels.Run{
				{
					ID:              1,
					ShopID:          int32(shopID),
					CreatedAt:       createdAt,
					ProductsVersion: version,
					Success:         lo.ToPtr(true),
					FinishedAt:      &finishedAt,
					CreatedProducts: &createdProducts,
					UpdatedProducts: &updatedProducts,
					DeletedProducts: &deletedProducts,
				},
			},
		},
		"many runs": {
			run: models.Run{
				ID:              1,
				ShopID:          shopID,
				CreatedAt:       createdAt,
				ProductsVersion: version,
				IsSuccess:       lo.ToPtr(true),
				FinishedAt:      &finishedAt,
				CreatedProducts: &createdProducts,
				UpdatedProducts: &updatedProducts,
				DeletedProducts: &deletedProducts,
			},
			storedRuns: runsState,
			wantRunsState: []pgmodels.Run{
				{
					ID:              1,
					ShopID:          int32(shopID),
					CreatedAt:       createdAt,
					ProductsVersion: version,
					Success:         lo.ToPtr(true),
					FinishedAt:      &finishedAt,
					CreatedProducts: &createdProducts,
					UpdatedProducts: &updatedProducts,
					DeletedProducts: &deletedProducts,
				},
				runsState[1],
				runsState[2],
			},
		},
		"not existing shop error": {
			run: models.Run{
				ID:              1,
				ShopID:          2, // ID of not existing shop
				CreatedAt:       createdAt,
				ProductsVersion: version,
				IsSuccess:       lo.ToPtr(true),
				FinishedAt:      &finishedAt,
				CreatedProducts: &createdProducts,
				UpdatedProducts: &updatedProducts,
				DeletedProducts: &deletedProducts,
			},
			storedRuns: runsState,
			wantErr:    true,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			defer storagetesting.CleanupData(s.T(), s.DB)

			storagetesting.InsertShops(s.T(), s.DB, pgmodels.Shop{ID: int32(shopID), URL: faker.Word()})
			storagetesting.InsertRuns(s.T(), s.DB, tt.storedRuns...)

			post := storage.NewPostgres(s.DB)

			err := post.FinishRun(context.TODO(), &tt.run)

			if tt.wantErr {
				s.Require().Error(err, "should return error")
			} else {
				s.Require().NoError(err, "shouldn't return any error")
				assertRuns(s.T(), tt.wantRunsState, storagetesting.GetRuns(s.T(), s.DB))
			}
		})
	}
}

func (s *PostgresTestSuite) TestIntegrationUpdateProducts() {
	storagetesting.CleanupData(s.T(), s.DB)
	version := rand.Int63()
	createdAt := time.Date(2024, time.April, 1, 1, 1, 1, 0, loc)
	deletedAt := time.Date(2024, time.April, 1, 2, 1, 1, 0, loc)
	shopID := int32(1)

	setProductData := func(product *models.Product) {
		product.CreatedAt = createdAt
		product.DeletedAt = nil
		product.Version = version
		product.Brand = nil
		product.GTIN = nil
		product.MPN = nil
		product.ProductCategory = nil
		product.ProductType = nil
		product.Color = nil
		product.Size = nil
		product.ItemGroupID = nil
		product.Gender = nil
		product.AgeGroup = nil
	}
	setProductID := func(id string) func(*models.Product) {
		return func(p *models.Product) {
			p.ProductID = id
		}
	}

	products := []models.Product{
		modelstesting.FakeProduct(setProductData, setProductID("1")),
		modelstesting.FakeProduct(setProductData, setProductID("2")),
		modelstesting.FakeProduct(setProductData, setProductID("3")),
		modelstesting.FakeProduct(setProductData, setProductID("4")),
		modelstesting.FakeProduct(setProductData, setProductID("5")),
	}

	tests := map[string]struct {
		storedProducts  []pgmodels.Product
		storedShippings []pgmodels.Product
		wantProducts    []models.Product
		wantCreated     int32
		wantUpdated     int32
		wantErr         bool
	}{
		"ok": {
			storedProducts: []pgmodels.Product{
				{
					ProductID: "1",
					ShopID:    shopID,
					Version:   version - 10,
					CreatedAt: createdAt,
				},
				{
					ProductID: "4",
					ShopID:    shopID,
					Version:   version - 10,
					CreatedAt: createdAt,
					DeletedAt: &deletedAt,
				},
			},
			wantProducts: products,
			wantCreated:  3,
			wantUpdated:  2,
		},
		"skip lower version": {
			storedProducts: []pgmodels.Product{
				{
					ProductID: "1",
					ShopID:    shopID,
					Version:   version + 10,
					CreatedAt: createdAt,
				},
				{
					ProductID: "4",
					ShopID:    shopID,
					Version:   version - 10,
					CreatedAt: createdAt,
					DeletedAt: &deletedAt,
				},
			},
			wantProducts: []models.Product{
				{
					ProductID: "1",
					Version:   version + 10,
					CreatedAt: createdAt,
				},
				products[1],
				products[2],
				products[3],
				products[4],
			},
			wantCreated: 3,
			wantUpdated: 1,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			defer storagetesting.CleanupData(s.T(), s.DB)

			storagetesting.InsertShops(s.T(), s.DB, pgmodels.Shop{ID: shopID, URL: faker.Word()})
			storagetesting.InsertProducts(s.T(), s.DB, tt.storedProducts...)

			post := storage.NewPostgres(s.DB)

			created, updated, err := post.UpdateProducts(context.TODO(), products, int(shopID))

			if tt.wantErr {
				s.Require().Error(err, "should return error")
			} else {
				s.Require().NoError(err, "shouldn't return any error")
				s.Equal(tt.wantCreated, created, "should return correct number of created products")
				s.Equal(tt.wantUpdated, updated, "should return correct number of updated products")
				assertProducts(s.T(), tt.wantProducts, storagetesting.GetProducts(s.T(), s.DB), int64(shopID))
				assertShippings(s.T(), tt.wantProducts, storagetesting.GetShippings(s.T(), s.DB))
			}
		})
	}
}

func (s *PostgresTestSuite) TestIntegrationDeleteOldProducts() {
	defer storagetesting.CleanupData(s.T(), s.DB)
	storagetesting.CleanupData(s.T(), s.DB)

	version := rand.Int63()
	createdAt := time.Date(2024, time.April, 1, 1, 1, 1, 0, loc)
	deletedAt := time.Date(2024, time.April, 1, 2, 1, 1, 0, loc)
	shopID := int32(1)

	storageState := []pgmodels.Product{
		{
			ProductID: "1",
			ShopID:    shopID,
			Version:   version - 10,
			CreatedAt: createdAt,
		},
		{
			ProductID: "2",
			ShopID:    shopID,
			Version:   version,
			CreatedAt: createdAt,
			DeletedAt: &deletedAt,
		},
		{
			ProductID: "3",
			ShopID:    shopID,
			Version:   version,
			CreatedAt: createdAt,
		},
		{
			ProductID: "4",
			ShopID:    shopID,
			Version:   version - 10,
			CreatedAt: createdAt,
			DeletedAt: &deletedAt,
		},
		{
			ProductID: "5",
			ShopID:    shopID,
			Version:   version - 10,
			CreatedAt: createdAt,
		},
	}

	wantState := []models.Product{
		{
			ProductID: "1",
			Version:   version - 10,
			CreatedAt: createdAt,
			DeletedAt: &deletedAt,
		},
		{
			ProductID: "2",
			Version:   version,
			CreatedAt: createdAt,
			DeletedAt: &deletedAt,
		},
		{
			ProductID: "3",
			Version:   version,
			CreatedAt: createdAt,
		},
		{
			ProductID: "4",
			Version:   version - 10,
			CreatedAt: createdAt,
			DeletedAt: &deletedAt,
		},
		{
			ProductID: "5",
			Version:   version - 10,
			DeletedAt: &deletedAt,
		},
	}

	storagetesting.InsertShops(s.T(), s.DB, pgmodels.Shop{ID: shopID, URL: faker.Word()})
	storagetesting.InsertProducts(s.T(), s.DB, storageState...)

	post := storage.NewPostgres(s.DB)

	deleted, err := post.DeleteOldProducts(context.TODO(), int(shopID), version, 1)

	s.Require().NoError(err, "shouldn't return any error")
	s.Equal(int32(2), deleted, "should return correct number of deleted products")
	state := storagetesting.GetProducts(s.T(), s.DB)
	lo.ForEach(state, func(_ pgmodels.Product, ix int) {
		if state[ix].DeletedAt != nil {
			state[ix].DeletedAt = &deletedAt
		}
	})
	assertProducts(s.T(), wantState, state, int64(shopID))
}

// assertProducts is a helper test function to assert products slice.
func assertProducts(t *testing.T, expected []models.Product, actual []pgmodels.Product, shopID int64) {
	t.Helper()

	require.Len(t, actual, len(expected), "products slice should have correct length")

	exp := make([]pgmodels.Product, 0, len(expected))
	for ix := range expected {
		exp = append(exp, *storage.ToDBProduct(&expected[ix], shopID, nil))
	}

	slices.SortFunc(exp, func(a, b pgmodels.Product) int { return strings.Compare(a.ProductID, b.ProductID) })
	slices.SortFunc(
		actual,
		func(a, b pgmodels.Product) int {
			return strings.Compare(a.ProductID, b.ProductID)
		},
	)
	lo.ForEach(actual, func(_ pgmodels.Product, ix int) {
		actual[ix].ID = 0
		actual[ix].CreatedAt = time.Time{}
		exp[ix].CreatedAt = time.Time{}
	})

	for ix := range actual {
		assert.EqualValues(t, exp[ix], actual[ix], "product at index %d has incorrect values", ix)
	}
}

// assertRuns is a helper test function to assert runs slice.
func assertRuns(t *testing.T, expected, actual []pgmodels.Run) {
	t.Helper()

	require.Len(t, actual, len(expected), "should have correct length")

	slices.SortFunc(expected, func(a, b pgmodels.Run) int { return int(b.ID - a.ID) })
	slices.SortFunc(actual, func(a, b pgmodels.Run) int { return int(b.ID - a.ID) })

	for ix := range expected {
		assert.Equalf(t, expected[ix], actual[ix], "run  at index %d has incorrect values", ix)
	}
}

// assertRun is a helper test function to assert run.
func assertRun(t *testing.T, expected, actual *models.Run) {
	t.Helper()

	if expected == nil {
		require.Nil(t, actual, "run should be nil")
		return
	}

	require.NotNil(t, actual, "run should not be nil")

	require.NotZero(t, actual.ShopID, "run should have new shop id")
	require.NotZero(t, actual.ID, "run should have id")
	require.NotZero(t, actual.CreatedAt.UnixMilli(), "run should have \"created at\" set")

	actual.CreatedAt = time.Time{}
	actual.ID = 0
	if expected.ShopID == 0 {
		actual.ShopID = 0
	}

	assert.Equal(t, *expected, *actual, "run has incorrect values")
}

// assertShippings is a helper test function to assert shippings slice.
func assertShippings(t *testing.T, expected []models.Product, actual []pgmodels.Shipping) {
	t.Helper()

	exp := []pgmodels.Shipping{}
	for ix := range expected {
		exp = append(exp, storage.ToDBShippings(0, expected[ix].Shippings)...)
	}

	require.Len(t, actual, len(exp), "shippings slice should have correct length")

	slices.SortFunc(exp, func(a, b pgmodels.Shipping) int { return strings.Compare(a.Country, b.Country) })
	slices.SortFunc(
		actual,
		func(a, b pgmodels.Shipping) int {
			return strings.Compare(a.Country, b.Country)
		},
	)
	lo.ForEach(actual, func(_ pgmodels.Shipping, ix int) {
		actual[ix].ID = 0
		actual[ix].ProductID = 0
	})

	for ix := range actual {
		assert.EqualValues(t, exp[ix], actual[ix], "shipping at index %d has incorrect values", ix)
	}
}
