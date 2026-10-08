package magento2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordingRT is a RoundTripper that records every request it sees and
// answers with respond (or a 200 "{}" when respond is nil).
type recordingRT struct {
	mu      sync.Mutex
	reqs    []*http.Request
	bodies  [][]byte
	respond func(*http.Request) (*http.Response, error)
}

func (rt *recordingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
	}
	rt.mu.Lock()
	rt.reqs = append(rt.reqs, r)
	rt.bodies = append(rt.bodies, body)
	rt.mu.Unlock()
	if rt.respond != nil {
		return rt.respond(r)
	}
	return textResponse(r, http.StatusOK, "{}"), nil
}

func (rt *recordingRT) count() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return len(rt.reqs)
}

func (rt *recordingRT) last() *http.Request {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if len(rt.reqs) == 0 {
		return nil
	}
	return rt.reqs[len(rt.reqs)-1]
}

func textResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}
}

func mustNew(t *testing.T, base string, opts ...ClientOption) *Client {
	t.Helper()
	c, err := New(base, opts...)
	if err != nil {
		t.Fatalf("New(%q): %v", base, err)
	}
	return c
}

// rtClient returns a client whose requests go to rt via a supplied
// *http.Client.
func rtClient(t *testing.T, rt *recordingRT, opts ...ClientOption) *Client {
	t.Helper()
	return mustNew(t, "https://shop.example", append([]ClientOption{WithHTTPClient(&http.Client{Transport: rt})}, opts...)...)
}

// captureHandler is a slog.Handler that keeps every record, at every level.
type captureHandler struct {
	mu    *sync.Mutex
	recs  *[]string
	attrs []slog.Attr
	group string
}

func newCaptureLogger() (*slog.Logger, func() []string) {
	h := &captureHandler{mu: &sync.Mutex{}, recs: &[]string{}}
	return slog.New(h), func() []string {
		h.mu.Lock()
		defer h.mu.Unlock()
		return append([]string(nil), *h.recs...)
	}
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Level.String() + " " + r.Message)
	for _, a := range h.attrs {
		b.WriteString(" " + h.group + a.String())
	}
	r.Attrs(func(a slog.Attr) bool {
		b.WriteString(" " + h.group + a.Key + "=" + a.Value.Resolve().String())
		return true
	})
	h.mu.Lock()
	*h.recs = append(*h.recs, b.String())
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr(nil), h.attrs...), as...)
	return &c
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.group += name + "."
	return &c
}

func TestSuppliedHTTPClientUsedAsGiven(t *testing.T) {
	rt := &recordingRT{}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	checkRedirect := func(*http.Request, []*http.Request) error { return nil }
	hc := &http.Client{Transport: rt, Jar: jar, CheckRedirect: checkRedirect}

	assertUnchanged := func(when string) {
		t.Helper()
		if hc.Transport != rt {
			t.Errorf("%s: Transport replaced", when)
		}
		if hc.Jar != jar {
			t.Errorf("%s: Jar replaced", when)
		}
		if reflect.ValueOf(hc.CheckRedirect).Pointer() != reflect.ValueOf(checkRedirect).Pointer() {
			t.Errorf("%s: CheckRedirect replaced", when)
		}
		if hc.Timeout != 0 {
			t.Errorf("%s: Timeout = %v, want untouched 0", when, hc.Timeout)
		}
	}

	c := mustNew(t, "https://shop.example", WithHTTPClient(hc), WithToken("tok"))
	assertUnchanged("after New")
	c.SetTimeout(time.Second)
	c.SetRetryPolicy(0, time.Millisecond, time.Millisecond)
	assertUnchanged("after SetTimeout")

	if _, err := c.Do(context.Background(), Request{Path: "/V1/orders"}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	assertUnchanged("after Do")
	if rt.count() != 1 {
		t.Fatalf("RoundTripper saw %d requests, want 1", rt.count())
	}
	got := rt.last()
	if got.URL.String() != "https://shop.example/rest/default/V1/orders" {
		t.Errorf("URL = %q", got.URL.String())
	}
	if got.Header.Get("Authorization") != "Bearer tok" {
		t.Errorf("Authorization = %q", got.Header.Get("Authorization"))
	}
}

func TestDefaultOnlyOptionsRefusedWithSuppliedClient(t *testing.T) {
	for name, opt := range map[string]ClientOption{
		"WithTimeout":         WithTimeout(time.Second),
		"WithFollowRedirects": WithFollowRedirects(false),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New("https://shop.example", WithHTTPClient(&http.Client{}), opt); err == nil {
				t.Fatalf("New accepted %s together with WithHTTPClient", name)
			}
		})
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		base string
		opts []ClientOption
	}{
		{name: "empty base", base: ""},
		{name: "relative base", base: "/shop"},
		{name: "unsupported scheme", base: "ftp://shop.example"},
		{name: "base without host", base: "https://"},
		{name: "base with query", base: "https://shop.example/?a=b"},
		{name: "base with fragment", base: "https://shop.example/#x"},
		{name: "base with user info", base: "https://u:p@shop.example"},
		{name: "user agent with newline", base: "https://shop.example", opts: []ClientOption{WithUserAgent("a\r\nX-Evil: 1")}},
		{name: "empty user agent", base: "https://shop.example", opts: []ClientOption{WithUserAgent("")}},
		{name: "negative body cap", base: "https://shop.example", opts: []ClientOption{WithMaxBodyBytes(-1)}},
		{name: "negative retry count", base: "https://shop.example", opts: []ClientOption{WithRetryPolicy(-1, 0, 0)}},
		{name: "nil http client", base: "https://shop.example", opts: []ClientOption{WithHTTPClient(nil)}},
		{name: "invalid store code", base: "https://shop.example", opts: []ClientOption{WithStoreCode("de/../admin")}},
		{name: "empty store code", base: "https://shop.example", opts: []ClientOption{WithStoreCode("")}},
		{name: "invalid allowed method", base: "https://shop.example", opts: []ClientOption{WithAllowedMethods("GET POST")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if c, err := New(tt.base, tt.opts...); err == nil {
				t.Fatalf("New(%q) = %+v, want an error", tt.base, c)
			}
		})
	}
}

