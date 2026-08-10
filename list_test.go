package magento2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
)

func updatedAtCriteria(value string) []SearchQueryCriteria {
	return []SearchQueryCriteria{
		{
			Fields: []FilterFields{
				{
					Field:         Filter{FilterGroups: 0, Filters: 0, FilterFor: "updated_at"},
					Value:         Filter{FilterGroups: 0, Filters: 0, FilterFor: value},
					ConditionType: Filter{FilterGroups: 0, Filters: 0, FilterFor: "gt"},
				},
			},
		},
	}
}

func TestListOptionsEncode(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		query, err := url.ParseQuery(ListOptions{}.Encode())
		if err != nil {
			t.Fatalf("failed to parse encoded query: %v", err)
		}
		if got := query.Get("searchCriteria[pageSize]"); got != "100" {
			t.Errorf("pageSize = %q, want 100 (PageSize<=0 default)", got)
		}
		if query.Has("searchCriteria[currentPage]") {
			t.Error("currentPage should not be encoded when unset")
		}
		if query.Has("searchCriteria[sortOrders][0][field]") {
			t.Error("sortOrders should not be encoded when unset")
		}
	})

	t.Run("page size, current page and sort orders", func(t *testing.T) {
		opts := ListOptions{PageSize: 25, CurrentPage: 3, SortField: "updated_at", SortDir: "ASC"}
		query, err := url.ParseQuery(opts.Encode())
		if err != nil {
			t.Fatalf("failed to parse encoded query: %v", err)
		}
		if got := query.Get("searchCriteria[pageSize]"); got != "25" {
			t.Errorf("pageSize = %q, want 25", got)
		}
		if got := query.Get("searchCriteria[currentPage]"); got != "3" {
			t.Errorf("currentPage = %q, want 3", got)
		}
		if got := query.Get("searchCriteria[sortOrders][0][field]"); got != "updated_at" {
			t.Errorf("sort field = %q, want updated_at", got)
		}
		if got := query.Get("searchCriteria[sortOrders][0][direction]"); got != "ASC" {
			t.Errorf("sort direction = %q, want ASC", got)
		}
	})

	t.Run("updated_at gt filter group", func(t *testing.T) {
		opts := ListOptions{Criteria: updatedAtCriteria("2026-01-01 00:00:00")}
		query, err := url.ParseQuery(opts.Encode())
		if err != nil {
			t.Fatalf("failed to parse encoded query: %v", err)
		}
		if got := query.Get("searchCriteria[filter_groups][0][filters][0][field]"); got != "updated_at" {
			t.Errorf("filter field = %q, want updated_at", got)
		}
		if got := query.Get("searchCriteria[filter_groups][0][filters][0][value]"); got != "2026-01-01 00:00:00" {
			t.Errorf("filter value = %q, want the updated_at threshold", got)
		}
		if got := query.Get("searchCriteria[filter_groups][0][filters][0][condition_type]"); got != "gt" {
			t.Errorf("filter condition_type = %q, want gt", got)
		}
	})

	t.Run("extra fields", func(t *testing.T) {
		opts := ListOptions{Extra: []Fields{{Key: "fields", Value: "items[sku]"}}}
		query, err := url.ParseQuery(opts.Encode())
		if err != nil {
			t.Fatalf("failed to parse encoded query: %v", err)
		}
		if got := query.Get("fields"); got != "items[sku]" {
			t.Errorf("extra field = %q, want items[sku]", got)
		}
	})
}

