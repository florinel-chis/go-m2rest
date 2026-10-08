package magento2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Defaults applied by New and the compat constructors.
const (
	// DefaultTimeout is the per-attempt timeout of the default *http.Client.
	DefaultTimeout = 30 * time.Second
	// DefaultRetryCount is the number of retries after the initial attempt
	// (i.e. 5 total attempts).
	DefaultRetryCount = 4
	// DefaultRetryWaitTime is the base wait time between retries.
	DefaultRetryWaitTime = 500 * time.Millisecond
	// DefaultRetryMaxWaitTime caps the wait between retries, including a
	// server-provided Retry-After. It is deliberately generous so a rate
	// limiter's Retry-After is honoured; the computed jittered backoff stays
	// far below it with the default policy (500ms base, 4 retries gives at
	// most ~4s).
	DefaultRetryMaxWaitTime = 2 * time.Minute
	// DefaultStoreCode is the store code used when neither WithStoreCode nor
	// Request.StoreCode names one.
	DefaultStoreCode = "default"
	// DefaultUserAgent is sent unless WithUserAgent overrides it.
	DefaultUserAgent = "go-m2rest"
)

// defaultErrorBodyBytes bounds APIError.Body: the effective body cap
// (Request.MaxBodyBytes, else WithMaxBodyBytes) applies when it is smaller,
// and this default applies when the cap is larger or unlimited.
const defaultErrorBodyBytes = 64 << 10

// ErrBodyTruncated is returned by DoJSON when the response body exceeded the
// body cap: a truncated body is never decoded.
var ErrBodyTruncated = errors.New("magento2: response body truncated")

// ErrMethodNotAllowed is returned, before anything is sent, for a request
// whose method is outside WithAllowedMethods.
var ErrMethodNotAllowed = errors.New("magento2: method not allowed")

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

// storeCodeRe is what a store code may look like: it becomes a path segment.
var storeCodeRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// methodRe is an HTTP method token.
var methodRe = regexp.MustCompile(`^[A-Za-z]+$`)

// Client talks to one Magento store over REST. It is safe for concurrent
// use. Build it with New (or a compat constructor); its zero value is not
// usable.
type Client struct {
	base      string // scheme://host[/prefix], no trailing slash
	http      atomic.Pointer[http.Client]
	owned     bool // http was built by New, not supplied by the embedder
	cred      *credential
	userAgent string
	logger    *slog.Logger
	storeCode string
	allowed   map[string]bool // nil = every method
	retry     atomic.Pointer[retryPolicy]
	maxBody   int
	redact    func(string) string
	reqHook   func(context.Context, *http.Request) error
	respHook  func(context.Context, *http.Response) error
	initErr   error // set only by the compat constructors, which cannot return an error
}

// Format prints the Client without its configuration: whatever the verb,
// the token (and every hook or redactor) stays out of logs and panics.
func (c *Client) Format(f fmt.State, verb rune) {
	if c == nil {
		_, _ = io.WriteString(f, "<nil>")
		return
	}
	_, _ = io.WriteString(f, "&magento2.Client{base: "+c.base+"}")
}

// credential keeps the token behind a pointer so printing a Client shows an
// address, never the token.
type credential struct{ token string }

func (*credential) String() string   { return "[redacted]" }
func (*credential) GoString() string { return "[redacted]" }

type retryPolicy struct {
	count         int
	wait, maxWait time.Duration
}

// config collects the options before New validates them.
type config struct {
	token           string
	httpClient      *http.Client
	httpClientSet   bool
	timeout         time.Duration
	timeoutSet      bool
	followRedirects bool
	redirectsSet    bool
	userAgent       string
	logger          *slog.Logger
	storeCode       string
	allowed         []string
	retry           retryPolicy
	maxBody         int
	redact          func(string) string
	reqHook         func(context.Context, *http.Request) error
	respHook        func(context.Context, *http.Response) error
}

// ClientOption configures a Client in New. (The name Option is taken by
// the attribute option type.)
type ClientOption func(*config)

