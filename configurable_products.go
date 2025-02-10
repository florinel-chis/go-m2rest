package magento2

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type MConfigurableProduct struct {
	Route     string
	Options   *[]Option
	APIClient *Client
}

func SetOptionForExistingConfigurableProduct(sku string, o *ConfigurableProductOption, apiClient *Client) (*MConfigurableProduct, error) {
	mConfigurableProduct := &MConfigurableProduct{
		Route:     configurableProducts + "/" + sku,
		Options:   &[]Option{},
		APIClient: apiClient,
	}
	endpoint := mConfigurableProduct.Route + "/" + configurableProductsOptionsRelative
	httpClient := apiClient.HTTPClient

	payLoad := createConfigurableProductByOptionPayload{
		Option: *o,
	}

	log.Debug().
		Str("sku", sku).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Setting option for configurable product")

	resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)

	if err != nil {
		return mConfigurableProduct, fmt.Errorf("error setting option for configurable product: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "create configurable product option")
	if httpErr != nil {
		return mConfigurableProduct, httpErr
	}

	err = mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return mConfigurableProduct, fmt.Errorf("error updating options from remote after setting option: %w", err)
	}

	return mConfigurableProduct, nil
}

func (mConfigurableProduct *MConfigurableProduct) UpdateOptionsFromRemote() error {
	httpClient := mConfigurableProduct.APIClient.HTTPClient
	optionsRoute := mConfigurableProduct.Route + "/" + configurableProductsOptionsAllRelative

	log.Debug().
		Str("route", optionsRoute).
		Msg("Updating options for configurable product from remote")

	resp, err := httpClient.R().SetResult(mConfigurableProduct.Options).Get(optionsRoute)

	if err != nil {
		log.Error().Err(err).Msg("Error updating options for configurable product from remote")
		return fmt.Errorf("error getting options for configurable product from remote: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get options for configurable product from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mConfigurableProduct *MConfigurableProduct) AddChildBySKU(sku string) error {
	httpClient := mConfigurableProduct.APIClient.HTTPClient
	payLoad := addChildSKUPayload{
		Sku: sku,
	}

	endpoint := fmt.Sprintf("%s/%s", mConfigurableProduct.Route, configurableProductsChildRelative)

	log.Debug().
		Str("sku", sku).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Adding child SKU to configurable product")

	resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)

	if err != nil {
		return fmt.Errorf("error adding child SKU to configurable product: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "add child by sku to configurable product")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func GetConfigurableProductBySKU(sku string, apiClient *Client) (*MConfigurableProduct, error) {
	mConfigurableProduct := &MConfigurableProduct{
		Route:     configurableProducts + "/" + sku,
		Options:   &[]Option{},
		APIClient: apiClient,
	}

	log.Debug().Str("sku", sku).Msg("Getting configurable product by SKU")

	err := mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return mConfigurableProduct, fmt.Errorf("error updating options from remote when getting configurable product by sku: %w", err)
	}
	return mConfigurableProduct, nil
}

func (mConfigurableProduct *MConfigurableProduct) UpdateOptionByID(o *ConfigurableProductOption) error {
	httpClient := mConfigurableProduct.APIClient.HTTPClient
	endpoint := fmt.Sprintf("%s/%s/%d", mConfigurableProduct.Route, configurableProductsOptionsRelative, o.ID)

	payLoad := createConfigurableProductByOptionPayload{
		Option: *o,
	}

	log.Debug().
		Int("optionID", o.ID).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Updating option by ID for configurable product")

	resp, err := httpClient.R().SetBody(payLoad).Put(endpoint)

	if err != nil {
		return fmt.Errorf("error updating option by ID for configurable product: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "update option for configurable product")
	if httpErr != nil {
		return httpErr
	}

	err = mConfigurableProduct.UpdateOptionsFromRemote()
	if err != nil {
		return fmt.Errorf("error updating options from remote after updating option by ID: %w", err)
	}
	return nil
}
