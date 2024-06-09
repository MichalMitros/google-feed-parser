package helpers

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MichalMitros/google-feed-parser/internal/decoder"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models/modelstesting"
	pgmodels "github.com/MichalMitros/google-feed-parser/internal/platform/storage/gen/postgres/public/model"
	"github.com/MichalMitros/google-feed-parser/internal/platform/storage/storagetesting"
	"github.com/go-jet/jet/v2/qrm"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "Content-Type"
)

// WaitForRunToBeFinished is blocking helper function, returns latest run after it is finished.
func WaitForRunToBeFinished(t *testing.T, queryable qrm.Queryable, shopURL string) *models.Run {
	t.Helper()

	var shopID int
	for {
		<-time.After(time.Millisecond * 250)
		shopID = storagetesting.GetShopID(t, queryable, shopURL)
		if shopID != 0 {
			break
		}
	}

	var latestRun *models.Run
	for {
		<-time.After(time.Millisecond * 500)
		latestRun = storagetesting.GetLatestRun(t, queryable, shopID)
		if latestRun != nil && latestRun.FinishedAt != nil {
			return latestRun
		}
	}
}

// GetProducts is helper function for getting products from db ordered by ProductID (must be integer).
func GetProducts(t *testing.T, queryable qrm.Queryable, shopURL string) []models.Product {
	t.Helper()

	shopID := storagetesting.GetShopID(t, queryable, shopURL)
	dbProducts := storagetesting.GetProductsByShopID(t, queryable, shopID)

	products := make([]models.Product, len(dbProducts))
	for ix := range dbProducts {
		products[ix] = *fromDBProduct(
			&dbProducts[ix],
			storagetesting.GetShippingByProductID(t, queryable, int(dbProducts[ix].ID)),
		)
	}

	sort.Slice(products, func(i, j int) bool {
		var aID, bID int
		var err error
		if aID, err = strconv.Atoi(products[i].ProductID); err != nil {
			require.FailNow(t, "expected productID should be integer")
		}
		if bID, err = strconv.Atoi(products[j].ProductID); err != nil {
			require.FailNow(t, "expected productID should be integer")
		}
		return aID < bID
	})

	return products
}

// PrepareMockedHTTPServer is helper function for mocking http srv and client.
// Returns function for setting feed file to return, feed number is from 0 to len(feedFiles) inclusive.
func PrepareMockedHTTPServer(t *testing.T, feedFiles [][]byte, statusCode int) (*httptest.Server, func(int)) {
	t.Helper()

	feedFileToReturnIx := 0

	srv := httptest.NewServer(http.HandlerFunc(func(wrt http.ResponseWriter, req *http.Request) {
		wrt.Header().Add(contentType, "application/xml")
		wrt.WriteHeader(statusCode)
		_, _ = wrt.Write(feedFiles[feedFileToReturnIx])
	}))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv, func(i int) { feedFileToReturnIx = i }
}

// DeclareRMQExchange is helper function for declaring RMQ exchange.
func DeclareRMQExchange(t *testing.T, ch *amqp.Channel, exchange string) {
	t.Helper()

	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		require.FailNow(t, "can't declare exchange", exchange, err)
	}
}

// DeclareRMQQueue is helper function for declaring RMQ queue and binding and cleaning them after test is finished.
func DeclareRMQQueue(t *testing.T, channel *amqp.Channel, queueName, exchange, routingKey string) {
	t.Helper()

	_, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		require.FailNow(t, "can't declare queue", queueName, err)
	}

	err = channel.QueueBind(queueName, routingKey, exchange, false, nil)
	if err != nil {
		require.FailNow(t, "can't bind queue", queueName, routingKey, err)
	}

	t.Cleanup(func() {
		_, err := channel.QueueDelete(queueName, false, false, true)
		if err != nil {
			require.FailNow(t, "can't delete queue", queueName, err)
		}
	})
}