// WithToken sets the bearer token sent as "Authorization: Bearer <token>".
// An empty token sends no Authorization header.
func WithToken(token string) ClientOption { return func(c *config) { c.token = token } }

// WithHTTPClient makes the Client send every request through hc. hc is used
// exactly as given: only hc.Do is called; its Transport, Jar, CheckRedirect
// and Timeout are never read or written. WithTimeout and WithFollowRedirects
// apply to the default client only and are refused together with this
// option.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *config) { c.httpClient, c.httpClientSet = hc, true }
}

// WithTimeout sets the per-attempt timeout of the default *http.Client
// (DefaultTimeout otherwise). Bound a whole call, retries included, with the
// context instead.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *config) { c.timeout, c.timeoutSet = d, true }
}

// WithFollowRedirects sets whether the default *http.Client follows
// redirects (default true, up to net/http's limit of 10). With false, a 3xx
// is returned as an *APIError and its Location is never requested.
func WithFollowRedirects(follow bool) ClientOption {
	return func(c *config) { c.followRedirects, c.redirectsSet = follow, true }
}

// WithUserAgent sets the User-Agent header (DefaultUserAgent otherwise). It
// must be non-empty printable ASCII.
func WithUserAgent(ua string) ClientOption { return func(c *config) { c.userAgent = ua } }

// WithLogger sets the logger that receives one debug record per attempt
// (method, path, status, elapsed, bytes, truncated). A query string, a
// header or a body is never logged. nil discards.
func WithLogger(l *slog.Logger) ClientOption { return func(c *config) { c.logger = l } }

// WithStoreCode sets the store code requests are scoped to (the {store}
// in /rest/{store}/V1/...); Request.StoreCode overrides it per request.
// Default DefaultStoreCode.
func WithStoreCode(code string) ClientOption { return func(c *config) { c.storeCode = code } }

// WithAllowedMethods restricts the HTTP methods the Client sends; any other
// method is refused with ErrMethodNotAllowed before a request is built.
// Methods are compared upper-cased. No methods (the default) allows all.
func WithAllowedMethods(methods ...string) ClientOption {
	return func(c *config) { c.allowed = append([]string(nil), methods...) }
}

// WithRetryPolicy sets how many times a failed idempotent request (GET,
// HEAD, OPTIONS, DELETE) is retried after the first attempt, and the base
// and maximum wait between attempts. count 0 means exactly one attempt.
// Retries happen on 429, 500, 502, 503, 504 and transport errors; a
// Retry-After header (delta-seconds or HTTP-date) replaces the computed
// backoff, and both are capped at maxWait. Default DefaultRetryCount,
// DefaultRetryWaitTime, DefaultRetryMaxWaitTime.
func WithRetryPolicy(count int, wait, maxWait time.Duration) ClientOption {
	return func(c *config) { c.retry = retryPolicy{count: count, wait: wait, maxWait: maxWait} }
}

// WithMaxBodyBytes caps how many bytes of a response body are read; a longer
// body is cut and Response.Truncated is set. 0 (the default) is unlimited.
// Request.MaxBodyBytes overrides it per request.
func WithMaxBodyBytes(n int) ClientOption { return func(c *config) { c.maxBody = n } }

// WithRedactor sets a function applied to every string that enters an
// *APIError (message, path, body, parameters, header values), a transport
// error, or a log record. Typically it replaces secrets the store might echo
// back. Default: identity.
func WithRedactor(redact func(string) string) ClientOption {
	return func(c *config) { c.redact = redact }
}

// WithRequestHook sets a function called with every fully built request
// (URL, headers, body) right before it is sent, once per attempt. An error
// aborts the call: nothing is sent and Do returns the error wrapped.
func WithRequestHook(hook func(context.Context, *http.Request) error) ClientOption {
	return func(c *config) { c.reqHook = hook }
}

// WithResponseHook sets a function called with every response before its
// body is read, once per attempt. An error aborts the call: the body is
// closed unread, nothing is retried, and Do returns the error wrapped.
func WithResponseHook(hook func(context.Context, *http.Response) error) ClientOption {
	return func(c *config) { c.respHook = hook }
}

