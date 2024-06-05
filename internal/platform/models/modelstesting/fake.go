package modelstesting

import (
	"math/rand"

	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/go-faker/faker/v4"
	"github.com/samber/lo"
)

// FakeProduct returns models.Product with fake data and random number of fake shippings.
func FakeProduct(ops ...func(p *models.Product)) models.Product {
	product := models.Product{
		Version:             rand.Int63(),
		ProductID:           faker.Word(),
		Title:               faker.Word(),
		Description:         faker.Word(),
		URL:                 faker.Word(),
		ImageURL:            faker.Word(),
		AdditionalImageURLs: fakeAdditionalImageURLs(),
		Condition:           faker.Word(),
		Availability:        faker.Word(),
		Price:               faker.Word(),
		Shippings:           fakeShippings(),
		Brand:               lo.ToPtr(faker.Word()),
		GTIN:                lo.ToPtr(faker.Word()),
		MPN:                 lo.ToPtr(faker.Word()),
		ProductCategory:     lo.ToPtr(faker.Word()),
		ProductType:         lo.ToPtr(faker.Word()),
		Color:               lo.ToPtr(faker.Word()),
		Size:                lo.ToPtr(faker.Word()),
		ItemGroupID:         lo.ToPtr(faker.Word()),
		Gender:              lo.ToPtr(faker.Word()),
		AgeGroup:            lo.ToPtr(faker.Word()),
	}

	for _, op := range ops {
		op(&product)
	}

	return product
}

// FakeShipping returns models.Shipping with fake data.
func FakeShipping(ops ...func(s *models.Shipping)) models.Shipping {
	shipping := models.Shipping{
		Country: faker.Word(),
		Service: faker.Word(),
		Price:   faker.Word(),
	}

	for _, op := range ops {
		op(&shipping)
	}

	return shipping
}

func fakeAdditionalImageURLs() []string {
	additionalImgURLsLen := rand.Intn(5)
	additionalImgURLs := make([]string, 0, additionalImgURLsLen)
	for range additionalImgURLsLen {
		additionalImgURLs = append(additionalImgURLs, faker.Word())
	}

	return additionalImgURLs
}

func fakeShippings() []models.Shipping {
	shippingsLen := rand.Intn(5)
	shippings := make([]models.Shipping, 0, shippingsLen)
	for range shippingsLen {
		shippings = append(shippings, FakeShipping())
	}

	return shippings
}
