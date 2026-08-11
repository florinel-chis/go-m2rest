# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking - 2026-08-10
- `Attribute.IsFilterable` and `Attribute.IsFilterableInSearch` changed from `bool` to the new `FlexBool` type (a struct with a `Value bool` field, a `Bool()` accessor and a `NewFlexBool` constructor). Code assigning these fields to/from plain `bool` needs `attr.IsFilterable.Bool()` for reads and `attr.IsFilterable = magento2.NewFlexBool(true)` for writes. In exchange, Magento's `is_filterable=2` ("Filterable (no results)") now decodes correctly and round-trips losslessly instead of being silently rewritten to `1` on update
- Corrupted generated field `WrappingAddPrfloat64edCard` (json `wrapping_add_prfloat64ed_card`) renamed to `WrappingAddPrintedCard` (json `wrapping_add_printed_card`) — references to the old Go field name no longer compile; the old JSON tag never matched Magento's real key
- 404 responses now return a `*APIError` wrapping `ErrNotFound` instead of the bare sentinel: `errors.Is(err, magento2.ErrNotFound)` keeps working on every path, but `err == magento2.ErrNotFound` comparisons no longer match and must be migrated to `errors.Is`; the 404 error string also changed
- All constructors now apply a 30s per-attempt HTTP timeout (previously none). Callers with requests legitimately slower than 30s (large base64 media galleries, bulk updates) must call `(*Client).SetTimeout` with a larger value
- Automatic retries are now restricted to idempotent methods (GET, HEAD, OPTIONS, DELETE). POST/PUT requests — order placement, entity creation — are never retried automatically, because a 429/502/504 from a proxy can arrive after Magento has already committed the write (duplicate orders/creates)
- `DisableDebugLogging` now means "revert to the logger installed via `SetZeroLogger`, or to the silent default when none was set" (see logging changes below)

### Added - 2026-08-10

#### Catalog sync / read API (context-first)
- `ListOptions` with `Encode()` producing Magento `searchCriteria` query params (filter groups, `pageSize` — defaults to 100 when unset, `currentPage`, `sortOrders`)
- `GetProductsPage` / `IterateProducts` (GET `/products`) with pagination that terminates on `total_count`, an empty page, or a page that repeats the previous one verbatim (Magento clamps out-of-range pages to the last page instead of returning an empty one, so the repeat guard prevents duplicate delivery when `total_count` drifts while iterating)
- `GetAttributesPage` / `IterateAttributes` (GET `/products/attributes`)
- `GetAttributeSetsList` (GET `/products/attribute-sets/sets/list`, paginates internally) and `GetAttributeSetAttributes` (GET `/products/attribute-sets/{id}/attributes`)
- `GetCategoryTree` (GET `/categories`) returning a nested `CategoryTreeNode`
- `GetStoreViews` (GET `/store/storeViews`) and `GetWebsites` (GET `/store/websites`)
- Context plumbing: `(*Client).GetRouteAndDecodeCtx` / `PostRouteAndDecodeCtx`; existing non-ctx helpers delegate with `context.Background()`
- `(*Client).SetTimeout` and `(*Client).SetRetryPolicy`
- `StoreConfig.BasePath` for installations served from a sub-directory (`{scheme}://{host}/{basePath}/rest/{storeCode}/V1`, slashes normalized; empty keeps previous behavior; `HostName` may still include a port)
- `FlexBool` type accepting JSON bools, numbers (0 = false, anything else true — `is_filterable` uses 0/1/2) and strings `"0"`/`"1"`/`"2"`/`"true"`/`"false"`. It preserves the original JSON token so non-bool values (e.g. `2`) survive read-modify-write round-trips unchanged; a value mutated after decoding marshals as a plain bool. `Attribute.IsFilterable` and `Attribute.IsFilterableInSearch` now use it (breaking, see above)
- Offline unit test suite (httptest-based) covering pagination, searchCriteria encoding, retries/Retry-After, typed errors, context cancellation, FlexBool and response decoding