// New returns a Client for the store at baseURL, the store root with an
// optional path prefix ("https://shop.example", "https://shop.example/magento");
// requests go to {baseURL}/rest/{storeCode}{Request.Path}.
//
// Without WithHTTPClient the Client builds its own *http.Client: no cookie
// jar, WithTimeout's timeout, redirects per WithFollowRedirects, and a clone
// of http.DefaultTransport — which keeps its Proxy setting,
// http.ProxyFromEnvironment, so HTTP_PROXY/HTTPS_PROXY/NO_PROXY apply. Supply
// a client to control proxies, TLS or dialing.
func New(baseURL string, opts ...ClientOption) (*Client, error) {
	cfg := config{
		followRedirects: true,
		timeout:         DefaultTimeout,
		userAgent:       DefaultUserAgent,
		storeCode:       DefaultStoreCode,
		retry:           retryPolicy{count: DefaultRetryCount, wait: DefaultRetryWaitTime, maxWait: DefaultRetryMaxWaitTime},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	base, err := parseBase(baseURL)
	if err != nil {
		return nil, err
	}
	if cfg.httpClientSet {
		if cfg.httpClient == nil {
			return nil, errors.New("magento2: WithHTTPClient(nil)")
		}
		if cfg.timeoutSet || cfg.redirectsSet {
			return nil, errors.New("magento2: WithTimeout and WithFollowRedirects apply to the default client only; configure the supplied *http.Client instead")
		}
	}
	if cfg.timeout < 0 {
		return nil, errors.New("magento2: negative timeout")
	}
	if !validUserAgent(cfg.userAgent) {
		return nil, errors.New("magento2: user agent must be non-empty printable ASCII")
	}
	if !storeCodeRe.MatchString(cfg.storeCode) {
		return nil, fmt.Errorf("magento2: invalid store code %q", cfg.storeCode)
	}
	if cfg.retry.count < 0 || cfg.retry.wait < 0 || cfg.retry.maxWait < 0 {
		return nil, errors.New("magento2: negative retry policy")
	}
	if cfg.maxBody < 0 {
		return nil, errors.New("magento2: negative body cap")
	}
	var allowed map[string]bool
	if len(cfg.allowed) > 0 {
		allowed = make(map[string]bool, len(cfg.allowed))
		for _, m := range cfg.allowed {
			if !methodRe.MatchString(m) {
				return nil, fmt.Errorf("magento2: invalid method %q", m)
			}
			allowed[strings.ToUpper(m)] = true
		}
	}

	c := &Client{
		base:      base,
		owned:     !cfg.httpClientSet,
		userAgent: cfg.userAgent,
		logger:    cfg.logger,
		storeCode: cfg.storeCode,
		allowed:   allowed,
		maxBody:   cfg.maxBody,
		redact:    cfg.redact,
		reqHook:   cfg.reqHook,
		respHook:  cfg.respHook,
	}
	if cfg.token != "" {
		c.cred = &credential{token: cfg.token}
	}
	if c.logger == nil {
		c.logger = slog.New(slog.DiscardHandler)
	}
	if c.redact == nil {
		c.redact = func(s string) string { return s }
	}
	if cfg.httpClientSet {
		c.http.Store(cfg.httpClient)
	} else {
		c.http.Store(defaultHTTPClient(cfg.timeout, cfg.followRedirects))
	}
	retry := cfg.retry
	c.retry.Store(&retry)
	return c, nil
}

func defaultHTTPClient(timeout time.Duration, followRedirects bool) *http.Client {
	var transport http.RoundTripper
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = t.Clone()
	} else {
		transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	hc := &http.Client{Timeout: timeout, Transport: transport}
	if !followRedirects {
		hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return hc
}

func parseBase(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("magento2: invalid base URL")
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return "", errors.New("magento2: base URL scheme must be http or https")
	case u.Host == "":
		return "", errors.New("magento2: base URL has no host")
	case u.User != nil:
		return "", errors.New("magento2: base URL must not carry user info")
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(raw, "?#"):
		return "", errors.New("magento2: base URL must not carry a query or fragment")
	}
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.EscapedPath(), "/"), nil
}

