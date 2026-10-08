package magento2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// fixtureClient answers every request with status 200 and body, recording
// the requests.
func fixtureClient(t *testing.T, body string) (*recordingRT, *Client) {
	t.Helper()
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		return textResponse(r, http.StatusOK, body), nil
	}}
	return rt, rtClient(t, rt, WithRetryPolicy(0, 0, 0))
}

// assertSubset fails unless every key of want (recursively) is present in
// got with an equal value: a fixture key the Go type has no json tag for
// disappears in the decode → encode round trip.
func assertSubset(t *testing.T, where string, want, got any) {
	t.Helper()
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			t.Errorf("%s: got %T, want an object", where, got)
			return
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok {
				t.Errorf("%s.%s: lost in the round trip (no json tag?)", where, k)
				continue
			}
			assertSubset(t, where+"."+k, wv, gv)
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			t.Errorf("%s: got %v, want %d elements", where, got, len(w))
			return
		}
		for i := range w {
			assertSubset(t, fmt.Sprintf("%s[%d]", where, i), w[i], g[i])
		}
	default:
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%s = %v, want %v", where, got, want)
		}
	}
}

func roundTrip(t *testing.T, fixture string, decoded any) {
	t.Helper()
	var want, got any
	if err := json.Unmarshal([]byte(fixture), &want); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	out, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	assertSubset(t, "$", want, got)
}

const (
	fixtureOrders = `{"items":[{"entity_id":42,"increment_id":"000000042","state":"processing","status":"processing","grand_total":61.3,"customer_email":"a@example.com","items":[{"item_id":7,"sku":"24-MB01","qty_ordered":1,"price":34}]}],"total_count":1}`

	fixtureInvoice = `{"entity_id":3,"order_id":42,"increment_id":"000000003","state":2,"grand_total":61.3,"base_currency_code":"USD","total_qty":1,"created_at":"2026-01-02 03:04:05","extension_attributes":{"x":1},
		"items":[{"entity_id":5,"parent_id":3,"order_item_id":7,"sku":"24-MB01","name":"Bag","qty":1,"price":34,"row_total":34}],
		"comments":[{"entity_id":9,"parent_id":3,"comment":"paid","is_customer_notified":0,"is_visible_on_front":1,"created_at":"2026-01-02 03:04:05"}]}`

	fixtureCreditMemo = `{"entity_id":4,"order_id":42,"invoice_id":3,"increment_id":"000000004","state":2,"creditmemo_status":1,"grand_total":10,"adjustment_positive":1.5,"adjustment_negative":0.5,
		"items":[{"entity_id":6,"parent_id":4,"order_item_id":7,"sku":"24-MB01","qty":1,"price":10,"weee_tax_applied":"[]"}],
		"comments":[{"entity_id":8,"parent_id":4,"comment":"refund","is_customer_notified":1,"is_visible_on_front":0}]}`

	fixtureShipment = `{"entity_id":2,"order_id":42,"increment_id":"000000002","total_qty":1,"total_weight":1.5,"shipment_status":1,"packages":[],
		"items":[{"entity_id":1,"parent_id":2,"order_item_id":7,"sku":"24-MB01","name":"Bag","qty":1,"weight":1.5}],
		"tracks":[{"entity_id":1,"parent_id":2,"order_id":42,"track_number":"1Z999","title":"UPS","carrier_code":"ups"}],
		"comments":[{"entity_id":1,"parent_id":2,"comment":"shipped","is_customer_notified":1,"is_visible_on_front":1}]}`

	fixtureCustomers = `{"items":[{"id":1,"group_id":1,"default_billing":"1","default_shipping":"1","email":"roni_cost@example.com","firstname":"Veronica","lastname":"Costello","store_id":1,"website_id":1,"disable_auto_group_change":1,"created_in":"Default Store View","dob":"1973-12-15","gender":2}],"total_count":1}`

	fixtureSourceItems = `{"items":[{"sku":"24-MB01","source_code":"default","quantity":100,"status":1}],"total_count":1}`

	fixtureSources = `{"items":[{"source_code":"default","name":"Default Source","enabled":true,"description":"Default Source","latitude":0.5,"longitude":1.5,"country_id":"US","postcode":"00000","use_default_carrier_config":true,"carrier_links":[{"carrier_code":"ups","position":1}]}],"total_count":1}`

	fixtureStocks = `{"items":[{"stock_id":1,"name":"Default Stock","extension_attributes":{"sales_channels":[{"type":"website","code":"base"}]}}],"total_count":1}`

	fixtureStockItem = `{"item_id":1,"product_id":1,"stock_id":1,"qty":3,"is_in_stock":true,"manage_stock":true,"low_stock_date":"2026-01-01 00:00:00"}`

	fixtureStoreConfigs = `[{"id":1,"code":"default","website_id":1,"locale":"en_US","base_currency_code":"USD","default_display_currency_code":"USD","timezone":"America/Chicago","weight_unit":"lbs","base_url":"http://shop.example/","secure_base_url":"https://shop.example/","base_media_url":"http://shop.example/media/"}]`

	fixtureStoreGroups = `[{"id":1,"website_id":1,"root_category_id":2,"default_store_id":1,"name":"Main Website Store","code":"main_website_store"}]`

	fixtureCarts = `{"items":[{"id":81,"created_at":"2026-09-15 21:37:40","is_active":true,"is_virtual":false,"items_count":1,"customer_is_guest":true,"store_id":1,"items":[{"item_id":159,"sku":"24-MB04","qty":1,"name":"Strive Shoulder Pack","price":16.7}]}],"total_count":1}`
)

