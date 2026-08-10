package magento2

import (
	"context"
	"fmt"
	"reflect"
)

// GetProductsPage fetches a single page of products (GET /products) using
// the given search criteria.
func GetProductsPage(ctx context.Context, c *Client, opts ListOptions) (*ProductListResponse, error) {
	result := &ProductListResponse{}
	route := products + "?" + opts.Encode()
	if err := c.GetRouteAndDecodeCtx(ctx, route, result, "get products page"); err != nil {
		return nil, err
	}
	return result, nil
}

// IterateProducts pages through GET /products, calling fn for every product.
// Pagination starts at opts.CurrentPage (or page 1 when unset) and stops when
// all total_count items were seen, an empty page is returned, or a page
// repeats the previous one (Magento clamps out-of-range pages to the last
// page, so a repeated page signals total_count drift while iterating). If fn
// returns an error the iteration is aborted and that error is returned.
func IterateProducts(ctx context.Context, c *Client, opts ListOptions, fn func(Product) error) error {
	return iteratePages(ctx, opts, func(pageOpts ListOptions) ([]Product, int, error) {
		page, err := GetProductsPage(ctx, c, pageOpts)
		if err != nil {
			return nil, 0, err
		}
		return page.Items, page.TotalCount, nil
	}, fn)
}

// GetAttributesPage fetches a single page of product attributes
// (GET /products/attributes) using the given search criteria.
func GetAttributesPage(ctx context.Context, c *Client, opts ListOptions) (*AttributeListResponse, error) {
	result := &AttributeListResponse{}
	route := productsAttribute + "?" + opts.Encode()
	if err := c.GetRouteAndDecodeCtx(ctx, route, result, "get attributes page"); err != nil {
		return nil, err
	}
	return result, nil
}

// IterateAttributes pages through GET /products/attributes, calling fn for
// every attribute. See IterateProducts for the pagination semantics.
func IterateAttributes(ctx context.Context, c *Client, opts ListOptions, fn func(Attribute) error) error {
	return iteratePages(ctx, opts, func(pageOpts ListOptions) ([]Attribute, int, error) {
		page, err := GetAttributesPage(ctx, c, pageOpts)
		if err != nil {
			return nil, 0, err
		}
		return page.Items, page.TotalCount, nil
	}, fn)
}

// GetAttributeSetsList returns all attribute sets
// (GET /products/attribute-sets/sets/list), paginating internally.
func GetAttributeSetsList(ctx context.Context, c *Client) ([]AttributeSet, error) {
	var sets []AttributeSet
	err := iteratePages(ctx, ListOptions{}, func(pageOpts ListOptions) ([]AttributeSet, int, error) {
		result := &attributeSetListResponse{}
		route := productsAttributeSetList + "?" + pageOpts.Encode()
		if err := c.GetRouteAndDecodeCtx(ctx, route, result, "get attribute-sets page"); err != nil {
			return nil, 0, err
		}
		return result.Items, result.TotalCount, nil
	}, func(set AttributeSet) error {
		sets = append(sets, set)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sets, nil
}

// GetAttributeSetAttributes returns the attributes assigned to an attribute
// set (GET /products/attribute-sets/{id}/attributes).
func GetAttributeSetAttributes(ctx context.Context, c *Client, attributeSetID int) ([]Attribute, error) {
	var attributes []Attribute
	route := fmt.Sprintf("%s/%d/%s", productsAttributeSet, attributeSetID, productsAttributeSetAttributesRelative)
	if err := c.GetRouteAndDecodeCtx(ctx, route, &attributes, "get attributes for attribute-set"); err != nil {
		return nil, err
	}
	return attributes, nil
}

// GetCategoryTree returns the full category tree (GET /categories).
func GetCategoryTree(ctx context.Context, c *Client) (*CategoryTreeNode, error) {
	tree := &CategoryTreeNode{}
	if err := c.GetRouteAndDecodeCtx(ctx, categories, tree, "get category tree"); err != nil {
		return nil, err
	}
	return tree, nil
}

// iteratePages drives page-by-page iteration for list endpoints. It stops
// when the collected item count reaches the reported total_count, when a
// page comes back empty, or when a page repeats the previous one verbatim.
// The last two guard against total_count drifting while iterating (and
// against infinite loops): Magento's collection layer clamps an
// out-of-range currentPage to the last page and re-serves the final page's
// items instead of returning an empty page, so a repeated page means the
// end was reached and must not be delivered twice.
func iteratePages[T any](ctx context.Context, opts ListOptions, fetch func(ListOptions) ([]T, int, error), fn func(T) error) error {
	if opts.CurrentPage <= 0 {
		opts.CurrentPage = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = DefaultPageSize
	}

	collected := 0
	var prevPage []T
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		items, totalCount, err := fetch(opts)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		if prevPage != nil && reflect.DeepEqual(items, prevPage) {
			// Magento clamped the page number to the last page: the end
			// was reached even though total_count claims more items.
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
		opts.CurrentPage++
	}
}
