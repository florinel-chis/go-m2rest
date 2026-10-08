# go-m2rest

Go client library for the Magento 2 / Adobe Commerce REST API
(module `github.com/florinel-chis/go-m2rest`, package `magento2`). Plain `net/http`,
no non-standard-library dependencies. Callers of the library are called *embedders*.

Requires Go 1.27+ (`go 1.27.0`, `toolchain go1.27.1` in go.mod).

## Commands
- `go vet ./...`
- `go test -race ./...` — unit tests (httptest only, no network)
- Live tests: `MAGENTO_HOST=… MAGENTO_BEARER_TOKEN=… go test -race ./tests/...`
  — they skip without `MAGENTO_HOST`. Tests that create, modify or delete data additionally
  need `MAGENTO_TEST_ALLOW_WRITES=1` and skip without it.

## Conventions
- `context.Context` is the first parameter of every function that does I/O.
- Errors are wrapped with `%w`; non-2xx answers are `*APIError` (`errors.As`), and
  `errors.Is(err, ErrNotFound / ErrBadRequest)` keeps working.
- No package-level mutable state (no global logger, no global client). Configuration lives on
  the `Client` via `Option`s.
- No `panic` in library code.
- Never log a request or response body, a header or a query string — at any level. Every
  string that enters a log record or an error passes the embedder's redactor.
- An embedder-supplied `*http.Client` is used exactly as given: only its `Do` is called.
- Test first, table-driven, `net/http/httptest`.

## Live-test rule
Live tests run only against a disposable store. On this machine that is the sandbox
`http://127.0.0.1:8084` and nothing else; `http://localhost` (port 80) is a client's store and
is never a target. There is no default `MAGENTO_HOST`.

## Keeping current
Cadence: every Magento 2.4.x release, otherwise quarterly, always before a tag.
1. `curl -s 'https://go.dev/VERSION?m=text'` → `toolchain` line in go.mod (the `go` directive
   only moves when dependents can follow; note it under Breaking).
2. `go get -u ./... && go mod tidy` — there are no dependencies; adding one needs a reason.
3. `govulncheck ./...`, `go vet ./...`, `go test -race ./...`.
4. Live read tests against the sandbox (see Live-test rule).
5. Drift check, then commit the report:
   `MAGENTO_HOST=http://127.0.0.1:8084 MAGENTO_BEARER_TOKEN=… go run ./cmd/m2drift -vendor ~/fch/magento248/vendor -out DRIFT.md`
   (exit 1 on DRIFT: a registered route the store does not serve, or a json tag its schema
   lacks; INFO covers token-hidden routes, extension attributes, unmodelled properties).
   Every new service registers its route and response type in `routes.go`, citing the
   `etc/webapi.xml` that declares it; json tags come from the vendor `Api/Data/*Interface.php`.
6. CHANGELOG entry (Breaking items with a migration snippet), then a semver tag.
