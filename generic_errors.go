package magento2

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
// magento2.ErrNotFound) (404) and errors.Is(err, magento2.ErrBadRequest)
// (any other status) keep working.
//
// Every string field, Body, Parameters and the Header values have passed the
// Client's redactor (WithRedactor).
type APIError struct {
	StatusCode int
	Method     string
	// Path is Request.Path (relative to /rest, without the store code or the
	// query string).
	Path   string
	Header http.Header
	// Body is the response body, capped at the request's body cap
	// (Request.MaxBodyBytes, else WithMaxBodyBytes) or 64 KiB, whichever is
	// smaller; 64 KiB when the cap is unlimited.
	Body []byte
	// Message is the error document's message with its parameters
	// substituted (see ErrorDocument.Substituted), or the HTTP status text
	// when the body is not a Magento error document.
	Message string
	// Parameters is the error document's raw "parameters" (an object of
	// named or an array of positional placeholders), nil when absent.
	Parameters json.RawMessage

	sentinel error // ErrNotFound / ErrBadRequest, exposed via Unwrap
}

// Error is "magento2: GET /V1/orders: status 404: <message>"; it never
// carries the host or the query string.
func (e *APIError) Error() string {
	msg := "magento2: "
	if e.Method != "" || e.Path != "" {
		msg += e.Method + " " + e.Path + ": "
	}
	msg += "status " + strconv.Itoa(e.StatusCode)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

func (e *APIError) Unwrap() error {
	return e.sentinel
}

func (c *Client) newAPIError(method, path string, resp *http.Response, body []byte) *APIError {
	e := &APIError{
		StatusCode: resp.StatusCode,
		Method:     c.redact(method),
		Path:       c.redact(path),
		Body:       []byte(c.redact(string(body))),
		Message:    http.StatusText(resp.StatusCode),
		sentinel:   ErrBadRequest,
	}
	if resp.StatusCode == http.StatusNotFound {
		e.sentinel = ErrNotFound
	}
	if e.Message == "" {
		e.Message = "unexpected status"
	}
	if len(resp.Header) > 0 {
		e.Header = make(http.Header, len(resp.Header))
		for k, vs := range resp.Header {
			rv := make([]string, len(vs))
			for i, v := range vs {
				rv[i] = c.redact(v)
			}
			e.Header[k] = rv
		}
	}
	if doc, ok := ParseErrorDocument(body); ok {
		e.Message = doc.Substituted(c.redact)
		if len(doc.Parameters) > 0 {
			e.Parameters = json.RawMessage(c.redact(string(doc.Parameters)))
		}
	} else {
		e.Message = c.redact(e.Message)
	}
	return e
}