func TestReadServices(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		call      func(ctx context.Context, c *Client) (any, error)
		wantPath  string     // escaped path on the wire
		wantQuery url.Values // nil: no query
	}{
		{
			name: "orders page", fixture: fixtureOrders, wantPath: "/rest/V1/orders",
			wantQuery: url.Values{"searchCriteria[filter_groups][0][filters][0][field]": {"status"}, "searchCriteria[filter_groups][0][filters][0][value]": {"pending"}, "searchCriteria[filter_groups][0][filters][0][condition_type]": {"eq"}},
			call: func(ctx context.Context, c *Client) (any, error) {
				return GetOrdersPage(ctx, c, NewSearchCriteria().Filter("status", Eq, "pending"))
			},
		},
		{
			name: "order", fixture: `{"entity_id":42,"increment_id":"000000042","grand_total":61.3}`, wantPath: "/rest/V1/orders/42",
			call: func(ctx context.Context, c *Client) (any, error) { return GetOrder(ctx, c, 42) },
		},
		{
			name: "invoices page", fixture: `{"items":[` + fixtureInvoice + `],"total_count":1}`, wantPath: "/rest/V1/invoices", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetInvoicesPage(ctx, c, nil) },
		},
		{
			name: "invoice", fixture: fixtureInvoice, wantPath: "/rest/V1/invoices/3",
			call: func(ctx context.Context, c *Client) (any, error) { return GetInvoice(ctx, c, 3) },
		},
		{
			name: "credit memos page", fixture: `{"items":[` + fixtureCreditMemo + `],"total_count":1}`, wantPath: "/rest/V1/creditmemos",
			wantQuery: url.Values{"searchCriteria[pageSize]": {"10"}, "searchCriteria[currentPage]": {"1"}},
			call: func(ctx context.Context, c *Client) (any, error) {
				return GetCreditMemosPage(ctx, c, NewSearchCriteria().Page(10, 1))
			},
		},
		{
			name: "credit memo", fixture: fixtureCreditMemo, wantPath: "/rest/V1/creditmemo/4",
			call: func(ctx context.Context, c *Client) (any, error) { return GetCreditMemo(ctx, c, 4) },
		},
		{
			name: "shipments page", fixture: `{"items":[` + fixtureShipment + `],"total_count":1}`, wantPath: "/rest/V1/shipments", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetShipmentsPage(ctx, c, NewSearchCriteria()) },
		},
		{
			name: "shipment", fixture: fixtureShipment, wantPath: "/rest/V1/shipment/2",
			call: func(ctx context.Context, c *Client) (any, error) { return GetShipment(ctx, c, 2) },
		},
		{
			name: "customers page", fixture: fixtureCustomers, wantPath: "/rest/V1/customers/search", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetCustomersPage(ctx, c, nil) },
		},
		{
			name: "customer", fixture: `{"id":1,"email":"roni_cost@example.com","firstname":"Veronica","group_id":1}`, wantPath: "/rest/V1/customers/1",
			call: func(ctx context.Context, c *Client) (any, error) { return GetCustomer(ctx, c, 1) },
		},
		{
			name: "source items page", fixture: fixtureSourceItems, wantPath: "/rest/V1/inventory/source-items", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetSourceItemsPage(ctx, c, nil) },
		},
		{
			name: "sources page", fixture: fixtureSources, wantPath: "/rest/V1/inventory/sources", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetSourcesPage(ctx, c, nil) },
		},
		{
			name: "stocks page", fixture: fixtureStocks, wantPath: "/rest/V1/inventory/stocks", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetStocksPage(ctx, c, nil) },
		},
		{
			name: "salable quantity", fixture: `12.5`, wantPath: "/rest/V1/inventory/get-product-salable-quantity/WS12%20Blue%2F1/1",
			call: func(ctx context.Context, c *Client) (any, error) { return GetSalableQuantity(ctx, c, "WS12 Blue/1", 1) },
		},
		{
			name: "stock item", fixture: fixtureStockItem, wantPath: "/rest/V1/stockItems/24-MB01",
			call: func(ctx context.Context, c *Client) (any, error) { return GetStockItem(ctx, c, "24-MB01") },
		},
		{
			name: "low stock items", fixture: `{"items":[` + fixtureStockItem + `],"total_count":1}`, wantPath: "/rest/V1/stockItems/lowStock/",
			wantQuery: url.Values{"scopeId": {"0"}, "qty": {"10"}, "currentPage": {"1"}, "pageSize": {"50"}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetLowStockItems(ctx, c, 0, 10, 0, 50) },
		},
		{
			name: "store configs", fixture: fixtureStoreConfigs, wantPath: "/rest/V1/store/storeConfigs",
			wantQuery: url.Values{"storeCodes[]": {"default", "de"}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetStoreConfigs(ctx, c, "default", "de") },
		},
		{
			name: "store configs, all", fixture: fixtureStoreConfigs, wantPath: "/rest/V1/store/storeConfigs",
			call: func(ctx context.Context, c *Client) (any, error) { return GetStoreConfigs(ctx, c) },
		},
		{
			name: "store groups", fixture: fixtureStoreGroups, wantPath: "/rest/V1/store/storeGroups",
			call: func(ctx context.Context, c *Client) (any, error) { return GetStoreGroups(ctx, c) },
		},
		{
			name: "carts search", fixture: fixtureCarts, wantPath: "/rest/V1/carts/search", wantQuery: url.Values{"searchCriteria": {""}},
			call: func(ctx context.Context, c *Client) (any, error) { return GetCartsPage(ctx, c, nil) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt, c := fixtureClient(t, tt.fixture)
			got, err := tt.call(context.Background(), c)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			r := rt.last()
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if r.URL.EscapedPath() != tt.wantPath {
				t.Errorf("path = %q, want %q", r.URL.EscapedPath(), tt.wantPath)
			}
			if gotQ := r.URL.Query(); !(len(gotQ) == 0 && len(tt.wantQuery) == 0) && !reflect.DeepEqual(gotQ, tt.wantQuery) {
				t.Errorf("query = %v, want %v", gotQ, tt.wantQuery)
			}
			roundTrip(t, tt.fixture, got)
		})
	}
}

