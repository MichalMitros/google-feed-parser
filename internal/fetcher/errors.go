package fetcher

import "errors"

var (
	// ErrStatusNotOK is returned when http response had status differen than 200 OK.
	ErrStatusNotOK = errors.New("response status is not 200 OK")
	// ErrContentTypeNotSupported is returned when response content type is not supported.
	ErrContentTypeNotSupported = errors.New("response content type not supported")
)
