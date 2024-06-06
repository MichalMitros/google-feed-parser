package fetcher_test

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MichalMitros/google-feed-parser/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	userAgent   = "test/0.0.0"
	response    = "hello-world"
	endpoint    = "/file"
	contentType = "Content-Type"
)

func TestUniFetchFile(t *testing.T) {
	wantHeaders := map[string]string{
		"User-Agent":      userAgent,
		"Accept":          "application/xml",
		"Accept-Encoding": "gzip",
	}

	tests := map[string]struct {
		serverHandler http.Handler
		wantBody      string
		wantErr       error
	}{
		"ok xml": {
			serverHandler: http.HandlerFunc(func(wrt http.ResponseWriter, req *http.Request) {
				validateHeaders(t, req.Header, wantHeaders)
				wrt.Header().Add(contentType, "application/xml")
				wrt.Write([]byte(response))
				wrt.WriteHeader(http.StatusOK)
			}),
			wantBody: response,
		},
		"ok gzip": {
			serverHandler: http.HandlerFunc(func(wrt http.ResponseWriter, req *http.Request) {
				validateHeaders(t, req.Header, wantHeaders)
				wrt.Header().Add(contentType, "application/zip")
				compressedWrt := gzip.NewWriter(wrt)
				compressedWrt.Write([]byte(response))
				compressedWrt.Flush()
				compressedWrt.Close()
				wrt.WriteHeader(http.StatusOK)
			}),
			wantBody: response,
		},
		"bad status error": {
			serverHandler: http.HandlerFunc(func(wrt http.ResponseWriter, req *http.Request) {
				validateHeaders(t, req.Header, wantHeaders)
				wrt.WriteHeader(http.StatusInternalServerError)
			}),
			wantBody: "",
			wantErr:  fetcher.ErrStatusNotOK,
		},
		"bad content type error": {
			serverHandler: http.HandlerFunc(func(wrt http.ResponseWriter, req *http.Request) {
				validateHeaders(t, req.Header, wantHeaders)
				wrt.Header().Add(contentType, "application/http")
				wrt.Write([]byte(response))
				wrt.WriteHeader(http.StatusOK)
			}),
			wantBody: "",
			wantErr:  fetcher.ErrContentTypeNotSupported,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				srv.Close()
			})

			fet := fetcher.NewFetcher(srv.Client(), userAgent)
			resp, err := fet.FetchFile(context.TODO(), srv.URL+"/"+endpoint)

			require.ErrorIs(t, err, tt.wantErr, "should return correct error")

			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, readAndClose(t, resp), "should return correct response")
			}
		})
	}
}

// readAndClose reads ReadCloser, closes it and returns result as string.
func readAndClose(t *testing.T, reader io.ReadCloser) string {
	t.Helper()

	if !assert.NotNil(t, reader, "reader shouldn't be nil") {
		return ""
	}

	result, err := io.ReadAll(reader)
	if !assert.NoError(t, err, "can't read reader") {
		return ""
	}

	assert.NoError(t, reader.Close(), "can't close reader")

	return string(result)
}

// ReadAndClose reads ReadCloser, closes it and returns result as string.
func validateHeaders(t *testing.T, headers http.Header, expected map[string]string) {
	t.Helper()

	for header, expectedValue := range expected {
		assert.Equalf(t, expectedValue, headers.Get(header), "request should contain correct value for header %s", header)
	}
}
