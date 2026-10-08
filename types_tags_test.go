package magento2

import (
	"encoding/json"
	"testing"
)

// Magento's custom-option key is is_require (ProductCustomOptionInterface::
// getIsRequire), not is_required.
func TestOptionsIsRequireTag(t *testing.T) {
	var o Options
	if err := json.Unmarshal([]byte(`{"option_id":1,"is_require":true}`), &o); err != nil || !o.IsRequired {
		t.Fatalf("Options = %+v, %v; is_require not decoded", o, err)
	}
}

// Order extension attributes that a search-and-replace once mangled
// ("point" → "pofloat64") decode under their real keys.
func TestOrderPointTags(t *testing.T) {
	var o Order
	body := `{"extension_attributes":{"reward_points_balance":12,
		"shipping_assignments":[{"shipping":{"extension_attributes":{"collection_point":{"collection_point_id":"cp1","name":"Locker"}}}}]}}`
	if err := json.Unmarshal([]byte(body), &o); err != nil {
		t.Fatal(err)
	}
	if o.ExtensionAttributes.RewardPointsBalance != 12 {
		t.Errorf("reward_points_balance not decoded: %+v", o.ExtensionAttributes)
	}
	cp := o.ExtensionAttributes.ShippingAssignments[0].Shipping.ExtensionAttributes.CollectionPoint
	if cp == nil || cp.CollectionPointID != "cp1" || cp.Name != "Locker" {
		t.Errorf("collection_point not decoded: %+v", cp)
	}
}

// Order addresses are sales order addresses (OrderAddressInterface), not
// quote addresses.
func TestOrderAddressTags(t *testing.T) {
	addr := `{"entity_id":5,"parent_id":42,"address_type":"billing","customer_address_id":3,"customer_id":1,
		"firstname":"Veronica","lastname":"Costello","street":["6146 Honey Bluff Parkway"],"city":"Calder","region":"Michigan",
		"region_code":"MI","region_id":33,"postcode":"49628-7978","country_id":"US","telephone":"(555) 229-3326",
		"email":"roni_cost@example.com","vat_id":"V1","vat_is_valid":1,"vat_request_id":"R1","vat_request_date":"2026-01-01","vat_request_success":1}`
	body := `{"billing_address":` + addr + `,"extension_attributes":{"shipping_assignments":[{"shipping":{"address":` + addr + `}}]}}`
	var o Order
	if err := json.Unmarshal([]byte(body), &o); err != nil {
		t.Fatal(err)
	}
	roundTrip(t, addr, o.BillingAddress)
	roundTrip(t, addr, o.ExtensionAttributes.ShippingAssignments[0].Shipping.Address)
	if o.BillingAddress.EntityID != 5 || o.BillingAddress.AddressType != "billing" {
		t.Fatalf("billing address = %+v", o.BillingAddress)
	}
}

// Customer addresses are customer addresses (customer AddressInterface):
// region is an object, default_billing/default_shipping are booleans.
func TestCustomerAddressTags(t *testing.T) {
	addr := `{"id":1,"customer_id":1,"region":{"region_code":"MI","region":"Michigan","region_id":33},"region_id":33,
		"country_id":"US","street":["6146 Honey Bluff Parkway"],"company":"ACME","telephone":"(555) 229-3326","fax":"1",
		"postcode":"49628-7978","city":"Calder","firstname":"Veronica","lastname":"Costello","middlename":"M","prefix":"Ms",
		"suffix":"Jr","vat_id":"V1","default_shipping":true,"default_billing":true}`
	var c Customer
	if err := json.Unmarshal([]byte(`{"id":1,"addresses":[`+addr+`]}`), &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Addresses) != 1 || c.Addresses[0].Region == nil || c.Addresses[0].Region.RegionCode != "MI" || !c.Addresses[0].DefaultBilling {
		t.Fatalf("addresses = %+v", c.Addresses)
	}
	roundTrip(t, addr, c.Addresses[0])
}

// /V1/configurable-products/{sku}/options/all answers configurable-product
// options (OptionInterface), not attribute options.
func TestConfigurableOptionsDecode(t *testing.T) {
	const opts = `[{"id":3,"attribute_id":"93","label":"Color","position":0,"is_use_default":false,"values":[{"value_index":49},{"value_index":52}],"product_id":67}]`
	_, c := fixtureClient(t, opts)
	mc, err := GetConfigurableProductBySKU("MH01", c)
	if err != nil {
		t.Fatal(err)
	}
	got := *mc.Options
	if len(got) != 1 || got[0].AttributeID != "93" || len(got[0].Values) != 2 || got[0].Values[1].ValueIndex != 52 {
		t.Fatalf("options = %+v", got)
	}
	roundTrip(t, opts, got)
}
