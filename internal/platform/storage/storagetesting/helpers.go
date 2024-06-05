package storagetesting

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	pgmodels "github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/model"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/table"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"

	_ "github.com/lib/pq"
)

// Open opens connection to DB.
func Open(t *testing.T) *sql.DB {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("please provide database URL via DATABASE_URL environment variable")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("can't open connection to %q: %s", dbURL, err)
	}

	return db
}

// BeginTx begins DB transaction. Returns function to roll it back.
func BeginTx(t *testing.T, db *sql.DB) (*sql.Tx, func()) {
	t.Helper()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal("begin transaction", err)
	}

	rollback := func() {
		if err := tx.Rollback(); err != nil {
			t.Fatal("can't rollback transaction", err)
		}
	}

	return tx, rollback
}

// InsertShops is a helper test function to insert jobs.
func InsertShops(t *testing.T, exc qrm.Executable, shops ...pgmodels.Shop) {
	t.Helper()

	if len(shops) == 0 {
		return
	}

	toInsert := make([]pgmodels.Shop, 0, len(shops))
	toInsert = append(toInsert, shops...)

	_, err := table.Shop.INSERT(table.Shop.AllColumns).MODELS(toInsert).Exec(exc)
	if err != nil {
		t.Fatal("can't insert shops", err)
	}
}

// InsertRuns is a helper test function to insert runs.
func InsertRuns(t *testing.T, exc qrm.Executable, runs ...pgmodels.Run) {
	t.Helper()

	if len(runs) == 0 {
		return
	}

	toInsert := make([]pgmodels.Run, 0, len(runs))
	toInsert = append(toInsert, runs...)

	_, err := table.Run.INSERT(table.Run.AllColumns).MODELS(toInsert).Exec(exc)
	if err != nil {
		t.Fatal("can't insert runs", err)
	}
}

// InsertRuns is a helper test function to insert products.
func InsertProducts(t *testing.T, exc qrm.Executable, products ...pgmodels.Product) {
	t.Helper()

	if len(products) == 0 {
		return
	}

	toInsert := make([]pgmodels.Product, 0, len(products))
	toInsert = append(toInsert, products...)

	_, err := table.Product.INSERT(table.Product.AllColumns.Except(table.Product.ID)).MODELS(toInsert).Exec(exc)
	if err != nil {
		t.Fatal("can't insert products", err)
	}
}

// InsertShippings is a helper test function to insert shippings.
func InsertShippings(t *testing.T, exc qrm.Executable, shippings ...pgmodels.Shipping) {
	t.Helper()

	if len(shippings) == 0 {
		return
	}

	toInsert := make([]pgmodels.Shipping, 0, len(shippings))
	toInsert = append(toInsert, shippings...)

	_, err := table.Shipping.INSERT(table.Shipping.AllColumns).MODELS(toInsert).Exec(exc)
	if err != nil {
		t.Fatal("can't insert shippings", err)
	}
}

// GetRuns is a helper test function to get all runs.
func GetRuns(t *testing.T, queryable qrm.Queryable) []pgmodels.Run {
	t.Helper()

	runs := []pgmodels.Run{}
	err := table.Run.SELECT(table.Run.AllColumns).
		WHERE(table.Run.ID.IS_NOT_NULL()).
		Query(queryable, &runs)
	if err != nil {
		t.Fatal("can't get runs", err)
	}

	return runs
}

// GetProducts is a helper test function to get all products.
func GetProducts(t *testing.T, queryable qrm.Queryable) []pgmodels.Product {
	t.Helper()

	products := []pgmodels.Product{}
	err := table.Product.SELECT(table.Product.AllColumns).
		WHERE(table.Product.ID.IS_NOT_NULL()).
		Query(queryable, &products)
	if err != nil {
		t.Fatal("can't get products", err)
	}

	return products
}

// GetShippings is a helper test function to get all shippings.
func GetShippings(t *testing.T, queryable qrm.Queryable) []pgmodels.Shipping {
	t.Helper()

	shippings := []pgmodels.Shipping{}
	err := table.Shipping.SELECT(table.Shipping.AllColumns).
		WHERE(table.Shipping.ID.IS_NOT_NULL()).
		Query(queryable, &shippings)
	if err != nil {
		t.Fatal("can't get products", err)
	}

	return shippings
}

