# go-m2rest

Go client library for the Magento 2 / Adobe Commerce REST API
(module `github.com/florinel-chis/go-m2rest`, package `magento2`). Plain `net/http`,
no non-standard-library dependencies. Callers of the library are called *embedders*.

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
Placeholder: a drift check (`cmd/m2drift`, report in `DRIFT.md`) comparing the client's routes
and types against a store's `/rest/all/schema` is added in a follow-up; the routine (toolchain,
dependencies, `govulncheck`, drift, CHANGELOG, tag) will be documented here and in the README.