func validUserAgent(ua string) bool {
	if ua == "" {
		return false
	}
	for i := 0; i < len(ua); i++ {
		if ua[i] < 0x20 || ua[i] > 0x7e {
			return false
		}
	}
	return true
}

// SetTimeout sets the per-attempt timeout of the default *http.Client. It
// has no effect on a client supplied with WithHTTPClient (which is never
// modified). Prefer WithTimeout.
func (c *Client) SetTimeout(d time.Duration) {
	if !c.owned {
		return
	}
	cur := c.http.Load()
	if cur == nil {
		return
	}
	next := *cur
	next.Timeout = d
	c.http.Store(&next)
}

// SetRetryPolicy replaces the retry policy; see WithRetryPolicy. Negative
// values are treated as 0. Prefer WithRetryPolicy.
func (c *Client) SetRetryPolicy(count int, wait, maxWait time.Duration) {
	p := retryPolicy{count: max(count, 0), wait: max(wait, 0), maxWait: max(maxWait, 0)}
	c.retry.Store(&p)
}

// Request is one REST call.
type Request struct {
	// Method defaults to GET.
	Method string
	// Path is relative to /rest and starts with "/": "/V1/orders",
	// "/schema". Percent-escapes are sent exactly as written; '?', '#',
	// control characters and "." / ".." segments are refused.
	Path string
	// StoreCode overrides the Client's store code for this request ("all"
	// for /rest/all/schema).
	StoreCode string
	// Query is encoded into the URL; it is never logged or put into errors.
	Query url.Values
	// Fields, when non-empty, is sent as the fields= projection, replacing
	// any "fields" in Query.
	Fields string
	// Body, when non-nil, is sent JSON-encoded (json.RawMessage verbatim).
	Body any
	// MaxBodyBytes overrides the Client's body cap for this request when > 0.
	MaxBodyBytes int
}

// Response is a 2xx answer.
type Response struct {
	Status int
	Header http.Header
	// Body is the response body, cut at the body cap when Truncated.
	Body      []byte
	Truncated bool
	// Elapsed covers the last attempt, from sending to reading the body.
	Elapsed time.Duration
}

// requestError is a transport or hook failure: its text is redacted and
// carries neither the host nor the query string; Unwrap keeps errors.Is
// (context.DeadlineExceeded, the hook's error) working.
type requestError struct {
	msg string
	err error
}

func (e *requestError) Error() string { return e.msg }
func (e *requestError) Unwrap() error { return e.err }

// Do sends req and returns the 2xx response, or an *APIError for any other
// status, or an error for a refused, failed or canceled request. Idempotent
// requests are retried per the retry policy.
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}
	if ctx == nil {
		ctx = context.Background()
	}
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	if c.allowed != nil && !c.allowed[method] {
		return nil, fmt.Errorf("magento2: %s %s: %w", c.redact(method), c.redact(req.Path), ErrMethodNotAllowed)
	}
	target, err := c.buildURL(req)
	if err != nil {
		return nil, err
	}
	var body []byte
	if req.Body != nil {
		if body, err = json.Marshal(req.Body); err != nil {
			return nil, fmt.Errorf("magento2: %s %s: encode body: %w", method, c.redact(req.Path), err)
		}
	}
	bodyCap := c.maxBody
	if req.MaxBodyBytes > 0 {
		bodyCap = req.MaxBodyBytes
	}

	policy := *c.retry.Load()
	retryable := retryableMethods[method]
	for attempt := 0; ; attempt++ {
		resp, retryAfter, err := c.attempt(ctx, method, req.Path, target, body, bodyCap)
		if err == nil {
			return resp, nil
		}
		var final bool
		var apiErr *APIError
		switch {
		case errors.As(err, &apiErr):
			final = !retryStatusCodes[apiErr.StatusCode]
		case errors.Is(err, errNoRetry):
			final = true
		default:
			final = ctx.Err() != nil
		}
		if final || !retryable || attempt >= policy.count {
			return nil, stripNoRetry(err)
		}
		timer := time.NewTimer(policy.backoff(attempt+1, retryAfter))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, &requestError{msg: fmt.Sprintf("magento2: %s %s: %v", method, c.redact(req.Path), ctx.Err()), err: ctx.Err()}
		case <-timer.C:
		}
	}
}

