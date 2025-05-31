package magento2

import (
	"context" // Added context package
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

const (
	RetryAttempts       = 3
	RetryWaitSeconds    = 5
	RetryMaxWaitSeconds = 20
)

// SetLogger is deprecated. Use SetZeroLogger from logger.go instead
func SetLogger(l any) {
	log.Warn().Msg("SetLogger is deprecated. Use SetZeroLogger instead")
}

type Client struct {
	HTTPClient *resty.Client
}

type StoreConfig struct {
	Scheme    string
	HostName  string
	StoreCode string
}

// GetRouteAndDecode now accepts context.Context
func (c *Client) GetRouteAndDecode(ctx context.Context, route string, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	log.Debug().Str("route", route).Msg("GET request with context")
	// Pass context to Resty request
	resp, err := c.HTTPClient.R().SetContext(ctx).SetResult(target).Get(route)
	if err != nil {
		// Check if the error is due to context cancellation or deadline
		if ctx.Err() != nil {
			log.Error().Err(ctx.Err()).Str("route", route).Msg("Context error during GET request")
			return ctx.Err()
		}
		log.Error().Err(err).Str("route", route).Msg("GET request failed")
		return err
	}
	// No change needed for mayReturnErrorForHTTPResponse, as it handles response after it's received
	log.Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("GET request completed")
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

// PostRouteAndDecode now accepts context.Context
func (c *Client) PostRouteAndDecode(ctx context.Context, route string, body, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	log.Debug().Str("route", route).Interface("body", body).Msg("POST request with context")
	// Pass context to Resty request
	resp, err := c.HTTPClient.R().SetContext(ctx).SetResult(target).SetBody(body).Post(route)
	if err != nil {
		// Check if the error is due to context cancellation or deadline
		if ctx.Err() != nil {
			log.Error().Err(ctx.Err()).Str("route", route).Msg("Context error during POST request")
			return ctx.Err()
		}
		log.Error().Err(err).Str("route", route).Msg("POST request failed")
		return err
	}
	log.Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("POST request completed")
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

// NewAPIClientWithoutAuthentication does not perform I/O, no context needed in its signature.
func NewAPIClientWithoutAuthentication(storeConfig *StoreConfig) *Client {
	httpClient := buildBasicHTTPClient(storeConfig)
	log.Info().Interface("storeConfig", storeConfig).Msg("Created API client without authentication")

	return &Client{
		HTTPClient: httpClient,
	}
}

// NewAPIClientFromAuthentication now accepts context.Context for the auth request.
func NewAPIClientFromAuthentication(ctx context.Context, storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	log.Info().Interface("storeConfig", storeConfig).Str("authenticationType", authenticationType.Route()).Interface("payload", payload).Msg("Authenticating API client with context")
	// Pass context to Resty request for authentication
	resp, err := client.R().SetContext(ctx).SetBody(payload).Post(authenticationType.Route())
	if err != nil {
		if ctx.Err() != nil {
			log.Error().Err(ctx.Err()).Str("authenticationType", authenticationType.Route()).Msg("Context error during authentication")
			return nil, ctx.Err()
		}
		log.Error().Err(err).Str("authenticationType", authenticationType.Route()).Msg("Authentication request failed")
		return nil, err
	}

	// Check for HTTP errors after successful request execution
	// mayReturnErrorForHTTPResponse could be used here if auth errors are structured like other API errors
	// For now, assuming a non-2xx is an auth failure not covered by mayReturnErrorForHTTPResponse's specific logic for ErrNotFound etc.
	if resp.IsError() {
		err := fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode(), resp.String())
		log.Error().Err(err).Str("authenticationType", authenticationType.Route()).Msg("Authentication HTTP error")
		return nil, err
	}


	token := mayTrimSurroundingQuotes(resp.String())
	client.SetAuthToken(token) // This method is on resty.Client, not our Client struct.
	// To make this work, buildBasicHTTPClient should return the *resty.Client,
	// and it should be configured with the token before being assigned to our Client struct.
	// OR, our Client struct should have a SetAuthToken method that configures its internal resty.Client.
	// For now, assuming `client.SetAuthToken(token)` correctly configures the underlying Resty client instance.
	// If `client` here is the `*resty.Client`, then the return should be `&Client{HTTPClient: client}`.
	// If `client` is `*Client`, then `client.HTTPClient.SetAuthToken(token)` is needed.
	// The current code implies `client` is `*resty.Client`.

	// Correcting based on typical Resty usage:
	// `buildBasicHTTPClient` returns a `*resty.Client`. Let's name it `httpClient`.
	// Then we use `httpClient` to make the auth call.
	// After getting the token, we set it on `httpClient`.
	// Finally, we return `&Client{HTTPClient: httpClient}`.

	// Re-evaluating the original code: `client` in `NewAPIClientFromAuthentication` IS the `*resty.Client`.
	// So `client.SetAuthToken(token)` is correct for the `resty.Client`.
	// The return `&Client{HTTPClient: client}` is also correct.

	log.Info().Str("authenticationType", authenticationType.Route()).Msg("API client authenticated successfully")

	return &Client{
		HTTPClient: client,
	}, nil
}

// NewAPIClientFromIntegration does not perform I/O, no context needed in its signature.
func NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string) (*Client, error) {
	httpClient := buildBasicHTTPClient(storeConfig)

	httpClient.SetAuthToken(bearer) // Correctly setting token on resty.Client
	log.Info().Interface("storeConfig", storeConfig).Msg("Created API client from integration")

	return &Client{
		HTTPClient: httpClient,
	}, nil
}

func buildBasicHTTPClient(storeConfig *StoreConfig) *resty.Client {
	apiVersion := "/V1"
	restPrefix := "/rest/" + storeConfig.StoreCode
	fullRestRoute := storeConfig.Scheme + "://" + storeConfig.HostName + restPrefix + apiVersion
	client := resty.New()
	client.SetHostURL(fullRestRoute)
	client.SetHeaders(map[string]string{
		"User-Agent": "go-m2rest",
	})
	client.SetDebug(false)

	retryWait := time.Duration(RetryWaitSeconds)
	retryMaxWait := time.Duration(RetryMaxWaitSeconds)
	client.SetRetryCount(RetryAttempts).
		SetRetryWaitTime(retryWait * time.Second).
		SetRetryMaxWaitTime(retryMaxWait * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r != nil {
					status := r.StatusCode()
					// Also check for context errors on the request if possible, though Resty might handle this internally.
					// If err is context.DeadlineExceeded or context.Canceled, should not retry.
					if err != nil {
						if err == context.Canceled || err == context.DeadlineExceeded {
							return false
						}
					}
					return status == http.StatusServiceUnavailable || status == http.StatusInternalServerError
				}
				return false
			},
		)
	log.Debug().Str("route", fullRestRoute).Msg("Built basic HTTP client")
	return client
}
