package testdata

import (
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/samber/lo"
)

var Products = []models.Product{
	{
		ProductID:    "TV_123456",
		Title:        `LG 22LB4510 - 22" LED TV - 1080p (FullHD)`,
		Description:  `Attractively styled and boasting stunning picture quality, the LG 22LB4510 - 22" LED TV - 1080p (FullHD) is an excellent television/monitor. The LG 22LB4510 - 22" LED TV - 1080p (FullHD) sports a widescreen 1080p panel, perfect for watching movies in their original format, whilst also providing plenty of working space for your other applications.`,
		URL:          "http://www.example.com/electronics/tv/22LB4510.html",
		ImageURL:     "http://images.example.com/TV_123456.png",
		Condition:    "used",
		Availability: "in stock",
		Price:        "159.00 USD",
		Shippings: []models.Shipping{
			{
				Country: "US",
				Service: "Standard",
				Price:   "14.95 USD",
			},
		},
		GTIN:            lo.ToPtr("71919219405200"),
		Brand:           lo.ToPtr("LG"),
		MPN:             lo.ToPtr("22LB4510/US"),
		ProductCategory: lo.ToPtr("Electronics > Video > Televisions > Flat Panel Televisions"),
		ProductType:     lo.ToPtr("Consumer Electronics > TVs > Flat Panel TVs"),
	},
	{
		ProductID:    "DVD-0564738",
		Title:        `Merlin: Series 3 - Volume 2 - 3 DVD Box set`,
		Description:  `Episodes 7-13 from the third series of the BBC fantasy drama set in the mythical city of Camelot, telling the tale of the relationship between the young King Arthur (Bradley James) & Merlin (Colin Morgan), the wise sorcerer who guides him to power and beyond. Episodes are: 'The Castle of Fyrien', 'The Eye of the Phoenix', 'Love in the Time of Dragons', 'Queen of Hearts', 'The Sorcerer's Shadow', 'The Coming of Arthur: Part 1' & 'The Coming of Arthur: Part 2'`,
		URL:          "http://www.example.com/media/dvd/?sku=384616&src=gshopping&lang=en",
		ImageURL:     "http://images.example.com/DVD-0564738?size=large&format=PNG",
		Condition:    "new",
		Availability: "in stock",
		Price:        "11.99 USD",
		Shippings: []models.Shipping{
			{
				Country: "US",
				Service: "Express Mail",
				Price:   "3.80 USD",
			},
		},
		GTIN:            lo.ToPtr("88392916560500"),
		Brand:           lo.ToPtr("BBC"),
		ProductCategory: lo.ToPtr("Media > DVDs & Videos"),
		ProductType:     lo.ToPtr("DVDs & Movies > TV Series > Fantasy Drama"),
	},
	{
		ProductID:   "PFM654321",
		Title:       `Dior Capture XP Ultimate Wrinkle Correction Creme 1.7 oz`,
		Description: `Dior Capture XP Ultimate Wrinkle Correction Creme 1.7 oz reinvents anti-wrinkle care by protecting and relaunching skin cell activity to encourage faster, healthier regeneration.`,
		URL:         "http://www.example.com/perfumes/product?Dior%20Capture%20R6080%20XP",
		ImageURL:    "http://images.example.com/PFM654321_1.jpg",
		AdditionalImageURLs: []string{
			"http://images.example.com/PFM654321_2.jpg",
			"http://images.example.com/PFM654321_3.jpg",
		},
		Condition:    "new",
		Availability: "in stock",
		Price:        "99 USD",
		Shippings: []models.Shipping{
			{
				Country: "US",
				Service: "Standard Rate",
				Price:   "4.95 USD",
			},
			{
				Country: "US",
				Service: "Next Day",
				Price:   "8.50 USD",
			},
		},
		GTIN:            lo.ToPtr("3348901056069"),
		Brand:           lo.ToPtr("Dior"),
		ProductCategory: lo.ToPtr("Health & Beauty > Personal Care > Cosmetics > Skin Care > Anti-Aging Skin Care Kits"),
		ProductType:     lo.ToPtr("Health & Beauty > Personal Care > Cosmetics > Skin Care > Lotion"),
	},
	{
		ProductID:   "CLO-29473856-1",
		Title:       `Roma Cotton Rich Bootcut Jeans - Size 8 Standard`,
		Description: `A smart pair of bootcut jeans in stretch cotton.`,
		URL:         "http://www.example.com/clothing/women/Roma-Cotton-Bootcut-Jeans/?extid=CLO-29473856",
		ImageURL:    "http://images.example.com/CLO-29473856-front.jpg",
		AdditionalImageURLs: []string{
			"http://images.example.com/CLO-29473856-side.jpg",
			"http://images.example.com/CLO-29473856-back.jpg",
		},
		Condition:       "new",
		Availability:    "out of stock",
		Price:           "29.50 USD",
		Brand:           lo.ToPtr("M&S"),
		Gender:          lo.ToPtr("Female"),
		AgeGroup:        lo.ToPtr("Adult"),
		Color:           lo.ToPtr("Navy"),
		Size:            lo.ToPtr("8 Standard"),
		ItemGroupID:     lo.ToPtr("CLO-29473856"),
		MPN:             lo.ToPtr("B003J5F5EY"),
		ProductCategory: lo.ToPtr("Apparel & Accessories > Clothing > Pants > Jeans"),
		ProductType:     lo.ToPtr("Women's Clothing > Jeans > Bootcut Jeans"),
	},
}
