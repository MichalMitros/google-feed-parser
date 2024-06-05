package storage

import (
	"fmt"
	"strings"

	"github.com/MichalMitros/google-feed-parser/internal/platform/models"

	pgmodels "github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/model"
)

//go:generate make -C ../../../ generate-db

func toDBRun(run *models.Run) *pgmodels.Run {
	return &pgmodels.Run{
		ProductsVersion: run.ProductsVersion,
		ShopID:          int32(run.ShopID),
		FinishedAt:      run.FinishedAt,
		Success:         run.IsSuccess,
		StatusMessage:   run.StatusMessage,
		CreatedProducts: run.CreatedProducts,
		UpdatedProducts: run.UpdatedProducts,
		DeletedProducts: run.DeletedProducts,
		FailedProducts:  run.FailedProducts,
	}
}

// ToDBProduct converts models.Product into postgres product model.
func ToDBProduct(product *models.Product, shopID int64, id *int32) *pgmodels.Product {
	dbProduct := pgmodels.Product{
		Version:           product.Version,
		ShopID:            int32(shopID),
		ProductID:         product.ProductID,
		Title:             product.Title,
		Description:       product.Description,
		URL:               product.URL,
		ImgURL:            product.ImageURL,
		AdditionalImgUrls: toDBAdditionalImageURLs(product.AdditionalImageURLs),
		Condition:         product.Condition,
		Availability:      product.Availability,
		Price:             product.Price,
		Brand:             product.Brand,
		Gtin:              product.GTIN,
		Mpn:               product.MPN,
		ProductCategory:   product.ProductCategory,
		ProductType:       product.ProductType,
		Color:             product.Color,
		Size:              product.Size,
		ItemGroupID:       product.ItemGroupID,
		Gender:            product.Gender,
		AgeGroup:          product.AgeGroup,
		DeletedAt:         product.DeletedAt,
	}

	if id != nil {
		dbProduct.ID = int32(*id)
	}

	return &dbProduct
}

// ToDBShippings converts models.Shipping slice into postgres shipping slice.
func ToDBShippings(productID int32, shippings []models.Shipping) []pgmodels.Shipping {
	if len(shippings) == 0 {
		return []pgmodels.Shipping{}
	}

	dbShipping := make([]pgmodels.Shipping, 0, len(shippings))
	for ix := range shippings {
		dbShipping = append(dbShipping, pgmodels.Shipping{
			ProductID: productID,
			Country:   shippings[ix].Country,
			Service:   shippings[ix].Service,
			Price:     shippings[ix].Price,
		})
	}
	return dbShipping
}

func toDBAdditionalImageURLs(urls []string) string {
	if len(urls) == 0 {
		return ""
	}

	result := strings.Builder{}
	for ix, url := range urls {
		if ix == len(urls)-1 {
			result.WriteString(url)
			break
		}
		result.WriteString(fmt.Sprintf("%s\n", url))
	}
	return result.String()
}
