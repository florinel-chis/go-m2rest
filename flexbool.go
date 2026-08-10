package magento2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexBool is a boolean-ish value that tolerates the mixed representations
// the Magento API uses: JSON true/false, numbers (0 is false, anything else
// is true — is_filterable for example uses 0/1/2) and strings such as "0",
// "1", "2", "true" and "false".
//
// It remembers the original JSON token, so values a plain bool cannot
// express survive a read-modify-write round-trip unchanged: an attribute
// stored with is_filterable=2 ("Filterable (no results)") is marshaled back
// as 2, not silently downgraded to true/1. If Value is changed after
// unmarshaling (so it no longer matches the remembered token), the plain
// bool representation of Value is marshaled instead.
type FlexBool struct {
	// Value is the parsed boolean interpretation.
	Value bool

	// raw is the original JSON token (only stored when it carries more
	// information than a plain bool, e.g. 2 or "1"); rawValue is the bool
	// it decoded to, used to detect Value being changed afterwards.
	raw      string
	rawValue bool
}

// NewFlexBool returns a FlexBool holding v that marshals as a plain JSON
// bool.
func NewFlexBool(v bool) FlexBool {
	return FlexBool{Value: v}
}

// Bool returns the parsed boolean interpretation.
func (b FlexBool) Bool() bool {
	return b.Value
}

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	switch raw {
	case "true":
		*b = FlexBool{Value: true}
		return nil
	case "false", "null":
		*b = FlexBool{}
		return nil
	}

	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool: %w", raw, err)
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "false":
			*b = FlexBool{Value: false, raw: raw, rawValue: false}
			return nil
		case "true":
			*b = FlexBool{Value: true, raw: raw, rawValue: true}
			return nil
		default:
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool", raw)
			}
			v := f != 0
			*b = FlexBool{Value: v, raw: raw, rawValue: v}
			return nil
		}
	}

	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool", raw)
	}
	v := f != 0
	*b = FlexBool{Value: v, raw: raw, rawValue: v}
	return nil
}

func (b FlexBool) MarshalJSON() ([]byte, error) {
	if b.raw != "" && b.rawValue == b.Value {
		return []byte(b.raw), nil
	}
	if b.Value {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}
