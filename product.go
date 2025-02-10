package magento2

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

const (
	products = "/products"
)

type MProduct struct {
	Route     string
	Product   *Product
	APIClient *Client
}

func CreateOrReplaceProduct(product *Product, saveOptions bool, apiClient *Client) (*MProduct, error) {
	mp := &MProduct{
		Product:   product,
		APIClient: apiClient,
	}

	err := mp.createOrReplaceProduct(saveOptions)
	if err != nil {
		return mp, fmt.Errorf("error creating or replacing product: %w", err)
	}

	return mp, nil
}

func GetProductBySKU(sku string, apiClient *Client) (*MProduct, error) {
	mProduct := &MProduct{
		Route:     products + "/" + sku,
		Product:   &Product{},
		APIClient: apiClient,
	}

	err := mProduct.UpdateProductFromRemote()
	if err != nil {
		return mProduct, fmt.Errorf("error updating product from remote when getting by SKU: %w", err)
	}

	return mProduct, nil
}

func (mProduct *MProduct) createOrReplaceProduct(saveOptions bool) error {
	endpoint := products
	httpClient := mProduct.APIClient.HTTPClient

	payLoad := AddProductPayload{
		Product:     *mProduct.Product,
		SaveOptions: saveOptions,
	}

	log.Debug().
		Str("sku", mProduct.Product.Sku).
		Bool("saveOptions", saveOptions).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Creating or replacing product")

	resp, err := httpClient.R().SetBody(payLoad).SetResult(mProduct.Product).Post(endpoint)
	productSKU := mayTrimSurroundingQuotes(mProduct.Product.Sku)
	mProduct.Route = products + "/" + productSKU

	if err != nil {
		log.Error().Err(err).Msg("Error creating or replacing product")
		return fmt.Errorf("error creating or replacing product: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Product creation/replacement response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "create new product on remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mProduct *MProduct) UpdateProductFromRemote() error {
	httpClient := mProduct.APIClient.HTTPClient

	log.Debug().
		Str("route", mProduct.Route).
		Msg("Updating product details from remote")

	resp, err := httpClient.R().SetResult(mProduct.Product).Get(mProduct.Route)

	if err != nil {
		log.Error().Err(err).Msg("Error updating product details from remote")
		return fmt.Errorf("error updating product details from remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Product details updated response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "get detailed product from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mProduct *MProduct) UpdateQuantityForStockItem(stockItem string, quantity int, isInStock bool) error {
	httpClient := mProduct.APIClient.HTTPClient

	updateStockPayload := updateStockPayload{StockItem: StockItem{Qty: quantity, IsInStock: isInStock}}

	log.Debug().
		Str("stockItem", stockItem).
		Str("sku", mProduct.Product.Sku).
		Int("quantity", quantity).
		Bool("isInStock", isInStock).
		Str("endpoint", mProduct.Route+"/"+stockItemsRelative+"/"+stockItem).
		Interface("payload", updateStockPayload).
		Msg("Updating quantity for stock item of product")

	resp, err := httpClient.R().SetBody(updateStockPayload).Put(mProduct.Route + "/" + stockItemsRelative + "/" + stockItem)

	if err != nil {
		log.Error().Err(err).Msg("Error updating stock for product")
		return fmt.Errorf("error updating stock for product: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Product stock updated response status from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "update stock for product")
	if httpErr != nil {
		return httpErr
	}
	return nil
}
