package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	magento2 "github.com/florinel-chis/go-m2rest"
)

const fixtureSchema = `{"swagger":"2.0","info":{"version":"2.4","title":"Magento Community"},
"paths":{
  "/V1/orders/{id}":{"get":{"responses":{"200":{"schema":{"$ref":"#/definitions/sales-data-order-interface"}}}}},
  "/V1/orders":{"get":{"responses":{"200":{"schema":{"$ref":"#/definitions/sales-data-order-search-result-interface"}}}}},
  "/V1/store/storeViews":{"get":{"responses":{"200":{"schema":{"type":"array","items":{"$ref":"#/definitions/store-data-store-interface"}}}}}},
  "/V1/plain":{"get":{"responses":{"200":{"schema":{"type":"string"}}}}},
  "/V1/untyped":{"get":{"responses":{"200":{"schema":{"type":"object"}}}}}
},
"definitions":{
  "sales-data-order-interface":{"type":"object","properties":{
    "entity_id":{"type":"integer"},"increment_id":{"type":"string"},"state":{"type":"string"},
    "items":{"type":"array","items":{"$ref":"#/definitions/sales-data-order-item-interface"}},
    "payment":{"$ref":"#/definitions/sales-data-order-payment-interface"},
    "extension_attributes":{"$ref":"#/definitions/sales-data-order-extension-interface"},
    "shipping":{"$ref":"#/definitions/sales-data-shipping-interface"}}},
  "sales-data-order-extension-interface":{"type":"object","properties":{"applied_taxes":{"type":"array"}}},
  "sales-data-shipping-interface":{"type":"object","properties":{"method":{"type":"string"}}},
  "sales-data-order-item-interface":{"type":"object","properties":{"item_id":{"type":"integer"},"sku":{"type":"string"}}},
  "sales-data-order-payment-interface":{"type":"object","properties":{"method":{"type":"string"}}},
  "sales-data-order-search-result-interface":{"type":"object","properties":{"items":{"type":"array","items":{"$ref":"#/definitions/sales-data-order-interface"}},"total_count":{"type":"integer"},"search_criteria":{"$ref":"#/definitions/framework-search-criteria-interface"}}},
  "store-data-store-interface":{"type":"object","properties":{"id":{"type":"integer"},"code":{"type":"string"}}},
  "pinned-interface":{"type":"object","properties":{"a":{"type":"string"}}}
}}`

type tOrderItem struct {
	ItemID int    `json:"item_id"`
	Sku    string `json:"sku"`
	Bogus  string `json:"bogus_item_field"` // DRIFT, found by recursion
}

type tPayment struct {
	Method string `json:"method"`
}

type tCommon struct {
	State string `json:"state"`
}

type tOrder struct {
	tCommon                     // embedded: flattened
	EntityID    int             `json:"entity_id"`
	IncrementID string          `json:"increment_id,omitempty"`
	Items       []tOrderItem    `json:"items"`
	Payment     *tPayment       `json:"payment"`
	Extension   json.RawMessage `json:"extension_attributes"` // not recursed
	Ignored     string          `json:"-"`
	unexported  string
	Wrong       string `json:"wrong_tag"` // DRIFT
	Shipping    struct {
		Method  string `json:"method"`
		Carrier string `json:"carrier"` // DRIFT, reported as tOrder.shipping.carrier
	} `json:"shipping"`
}

// tOrderExt decodes extension attributes: a tag the store's schema lacks is
// a module that is not installed there (INFO), not drift.
type tOrderExt struct {
	EntityID   int `json:"entity_id"`
	Extensions struct {
		AppliedTaxes []string `json:"applied_taxes"`
		GiftCards    []string `json:"gift_cards"`
	} `json:"extension_attributes"`
}

type tOrderList struct {
	Items      []tOrder `json:"items"`
	TotalCount int      `json:"total_count"`
}

type tStore struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

type tPinned struct {
	A string `json:"a"`
	B string `json:"b"` // DRIFT against the pinned definition
}

