package magento2

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// Inventory read routes: MSI (module-inventory-api, module-inventory-sales-api)
// and the legacy CatalogInventory stock registry.
const (
	routeSourceItems   = "/V1/inventory/source-items"                                 // GET module-inventory-api/etc/webapi.xml:92
	routeSources       = "/V1/inventory/sources"                                      // GET module-inventory-api/etc/webapi.xml:11
	routeStocks        = "/V1/inventory/stocks"                                       // GET module-inventory-api/etc/webapi.xml:42
	routeSalableQty    = "/V1/inventory/get-product-salable-quantity/{sku}/{stockId}" // GET module-inventory-sales-api/etc/webapi.xml:10
	routeStockItem     = "/V1/stockItems/{productSku}"                                // GET module-catalog-inventory/etc/webapi.xml:10
	routeLowStockItems = "/V1/stockItems/lowStock/"                                   // GET module-catalog-inventory/etc/webapi.xml:22 (trailing slash as declared)
)

// SourceItem is the quantity of one SKU at one source
// (module-inventory-api/Api/Data/SourceItemInterface.php).
type SourceItem struct {
	Sku                 string          `json:"sku"`
	SourceCode          string          `json:"source_code"`
	Quantity            float64         `json:"quantity"`
	Status              int             `json:"status"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes,omitempty"`
}

// Source is an MSI source (module-inventory-api/Api/Data/SourceInterface.php).
type Source struct {
	SourceCode              string              `json:"source_code"`
	Name                    string              `json:"name"`
	ContactName             string              `json:"contact_name,omitempty"`
	Email                   string              `json:"email,omitempty"`
	Enabled                 bool                `json:"enabled"`
	Description             string              `json:"description,omitempty"`
	Latitude                float64             `json:"latitude,omitempty"`
	Longitude               float64             `json:"longitude,omitempty"`
	CountryID               string              `json:"country_id,omitempty"`
	RegionID                int                 `json:"region_id,omitempty"`
	Region                  string              `json:"region,omitempty"`
	City                    string              `json:"city,omitempty"`
	Street                  string              `json:"street,omitempty"`
	Postcode                string              `json:"postcode,omitempty"`
	Phone                   string              `json:"phone,omitempty"`
	Fax                     string              `json:"fax,omitempty"`
	UseDefaultCarrierConfig bool                `json:"use_default_carrier_config"`
	CarrierLinks            []SourceCarrierLink `json:"carrier_links,omitempty"`
	ExtensionAttributes     json.RawMessage     `json:"extension_attributes,omitempty"`
}

// SourceCarrierLink links a source to a shipping carrier
// (module-inventory-api/Api/Data/SourceCarrierLinkInterface.php).
type SourceCarrierLink struct {
	CarrierCode         string          `json:"carrier_code"`
	Position            int             `json:"position"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes,omitempty"`
}

// Stock is an MSI stock (module-inventory-api/Api/Data/StockInterface.php);
// its sales channels are in ExtensionAttributes.
type Stock struct {
	StockID             int             `json:"stock_id"`
	Name                string          `json:"name"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes,omitempty"`
}

// SourceItemListResponse is a page of GET /V1/inventory/source-items.
type SourceItemListResponse struct {
	Items      []SourceItem `json:"items"`
	TotalCount int          `json:"total_count"`
}

// SourceListResponse is a page of GET /V1/inventory/sources.
type SourceListResponse struct {
	Items      []Source `json:"items"`
	TotalCount int      `json:"total_count"`
}

// StockListResponse is a page of GET /V1/inventory/stocks.
type StockListResponse struct {
	Items      []Stock `json:"items"`
	TotalCount int     `json:"total_count"`
}

// StockItemListResponse is a page of GET /V1/stockItems/lowStock/.
type StockItemListResponse struct {
	Items      []StockItem `json:"items"`
	TotalCount int         `json:"total_count"`
}

// GetSourceItemsPage fetches one page of GET /V1/inventory/source-items;
// sc may be nil.
func GetSourceItemsPage(ctx context.Context, c *Client, sc *SearchCriteria) (*SourceItemListResponse, error) {
	out := &SourceItemListResponse{}
	if err := getSearchPage(ctx, c, routeSourceItems, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateSourceItems pages through GET /V1/inventory/source-items.
func IterateSourceItems(ctx context.Context, c *Client, sc *SearchCriteria, fn func(SourceItem) error) error {
	return iterateList(ctx, c, routeSourceItems, sc, fn)
}

// GetSourcesPage fetches one page of GET /V1/inventory/sources; sc may be
// nil.
func GetSourcesPage(ctx context.Context, c *Client, sc *SearchCriteria) (*SourceListResponse, error) {
	out := &SourceListResponse{}
	if err := getSearchPage(ctx, c, routeSources, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateSources pages through GET /V1/inventory/sources.
func IterateSources(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Source) error) error {
	return iterateList(ctx, c, routeSources, sc, fn)
}

// GetStocksPage fetches one page of GET /V1/inventory/stocks; sc may be
// nil.
func GetStocksPage(ctx context.Context, c *Client, sc *SearchCriteria) (*StockListResponse, error) {
	out := &StockListResponse{}
	if err := getSearchPage(ctx, c, routeStocks, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateStocks pages through GET /V1/inventory/stocks.
func IterateStocks(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Stock) error) error {
	return iterateList(ctx, c, routeStocks, sc, fn)
}

// GetSalableQuantity returns the salable quantity of sku in stock stockID
// (GET /V1/inventory/get-product-salable-quantity/{sku}/{stockId}).
func GetSalableQuantity(ctx context.Context, c *Client, sku string, stockID int) (float64, error) {
	var qty float64
	path := "/V1/inventory/get-product-salable-quantity/" + url.PathEscape(sku) + "/" + strconv.Itoa(stockID)
	if err := c.DoJSON(ctx, Request{Path: path}, &qty); err != nil {
		return 0, err
	}
	return qty, nil
}

// GetStockItem returns the legacy (single-source) stock item of a product
// (GET /V1/stockItems/{productSku}).
func GetStockItem(ctx context.Context, c *Client, sku string) (*StockItem, error) {
	out := &StockItem{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/stockItems/" + url.PathEscape(sku)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetLowStockItems returns legacy stock items whose quantity is below qty in
// scope scopeID (GET /V1/stockItems/lowStock/; StockRegistryInterface::
// getLowStockItems). currentPage starts at 1; pageSize 0 lets Magento choose.
func GetLowStockItems(ctx context.Context, c *Client, scopeID int, qty float64, currentPage, pageSize int) (*StockItemListResponse, error) {
	q := url.Values{
		"scopeId":     {strconv.Itoa(scopeID)},
		"qty":         {strconv.FormatFloat(qty, 'f', -1, 64)},
		"currentPage": {strconv.Itoa(max(currentPage, 1))},
		"pageSize":    {strconv.Itoa(max(pageSize, 0))},
	}
	out := &StockItemListResponse{}
	if err := c.DoJSON(ctx, Request{Path: routeLowStockItems, Query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}