func TestURLJoin(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		opts      []ClientOption
		req       Request
		wantPath  string // EscapedPath as sent
		wantQuery string // RawQuery as sent
		wantErr   bool
	}{
		{name: "store root", base: "https://shop.example", req: Request{Path: "/V1/orders"}, wantPath: "/rest/default/V1/orders"},
		{name: "root with trailing slash", base: "https://shop.example/", req: Request{Path: "/V1/orders"}, wantPath: "/rest/default/V1/orders"},
		{name: "base with path prefix", base: "https://shop.example/magento", req: Request{Path: "/V1/orders"}, wantPath: "/magento/rest/default/V1/orders"},
		{name: "prefix with trailing slash", base: "https://shop.example/sub/dir/", req: Request{Path: "/V1/orders"}, wantPath: "/sub/dir/rest/default/V1/orders"},
		{name: "client store code", base: "https://shop.example", opts: []ClientOption{WithStoreCode("de")}, req: Request{Path: "/V1/orders"}, wantPath: "/rest/de/V1/orders"},
		{name: "request store code overrides client", base: "https://shop.example", opts: []ClientOption{WithStoreCode("de")}, req: Request{Path: "/V1/orders", StoreCode: "fr"}, wantPath: "/rest/fr/V1/orders"},
		{name: "schema under all", base: "https://shop.example", req: Request{Path: "/schema", StoreCode: "all"}, wantPath: "/rest/all/schema"},
		{name: "escaped segments preserved", base: "https://shop.example", req: Request{Path: "/V1/products/a%2Fb%20c"}, wantPath: "/rest/default/V1/products/a%2Fb%20c"},
		{name: "escaped plus and percent preserved", base: "https://shop.example", req: Request{Path: "/V1/products/x%2By%25z"}, wantPath: "/rest/default/V1/products/x%2By%25z"},
		{name: "query encoded", base: "https://shop.example", req: Request{Path: "/V1/orders", Query: url.Values{"searchCriteria[pageSize]": {"5"}}}, wantPath: "/rest/default/V1/orders", wantQuery: "searchCriteria%5BpageSize%5D=5"},
		{name: "fields set once", base: "https://shop.example", req: Request{Path: "/V1/orders", Query: url.Values{"fields": {"a", "b"}}, Fields: "items[sku]"}, wantPath: "/rest/default/V1/orders", wantQuery: "fields=items%5Bsku%5D"},
		{name: "path without leading slash", base: "https://shop.example", req: Request{Path: "V1/orders"}, wantErr: true},
		{name: "empty path", base: "https://shop.example", req: Request{}, wantErr: true},
		{name: "path with query", base: "https://shop.example", req: Request{Path: "/V1/orders?x=1"}, wantErr: true},
		{name: "path with fragment", base: "https://shop.example", req: Request{Path: "/V1/orders#x"}, wantErr: true},
		{name: "dot-dot segment", base: "https://shop.example", req: Request{Path: "/V1/../../admin"}, wantErr: true},
		{name: "dot segment", base: "https://shop.example", req: Request{Path: "/V1/./orders"}, wantErr: true},
		{name: "control character", base: "https://shop.example", req: Request{Path: "/V1/ord\ners"}, wantErr: true},
		{name: "invalid escape", base: "https://shop.example", req: Request{Path: "/V1/a%zz"}, wantErr: true},
		{name: "raw non-ASCII", base: "https://shop.example", req: Request{Path: "/V1/products/é%2F"}, wantErr: true},
		{name: "unescaped brace", base: "https://shop.example", req: Request{Path: "/V1/products/{sku}%2F"}, wantErr: true},
		{name: "invalid request store code", base: "https://shop.example", req: Request{Path: "/V1/orders", StoreCode: "a/b"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{}
			c := mustNew(t, tt.base, append([]ClientOption{WithHTTPClient(&http.Client{Transport: rt})}, tt.opts...)...)
			_, err := c.Do(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Do(%+v) succeeded, want an error", tt.req)
				}
				if rt.count() != 0 {
					t.Fatalf("a refused request reached the RoundTripper")
				}
				return
			}
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			got := rt.last()
			if got.URL.EscapedPath() != tt.wantPath {
				t.Errorf("path = %q, want %q", got.URL.EscapedPath(), tt.wantPath)
			}
			if got.URL.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.RawQuery, tt.wantQuery)
			}
			if got.URL.Host != "shop.example" || got.URL.Scheme != "https" {
				t.Errorf("origin = %s://%s", got.URL.Scheme, got.URL.Host)
			}
		})
	}
}

