package magento2

import (
	"context"
	"net/url"
	"reflect"
	"strconv"
)

// searchValues renders sc for a Magento getList endpoint. Those endpoints
// refuse a request without any searchCriteria key ("searchCriteria" is
// required), so an empty or nil sc sends "searchCriteria=" (no filter, the
// store's default page).
func searchValues(sc *SearchCriteria) (url.Values, error) {
	if sc == nil {
		return url.Values{"searchCriteria": {""}}, nil
	}
	v, err := sc.Values()
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		v.Set("searchCriteria", "")
	}
	return v, nil
}

// getSearchPage fetches one page of a getList endpoint into out.
func getSearchPage(ctx context.Context, c *Client, path string, sc *SearchCriteria, out any) error {
	q, err := searchValues(sc)
	if err != nil {
		return err
	}
	return c.DoJSON(ctx, Request{Path: path, Query: q}, out)
}

// iterateSearch drives page-by-page iteration over a getList endpoint with
// the same termination rules as IterateProducts: all total_count items seen,
// an empty page, or a page repeating the previous one. Pagination starts at
// sc's current page (page 1 when sc sets none) with sc's page size
// (DefaultPageSize when sc sets none); sc itself is not modified.
func iterateSearch[T any](ctx context.Context, sc *SearchCriteria, fetch func(url.Values) ([]T, int, error), fn func(T) error) error {
	base, err := searchValues(sc)
	if err != nil {
		return err
	}
	base.Del("searchCriteria")
	page, _ := strconv.Atoi(base.Get("searchCriteria[currentPage]"))
	page = max(page, 1)
	if base.Get("searchCriteria[pageSize]") == "" {
		base.Set("searchCriteria[pageSize]", strconv.Itoa(DefaultPageSize))
	}

	collected := 0
	var prevPage []T
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		q := cloneValues(base)
		q.Set("searchCriteria[currentPage]", strconv.Itoa(page))
		items, totalCount, err := fetch(q)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		if prevPage != nil && reflect.DeepEqual(items, prevPage) {
			return nil
		}
		for i := range items {
			if err := fn(items[i]); err != nil {
				return err
			}
		}
		collected += len(items)
		if collected >= totalCount {
			return nil
		}
		prevPage = items
		page++
	}
}

// listPage is the common shape of Magento search results.
type listPage[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"total_count"`
}

// iterateList is iterateSearch for an endpoint at path answering a
// listPage of T.
func iterateList[T any](ctx context.Context, c *Client, path string, sc *SearchCriteria, fn func(T) error) error {
	return iterateSearch(ctx, sc, func(q url.Values) ([]T, int, error) {
		var page listPage[T]
		if err := c.DoJSON(ctx, Request{Path: path, Query: q}, &page); err != nil {
			return nil, 0, err
		}
		return page.Items, page.TotalCount, nil
	}, fn)
}
