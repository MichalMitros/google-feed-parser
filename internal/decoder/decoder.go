package decoder

import (
	"context"
	"encoding/xml"
	"errors"
	"html"
	"io"

	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/samber/lo"
)

// Decoder decodes xml files into products.
type Decoder struct{}

// Decode decodes products from xmlFile and returns each file with decoding error into output channel.
func (d Decoder) Decode(ctx context.Context, xmlFile io.Reader, output chan<- models.ParsingResult) error {
	dec := xml.NewDecoder(xmlFile)
	dec.Strict = true

	for {
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local != "item" {
				continue
			}
			var product Product
			err = dec.DecodeElement(&product, &element)

			unescapeProductFields(&product)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case output <- models.ParsingResult{
				Product: *toAppProduct(&product),
				Error:   err,
			}:
			}
		default:
			continue
		}
	}
}

// unescapeProductFields unescapes html characters from product title, description, category and type.
func unescapeProductFields(product *Product) {
	product.Title = html.UnescapeString(product.Title)
	product.Description = html.UnescapeString(product.Description)
	if product.ProductCategory != nil {
		product.ProductCategory = lo.ToPtr(html.UnescapeString(*product.ProductCategory))
	}
	if product.ProductType != nil {
		product.ProductType = lo.ToPtr(html.UnescapeString(*product.ProductType))
	}
}
