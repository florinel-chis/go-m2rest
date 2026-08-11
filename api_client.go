package magento2

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Deprecated: legacy retry constants, kept for API compatibility. The client
// defaults are DefaultRetryCount, DefaultRetryWaitTime and
// DefaultRetryMaxWaitTime; use (*Client).SetRetryPolicy to override them.
const (
	RetryAttempts       = 3
	RetryWaitSeconds    = 5
	RetryMaxWaitSeconds = 20
)

// Defaults applied by all client constructors.
const (
	// DefaultTimeout is the per-attempt HTTP timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultRetryCount is the number of retries after the initial attempt
	// (i.e. 5 total attempts).
	DefaultRetryCount = 4
	// DefaultRetryWaitTime is the base wait time between retries.
	DefaultRetryWaitTime = 500 * time.Millisecond
	// DefaultRetryMaxWaitTime caps the wait time between retries. It is
	// deliberately generous because resty clamps a server-provided
	// Retry-After value to this maximum; the computed jittered backoff
	// stays far below it with the default policy (500ms base, 4 retries
	// gives at most ~4s).
	DefaultRetryMaxWaitTime = 2 * time.Minute
)

// retryStatusCodes are the HTTP status codes that trigger a retry.
var retryStatusCodes = map[int]bool{
	http.StatusTooManyRequests:     true, // 429
	http.StatusInternalServerError: true, // 500
	http.StatusBadGateway:          true, // 502
	http.StatusServiceUnavailable:  true, // 503
	http.StatusGatewayTimeout:      true, // 504
}

// retryableMethods are the HTTP methods that are retried automatically.
// POST and PUT are deliberately excluded: Magento uses them for
// non-idempotent writes (order placement via PUT /carts/{id}/order, entity
// creation via POST), and a 429/502/504 from a proxy can arrive after the
// backend has already committed the write, so an automatic retry could
// duplicate orders or created entities.
var retryableMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodDelete:  true,
}

// SetLogger is deprecated. Use SetZeroLogger from logger.go instead
func SetLogger(l any) {
	logger().Warn().Msg("SetLogger is deprecated. Use SetZeroLogger instead")
}

type Client struct {
	HTTPClient *resty.Client
}

type StoreConfig struct {
	Scheme    string
	HostName  string // may include a port, e.g. "shop.example.com:8080"
	StoreCode string
	// BasePath is an optional path prefix for Magento installations served
	// from a sub-directory, e.g. "shop" results in
	// {scheme}://{host}/shop/rest/{storeCode}/V1. Leading/trailing slashes
	// are normalized. Empty keeps the previous behavior.
	BasePath string
}

// baseURL builds {scheme}://{host}[/basePath]/rest/{storeCode}/V1.
func (storeConfig *StoreConfig) baseURL() string {
	base := storeConfig.Scheme + "://" + storeConfig.HostName
	if basePath := strings.Trim(storeConfig.BasePath, "/"); basePath != "" {
		base += "/" + basePath
	}
	return base + "/rest/" + storeConfig.StoreCode + "/V1"
}

// SetTimeout sets the per-attempt HTTP timeout. All constructors default to
// DefaultTimeout.
func (c *Client) SetTimeout(d time.Duration) {
	c.HTTPClient.SetTimeout(d)
}

// SetRetryPolicy configures how many times a failed request is retried
// (count is the number of retries after the initial attempt) and the
// base/maximum wait times between attempts. The Retry-After response header,
// when present, still takes precedence over the computed backoff, but both
// are capped at maxWait (so keep maxWait large enough to honor the
// Retry-After values your rate limiter emits; the default is
// DefaultRetryMaxWaitTime). Only idempotent requests (GET, HEAD, OPTIONS,
// DELETE) are retried; see retryableMethods.
func (c *Client) SetRetryPolicy(count int, wait, maxWait time.Duration) {
	c.HTTPClient.
		SetRetryCount(count).
		SetRetryWaitTime(wait).
		SetRetryMaxWaitTime(maxWait)
}