// GenerateTestData generates n products with ProductID in [1;n].
func GenerateTestData(t *testing.T, n int) []models.Product {
	t.Helper()

	results := make([]models.Product, n)

	for ix := range n {
		results[ix] = modelstesting.FakeProduct(func(p *models.Product) { p.ProductID = strconv.Itoa(ix + 1) })
	}

	return results
}

// ToDecoderProduct is helper function for converting products to model from internal/decoder package.
func ToDecoderProduct(t *testing.T, products []models.Product) []decoder.Product {
	t.Helper()

	results := make([]decoder.Product, len(products))

	for ix := range products {
		results[ix] = *toDecoderProduct(&products[ix])
	}

	return results
}

// ProductsToXML is helper function which converts products to xml and returns them as byte slice.
func ProductsToXML(t *testing.T, products []decoder.Product) []byte {
	t.Helper()

	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)

	for ix := range products {
		err := encoder.EncodeElement(&products[ix], xml.StartElement{Name: xml.Name{Local: "item"}})
		if err != nil {
			require.FailNow(t, "can't encode product to xml", err)
		}
	}

	err := encoder.Flush()
	if err != nil {
		require.FailNow(t, "can't flush xml encoder", err)
	}

	err = encoder.Close()
	if err != nil {
		require.FailNow(t, "can't close xml encoder", err)
	}

	return buf.Bytes()
}

func toDecoderProduct(product *models.Product) *decoder.Product {
	return &decoder.Product{
		ID:                  product.ProductID,
		Title:               product.Title,
		Description:         product.Description,
		URL:                 product.URL,
		ImageURL:            product.ImageURL,
		AdditionalImageURLs: product.AdditionalImageURLs,
		Condition:           product.Condition,
		Availability:        product.Availability,
		Price:               product.Price,
		Shippings:           toDecoderShippings(product.Shippings),
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

func toDecoderShippings(shippings []models.Shipping) []decoder.Shipping {
	if len(shippings) == 0 {
		return nil
	}

	result := make([]decoder.Shipping, len(shippings))

	for ix := range shippings {
		result[ix] = *toDecoderShipping(&shippings[ix])
	}
	return result
}

func toDecoderShipping(shipping *models.Shipping) *decoder.Shipping {
	return &decoder.Shipping{
		Country: shipping.Country,
		Service: shipping.Service,
		Price:   shipping.Price,
	}
}

// ToDBProduct converts models.Product into postgres product model.
func fromDBProduct(product *pgmodels.Product, shippings []pgmodels.Shipping) *models.Product {
	return &models.Product{
		Version:             product.Version,
		ProductID:           product.ProductID,
		Title:               product.Title,
		Description:         product.Description,
		URL:                 product.URL,
		ImageURL:            product.ImgURL,
		AdditionalImageURLs: fromDBAdditionalImageURLs(product.AdditionalImgUrls),
		Shippings:           fromDBShippings(shippings),
		Condition:           product.Condition,
		Availability:        product.Availability,
		Price:               product.Price,
		Brand:               product.Brand,
		GTIN:                product.Gtin,
		MPN:                 product.Mpn,
		ProductCategory:     product.ProductCategory,
		ProductType:         product.ProductType,
		Color:               product.Color,
		Size:                product.Size,
		ItemGroupID:         product.ItemGroupID,
		Gender:              product.Gender,
		AgeGroup:            product.AgeGroup,
		CreatedAt:           product.CreatedAt,
		DeletedAt:           product.DeletedAt,
	}
}

// ToDBShippings converts models.Shipping slice into postgres shipping slice.
func fromDBShippings(shippings []pgmodels.Shipping) []models.Shipping {
	if len(shippings) == 0 {
		return []models.Shipping{}
	}

	result := make([]models.Shipping, 0, len(shippings))
	for ix := range shippings {
		result = append(result, models.Shipping{
			Country: shippings[ix].Country,
			Service: shippings[ix].Service,
			Price:   shippings[ix].Price,
		})
	}
	return result
}

func fromDBAdditionalImageURLs(urls string) []string {
	if urls == "" {
		return []string{}
	}
	return strings.Split(urls, "\n")
}
