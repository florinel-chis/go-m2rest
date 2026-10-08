# go-m2rest

[![GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest?status.svg)](https://godoc.org/github.com/florinel-chis/go-m2rest)

**A Go client for the Magento 2 / Adobe Commerce REST API**

Plain `net/http`, no dependencies outside the standard library. Context-first reads with paging,
a `searchCriteria` builder, typed errors that parse Magento's error documents, and the switches
an embedder needs to run it hardened: its own `*http.Client`, an allowed-methods list, retries
off, a body cap, a redactor and request/response hooks. A drift check compares the client with a
live store's schema.

## Features

* **`net/http` core:** `New(baseURL, options...)`, `Do` / `DoJSON` over a `Request` relative to
  `/rest` (`/V1/orders`, or `/schema` under the `all` store code). An embedder-supplied
  `*http.Client` is used exactly as given.
* **Context-first reads with paging:** products, attributes, attribute sets, categories, orders,
  invoices, credit memos, shipments, customers, MSI source items / sources / stocks, salable
  quantity, legacy stock items, store views / websites / groups / configs, carts search, and the
  REST schema. `Get<X>Page` fetches one page, `Iterate<X>` walks them all.
* **`SearchCriteria` builder:** AND groups / OR filters, several sorts, page clamping, optional
  field allowlist; mistakes are reported at `Values()`.
* **Typed errors:** non-2xx answers are `*APIError` (status, method, path, capped body, the
  Magento message with its `%placeholders` substituted); `errors.Is(err, ErrNotFound)` keeps
  working.
* **Safe by default:** retries only idempotent methods (GET, HEAD, OPTIONS, DELETE) on
  429/500/502/503/504 and transport errors, honouring `Retry-After`; no cookie jar; per-client
  `slog` logger that never sees a body, a header or a query string.
* **Write helpers:** products, attributes and options, attribute sets and groups, categories,
  configurable products, guest and customer carts through checkout, order updates and comments.

## Getting Started

### Prerequisites

1. **Go 1.27+** - the module's `go` directive is 1.27.0 (toolchain go1.27.1)
2. **Magento 2 / Adobe Commerce** with the REST API enabled
3. **A token** - integration (bearer) token, or admin/customer credentials

Requests go to `/rest/V1/...` (the default store view) unless `WithStoreCode` or
`Request.StoreCode` scopes them to `/rest/<code>/V1/...`.

### Installation

```bash
go get github.com/florinel-chis/go-m2rest@v0.2.0
```

### Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    magento2 "github.com/florinel-chis/go-m2rest"
)

