package magento2

import (
	"encoding/json"
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
			if bool(target.V) != tt.want {
				t.Errorf("FlexBool(%s) = %v, want %v", tt.input, bool(target.V), tt.want)
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
	gotTrue, err := json.Marshal(FlexBool(true))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(gotTrue) != "true" {
		t.Errorf("marshal FlexBool(true) = %s, want true", gotTrue)
	}

	gotFalse, err := json.Marshal(FlexBool(false))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(gotFalse) != "false" {
		t.Errorf("marshal FlexBool(false) = %s, want false", gotFalse)
	}
}
