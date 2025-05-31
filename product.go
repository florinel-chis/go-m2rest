package magento2

import (
	"context" // Added context
	"fmt"

	"github.com/rs/zerolog/log"
	// "github.com/go-resty/resty/v2" // No longer needed directly here if all calls go via APIClient
)

const (
	products = "/products"
)

type MProduct struct {
	Route     string
	Product   *Product
	APIClient *Client
}

// CreateOrReplaceProduct now accepts context.Context
func CreateOrReplaceProduct(ctx context.Context, product *Product, saveOptions bool, apiClient *Client) (*MProduct, error) {
	mp := &MProduct{
		Product:   product,
		APIClient: apiClient,
	}

	// Pass context to internal method
	err := mp.createOrReplaceProduct(ctx, saveOptions)
	if err != nil {
		return mp, fmt.Errorf("error creating or replacing product: %w", err)
	}

	return mp, nil
}

// GetProductBySKU now accepts context.Context
func GetProductBySKU(ctx context.Context, sku string, apiClient *Client) (*MProduct, error) {
	mProduct := &MProduct{
		Route:     products + "/" + sku,
		Product:   &Product{},
		APIClient: apiClient,
	}

	// Pass context to internal method
	err := mProduct.UpdateProductFromRemote(ctx)
	if err != nil {
		return mProduct, fmt.Errorf("error updating product from remote when getting by SKU: %w", err)
	}

	return mProduct, nil
}

// createOrReplaceProduct now accepts context.Context
func (mProduct *MProduct) createOrReplaceProduct(ctx context.Context, saveOptions bool) error {
	endpoint := products
	// httpClient := mProduct.APIClient.HTTPClient // Not needed directly, use APIClient methods

	payLoad := AddProductPayload{
		Product:     *mProduct.Product,
		SaveOptions: saveOptions,
	}

	log.Debug().
		Str("sku", mProduct.Product.Sku).
		Bool("saveOptions", saveOptions).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Creating or replacing product with context")

	// Use APIClient's PostRouteAndDecode which now handles context
	// Assuming mProduct.Product will be populated by PostRouteAndDecode's target parameter
	err := mProduct.APIClient.PostRouteAndDecode(ctx, endpoint, payLoad, mProduct.Product, "create new product on remote")
	if err != nil {
		// PostRouteAndDecode will return context errors if applicable
		return err // No need to wrap again if PostRouteAndDecode does it well
	}

	productSKU := mayTrimSurroundingQuotes(mProduct.Product.Sku)
	mProduct.Route = products + "/" + productSKU

	// Note: The original code had `resp, err := httpClient.R()...` and then `mayReturnErrorForHTTPResponse(resp, ...)`
	// This logic is now encapsulated within PostRouteAndDecode.
	// If PostRouteAndDecode successfully unmarshals into mProduct.Product, we are good.

	return nil
}

// UpdateProductFromRemote now accepts context.Context
func (mProduct *MProduct) UpdateProductFromRemote(ctx context.Context) error {
	// httpClient := mProduct.APIClient.HTTPClient // Not needed directly

	log.Debug().
		Str("route", mProduct.Route).
		Msg("Updating product details from remote with context")

	// Use APIClient's GetRouteAndDecode
	err := mProduct.APIClient.GetRouteAndDecode(ctx, mProduct.Route, mProduct.Product, "get detailed product from remote")
	if err != nil {
		return err
	}

	// Logic previously handling resp and mayReturnErrorForHTTPResponse is now in GetRouteAndDecode
	return nil
}

// UpdateQuantityForStockItem now accepts context.Context
func (mProduct *MProduct) UpdateQuantityForStockItem(ctx context.Context, stockItem string, quantity int, isInStock bool) error {
	// httpClient := mProduct.APIClient.HTTPClient // Not needed directly

	updateStockPayload := updateStockPayload{StockItem: StockItem{Qty: quantity, IsInStock: isInStock}}
	endpoint := mProduct.Route + "/" + stockItemsRelative + "/" + stockItem

	log.Debug().
		Str("stockItem", stockItem).
		Str("sku", mProduct.Product.Sku).
		Int("quantity", quantity).
		Bool("isInStock", isInStock).
		Str("endpoint", endpoint).
		Interface("payload", updateStockPayload).
		Msg("Updating quantity for stock item of product with context")

	// Need a PutRouteAndDecode or similar in APIClient, or use httpClient directly for now if not available
	// For this subtask, I'll assume direct usage if PutRouteAndDecode isn't available yet in APIClient from prior step.
	// However, the goal is to use APIClient methods.
	// Let's assume a generic `PutRouteAndDecode` would be similar to PostRouteAndDecode.
	// If it doesn't exist, this part of the code would need adjustment after APIClient is extended.
	// For now, I'll modify it as if a `Put` method exists in the client that takes context.

	// Simulating how it would look if a Put method existed in APIClient:
	// err := mProduct.APIClient.PutRouteAndDecode(ctx, endpoint, updateStockPayload, nil, "update stock for product")
	// if err != nil {
	// 	return err
	// }
	// return nil

	// Reverting to direct use of httpClient for PUT as PutRouteAndDecode wasn't part of previous api_client.go changes.
	// This highlights a dependency: this function needs a way to make context-aware PUT requests.
	// The previous step only created GetRouteAndDecode and PostRouteAndDecode.
	// This subtask will use the existing httpClient and pass context.

	resp, err := mProduct.APIClient.HTTPClient.R().SetContext(ctx).SetBody(updateStockPayload).Put(endpoint)

	if err != nil {
		if ctx.Err() != nil {
			log.Error().Err(ctx.Err()).Str("endpoint", endpoint).Msg("Context error during PUT request for stock update")
			return ctx.Err()
		}
		log.Error().Err(err).Msg("Error updating stock for product")
		return fmt.Errorf("error updating stock for product: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Product stock updated response status from remote")

	// Using mayReturnErrorForHTTPResponse directly as we used httpClient directly
	httpErr := mayReturnErrorForHTTPResponse(resp, "update stock for product")
	if httpErr != nil {
		return httpErr
	}
	return nil
}
