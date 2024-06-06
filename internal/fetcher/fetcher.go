package fetcher

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
)

// Fetcher builds http requests and fetches files via http.
type Fetcher struct {
	client    *http.Client
	userAgent string
}

// NewFetcher returns new Fetcher.
func NewFetcher(client *http.Client, userAgent string) *Fetcher {
	return &Fetcher{
		client:    client,
		userAgent: userAgent,
	}
}

// FetchFile returns ReadCloser with file fetched from provided url or error.
// The caller is responsible for closing returned ReadCloser.
func (f *Fetcher) FetchFile(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("can't build http request: %w", err)
	}

	req.Header.Add("Accept", "application/xml")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't get http response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, ErrStatusNotOK
	}

	switch resp.Header.Get("Content-Type") {
	case "application/xml":
		return resp.Body, nil
	case "application/zip":
		return decompressResponse(resp.Body)
	default:
		return nil, ErrContentTypeNotSupported
	}
}

// decompressResponse returns io.ReadCloser with decompressed http response and error.
func decompressResponse(response io.ReadCloser) (io.ReadCloser, error) {
	decompressed, err := gzip.NewReader(response)
	if err != nil {
		return nil, fmt.Errorf("can't decompress response: %w", err)
	}

	return &decompressedReadCloser{
		compressed:   response,
		decompressed: decompressed,
	}, nil
}

// decompressedReadCloser wraps decompressed Reader and compressed ReadCloser.
// It reads from decompressed Reader, but closes compressed ReadCloser.
type decompressedReadCloser struct {
	compressed   io.ReadCloser
	decompressed io.Reader
}

// Read reads uncompressed bytes from underlying Reader into p.
// Returns number of read bytes and error.
func (r decompressedReadCloser) Read(p []byte) (n int, err error) {
	return r.decompressed.Read(p)
}

// Close closes underlying compressed ReadCloser.
func (r decompressedReadCloser) Close() error {
	return r.compressed.Close()
}
