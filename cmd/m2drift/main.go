// Command m2drift compares go-m2rest's routes and response types with a
// Magento store's REST schema (/rest/{store}/schema) and prints a Markdown
// report; it exits 1 when it finds drift.
//
//	MAGENTO_HOST=https://shop.example MAGENTO_BEARER_TOKEN=... \
//	    go run ./cmd/m2drift -vendor /path/to/magento/vendor -out DRIFT.md
//
// The schema only lists routes the token may call; with -vendor, a route
// missing from it but declared in an etc/webapi.xml is reported as INFO
// (hidden by token scope) instead of DRIFT.
package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("m2drift", flag.ContinueOnError)
	fs.SetOutput(stderr)
	host := fs.String("host", os.Getenv("MAGENTO_HOST"), "store root URL (default $MAGENTO_HOST)")
	token := fs.String("token", "", "bearer token (default $MAGENTO_BEARER_TOKEN; prefer the variable: flags show in ps)")
	store := fs.String("store", "all", "store code whose schema is fetched")
	vendor := fs.String("vendor", "", "Magento vendor directory; routes declared in its etc/webapi.xml but hidden from the schema are INFO, not DRIFT")
	outFile := fs.String("out", "", "write the report to this file instead of stdout")
	timeout := fs.Duration("timeout", 2*time.Minute, "overall timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *token == "" {
		*token = os.Getenv("MAGENTO_BEARER_TOKEN")
	}
	if *host == "" {
		fmt.Fprintln(stderr, "m2drift: -host or MAGENTO_HOST is required")
		return 2
	}
	redact := func(s string) string { return s }
	if *token != "" {
		tok := *token
		redact = func(s string) string { return strings.ReplaceAll(s, tok, "[redacted]") }
	}

	c, err := magento2.New(*host, magento2.WithToken(*token), magento2.WithAllowedMethods("GET"),
		magento2.WithRedactor(redact), magento2.WithUserAgent("go-m2rest-m2drift"))
	if err != nil {
		fmt.Fprintln(stderr, "m2drift:", err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	schema, err := fetchSchema(ctx, c, *store)
	if err != nil {
		fmt.Fprintln(stderr, "m2drift:", redact(err.Error()))
		return 2
	}

	var vendorRoutes map[string]string
	var magentoVersion string
	if *vendor != "" {
		if vendorRoutes, err = scanVendor(*vendor); err != nil {
			fmt.Fprintln(stderr, "m2drift:", err)
			return 2
		}
		magentoVersion = vendorVersion(*vendor)
	}

	routes := magento2.Routes()
	findings := compare(schema, routes, vendorRoutes)
	report := render(Header{
		Host:           *host,
		SchemaVersion:  schema.Info.Version,
		SchemaTitle:    schema.Info.Title,
		MagentoVersion: magentoVersion,
		Commit:         commit(),
		Date:           time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		Vendor:         *vendor,
		Paths:          len(schema.Paths),
		Routes:         len(routes),
	}, findings)

	w := stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintln(stderr, "m2drift:", err)
			return 2
		}
		defer f.Close()
		w = f
	}
	if _, err := io.WriteString(w, report); err != nil {
		fmt.Fprintln(stderr, "m2drift:", err)
		return 2
	}
	for _, f := range findings {
		if f.Severity == Drift {
			return 1
		}
	}
	return 0
}

func fetchSchema(ctx context.Context, c *magento2.Client, store string) (*magento2.Schema, error) {
	if store == "all" {
		return magento2.GetSchema(ctx, c)
	}
	resp, err := c.Do(ctx, magento2.Request{Path: "/schema", StoreCode: store})
	if err != nil {
		return nil, err
	}
	s := &magento2.Schema{}
	if err := json.Unmarshal(resp.Body, s); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	s.Raw = resp.Body
	return s, nil
}

var webapiParamRe = regexp.MustCompile(`:(\w+)`)

// scanVendor reads every */*/etc/webapi.xml under dir and returns routeKey →
// the declaring file (relative to dir).
func scanVendor(dir string) (map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*", "*", "etc", "webapi.xml"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no */*/etc/webapi.xml under " + dir)
	}
	out := map[string]string{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var doc struct {
			Routes []struct {
				URL    string `xml:"url,attr"`
				Method string `xml:"method,attr"`
			} `xml:"route"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		rel, _ := filepath.Rel(dir, f)
		for _, r := range doc.Routes {
			key := routeKey(r.Method, webapiParamRe.ReplaceAllString(r.URL, "{$1}"))
			if _, ok := out[key]; !ok {
				out[key] = rel
			}
		}
	}
	return out, nil
}

// vendorVersion reads the Magento release from the vendor tree, empty when
// unknown.
func vendorVersion(dir string) string {
	for _, f := range []string{
		filepath.Join(dir, "magento", "magento2-base", "composer.json"),
		filepath.Join(dir, "..", "composer.json"),
	} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var doc struct {
			Name    string            `json:"name"`
			Version string            `json:"version"`
			Require map[string]string `json:"require"`
		}
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		if v := doc.Require["magento/product-community-edition"]; v != "" {
			return "magento/product-community-edition " + v
		}
		if doc.Version != "" {
			return doc.Name + " " + doc.Version
		}
	}
	return ""
}

// commit is the go-m2rest revision: from the build info when stamped, else
// from git in the working directory; "-dirty" marks uncommitted changes.
func commit() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		var rev, dirty string
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value[:min(12, len(s.Value))]
			case "vcs.modified":
				if s.Value == "true" {
					dirty = "-dirty"
				}
			}
		}
		if rev != "" {
			return rev + dirty
		}
	}
	out, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	rev := strings.TrimSpace(string(out))
	if st, err := exec.Command("git", "status", "--porcelain").Output(); err == nil && len(strings.TrimSpace(string(st))) > 0 {
		rev += "-dirty"
	}
	return rev
}