func TestRequestQueryNotMutated(t *testing.T) {
	rt := &recordingRT{}
	c := rtClient(t, rt)
	q := url.Values{"fields": {"a"}}
	if _, err := c.Do(context.Background(), Request{Path: "/V1/orders", Query: q, Fields: "b"}); err != nil {
		t.Fatal(err)
	}
	if q.Get("fields") != "a" || len(q["fields"]) != 1 {
		t.Fatalf("caller's Query was mutated: %v", q)
	}
}

func TestRequestHeadersAndBody(t *testing.T) {
	rt := &recordingRT{}
	c := rtClient(t, rt, WithToken("tok"), WithUserAgent("embedder/1.0"))
	if _, err := c.Do(context.Background(), Request{Method: http.MethodPost, Path: "/V1/x", Body: map[string]int{"a": 1}}); err != nil {
		t.Fatal(err)
	}
	r := rt.last()
	checks := map[string]string{
		"Authorization": "Bearer tok",
		"User-Agent":    "embedder/1.0",
		"Accept":        "application/json",
		"Content-Type":  "application/json",
	}
	for k, want := range checks {
		if got := r.Header.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if r.Header.Get("Cookie") != "" {
		t.Errorf("Cookie header sent: %q", r.Header.Get("Cookie"))
	}
	if string(rt.bodies[0]) != `{"a":1}` {
		t.Errorf("body = %q", rt.bodies[0])
	}

	// Without a token no Authorization header is sent; the default user agent applies.
	rt2 := &recordingRT{}
	if _, err := rtClient(t, rt2).Do(context.Background(), Request{Path: "/V1/x"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := rt2.last().Header["Authorization"]; ok {
		t.Errorf("Authorization sent without a token: %q", v)
	}
	if got := rt2.last().Header.Get("User-Agent"); got != DefaultUserAgent {
		t.Errorf("default User-Agent = %q, want %q", got, DefaultUserAgent)
	}
	if got := rt2.last().Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q on a request without body", got)
	}
}

func TestDefaultClientKeepsNoCookies(t *testing.T) {
	var sawCookie atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			sawCookie.Store(true)
		}
		http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "x", Path: "/"})
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := mustNew(t, srv.URL)
	for i := 0; i < 2; i++ {
		if _, err := c.Do(context.Background(), Request{Path: "/V1/x"}); err != nil {
			t.Fatal(err)
		}
	}
	if sawCookie.Load() {
		t.Fatal("the default client replayed a cookie the store set")
	}
}

