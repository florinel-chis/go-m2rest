package magento2

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// maxErrorBodyBytes limits how much of an error response body is retained in
// an APIError.
const maxErrorBodyBytes = 500

func truncateErrorBody(body []byte) string {
	if len(body) > maxErrorBodyBytes {
		return string(body[:maxErrorBodyBytes])
	}
	return string(body)
}

func mayReturnErrorForHTTPResponse(resp *resty.Response, triedTo string) error {
	if !resp.IsError() {
		return nil
	}

	endpoint := ""
	if resp.Request != nil {
		endpoint = resp.Request.Method + " " + resp.Request.URL
	}
	apiErr := &APIError{
		StatusCode: resp.StatusCode(),
		Endpoint:   endpoint,
		Body:       truncateErrorBody(resp.Body()),
	}

	if resp.StatusCode() == http.StatusNotFound {
		apiErr.sentinel = ErrNotFound
		logger.Warn().
			Int("statusCode", resp.StatusCode()).
			Str("operation", triedTo).
			Str("responseBody", apiErr.Body).
			Msg("Not found error")
		return apiErr
	}

	// All other non-2xx responses keep the historical ErrBadRequest sentinel.
	apiErr.sentinel = ErrBadRequest
	logger.Error().
		Int("statusCode", resp.StatusCode()).
		Str("operation", triedTo).
		Str("responseBody", apiErr.Body).
		Msg("HTTP error")
	return fmt.Errorf("error while trying to %s: %w", triedTo, apiErr)
}

func mayTrimSurroundingQuotes(s string) string {
	minQuotes := 2
	if len(s) >= minQuotes {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