// errNoRetry marks a hook refusal: it is never retried.
var errNoRetry = errors.New("no retry")

type noRetry struct{ error }

func (n noRetry) Is(target error) bool { return target == errNoRetry }
func (n noRetry) Unwrap() error        { return n.error }

func stripNoRetry(err error) error {
	if n, ok := err.(noRetry); ok {
		return n.error
	}
	return err
}

// attempt performs one exchange. retryAfter is the parsed Retry-After of a
// non-2xx answer, 0 when absent.
func (c *Client) attempt(ctx context.Context, method, path, target string, body []byte, bodyCap int) (*Response, time.Duration, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	hreq, err := http.NewRequestWithContext(ctx, method, target, rd)
	if err != nil {
		return nil, 0, noRetry{fmt.Errorf("magento2: %s %s: build request: %w", method, c.redact(path), unwrapURLError(err))}
	}
	hreq.Header.Set("Accept", "application/json")
	hreq.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		hreq.Header.Set("Content-Type", "application/json")
	}
	if c.cred != nil {
		hreq.Header.Set("Authorization", "Bearer "+c.cred.token)
	}
	if c.reqHook != nil {
		if err := c.reqHook(ctx, hreq); err != nil {
			return nil, 0, noRetry{c.wrap(method, path, "request hook", err)}
		}
	}

	start := time.Now()
	hresp, err := c.http.Load().Do(hreq)
	if err != nil {
		err = c.wrap(method, path, "", unwrapURLError(err))
		c.logAttempt(ctx, method, path, 0, time.Since(start), 0, false, err)
		return nil, 0, err
	}
	defer hresp.Body.Close()
	if c.respHook != nil {
		if err := c.respHook(ctx, hresp); err != nil {
			c.logAttempt(ctx, method, path, hresp.StatusCode, time.Since(start), 0, false, nil)
			return nil, 0, noRetry{c.wrap(method, path, "response hook", err)}
		}
	}

	ok := hresp.StatusCode >= 200 && hresp.StatusCode < 300
	readCap := bodyCap
	if !ok && (readCap <= 0 || readCap > defaultErrorBodyBytes) {
		readCap = defaultErrorBodyBytes
	}
	data, truncated, readErr := readCapped(hresp.Body, readCap)
	elapsed := time.Since(start)
	if readErr != nil {
		err = c.wrap(method, path, "read body", readErr)
		c.logAttempt(ctx, method, path, hresp.StatusCode, elapsed, len(data), truncated, err)
		return nil, 0, err
	}
	c.logAttempt(ctx, method, path, hresp.StatusCode, elapsed, len(data), truncated, nil)
	if !ok {
		return nil, parseRetryAfter(hresp.Header.Get("Retry-After")), c.newAPIError(method, path, hresp, data)
	}
	return &Response{Status: hresp.StatusCode, Header: hresp.Header, Body: data, Truncated: truncated, Elapsed: elapsed}, 0, nil
}

// wrap builds a requestError whose text is redacted.
func (c *Client) wrap(method, path, stage string, err error) error {
	msg := "magento2: " + method + " " + path + ": "
	if stage != "" {
		msg += stage + ": "
	}
	return &requestError{msg: c.redact(msg + err.Error()), err: err}
}

// unwrapURLError drops *url.Error's wrapper, whose text repeats the full
// URL (host and query string).
func unwrapURLError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

