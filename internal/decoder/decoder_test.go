package decoder_test

import (
	"context"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/MichalMitros/google-feed-parser/internal/decoder"
	"github.com/MichalMitros/google-feed-parser/internal/decoder/testdata"
	"github.com/MichalMitros/google-feed-parser/internal/platform/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

const feedFileName = "feed.xml"

func TestUnitDecode(t *testing.T) {
	file := FeedFileAsReader(t)

	results := make(chan models.ParsingResult)
	dec := decoder.Decoder{}

	var eg errgroup.Group

	eg.Go(func() error {
		defer close(results)
		return dec.Decode(context.TODO(), file, results)
	})

	var (
		products       []models.Product
		decodingErrors []error
	)
	eg.Go(func() error {
		products, decodingErrors = collect(results)
		return nil
	})

	require.NoError(t, eg.Wait(), "should not return any error")
	assert.Equal(t, testdata.Products, products, "should correctly decode all products")
	assert.Equal(t, []error{nil, nil, nil, nil}, decodingErrors,
		"should decode all products without any error",
	)
}

func TestUnitDecodeBadXMLFormat(t *testing.T) {
	badFile := strings.NewReader("<item><g:id></item>")

	results := make(chan models.ParsingResult)
	dec := decoder.Decoder{}

	var eg errgroup.Group

	eg.Go(func() error {
		defer close(results)
		return dec.Decode(context.TODO(), badFile, results)
	})

	var (
		products       []models.Product
		decodingErrors []error
	)
	eg.Go(func() error {
		products, decodingErrors = collect(results)
		return nil
	})

	require.EqualError(t, eg.Wait(),
		"XML syntax error on line 1: element <id> closed by </item>",
		"should return correct decoding error",
	)
	assert.Equal(t, []models.Product{{}}, products, "should return empty product")
	require.EqualError(t, decodingErrors[0],
		"XML syntax error on line 1: element <id> closed by </item>",
		"should return correct decoding error",
	)
}

func collect(resultsCh <-chan models.ParsingResult) ([]models.Product, []error) {
	var (
		products []models.Product
		errors   []error
	)

	for result := range resultsCh {
		products = append(products, result.Product)
		errors = append(errors, result.Error)
	}

	return products, errors
}

// FeedFileAsReader returns io.Reader with feed file.
func FeedFileAsReader(t *testing.T) io.Reader {
	t.Helper()

	f, err := os.Open(path.Join("testdata", feedFileName))
	require.NoError(t, err)

	return f
}
