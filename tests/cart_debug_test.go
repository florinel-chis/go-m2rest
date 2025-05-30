package magento2

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest"
	"github.com/rs/zerolog/log"
)

func TestCartDebug_AddItemPayload(t *testing.T) {
	client, _, err := SetupTestClient()
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}

	// Create guest cart
	guestCart, err := magento2.NewGuestCartFromAPIClient(client)
	if err != nil {
		t.Fatalf("Failed to create guest cart: %v", err)
	}

	// Create a simple test product
	sku := fmt.Sprintf("debug-product-%d", time.Now().Unix())
	product := magento2.Product{
		Sku:            sku,
		Name:           fmt.Sprintf("Debug Product %d", time.Now().Unix()),
		AttributeSetID: 4,
		Price:          5.00,
		TypeID:         "simple",
		Status:         1,
		Visibility:     4,
		Weight:         0.1,
	}

	mProduct, err := magento2.CreateOrReplaceProduct(&product, true, client)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	// Set stock
	if mProduct.Product.ExtensionAttributes != nil {
		if stockData, ok := mProduct.Product.ExtensionAttributes["stock_item"]; ok {
			if stockMap, ok := stockData.(map[string]any); ok {
				if itemID, ok := stockMap["item_id"]; ok {
					stockItemID := fmt.Sprintf("%v", itemID)
					err = mProduct.UpdateQuantityForStockItem(stockItemID, 10, true)
					if err != nil {
						t.Logf("Failed to set stock: %v", err)
					}
				}
			}
		}
	}

	log.Info().Str("cartID", guestCart.QuoteID).Str("sku", sku).Msg("Testing cart item addition")

	// Let's try different payload structures to see what Magento expects
	
	// Test 1: Current structure
	t.Run("Current Structure", func(t *testing.T) {
		item := magento2.CartItem{
			Sku: sku,
			Qty: 1,
			QuoteID: guestCart.QuoteID,
		}
		
		type PayLoad struct {
			CartItem magento2.CartItem `json:"cartItem"`
		}
		
		payload := &PayLoad{CartItem: item}
		payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
		t.Logf("Current payload structure:\n%s", payloadJSON)
		
		// Try adding with current structure
		err := guestCart.AddItems([]magento2.CartItem{item})
		if err != nil {
			t.Logf("Current structure failed: %v", err)
		} else {
			t.Log("Current structure worked!")
			return
		}
	})

	// Test 2: Try different payload structure based on Magento standards
	t.Run("Alternative Structure 1", func(t *testing.T) {
		// Based on Magento 2 docs, it might expect a different structure
		endpoint := guestCart.Route + "/items"
		httpClient := client.HTTPClient
		
		// Try structure: {"cartItem": {"sku": "...", "qty": 1}}
		type SimpleCartItem struct {
			Sku string  `json:"sku"`
			Qty float64 `json:"qty"`
		}
		
		type PayLoad struct {
			CartItem SimpleCartItem `json:"cartItem"`
		}
		
		item := SimpleCartItem{
			Sku: sku,
			Qty: 1,
		}
		
		payload := &PayLoad{CartItem: item}
		payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
		t.Logf("Alternative payload structure 1:\n%s", payloadJSON)
		
		resp, err := httpClient.R().SetBody(payload).Post(endpoint)
		if err != nil {
			t.Logf("Request error: %v", err)
		} else {
			t.Logf("Response status: %d", resp.StatusCode())
			t.Logf("Response body: %s", resp.String())
			
			if resp.IsSuccess() {
				t.Log("Alternative structure 1 worked!")
				return
			}
		}
	})

	// Test 3: Try direct item structure
	t.Run("Alternative Structure 2", func(t *testing.T) {
		endpoint := guestCart.Route + "/items"
		httpClient := client.HTTPClient
		
		// Try direct structure without wrapper
		type DirectCartItem struct {
			Sku string  `json:"sku"`
			Qty float64 `json:"qty"`
		}
		
		item := DirectCartItem{
			Sku: sku,
			Qty: 1,
		}
		
		payloadJSON, _ := json.MarshalIndent(item, "", "  ")
		t.Logf("Direct payload structure:\n%s", payloadJSON)
		
		resp, err := httpClient.R().SetBody(item).Post(endpoint)
		if err != nil {
			t.Logf("Request error: %v", err)
		} else {
			t.Logf("Response status: %d", resp.StatusCode())
			t.Logf("Response body: %s", resp.String())
			
			if resp.IsSuccess() {
				t.Log("Direct structure worked!")
				return
			}
		}
	})

	// Test 4: Check what Magento expects by looking at the current cart structure
	t.Run("Inspect Current Cart", func(t *testing.T) {
		endpoint := guestCart.Route
		httpClient := client.HTTPClient
		
		resp, err := httpClient.R().Get(endpoint)
		if err != nil {
			t.Logf("Failed to get cart: %v", err)
		} else {
			t.Logf("Current cart structure:\n%s", resp.String())
		}
	})
}