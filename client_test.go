package magento2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient spins up an httptest server and returns a client whose base
// URL points at it (store code "default", so routes live under
// /rest/default/V1).
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	return NewAPIClientWithoutAuthentication(&StoreConfig{
		Scheme:    "http",
		HostName:  parsed.Host,
		StoreCode: "default",
	})
}

func TestBaseURLBuilding(t *testing.T) {
	tests := []struct {
		name    string
		config  StoreConfig
		want    string // StoreConfig.baseURL()
		wantURL string // URL requested for the compat route "/products"
	}{
		{
			name:    "plain host",
			config:  StoreConfig{Scheme: "https", HostName: "shop.example.com", StoreCode: "default"},
			want:    "https://shop.example.com/rest/default/V1",
			wantURL: "https://shop.example.com/rest/default/V1/products",
		},
		{
			name:    "empty store code sends no segment",
			config:  StoreConfig{Scheme: "https", HostName: "shop.example.com"},
			want:    "https://shop.example.com/rest/V1",
			wantURL: "https://shop.example.com/rest/V1/products",
		},
		{
			name:    "host with port",
			config:  StoreConfig{Scheme: "http", HostName: "localhost:8080", StoreCode: "default"},
			want:    "http://localhost:8080/rest/default/V1",
			wantURL: "http://localhost:8080/rest/default/V1/products",
		},
		{
			name:    "base path",
			config:  StoreConfig{Scheme: "https", HostName: "example.com", StoreCode: "de", BasePath: "shop"},
			want:    "https://example.com/shop/rest/de/V1",
			wantURL: "https://example.com/shop/rest/de/V1/products",
		},
		{
			name:    "base path with surrounding slashes",
			config:  StoreConfig{Scheme: "https", HostName: "example.com", StoreCode: "default", BasePath: "/shop/"},
			want:    "https://example.com/shop/rest/default/V1",
			wantURL: "https://example.com/shop/rest/default/V1/products",
		},
		{
			name:    "multi-segment base path and port",
			config:  StoreConfig{Scheme: "http", HostName: "example.com:8443", StoreCode: "default", BasePath: "sub/dir"},
			want:    "http://example.com:8443/sub/dir/rest/default/V1",
			wantURL: "http://example.com:8443/sub/dir/rest/default/V1/products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.baseURL(); got != tt.want {
				t.Errorf("base URL = %q, want %q", got, tt.want)
			}
			// The (unexported) baseURL is only a description; what counts is
			// the URL the compat helpers actually request.
			rt := &recordingRT{}
			client := NewAPIClientWithoutAuthentication(&tt.config, WithHTTPClient(&http.Client{Transport: rt}))
			var target map[string]any
			if err := client.GetRouteAndDecodeCtx(context.Background(), "/products", &target, "base url test"); err != nil {
				t.Fatalf("GetRouteAndDecodeCtx: %v", err)
			}
			if got := rt.last().URL.String(); got != tt.wantURL {
				t.Errorf("requested URL = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

// An invalid StoreConfig cannot fail NewAPIClientWithoutAuthentication (its
// signature has no error); the error surfaces on the first request instead.
func TestCompatConstructorInvalidConfigFailsOnUse(t *testing.T) {
	client := NewAPIClientWithoutAuthentication(&StoreConfig{Scheme: "ftp", HostName: "example.com", StoreCode: "default"})
	var target map[string]any
	if err := client.GetRouteAndDecodeCtx(context.Background(), "/products", &target, "x"); err == nil {
		t.Fatal("want an error from a client built from an invalid StoreConfig")
	}
	if _, err := NewAPIClientFromIntegration(&StoreConfig{Scheme: "ftp", HostName: "example.com"}, "tok"); err == nil {
		t.Fatal("NewAPIClientFromIntegration accepted an invalid StoreConfig")
	}
}

func TestCompatRouteWithQuery(t *testing.T) {
	rt := &recordingRT{}
	client := NewAPIClientWithoutAuthentication(&StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"}, WithHTTPClient(&http.Client{Transport: rt}))
	var target map[string]any
	route := "/orders?" + BuildSearchQuery("increment_id", "0001", "eq")
	if err := client.GetRouteAndDecodeCtx(context.Background(), route, &target, "x"); err != nil {
		t.Fatal(err)
	}
	got := rt.last().URL
	if got.Path != "/rest/default/V1/orders" || got.Query().Get("searchCriteria[filter_groups][0][filters][0][value]") != "0001" {
		t.Fatalf("requested %s", got)
	}
	if err := client.GetRouteAndDecodeCtx(context.Background(), "/orders", target, "x"); !errors.Is(err, ErrNoPointer) {
		t.Fatalf("non-pointer target = %v, want ErrNoPointer", err)
	}
	if err := client.GetRouteAndDecodeCtx(context.Background(), "/orders", nil, "x"); !errors.Is(err, ErrNoPointer) {
		t.Fatalf("nil target = %v, want ErrNoPointer", err)
	}
}

func TestNewAPIClientFromIntegrationSendsToken(t *testing.T) {
	rt := &recordingRT{}
	client, err := NewAPIClientFromIntegration(&StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"}, "integration-token", WithHTTPClient(&http.Client{Transport: rt}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GetStoreViews(context.Background(), client); err == nil {
		// "{}" does not decode into a slice; only the header matters here.
		t.Log("decoded")
	}
	if got := rt.last().Header.Get("Authorization"); got != "Bearer integration-token" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestNewAPIClientFromAuthentication(t *testing.T) {
	tests := []struct {
		name     string
		authType AuthenticationType
		wantPath string
		status   int
		wantErr  bool
	}{
		{name: "admin", authType: Administrator, wantPath: "/rest/default/V1/integration/admin/token", status: 200},
		{name: "customer", authType: CustomerAuth, wantPath: "/rest/default/V1/integration/customer/token", status: 200},
		{name: "refused", authType: Administrator, wantPath: "/rest/default/V1/integration/admin/token", status: 401, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == tt.wantPath {
					if tt.status != 200 {
						return textResponse(r, tt.status, `{"message":"The account sign-in was incorrect."}`), nil
					}
					return textResponse(r, 200, `"issued-token"`), nil
				}
				return textResponse(r, 200, `[]`), nil
			}}
			cfg := &StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"}
			client, err := NewAPIClientFromAuthentication(cfg, AuthenticationRequestPayload{Username: "u", Password: "p"}, tt.authType, WithHTTPClient(&http.Client{Transport: rt}))
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error for a refused authentication")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if rt.last().Method != http.MethodPost || rt.last().Header.Get("Authorization") != "" {
				t.Fatalf("token request = %s %s", rt.last().Method, rt.last().Header)
			}
			if string(rt.bodies[0]) != `{"username":"u","password":"p"}` {
				t.Fatalf("payload = %s", rt.bodies[0])
			}
			if _, err := GetStoreViews(context.Background(), client); err != nil {
				t.Fatal(err)
			}
			if got := rt.last().Header.Get("Authorization"); got != "Bearer issued-token" {
				t.Fatalf("Authorization = %q", got)
			}
		})
	}
}

func TestDefaultTimeoutAndSetTimeout(t *testing.T) {
	// SetTimeout must actually bound request duration: against a server
	// slower than the configured timeout the request fails promptly instead
	// of blocking (retries disabled — a timed-out attempt is a transport
	// error and would otherwise be retried).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	slow := newTestClient(t, handler)
	slow.SetRetryPolicy(0, time.Millisecond, time.Millisecond)
	slow.SetTimeout(100 * time.Millisecond)

	start := time.Now()
	var target map[string]any
	err := slow.GetRouteAndDecodeCtx(context.Background(), "products", &target, "timeout test")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error from slow server, got nil")
	}
	if elapsed > 2*time.Second {
		t.Errorf("request against slow server took %v, want ~100ms (SetTimeout not applied)", elapsed)
	}
}

func TestRetryOn429HonorsRetryAfter(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"sku":"p1","name":"P1"}],"total_count":1}`))
	})

	client := newTestClient(t, handler)
	start := time.Now()
	page, err := GetProductsPage(context.Background(), client, ListOptions{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetProductsPage returned error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
	if len(page.Items) != 1 || page.Items[0].Sku != "p1" {
		t.Errorf("unexpected page after retry: %+v", page)
	}
	// Retry-After: 1 means the client should have waited about a second;
	// assert loosely in both directions.
	if elapsed < 900*time.Millisecond {
		t.Errorf("retry happened after %v, expected at least ~1s (Retry-After honored)", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Errorf("retry took unexpectedly long: %v", elapsed)
	}
}

func TestRetriesExhaustedReturnsAPIError(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})

	client := newTestClient(t, handler)
	client.SetRetryPolicy(2, 5*time.Millisecond, 20*time.Millisecond)

	_, err := GetProductsPage(context.Background(), client, ListOptions{})
	if err == nil {
		t.Fatal("expected error after exhausted retries, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3 (1 initial + 2 retries)", got)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As(*APIError) failed for %v", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("APIError.StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if !strings.Contains(string(apiErr.Body), "boom") {
		t.Errorf("APIError.Body = %q, want it to contain the response body", apiErr.Body)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("errors.Is(err, ErrBadRequest) = false, want true (historical behavior)")
	}
}

func TestNotFoundErrorSentinelAndAPIError(t *testing.T) {
	longBody := strings.Repeat("x", 2*defaultErrorBodyBytes)
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, longBody, http.StatusNotFound)
	})

	client := newTestClient(t, handler)
	_, err := GetProductsPage(context.Background(), client, ListOptions{})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (404 must not be retried)", got)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false, want true")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("errors.As(*APIError) failed for %v", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError.StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if len(apiErr.Body) != defaultErrorBodyBytes {
		t.Errorf("APIError.Body length = %d, want truncated to %d", len(apiErr.Body), defaultErrorBodyBytes)
	}
	if apiErr.Method != http.MethodGet || apiErr.Path != "/V1/products" {
		t.Errorf("APIError Method/Path = %q %q, want GET /V1/products", apiErr.Method, apiErr.Path)
	}
}

// The Retry-After header may also be an HTTP-date (RFC 9110); it must be
// honored just like the delta-seconds form.
func TestRetryOn429HonorsRetryAfterHTTPDate(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", time.Now().Add(2*time.Second).UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"sku":"p1","name":"P1"}],"total_count":1}`))
	})

	client := newTestClient(t, handler)
	start := time.Now()
	page, err := GetProductsPage(context.Background(), client, ListOptions{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetProductsPage returned error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
	if len(page.Items) != 1 {
		t.Errorf("unexpected page after retry: %+v", page)
	}
	// The date is ~2s in the future (HTTP-dates have second granularity, so
	// the effective wait is 1-2s); the default first jittered backoff would
	// be well under 500ms.
	if elapsed < 900*time.Millisecond {
		t.Errorf("retry happened after %v, expected at least ~1s (HTTP-date Retry-After honored)", elapsed)
	}
}

// Non-idempotent methods must not be retried automatically: a 502 on order
// placement may arrive after the backend committed the order.
func TestPostNotRetriedOn5xx(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, `{"message":"bad gateway"}`, http.StatusBadGateway)
	})

	client := newTestClient(t, handler)
	var target map[string]any
	err := client.PostRouteAndDecodeCtx(context.Background(), "orders", map[string]string{"k": "v"}, &target, "test post")
	if err == nil {
		t.Fatal("expected error for 502, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (POST must not be retried)", got)
	}
}

