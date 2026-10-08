package magento2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

// GetProductBySKU fetches a product (GET /products/{sku}); the SKU is
// path-escaped.
func GetProductBySKU(sku string, apiClient *Client) (*MProduct, error) {
	mProduct := &MProduct{
		Route:     products + "/" + url.PathEscape(sku),
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
	payLoad := AddProductPayload{
		Product:     *mProduct.Product,
		SaveOptions: saveOptions,
	}

	_, err := mProduct.APIClient.v1(context.Background(), http.MethodPost, products, payLoad, mProduct.Product, "create new product on remote")
	mProduct.Route = products + "/" + url.PathEscape(mayTrimSurroundingQuotes(mProduct.Product.Sku))
	return err
}

func (mProduct *MProduct) UpdateProductFromRemote() error {
	_, err := mProduct.APIClient.v1(context.Background(), http.MethodGet, mProduct.Route, nil, mProduct.Product, "get detailed product from remote")
	return err
}

func (mProduct *MProduct) UpdateQuantityForStockItem(stockItem string, quantity int, isInStock bool) error {
	payLoad := updateStockPayload{StockItem: StockItem{Qty: quantity, IsInStock: isInStock}}
	route := mProduct.Route + "/" + stockItemsRelative + "/" + url.PathEscape(stockItem)
	_, err := mProduct.APIClient.v1(context.Background(), http.MethodPut, route, payLoad, nil, "update stock for product")
	return err
}
