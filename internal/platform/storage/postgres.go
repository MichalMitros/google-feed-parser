package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/MichalMitros/google-feed-parser/internal/platform"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/table"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	pgmodels "github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/model"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

// Postgres is storage for shops, runs, products and shippings.
type Postgres struct {
	db            *sql.DB
	parallelLimit int
}

// NewPostgres returns new Postgres.
func NewPostgres(db *sql.DB) Postgres {
	return Postgres{
		db:            db,
		parallelLimit: 5,
	}
}

// StartRun creates new unfinished run in database and returns it.
// It returns ErrAlreadyRunning if previous run is not finished yet.
func (p Postgres) StartRun(ctx context.Context, shopURL string, version int64) (*models.Run, error) {
	run := &models.Run{
		ProductsVersion: version,
	}

	err := runInTransaction(ctx, p.db, func(tx *sql.Tx) error {
		shop, err := getShop(ctx, tx, shopURL)
		if err != nil {
			return fmt.Errorf("can't get shop from database: %w", err)
		}

		run.ShopID = int(shop.ID)

		lastRun, err := getLastRun(ctx, tx, int64(shop.ID))

		if err != nil && !errors.Is(err, qrm.ErrNoRows) {
			return fmt.Errorf("can't get last run from database: %w", err)
		}

		if lastRun != nil && lastRun.FinishedAt == nil && lastRun.Success == nil {
			return platform.ErrAlreadyRunning
		}

		newRun := toDBRun(run)
		err = table.Run.INSERT(
			table.Run.ProductsVersion,
			table.Run.ShopID,
		).
			MODEL(newRun).
			RETURNING(table.Run.ID).
			QueryContext(ctx, tx, newRun)
		if err != nil {
			return fmt.Errorf("can't insert run into database: %w", err)
		}

		run.ID = int(newRun.ID)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("can't add run: %w", err)
	}

	return run, nil
}

// FinishRun sets run as finished and updates run's statistics.
func (p Postgres) FinishRun(ctx context.Context, run *models.Run) error {
	columnList := table.Run.AllColumns.Except(table.Run.ID, table.Run.CreatedAt, table.Run.ProductsVersion)

	result, err := table.Run.UPDATE(columnList).
		MODEL(toDBRun(run)).
		WHERE(table.Run.ID.EQ(pg.Int32(int32(run.ID)))).
		ExecContext(ctx, p.db)
	if err != nil {
		return fmt.Errorf("can't update run: %w", err)
	}

	if rowsAffected, err := result.RowsAffected(); rowsAffected == 0 || err != nil {
		return fmt.Errorf("can't update run: %w", err)
	}

	return nil
}

// Update products upserts products and their shippings.
// It returns number of new products and number of updated products or error.
func (p Postgres) UpdateProducts(ctx context.Context, products []models.Product, shopID int) (int32, int32, error) {
	createdProductsNumber := lo.ToPtr(int32(0))
	updatedProductsNumber := lo.ToPtr(int32(0))

	err := runInTransaction(ctx, p.db, func(tx *sql.Tx) error {
		productIDs := lo.Map(products, func(_ models.Product, ix int) string {
			return products[ix].ProductID
		})
		storedVersions, err := getProductsVersions(ctx, tx, int64(shopID), productIDs)
		if err != nil {
			return fmt.Errorf("can't get existing products: %w", err)
		}

		newProducts, updatedProducts := compareProducts(products, storedVersions)

		if newProducts, err = upsertProducts(ctx, tx, newProducts, int32(shopID)); err != nil {
			return fmt.Errorf("can't insert new products: %w", err)
		}

		if updatedProducts, err = upsertProducts(ctx, tx, updatedProducts, int32(shopID)); err != nil {
			return fmt.Errorf("can't insert new products: %w", err)
		}

		if err = insertShippings(ctx, tx, newProducts); err != nil {
			return fmt.Errorf("can't create new products shippings: %w", err)
		}

		if err = insertShippings(ctx, tx, updatedProducts); err != nil {
			return fmt.Errorf("can't update products shippings: %w", err)
		}

		*createdProductsNumber = int32(len(newProducts))
		*updatedProductsNumber = int32(len(updatedProducts))

		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return *createdProductsNumber, *updatedProductsNumber, nil
}

// DeleteOldProducts updates DeletedAt field of shop products with version lower than provided.
// Returns number of deleted products or error.
func (p Postgres) DeleteOldProducts(ctx context.Context, shopID int, version int64, batchSize uint) (int32, error) {
	deletedProductsNumber := int32(0)

	err := runInTransaction(ctx, p.db, func(tx *sql.Tx) error {
		toDelete := make(chan []int32)

		errGroup, egCtx := errgroup.WithContext(ctx)

		errGroup.Go(func() error {
			return getOutdatedProductsAsync(egCtx, p.db, int32(shopID), version, batchSize, toDelete)
		})

		errGroup.Go(func() error {
			deletedCount, err := deleteProductsAsync(egCtx, p.db, toDelete)
			if err == nil {
				atomic.AddInt32(&deletedProductsNumber, int32(deletedCount))
			}
			return err
		})

		return errGroup.Wait()
	})
	if err != nil {
		return 0, err
	}

	return deletedProductsNumber, nil
}

func compareProducts(parsed []models.Product, storedVersions map[string]int64) ([]models.Product, []models.Product) {
	newProducts := make([]models.Product, 0, len(parsed))
	updatedProducts := lo.Filter(parsed, func(_ models.Product, ix int) bool {
		if version, ok := storedVersions[parsed[ix].ProductID]; ok {
			return parsed[ix].Version > version
		}
		newProducts = append(newProducts, parsed[ix])
		return false
	})

	return newProducts, updatedProducts
}

func upsertProducts(ctx context.Context, db qrm.DB, products []models.Product, shopID int32) ([]models.Product, error) {
	if len(products) == 0 {
		return nil, nil
	}

	columnList := table.Product.AllColumns.Except(table.Product.ID, table.Product.CreatedAt)

	dbProducts := make([]pgmodels.Product, 0, len(products))
	for ix := range products {
		dbProducts = append(dbProducts, *ToDBProduct(&products[ix], int64(shopID), nil))
	}

	excludedExpressions := make([]pg.Expression, 0, len(columnList)) // converting to expression
	for _, col := range table.Product.EXCLUDED.AllColumns.Except(table.Product.ID, table.Product.CreatedAt) {
		excludedExpressions = append(excludedExpressions, col)
	}

	_, err := table.Product.INSERT(columnList).
		MODELS(dbProducts).
		ON_CONFLICT(table.Product.ShopID, table.Product.ProductID).
		DO_UPDATE(
			pg.SET(
				columnList.SET(pg.ROW(excludedExpressions...)),
			),
		).
		ExecContext(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("can't upsert products into database: %w", err)
	}

	ids := make([]pg.Expression, 0, len(products))
	upsertedIDs := make([]pgmodels.Product, 0, len(products))
	for ix := range products {
		ids = append(ids, pg.String(products[ix].ProductID))
	}
	err = table.Product.SELECT(table.Product.ID, table.Product.ProductID).
		WHERE(pg.AND(
			table.Product.ShopID.EQ(pg.Int32(int32(shopID))),
			table.Product.ProductID.IN(ids...),
		)).
		QueryContext(ctx, db, &upsertedIDs)
	if err != nil {
		return nil, fmt.Errorf("can't get upserted products ids: %w", err)
	}

	upsertedProducts := make([]models.Product, 0, len(products))
	lo.ForEach(products, func(_ models.Product, i int) {
		for ix := range upsertedIDs {
			if upsertedIDs[ix].ProductID == products[i].ProductID {
				upsertedProducts = append(upsertedProducts, products[i])
				upsertedProducts[len(upsertedProducts)-1].ID = int(upsertedIDs[ix].ID)
				return
			}
		}
	})

	return upsertedProducts, nil
}

func insertShippings(ctx context.Context, db qrm.DB, products []models.Product) error {
	shippings := []pgmodels.Shipping{}
	for ix := range products {
		shippings = append(shippings, ToDBShippings(int32(products[ix].ID), products[ix].Shippings)...)
	}
	if len(shippings) == 0 {
		return nil
	}

	ids := make([]pg.Expression, 0, len(products))
	for ix := range products {
		ids = append(ids, pg.Int32(int32(products[ix].ID)))
	}

	_, err := table.Shipping.DELETE().
		WHERE(table.Shipping.ProductID.IN(ids...)).
		ExecContext(ctx, db)
	if err != nil {
		return fmt.Errorf("can't delete outdated products shippings from database: %w", err)
	}

	_, err = table.Shipping.INSERT(table.Shipping.AllColumns.Except(table.Shipping.ID)).
		MODELS(shippings).
		ExecContext(ctx, db)
	if err != nil {
		return fmt.Errorf("can't insert shippings into database: %w", err)
	}

	return nil
}

func getShop(ctx context.Context, db qrm.DB, url string) (*pgmodels.Shop, error) {
	var shop pgmodels.Shop
	err := table.Shop.SELECT(table.Shop.AllColumns).
		WHERE(table.Shop.URL.EQ(pg.String(url))).
		QueryContext(ctx, db, &shop)

	if errors.Is(err, qrm.ErrNoRows) {
		return insertShop(ctx, db, url)
	}

	if err != nil {
		return nil, err
	}

	return &shop, nil
}

func insertShop(ctx context.Context, db qrm.DB, url string) (*pgmodels.Shop, error) {
	shop := pgmodels.Shop{
		URL: url,
	}
	_, err := table.Shop.INSERT(table.Shop.URL).
		MODEL(pgmodels.Shop{
			URL: url,
		}).
		ExecContext(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("can't add shop: %w", err)
	}

	err = table.Shop.SELECT(table.Shop.AllColumns).
		WHERE(table.Shop.URL.EQ(pg.String(url))).
		QueryContext(ctx, db, &shop)
	if err != nil {
		return nil, fmt.Errorf("can't get added shop: %w", err)
	}

	return &shop, nil
}

func getLastRun(ctx context.Context, db qrm.DB, shopID int64) (*pgmodels.Run, error) {
	var run pgmodels.Run
	err := table.Run.SELECT(
		table.Run.CreatedAt,
		table.Run.FinishedAt,
		table.Run.Success,
		table.Run.StatusMessage,
		table.Run.FailedProducts,
	).
		WHERE(table.Run.ShopID.EQ(pg.Int(shopID))).
		ORDER_BY(table.Run.CreatedAt.DESC()).
		LIMIT(1).
		QueryContext(ctx, db, &run)
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func getOutdatedProductsAsync(
	ctx context.Context,
	db qrm.DB,
	shopID int32,
	version int64,
	batchSize uint,
	toDelete chan []int32,
) error {
	defer close(toDelete)
	previousID := int32(0)
	for {
		var products []pgmodels.Product
		err := table.Product.SELECT(table.Product.ID, table.Product.Version).
			WHERE(pg.AND(
				table.Product.ShopID.EQ(pg.Int32(int32(shopID))),
				table.Product.Version.LT(pg.Int64(version)),
				table.Product.DeletedAt.IS_NULL(),
				table.Product.ID.GT(pg.Int64(int64(previousID))),
			)).
			ORDER_BY(table.Product.ID.ASC()).
			LIMIT(int64(batchSize)).
			QueryContext(ctx, db, &products)

		if errors.Is(err, qrm.ErrNoRows) || len(products) == 0 {
			return nil
		}

		if err != nil && !errors.Is(err, qrm.ErrNoRows) {
			return err
		}

		ids := make([]int32, 0, len(products))
		for ix := range products {
			ids = append(ids, products[ix].ID)
		}

		previousID = products[len(products)-1].ID

		select {
		case <-ctx.Done():
			return ctx.Err()
		case toDelete <- ids:
		}
	}
}

func deleteProductsAsync(ctx context.Context, db qrm.DB, toDelete chan []int32) (int, error) {
	deletedCount := 0
	now := time.Now()
	for batch := range toDelete {
		ids := make([]pg.Expression, 0, len(batch))
		for _, id := range batch {
			ids = append(ids, pg.Int32(id))
		}

		_, err := table.Product.UPDATE().
			SET(
				table.Product.DeletedAt.SET(pg.TimestampzT(now)),
			).
			WHERE(table.Product.ID.IN(ids...)).
			ExecContext(ctx, db)
		if err != nil {
			return deletedCount, err
		}
		deletedCount += len(batch)
	}
	return deletedCount, nil
}

func getProductsVersions(ctx context.Context, db qrm.DB, shopID int64, productIDs []string) (map[string]int64, error) {
	ids := make([]pg.Expression, 0, len(productIDs))
	for ix := range productIDs {
		ids = append(ids, pg.String(productIDs[ix]))
	}

	products := make([]pgmodels.Product, 0, len(productIDs))
	err := table.Product.SELECT(table.Product.AllColumns).
		WHERE(pg.AND(
			table.Product.ShopID.EQ(pg.Int32(int32(shopID))),
			table.Product.ProductID.IN(ids...),
		)).
		QueryContext(ctx, db, &products)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(products))
	for ix := range products {
		result[products[ix].ProductID] = products[ix].Version
	}

	return result, nil
}

func runInTransaction(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	var (
		tx  *sql.Tx
		err error
	)

	if tx, err = db.BeginTx(ctx, nil); err != nil {
		return fmt.Errorf("can't begin transaction: %w", err)
	}

	if err = fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("can't rollback transaction: %w (rollback reason: %w)", rbErr, err)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit transaction: %w", err)
	}

	return nil
}