func TestIterateProductsTotalCountTermination(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) > 6 {
			http.Error(w, "pagination failed to terminate", http.StatusBadRequest)
			return
		}
		query := r.URL.Query()

		// The updated_at filter must be present on every page request.
		if got := query.Get("searchCriteria[filter_groups][0][filters][0][field]"); got != "updated_at" {
			t.Errorf("filter field = %q, want updated_at", got)
		}
		if got := query.Get("searchCriteria[filter_groups][0][filters][0][condition_type]"); got != "gt" {
			t.Errorf("filter condition_type = %q, want gt", got)
		}
		if got := query.Get("searchCriteria[pageSize]"); got != "2" {
			t.Errorf("pageSize = %q, want 2", got)
		}

		page, _ := strconv.Atoi(query.Get("searchCriteria[currentPage]"))
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			_, _ = w.Write([]byte(`{"items":[{"sku":"p1","name":"P1"},{"sku":"p2","name":"P2"}],"total_count":3}`))
		case 2:
			_, _ = w.Write([]byte(`{"items":[{"sku":"p3","name":"P3"}],"total_count":3}`))
		default:
			t.Errorf("unexpected page request: %d", page)
			_, _ = w.Write([]byte(`{"items":[],"total_count":3}`))
		}
	})

	client := newTestClient(t, handler)
	opts := ListOptions{PageSize: 2, Criteria: updatedAtCriteria("2026-01-01 00:00:00")}

	var skus []string
	err := IterateProducts(context.Background(), client, opts, func(p Product) error {
		skus = append(skus, p.Sku)
		return nil
	})
	if err != nil {
		t.Fatalf("IterateProducts returned error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("requests = %d, want 2 (terminate on total_count)", got)
	}
	want := []string{"p1", "p2", "p3"}
	if len(skus) != len(want) {
		t.Fatalf("collected skus = %v, want %v", skus, want)
	}
	for i := range want {
		if skus[i] != want[i] {
			t.Errorf("skus[%d] = %q, want %q", i, skus[i], want[i])
		}
	}
}

func TestIterateProductsEmptyPageTermination(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) > 6 {
			// Circuit breaker: fail fast (non-retryable status) instead of
			// hanging for the harness timeout if termination regresses.
			http.Error(w, "pagination failed to terminate", http.StatusBadRequest)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("searchCriteria[currentPage]"))
		w.Header().Set("Content-Type", "application/json")
		if page == 1 {
			// total_count claims far more items than the server will return.
			_, _ = w.Write([]byte(`{"items":[{"sku":"p1"},{"sku":"p2"}],"total_count":10}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[],"total_count":10}`))
	})

	client := newTestClient(t, handler)
	count := 0
	err := IterateProducts(context.Background(), client, ListOptions{PageSize: 2}, func(p Product) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("IterateProducts returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("collected %d products, want 2", count)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("requests = %d, want 2 (empty page terminates, no infinite loop)", got)
	}
}

// Real Magento clamps an out-of-range currentPage to the last page and
// re-serves the last page's items (it never returns an empty page past the
// end). When total_count overstates the reachable items, the repeated page
// must be detected and not delivered twice.
func TestIterateProductsClampedLastPageNotDuplicated(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) > 6 {
			http.Error(w, "pagination failed to terminate", http.StatusBadRequest)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("searchCriteria[currentPage]"))
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			// total_count claims 6 but only 4 items are reachable.
			_, _ = w.Write([]byte(`{"items":[{"sku":"p1"},{"sku":"p2"}],"total_count":6}`))
		default:
			// Pages 2, 3, ... all serve the (clamped) last page.
			_, _ = w.Write([]byte(`{"items":[{"sku":"p3"},{"sku":"p4"}],"total_count":6}`))
		}
	})

	client := newTestClient(t, handler)
	var skus []string
	err := IterateProducts(context.Background(), client, ListOptions{PageSize: 2}, func(p Product) error {
		skus = append(skus, p.Sku)
		return nil
	})
	if err != nil {
		t.Fatalf("IterateProducts returned error: %v", err)
	}
	want := []string{"p1", "p2", "p3", "p4"}
	if len(skus) != len(want) {
		t.Fatalf("collected skus = %v, want %v (no duplicates from the clamped last page)", skus, want)
	}
	for i := range want {
		if skus[i] != want[i] {
			t.Errorf("skus[%d] = %q, want %q", i, skus[i], want[i])
		}
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Errorf("requests = %d, want 3 (repeated page detected on the third request)", got)
	}
}

