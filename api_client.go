package magento2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

// StoreConfig describes a store for the compat constructors
// (NewAPIClientFromIntegration, NewAPIClientFromAuthentication,
// NewAPIClientWithoutAuthentication). New takes the store root URL instead.
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

// root builds {scheme}://{host}[/basePath], the store root New expects.
func (storeConfig *StoreConfig) root() string {
	base := storeConfig.Scheme + "://" + storeConfig.HostName
	if basePath := strings.Trim(storeConfig.BasePath, "/"); basePath != "" {
		base += "/" + basePath
	}
	return base
}

// baseURL builds {scheme}://{host}[/basePath]/rest/{storeCode}/V1, the
// prefix the compat route helpers resolve their routes against.
func (storeConfig *StoreConfig) baseURL() string {
	return storeConfig.root() + "/rest/" + storeConfig.StoreCode + "/V1"
}

// newFromStoreConfig is New for the compat constructors: the store code
// comes first so opts can still override it.
func newFromStoreConfig(storeConfig *StoreConfig, opts []ClientOption) (*Client, error) {
	if storeConfig == nil {
		return nil, fmt.Errorf("magento2: nil StoreConfig")
	}
	all := make([]ClientOption, 0, len(opts)+1)
	if storeConfig.StoreCode != "" {
		all = append(all, WithStoreCode(storeConfig.StoreCode))
	}
	return New(storeConfig.root(), append(all, opts...)...)
}

// NewAPIClientWithoutAuthentication returns a client that sends no
// credentials. opts are New's options. Its signature has no error: an
// invalid StoreConfig or option makes every request of the returned client
// fail with the construction error.
func NewAPIClientWithoutAuthentication(storeConfig *StoreConfig, opts ...ClientOption) *Client {
	c, err := newFromStoreConfig(storeConfig, opts)
	if err != nil {
		return &Client{initErr: err}
	}
	return c
}

// NewAPIClientFromAuthentication exchanges username and password for a token
// (POST /V1/integration/admin/token or /V1/integration/customer/token) and
// returns a client that sends it. opts are New's options; a WithToken among
// them is replaced by the issued token.
func NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType, opts ...ClientOption) (*Client, error) {
	anon, err := newFromStoreConfig(storeConfig, append(append([]ClientOption(nil), opts...), WithToken("")))
	if err != nil {
		return nil, err
	}
	var token string
	if err := anon.PostRouteAndDecodeCtx(context.Background(), authenticationType.Route(), payload, &token, "authenticate"); err != nil {
		return nil, err
	}
	return newFromStoreConfig(storeConfig, append(append([]ClientOption(nil), opts...), WithToken(token)))
}

// NewAPIClientFromIntegration returns a client that sends bearer (an
// integration or admin token). opts are New's options.
func NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string, opts ...ClientOption) (*Client, error) {
	return newFromStoreConfig(storeConfig, append(append([]ClientOption(nil), opts...), WithToken(bearer)))
}

// GetRouteAndDecodeCtx performs a GET request on route with the given
// context, decoding the JSON response into target (which must be a pointer).
// route is relative to /rest/{storeCode}/V1 and may carry a query string
// ("/orders?searchCriteria..."); new code should use DoJSON.
func (c *Client) GetRouteAndDecodeCtx(ctx context.Context, route string, target any, tryTo string) error {
	return c.routeAndDecode(ctx, http.MethodGet, route, nil, target, tryTo)
}

// PostRouteAndDecodeCtx performs a POST request on route with the given
// context, decoding the JSON response into target (which must be a pointer).
func (c *Client) PostRouteAndDecodeCtx(ctx context.Context, route string, body, target any, tryTo string) error {
	return c.routeAndDecode(ctx, http.MethodPost, route, body, target, tryTo)
}

func (c *Client) GetRouteAndDecode(route string, target any, tryTo string) error {
	return c.GetRouteAndDecodeCtx(context.Background(), route, target, tryTo)
}

func (c *Client) PostRouteAndDecode(route string, body, target any, tryTo string) error {
	return c.PostRouteAndDecodeCtx(context.Background(), route, body, target, tryTo)
}

func (c *Client) routeAndDecode(ctx context.Context, method, route string, body, target any, tryTo string) error {
	if target == nil || reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("%w", ErrNoPointer)
	}
	_, err := c.v1(ctx, method, route, body, target, tryTo)
	return err
}

// v1 is the compat call path used by the service functions: route is
// relative to /V1 and may carry a query string; target (may be nil)
// receives the decoded body, which is also returned raw. A non-404 APIError
// is wrapped with tryTo as before.
func (c *Client) v1(ctx context.Context, method, route string, body, target any, tryTo string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req := Request{Method: method, Body: body}
	path, rawQuery, _ := strings.Cut(route, "?")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req.Path = "/V1" + path
	if rawQuery != "" {
		q, err := url.ParseQuery(rawQuery)
		if err != nil {
			return nil, fmt.Errorf("magento2: %s %s: invalid query: %w", method, req.Path, err)
		}
		req.Query = q
	}
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, wrapTryTo(err, tryTo)
	}
	if target != nil {
		if resp.Truncated {
			return nil, fmt.Errorf("magento2: %s %s: %w", method, c.redact(req.Path), ErrBodyTruncated)
		}
		if err := decodeJSON(resp.Body, target); err != nil {
			return nil, fmt.Errorf("magento2: %s %s: decode response: %w", method, c.redact(req.Path), err)
		}
	}
	return resp.Body, nil
}

// wrapTryTo keeps the historical shape: a 404 *APIError is returned bare,
// any other failure is prefixed with what the caller tried to do.
func wrapTryTo(err error, tryTo string) error {
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
		return err
	}
	if tryTo == "" {
		return err
	}
	return fmt.Errorf("error while trying to %s: %w", tryTo, err)
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
