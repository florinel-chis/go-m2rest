package magento2

import (
	"context"
	"encoding/json"
	"fmt"
)

// routeSchema is GET /rest/all/schema (module-webapi
// Controller/Rest/SchemaRequestProcessor): the store's Swagger 2.0 document,
// limited to the routes the token may call.
const routeSchema = "/schema"

// Schema is the store's REST schema. Raw is the document as received; Paths
// and Definitions are the parts a client needs to compare itself against.
type Schema struct {
	Raw         json.RawMessage                 `json:"-"`
	Swagger     string                          `json:"swagger"`
	Info        SchemaInfo                      `json:"info"`
	Paths       map[string]map[string]Operation `json:"paths"`
	Definitions map[string]Definition           `json:"definitions"`
}

// SchemaInfo is the schema's info block; Version is Magento's API version
// (e.g. "2.4"), not the release.
type SchemaInfo struct {
	Version string `json:"version"`
	Title   string `json:"title"`
}

// Operation is one method of a path. Responses maps a status ("200",
// "default") to its response.
type Operation struct {
	OperationID string                       `json:"operationId"`
	Tags        []string                     `json:"tags"`
	Description string                       `json:"description"`
	Parameters  []json.RawMessage            `json:"parameters"`
	Responses   map[string]OperationResponse `json:"responses"`
}

// OperationResponse is one response of an operation.
type OperationResponse struct {
	Description string   `json:"description"`
	Schema      Property `json:"schema"`
}

// Definition is a named type ("sales-data-order-interface").
type Definition struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Properties  map[string]Property `json:"properties"`
	Required    []string            `json:"required"`
}

// Property is a definition property or a response schema: a scalar Type, a
// Ref ("#/definitions/<name>"), or an array with Items.
type Property struct {
	Type        string    `json:"type"`
	Ref         string    `json:"$ref"`
	Items       *Property `json:"items"`
	Description string    `json:"description"`
}

// RefName returns the definition name p refers to, directly or as the item
// type of an array; empty when there is none.
func (p Property) RefName() string {
	const prefix = "#/definitions/"
	if len(p.Ref) > len(prefix) && p.Ref[:len(prefix)] == prefix {
		return p.Ref[len(prefix):]
	}
	if p.Items != nil {
		return p.Items.RefName()
	}
	return ""
}

// GetSchema fetches GET /rest/all/schema. The document is large (hundreds of
// KiB to a few MiB); the client's body cap (WithMaxBodyBytes) applies, and a
// cap below the document size makes this fail with ErrBodyTruncated.
func GetSchema(ctx context.Context, c *Client) (*Schema, error) {
	resp, err := c.Do(ctx, Request{Path: routeSchema, StoreCode: "all"})
	if err != nil {
		return nil, err
	}
	if resp.Truncated {
		return nil, fmt.Errorf("magento2: GET %s: %w", routeSchema, ErrBodyTruncated)
	}
	out := &Schema{}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("magento2: GET %s: decode schema: %w", routeSchema, err)
	}
	out.Raw = resp.Body
	return out, nil
}