### Changed - 2026-08-10
- Default per-attempt HTTP timeout of 30s on all constructors (breaking for slow requests, see above)
- Retries now cover 429, 500, 502, 503 and 504 for idempotent methods only (GET/HEAD/OPTIONS/DELETE), also retry transport-level failures (per-attempt timeout, connection reset) without retrying user-canceled requests, honor the `Retry-After` header in both delta-seconds and HTTP-date form, and default to 4 retries (5 total attempts) with 500ms base wait. `DefaultRetryMaxWaitTime` is now 2 minutes so that server-provided `Retry-After` values up to that long are honored (resty clamps `Retry-After` to the retry max wait; the computed jittered backoff stays ≤ ~4s with the default policy). With `SetRetryPolicy`, `maxWait` caps both the backoff and honored `Retry-After` values
- HTTP errors now carry a typed `*APIError` (`StatusCode`, `Endpoint`, `Body` truncated to 500 bytes) extractable via `errors.As`; `errors.Is(err, ErrNotFound)` / `errors.Is(err, ErrBadRequest)` keep working (`==` sentinel comparisons do not, see Breaking)
- The logger no longer hijacks the process-global zerolog logger on import: the package is silent by default (no-op logger); use `SetZeroLogger` or `EnableDebugLogging` to opt in. `EnableDebugLogging` now keeps a logger installed via `SetZeroLogger` (only lowering its level to debug) and `DisableDebugLogging` restores it at its original level; the package logger is stored behind an atomic pointer so toggling logging while requests are in flight is race-free
- Replaced deprecated resty `SetHostURL` with `SetBaseURL`
- Live functional tests under `tests/` now `t.Skip` when `MAGENTO_HOST` is unset instead of failing

### Fixed - 2026-08-10
- Corrupted generated field `WrappingAddPrfloat64edCard` (json `wrapping_add_prfloat64ed_card`) renamed to `WrappingAddPrintedCard` with json tag `wrapping_add_printed_card` in both `common_types.go` and `orders_types.go`

---

### Changed - 2025-01-15

#### Dependencies Update
- **Go Version:** Updated from 1.22.12 to 1.25.0 with toolchain 1.25.1
  - Provides latest language features and improvements
  - Better performance and security baseline
  - Note: Some Go stdlib vulnerabilities (GO-2025-4007 through GO-2025-4013) are pending fixes in Go 1.25.2+

#### Updated Dependencies
- `golang.org/x/net`: v0.33.0 → v0.47.0
- `golang.org/x/sys`: v0.28.0 → v0.38.0
- `github.com/mattn/go-colorable`: v0.1.13 → v0.1.14
- `github.com/mattn/go-isatty`: v0.0.19 → v0.0.20

#### Maintained at Latest Versions
- `github.com/go-resty/resty/v2`: v2.16.5 (no updates available)
- `github.com/rs/zerolog`: v1.34.0 (no updates available)

### Validation
- ✅ All packages compile successfully with Go 1.25.1
- ✅ `go vet` passes without warnings
- ✅ `go mod tidy` completes cleanly
- ✅ No breaking changes detected
- ✅ Tests run successfully (require Magento credentials)

### Security Notes
- Dependency updates include important security and bug fixes
- Upgraded to latest stable Go toolchain (1.25.1)
- Known stdlib vulnerabilities will be addressed when Go 1.25.2+ is released
- No vulnerabilities found in project dependencies

---

## Previous Updates

### Go 1.21+ Features
- Added support for modern Go features including the `any` type
- Implemented structured errors with wrapping
- Enhanced context support

### Resty v2 Migration
- Migrated to resty v2 for better performance
- Improved connection pooling
- Better HTTP client management

### Logging Enhancements
- Added structured logging with zerolog
- Improved debugging and monitoring capabilities
- Better error context and tracing

### Testing Improvements
- Improved test coverage and organization
- Added functional tests for all major features
- Better test configuration management

### Bulk Operations
- Added bulk operations utilities
- Support for concurrent product creation
- CSV-based bulk updates

### Error Handling
- Enhanced error handling with wrapped errors
- Custom error types for Magento 2 API scenarios
- Better error context and debugging