func main() {
    // The store root (an optional path prefix is fine); the client appends /rest.
    // No store code: requests go to /rest/V1/... (default store view); add
    // magento2.WithStoreCode("de") or Request.StoreCode to scope them.
    client, err := magento2.New("https://shop.example",
        magento2.WithToken("integration-token"),
        magento2.WithUserAgent("my-sync/1.0"),
    )
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()

    // Typed reads with a searchCriteria built for you.
    sc := magento2.NewSearchCriteria().
        Filter("status", magento2.Eq, "processing").
        Sort("created_at", magento2.Desc).
        Page(50, 1)
    page, err := magento2.GetOrdersPage(ctx, client, sc)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(page.TotalCount, "processing orders")

    // Any route: Path is relative to /rest; StoreCode scopes it ("" = default view).
    var views []magento2.StoreView
    if err := client.DoJSON(ctx, magento2.Request{Path: "/V1/store/storeViews"}, &views); err != nil {
        log.Fatal(err)
    }

    // fields= projection, set once per request.
    resp, err := client.Do(ctx, magento2.Request{
        Path:   "/V1/products",
        Query:  must(magento2.NewSearchCriteria().Page(10, 1).Values()),
        Fields: "items[sku,name],total_count",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(resp.Body))
}

func must[T any](v T, err error) T {
    if err != nil {
        log.Fatal(err)
    }
    return v
}
```

The pre-v0.2 constructors keep working and take the same options:

```go
client, err := magento2.NewAPIClientFromIntegration(
    &magento2.StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"},
    "integration-token",
    magento2.WithTimeout(time.Minute),
)
```

### Store codes

Requests go to `{base}/rest{Path}` when no store code is set — Magento serves those from the
default store view, whatever its code — and to `{base}/rest/{code}{Path}` when `WithStoreCode`
or `Request.StoreCode` names one (`Request.StoreCode` wins). `/rest/all/schema` is
`Request{Path: "/schema", StoreCode: "all"}` (or `GetSchema`).

## Hardened embedding

An embedder that must control exactly what leaves the process — a gateway, an agent tool, a
multi-tenant service — supplies its own `*http.Client` and turns the defaults down:

```go
hc := &http.Client{
    Transport:     myTransport,   // your dialer, proxy and TLS policy
    CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
    // no Jar: no cookies
}
redact := func(s string) string { return strings.ReplaceAll(s, token, "[redacted]") }

client, err := magento2.New(storeURL,
    magento2.WithHTTPClient(hc),               // used as given: only hc.Do is called
    magento2.WithToken(token),                 // kept unprintable: %+v of the client never shows it
    magento2.WithAllowedMethods(http.MethodGet), // anything else is refused before a request exists
    magento2.WithRetryPolicy(0, 0, 0),         // exactly one attempt per call
    magento2.WithMaxBodyBytes(4<<20),          // longer bodies are cut and flagged Truncated
    magento2.WithRedactor(redact),             // applied to every error string and log record
    magento2.WithLogger(logger),               // nil discards; never sees bodies, headers or queries
    magento2.WithRequestHook(func(ctx context.Context, r *http.Request) error {
        return policy.Check(r)                 // an error aborts: nothing is sent
    }),
    magento2.WithResponseHook(func(ctx context.Context, r *http.Response) error {
        return policy.CheckResponse(r)         // runs before the body is read
    }),
)
...
resp, err := client.Do(ctx, magento2.Request{Path: "/V1/orders", Query: q, MaxBodyBytes: 1 << 20})
if errors.Is(err, magento2.ErrMethodNotAllowed) { ... }
var out OrderPage
err = client.DoJSON(ctx, req, &out) // refuses a truncated body with ErrBodyTruncated
```

A supplied client gets **no default timeout**: set its `Timeout` (or use context deadlines).
`WithTimeout` and `WithFollowRedirects` configure the client go-m2rest builds itself (default:
30s per attempt, redirects followed, a clone of `http.DefaultTransport` — which keeps
`ProxyFromEnvironment`) and are refused together with `WithHTTPClient`.

`ParseErrorDocument(body).Substituted(redact)` and `PageSizeLimit(apiErr.Message)` are exported
for embedders that keep their own error types.

**Carts:** prefer `GetCartsPage` (`/V1/carts/search`) for reads. Every per-cart GET
(`/V1/carts/{id}…`, `/V1/carts/mine…`, `/V1/guest-carts/{id}…`) loads the quote model, and
`Quote::_afterLoad` saves a quote flagged for recollection — a "read" that can write.

## Catalog sync

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

sc := magento2.NewSearchCriteria().
    Filter("updated_at", magento2.Gt, "2026-01-01 00:00:00").
    Sort("updated_at", magento2.Asc).
    Page(200, 1)
err := magento2.IterateOrders(ctx, client, sc, func(o magento2.Order) error {
    fmt.Println(o.IncrementID, o.Status)
    return nil // return an error to abort the iteration
})

// The product and attribute helpers take the older ListOptions:
err = magento2.IterateProducts(ctx, client, magento2.ListOptions{PageSize: 200}, func(p magento2.Product) error {
    return nil
})
//   magento2.GetAttributeSetsList(ctx, client)
//   magento2.GetCategoryTree(ctx, client)
//   magento2.GetStoreViews(ctx, client) / GetWebsites / GetStoreGroups / GetStoreConfigs
```

Iteration stops when `total_count` items were seen, a page is empty, or a page repeats the
previous one (Magento clamps an out-of-range page to the last page). `Attribute.IsFilterable`
uses `FlexBool`, which tolerates Magento's mixed bool/int/string encodings.

## Error Handling

```go
order, err := magento2.GetOrder(ctx, client, 42)
if errors.Is(err, magento2.ErrNotFound) {
    // 404
}
var apiErr *magento2.APIError
if errors.As(err, &apiErr) {
    log.Printf("%s %s: %d %s", apiErr.Method, apiErr.Path, apiErr.StatusCode, apiErr.Message)
    if n, ok := magento2.PageSizeLimit(apiErr.Message); ok {
        // the store caps searchCriteria[pageSize] at n
    }
}
```

`Error()` never contains the host or the query string.

## Logging

Each client has its own `*slog.Logger` (`WithLogger`; silent by default). It receives one debug
record per attempt with `method`, `path`, `status`, `elapsed`, `bytes` and `truncated` — never a
body, a header or a query string — after the redactor.

## Testing

```bash
go vet ./...
go test -race ./...   # offline (httptest); live tests skip without MAGENTO_HOST

# Live tests: only against a disposable store
MAGENTO_HOST=https://sandbox.example MAGENTO_BEARER_TOKEN=... go test -race ./tests/...
# Tests that create, modify or delete data also need
MAGENTO_TEST_ALLOW_WRITES=1
```

A `.env` file in the project root (`MAGENTO_HOST`, `MAGENTO_BEARER_TOKEN`, `MAGENTO_STORE_CODE`,
`TEST_DEBUG`, `MAGENTO_TEST_ALLOW_WRITES`) is read by the live tests.

## Keeping current

The client is checked against Magento itself, not against memory:

```bash
# Drift: routes and response types vs a store's /rest/all/schema
MAGENTO_HOST=https://sandbox.example MAGENTO_BEARER_TOKEN=... \
    go run ./cmd/m2drift -vendor /path/to/magento/vendor -out DRIFT.md
```

`DRIFT` lines (a route the store does not serve, a json tag its schema does not define) fail the
check with exit code 1; `INFO` lines report schema properties a type does not model, extension
attributes of modules the store lacks, and routes hidden from the schema by the token's ACL
(`-vendor` finds them in `etc/webapi.xml`). `DRIFT.md` holds the latest report.

**Cadence:** on every Magento 2.4.x release, otherwise quarterly, and always before a tag.

**Routine:**

1. `curl -s 'https://go.dev/VERSION?m=text'` → update the `toolchain` line in `go.mod`
2. `go get -u ./... && go mod tidy` (there are no dependencies today; keep it that way)
3. `govulncheck ./...`
4. `go vet ./...` and `go test -race ./...`
5. Live read tests against a disposable store
6. The drift check above; commit `DRIFT.md`
7. CHANGELOG entry, then a semver tag

## API Coverage

| Area | Functions |
|---|---|
| Generic | `New`, `(*Client).Do`, `DoJSON`, `GetRouteAndDecodeCtx`, `PostRouteAndDecodeCtx`, `Routes`, `GetSchema` |
| Products | `GetProductsPage`, `IterateProducts`, `GetProductBySKU`, `CreateOrReplaceProduct`, `(*MProduct).UpdateQuantityForStockItem` |
| Attributes | `GetAttributesPage`, `IterateAttributes`, `GetAttributeByAttributeCode`, `CreateAttribute`, `(*MAttribute).AddOption` |
| Attribute sets | `GetAttributeSetsList`, `GetAttributeSetAttributes`, `GetAttributeSetByName`, `CreateAttributeSet` |
| Categories | `GetCategoryTree`, `GetCategoryByName`, `CreateCategory`, `(*MCategory).AssignProductByProductLink` |
| Configurable products | `GetConfigurableProductBySKU`, `SetOptionForExistingConfigurableProduct`, `(*MConfigurableProduct).AddChildBySKU` |
| Carts | `GetCartsPage`, `IterateCarts`, `NewGuestCartFromAPIClient`, `NewCustomerCartFromAPIClient`, `(*MCart)` checkout methods |
| Sales | `GetOrdersPage`, `IterateOrders`, `GetOrder`, `GetOrderByIncrementID`, invoices / credit memos / shipments `Get*Page`, `Iterate*`, `Get*` |
| Customers | `GetCustomersPage`, `IterateCustomers`, `GetCustomer` |
| Inventory | `GetSourceItemsPage`, `GetSourcesPage`, `GetStocksPage` (+ `Iterate*`), `GetSalableQuantity`, `GetStockItem`, `GetLowStockItems` |
| Stores | `GetStoreViews`, `GetWebsites`, `GetStoreGroups`, `GetStoreConfigs` |

## Advanced Usage

### Working with Different Product Types

```go
// Virtual Product (no shipping)
virtualProduct := magento2.Product{
    Sku:    "virtual-service-001",
    Name:   "Virtual Service",
    TypeID: "virtual",
    Price:  99.99,
    // No weight needed for virtual products
}

// Grouped Product
groupedProduct := magento2.Product{
    Sku:    "grouped-product-001",
    Name:   "Product Bundle",
    TypeID: "grouped",
    // Price comes from associated products
}

// Configurable Product
configurableProduct := magento2.Product{
    Sku:    "configurable-001",
    Name:   "T-Shirt",
    TypeID: "configurable",
    // Requires attribute configuration
}
```

### Cart Operations

```go
// Create guest cart
guestCart, err := magento2.NewGuestCartFromAPIClient(client)

// Add items
item := magento2.CartItem{
    Sku:     "test-product-001",
    Qty:     2,
    QuoteID: guestCart.QuoteID,
}
err = guestCart.AddItems([]magento2.CartItem{item})

// Estimate shipping
shippingAddr := &magento2.ShippingAddress{
    Address: magento2.Address{
        CountryID: "US",
        Postcode:  "10001",
        City:      "New York",
        Street:    []string{"123 Main St"},
        Firstname: "John",
        Lastname:  "Doe",
        Telephone: "555-1234",
        Email:     "john@example.com",
    },
}
carriers, err := guestCart.EstimateShippingCarrier(shippingAddr)
```

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

The library includes utilities for bulk operations. See the `scripts/` directory for examples.

### Bulk Product Creation and Stock Update

```bash
cd scripts
./run_bulk_update.sh stock_updates.csv 100 10 both
```

This will:
- Create 100 simple products
- Update their stock from the CSV file
- Use 10 concurrent operations

## Bulk Operations

The `scripts/` directory has a bulk product creation and stock update example:

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

See [CHANGELOG.md](CHANGELOG.md). v0.2.0 replaces resty and zerolog with `net/http` and `slog`
(migration notes per item there).
