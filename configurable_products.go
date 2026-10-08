package magento2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type MConfigurableProduct struct {
	Route     string
	Options   *[]ConfigurableProductOption
	APIClient *Client
}

func SetOptionForExistingConfigurableProduct(sku string, o *ConfigurableProductOption, apiClient *Client) (*MConfigurableProduct, error) {
	mConfigurableProduct := &MConfigurableProduct{
		Route:     configurableProducts + "/" + url.PathEscape(sku),
		Options:   &[]ConfigurableProductOption{},
		APIClient: apiClient,
	}
	endpoint := mConfigurableProduct.Route + "/" + configurableProductsOptionsRelative

	payLoad := createConfigurableProductByOptionPayload{
		Option: *o,
	}

	if _, err := apiClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, nil, "create configurable product option"); err != nil {
		return mConfigurableProduct, err
	}

	err := mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return mConfigurableProduct, fmt.Errorf("error updating options from remote after setting option: %w", err)
	}

	return mConfigurableProduct, nil
}

func (mConfigurableProduct *MConfigurableProduct) UpdateOptionsFromRemote() error {
	optionsRoute := mConfigurableProduct.Route + "/" + configurableProductsOptionsAllRelative
	_, err := mConfigurableProduct.APIClient.v1(context.Background(), http.MethodGet, optionsRoute, nil, mConfigurableProduct.Options, "get options for configurable product from remote")
	return err
}

func (mConfigurableProduct *MConfigurableProduct) AddChildBySKU(sku string) error {
	payLoad := addChildSKUPayload{
		Sku: sku,
	}

	endpoint := fmt.Sprintf("%s/%s", mConfigurableProduct.Route, configurableProductsChildRelative)
	_, err := mConfigurableProduct.APIClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, nil, "add child by sku to configurable product")
	return err
}

func GetConfigurableProductBySKU(sku string, apiClient *Client) (*MConfigurableProduct, error) {
	mConfigurableProduct := &MConfigurableProduct{
		Route:     configurableProducts + "/" + url.PathEscape(sku),
		Options:   &[]ConfigurableProductOption{},
		APIClient: apiClient,
	}

	err := mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return mConfigurableProduct, fmt.Errorf("error updating options from remote when getting configurable product by sku: %w", err)
	}
	return mConfigurableProduct, nil
}

func (mConfigurableProduct *MConfigurableProduct) UpdateOptionByID(o *ConfigurableProductOption) error {
	endpoint := fmt.Sprintf("%s/%s/%d", mConfigurableProduct.Route, configurableProductsOptionsRelative, o.ID)

	payLoad := createConfigurableProductByOptionPayload{
		Option: *o,
	}

	if _, err := mConfigurableProduct.APIClient.v1(context.Background(), http.MethodPut, endpoint, payLoad, nil, "update option for configurable product"); err != nil {
		return err
	}

	err := mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return fmt.Errorf("error updating options from remote after updating option by ID: %w", err)
	}
	return nil
}