// GetShopID is a helper test function to get shop ID by shop URL.
func GetShopID(t *testing.T, queryable qrm.Queryable, shopURL string) int {
	t.Helper()

	var shop pgmodels.Shop
	err := table.Shop.SELECT(table.Shop.ID).
		WHERE(table.Shop.URL.EQ(pg.String(shopURL))).
		Query(queryable, &shop)

	if err != nil && !errors.Is(err, qrm.ErrNoRows) {
		t.Fatal("can't get shop ID", err)
	}

	return int(shop.ID)
}

// GetLatestRun is a helper test function to get latest run by shop ID.
func GetLatestRun(t *testing.T, queryable qrm.Queryable, shopID int) *models.Run {
	t.Helper()

	var runs []pgmodels.Run
	err := table.Run.SELECT(table.Run.AllColumns).
		WHERE(table.Run.ShopID.EQ(pg.Int32(int32(shopID)))).
		ORDER_BY(table.Run.CreatedAt.DESC()).
		LIMIT(1).
		Query(queryable, &runs)

	if err != nil || len(runs) == 0 {
		t.Fatal("can't get shop ID", err)
	}

	return &models.Run{
		ID:              int(runs[0].ID),
		ShopID:          int(runs[0].ShopID),
		CreatedAt:       runs[0].CreatedAt,
		FinishedAt:      runs[0].FinishedAt,
		IsSuccess:       runs[0].Success,
		StatusMessage:   runs[0].StatusMessage,
		CreatedProducts: runs[0].CreatedProducts,
		UpdatedProducts: runs[0].UpdatedProducts,
		DeletedProducts: runs[0].DeletedProducts,
		FailedProducts:  runs[0].FailedProducts,
		ProductsVersion: runs[0].ProductsVersion,
	}
}

// GetProductsByShopID is a helper test function to get products by shop ID.
func GetProductsByShopID(t *testing.T, queryable qrm.Queryable, shopID int) []pgmodels.Product {
	t.Helper()

	products := []pgmodels.Product{}
	err := table.Product.SELECT(table.Product.AllColumns).
		WHERE(pg.AND(
			table.Product.ID.IS_NOT_NULL(),
			table.Product.ShopID.EQ(pg.Int32(int32(shopID))),
		)).
		Query(queryable, &products)
	if err != nil {
		t.Fatal("can't get products", err)
	}

	return products
}

// GetShippingByProductID is a helper test function to get shipping by product ID.
func GetShippingByProductID(t *testing.T, queryable qrm.Queryable, productID int) []pgmodels.Shipping {
	t.Helper()

	shippings := []pgmodels.Shipping{}
	err := table.Shipping.SELECT(table.Shipping.AllColumns).
		WHERE(pg.AND(
			table.Shipping.ID.IS_NOT_NULL(),
			table.Shipping.ProductID.EQ(pg.Int32(int32(productID))),
		)).
		Query(queryable, &shippings)
	if err != nil {
		t.Fatal("can't get shippings", err)
	}

	return shippings
}

// InsertShops is a helper test function to insert jobs.
func CleanupData(t *testing.T, exc qrm.Executable) {
	t.Helper()

	_, err := table.Shipping.DELETE().WHERE(table.Shipping.ID.IS_NOT_NULL()).Exec(exc)
	if err != nil {
		t.Fatal("can't delete shippings data", err)
	}

	_, err = table.Product.DELETE().WHERE(table.Product.ID.IS_NOT_NULL()).Exec(exc)
	if err != nil {
		t.Fatal("can't delete products data", err)
	}

	_, err = table.Run.DELETE().WHERE(table.Run.ID.IS_NOT_NULL()).Exec(exc)
	if err != nil {
		t.Fatal("can't delete runs data", err)
	}

	_, err = table.Shop.DELETE().WHERE(table.Shop.ID.IS_NOT_NULL()).Exec(exc)
	if err != nil {
		t.Fatal("can't delete shops data", err)
	}
}