// GetRouteAndDecodeCtx performs a GET request on route with the given
// context, decoding the JSON response into target (which must be a pointer).
func (c *Client) GetRouteAndDecodeCtx(ctx context.Context, route string, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	logger().Debug().Str("route", route).Msg("GET request")
	req := c.HTTPClient.R()
	if ctx != nil {
		req.SetContext(ctx)
	}
	resp, err := req.SetResult(target).Get(route)
	if err != nil {
		logger().Error().Err(err).Str("route", route).Msg("GET request failed")
		return err
	}
	logger().Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("GET request completed")
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

// PostRouteAndDecodeCtx performs a POST request on route with the given
// context, decoding the JSON response into target (which must be a pointer).
func (c *Client) PostRouteAndDecodeCtx(ctx context.Context, route string, body, target any, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}

	logger().Debug().Str("route", route).Interface("body", body).Msg("POST request")
	req := c.HTTPClient.R()
	if ctx != nil {
		req.SetContext(ctx)
	}
	resp, err := req.SetResult(target).SetBody(body).Post(route)
	if err != nil {
		logger().Error().Err(err).Str("route", route).Msg("POST request failed")
		return err
	}
	logger().Debug().Str("route", route).Int("status", resp.StatusCode()).Msg("POST request completed")
	return mayReturnErrorForHTTPResponse(resp, tryTo)
}

func (c *Client) GetRouteAndDecode(route string, target any, tryTo string) error {
	return c.GetRouteAndDecodeCtx(context.Background(), route, target, tryTo)
}

func (c *Client) PostRouteAndDecode(route string, body, target any, tryTo string) error {
	return c.PostRouteAndDecodeCtx(context.Background(), route, body, target, tryTo)
}

func NewAPIClientWithoutAuthentication(storeConfig *StoreConfig) *Client {
	httpClient := buildBasicHTTPClient(storeConfig)
	logger().Info().Interface("storeConfig", storeConfig).Msg("Created API client without authentication")

	return &Client{
		HTTPClient: httpClient,
	}
}

func NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	logger().Info().Interface("storeConfig", storeConfig).Str("authenticationType", authenticationType.Route()).Msg("Authenticating API client")
	resp, err := client.R().SetBody(payload).Post(authenticationType.Route())
	if err != nil {
		return nil, err
	}

	token := mayTrimSurroundingQuotes(resp.String())
	client.SetAuthToken(token)
	logger().Info().Str("authenticationType", authenticationType.Route()).Msg("API client authenticated successfully")

	return &Client{
		HTTPClient: client,
	}, nil
}

func NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	client.SetAuthToken(bearer)
	logger().Info().Interface("storeConfig", storeConfig).Msg("Created API client from integration")

	return &Client{
		HTTPClient: client,
	}, nil
}

func buildBasicHTTPClient(storeConfig *StoreConfig) *resty.Client {
	fullRestRoute := storeConfig.baseURL()
	client := resty.New()
	client.SetBaseURL(fullRestRoute)
	client.SetHeaders(map[string]string{
		"User-Agent": "go-m2rest",
	})
	client.SetDebug(false) // Set to true for very verbose resty debugging
	client.SetTimeout(DefaultTimeout)

	client.SetRetryCount(DefaultRetryCount).
		SetRetryWaitTime(DefaultRetryWaitTime).
		SetRetryMaxWaitTime(DefaultRetryMaxWaitTime).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if r == nil || r.Request == nil || !retryableMethods[r.Request.Method] {
					return false
				}
				if err != nil {
					// Transport-level failure (per-attempt timeout,
					// connection reset, TLS error, ...): retry. resty
					// checks the request context before every retry and
					// during the backoff sleep, so a canceled/expired
					// caller context still aborts promptly.
					return true
				}
				return retryStatusCodes[r.StatusCode()]
			},
		).
		SetRetryAfter(func(cl *resty.Client, resp *resty.Response) (time.Duration, error) {
			if resp != nil {
				if retryAfter := strings.TrimSpace(resp.Header().Get("Retry-After")); retryAfter != "" {
					// RFC 9110 allows both delta-seconds and an HTTP-date.
					if seconds, convErr := strconv.Atoi(retryAfter); convErr == nil && seconds > 0 {
						return time.Duration(seconds) * time.Second, nil
					}
					if at, convErr := http.ParseTime(retryAfter); convErr == nil {
						if wait := time.Until(at); wait > 0 {
							return wait, nil
						}
					}
				}
			}
			return 0, nil // fall back to the default backoff algorithm
		})
	logger().Debug().Str("route", fullRestRoute).Msg("Built basic HTTP client")
	return client
}
