package magento2

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexBool is a bool that tolerates the mixed representations the Magento
// API uses for boolean-ish fields: JSON true/false, numbers (0 is false,
// anything else is true — is_filterable for example uses 0/1/2) and strings
// such as "0", "1", "2", "true" and "false". It always marshals as a plain
// JSON bool.
type FlexBool bool

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	switch raw {
	case "true":
		*b = true
		return nil
	case "false", "null":
		*b = false
		return nil
	}

	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool: %w", raw, err)
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "false":
			*b = false
			return nil
		case "true":
			*b = true
			return nil
		default:
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool", raw)
			}
			*b = f != 0
			return nil
		}
	}

	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("magento2: cannot unmarshal %s into FlexBool", raw)
	}
	*b = f != 0
	return nil
}

func (b FlexBool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}