func mustSchema(t *testing.T) *magento2.Schema {
	t.Helper()
	s := &magento2.Schema{}
	if err := json.Unmarshal([]byte(fixtureSchema), s); err != nil {
		t.Fatal(err)
	}
	return s
}

func find(fs []Finding, sev Severity, subject string) *Finding {
	for i := range fs {
		if fs[i].Severity == sev && fs[i].Subject == subject {
			return &fs[i]
		}
	}
	return nil
}

func TestCompare(t *testing.T) {
	_ = tOrder{}.unexported
	routes := []magento2.Route{
		{Method: "GET", Template: "/V1/orders/{orderId}", Type: &tOrder{}}, // parameter name differs from the schema's {id}
		{Method: "GET", Template: "/V1/orders", Type: &tOrderList{}},
		{Method: "GET", Template: "/V1/store/storeViews", Type: &[]tStore{}},
		{Method: "GET", Template: "/V1/plain", Type: new(string)},
		{Method: "GET", Template: "/V1/untyped", Type: &tStore{}},
		{Method: "GET", Template: "/V1/plain", Type: &tPinned{}, Definition: "pinned-interface"},
		{Method: "POST", Template: "/V1/orders/{id}", Type: &tOrder{}}, // method missing; vendor declares it
		{Method: "GET", Template: "/V1/gone", Type: nil},               // nowhere
		{Method: "GET", Template: "/schema", Type: &magento2.Schema{}}, // skipped
		{Method: "GET", Template: "/V1/orders/{id}", Type: &tOrder{}, Definition: "missing-interface"},
		{Method: "GET", Template: "/V1/orders/{id}", Type: &tOrderExt{}},
	}
	vendor := map[string]string{routeKey("POST", "/V1/orders/{x}"): "magento/module-sales/etc/webapi.xml"}
	fs := compare(mustSchema(t), routes, vendor)

	wantDrift := map[string]string{
		"tOrder.wrong_tag":            "json tag not in sales-data-order-interface",
		"tOrderItem.bogus_item_field": "json tag not in sales-data-order-item-interface",
		"tPinned.b":                   "json tag not in pinned-interface",
		"GET /V1/gone":                "route not in the schema nor in any etc/webapi.xml",
		"tOrder":                      `definition "missing-interface" not in the schema`,
		"tOrder.shipping.carrier":     "json tag not in sales-data-shipping-interface",
	}
	for subject, detail := range wantDrift {
		f := find(fs, Drift, subject)
		if f == nil || f.Detail != detail {
			t.Errorf("DRIFT %s = %+v, want %q", subject, f, detail)
		}
	}
	wantInfo := map[string]string{
		"POST /V1/orders/{id}": "not in the schema (hidden by token scope); declared in magento/module-sales/etc/webapi.xml",
		"tOrderList":           "1 schema properties of sales-data-order-search-result-interface not in the type: search_criteria",
		"tStore":               "response has no schema definition to compare with",
		"tOrderExt.extension_attributes.gift_cards": "extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it)",
	}
	for subject, detail := range wantInfo {
		f := find(fs, Info, subject)
		if f == nil || f.Detail != detail {
			t.Errorf("INFO %s = %+v, want %q", subject, f, detail)
		}
	}
	// tOrder covers every property of its definition, the embedded tCommon's
	// "state" included.
	if f := find(fs, Info, "tOrder"); f != nil {
		t.Errorf("unexpected INFO for tOrder: %+v", f)
	}
	var drift int
	for _, f := range fs {
		if f.Severity == Drift {
			drift++
			if strings.Contains(f.Subject, "state") || strings.Contains(f.Subject, "extension_attributes") || strings.Contains(f.Subject, "Ignored") || strings.Contains(f.Subject, "unexported") {
				t.Errorf("false drift: %+v", f)
			}
		}
		if strings.Contains(f.Route, "/schema") {
			t.Errorf("the schema route was checked: %+v", f)
		}
	}
	if drift != len(wantDrift) {
		t.Errorf("%d DRIFT findings, want %d: %+v", drift, len(wantDrift), fs)
	}
	// DRIFT sorts first.
	if len(fs) == 0 || fs[0].Severity != Drift {
		t.Errorf("findings not sorted DRIFT first: %+v", fs)
	}

	// Without a vendor tree a route missing from the schema is DRIFT.
	fs = compare(mustSchema(t), []magento2.Route{{Method: "POST", Template: "/V1/orders/{id}"}}, nil)
	if len(fs) != 1 || fs[0].Severity != Drift || !strings.Contains(fs[0].Detail, "-vendor") {
		t.Errorf("no-vendor missing route = %+v", fs)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"/V1/orders/{id}":      "/V1/orders/{}",
		"/V1/orders/{orderId}": "/V1/orders/{}",
		"/V1/inventory/get-product-salable-quantity/{sku}/{stockId}": "/V1/inventory/get-product-salable-quantity/{}/{}",
		"/V1/stockItems/lowStock/":                                   "/V1/stockItems/lowStock/",
	} {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
	if routeKey("get", "/V1/x/{a}") != routeKey("GET", "/V1/x/{b}") {
		t.Error("routeKey is not case- and name-insensitive")
	}
}

func TestRender(t *testing.T) {
	out := render(Header{Host: "https://shop.example", SchemaVersion: "2.4", SchemaTitle: "Magento Community", Commit: "abc", Date: "2026-10-08 00:00 UTC", Paths: 3, Routes: 2},
		[]Finding{{Drift, "GET /V1/x", "T.a", "json tag | not in d"}, {Info, "GET /V1/y", "T", "hidden"}, {Info, "GET /V1/z", "U", "hidden"}})
	for _, want := range []string{
		"**Summary: 1 DRIFT, 2 INFO**", "- Store: https://shop.example", "info.version 2.4", "go-m2rest commit: abc",
		"| `GET /V1/x` | `T.a` | json tag \\| not in d |", "Token scope",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(render(Header{}, nil), "## DRIFT\n\nNone.") {
		t.Error("empty DRIFT section not rendered as None")
	}
}

func TestScanVendor(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, "magento", "module-sales", "etc")
	if err := os.MkdirAll(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0"?><routes><route url="/V1/orders/:id" method="GET"><service class="X" method="get"/></route><route url="/V1/stockItems/lowStock/" method="GET"/></routes>`
	if err := os.WriteFile(filepath.Join(mod, "webapi.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := scanVendor(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got[routeKey("GET", "/V1/orders/{orderId}")] != filepath.Join("magento", "module-sales", "etc", "webapi.xml") || got[routeKey("GET", "/V1/stockItems/lowStock/")] == "" {
		t.Fatalf("scanVendor = %v", got)
	}
	if _, err := scanVendor(t.TempDir()); err == nil {
		t.Fatal("an empty vendor tree must be an error")
	}
}

// run end to end against a stub store: the real route table, exit code 1 on
// drift, the report written to -out, and the token never in the output.
func TestRun(t *testing.T) {
	const token = "tok-SECRET-123"
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(fixtureSchema))
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "DRIFT.md")
	var stdout, stderr bytes.Buffer
	t.Setenv("MAGENTO_BEARER_TOKEN", token)
	code := run([]string{"-host", srv.URL, "-out", out}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d (stderr %s), want 1: the fixture schema lacks most registered routes", code, stderr.String())
	}
	if gotPath != "/rest/all/schema" || gotAuth != "Bearer "+token {
		t.Fatalf("requested %s with %q", gotPath, gotAuth)
	}
	report, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "# go-m2rest drift report") || strings.Contains(string(report), token) || strings.Contains(stderr.String(), token) {
		t.Fatalf("report:\n%s\nstderr: %s", report, stderr.String())
	}

	if code := run([]string{"-host", ""}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing host exit %d, want 2", code)
	}
}
