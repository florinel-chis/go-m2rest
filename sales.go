package magento2

import (
	"context"
	"strconv"
)

// Sales read routes (module-sales/etc/webapi.xml).
const (
	routeOrders      = "/V1/orders"          // GET  module-sales/etc/webapi.xml:16
	routeOrder       = "/V1/orders/{id}"     // GET  module-sales/etc/webapi.xml:10
	routeInvoices    = "/V1/invoices"        // GET  module-sales/etc/webapi.xml:94
	routeInvoice     = "/V1/invoices/{id}"   // GET module-sales/etc/webapi.xml:88
	routeCreditMemos = "/V1/creditmemos"     // GET  module-sales/etc/webapi.xml:148
	routeCreditMemo  = "/V1/creditmemo/{id}" // GET module-sales/etc/webapi.xml:154
	routeShipments   = "/V1/shipments"       // GET  module-sales/etc/webapi.xml:202
	routeShipment    = "/V1/shipment/{id}"   // GET module-sales/etc/webapi.xml:196
)

// OrderListResponse is a page of GET /V1/orders.
type OrderListResponse struct {
	Items      []Order `json:"items"`
	TotalCount int     `json:"total_count"`
}

// InvoiceListResponse is a page of GET /V1/invoices.
type InvoiceListResponse struct {
	Items      []Invoice `json:"items"`
	TotalCount int       `json:"total_count"`
}

// CreditMemoListResponse is a page of GET /V1/creditmemos.
type CreditMemoListResponse struct {
	Items      []CreditMemo `json:"items"`
	TotalCount int          `json:"total_count"`
}

// ShipmentListResponse is a page of GET /V1/shipments.
type ShipmentListResponse struct {
	Items      []Shipment `json:"items"`
	TotalCount int        `json:"total_count"`
}

// GetOrdersPage fetches one page of GET /V1/orders; sc may be nil.
func GetOrdersPage(ctx context.Context, c *Client, sc *SearchCriteria) (*OrderListResponse, error) {
	out := &OrderListResponse{}
	if err := getSearchPage(ctx, c, routeOrders, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateOrders pages through GET /V1/orders, calling fn for every order
// (see iterateSearch for the paging rules). An error from fn aborts.
func IterateOrders(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Order) error) error {
	return iterateList(ctx, c, routeOrders, sc, fn)
}

// GetOrder fetches one order (GET /V1/orders/{id}).
func GetOrder(ctx context.Context, c *Client, id int) (*Order, error) {
	out := &Order{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/orders/" + strconv.Itoa(id)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetInvoicesPage fetches one page of GET /V1/invoices; sc may be nil.
func GetInvoicesPage(ctx context.Context, c *Client, sc *SearchCriteria) (*InvoiceListResponse, error) {
	out := &InvoiceListResponse{}
	if err := getSearchPage(ctx, c, routeInvoices, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateInvoices pages through GET /V1/invoices.
func IterateInvoices(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Invoice) error) error {
	return iterateList(ctx, c, routeInvoices, sc, fn)
}

// GetInvoice fetches one invoice (GET /V1/invoices/{id}).
func GetInvoice(ctx context.Context, c *Client, id int) (*Invoice, error) {
	out := &Invoice{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/invoices/" + strconv.Itoa(id)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreditMemosPage fetches one page of GET /V1/creditmemos; sc may be nil.
func GetCreditMemosPage(ctx context.Context, c *Client, sc *SearchCriteria) (*CreditMemoListResponse, error) {
	out := &CreditMemoListResponse{}
	if err := getSearchPage(ctx, c, routeCreditMemos, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateCreditMemos pages through GET /V1/creditmemos.
func IterateCreditMemos(ctx context.Context, c *Client, sc *SearchCriteria, fn func(CreditMemo) error) error {
	return iterateList(ctx, c, routeCreditMemos, sc, fn)
}

// GetCreditMemo fetches one credit memo (GET /V1/creditmemo/{id} — singular
// in Magento's route).
func GetCreditMemo(ctx context.Context, c *Client, id int) (*CreditMemo, error) {
	out := &CreditMemo{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/creditmemo/" + strconv.Itoa(id)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetShipmentsPage fetches one page of GET /V1/shipments; sc may be nil.
func GetShipmentsPage(ctx context.Context, c *Client, sc *SearchCriteria) (*ShipmentListResponse, error) {
	out := &ShipmentListResponse{}
	if err := getSearchPage(ctx, c, routeShipments, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateShipments pages through GET /V1/shipments.
func IterateShipments(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Shipment) error) error {
	return iterateList(ctx, c, routeShipments, sc, fn)
}

// GetShipment fetches one shipment (GET /V1/shipment/{id} — singular in
// Magento's route).
func GetShipment(ctx context.Context, c *Client, id int) (*Shipment, error) {
	out := &Shipment{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/shipment/" + strconv.Itoa(id)}, out); err != nil {
		return nil, err
	}
	return out, nil
}
