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
