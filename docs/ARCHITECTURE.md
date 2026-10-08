# go-m2rest architecture

go-m2rest is one Go package (`magento2`) over `net/http`, plus the `cmd/m2drift` tool. Everything
goes through `(*Client).Do`: the typed services build a `Request`, `Do` sends it, and errors come
back as `*APIError`. The diagrams below follow the code at v0.2.0.

## Layers

```mermaid
flowchart TD
    subgraph embedder["Embedder code"]
        app["application"]
    end

    subgraph compat["Compat constructors (api_client.go)"]
        sc["StoreConfig"]
        ctor["NewAPIClientFromIntegration / NewAPIClientFromAuthentication / NewAPIClientWithoutAuthentication"]
        legacy["GetRouteAndDecodeCtx / PostRouteAndDecodeCtx (route relative to /V1)"]
    end

    subgraph services["Typed services"]
        catalog["products, attributes, attribute sets, categories, configurable products"]
        carts["carts: guest / mine checkout, carts/search"]
        sales["orders, invoices, creditmemos, shipments"]
        customers["customers"]
        inventory["inventory / MSI: source items, sources, stocks, salable qty"]
        stock["legacy stock items, low stock"]
        stores["store scopes: views, websites, groups, configs"]
        schema["GetSchema: /rest/all/schema"]
    end

    subgraph query["Query building"]
        crit["SearchCriteria builder (searchcriteria.go)"]
        listopts["ListOptions.Encode + BuildSearchQuery (compat)"]
        paging["getSearchPage / iterateSearch (search.go), iteratePages (list.go)"]
    end

    subgraph core["Client core (client.go)"]
        opts["ClientOption set: WithToken, WithHTTPClient, WithTimeout, WithUserAgent, WithLogger, WithStoreCode, WithAllowedMethods, WithRetryPolicy, WithFollowRedirects, WithMaxBodyBytes, WithRedactor, WithRequestHook, WithResponseHook"]
        newc["New(baseURL, opts...)"]
        do["Do / DoJSON"]
        hooks["request hook / response hook"]
        retry["retry policy + Retry-After backoff"]
        log["per-client slog logger"]
    end

    subgraph errs["Errors"]
        apierr["APIError + ErrNotFound / ErrBadRequest (generic_errors.go)"]
        doc["ErrorDocument: ParseErrorDocument, Substituted, Resources (errordocument.go)"]
        redact["redactor"]
    end

    registry["routes.go: Route registry, Routes()"]
    drift["cmd/m2drift: schema vs Routes()"]
    httpc["net/http Client (supplied or default)"]

    app --> newc
    app --> ctor
    app --> services
    app --> crit
    ctor --> sc
    ctor --> newc
    legacy --> do
    opts --> newc
    newc --> do
    services --> query
    services --> do
    services --> legacy
    paging --> do
    do --> hooks
    do --> retry
    do --> log
    do --> httpc
    do --> apierr
    apierr --> doc
    apierr --> redact
    log --> redact
    registry -.->|lists routes and types of| services
    drift --> registry
    drift --> schema
```

## One request through `Do`

```mermaid
sequenceDiagram
    autonumber
    participant Caller
    participant Do as Client.Do
    participant Attempt as attempt
    participant ReqHook as request hook
    participant HTTP as http.Client.Do
    participant RespHook as response hook
    participant Err as newAPIError

    Caller->>Do: Do(ctx, Request)
    Note over Do: initErr set by a compat constructor? return it. nil ctx becomes Background
    Note over Do: method upper-cased, empty means GET, non-token refused, WithAllowedMethods checked (ErrMethodNotAllowed)
    Note over Do: buildURL: base + /rest + optional /storeCode + Path. Path must start with /, no query or fragment marks, control or non-ASCII bytes, no dot segments, escapes kept. Fields replaces fields= in the query
    Note over Do: Body JSON-encoded once. Cap = Request.MaxBodyBytes, else WithMaxBodyBytes

    loop attempt 0..retry count
        Do->>Attempt: method, path, URL, body, cap
        Note over Attempt: http.NewRequestWithContext, Accept, User-Agent, Content-Type, Authorization Bearer
        Attempt->>ReqHook: hook(ctx, *http.Request)
        ReqHook-->>Attempt: error aborts, nothing sent, never retried
        Attempt->>HTTP: Do(req)
        HTTP-->>Attempt: response or transport error (URL stripped, redacted)
        Attempt->>RespHook: hook(ctx, *http.Response) before the body is read
        RespHook-->>Attempt: error aborts, body closed unread, never retried
        Note over Attempt: read body up to the cap. An explicit cap applies to 2xx and errors alike. No cap on a non-2xx means 64 KiB. One debug log line per attempt
        alt 2xx
            Attempt-->>Do: Response{Status, Header, Body, Truncated, Elapsed}
            Do-->>Caller: *Response
        else non-2xx
            Attempt->>Err: status, header, capped body
            Note over Err: ParseErrorDocument (BOM tolerant), Message = Substituted(redact), Parameters, Body and header values redacted. Unwrap gives ErrNotFound (404) or ErrBadRequest
            Err-->>Attempt: *APIError + Retry-After
            Attempt-->>Do: *APIError
        end
        Note over Do: retry only GET HEAD OPTIONS DELETE, on 429 500 502 503 504 or a transport error, while attempts remain. Wait = Retry-After or jittered backoff, capped at max wait. ctx done during the wait returns the ctx error
    end
    Do-->>Caller: *APIError or error

    Note over Caller,Do: DoJSON = Do, then a Truncated body returns ErrBodyTruncated, an empty body or nil target leaves the target untouched, else json.Unmarshal
```