func TestFollowRedirects(t *testing.T) {
	tests := []struct {
		name       string
		opts       []ClientOption
		wantStatus int // 0 = success
		wantHits   int32
	}{
		{name: "not followed", opts: []ClientOption{WithFollowRedirects(false)}, wantStatus: http.StatusFound, wantHits: 0},
		{name: "followed by default", wantHits: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits atomic.Int32
			mux := http.NewServeMux()
			mux.HandleFunc("/rest/default/V1/x", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/elsewhere", http.StatusFound)
			})
			mux.HandleFunc("/elsewhere", func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				_, _ = w.Write([]byte("{}"))
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			c := mustNew(t, srv.URL, tt.opts...)
			resp, err := c.Do(context.Background(), Request{Path: "/V1/x"})
			if tt.wantStatus != 0 {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.wantStatus {
					t.Fatalf("Do = %v, %v; want *APIError with status %d", resp, err, tt.wantStatus)
				}
			} else if err != nil {
				t.Fatalf("Do: %v", err)
			}
			if got := hits.Load(); got != tt.wantHits {
				t.Fatalf("Location requested %d times, want %d", got, tt.wantHits)
			}
		})
	}
}

func TestWithTimeoutBoundsDefaultClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()
	c := mustNew(t, srv.URL, WithTimeout(100*time.Millisecond), WithRetryPolicy(0, 0, 0))
	start := time.Now()
	if _, err := c.Do(context.Background(), Request{Path: "/V1/x"}); err == nil {
		t.Fatal("want a timeout error")
	}
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("took %v; WithTimeout not applied", el)
	}
}

func TestBodyCap(t *testing.T) {
	body := strings.Repeat("a", 100)
	tests := []struct {
		name          string
		clientCap     int
		requestCap    int
		wantLen       int
		wantTruncated bool
	}{
		{name: "unlimited", wantLen: 100},
		{name: "client cap truncates", clientCap: 10, wantLen: 10, wantTruncated: true},
		{name: "exact fit is not truncated", clientCap: 100, wantLen: 100},
		{name: "request cap overrides smaller client cap", clientCap: 10, requestCap: 50, wantLen: 50, wantTruncated: true},
		{name: "request cap overrides larger client cap", clientCap: 1000, requestCap: 20, wantLen: 20, wantTruncated: true},
		{name: "request cap above body", clientCap: 10, requestCap: 1000, wantLen: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				return textResponse(r, http.StatusOK, body), nil
			}}
			c := rtClient(t, rt, WithMaxBodyBytes(tt.clientCap))
			resp, err := c.Do(context.Background(), Request{Path: "/V1/x", MaxBodyBytes: tt.requestCap})
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Body) != tt.wantLen || resp.Truncated != tt.wantTruncated {
				t.Fatalf("len(Body)=%d Truncated=%v, want %d %v", len(resp.Body), resp.Truncated, tt.wantLen, tt.wantTruncated)
			}
			if resp.Status != http.StatusOK || resp.Header.Get("Content-Type") != "application/json" || resp.Elapsed <= 0 {
				t.Fatalf("Response = %+v", resp)
			}
		})
	}
}

func TestDoJSON(t *testing.T) {
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		return textResponse(r, http.StatusOK, `{"name":"`+strings.Repeat("n", 50)+`"}`), nil
	}}
	c := rtClient(t, rt)

	var out struct{ Name string }
	err := c.DoJSON(context.Background(), Request{Path: "/V1/x", MaxBodyBytes: 10}, &out)
	if !errors.Is(err, ErrBodyTruncated) {
		t.Fatalf("DoJSON on a truncated body = %v, want ErrBodyTruncated", err)
	}
	if out.Name != "" {
		t.Fatalf("truncated body was decoded into the target: %+v", out)
	}
	if err := c.DoJSON(context.Background(), Request{Path: "/V1/x"}, &out); err != nil || len(out.Name) != 50 {
		t.Fatalf("DoJSON = %v, %+v", err, out)
	}
	if err := c.DoJSON(context.Background(), Request{Path: "/V1/x"}, nil); err != nil {
		t.Fatalf("DoJSON with a nil target = %v, want nil (body discarded)", err)
	}
}

