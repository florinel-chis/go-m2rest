package magento2

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// routeResponder answers by "METHOD path" and records every exchange.
func routeResponder(t *testing.T, routes map[string]string) (*recordingRT, *Client) {
	t.Helper()
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		key := r.Method + " " + r.URL.EscapedPath()
		body, ok := routes[key]
		if !ok {
			return textResponse(r, http.StatusNotFound, `{"message":"no route `+key+`"}`), nil
		}
		return textResponse(r, http.StatusOK, body), nil
	}}
	client, err := NewAPIClientFromIntegration(&StoreConfig{Scheme: "https", HostName: "shop.example", StoreCode: "default"}, "tok",
		WithHTTPClient(&http.Client{Transport: rt}), WithRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	return rt, client
}

func exchanges(rt *recordingRT) []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]string, len(rt.reqs))
	for i, r := range rt.reqs {
		out[i] = r.Method + " " + r.URL.EscapedPath()
		if r.URL.RawQuery != "" {
			out[i] += "?" + r.URL.RawQuery
		}
		if len(rt.bodies[i]) > 0 {
			out[i] += " " + string(rt.bodies[i])
		}
	}
	return out
}

func assertExchangePrefixes(t *testing.T, rt *recordingRT, want ...string) {
	t.Helper()
	got := exchanges(rt)
	if len(got) != len(want) {
		t.Fatalf("exchanges:\n  %s\nwant prefixes:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) {
			t.Errorf("exchange %d = %q, want prefix %q", i, got[i], want[i])
		}
	}
}

const v1 = "/rest/default/V1"

func TestProductServices(t *testing.T) {
	rt, c := routeResponder(t, map[string]string{
		"POST " + v1 + "/products":                   `{"sku":"a/b","name":"N"}`,
		"GET " + v1 + "/products/a%2Fb":              `{"sku":"a/b","name":"N2"}`,
		"PUT " + v1 + "/products/a%2Fb/stockItems/1": `1`,
	})
	mp, err := CreateOrReplaceProduct(&Product{Sku: "a/b"}, true, c)
	if err != nil {
		t.Fatal(err)
	}
	if mp.Product.Name != "N" || mp.Route != "/products/a%2Fb" {
		t.Fatalf("product = %+v route %q", mp.Product, mp.Route)
	}
	got, err := GetProductBySKU("a/b", c)
	if err != nil || got.Product.Name != "N2" {
		t.Fatalf("GetProductBySKU = %+v, %v", got, err)
	}
	if err := got.UpdateQuantityForStockItem("1", 5, true); err != nil {
		t.Fatal(err)
	}
	assertExchangePrefixes(t, rt,
		`POST `+v1+`/products {"product":{"sku":"a/b",`,
		`GET `+v1+`/products/a%2Fb`,
		`PUT `+v1+`/products/a%2Fb/stockItems/1 {"stockItem":{`,
	)

	_, err = GetProductBySKU("missing", c)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProductBySKU(missing) = %v, want ErrNotFound", err)
	}
}

func TestCartServices(t *testing.T) {
	rt, c := routeResponder(t, map[string]string{
		"POST " + v1 + "/guest-carts":                              `"q1"`,
		"GET " + v1 + "/guest-carts/q1":                            `{"id":7}`,
		"POST " + v1 + "/guest-carts/q1/items":                     `{"item_id":3}`,
		"POST " + v1 + "/guest-carts/q1/estimate-shipping-methods": `[{"carrier_code":"flatrate"}]`,
		"POST " + v1 + "/guest-carts/q1/shipping-information":      `{}`,
		"GET " + v1 + "/guest-carts/q1/payment-methods":            `[{"code":"checkmo"}]`,
		"PUT " + v1 + "/guest-carts/q1/order":                      `"42"`,
		"DELETE " + v1 + "/guest-carts/q1/items/3":                 `true`,
	})
	cart, err := NewGuestCartFromAPIClient(c)
	if err != nil {
		t.Fatal(err)
	}
	if cart.QuoteID != "q1" || cart.Route != "/guest-carts/q1" || cart.Cart.ID != 7 {
		t.Fatalf("cart = %+v", cart)
	}
	if err := cart.AddItems([]CartItem{{Sku: "s", Qty: 1}}); err != nil {
		t.Fatal(err)
	}
	carriers, err := cart.EstimateShippingCarrier(&ShippingAddress{})
	if err != nil || len(carriers) != 1 || carriers[0].CarrierCode != "flatrate" {
		t.Fatalf("carriers = %+v, %v", carriers, err)
	}
	if err := cart.AddShippingInformation(&AddressInformation{}); err != nil {
		t.Fatal(err)
	}
	methods, err := cart.EstimatePaymentMethods()
	if err != nil || len(methods) != 1 {
		t.Fatalf("methods = %+v, %v", methods, err)
	}
	order, err := cart.CreateOrder(methods[0])
	if err != nil || order.Order.EntityID != 42 || order.Route != "/orders/42" {
		t.Fatalf("order = %+v, %v", order, err)
	}
	if err := cart.DeleteItem(3); err != nil {
		t.Fatal(err)
	}
	assertExchangePrefixes(t, rt,
		"POST "+v1+"/guest-carts",
		"GET "+v1+"/guest-carts/q1",
		"POST "+v1+"/guest-carts/q1/items {\"cartItem\":",
		"POST "+v1+"/guest-carts/q1/estimate-shipping-methods {\"address\":",
		"POST "+v1+"/guest-carts/q1/shipping-information {\"addressInformation\":",
		"GET "+v1+"/guest-carts/q1/payment-methods",
		"PUT "+v1+"/guest-carts/q1/order {\"paymentMethod\":{\"method\":\"checkmo\"}}",
		"DELETE "+v1+"/guest-carts/q1/items/3",
	)
}