## Files

```mermaid
flowchart LR
    subgraph corefiles["Core"]
        client_go["client.go: Client, ClientOption, New, Request, Response, Do, DoJSON, retries, buildURL, logging"]
        errors_go["generic_errors.go: APIError, ErrNotFound, ErrBadRequest, ErrNoPointer"]
        doc_go["errordocument.go: ErrorDocument, ParseErrorDocument"]
        compat_go["api_client.go: StoreConfig, NewAPIClientFrom*, GetRouteAndDecodeCtx"]
    end
    subgraph queryfiles["Query"]
        sc_go["searchcriteria.go: SearchCriteria, Cond, SortDir, PageSizeLimit"]
        search_go["search.go: getSearchPage, iterateSearch"]
        list_go["list.go + list_types.go: ListOptions, Get*Page, Iterate*, iteratePages"]
        bsq_go["internal_build_search_query.go: BuildSearchQuery (compat)"]
    end
    subgraph servicefiles["Services"]
        svc["product.go, attribute.go, attribute_set.go, categories.go, configurable_products.go, cart.go, orders.go"]
        reads["sales.go, customers.go, inventory.go, carts_search.go, stores.go, schema.go"]
        types["*_types.go, read_types.go, addresses.go, common_types.go, generic_types.go, api_types.go, flexbool.go"]
        rconst["*_routes.go, api_routes.go: route constants"]
    end
    routes_go["routes.go: Route, Routes()"]
    subgraph tool["cmd/m2drift"]
        main_go["main.go: flags, GetSchema, vendor webapi.xml scan, report file"]
        drift_go["drift.go: compare, compareType, render"]
    end
    tests_dir["tests/: live tests (MAGENTO_HOST, MAGENTO_TEST_ALLOW_WRITES)"]

    svc --> client_go
    reads --> search_go
    reads --> client_go
    svc --> compat_go
    compat_go --> client_go
    list_go --> compat_go
    search_go --> sc_go
    client_go --> errors_go
    errors_go --> doc_go
    routes_go --> types
    main_go --> drift_go
    drift_go --> routes_go
    tests_dir --> compat_go
```

| File | Holds |
|---|---|
| `client.go` | `Client`, `ClientOption` and every `With*`, `New`, `Request`, `Response`, `Do`, `DoJSON`, retry policy, URL building, per-attempt logging, `SetTimeout`, `SetRetryPolicy` |
| `generic_errors.go` | `APIError` (and building it from a response), `ErrNotFound`, `ErrBadRequest`, `ErrNoPointer` |
| `errordocument.go` | `ErrorDocument`, `ParseErrorDocument`, `Substituted`, `Resources`, capped placeholder substitution |
| `searchcriteria.go` | `SearchCriteria`, `Cond`, `SortDir`, `MinPageSize`, `MaxPageSize`, `PageSizeLimit` |
| `search.go` | getList paging over `SearchCriteria` (`getSearchPage`, `iterateSearch`, `iterateList`) |
| `list.go`, `list_types.go` | `ListOptions`, `GetProductsPage`, `IterateProducts`, attributes, attribute sets list, category tree, `iteratePages` |
| `internal_build_search_query.go` | `BuildSearchQuery`, `BuildFlexibleSearchQuery` (compat) |
| `routes.go` | `Route`, `Routes()`: every route the package calls, with its response type |
| `product.go`, `attribute.go`, `attribute_set.go`, `categories.go`, `configurable_products.go`, `cart.go`, `orders.go` | write and read helpers on `M*` wrappers (route constants in the matching `*_routes.go`) |
| `sales.go`, `customers.go`, `inventory.go`, `carts_search.go`, `stores.go`, `schema.go` | context-first read services and `GetSchema` |
| `*_types.go`, `read_types.go`, `addresses.go`, `common_types.go`, `generic_types.go`, `api_types.go`, `flexbool.go` | response and payload types, `FlexBool` |
| `api_client.go` | `StoreConfig`, `NewAPIClientFrom*`, `GetRouteAndDecode*`, `PostRouteAndDecode*` |
| `cmd/m2drift/main.go`, `cmd/m2drift/drift.go` | the drift check: fetch, compare, Markdown report, exit code |
| `tests/` | live tests against a disposable store |