func TestAllowedMethods(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		method  string
		wantOK  bool
	}{
		{name: "get allowed", allowed: []string{"GET"}, method: "GET", wantOK: true},
		{name: "empty method is GET", allowed: []string{"GET"}, method: "", wantOK: true},
		{name: "lower-case option normalised", allowed: []string{"get"}, method: "GET", wantOK: true},
		{name: "post refused", allowed: []string{"GET"}, method: "POST"},
		{name: "lower-case request method refused", allowed: []string{"GET"}, method: "delete"},
		{name: "no allowlist allows all", method: "DELETE", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{}
			var opts []ClientOption
			if tt.allowed != nil {
				opts = append(opts, WithAllowedMethods(tt.allowed...))
			}
			c := rtClient(t, rt, opts...)
			_, err := c.Do(context.Background(), Request{Method: tt.method, Path: "/V1/x"})
			if tt.wantOK {
				if err != nil || rt.count() != 1 {
					t.Fatalf("Do = %v, requests %d", err, rt.count())
				}
				return
			}
			if !errors.Is(err, ErrMethodNotAllowed) {
				t.Fatalf("Do = %v, want ErrMethodNotAllowed", err)
			}
			if rt.count() != 0 {
				t.Fatalf("refused method reached the RoundTripper")
			}
		})
	}
}

func TestRequestHook(t *testing.T) {
	hookErr := errors.New("refused by embedder")
	rt := &recordingRT{}
	var seen *http.Request
	c := rtClient(t, rt, WithToken("tok"), WithRequestHook(func(ctx context.Context, r *http.Request) error {
		seen = r
		return hookErr
	}))
	_, err := c.Do(context.Background(), Request{Path: "/V1/orders", Query: url.Values{"a": {"1"}}})
	if !errors.Is(err, hookErr) {
		t.Fatalf("Do = %v, want the hook's error", err)
	}
	if rt.count() != 0 {
		t.Fatal("a request the hook refused was sent")
	}
	if seen == nil || seen.URL.String() != "https://shop.example/rest/default/V1/orders?a=1" || seen.Header.Get("Authorization") != "Bearer tok" {
		t.Fatalf("hook saw %+v, want the fully built request", seen)
	}
}

// trackingBody records whether anything read from it.
type trackingBody struct {
	r      io.Reader
	read   atomic.Bool
	closed atomic.Bool
}

func (b *trackingBody) Read(p []byte) (int, error) { b.read.Store(true); return b.r.Read(p) }
func (b *trackingBody) Close() error               { b.closed.Store(true); return nil }

func TestResponseHook(t *testing.T) {
	t.Run("runs before the body is read", func(t *testing.T) {
		body := &trackingBody{r: strings.NewReader(`{"a":1}`)}
		rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
			resp := textResponse(r, http.StatusOK, "")
			resp.Body = body
			return resp, nil
		}}
		var readBeforeHook = true
		c := rtClient(t, rt, WithResponseHook(func(ctx context.Context, resp *http.Response) error {
			readBeforeHook = body.read.Load()
			return nil
		}))
		resp, err := c.Do(context.Background(), Request{Path: "/V1/x"})
		if err != nil {
			t.Fatal(err)
		}
		if readBeforeHook {
			t.Fatal("the body was read before the response hook ran")
		}
		if string(resp.Body) != `{"a":1}` || !body.closed.Load() {
			t.Fatalf("Body = %q closed=%v", resp.Body, body.closed.Load())
		}
	})
	t.Run("an error aborts", func(t *testing.T) {
		hookErr := errors.New("unexpected response")
		body := &trackingBody{r: strings.NewReader(`{"a":1}`)}
		rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
			resp := textResponse(r, http.StatusServiceUnavailable, "")
			resp.Body = body
			return resp, nil
		}}
		c := rtClient(t, rt, WithResponseHook(func(context.Context, *http.Response) error { return hookErr }))
		resp, err := c.Do(context.Background(), Request{Path: "/V1/x"})
		if !errors.Is(err, hookErr) || resp != nil {
			t.Fatalf("Do = %v, %v; want the hook's error", resp, err)
		}
		if body.read.Load() || !body.closed.Load() {
			t.Fatalf("body read=%v closed=%v; want unread and closed", body.read.Load(), body.closed.Load())
		}
		if rt.count() != 1 {
			t.Fatalf("requests = %d; a hook refusal must not be retried", rt.count())
		}
	})
}

