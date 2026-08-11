package magento2

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlexBoolUnmarshal(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`true`, true},
		{`false`, false},
		{`0`, false},
		{`1`, true},
		{`2`, true}, // Magento is_filterable uses 0/1/2
		{`"0"`, false},
		{`"1"`, true},
		{`"2"`, true},
		{`"true"`, true},
		{`"false"`, false},
		{`1.0`, true},
		{`0.0`, false},
		{`null`, false},
		{`""`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var target struct {
				V FlexBool `json:"v"`
			}
			if err := json.Unmarshal([]byte(`{"v":`+tt.input+`}`), &target); err != nil {
				t.Fatalf("unmarshal %s failed: %v", tt.input, err)
			}
			if target.V.Bool() != tt.want {
				t.Errorf("FlexBool(%s) = %v, want %v", tt.input, target.V.Bool(), tt.want)
			}
		})
	}
}

func TestFlexBoolUnmarshalInvalid(t *testing.T) {
	var b FlexBool
	if err := json.Unmarshal([]byte(`"maybe"`), &b); err == nil {
		t.Error(`expected error for "maybe", got nil`)
	}
	if err := json.Unmarshal([]byte(`[1]`), &b); err == nil {
		t.Error("expected error for array input, got nil")
	}
}

func TestFlexBoolMarshal(t *testing.T) {
	gotTrue, err := json.Marshal(NewFlexBool(true))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(gotTrue) != "true" {
		t.Errorf("marshal NewFlexBool(true) = %s, want true", gotTrue)
	}

	gotFalse, err := json.Marshal(NewFlexBool(false))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(gotFalse) != "false" {
		t.Errorf("marshal NewFlexBool(false) = %s, want false", gotFalse)
	}
}

// The original JSON token must survive a decode/encode round-trip so that
// e.g. is_filterable=2 ("Filterable (no results)") is not silently
// downgraded to true/1 by a read-modify-write of an attribute.
func TestFlexBoolRoundTripPreservesRaw(t *testing.T) {
	tests := []struct{ in, want string }{
		{`{"v":2}`, `{"v":2}`},
		{`{"v":1}`, `{"v":1}`},
		{`{"v":0}`, `{"v":0}`},
		{`{"v":"2"}`, `{"v":"2"}`},
		{`{"v":"1"}`, `{"v":"1"}`},
		{`{"v":"true"}`, `{"v":"true"}`},
		{`{"v":true}`, `{"v":true}`},
		{`{"v":false}`, `{"v":false}`},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var target struct {
				V FlexBool `json:"v"`
			}
			if err := json.Unmarshal([]byte(tt.in), &target); err != nil {
				t.Fatalf("unmarshal %s failed: %v", tt.in, err)
			}
			out, err := json.Marshal(target)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("round-trip of %s = %s, want %s", tt.in, out, tt.want)
			}
		})
	}
}

// If the caller changes Value after unmarshaling, the stale raw token must
// not win: the new bool value is marshaled instead.
func TestFlexBoolMutatedValueOverridesRaw(t *testing.T) {
	var b FlexBool
	if err := json.Unmarshal([]byte(`2`), &b); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	b.Value = false
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(out) != "false" {
		t.Errorf("marshal after mutation = %s, want false", out)
	}
}

// Attribute-level round-trip: is_filterable=2 must survive re-marshaling the
// whole struct (this is what UpdateAttributeOnRemote sends back), and unset
// FlexBool fields must be omitted entirely (omitzero).
func TestAttributeIsFilterableRoundTrip(t *testing.T) {
	raw := `{"attribute_code":"color","frontend_input":"select","default_frontend_label":"Color","is_filterable":2,"is_filterable_in_search":"1"}`
	var a Attribute
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !a.IsFilterable.Bool() || !a.IsFilterableInSearch.Bool() {
		t.Fatalf("unexpected decode: %+v", a)
	}

	out, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(out), `"is_filterable":2`) {
		t.Errorf("marshaled attribute = %s, want is_filterable preserved as 2", out)
	}
	if !strings.Contains(string(out), `"is_filterable_in_search":"1"`) {
		t.Errorf(`marshaled attribute = %s, want is_filterable_in_search preserved as "1"`, out)
	}

	empty, err := json.Marshal(Attribute{AttributeCode: "x", FrontendInput: "text"})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(empty), "is_filterable") {
		t.Errorf("marshaled zero attribute = %s, want is_filterable omitted", empty)
	}
}
