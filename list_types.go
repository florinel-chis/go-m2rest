package magento2

import (
	"net/url"
	"strconv"
	"strings"
)

// DefaultPageSize is the page size used when ListOptions.PageSize is not set.
const DefaultPageSize = 100

const (
	searchCriteriaPageSizeKey    = "searchCriteria[pageSize]"
	searchCriteriaCurrentPageKey = "searchCriteria[currentPage]"
	searchCriteriaSortFieldKey   = "searchCriteria[sortOrders][0][field]"
	searchCriteriaSortDirKey     = "searchCriteria[sortOrders][0][direction]"
)

// ListOptions describes a searchCriteria query for Magento list endpoints.
type ListOptions struct {
	// PageSize is the number of items per page; values <= 0 encode as
	// DefaultPageSize.
	PageSize int
	// CurrentPage is the 1-based page to request; it is only encoded when
	// > 0.
	CurrentPage int
	// SortField/SortDir encode searchCriteria[sortOrders][0]; SortDir is
	// typically "ASC" or "DESC".
	SortField string
	SortDir   string
	// Criteria are filter groups, encoded with the same group/filter
	// indexing as BuildFlexibleSearchQuery.
	Criteria []SearchQueryCriteria
	// Extra are additional raw query parameters (e.g. fields=items[sku]).
	Extra []Fields
}

// Encode renders the options as a URL query string (without leading "?").
func (o ListOptions) Encode() string {
	params := url.Values{}

	for i := range o.Criteria {
		for y := range o.Criteria[i].Fields {
			filterFields := o.Criteria[i].Fields[y]
			params.Add(
				strings.TrimSuffix(searchCriteriaField(filterFields.Field.FilterGroups, filterFields.Field.Filters), "="),
				filterFields.Field.FilterFor,
			)
			params.Add(
				strings.TrimSuffix(searchCriteriaValue(filterFields.Value.FilterGroups, filterFields.Value.Filters), "="),
				filterFields.Value.FilterFor,
			)
			params.Add(
				strings.TrimSuffix(searchCriteriaConditionType(filterFields.ConditionType.FilterGroups, filterFields.ConditionType.Filters), "="),
				filterFields.ConditionType.FilterFor,
			)
		}
	}

	for i := range o.Extra {
		params.Add(o.Extra[i].Key, o.Extra[i].Value)
	}

	pageSize := o.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	params.Set(searchCriteriaPageSizeKey, strconv.Itoa(pageSize))

	if o.CurrentPage > 0 {
		params.Set(searchCriteriaCurrentPageKey, strconv.Itoa(o.CurrentPage))
	}

	if o.SortField != "" {
		params.Set(searchCriteriaSortFieldKey, o.SortField)
		if o.SortDir != "" {
			params.Set(searchCriteriaSortDirKey, o.SortDir)
		}
	}

	return params.Encode()
}

// ProductListResponse is a single page of GET /products.
type ProductListResponse struct {
	Items      []Product `json:"items"`
	TotalCount int       `json:"total_count"`
}

// AttributeListResponse is a single page of GET /products/attributes.
type AttributeListResponse struct {
	Items      []Attribute `json:"items"`
	TotalCount int         `json:"total_count"`
}

type attributeSetListResponse struct {
	Items      []AttributeSet `json:"items"`
	TotalCount int            `json:"total_count"`
}

// CategoryTreeNode is a node of the category tree returned by
// GET /categories.
type CategoryTreeNode struct {
	ID           int                `json:"id"`
	ParentID     int                `json:"parent_id"`
	Name         string             `json:"name"`
	IsActive     bool               `json:"is_active"`
	Level        int                `json:"level"`
	ProductCount int                `json:"product_count"`
	Children     []CategoryTreeNode `json:"children_data"`
}
