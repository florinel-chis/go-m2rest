package magento2

import (
	"context" // Added context
	"fmt"

	"github.com/rs/zerolog/log"
)

// Assuming these constants are defined elsewhere (e.g., categories_routes.go)
// For the subtask, define placeholders if needed.
var (
	categories                 = "/categories"
	categoriesList             = "/categories/list" // Used in GetCategoryByName
	categoriesProductsRelative = "products"         // Used in UpdateCategoryProductsFromRemote, AssignProductByProductLink
)

type MCategory struct {
	Route     string
	Category  *Category
	Products  *[]ProductLink
	APIClient *Client
}

// CreateCategory now accepts context.Context
func CreateCategory(ctx context.Context, c *Category, apiClient *Client) (*MCategory, error) {
	mC := &MCategory{
		Category:  &Category{},
		Products:  &[]ProductLink{},
		APIClient: apiClient,
	}
	endpoint := categories
	// httpClient := apiClient.HTTPClient // No longer needed directly for POST if using PostRouteAndDecode

	payLoad := createCategoryPayload{ // Assuming createCategoryPayload is defined
		Category: *c,
	}

	log.Debug().Interface("payload", payLoad).Str("endpoint", endpoint).Msg("Creating category with context")

	// Use APIClient's PostRouteAndDecode
	err := apiClient.PostRouteAndDecode(ctx, endpoint, payLoad, mC.Category, "create category")
	if err != nil {
		// Error is already wrapped by PostRouteAndDecode
		return mC, err // Return mC to allow inspection even on partial failure if needed, or nil
	}
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)
	return mC, nil
}

// GetCategoryByName now accepts context.Context
func GetCategoryByName(ctx context.Context, name string, apiClient *Client) (*MCategory, error) {
	mC := &MCategory{
		Category:  &Category{},
		Products:  &[]ProductLink{},
		APIClient: apiClient,
	}
	searchQuery := BuildSearchQuery("name", name, "in") // Assuming BuildSearchQuery is available
	endpoint := categoriesList + "?" + searchQuery
	// httpClient := apiClient.HTTPClient // No longer needed directly

	response := &categorySearchQueryResponse{} // Assuming categorySearchQueryResponse is defined

	log.Debug().Str("name", name).Str("endpoint", endpoint).Msg("Getting category by name with context")

	// Use APIClient's GetRouteAndDecode
	err := apiClient.GetRouteAndDecode(ctx, endpoint, response, "get category by name from remote")
	if err != nil {
		return nil, err
	}

	if len(response.Categories) == 0 {
		log.Warn().Str("name", name).Msg("Category not found by name")
		return nil, ErrNotFound
	}

	mC.Category = &response.Categories[0]
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)

	err = mC.UpdateCategoryFromRemote(ctx) // Pass context
	if err != nil {
		// Note: original code returned mC here, which might be partially populated.
		// Consider returning nil for mC if UpdateCategoryFromRemote fails critically.
		return mC, fmt.Errorf("error updating category from remote after getting by name: %w", err)
	}
	// httpErr := mayReturnErrorForHTTPResponse(resp, "get detailed category by name from remote")
	// This check would have been part of GetRouteAndDecode now.

	return mC, nil
}

// UpdateCategoryFromRemote now accepts context.Context
func (mC *MCategory) UpdateCategoryFromRemote(ctx context.Context) error {
	log.Debug().Str("route", mC.Route).Msg("Updating category details from remote with context")

	err := mC.APIClient.GetRouteAndDecode(ctx, mC.Route, mC.Category, "get category from remote")
	if err != nil {
		return err
	}

	err = mC.UpdateCategoryProductsFromRemote(ctx) // Pass context
	if err != nil {
		return fmt.Errorf("error updating category products from remote after updating category details: %w", err)
	}
	return nil
}

// UpdateCategoryProductsFromRemote now accepts context.Context
func (mC *MCategory) UpdateCategoryProductsFromRemote(ctx context.Context) error {
	productsRoute := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)
	log.Debug().Str("route", productsRoute).Msg("Updating category products from remote with context")

	err := mC.APIClient.GetRouteAndDecode(ctx, productsRoute, mC.Products, "get category products from remote")
	if err != nil {
		return err
	}
	return nil
}

// AssignProductByProductLink now accepts context.Context
func (mC *MCategory) AssignProductByProductLink(ctx context.Context, pl *ProductLink) error {
	if pl.CategoryID == "" {
		pl.CategoryID = fmt.Sprintf("%d", mC.Category.ID)
	}

	// httpClient := mC.APIClient.HTTPClient // Not needed directly for PUT if PutRouteAndDecode exists
	endpoint := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)

	payLoad := assignProductPayload{ProductLink: *pl} // Assuming assignProductPayload defined

	log.Debug().Str("sku", pl.Sku).Int("categoryID", mC.Category.ID).Str("endpoint", endpoint).Interface("payload", payLoad).Msg("Assigning product to category with context")

	// This is a PUT request. Using direct client as PutRouteAndDecode wasn't in api_client.go scope.
	resp, err := mC.APIClient.HTTPClient.R().SetContext(ctx).SetBody(payLoad).Put(endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Error().Err(err).Msg("Error assigning product to category")
		return fmt.Errorf("error assigning product to category: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "assign product to category")
	if httpErr != nil {
		return httpErr
	}

	*mC.Products = append(*mC.Products, *pl) // Optimistic update
	return nil
}
