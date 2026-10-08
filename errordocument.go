package magento2

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Bounds on what an error document may become in memory: the body is
// server-supplied text.
const (
	// maxSubstituted bounds ErrorDocument.Substituted, in bytes.
	maxSubstituted = 2048
	// maxResourcesLen bounds ErrorDocument.Resources, in bytes.
	maxResourcesLen = 256
)

var utf8BOM = []byte("\xef\xbb\xbf")

// ErrorDocument is Magento's REST error body: a message with placeholders
// and their parameters, either named ({"resources": "..."} for "%resources")
// or positional (["a", "b"] for "%1", "%2"). Other keys (such as a
// developer-mode "trace") are not kept.
type ErrorDocument struct {
	Message    string          `json:"message"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// ParseErrorDocument parses body as a Magento error document. It tolerates
// a UTF-8 BOM and surrounding whitespace; ok is false unless body is a JSON
// object with a non-blank "message".
func ParseErrorDocument(body []byte) (ErrorDocument, bool) {
	body = bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimSpace(body), utf8BOM))
	if len(body) == 0 || body[0] != '{' {
		return ErrorDocument{}, false
	}
	var doc ErrorDocument
	if json.Unmarshal(body, &doc) != nil || strings.TrimSpace(doc.Message) == "" {
		return ErrorDocument{}, false
	}
	if bytes.Equal(doc.Parameters, []byte("null")) {
		doc.Parameters = nil
	}
	return doc, true
}

// Substituted returns the message with its placeholders replaced: named
// ("%key", longest key first so %fieldName is not eaten by %field) or
// positional ("%1", longest digit run). An unknown placeholder is left as it
// is. redact (nil = identity) is applied to the message and to every
// parameter before substitution, and to the result after it, since
// substitution can assemble a secret from parts. The result is trimmed and
// capped at 2048 bytes on a rune boundary.
func (d ErrorDocument) Substituted(redact func(string) string) string {
	if redact == nil {
		redact = func(s string) string { return s }
	}
	msg := redact(d.Message)
	switch {
	case len(d.Parameters) == 0:
		msg = truncateBytes(msg, maxSubstituted)
	case d.Parameters[0] == '{':
		var named map[string]json.RawMessage
		if json.Unmarshal(d.Parameters, &named) == nil {
			params := make(map[string]string, len(named))
			for k, v := range named {
				params[k] = redact(rawToString(v))
			}
			msg = substituteNamed(msg, params)
		}
	case d.Parameters[0] == '[':
		var positional []json.RawMessage
		if json.Unmarshal(d.Parameters, &positional) == nil {
			params := make([]string, len(positional))
			for i, v := range positional {
				params[i] = redact(rawToString(v))
			}
			msg = substitutePositional(msg, params)
		}
	}
	return truncateBytes(strings.TrimSpace(redact(truncateBytes(msg, maxSubstituted))), maxSubstituted)
}

// Resources returns the "resources" named parameter (the ACL resource of a
// 401 "The consumer isn't authorized to access %resources."), capped at 256
// bytes; empty when there is none. It is not redacted.
func (d ErrorDocument) Resources() string {
	if len(d.Parameters) == 0 || d.Parameters[0] != '{' {
		return ""
	}
	var named map[string]json.RawMessage
	if json.Unmarshal(d.Parameters, &named) != nil {
		return ""
	}
	raw, ok := named["resources"]
	if !ok {
		return ""
	}
	return truncateBytes(rawToString(raw), maxResourcesLen)
}

// substituteNamed replaces %key with its value, longest key first. Output
// is capped at maxSubstituted bytes; an unknown placeholder is left as it is.
func substituteNamed(msg string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	return substitute(msg, func(after string) (value string, n int) {
		for _, k := range keys {
			if strings.HasPrefix(after, k) {
				return params[k], len(k)
			}
		}
		return "", 0
	})
}

// substitutePositional replaces %N (1-based, longest digit run) with
// params[N-1]. Output is capped at maxSubstituted bytes; an out-of-range
// placeholder is left as it is.
func substitutePositional(msg string, params []string) string {
	return substitute(msg, func(after string) (value string, n int) {
		digits := 0
		for digits < len(after) && after[digits] >= '0' && after[digits] <= '9' {
			digits++
		}
		if digits == 0 {
			return "", 0
		}
		idx, err := strconv.Atoi(after[:digits])
		if err != nil || idx < 1 || idx > len(params) {
			return "", 0
		}
		return params[idx-1], digits
	})
}

// substitute copies msg into a builder, asking lookup at each '%' what
// follows; it stops at maxSubstituted bytes whatever the input, so a hostile
// body cannot amplify.
func substitute(msg string, lookup func(after string) (value string, n int)) string {
	var b strings.Builder
	write := func(s string) bool {
		room := maxSubstituted - b.Len()
		if room <= 0 {
			return false
		}
		if len(s) > room {
			s = truncateBytes(s, room)
		}
		b.WriteString(s)
		return b.Len() < maxSubstituted
	}
	for i := 0; i < len(msg); {
		if msg[i] != '%' {
			// Copy the run up to the next '%' whole, so the cap cuts it on
			// a rune boundary.
			j := strings.IndexByte(msg[i:], '%')
			if j < 0 {
				j = len(msg) - i
			}
			if !write(msg[i : i+j]) {
				break
			}
			i += j
			continue
		}
		value, n := lookup(msg[i+1:])
		if n == 0 {
			if !write("%") {
				break
			}
			i++
			continue
		}
		if !write(value) {
			break
		}
		i += 1 + n
	}
	return b.String()
}

// rawToString renders a JSON scalar as text: strings unquoted, anything else
// verbatim.
func rawToString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}

// truncateBytes cuts s to at most n bytes on a rune boundary.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