func TestReadServicesRefuseBadCriteria(t *testing.T) {
	rt, c := fixtureClient(t, `{}`)
	_, err := GetOrdersPage(context.Background(), c, NewSearchCriteria().Filter("status", Cond("bogus"), "x"))
	if err == nil || rt.count() != 0 {
		t.Fatalf("GetOrdersPage with an invalid criteria = %v, %d requests; want an error and nothing sent", err, rt.count())
	}
}

func TestIterateInvoicesPages(t *testing.T) {
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()
		if q.Get("searchCriteria[pageSize]") != "2" || q.Get("searchCriteria[filter_groups][0][filters][0][field]") != "state" {
			return textResponse(r, http.StatusBadRequest, `{"message":"criteria lost"}`), nil
		}
		page, _ := strconv.Atoi(q.Get("searchCriteria[currentPage]"))
		switch page {
		case 2:
			return textResponse(r, 200, `{"items":[{"entity_id":1},{"entity_id":2}],"total_count":3}`), nil
		case 3:
			return textResponse(r, 200, `{"items":[{"entity_id":3}],"total_count":3}`), nil
		}
		return textResponse(r, 200, `{"items":[],"total_count":3}`), nil
	}}
	c := rtClient(t, rt)
	sc := NewSearchCriteria().Filter("state", Eq, "2").Page(2, 2)
	var ids []int
	err := IterateInvoices(context.Background(), c, sc, func(inv Invoice) error {
		ids = append(ids, inv.EntityID)
		return nil
	})
	if err != nil || fmt.Sprint(ids) != "[1 2 3]" || rt.count() != 2 {
		t.Fatalf("IterateInvoices = %v, ids %v, %d requests", err, ids, rt.count())
	}
	if v, _ := sc.Values(); v.Get("searchCriteria[currentPage]") != "2" {
		t.Fatalf("IterateInvoices modified the caller's criteria: %v", v)
	}

	stop := errors.New("stop")
	if err := IterateInvoices(context.Background(), c, NewSearchCriteria().Filter("state", Eq, "2").Page(2, 2), func(Invoice) error { return stop }); !errors.Is(err, stop) {
		t.Fatalf("callback error = %v, want stop", err)
	}
}