func TestIterateProductsCallbackErrorAborts(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"sku":"p1"},{"sku":"p2"}],"total_count":4}`))
	})

	client := newTestClient(t, handler)
	sentinel := errors.New("stop here")
	calls := 0
	err := IterateProducts(context.Background(), client, ListOptions{PageSize: 2}, func(p Product) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("IterateProducts error = %v, want the callback error", err)
	}
	if calls != 1 {
		t.Errorf("callback calls = %d, want 1", calls)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("requests = %d, want 1 (abort on callback error)", got)
	}
}

func TestGetAttributesPageAndIterateAttributes(t *testing.T) {
	attributesJSON := `{
		"items": [
			{
				"attribute_id": 93,
				"attribute_code": "color",
				"frontend_input": "select",
				"default_frontend_label": "Color",
				"is_filterable": 2,
				"is_filterable_in_search": "1",
				"is_searchable": "1"
			},
			{
				"attribute_id": 94,
				"attribute_code": "size",
				"frontend_input": "select",
				"default_frontend_label": "Size",
				"is_filterable": 0,
				"is_filterable_in_search": false
			}
		],
		"total_count": 2
	}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/default/V1/products/attributes" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(attributesJSON))
	})

	client := newTestClient(t, handler)
	page, err := GetAttributesPage(context.Background(), client, ListOptions{})
	if err != nil {
		t.Fatalf("GetAttributesPage returned error: %v", err)
	}
	if page.TotalCount != 2 || len(page.Items) != 2 {
		t.Fatalf("unexpected page: total=%d items=%d", page.TotalCount, len(page.Items))
	}
	color := page.Items[0]
	if color.AttributeCode != "color" || color.AttributeID != 93 {
		t.Errorf("unexpected first attribute: %+v", color)
	}
	if !color.IsFilterable.Bool() {
		t.Error("is_filterable: 2 should decode to true")
	}
	if !color.IsFilterableInSearch.Bool() {
		t.Error(`is_filterable_in_search: "1" should decode to true`)
	}
	if color.IsSearchable != "1" {
		t.Errorf("is_searchable should stay a string, got %q", color.IsSearchable)
	}
	if page.Items[1].IsFilterable.Bool() || page.Items[1].IsFilterableInSearch.Bool() {
		t.Errorf("unexpected filterable flags on second attribute: %+v", page.Items[1])
	}

	var codes []string
	err = IterateAttributes(context.Background(), client, ListOptions{}, func(a Attribute) error {
		codes = append(codes, a.AttributeCode)
		return nil
	})
	if err != nil {
		t.Fatalf("IterateAttributes returned error: %v", err)
	}
	if len(codes) != 2 || codes[0] != "color" || codes[1] != "size" {
		t.Errorf("iterated codes = %v, want [color size]", codes)
	}
}

func TestGetAttributeSetsListPaginatesInternally(t *testing.T) {
	var requests int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) > 6 {
			http.Error(w, "pagination failed to terminate", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/rest/default/V1/products/attribute-sets/sets/list" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("searchCriteria[currentPage]"))
		w.Header().Set("Content-Type", "application/json")
		if page == 1 {
			_, _ = w.Write([]byte(`{"items":[{"attribute_set_id":4,"attribute_set_name":"Default"},{"attribute_set_id":9,"attribute_set_name":"Shoes"}],"total_count":3}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"attribute_set_id":10,"attribute_set_name":"Bags"}],"total_count":3}`))
	})

	client := newTestClient(t, handler)
	sets, err := GetAttributeSetsList(context.Background(), client)
	if err != nil {
		t.Fatalf("GetAttributeSetsList returned error: %v", err)
	}
	if len(sets) != 3 {
		t.Fatalf("got %d attribute sets, want 3", len(sets))
	}
	if sets[0].AttributeSetID != 4 || sets[0].AttributeSetName != "Default" {
		t.Errorf("unexpected first set: %+v", sets[0])
	}
	if sets[2].AttributeSetID != 10 || sets[2].AttributeSetName != "Bags" {
		t.Errorf("unexpected last set: %+v", sets[2])
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("requests = %d, want 2 (internal pagination)", got)
	}
}

