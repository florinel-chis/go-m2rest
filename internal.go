package magento2

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
	"gopkg.in/resty.v1"
)

var ErrBadRequest = fmt.Errorf("%s", "bad request")
var ErrNotFound = fmt.Errorf("%s", "not found") // Declare ErrNotFound here if it's not already declared elsewhere

func wrapError(err error, triedTo string, response ...map[string]interface{}) error {
	if len(response) == 0 {
		log.Error().Err(err).Str("operation", triedTo).Msg("Error while trying to")
		return fmt.Errorf("error while trying to %w - %s", err, triedTo)
	}
	log.Error().Err(err).Str("operation", triedTo).Interface("responseDetails", response).Msg("Error while trying to, with response details")
	return fmt.Errorf("error while trying to %w - %s. %+v", err, triedTo, response)
}

func mayReturnErrorForHTTPResponse(resp *resty.Response, triedTo string) error {
	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			log.Warn().
				Int("statusCode", resp.StatusCode()).
				Str("operation", triedTo).
				Str("responseBody", string(resp.Body())).
				Msg("Not found error")
			return ErrNotFound
		} else if resp.StatusCode() >= http.StatusBadRequest {
			additional := map[string]interface{}{
				"statusCode": resp.StatusCode(),
				"response":   string(resp.Body()),
			}
			log.Error().
				Int("statusCode", resp.StatusCode()).
				Str("operation", triedTo).
				Interface("additionalDetails", additional).
				Msg("Bad request error")
			return wrapError(ErrBadRequest, triedTo, additional)
		}
		// For other non-2xx and non-404 errors, still wrap and log
		additional := map[string]interface{}{
			"statusCode": resp.StatusCode(),
			"response":   string(resp.Body()),
		}
		log.Error().
			Int("statusCode", resp.StatusCode()).
			Str("operation", triedTo).
			Interface("additionalDetails", additional).
			Msg("HTTP error")
		return wrapError(fmt.Errorf("http status error: %d", resp.StatusCode()), triedTo, additional) // Wrap with a generic HTTP error
	}
	return nil
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
