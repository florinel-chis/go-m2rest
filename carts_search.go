package magento2

import "context"

// routeCartsSearch is GET module-quote/etc/webapi.xml:19.
const routeCartsSearch = "/V1/carts/search"

// CartListResponse is a page of GET /V1/carts/search.
type CartListResponse struct {
	Items      []Cart `json:"items"`
	TotalCount int    `json:"total_count"`
}

// GetCartsPage fetches one page of GET /V1/carts/search; sc may be nil.
//
// /V1/carts/search reads quotes through a collection and is a plain read.
// Every per-cart GET is not: /V1/carts/{cartId}..., /V1/carts/mine... and
// /V1/guest-carts/{cartId}... load the quote model, and
// Magento\Quote\Model\Quote::_afterLoad (module-quote/Model/Quote.php,
// Magento 2.4.8 line 2486) collects totals and SAVES a quote whose
// trigger_recollect flag is set (for example after a catalog price rule or
// product change). A "read" of a single cart can therefore write to the
// store; embedders that must not write should use this search instead.
func GetCartsPage(ctx context.Context, c *Client, sc *SearchCriteria) (*CartListResponse, error) {
	out := &CartListResponse{}
	if err := getSearchPage(ctx, c, routeCartsSearch, sc, out); err != nil {
		return nil, err
	}
	return out, nil
}

// IterateCarts pages through GET /V1/carts/search (see GetCartsPage).
func IterateCarts(ctx context.Context, c *Client, sc *SearchCriteria, fn func(Cart) error) error {
	return iterateList(ctx, c, routeCartsSearch, sc, fn)
}
