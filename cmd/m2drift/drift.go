package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	magento2 "github.com/florinel-chis/go-m2rest"
)

// Severity of a finding: Drift fails the check, Info is reported only.
type Severity string

const (
	Drift Severity = "DRIFT"
	Info  Severity = "INFO"
)

// Finding is one line of the report.
type Finding struct {
	Severity Severity
	Route    string // "GET /V1/orders"
	Subject  string // Go type and json tag, or the route
	Detail   string
}

var paramRe = regexp.MustCompile(`\{[^}]*\}`)

// normalize makes templates comparable whatever their parameter names:
// "/V1/orders/{id}" and "/V1/orders/{orderId}" both become "/V1/orders/{}".
func normalize(template string) string {
	return paramRe.ReplaceAllString(template, "{}")
}

// routeKey is method (upper case) + normalized template.
func routeKey(method, template string) string {
	return strings.ToUpper(method) + " " + normalize(template)
}

// compare checks every route against the schema. vendorRoutes (routeKey →
// declaring file) lists routes found in etc/webapi.xml; nil when -vendor was
// not given.
func compare(schema *magento2.Schema, routes []magento2.Route, vendorRoutes map[string]string) []Finding {
	paths := map[string]magento2.Operation{}
	for p, ops := range schema.Paths {
		for m, op := range ops {
			paths[routeKey(m, p)] = op
		}
	}
	var out []Finding
	for _, r := range routes {
		name := r.Method + " " + r.Template
		if r.Template == "/schema" {
			// The schema endpoint is not part of its own document; fetching it
			// is the check.
			continue
		}
		op, ok := paths[routeKey(r.Method, r.Template)]
		if !ok {
			if file, found := vendorRoutes[routeKey(r.Method, r.Template)]; found {
				out = append(out, Finding{Info, name, name, "not in the schema (hidden by token scope); declared in " + file})
			} else if vendorRoutes != nil {
				out = append(out, Finding{Drift, name, name, "route not in the schema nor in any etc/webapi.xml"})
			} else {
				out = append(out, Finding{Drift, name, name, "route not in the schema (run with -vendor to tell token scope from removal)"})
			}
			continue
		}
		if r.Type == nil {
			continue
		}
		def := r.Definition
		if def == "" {
			def = op.Responses["200"].Schema.RefName()
		}
		t := reflect.TypeOf(r.Type)
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || t == reflect.TypeOf(magento2.Schema{}) {
			continue
		}
		if def == "" {
			out = append(out, Finding{Info, name, t.Name(), "response has no schema definition to compare with"})
			continue
		}
		out = append(out, compareType(schema, name, t.Name(), t, def, map[string]bool{})...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity == Drift
		}
		if out[i].Route != out[j].Route {
			return out[i].Route < out[j].Route
		}
		return out[i].Subject < out[j].Subject
	})
	return dedupe(out)
}

var rawMessage = reflect.TypeOf(json.RawMessage(nil))

