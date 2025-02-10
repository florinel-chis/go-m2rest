package magento2

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type MCategory struct {
	Route     string
	Category  *Category
	Products  *[]ProductLink
	APIClient *Client
}

func CreateCategory(c *Category, apiClient *Client) (*MCategory, error) {
	mC := &MCategory{
		Category:  &Category{},
		Products:  &[]ProductLink{},
		APIClient: apiClient,
	}
	endpoint := categories
	httpClient := apiClient.HTTPClient

	payLoad := createCategoryPayload{
		Category: *c,
	}

	log.Debug().
		Interface("payload", payLoad).
		Str("endpoint", endpoint).
		Msg("Creating category")

	resp, err := httpClient.R().SetBody(payLoad).SetResult(mC.Category).Post(endpoint)
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)

	if err != nil {
		return mC, fmt.Errorf("error creating category: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "create category")
	if httpErr != nil {
		return mC, httpErr
	}

	return mC, nil
}

func GetCategoryByName(name string, apiClient *Client) (*MCategory, error) {
	mC := &MCategory{
		Category:  &Category{},
		Products:  &[]ProductLink{},
		APIClient: apiClient,
	}
	searchQuery := BuildSearchQuery("name", name, "in")
	endpoint := categoriesList + "?" + searchQuery
	httpClient := apiClient.HTTPClient

	response := &categorySearchQueryResponse{}

	log.Debug().
		Str("name", name).
		Str("endpoint", endpoint).
		Msg("Getting category by name")

	resp, err := httpClient.R().SetResult(response).Get(endpoint)

	if err != nil {
		return nil, fmt.Errorf("error getting category by name: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get category by name from remote")
	if httpErr != nil {
		return nil, httpErr
	}

	if len(response.Categories) == 0 {
		log.Warn().Str("name", name).Msg("Category not found by name")
		return nil, ErrNotFound
	}

	mC.Category = &response.Categories[0]
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)

	err = mC.UpdateCategoryFromRemote()
	if err != nil {
		return mC, fmt.Errorf("error updating category from remote after getting by name: %w", err)
	}

	httpErr = mayReturnErrorForHTTPResponse(resp, "get detailed category by name from remote")
	if httpErr != nil {
		return mC, httpErr
	}

	return mC, nil
}

func (mC *MCategory) UpdateCategoryFromRemote() error {
	log.Debug().
		Str("route", mC.Route).
		Msg("Updating category details from remote")

	resp, err := mC.APIClient.HTTPClient.R().SetResult(mC.Category).Get(mC.Route)

	if err != nil {
		log.Error().Err(err).Msg("Error updating category details from remote")
		return fmt.Errorf("error getting category from remote: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get category from remote")
	if httpErr != nil {
		return httpErr
	}

	err = mC.UpdateCategoryProductsFromRemote()
	if err != nil {
		return fmt.Errorf("error updating category products from remote after updating category details: %w", err)
	}
	return nil
}

func (mC *MCategory) UpdateCategoryProductsFromRemote() error {
	productsRoute := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)
	log.Debug().
		Str("route", productsRoute).
		Msg("Updating category products from remote")

	resp, err := mC.APIClient.HTTPClient.R().SetResult(mC.Products).Get(productsRoute)

	if err != nil {
		log.Error().Err(err).Msg("Error updating category products from remote")
		return fmt.Errorf("error getting category products from remote: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get category products from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mC *MCategory) AssignProductByProductLink(pl *ProductLink) error {
	if pl.CategoryID == "" {
		pl.CategoryID = fmt.Sprintf("%d", mC.Category.ID)
	}

	httpClient := mC.APIClient.HTTPClient
	endpoint := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)

	payLoad := assignProductPayload{ProductLink: *pl}

	log.Debug().
		Str("sku", pl.Sku).
		Int("categoryID", mC.Category.ID).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Assigning product to category")

	resp, err := httpClient.R().SetBody(payLoad).Put(endpoint)

	if err != nil {
		log.Error().Err(err).Msg("Error assigning product to category")
		return fmt.Errorf("error assigning product to category: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "assign product to category")
	if httpErr != nil {
		return httpErr
	}

	*mC.Products = append(*mC.Products, *pl)

	return nil
}