func TestCartAddItemNotFound(t *testing.T) {
	_, c := routeResponder(t, map[string]string{})
	cart := &MCart{Route: "/guest-carts/q1", QuoteID: "q1", Cart: &Cart{}, APIClient: c}
	err := cart.AddItems([]CartItem{{ItemID: 9, Sku: "s"}})
	var nf *ItemNotFoundError
	if !errors.As(err, &nf) || nf.ItemID != 9 {
		t.Fatalf("AddItems = %v, want *ItemNotFoundError", err)
	}
}

func TestAttributeServices(t *testing.T) {
	rt, c := routeResponder(t, map[string]string{
		"POST " + v1 + "/products/attributes":               `{"attribute_code":"color"}`,
		"GET " + v1 + "/products/attributes/color":          `{"attribute_code":"color","default_frontend_label":"Color"}`,
		"PUT " + v1 + "/products/attributes/color":          `{"attribute_code":"color"}`,
		"POST " + v1 + "/products/attributes/color/options": `"id_12"`,
	})
	a, err := CreateAttribute(&Attribute{AttributeCode: "color"}, c)
	if err != nil || a.Route != "/products/attributes/color" {
		t.Fatalf("CreateAttribute = %+v, %v", a, err)
	}
	got, err := GetAttributeByAttributeCode("color", c)
	if err != nil || got.Attribute.DefaultFrontendLabel != "Color" {
		t.Fatalf("GetAttributeByAttributeCode = %+v, %v", got, err)
	}
	if err := got.UpdateAttributeOnRemote(); err != nil {
		t.Fatal(err)
	}
	v, err := got.AddOption(Option{Label: "Red"})
	if err != nil || v != "12" {
		t.Fatalf("AddOption = %q, %v", v, err)
	}
	if n := rt.count(); n != 5 {
		t.Fatalf("requests = %d (%q)", n, exchanges(rt))
	}
}

func TestOrderServices(t *testing.T) {
	rt, c := routeResponder(t, map[string]string{
		"GET " + v1 + "/orders":              `{"items":[{"entity_id":42}]}`,
		"GET " + v1 + "/orders/42":           `{"entity_id":42,"increment_id":"000000042"}`,
		"POST " + v1 + "/orders/42/comments": `{"comment":"hi"}`,
	})
	o, err := GetOrderByIncrementID("000000042", c)
	if err != nil || o.Order.IncrementID != "000000042" || o.Route != "/orders/42" {
		t.Fatalf("GetOrderByIncrementID = %+v, %v", o, err)
	}
	if _, err := o.AddComment(&StatusHistory{Comment: "hi"}); err != nil {
		t.Fatal(err)
	}
	got := exchanges(rt)
	if len(got) != 3 || !strings.Contains(got[0], "searchCriteria%5Bfilter_groups%5D%5B2%5D%5Bfilters%5D%5B0%5D%5Bfield%5D=increment_id") || !strings.Contains(got[0], "fields=items%5Bentity_id%5D") {
		t.Fatalf("exchanges = %q", got)
	}
}

func TestCategoryAndAttributeSetServices(t *testing.T) {
	rt, c := routeResponder(t, map[string]string{
		"GET " + v1 + "/categories/list":                         `{"items":[{"id":5,"name":"Shoes"}]}`,
		"GET " + v1 + "/categories/5":                            `{"id":5,"name":"Shoes"}`,
		"GET " + v1 + "/categories/5/products":                   `[{"sku":"s","position":0,"category_id":"5"}]`,
		"PUT " + v1 + "/categories/5/products":                   `true`,
		"GET " + v1 + "/products/attribute-sets/sets/list":       `{"items":[{"attribute_set_id":4,"attribute_set_name":"Default"}]}`,
		"GET " + v1 + "/products/attribute-sets/4":               `{"attribute_set_id":4,"attribute_set_name":"Default"}`,
		"GET " + v1 + "/products/attribute-sets/groups/list":     `{"items":[{"attribute_group_id":"7"}]}`,
		"GET " + v1 + "/products/attribute-sets/4/attributes":    `[]`,
		"GET " + v1 + "/configurable-products/c%201/options/all": `[]`,
	})
	cat, err := GetCategoryByName("Shoes", c)
	if err != nil || cat.Route != "/categories/5" || len(*cat.Products) != 1 {
		t.Fatalf("GetCategoryByName = %+v, %v", cat, err)
	}
	if err := cat.AssignProductByProductLink(&ProductLink{Sku: "t"}); err != nil {
		t.Fatal(err)
	}
	set, err := GetAttributeSetByName("Default", c)
	if err != nil || set.AttributeSet.AttributeSetID != 4 || len(set.AttributeSetGroups) != 1 {
		t.Fatalf("GetAttributeSetByName = %+v, %v", set, err)
	}
	if _, err := GetConfigurableProductBySKU("c 1", c); err != nil {
		t.Fatal(err)
	}
	if n := rt.count(); n != 9 {
		t.Fatalf("requests = %d: %q", n, exchanges(rt))
	}
}