func TestRedactorAppliedToErrorsAndLogs(t *testing.T) {
	const secret = "s3cr3t"
	redact := func(s string) string { return strings.ReplaceAll(s, secret, "[redacted]") }
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		resp := textResponse(r, http.StatusBadRequest, `{"message":"bad `+secret+` %1","parameters":["`+secret+`"]}`)
		resp.Header.Set("X-Debug", secret)
		return resp, nil
	}}
	logger, records := newCaptureLogger()
	c := rtClient(t, rt, WithRedactor(redact), WithLogger(logger))

	_, err := c.Do(context.Background(), Request{Path: "/V1/products/" + secret})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Do = %v, want *APIError", err)
	}
	for what, s := range map[string]string{
		"Error()":    apiErr.Error(),
		"err":        err.Error(),
		"Body":       string(apiErr.Body),
		"Message":    apiErr.Message,
		"Path":       apiErr.Path,
		"Parameters": string(apiErr.Parameters),
		"Header":     fmt.Sprint(apiErr.Header),
	} {
		if strings.Contains(s, secret) {
			t.Errorf("%s carries the secret: %q", what, s)
		}
	}
	if !strings.Contains(apiErr.Message, "[redacted]") {
		t.Errorf("Message = %q, want the redacted marker", apiErr.Message)
	}
	recs := records()
	if len(recs) == 0 {
		t.Fatal("no log record")
	}
	for _, r := range recs {
		if strings.Contains(r, secret) {
			t.Errorf("log record carries the secret: %q", r)
		}
	}

	// A transport error is redacted too.
	rt2 := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed for " + secret)
	}}
	c2 := rtClient(t, rt2, WithRedactor(redact), WithRetryPolicy(0, 0, 0))
	_, err = c2.Do(context.Background(), Request{Path: "/V1/x"})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("transport error = %v", err)
	}
}

