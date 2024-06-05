package models

import "time"

// ParsingResult contains product with parsing error if there is any.
type ParsingResult struct {
	Product Product
	Error   error
}

// Shop is shop model.
type Shop struct {
	ID        int
	Name      string
	URL       string
	CreatedAt time.Time
	DeletedAt *time.Time

	LastRuns []Run
}

// Run is parsing process run model.
type Run struct {
	ID              int
	ShopID          int
	CreatedAt       time.Time
	FinishedAt      *time.Time
	IsSuccess       *bool
	StatusMessage   *string
	CreatedProducts *int32
	UpdatedProducts *int32
	DeletedProducts *int32
	FailedProducts  *int32
	ProductsVersion int64
}

// Product is product model.
type Product struct {
	ID                  int
	Version             int64
	CreatedAt           time.Time
	DeletedAt           *time.Time
	ProductID           string
	Title               string
	Description         string
	URL                 string
	ImageURL            string
	AdditionalImageURLs []string
	Condition           string
	Availability        string
	Price               string
	Shippings           []Shipping
	Brand               *string
	GTIN                *string
	MPN                 *string
	ProductCategory     *string
	ProductType         *string
	Color               *string
	Size                *string
	ItemGroupID         *string
	Gender              *string
	AgeGroup            *string
}

// Shipping is product's shipping model.
type Shipping struct {
	Country string
	Service string
	Price   string
}