func TestGetAttributeSetAttributes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/default/V1/products/attribute-sets/4/attributes" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"attribute_id":73,"attribute_code":"name","frontend_input":"text","default_frontend_label":"Name"},
			{"attribute_id":77,"attribute_code":"price","frontend_input":"price","default_frontend_label":"Price"}
		]`))
	})

	client := newTestClient(t, handler)
	attributes, err := GetAttributeSetAttributes(context.Background(), client, 4)
	if err != nil {
		t.Fatalf("GetAttributeSetAttributes returned error: %v", err)
	}
	if len(attributes) != 2 {
		t.Fatalf("got %d attributes, want 2", len(attributes))
	}
	if attributes[0].AttributeCode != "name" || attributes[1].AttributeCode != "price" {
		t.Errorf("unexpected attribute codes: %+v", attributes)
	}
}

func TestGetCategoryTree(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/default/V1/categories" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 2, "parent_id": 1, "name": "Default Category", "is_active": true,
			"level": 1, "product_count": 42,
			"children_data": [
				{"id": 3, "parent_id": 2, "name": "Men", "is_active": true, "level": 2, "product_count": 10,
				 "children_data": [
					{"id": 5, "parent_id": 3, "name": "Tops", "is_active": false, "level": 3, "product_count": 4, "children_data": []}
				 ]},
				{"id": 4, "parent_id": 2, "name": "Women", "is_active": true, "level": 2, "product_count": 12, "children_data": []}
			]
		}`))
	})

	client := newTestClient(t, handler)
	tree, err := GetCategoryTree(context.Background(), client)
	if err != nil {
		t.Fatalf("GetCategoryTree returned error: %v", err)
	}
	if tree.ID != 2 || tree.Name != "Default Category" || tree.ProductCount != 42 {
		t.Errorf("unexpected root node: %+v", tree)
	}
	if len(tree.Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(tree.Children))
	}
	men := tree.Children[0]
	if men.ID != 3 || men.Name != "Men" || men.ParentID != 2 || men.Level != 2 {
		t.Errorf("unexpected child node: %+v", men)
	}
	if len(men.Children) != 1 || men.Children[0].Name != "Tops" || men.Children[0].IsActive {
		t.Errorf("unexpected grandchild: %+v", men.Children)
	}
}

func TestGetStoreViewsAndWebsites(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/default/V1/store/storeViews":
			_, _ = w.Write([]byte(`[
				{"id":0,"code":"admin","name":"Admin","website_id":0,"store_group_id":0},
				{"id":1,"code":"default","name":"Default Store View","website_id":1,"store_group_id":1}
			]`))
		case "/rest/default/V1/store/websites":
			_, _ = w.Write([]byte(`[
				{"id":0,"code":"admin","name":"Admin","default_group_id":0},
				{"id":1,"code":"base","name":"Main Website","default_group_id":1}
			]`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	})

	client := newTestClient(t, handler)

	storeViews, err := GetStoreViews(context.Background(), client)
	if err != nil {
		t.Fatalf("GetStoreViews returned error: %v", err)
	}
	if len(storeViews) != 2 {
		t.Fatalf("got %d store views, want 2", len(storeViews))
	}
	if storeViews[1].Code != "default" || storeViews[1].WebsiteID != 1 || storeViews[1].StoreGroupID != 1 {
		t.Errorf("unexpected store view: %+v", storeViews[1])
	}

	websites, err := GetWebsites(context.Background(), client)
	if err != nil {
		t.Fatalf("GetWebsites returned error: %v", err)
	}
	if len(websites) != 2 {
		t.Fatalf("got %d websites, want 2", len(websites))
	}
	if websites[1].Code != "base" || websites[1].Name != "Main Website" {
		t.Errorf("unexpected website: %+v", websites[1])
	}
}

// Ensure the response types themselves round-trip cleanly.
func TestProductListResponseDecoding(t *testing.T) {
	raw := `{"items":[{"sku":"a","name":"A","price":9.99}],"total_count":7}`
	var page ProductListResponse
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if page.TotalCount != 7 || len(page.Items) != 1 || page.Items[0].Sku != "a" {
		t.Errorf("unexpected decode result: %+v", page)
	}
	if fmt.Sprintf("%.2f", page.Items[0].Price) != "9.99" {
		t.Errorf("price = %v, want 9.99", page.Items[0].Price)
	}
}