// compareType compares the json tags of struct t (reported as name: the Go
// type name, or the field path for an anonymous struct) with definition def
// and recurses into nested structs the property's $ref names. Extension
// attributes ("*-extension-interface") depend on the modules installed on
// the store, so a tag missing there is INFO, not drift.
func compareType(schema *magento2.Schema, route, name string, t reflect.Type, def string, seen map[string]bool) []Finding {
	key := t.PkgPath() + "." + name + "@" + def
	if seen[key] {
		return nil
	}
	seen[key] = true
	d, ok := schema.Definitions[def]
	if !ok {
		return []Finding{{Drift, route, name, fmt.Sprintf("definition %q not in the schema", def)}}
	}
	var out []Finding
	tags := map[string]reflect.Type{}
	collectTags(t, tags)
	for tag, ft := range tags {
		prop, ok := d.Properties[tag]
		if !ok {
			if strings.HasSuffix(def, "-extension-interface") {
				out = append(out, Finding{Info, route, name + "." + tag, fmt.Sprintf("extension attribute not in %s (its module is not installed on this store, or the token cannot see it)", def)})
			} else {
				out = append(out, Finding{Drift, route, name + "." + tag, fmt.Sprintf("json tag not in %s", def)})
			}
			continue
		}
		nested := ft
		for nested.Kind() == reflect.Pointer || nested.Kind() == reflect.Slice {
			if nested == rawMessage {
				break
			}
			nested = nested.Elem()
		}
		if nested.Kind() == reflect.Struct && nested != rawMessage {
			if ref := prop.RefName(); ref != "" {
				nestedName := nested.Name()
				if nestedName == "" {
					nestedName = name + "." + tag
				}
				out = append(out, compareType(schema, route, nestedName, nested, ref, seen)...)
			}
		}
	}
	var missing []string
	for p := range d.Properties {
		if _, ok := tags[p]; !ok {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		out = append(out, Finding{Info, route, name, fmt.Sprintf("%d schema properties of %s not in the type: %s", len(missing), def, strings.Join(missing, ", "))})
	}
	return out
}

// collectTags maps each json name of t's exported fields (embedded structs
// flattened) to the field type.
func collectTags(t reflect.Type, tags map[string]reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		// encoding/json promotes the fields of an untagged embedded struct,
		// exported or not.
		if f.Anonymous && name == "" {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				collectTags(ft, tags)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		tags[name] = f.Type
	}
}

// dedupe drops repeated findings: a type shared by several routes is
// reported under the first.
func dedupe(in []Finding) []Finding {
	seen := map[string]bool{}
	out := in[:0]
	for _, f := range in {
		k := string(f.Severity) + "|" + f.Subject + "|" + f.Detail
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, f)
	}
	return out
}

// Header is the report's preamble.
type Header struct {
	Host, SchemaVersion, SchemaTitle, MagentoVersion, Commit, Date, Vendor string
	Paths, Routes                                                          int
}

// render writes the Markdown report.
func render(h Header, findings []Finding) string {
	var drift, info int
	for _, f := range findings {
		if f.Severity == Drift {
			drift++
		} else {
			info++
		}
	}
	var b strings.Builder
	b.WriteString("# go-m2rest drift report\n\n")
	fmt.Fprintf(&b, "- Date: %s\n", h.Date)
	fmt.Fprintf(&b, "- Store: %s\n", h.Host)
	fmt.Fprintf(&b, "- Schema: %s, info.version %s, %d paths\n", h.SchemaTitle, h.SchemaVersion, h.Paths)
	if h.MagentoVersion != "" {
		fmt.Fprintf(&b, "- Magento (vendor composer.json): %s\n", h.MagentoVersion)
	}
	if h.Vendor != "" {
		fmt.Fprintf(&b, "- Vendor tree: %s\n", h.Vendor)
	}
	fmt.Fprintf(&b, "- go-m2rest commit: %s, %d registered routes\n", h.Commit, h.Routes)
	b.WriteString("- Token scope: /rest/all/schema lists only the routes the token may call; a route missing from it is DRIFT unless an etc/webapi.xml in the vendor tree declares it (then INFO).\n\n")
	fmt.Fprintf(&b, "**Summary: %d DRIFT, %d INFO**\n", drift, info)
	section := func(title string, sev Severity) {
		fmt.Fprintf(&b, "\n## %s\n\n", title)
		n := 0
		for _, f := range findings {
			if f.Severity != sev {
				continue
			}
			if n == 0 {
				b.WriteString("| Route | Subject | Finding |\n|---|---|---|\n")
			}
			n++
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", f.Route, f.Subject, strings.ReplaceAll(f.Detail, "|", "\\|"))
		}
		if n == 0 {
			b.WriteString("None.\n")
		}
	}
	section("DRIFT", Drift)
	section("INFO", Info)
	return b.String()
}
