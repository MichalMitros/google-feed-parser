package decoder

import "github.com/MichalMitros/google-feed-parser/internal/platform/models"

// Product is model for product items in feed files.
type Product struct {
	ID                  string     `xml:"id" ,json:"id"`
	Title               string     `xml:"title" ,json:"title"`
	Description         string     `xml:"description" ,json:"description"`
	URL                 string     `xml:"link" ,json:"link"`
	ImageURL            string     `xml:"image_link" ,json:"imageUrl"`
	AdditionalImageURLs []string   `xml:"additional_image_link" ,json:"additionalImageUrls"`
	Condition           string     `xml:"condition" ,json:"condition"`
	Availability        string     `xml:"availability" ,json:"availability"`
	Price               string     `xml:"price" ,json:"price"`
	Shippings           []Shipping `xml:"shipping" ,json:"shipping"`
	Brand               *string    `xml:"brand" ,json:"brand"`
	GTIN                *string    `xml:"gtin" ,json:"gtin"`
	MPN                 *string    `xml:"mpn" ,json:"mpn"`
	ProductCategory     *string    `xml:"google_product_category" ,json:"productCategory"`
	ProductType         *string    `xml:"product_type" ,json:"productType"`
	Color               *string    `xml:"color" ,json:"color"`
	Size                *string    `xml:"size" ,json:"size"`
	ItemGroupID         *string    `xml:"item_group_id" ,json:"itemGroupID"`
	Gender              *string    `xml:"gender" ,json:"gender"`
	AgeGroup            *string    `xml:"age_group" ,json:"ageGroup"`
}

// Shipping is model for product items shippings in feed files.
type Shipping struct {
	Country string `xml:"country" ,json:"country"`
	Service string `xml:"service" ,json:"service"`
	Price   string `xml:"price" ,json:"price"`
}

func toAppProduct(product *Product) *models.Product {
	return &models.Product{
		ProductID:           product.ID,
		Title:               product.Title,
		Description:         product.Description,
		URL:                 product.URL,
		ImageURL:            product.ImageURL,
		AdditionalImageURLs: product.AdditionalImageURLs,
		Condition:           product.Condition,
		Availability:        product.Availability,
		Price:               product.Price,
		Shippings:           toAppShippings(product.Shippings),
		Brand:               product.Brand,
		GTIN:                product.GTIN,
		MPN:                 product.MPN,
		ProductCategory:     product.ProductCategory,
		ProductType:         product.ProductType,
		Color:               product.Color,
		Size:                product.Size,
		ItemGroupID:         product.ItemGroupID,
		Gender:              product.Gender,
		AgeGroup:            product.AgeGroup,
	}
}

func toAppShippings(shippings []Shipping) []models.Shipping {
	if len(shippings) == 0 {
		return nil
	}
	appShippigs := make([]models.Shipping, 0, len(shippings))
	for ix := range shippings {
		appShippigs = append(appShippigs, *toAppShipping(&shippings[ix]))
	}
	return appShippigs
}

func toAppShipping(shipping *Shipping) *models.Shipping {
	return &models.Shipping{
		Country: shipping.Country,
		Service: shipping.Service,
		Price:   shipping.Price,
	}
}
