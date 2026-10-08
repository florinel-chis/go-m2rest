package magento2

import (
	"context"
	"fmt"
	"net/http"
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

	payLoad := createCategoryPayload{
		Category: *c,
	}

	_, err := apiClient.v1(context.Background(), http.MethodPost, categories, payLoad, mC.Category, "create category")
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)
	if err != nil {
		return mC, err
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

	response := &categorySearchQueryResponse{}
	if _, err := apiClient.v1(context.Background(), http.MethodGet, endpoint, nil, response, "get category by name from remote"); err != nil {
		return nil, err
	}

	if len(response.Categories) == 0 {
		return nil, ErrNotFound
	}

	mC.Category = &response.Categories[0]
	mC.Route = fmt.Sprintf("%s/%d", categories, mC.Category.ID)

	err := mC.UpdateCategoryFromRemote()
	if err != nil {
		return mC, fmt.Errorf("error updating category from remote after getting by name: %w", err)
	}

	return mC, nil
}

func (mC *MCategory) UpdateCategoryFromRemote() error {
	if _, err := mC.APIClient.v1(context.Background(), http.MethodGet, mC.Route, nil, mC.Category, "get category from remote"); err != nil {
		return err
	}

	err := mC.UpdateCategoryProductsFromRemote()
	if err != nil {
		return fmt.Errorf("error updating category products from remote after updating category details: %w", err)
	}
	return nil
}

func (mC *MCategory) UpdateCategoryProductsFromRemote() error {
	productsRoute := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)
	_, err := mC.APIClient.v1(context.Background(), http.MethodGet, productsRoute, nil, mC.Products, "get category products from remote")
	return err
}

func (mC *MCategory) AssignProductByProductLink(pl *ProductLink) error {
	if pl.CategoryID == "" {
		pl.CategoryID = fmt.Sprintf("%d", mC.Category.ID)
	}

	endpoint := fmt.Sprintf("%s/%s", mC.Route, categoriesProductsRelative)

	payLoad := assignProductPayload{ProductLink: *pl}

	if _, err := mC.APIClient.v1(context.Background(), http.MethodPut, endpoint, payLoad, nil, "assign product to category"); err != nil {
		return err
	}

	*mC.Products = append(*mC.Products, *pl)

	return nil
}
