package magento2

import (
	"testing"

	magento2 "github.com/florinel-chis/go-m2rest"
	"github.com/rs/zerolog/log"
)

func TestCartDebug_UseExistingProduct(t *testing.T) {
	client, _, err := SetupTestClient()
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}

	// Create guest cart
	guestCart, err := magento2.NewGuestCartFromAPIClient(client)
	if err != nil {
		t.Fatalf("Failed to create guest cart: %v", err)
	}

	log.Info().Str("cartID", guestCart.QuoteID).Msg("Testing with existing products")

	// Try with some common/existing SKUs that might exist in the system
	testSKUs := []string{
		"test-product-1748593413", // From our earlier successful test
		"24-MB01",    // Common Magento sample product
		"24-MB02",    // Common Magento sample product  
		"24-WG80",    // Common Magento sample product
		"simple",     // Generic simple product
		"configurable", // Generic configurable product
	}

	for _, sku := range testSKUs {
		t.Run("Test_SKU_"+sku, func(t *testing.T) {
			// First check if product exists
			product, err := magento2.GetProductBySKU(sku, client)
			if err != nil {
				t.Logf("Product %s does not exist: %v", sku, err)
				return
			}

			t.Logf("Found product: %s (ID: %d, Status: %d)", product.Product.Name, product.Product.ID, product.Product.Status)

			// Check if product is enabled and in stock
			if product.Product.Status != 1 {
				t.Logf("Product %s is not enabled (status: %d)", sku, product.Product.Status)
				return
			}

			// Try to add to cart
			item := magento2.CartItem{
				Sku: sku,
				Qty: 1,
				QuoteID: guestCart.QuoteID,
			}

			err = guestCart.AddItems([]magento2.CartItem{item})
			if err != nil {
				t.Logf("Failed to add existing product %s to cart: %v", sku, err)
			} else {
				t.Logf("Successfully added existing product %s to cart!", sku)
				return // Success! Stop testing other SKUs
			}
		})
	}
}