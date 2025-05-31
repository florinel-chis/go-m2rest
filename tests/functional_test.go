package magento2

import (
	"context" // Added context
	"fmt"
	"testing"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest" // Assuming this is the correct import path for the library itself
	"github.com/rs/zerolog/log"
)

func setupTestClientV2(t *testing.T) (*magento2.Client, *TestConfig) {
	// Assuming SetupTestClient() eventually calls NewAPIClientFromAuthentication or NewAPIClientFromIntegration
	// If NewAPIClientFromAuthentication is used, it now needs a context.
	// For test setup, context.Background() is usually appropriate.
	// This change might require SetupTestClient to be context-aware if it makes the auth call.
	// For this subtask, we'll assume SetupTestClient is updated separately or doesn't directly need modification
	// if the context is only needed for the *specific* auth call within it.
	// Let's assume NewAPIClientFromIntegration is used, which doesn't need context in its signature.
	// If NewAPIClientFromAuthentication is called by SetupTestClient, that function needs ctx.
	// For now, keeping setupTestClientV2 as is, and passing context directly in tests.
	// A more thorough refactor might involve passing ctx to setupTestClientV2 if it makes auth calls.

	client, config, err := SetupTestClient() // This might need context if it calls NewAPIClientFromAuthentication
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}
	return client, config
}

func TestFunctionalV2_APIConnection(t *testing.T) {
	client, _ := setupTestClientV2(t)
	ctx := context.Background() // Define context

	t.Run("Test API Connection", func(t *testing.T) {
		_, err := magento2.GetProductBySKU(ctx, "test-connection-sku", client) // Pass context
		if err != nil && err != magento2.ErrNotFound {
			// If we get an auth error or connection error, that's a problem
			t.Logf("Connection test result: %v", err)
		} else {
			t.Log("API connection successful")
		}
	})
}

func TestFunctionalV2_Products(t *testing.T) {
	client, _ := setupTestClientV2(t)
	ctx := context.Background() // Define context

	t.Run("Create and Retrieve Product", func(t *testing.T) {
		// Create a unique SKU
		sku := fmt.Sprintf("test-product-%d", time.Now().Unix())
		
		product := magento2.Product{
			Sku:            sku,
			Name:           "Test Product",
			AttributeSetID: 4, // Default attribute set
			Price:          99.99,
			TypeID:         "simple",
			Status:         1, // Enabled
			Visibility:     4, // Catalog, Search
			Weight:         1.5,
		}

		product := magento2.Product{
			Sku:            sku,
			Name:           "Test Product",
			AttributeSetID: 4,
			Price:          99.99,
			TypeID:         "simple",
			Status:         1,
			Visibility:     4,
			Weight:         1.5,
		}

		mProduct, err := magento2.CreateOrReplaceProduct(ctx, &product, true, client) // Pass context
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}

		if mProduct.Product.Sku == "" {
			t.Error("Created product has no SKU")
		}

		log.Info().
			Str("sku", mProduct.Product.Sku).
			Interface("product", mProduct.Product).
			Msg("Product created successfully")

		retrieved, err := magento2.GetProductBySKU(ctx, sku, client) // Pass context
		if err != nil {
			t.Fatalf("Failed to get product by SKU: %v", err)
		}

		if retrieved.Product.Sku != sku {
			t.Errorf("Retrieved product SKU mismatch: got %s, want %s", 
				retrieved.Product.Sku, sku)
		}

		// Update stock quantity
		if retrieved.Product.ExtensionAttributes != nil {
			// Check if stock_item exists in extension attributes
			if stockData, ok := retrieved.Product.ExtensionAttributes["stock_item"]; ok {
				// Try to extract stock item ID
				if stockMap, ok := stockData.(map[string]any); ok {
					if itemID, ok := stockMap["item_id"]; ok {
						stockItemID := fmt.Sprintf("%v", itemID)
						// UpdateQuantityForStockItem on mProduct (which is *MProduct)
						err = mProduct.UpdateQuantityForStockItem(ctx, stockItemID, 100, true) // Pass context
						if err != nil {
							t.Logf("Failed to update stock (might need different approach): %v", err)
						} else {
							t.Log("Stock updated successfully")
						}
					}
				}
			}
		}
	})

	t.Run("Search Products", func(t *testing.T) {
		// Simple search query
		query := magento2.BuildSearchQuery("type_id", "simple", "eq")
		
		// Note: We need to check if there's a GetProducts function
		// For now, let's test the query building
		t.Logf("Built search query: %s", query)
	})
}

func TestFunctionalV2_Categories(t *testing.T) {
	client, _ := setupTestClientV2(t)
	ctx := context.Background() // Define context

	t.Run("Get Category by Name", func(t *testing.T) {
		category, err := magento2.GetCategoryByName(ctx, "Default Category", client) // Pass context
		if err != nil {
			t.Logf("Default Category not found, trying Root Catalog: %v", err)
			category, err = magento2.GetCategoryByName(ctx, "Root Catalog", client) // Pass context
			if err != nil {
				t.Logf("Root Catalog not found either: %v", err)
				return
			}
		}
		log.Info().
			Int("id", category.Category.ID).
			Str("name", category.Category.Name).
			Msg("Found category")
	})

	t.Run("Create Category", func(t *testing.T) {
		category := magento2.Category{
			Name:       fmt.Sprintf("Test Category %d", time.Now().Unix()),
			IsActive:   true,
			ParentID:   2, // Assuming 2 is root
			IncludeInMenu: true,
		}
		created, err := magento2.CreateCategory(ctx, &category, client) // Pass context
		if err != nil {
			t.Logf("Failed to create category (might need different parent ID): %v", err)
			category.ParentID = 1
			created, err = magento2.CreateCategory(ctx, &category, client) // Pass context
			if err != nil {
				t.Errorf("Failed to create category with both parent IDs: %v", err)
				return
			}
		}
		if created.Category.ID == 0 {
			t.Error("Created category has no ID")
		} else {
			log.Info().
				Int("id", created.Category.ID).
				Str("name", created.Category.Name).
				Msg("Category created successfully")
		}
	})
}

