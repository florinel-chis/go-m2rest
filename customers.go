package magento2

import (
	"context"
	"strconv"
)

// Customer read routes.
const (
	routeCustomersSearch = "/V1/customers/search"       // GET module-customer/etc/webapi.xml:167
	routeCustomer        = "/V1/customers/{customerId}" // GET module-customer/etc/webapi.xml:119
)

// CustomerListResponse is a page of GET /V1/customers/search.
type CustomerListResponse struct {
	Items      []Customer `json:"items"`
	TotalCount int        `json:"total_count"`
}

// GetCustomersPage fetches one page of GET /V1/customers/search; sc may be
// nil.
func GetCustomersPage(ctx context.Context, c *Client, sc *SearchCriteria) (*CustomerListResponse, error) {
	out := &CustomerListResponse{}
	if err := getSearchPage(ctx, c, routeCustomersSearch, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateCustomers pages through GET /V1/customers/search.
func IterateCustomers(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Customer) error) error {
	return iterateList(ctx, c, routeCustomersSearch, sc, fn)
}

// GetCustomer fetches one customer (GET /V1/customers/{customerId}).
func GetCustomer(ctx context.Context, c *Client, id int) (*Customer, error) {
	out := &Customer{}
	if err := c.DoJSON(ctx, Request{Path: "/V1/customers/" + strconv.Itoa(id)}, out); err != nil {
		return nil, err
	}
	return out, nil
}
