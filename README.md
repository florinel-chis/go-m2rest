# go-m2rest

[![Go Reference](https://pkg.go.dev/badge/github.com/florinel-chis/go-m2rest.svg)](https://pkg.go.dev/github.com/florinel-chis/go-m2rest)

**A Go client for the Magento 2 / Adobe Commerce REST API.**

Plain `net/http`, no dependencies outside the standard library. Context-first typed reads with
paging, a `searchCriteria` builder, typed errors that parse Magento's error documents, write
helpers for catalog and checkout, and the switches an embedder needs to run it hardened: its own
`*http.Client`, an allowed-methods list, retries off, a body cap, a redactor and request/response
hooks. A drift check (`cmd/m2drift`) compares the client with a live store's schema.

How the pieces fit: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). Upgrading from v0.1.x (resty /
zerolog): see the migration notes in [CHANGELOG.md](CHANGELOG.md).

## Install

Requires **Go 1.27+**.

```bash
go get github.com/florinel-chis/go-m2rest@v0.2.0
```

```go
import magento2 "github.com/florinel-chis/go-m2rest"
```

## Quick start

```go
client, err := magento2.New("https://shop.example", // store root; an optional path prefix is fine
    magento2.WithToken(os.Getenv("MAGENTO_BEARER_TOKEN")),
    magento2.WithUserAgent("my-sync/1.0"),
)
if err != nil {
    log.Fatal(err)
}
ctx := context.Background()

// Typed read with paging.
page, err := magento2.GetOrdersPage(ctx, client, magento2.NewSearchCriteria().
    Filter("status", magento2.Eq, "processing").
    Sort("created_at", magento2.Desc).
    Page(50, 1))
if err != nil {
    log.Fatal(err)
}
fmt.Println(page.TotalCount, "processing orders")

// Any route: Path is relative to /rest.
var views []magento2.StoreView
err = client.DoJSON(ctx, magento2.Request{Path: "/V1/store/storeViews"}, &views)

// A fields= projection.
resp, err := client.Do(ctx, magento2.Request{
    Path:   "/V1/products",
    Query:  url.Values{"searchCriteria[pageSize]": {"10"}},
    Fields: "items[sku,name],total_count",
})
```

The v0.1 constructors still work and take the same options:

```go
client, err := magento2.NewAPIClientFromIntegration(
    &magento2.StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"},
    token,
    magento2.WithTimeout(time.Minute),
)
```

## Client options

| Option | Effect |
|---|---|
| `WithToken(t)` | sends `Authorization: Bearer t`; the token is never printed (`%+v` of the client shows no token) |
| `WithHTTPClient(hc)` | sends through `hc`, used exactly as given (only `hc.Do` is called). **No default timeout**: set `hc.Timeout` or use context deadlines |
| `WithTimeout(d)` | per-attempt timeout of the default client (30s) |
| `WithFollowRedirects(b)` | default client follows redirects (true); with false a 3xx is an `*APIError` |
| `WithUserAgent(ua)` | `User-Agent` (default `go-m2rest`) |
| `WithLogger(l)` | per-client `*slog.Logger`, one debug record per attempt; nil discards |
| `WithStoreCode(code)` | scope requests to `/rest/<code>/...`; default none (see Store codes) |
| `WithAllowedMethods(m...)` | any other method is refused before a request exists (`ErrMethodNotAllowed`) |
| `WithRetryPolicy(n, wait, max)` | retries after the first attempt; `0` = exactly one attempt |
| `WithMaxBodyBytes(n)` | read at most `n` bytes of a body; longer bodies are cut and `Truncated` is set |
| `WithRedactor(f)` | applied to every string that enters an error or a log record |
| `WithRequestHook(f)` | sees each fully built request; an error aborts, nothing is sent |
| `WithResponseHook(f)` | sees each response before its body is read; an error aborts |

`WithTimeout` and `WithFollowRedirects` apply only to the client go-m2rest builds (a clone of
`http.DefaultTransport`, which keeps `ProxyFromEnvironment`, and no cookie jar); combined with
`WithHTTPClient` they make `New` return an error.

## Hardened read-only embedding

An embedder that must control exactly what leaves the process supplies its own client and turns
the defaults down:

```go
hc := &http.Client{
    Transport: transport, // your dialer, proxy and TLS policy
    Timeout:   20 * time.Second,
    CheckRedirect: func(*http.Request, []*http.Request) error {
        return http.ErrUseLastResponse // never follow redirects
    },
    // no Jar: no cookies
}

client, err := magento2.New(storeURL,
    magento2.WithHTTPClient(hc),
    magento2.WithToken(token),
    magento2.WithAllowedMethods(http.MethodGet), // GET only
    magento2.WithRetryPolicy(0, 0, 0),           // one attempt per call
    magento2.WithMaxBodyBytes(4<<20),            // 4 MiB per body
    magento2.WithRedactor(func(s string) string {
        return strings.ReplaceAll(s, token, "[redacted]")
    }),
    magento2.WithRequestHook(func(ctx context.Context, r *http.Request) error {
        if r.Header.Get("Cookie") != "" {
            return errors.New("refusing a request with a cookie")
        }
        return nil
    }),
    magento2.WithResponseHook(func(ctx context.Context, r *http.Response) error {
        if ct := r.Header.Get("Content-Type"); r.StatusCode == http.StatusOK && !strings.Contains(ct, "json") {
            return fmt.Errorf("unexpected content type %q", ct)
        }
        return nil
    }),
)

resp, err := client.Do(ctx, magento2.Request{Path: "/V1/orders", Query: q, MaxBodyBytes: 1 << 20})
if errors.Is(err, magento2.ErrMethodNotAllowed) { /* a non-GET was attempted */ }

var out magento2.OrderListResponse
err = client.DoJSON(ctx, req, &out) // a truncated body is refused with ErrBodyTruncated
```

**Carts:** prefer `GetCartsPage` (`/V1/carts/search`) for reads. Every per-cart GET
(`/V1/carts/{id}…`, `/V1/carts/mine…`, `/V1/guest-carts/{id}…`) loads the quote model, and
Magento's `Quote::_afterLoad` saves a quote flagged for recollection: a "read" that can write.

## Store codes

With no store code, requests go to `/rest/V1/...`, which Magento serves from the default store
view whatever its code. `WithStoreCode("de")` scopes every request to `/rest/de/V1/...`;
`Request.StoreCode` overrides it for one request. The schema is `/rest/all/schema`:
`Request{Path: "/schema", StoreCode: "all"}` or `GetSchema`.

## Typed services

`sc` is a `*SearchCriteria` (nil = no criteria); `opts` is the older `ListOptions`. `Iterate*`
calls `fn` for every item across pages and stops on `total_count`, an empty page, or a repeated
page (Magento clamps an out-of-range page to the last one).

| Function | Route |
|---|---|
| `GetProductsPage(ctx, c, opts)`, `IterateProducts(ctx, c, opts, fn)` | `GET /V1/products` |
| `GetProductBySKU(sku, c)` | `GET /V1/products/{sku}` |
| `CreateOrReplaceProduct(p, saveOptions, c)` | `POST /V1/products` |
| `(*MProduct).UpdateQuantityForStockItem(itemID, qty, inStock)` | `PUT /V1/products/{sku}/stockItems/{itemId}` |
| `GetAttributesPage(ctx, c, opts)`, `IterateAttributes(ctx, c, opts, fn)` | `GET /V1/products/attributes` |
| `GetAttributeByAttributeCode(code, c)` | `GET /V1/products/attributes/{attributeCode}` |
| `CreateAttribute(a, c)` | `POST /V1/products/attributes` |
| `(*MAttribute).AddOption(o)` | `POST /V1/products/attributes/{attributeCode}/options` |
| `GetAttributeSetsList(ctx, c)`, `GetAttributeSetByName(name, c)` | `GET /V1/products/attribute-sets/sets/list` |
| `GetAttributeSetAttributes(ctx, c, id)` | `GET /V1/products/attribute-sets/{attributeSetId}/attributes` |
| `CreateAttributeSet(a, skeletonID, c)` | `POST /V1/products/attribute-sets` |
| `GetCategoryTree(ctx, c)` | `GET /V1/categories` |
| `GetCategoryByName(name, c)` | `GET /V1/categories/list` |
| `CreateCategory(cat, c)` | `POST /V1/categories` |
| `GetConfigurableProductBySKU(sku, c)` | `GET /V1/configurable-products/{sku}/options/all` |
| `SetOptionForExistingConfigurableProduct(sku, o, c)` | `POST /V1/configurable-products/{sku}/options` |
| `GetCartsPage(ctx, c, sc)`, `IterateCarts(ctx, c, sc, fn)` | `GET /V1/carts/search` |
| `NewGuestCartFromAPIClient(c)`, `NewCustomerCartFromAPIClient(c)` | `POST /V1/guest-carts`, `POST /V1/carts/mine` |
| `GetOrdersPage(ctx, c, sc)`, `IterateOrders(ctx, c, sc, fn)` | `GET /V1/orders` |
| `GetOrder(ctx, c, id)` | `GET /V1/orders/{id}` |
| `GetOrderByIncrementID(incrementID, c)` | `GET /V1/orders` + `GET /V1/orders/{id}` |
| `GetInvoicesPage`, `IterateInvoices`, `GetInvoice(ctx, c, id)` | `GET /V1/invoices`, `GET /V1/invoices/{id}` |
| `GetCreditMemosPage`, `IterateCreditMemos`, `GetCreditMemo(ctx, c, id)` | `GET /V1/creditmemos`, `GET /V1/creditmemo/{id}` |
| `GetShipmentsPage`, `IterateShipments`, `GetShipment(ctx, c, id)` | `GET /V1/shipments`, `GET /V1/shipment/{id}` |
| `GetCustomersPage`, `IterateCustomers`, `GetCustomer(ctx, c, id)` | `GET /V1/customers/search`, `GET /V1/customers/{customerId}` |
| `GetSourceItemsPage`, `IterateSourceItems` | `GET /V1/inventory/source-items` |
| `GetSourcesPage`, `IterateSources` | `GET /V1/inventory/sources` |
| `GetStocksPage`, `IterateStocks` | `GET /V1/inventory/stocks` |
| `GetSalableQuantity(ctx, c, sku, stockID)` | `GET /V1/inventory/get-product-salable-quantity/{sku}/{stockId}` |
| `GetStockItem(ctx, c, sku)` | `GET /V1/stockItems/{productSku}` |
| `GetLowStockItems(ctx, c, scopeID, qty, page, size)` | `GET /V1/stockItems/lowStock/` |
| `GetStoreViews`, `GetWebsites`, `GetStoreGroups`, `GetStoreConfigs(ctx, c, codes...)` | `GET /V1/store/storeViews`, `/websites`, `/storeGroups`, `/storeConfigs` |
| `GetSchema(ctx, c)` | `GET /rest/all/schema` |

The cart checkout methods on `*MCart` (`AddItems`, `EstimateShippingCarrier`,
`AddShippingInformation`, `EstimatePaymentMethods`, `CreateOrder`, `DeleteItem`) and the
`M*` update methods cover the remaining routes. `Routes()` lists every route the package calls
with the Go type its response decodes into.

## SearchCriteria

```go
q, err := magento2.NewSearchCriteria().
    RestrictFields("status", "created_at", "store_id"). // optional allowlist
    Filter("status", magento2.Eq, "pending").Or().Filter("status", magento2.Eq, "processing").
    And().Filter("created_at", magento2.Gteq, "2026-10-01 00:00:00").
    FilterIn("store_id", "1", "2").
    Sort("created_at", magento2.Desc).
    Page(100, 1). // clamped to [1, 300], page >= 1
    Values()      // url.Values, or every mistake joined in err
resp, err := client.Do(ctx, magento2.Request{Path: "/V1/orders", Query: q})
```

Each `Filter` opens a new AND group unless preceded by `Or()`. Conditions: `Eq Neq Gt Gteq Lt
Lteq Like Nlike In Nin From To Finset Null NotNull Moreq`; directions `Asc`, `Desc`. A store that
caps the page size answers 400 "Maximum SearchCriteria pageSize is N":
`magento2.PageSizeLimit(apiErr.Message)` returns N. `ListOptions` remains for the
`GetProductsPage`/`GetAttributesPage` helpers.

## Errors

```go
order, err := magento2.GetOrder(ctx, client, 42)
if errors.Is(err, magento2.ErrNotFound) {
    // 404
}
var apiErr *magento2.APIError
if errors.As(err, &apiErr) {
    log.Printf("%s %s: %d %s", apiErr.Method, apiErr.Path, apiErr.StatusCode, apiErr.Message)
}
```

Every non-2xx answer is an `*APIError` (`StatusCode`, `Method`, `Path`, `Header`, `Body`,
`Message`, `Parameters`) that unwraps to `ErrNotFound` (404) or `ErrBadRequest` (any other
status). `Message` is Magento's error document with its `%name` / `%1` placeholders substituted;
`Error()` never contains the host or the query string, and every string has passed the redactor.
`Body` is capped like a 2xx body, or at 64 KiB when no cap is set. Embedders with their own
error types can call `ParseErrorDocument(body)` and `ErrorDocument.Substituted(redact)`
directly. Other sentinels: `ErrMethodNotAllowed`, `ErrBodyTruncated`, `ErrNoPointer`.

## Retries

By default a failed GET, HEAD, OPTIONS or DELETE is retried up to 4 times on 429, 500, 502, 503,
504 and transport errors, waiting the server's `Retry-After` (seconds or HTTP-date) or a jittered
exponential backoff from 500ms, capped at 2 minutes; a canceled context stops the wait. POST and
PUT are never retried (a proxy error can arrive after Magento committed an order).
`WithRetryPolicy(0, 0, 0)` makes every call exactly one attempt.

## Logging

Silent by default. `WithLogger` gives each client its own `*slog.Logger`, which receives one
debug record per attempt — `method`, `path`, `status`, `elapsed`, `bytes`, `truncated` — after the
redactor. A body, a header or a query string is never logged at any level.

## Write helpers

```go
mp, err := magento2.CreateOrReplaceProduct(&magento2.Product{
    Sku: "tee-001", Name: "Tee", TypeID: "simple", AttributeSetID: 4, Price: 19.9, Status: 1, Visibility: 4,
}, true, client)

cart, err := magento2.NewGuestCartFromAPIClient(client)
err = cart.AddItems([]magento2.CartItem{{Sku: "tee-001", Qty: 1}})
carriers, err := cart.EstimateShippingCarrier(&magento2.ShippingAddress{Address: magento2.Address{
    CountryID: "US", Postcode: "10001", City: "New York", Street: []string{"1 Main St"},
    Firstname: "Jo", Lastname: "Doe", Telephone: "555-1234", Email: "jo@example.com",
}})
```

## Drift check

```bash
MAGENTO_HOST=https://sandbox.example MAGENTO_BEARER_TOKEN=... \
    go run ./cmd/m2drift -vendor /path/to/magento/vendor -out DRIFT.md
```

It fetches the store's schema, checks that every `Routes()` entry exists, and compares each
registered type's json tags with the schema definition. `DRIFT` (exit 1): a route the store does
not serve, or a json tag the schema does not define. `INFO`: schema properties a type does not
model, extension attributes of modules the store lacks, and routes the token's ACL hides from the
schema (`-vendor` finds them in `etc/webapi.xml`). [DRIFT.md](DRIFT.md) holds the latest report.

## Keeping current

On every Magento 2.4.x release, otherwise quarterly, and always before a tag:

1. `curl -s 'https://go.dev/VERSION?m=text'` → `toolchain` line in `go.mod`
2. `go get -u ./... && go mod tidy` (there are no dependencies; keep it that way)
3. `govulncheck ./...`
4. `go vet ./...` and `go test -race ./...`
5. Live read tests against a disposable store
6. The drift check; commit `DRIFT.md`
7. CHANGELOG entry, then a semver tag

## Testing

```bash
go vet ./...
go test -race ./...   # offline (httptest); live tests skip without MAGENTO_HOST

# Live tests: only against a disposable store
MAGENTO_HOST=https://sandbox.example MAGENTO_BEARER_TOKEN=... go test -race ./tests/...
# tests that create, modify or delete data also need MAGENTO_TEST_ALLOW_WRITES=1
```

A `.env` file in the project root (`MAGENTO_HOST`, `MAGENTO_BEARER_TOKEN`, `MAGENTO_STORE_CODE`,
`TEST_DEBUG`, `MAGENTO_TEST_ALLOW_WRITES`) is read by the live tests.

## Docker Support

The library includes Docker support for easy testing and development without installing Go locally.

### Quick Start with Docker

```bash
# One-line command to run all tests
docker run --rm -e MAGENTO_BEARER_TOKEN=your_token_here -e MAGENTO_HOST=http://magento.local ghcr.io/florinel-chis/go-m2rest:latest

# Or build and run locally
docker build -t go-m2rest . && docker run --rm -e MAGENTO_BEARER_TOKEN=your_token go-m2rest
```

### Using Docker Compose

```bash
# Run all tests
MAGENTO_BEARER_TOKEN=your_token docker-compose run --rm test

# Run specific test
MAGENTO_BEARER_TOKEN=your_token TEST_NAME=TestAdvancedProducts_VirtualProduct docker-compose run --rm test-specific

# Development mode with live code reload
docker-compose run --rm dev
```

### Using the Docker Helper Script

For even easier usage, use the included `docker-run.sh` script:

```bash
# Run all tests
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh test

# Run specific test
MAGENTO_BEARER_TOKEN=your_token TEST_NAME=TestAdvancedProducts_VirtualProduct ./docker-run.sh test-specific

# Quick connectivity test
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh test-quick

# Create 50 products
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh bulk-create 50

# Update stock from CSV
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh bulk-update stock_updates.csv

# Open shell in container
./docker-run.sh shell

# Build Docker images
./docker-run.sh build
```

### Docker Environment Variables

The Docker setup supports all the same environment variables as the native setup:

- `MAGENTO_HOST` - Magento URL (default: `http://magento.local`)
- `MAGENTO_BEARER_TOKEN` - Integration token (required)
- `MAGENTO_STORE_CODE` - Store code (default: `all`)
- `TEST_DEBUG` - Enable debug logging (default: `true`)
- `TEST_NAME` - Specific test to run (for test-specific command)

### Building Custom Docker Image

You can customize the Dockerfile to include your own environment variables:

```dockerfile
# In Dockerfile, update the ENV section
ENV MAGENTO_HOST=http://your-magento.com \
    MAGENTO_BEARER_TOKEN=your_permanent_token \
    MAGENTO_STORE_CODE=your_store
```

Then build and run:

```bash
docker build -t my-m2rest .
docker run --rm my-m2rest
```

## Bulk Operations

`scripts/` has a bulk product creation and stock update example:

```bash
cd scripts
./run_bulk_update.sh stock_updates.csv 100 10 both
```

## License

This project is licensed under the [MIT License](LICENSE).

## Support

For issues, feature requests, or questions:
- Open an issue on [GitHub](https://github.com/florinel-chis/go-m2rest/issues)
- Check the [GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest) for API documentation

## Changelog

See [CHANGELOG.md](CHANGELOG.md): v0.2.0 replaced resty and zerolog with `net/http` and `slog`;
each breaking change has a migration note there.