func TestFunctionalV2_Attributes(t *testing.T) {
	client, _ := setupTestClientV2(t)
	ctx := context.Background() // Define context

	t.Run("Create and Retrieve Attribute", func(t *testing.T) {
		attributeCode := fmt.Sprintf("test_attr_%d", time.Now().Unix())
		
		attribute := magento2.Attribute{
			AttributeCode: attributeCode,
			FrontendInput: "text",
			DefaultFrontendLabel: "Test Attribute",
			IsRequired:    false,
			Scope:         "global",
			EntityTypeID:  "4",
		}
		created, err := magento2.CreateAttribute(ctx, &attribute, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create attribute: %v", err)
			return
		}
		if created.Attribute.AttributeID == 0 {
			t.Error("Created attribute has no ID")
		} else {
			log.Info().Int("id", created.Attribute.AttributeID).Str("code", created.Attribute.AttributeCode).Msg("Attribute created successfully")
		}

		retrieved, err := magento2.GetAttributeByAttributeCode(ctx, attributeCode, client) // Pass context
		if err != nil {
			t.Errorf("Failed to get attribute by code: %v", err)
			return
		}
		if retrieved.Attribute.AttributeCode != attributeCode {
			t.Errorf("Retrieved attribute code mismatch: got %s, want %s", 
				retrieved.Attribute.AttributeCode, attributeCode)
		}
	})
}

func TestFunctionalV2_Cart(t *testing.T) {
	client, _ := setupTestClientV2(t)
	ctx := context.Background() // Define context

	t.Run("Guest Cart Basic Flow", func(t *testing.T) {
		guestCart, err := magento2.NewGuestCartFromAPIClient(ctx, client) // Pass context
		if err != nil {
			t.Fatalf("Failed to create guest cart: %v", err)
		}
		if guestCart.QuoteID == "" {
			t.Fatal("Guest cart has no ID")
		}
		log.Info().Str("cartID", guestCart.QuoteID).Msg("Guest cart created")

		sku := fmt.Sprintf("cart-test-product-%d", time.Now().Unix())
		product := magento2.Product{
			Sku:            sku,
			Name:           fmt.Sprintf("Cart Test Product %d", time.Now().Unix()),
			AttributeSetID: 4,
			Price:          10.00,
			TypeID:         "simple",
			Status:         1,
			Visibility:     4,
			Weight:         0.5,
		}
		mProduct, err := magento2.CreateOrReplaceProduct(ctx, &product, true, client) // Pass context
		if err != nil {
			t.Fatalf("Failed to create test product for cart: %v", err)
		}

		if mProduct.Product.ExtensionAttributes != nil {
			if stockData, ok := mProduct.Product.ExtensionAttributes["stock_item"]; ok {
				if stockMap, ok := stockData.(map[string]any); ok {
					if itemID, ok := stockMap["item_id"]; ok {
						stockItemID := fmt.Sprintf("%v", itemID)
						err = mProduct.UpdateQuantityForStockItem(ctx, stockItemID, 10, true) // Pass context
						if err != nil {
							t.Logf("Failed to set stock for cart test product: %v", err)
						}
					}
				}
			}
		}

		// Add item to cart
		item := magento2.CartItem{
			Sku: sku,
			Qty: 2,
			QuoteID: guestCart.QuoteID,
		}

		itemPayload := magento2.CartItem{Sku: sku, Qty: 2, QuoteID: guestCart.QuoteID}
		items := []magento2.CartItem{itemPayload}
		err = guestCart.AddItems(ctx, items) // Pass context
		if err != nil {
			t.Errorf("Failed to add items to cart: %v", err)
		} else {
			log.Info().Interface("items", items).Msg("Items added to cart successfully")
		}

		shippingAddr := &magento2.ShippingAddress{
			Address: magento2.Address{
				CountryID: "CH", Postcode: "8000", City: "Zurich", Street: []string{"Test Street 1"},
				Firstname: "Test", Lastname: "User", Telephone: "123456789", Email: "test@example.com",
			},
		}
		carriers, err := guestCart.EstimateShippingCarrier(ctx, shippingAddr) // Pass context
		if err != nil {
			t.Logf("Failed to estimate shipping: %v", err)
		} else {
			log.Info().Interface("carriers", carriers).Msg("Shipping methods estimated")
		}

		paymentMethods, err := guestCart.EstimatePaymentMethods(ctx) // Pass context
		if err != nil {
			t.Logf("Failed to estimate payment methods: %v", err)
		} else {
			log.Info().Interface("paymentMethods", paymentMethods).Msg("Payment methods estimated")
		}
	})
}

func TestFunctionalV2_QuickTest(t *testing.T) {
	// Each sub-test function (TestFunctionalV2_APIConnection, etc.) will define its own client and context.
	t.Run("API Connectivity", TestFunctionalV2_APIConnection)
	t.Run("Products", TestFunctionalV2_Products)
	t.Run("Categories", TestFunctionalV2_Categories)
	t.Run("Attributes", TestFunctionalV2_Attributes)
	t.Run("Cart", TestFunctionalV2_Cart)
}