// Transport-level failures (connection drop, per-attempt timeout) must be
// retried for idempotent requests.
func TestTransportErrorRetriedForGET(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			// Drop the connection mid-request to force a transport error.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("test server does not support hijacking")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"sku":"p1","name":"P1"}],"total_count":1}`))
	})

	client := newTestClient(t, handler)
	client.SetRetryPolicy(2, 5*time.Millisecond, 20*time.Millisecond)

	page, err := GetProductsPage(context.Background(), client, ListOptions{})
	if err != nil {
		t.Fatalf("GetProductsPage returned error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts = %d, want 2 (transport error retried once)", got)
	}
	if len(page.Items) != 1 || page.Items[0].Sku != "p1" {
		t.Errorf("unexpected page after retry: %+v", page)
	}
}

// Cancellation must also abort promptly while the client is sleeping
// between retries (here: a 5s Retry-After against a 100ms deadline).
func TestContextCancellationDuringRetryBackoff(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	client := newTestClient(t, handler)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := GetProductsPage(ctx, client, ListOptions{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("errors.Is(err, context.DeadlineExceeded) = false, got err: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("cancellation during backoff took %v, expected prompt abort", elapsed)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry after context expiry)", got)
	}
}

func TestContextCancellationAbortsPromptly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			_, _ = w.Write([]byte(`{"items":[],"total_count":0}`))
		}
	})

	client := newTestClient(t, handler)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := GetProductsPage(ctx, client, ListOptions{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("errors.Is(err, context.DeadlineExceeded) = false, got err: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("cancellation took %v, expected prompt abort", elapsed)
	}
}
