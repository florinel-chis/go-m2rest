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
		name   string
		config StoreConfig
		want   string
	}{
		{
			name:   "plain host",
			config: StoreConfig{Scheme: "https", HostName: "shop.example.com", StoreCode: "default"},
			want:   "https://shop.example.com/rest/default/V1",
		},
		{
			name:   "host with port",
			config: StoreConfig{Scheme: "http", HostName: "localhost:8080", StoreCode: "default"},
			want:   "http://localhost:8080/rest/default/V1",
		},
		{
			name:   "base path",
			config: StoreConfig{Scheme: "https", HostName: "example.com", StoreCode: "de", BasePath: "shop"},
			want:   "https://example.com/shop/rest/de/V1",
		},
		{
			name:   "base path with surrounding slashes",
			config: StoreConfig{Scheme: "https", HostName: "example.com", StoreCode: "default", BasePath: "/shop/"},
			want:   "https://example.com/shop/rest/default/V1",
		},
		{
			name:   "multi-segment base path and port",
			config: StoreConfig{Scheme: "http", HostName: "example.com:8443", StoreCode: "default", BasePath: "sub/dir"},
			want:   "http://example.com:8443/sub/dir/rest/default/V1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAPIClientWithoutAuthentication(&tt.config)
			if got := client.HTTPClient.BaseURL; got != tt.want {
				t.Errorf("base URL = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultTimeoutAndSetTimeout(t *testing.T) {
	client := NewAPIClientWithoutAuthentication(&StoreConfig{Scheme: "http", HostName: "example.com", StoreCode: "default"})
	if got := client.HTTPClient.GetClient().Timeout; got != DefaultTimeout {
		t.Errorf("default timeout = %v, want %v", got, DefaultTimeout)
	}
	client.SetTimeout(5 * time.Second)
	if got := client.HTTPClient.GetClient().Timeout; got != 5*time.Second {
		t.Errorf("timeout after SetTimeout = %v, want 5s", got)
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
	if !strings.Contains(apiErr.Body, "boom") {
		t.Errorf("APIError.Body = %q, want it to contain the response body", apiErr.Body)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("errors.Is(err, ErrBadRequest) = false, want true (historical behavior)")
	}
}

func TestNotFoundErrorSentinelAndAPIError(t *testing.T) {
	longBody := strings.Repeat("x", 1000)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, longBody, http.StatusNotFound)
	})

	client := newTestClient(t, handler)
	_, err := GetProductsPage(context.Background(), client, ListOptions{})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
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
	if len(apiErr.Body) != maxErrorBodyBytes {
		t.Errorf("APIError.Body length = %d, want truncated to %d", len(apiErr.Body), maxErrorBodyBytes)
	}
	if apiErr.Endpoint == "" || !strings.Contains(apiErr.Endpoint, "/products") {
		t.Errorf("APIError.Endpoint = %q, want it to reference the endpoint", apiErr.Endpoint)
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
