package magento2

import (
	"net/http"
	"reflect"
	"time"

	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/go-resty/resty/v2"
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

func (c *Client) GetRouteAndDecode(route string, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	log.Debug().Str("route", route).Msg("GET request")
	resp, err := c.HTTPClient.R().SetResult(target).Get(route)
	if err != nil {
		log.Error().Err(err).Str("route", route).Msg("GET request failed")
		return err
	} else {
		log.Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("GET request completed")
	}
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

func (c *Client) PostRouteAndDecode(route string, body, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	log.Debug().Str("route", route).Interface("body", body).Msg("POST request")
	resp, err := c.HTTPClient.R().SetResult(target).SetBody(body).Post(route)
	if err != nil {
		log.Error().Err(err).Str("route", route).Msg("POST request failed")
		return err
	} else {
		log.Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("POST request completed")
	}
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

func NewAPIClientWithoutAuthentication(storeConfig *StoreConfig) *Client {
	httpClient := buildBasicHTTPClient(storeConfig)
	log.Info().Interface("storeConfig", storeConfig).Msg("Created API client without authentication")

	return &Client{
		HTTPClient: httpClient,
	}
}

func NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	log.Info().Interface("storeConfig", storeConfig).Str("authenticationType", authenticationType.Route()).Interface("payload", payload).Msg("Authenticating API client")
	resp, err := client.R().SetBody(payload).Post(authenticationType.Route())
	if err != nil {
		return nil, err
	}

	token := mayTrimSurroundingQuotes(resp.String())
	client.SetAuthToken(token)
	log.Info().Str("authenticationType", authenticationType.Route()).Msg("API client authenticated successfully")

	return &Client{
		HTTPClient: client,
	}, nil
}

func NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	client.SetAuthToken(bearer)
	log.Info().Interface("storeConfig", storeConfig).Msg("Created API client from integration")

	return &Client{
		HTTPClient: client,
	}, nil
}

func buildBasicHTTPClient(storeConfig *StoreConfig) *resty.Client {
	apiVersion := "/V1"
	restPrefix := "/rest/" + storeConfig.StoreCode
	fullRestRoute := storeConfig.Scheme + "://" + storeConfig.HostName + restPrefix + apiVersion
	client := resty.New()
	// SetRESTMode is not needed in resty v2
	client.SetHostURL(fullRestRoute)
	client.SetHeaders(map[string]string{
		"User-Agent": "go-m2rest",
	})
	client.SetDebug(false) // Set to true for very verbose resty debugging

	retryWait := time.Duration(RetryWaitSeconds)
	retryMaxWait := time.Duration(RetryMaxWaitSeconds)
	client.SetRetryCount(RetryAttempts).
		SetRetryWaitTime(retryWait * time.Second).
		SetRetryMaxWaitTime(retryMaxWait * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r != nil {
					status := r.StatusCode()
					return status == http.StatusServiceUnavailable || status == http.StatusInternalServerError
				}
				return false
			},
		)
	log.Debug().Str("route", fullRestRoute).Msg("Built basic HTTP client")
	return client
}