func TestAPIErrorShape(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantError  string
		wantIs     error
		wantMsg    string
		wantParams string
	}{
		{
			name: "not found", status: 404, body: `{"message":"No such entity."}`,
			wantError: "magento2: GET /V1/orders: status 404: No such entity.", wantIs: ErrNotFound, wantMsg: "No such entity.",
		},
		{
			name: "not a magento document", status: 502, body: "<html>bad gateway</html>",
			wantError: "magento2: GET /V1/orders: status 502: Bad Gateway", wantIs: ErrBadRequest, wantMsg: "Bad Gateway",
		},
		{
			name: "parameters substituted and kept raw", status: 400, body: `{"message":"%1","parameters":["x"]}`,
			wantError: "magento2: GET /V1/orders: status 400: x", wantIs: ErrBadRequest, wantMsg: "x", wantParams: `["x"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				return textResponse(r, tt.status, tt.body), nil
			}}
			c := rtClient(t, rt, WithRetryPolicy(0, 0, 0))
			_, err := c.Do(context.Background(), Request{Path: "/V1/orders", Query: url.Values{"q": {"secret-query"}}})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("Do = %v", err)
			}
			if apiErr.Error() != tt.wantError {
				t.Errorf("Error() = %q, want %q", apiErr.Error(), tt.wantError)
			}
			if !errors.Is(err, tt.wantIs) {
				t.Errorf("errors.Is(%v) = false", tt.wantIs)
			}
			if apiErr.StatusCode != tt.status || apiErr.Method != "GET" || apiErr.Path != "/V1/orders" || string(apiErr.Body) != tt.body || apiErr.Message != tt.wantMsg {
				t.Errorf("APIError = %+v", apiErr)
			}
			if string(apiErr.Parameters) != tt.wantParams {
				t.Errorf("Parameters = %q, want %q", apiErr.Parameters, tt.wantParams)
			}
			if strings.Contains(err.Error(), "secret-query") || strings.Contains(err.Error(), "shop.example") {
				t.Errorf("error carries the query or the host: %q", err.Error())
			}
		})
	}
}

func TestAPIErrorBodyCap(t *testing.T) {
	big := strings.Repeat("x", 2*defaultErrorBodyBytes)
	tests := []struct {
		name      string
		clientCap int
		reqCap    int
		wantLen   int
	}{
		{name: "unlimited client uses the error default", wantLen: defaultErrorBodyBytes},
		{name: "smaller client cap wins", clientCap: 100, wantLen: 100},
		{name: "request cap wins", clientCap: 100, reqCap: 10, wantLen: 10},
		{name: "larger cap is bounded by the error default", clientCap: 10 * defaultErrorBodyBytes, wantLen: defaultErrorBodyBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				return textResponse(r, http.StatusNotFound, big), nil
			}}
			c := rtClient(t, rt, WithMaxBodyBytes(tt.clientCap))
			_, err := c.Do(context.Background(), Request{Path: "/V1/x", MaxBodyBytes: tt.reqCap})
			var apiErr *APIError
			if !errors.As(err, &apiErr) || len(apiErr.Body) != tt.wantLen {
				t.Fatalf("Do = %v (len %d), want body of %d", err, len(apiErr.Body), tt.wantLen)
			}
		})
	}
}

// The logger must never receive the Authorization value, a query string, a
// request body or a response body, at any level, whatever the outcome.
func TestLoggerNeverSeesSecrets(t *testing.T) {
	const (
		token     = "tok-ABCDEF"
		queryVal  = "qval-XYZ"
		fieldsVal = "items-fieldsval"
		reqBody   = "reqbody-123"
		respBody  = "respbody-456"
		hdrVal    = "hdrval-789"
	)
	outcomes := map[string]func(r *http.Request) (*http.Response, error){
		"200": func(r *http.Request) (*http.Response, error) {
			resp := textResponse(r, 200, `{"x":"`+respBody+`"}`)
			resp.Header.Set("X-Leak", hdrVal)
			return resp, nil
		},
		"500": func(r *http.Request) (*http.Response, error) {
			resp := textResponse(r, 500, `{"message":"`+respBody+`"}`)
			resp.Header.Set("X-Leak", hdrVal)
			return resp, nil
		},
		"transport error": func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset")
		},
	}
	for name, respond := range outcomes {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			t.Run(name+" "+method, func(t *testing.T) {
				logger, records := newCaptureLogger()
				rt := &recordingRT{respond: respond}
				c := rtClient(t, rt, WithToken(token), WithLogger(logger), WithRetryPolicy(1, time.Millisecond, time.Millisecond))
				_, _ = c.Do(context.Background(), Request{
					Method: method, Path: "/V1/orders",
					Query:  url.Values{"q": {queryVal}},
					Fields: fieldsVal,
					Body:   map[string]string{"b": reqBody},
				})
				recs := records()
				if len(recs) != rt.count() {
					t.Errorf("%d log records for %d attempts, want one per attempt", len(recs), rt.count())
				}
				for _, rec := range recs {
					for _, forbidden := range []string{token, "Bearer", queryVal, fieldsVal, reqBody, respBody, hdrVal, "?", "shop.example"} {
						if strings.Contains(rec, forbidden) {
							t.Errorf("log record carries %q: %s", forbidden, rec)
						}
					}
					for _, key := range []string{"method=", "path=/V1/orders", "elapsed=", "bytes=", "truncated="} {
						if !strings.Contains(rec, key) {
							t.Errorf("log record lacks %q: %s", key, rec)
						}
					}
				}
			})
		}
	}
}

func TestNilLoggerDiscards(t *testing.T) {
	c := rtClient(t, &recordingRT{}, WithLogger(nil))
	if _, err := c.Do(context.Background(), Request{Path: "/V1/x"}); err != nil {
		t.Fatal(err)
	}
}

func TestTokenNotPrinted(t *testing.T) {
	const token = "tok-PRINT-ME-NOT"
	c := mustNew(t, "https://shop.example", WithToken(token))
	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		if s := fmt.Sprintf(verb, c); strings.Contains(s, token) {
			t.Errorf("%s of the client shows the token: %s", verb, s)
		}
	}
}

func TestRetryPolicy(t *testing.T) {
	tests := []struct {
		name      string
		opts      []ClientOption
		method    string
		statuses  []int // answered in order; the last repeats
		wantReqs  int
		wantError bool
	}{
		{name: "count 0 is one attempt", opts: []ClientOption{WithRetryPolicy(0, time.Millisecond, time.Millisecond)}, statuses: []int{503}, wantReqs: 1, wantError: true},
		{name: "default policy retries 503 then 200", statuses: []int{503, 200}, wantReqs: 2},
		{name: "retries exhausted", opts: []ClientOption{WithRetryPolicy(2, time.Millisecond, 5*time.Millisecond)}, statuses: []int{500}, wantReqs: 3, wantError: true},
		{name: "POST never retried", method: http.MethodPost, statuses: []int{503}, wantReqs: 1, wantError: true},
		{name: "PUT never retried", method: http.MethodPut, statuses: []int{502}, wantReqs: 1, wantError: true},
		{name: "DELETE retried", method: http.MethodDelete, opts: []ClientOption{WithRetryPolicy(1, time.Millisecond, time.Millisecond)}, statuses: []int{504, 200}, wantReqs: 2},
		{name: "404 not retried", statuses: []int{404}, wantReqs: 1, wantError: true},
		{name: "429 retried", opts: []ClientOption{WithRetryPolicy(1, time.Millisecond, time.Millisecond)}, statuses: []int{429, 200}, wantReqs: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n atomic.Int32
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				i := int(n.Add(1)) - 1
				if i >= len(tt.statuses) {
					i = len(tt.statuses) - 1
				}
				return textResponse(r, tt.statuses[i], "{}"), nil
			}}
			c := rtClient(t, rt, tt.opts...)
			_, err := c.Do(context.Background(), Request{Method: tt.method, Path: "/V1/x", Body: bodyFor(tt.method)})
			if (err != nil) != tt.wantError {
				t.Fatalf("Do error = %v, wantError %v", err, tt.wantError)
			}
			if rt.count() != tt.wantReqs {
				t.Fatalf("requests = %d, want %d", rt.count(), tt.wantReqs)
			}
		})
	}
}

func bodyFor(method string) any {
	if method == http.MethodPost || method == http.MethodPut {
		return map[string]int{"a": 1}
	}
	return nil
}

// Every attempt re-sends the full request body.
func TestRetryResendsBody(t *testing.T) {
	var n atomic.Int32
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		if n.Add(1) == 1 {
			return textResponse(r, 503, ""), nil
		}
		return textResponse(r, 200, "{}"), nil
	}}
	c := rtClient(t, rt, WithRetryPolicy(1, time.Millisecond, time.Millisecond))
	if _, err := c.Do(context.Background(), Request{Method: http.MethodDelete, Path: "/V1/x", Body: []int{1, 2}}); err != nil {
		t.Fatal(err)
	}
	for i, b := range rt.bodies {
		if !bytes.Equal(b, []byte("[1,2]")) {
			t.Errorf("attempt %d body = %q", i, b)
		}
	}
}

func TestRetryAfterHonoured(t *testing.T) {
	tests := []struct {
		name  string
		value func() string
	}{
		{name: "delta seconds", value: func() string { return "1" }},
		{name: "http date", value: func() string { return time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n atomic.Int32
			rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
				if n.Add(1) == 1 {
					resp := textResponse(r, http.StatusTooManyRequests, "")
					resp.Header.Set("Retry-After", tt.value())
					return resp, nil
				}
				return textResponse(r, 200, "{}"), nil
			}}
			c := rtClient(t, rt, WithRetryPolicy(1, time.Millisecond, time.Minute))
			start := time.Now()
			if _, err := c.Do(context.Background(), Request{Path: "/V1/x"}); err != nil {
				t.Fatal(err)
			}
			if el := time.Since(start); el < 900*time.Millisecond || el > 10*time.Second {
				t.Fatalf("retried after %v, want ~1-2s (Retry-After)", el)
			}
		})
	}
}

func TestRetryAfterCappedAtMaxWait(t *testing.T) {
	var n atomic.Int32
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		if n.Add(1) == 1 {
			resp := textResponse(r, http.StatusServiceUnavailable, "")
			resp.Header.Set("Retry-After", "3600")
			return resp, nil
		}
		return textResponse(r, 200, "{}"), nil
	}}
	c := rtClient(t, rt, WithRetryPolicy(1, time.Millisecond, 50*time.Millisecond))
	start := time.Now()
	if _, err := c.Do(context.Background(), Request{Path: "/V1/x"}); err != nil {
		t.Fatal(err)
	}
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("waited %v; Retry-After must be capped at the max wait", el)
	}
}

func TestContextCancelDuringBackoff(t *testing.T) {
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		resp := textResponse(r, http.StatusTooManyRequests, "")
		resp.Header.Set("Retry-After", "5")
		return resp, nil
	}}
	c := rtClient(t, rt)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Do(ctx, Request{Path: "/V1/x"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Do = %v, want context.DeadlineExceeded", err)
	}
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("cancellation during backoff took %v", el)
	}
	if rt.count() != 1 {
		t.Fatalf("requests = %d, want 1", rt.count())
	}
}

func TestBackoffBounds(t *testing.T) {
	p := retryPolicy{count: 10, wait: 100 * time.Millisecond, maxWait: time.Second}
	for attempt := 1; attempt <= 10; attempt++ {
		for i := 0; i < 50; i++ {
			d := p.backoff(attempt, 0)
			if d < 0 || d > p.maxWait {
				t.Fatalf("backoff(%d) = %v, outside [0, %v]", attempt, d, p.maxWait)
			}
		}
	}
	if d := (retryPolicy{wait: 0, maxWait: 0}).backoff(1, 0); d != 0 {
		t.Fatalf("zero policy backoff = %v", d)
	}
}
