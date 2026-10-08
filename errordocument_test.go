package magento2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseErrorDocument(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		wantOK        bool
		wantMessage   string // Substituted(nil)
		wantResources string
	}{
		{
			name:          "parameters as a map (ACL)",
			body:          `{"message":"The consumer isn't authorized to access %resources.","parameters":{"resources":"Magento_Sales::actions_view"},"trace":"#0 ..."}`,
			wantOK:        true,
			wantMessage:   "The consumer isn't authorized to access Magento_Sales::actions_view.",
			wantResources: "Magento_Sales::actions_view",
		},
		{
			name:        "parameters as a list",
			body:        `{"message":"The entity that was requested doesn't exist. Verify the entity and try again. %1 %2","parameters":["order","99"]}`,
			wantOK:      true,
			wantMessage: "The entity that was requested doesn't exist. Verify the entity and try again. order 99",
		},
		{
			name:        "positional placeholders above nine",
			body:        `{"message":"%1 %10 %11","parameters":["a","b","c","d","e","f","g","h","i","j","k"]}`,
			wantOK:      true,
			wantMessage: "a j k",
		},
		{
			name:        "parameters absent",
			body:        `{"message":"Invalid searchCriteria."}`,
			wantOK:      true,
			wantMessage: "Invalid searchCriteria.",
		},
		{
			name:        "parameters null",
			body:        `{"message":"Invalid %1.","parameters":null}`,
			wantOK:      true,
			wantMessage: "Invalid %1.",
		},
		{
			name:        "parameters of another type are ignored",
			body:        `{"message":"Invalid %1.","parameters":"x"}`,
			wantOK:      true,
			wantMessage: "Invalid %1.",
		},
		{
			name:        "map with a non-string value and a placeholder that is a prefix of another",
			body:        `{"message":"%fieldName has %field %count","parameters":{"fieldName":"sku","field":"x","count":3}}`,
			wantOK:      true,
			wantMessage: "sku has x 3",
		},
		{
			name:        "unknown placeholders are left as they are",
			body:        `{"message":"%missing stays, %x goes, %2 stays","parameters":{"x":"X"}}`,
			wantOK:      true,
			wantMessage: "%missing stays, X goes, %2 stays",
		},
		{
			name:        "UTF-8 BOM and surrounding whitespace",
			body:        "\xef\xbb\xbf \n" + `{"message":"Nope %1","parameters":["a"]}` + "\n",
			wantOK:      true,
			wantMessage: "Nope a",
		},
		{name: "not JSON (a proxy page)", body: "<html><body>Bad Gateway</body></html>"},
		{name: "empty body", body: ""},
		{name: "JSON without a message", body: `{"error":"nope"}`},
		{name: "blank message", body: `{"message":"  "}`},
		{name: "JSON array", body: `["a","b"]`},
		{name: "invalid JSON", body: `{"message":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, ok := ParseErrorDocument([]byte(tt.body))
			if ok != tt.wantOK {
				t.Fatalf("ParseErrorDocument ok = %v, want %v (%+v)", ok, tt.wantOK, doc)
			}
			if !ok {
				return
			}
			if got := doc.Substituted(nil); got != tt.wantMessage {
				t.Errorf("Substituted = %q, want %q", got, tt.wantMessage)
			}
			if got := doc.Resources(); got != tt.wantResources {
				t.Errorf("Resources = %q, want %q", got, tt.wantResources)
			}
		})
	}
}

func TestErrorDocumentSubstitutedRedacts(t *testing.T) {
	const secret = "tok-SECRET"
	redact := func(s string) string { return strings.ReplaceAll(s, secret, "[redacted]") }
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "in the message", body: `{"message":"bad ` + secret + `"}`, want: "bad [redacted]"},
		{name: "in a named parameter", body: `{"message":"bad %p","parameters":{"p":"` + secret + `"}}`, want: "bad [redacted]"},
		{name: "in a positional parameter", body: `{"message":"bad %1","parameters":["` + secret + `"]}`, want: "bad [redacted]"},
		// Substitution can assemble the secret from a message part and a
		// parameter part; the result is redacted once more.
		{name: "assembled by substitution", body: `{"message":"tok-%1","parameters":["SECRET"]}`, want: "[redacted]"},
		// A parameter is redacted before it is substituted, so a placeholder
		// inside a secret cannot survive either.
		{name: "parameter value is redacted before substitution", body: `{"message":"%a","parameters":{"a":"` + secret + `%b","b":"x"}}`, want: "[redacted]%b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, ok := ParseErrorDocument([]byte(tt.body))
			if !ok {
				t.Fatal("not parsed")
			}
			if got := doc.Substituted(redact); got != tt.want {
				t.Fatalf("Substituted = %q, want %q", got, tt.want)
			}
		})
	}
}

// A hostile error body cannot make substitution allocate past a fixed bound,
// and the ACL resource is bounded on its own.
func TestErrorDocumentIsBounded(t *testing.T) {
	body := `{"message":"` + strings.Repeat("%x", 100_000) + `","parameters":{"x":"` + strings.Repeat("y", 100_000) +
		`","resources":"` + strings.Repeat("r", 10_000) + `"}}`
	doc, ok := ParseErrorDocument([]byte(body))
	if !ok {
		t.Fatal("not parsed")
	}
	if got := doc.Substituted(nil); len(got) > maxSubstituted {
		t.Fatalf("Substituted is %d bytes, want <= %d", len(got), maxSubstituted)
	}
	if got := doc.Resources(); len(got) != maxResourcesLen {
		t.Fatalf("len(Resources) = %d, want %d", len(got), maxResourcesLen)
	}
	long := ErrorDocument{Message: strings.Repeat("a", 5000)}
	if got := long.Substituted(nil); len(got) != maxSubstituted {
		t.Fatalf("a long message without placeholders is %d bytes, want %d", len(got), maxSubstituted)
	}

	named := map[string]string{"x": strings.Repeat("y", 100_000)}
	if out := substituteNamed(strings.Repeat("%x", 100_000), named); len(out) > maxSubstituted {
		t.Fatalf("substituteNamed produced %d bytes, want <= %d", len(out), maxSubstituted)
	}
	if out := substitutePositional(strings.Repeat("%1", 100_000), []string{strings.Repeat("y", 100_000)}); len(out) > maxSubstituted {
		t.Fatalf("substitutePositional produced %d bytes, want <= %d", len(out), maxSubstituted)
	}
	if out := substituteNamed(strings.Repeat("%x", 1000), map[string]string{"x": "é"}); !utf8.ValidString(out) || len(out) > maxSubstituted {
		t.Fatalf("substituteNamed cut inside a rune or over the cap: %d bytes, valid=%v", len(out), utf8.ValidString(out))
	}
	// An odd prefix puts the cap in the middle of a multi-byte rune; the cut must back up.
	for _, r := range []string{"é", "€", "😀"} {
		out := substituteNamed("a%x", map[string]string{"x": strings.Repeat(r, 2000)})
		if !utf8.ValidString(out) || len(out) > maxSubstituted || len(out) < maxSubstituted-utf8.UTFMax {
			t.Fatalf("substituteNamed(%q) cut inside a rune or too short: %d bytes, valid=%v", r, len(out), utf8.ValidString(out))
		}
	}
	// The cap can also fall inside a multi-byte rune of the message itself.
	for _, body := range []string{
		`{"message":"a` + strings.Repeat("é", 2000) + `"}`,
		`{"message":"a` + strings.Repeat("é", 2000) + `%1","parameters":["x"]}`,
		`{"message":"a` + strings.Repeat("é", 2000) + `%x","parameters":{"x":"y"}}`,
	} {
		doc, _ := ParseErrorDocument([]byte(body))
		if out := doc.Substituted(nil); !utf8.ValidString(out) || len(out) > maxSubstituted || len(out) < maxSubstituted-utf8.UTFMax {
			t.Fatalf("Substituted cut inside a rune of the message or too short: %d bytes, valid=%v", len(out), utf8.ValidString(out))
		}
	}
	if out := substitutePositional("%1 %3 %2", []string{"a", "b"}); out != "a %3 b" {
		t.Fatalf("substitutePositional = %q", out)
	}
	if out := substitutePositional("100% done %", []string{"a"}); out != "100% done %" {
		t.Fatalf("substitutePositional = %q", out)
	}
}

// APIError.Message is the substituted, redacted error document.
func TestAPIErrorMessageUsesErrorDocument(t *testing.T) {
	const secret = "s3cr3t"
	rt := &recordingRT{respond: func(r *http.Request) (*http.Response, error) {
		return textResponse(r, http.StatusUnauthorized, `{"message":"The consumer isn't authorized to access %resources.","parameters":{"resources":"Magento_Sales::actions_view `+secret+`"}}`), nil
	}}
	c := rtClient(t, rt, WithRedactor(func(s string) string { return strings.ReplaceAll(s, secret, "[r]") }))
	_, err := c.Do(context.Background(), Request{Path: "/V1/orders"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Do = %v", err)
	}
	want := "The consumer isn't authorized to access Magento_Sales::actions_view [r]."
	if apiErr.Message != want {
		t.Fatalf("Message = %q, want %q", apiErr.Message, want)
	}
	if apiErr.Error() != "magento2: GET /V1/orders: status 401: "+want {
		t.Fatalf("Error() = %q", apiErr.Error())
	}
}