// Every Iterate* function walks its own endpoint.
func TestIterateFunctionsUseTheirRoute(t *testing.T) {
	tests := []struct {
		path    string
		iterate func(ctx context.Context, c *Client) error
	}{
		{"/rest/V1/orders", func(ctx context.Context, c *Client) error {
			return IterateOrders(ctx, c, nil, func(Order) error { return nil })
		}},
		{"/rest/V1/creditmemos", func(ctx context.Context, c *Client) error {
			return IterateCreditMemos(ctx, c, nil, func(CreditMemo) error { return nil })
		}},
		{"/rest/V1/shipments", func(ctx context.Context, c *Client) error {
			return IterateShipments(ctx, c, nil, func(Shipment) error { return nil })
		}},
		{"/rest/V1/customers/search", func(ctx context.Context, c *Client) error {
			return IterateCustomers(ctx, c, nil, func(Customer) error { return nil })
		}},
		{"/rest/V1/inventory/source-items", func(ctx context.Context, c *Client) error {
			return IterateSourceItems(ctx, c, nil, func(SourceItem) error { return nil })
		}},
		{"/rest/V1/inventory/sources", func(ctx context.Context, c *Client) error {
			return IterateSources(ctx, c, nil, func(Source) error { return nil })
		}},
		{"/rest/V1/inventory/stocks", func(ctx context.Context, c *Client) error {
			return IterateStocks(ctx, c, nil, func(Stock) error { return nil })
		}},
		{"/rest/V1/carts/search", func(ctx context.Context, c *Client) error {
			return IterateCarts(ctx, c, nil, func(Cart) error { return nil })
		}},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rt, c := fixtureClient(t, `{"items":[{}],"total_count":1}`)
			if err := tt.iterate(context.Background(), c); err != nil {
				t.Fatal(err)
			}
			r := rt.last()
			if r.URL.EscapedPath() != tt.path || r.URL.Query().Get("searchCriteria[currentPage]") != "1" || r.URL.Query().Get("searchCriteria[pageSize]") != strconv.Itoa(DefaultPageSize) {
				t.Fatalf("requested %s", r.URL)
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	const doc = `{"swagger":"2.0","info":{"version":"2.4","title":"Magento Community"},
		"paths":{"/V1/orders/{id}":{"get":{"operationId":"GetV1OrdersId","tags":["salesOrderRepositoryV1"],
			"responses":{"200":{"description":"200 Success.","schema":{"$ref":"#/definitions/sales-data-order-interface"}}}}},
			"/V1/store/storeViews":{"get":{"responses":{"200":{"schema":{"type":"array","items":{"$ref":"#/definitions/store-data-store-interface"}}}}}}},
		"definitions":{"sales-data-order-interface":{"type":"object","properties":{"entity_id":{"type":"integer"},"items":{"type":"array","items":{"$ref":"#/definitions/sales-data-order-item-interface"}}},"required":["entity_id"]}}}`
	rt, c := fixtureClient(t, doc)
	s, err := GetSchema(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if got := rt.last().URL.EscapedPath(); got != "/rest/all/schema" {
		t.Fatalf("path = %q", got)
	}
	if s.Info.Version != "2.4" || s.Swagger != "2.0" || string(s.Raw) != doc {
		t.Fatalf("schema = %+v", s.Info)
	}
	op := s.Paths["/V1/orders/{id}"]["get"]
	if op.OperationID != "GetV1OrdersId" || op.Responses["200"].Schema.RefName() != "sales-data-order-interface" {
		t.Fatalf("operation = %+v", op)
	}
	if got := s.Paths["/V1/store/storeViews"]["get"].Responses["200"].Schema.RefName(); got != "store-data-store-interface" {
		t.Fatalf("array response ref = %q", got)
	}
	def := s.Definitions["sales-data-order-interface"]
	if def.Properties["entity_id"].Type != "integer" || def.Properties["items"].RefName() != "sales-data-order-item-interface" {
		t.Fatalf("definition = %+v", def)
	}

	// A body cap below the document size is refused, not decoded.
	c2 := rtClient(t, &recordingRT{respond: func(r *http.Request) (*http.Response, error) { return textResponse(r, 200, doc), nil }}, WithMaxBodyBytes(16))
	if _, err := GetSchema(context.Background(), c2); !errors.Is(err, ErrBodyTruncated) {
		t.Fatalf("GetSchema with a small cap = %v, want ErrBodyTruncated", err)
	}
}

func TestRoutesTable(t *testing.T) {
	rs := Routes()
	if len(rs) == 0 {
		t.Fatal("no routes")
	}
	seen := map[string]bool{}
	for _, r := range rs {
		key := r.Method + " " + r.Template
		if seen[key] {
			t.Errorf("duplicate route %s", key)
		}
		seen[key] = true
		switch r.Method {
		case "GET", "POST", "PUT", "DELETE":
		default:
			t.Errorf("%s: method", key)
		}
		if r.Template != "/schema" && !strings.HasPrefix(r.Template, "/V1/") {
			t.Errorf("%s: template must start with /V1/", key)
		}
		if strings.Contains(r.Template, ":") {
			t.Errorf("%s: parameters must be written {name}", key)
		}
		if r.Type != nil && reflect.TypeOf(r.Type).Kind() != reflect.Pointer {
			t.Errorf("%s: Type must be a pointer", key)
		}
	}
	for _, want := range []string{
		"GET /V1/orders", "GET /V1/orders/{id}", "GET /V1/invoices", "GET /V1/invoices/{id}",
		"GET /V1/creditmemos", "GET /V1/creditmemo/{id}", "GET /V1/shipments", "GET /V1/shipment/{id}",
		"GET /V1/customers/search", "GET /V1/customers/{customerId}",
		"GET /V1/inventory/source-items", "GET /V1/inventory/sources", "GET /V1/inventory/stocks",
		"GET /V1/inventory/get-product-salable-quantity/{sku}/{stockId}",
		"GET /V1/stockItems/{productSku}", "GET /V1/stockItems/lowStock/",
		"GET /V1/store/storeConfigs", "GET /V1/store/storeGroups", "GET /V1/store/storeViews", "GET /V1/store/websites",
		"GET /V1/carts/search", "GET /schema", "GET /V1/products", "POST /V1/integration/admin/token",
	} {
		if !seen[want] {
			t.Errorf("route %s not registered", want)
		}
	}
	rs[0].Template = "changed"
	if Routes()[0].Template == "changed" {
		t.Fatal("Routes returned the table itself, not a copy")
	}
}