func readCapped(r io.Reader, limit int) ([]byte, bool, error) {
	if limit <= 0 {
		data, err := io.ReadAll(r)
		return data, false, err
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if len(data) > limit {
		return data[:limit], true, err
	}
	return data, false, err
}

func (c *Client) logAttempt(ctx context.Context, method, path string, status int, elapsed time.Duration, n int, truncated bool, err error) {
	if !c.logger.Enabled(ctx, slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{
		slog.String("method", method),
		slog.String("path", c.redact(path)),
		slog.Int("status", status),
		slog.Duration("elapsed", elapsed),
		slog.Int("bytes", n),
		slog.Bool("truncated", truncated),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	c.logger.LogAttrs(ctx, slog.LevelDebug, "magento2 request", attrs...)
}

// buildURL joins base, /rest, the store code and req.Path, keeping the
// path's percent-escapes exactly as written.
func (c *Client) buildURL(req Request) (string, error) {
	p := req.Path
	bad := func(why string) error {
		return fmt.Errorf("magento2: invalid path %q: %s", c.redact(p), why)
	}
	if !strings.HasPrefix(p, "/") {
		return "", bad("must start with /")
	}
	for i := 0; i < len(p); i++ {
		if p[i] < 0x21 || p[i] >= 0x7f || p[i] == '?' || p[i] == '#' {
			return "", bad("contains a space, a control or non-ASCII byte, '?' or '#'")
		}
	}
	for _, seg := range strings.Split(p[1:], "/") {
		if un, err := url.PathUnescape(seg); err != nil {
			return "", bad("invalid percent-escape")
		} else if un == "." || un == ".." {
			return "", bad("dot segment")
		}
	}
	store := c.storeCode
	if req.StoreCode != "" {
		store = req.StoreCode
	}
	if !storeCodeRe.MatchString(store) {
		return "", fmt.Errorf("magento2: invalid store code %q", store)
	}
	target := c.base + "/rest/" + store + p
	u, err := url.Parse(target)
	if err != nil {
		return "", bad("not a valid URL path")
	}
	if u.EscapedPath() != strings.TrimPrefix(target, u.Scheme+"://"+u.Host) {
		// net/http would re-encode the path and lose escapes such as %2F.
		return "", bad("characters outside RFC 3986 path syntax must be percent-escaped")
	}
	q := req.Query
	if req.Fields != "" {
		q = cloneValues(q)
		q.Set("fields", req.Fields)
	}
	if enc := q.Encode(); enc != "" {
		target += "?" + enc
	}
	return target, nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v)+1)
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

// DoJSON is Do followed by decoding the body into target. A truncated body
// is refused with ErrBodyTruncated and never decoded; an empty body or a nil
// target leaves target untouched.
func (c *Client) DoJSON(ctx context.Context, req Request, target any) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	if resp.Truncated {
		return fmt.Errorf("magento2: %s %s: %w", methodOrGet(req.Method), c.redact(req.Path), ErrBodyTruncated)
	}
	if target == nil {
		return nil
	}
	if err := decodeJSON(resp.Body, target); err != nil {
		return fmt.Errorf("magento2: %s %s: decode response: %w", methodOrGet(req.Method), c.redact(req.Path), err)
	}
	return nil
}

// decodeJSON decodes body into target; an empty body leaves target untouched.
func decodeJSON(body []byte, target any) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, target)
}

func methodOrGet(m string) string {
	if m == "" {
		return http.MethodGet
	}
	return m
}

// backoff returns the wait before retry number attempt (1-based): the
// server's Retry-After when given, else wait*2^(attempt-1) with jitter in
// [d/2, d]; both capped at maxWait.
func (p retryPolicy) backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return min(retryAfter, p.maxWait)
	}
	if p.wait <= 0 {
		return 0
	}
	d := p.wait
	for i := 1; i < attempt && d < p.maxWait; i++ {
		d *= 2
	}
	d = min(d, p.maxWait)
	if d <= 1 {
		return d
	}
	half := d / 2
	return half + rand.N(d-half+1)
}

// parseRetryAfter reads RFC 9110 delta-seconds or an HTTP-date; 0 when
// absent, unparsable or in the past.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs > 0 {
			return time.Duration(secs) * time.Second
		}
		return 0
	}
	if at, err := http.ParseTime(v); err == nil {
		if d := time.Until(at); d > 0 {
			return d
		}
	}
	return 0
}
