package magento2

import (
	"errors"
	"fmt"
)

var ErrNoPointer = errors.New("target interface must be a pointer")

var ErrNotFound = errors.New("no document found")

var ErrBadRequest = errors.New("bad request")

// APIError describes a non-2xx HTTP response from the Magento API. Errors
// returned by this library can be inspected with errors.As:
//
//	var apiErr *magento2.APIError
//	if errors.As(err, &apiErr) { ... apiErr.StatusCode ... }
//
// APIError wraps the historical sentinel errors, so errors.Is(err,
// magento2.ErrNotFound) and errors.Is(err, magento2.ErrBadRequest) keep
// working.
type APIError struct {
	StatusCode int
	Endpoint   string
	Body       string // response body, truncated to 500 bytes

	sentinel error // ErrNotFound / ErrBadRequest, exposed via Unwrap
}

func (e *APIError) Error() string {
	msg := "magento2: "
	if e.Endpoint != "" {
		msg += e.Endpoint + ": "
	}
	msg += fmt.Sprintf("status %d", e.StatusCode)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

func (e *APIError) Unwrap() error {
	return e.sentinel
}
