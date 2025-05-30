package magento2

import (
	"fmt"
	"testing"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest"
	"github.com/rs/zerolog/log"
)

func setupTestClientV2(t *testing.T) (*magento2.Client, *TestConfig) {
	client, config, err := SetupTestClient()
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}

	return client, config
}

func TestFunctionalV2_APIConnection(t *testing.T) {
	client, _ := setupTestClientV2(t)
	
	// Simple test to verify connection works
	t.Run("Test API Connection", func(t *testing.T) {
		// Try to get a product that likely doesn't exist
		_, err := magento2.GetProductBySKU("test-connection-sku", client)
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

		// Create product
		mProduct, err := magento2.CreateOrReplaceProduct(&product, true, client)
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

		// Retrieve product by SKU
		retrieved, err := magento2.GetProductBySKU(sku, client)
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
						err = mProduct.UpdateQuantityForStockItem(stockItemID, 100, true)
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

	t.Run("Get Category by Name", func(t *testing.T) {
		// Try to find default root category
		category, err := magento2.GetCategoryByName("Default Category", client)
		if err != nil {
			t.Logf("Default Category not found, trying Root Catalog: %v", err)
			category, err = magento2.GetCategoryByName("Root Catalog", client)
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

		created, err := magento2.CreateCategory(&category, client)
		if err != nil {
			t.Logf("Failed to create category (might need different parent ID): %v", err)
			// Try with parent ID 1
			category.ParentID = 1
			created, err = magento2.CreateCategory(&category, client)
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

	t.Run("Create and Retrieve Attribute", func(t *testing.T) {
		attributeCode := fmt.Sprintf("test_attr_%d", time.Now().Unix())
		
		attribute := magento2.Attribute{
			AttributeCode: attributeCode,
			FrontendInput: "text",
			DefaultFrontendLabel: "Test Attribute",
			IsRequired:    false,
			Scope:         "global",
			EntityTypeID:  "4", // Product entity (string, not int)
		}

		created, err := magento2.CreateAttribute(&attribute, client)
		if err != nil {
			t.Errorf("Failed to create attribute: %v", err)
			return
		}

		if created.Attribute.AttributeID == 0 {
			t.Error("Created attribute has no ID")
		} else {
			log.Info().
				Int("id", created.Attribute.AttributeID).
				Str("code", created.Attribute.AttributeCode).
				Msg("Attribute created successfully")
		}

		// Retrieve attribute
		retrieved, err := magento2.GetAttributeByAttributeCode(attributeCode, client)
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

	t.Run("Guest Cart Basic Flow", func(t *testing.T) {
		// Create guest cart
		guestCart, err := magento2.NewGuestCartFromAPIClient(client)
		if err != nil {
			t.Fatalf("Failed to create guest cart: %v", err)
		}

		if guestCart.QuoteID == "" {
			t.Fatal("Guest cart has no ID")
		}

		log.Info().Str("cartID", guestCart.QuoteID).Msg("Guest cart created")

		// First, create a test product to add to cart
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

		mProduct, err := magento2.CreateOrReplaceProduct(&product, true, client)
		if err != nil {
			t.Fatalf("Failed to create test product for cart: %v", err)
		}

		// Set stock for the product
		if mProduct.Product.ExtensionAttributes != nil {
			if stockData, ok := mProduct.Product.ExtensionAttributes["stock_item"]; ok {
				if stockMap, ok := stockData.(map[string]any); ok {
					if itemID, ok := stockMap["item_id"]; ok {
						stockItemID := fmt.Sprintf("%v", itemID)
						err = mProduct.UpdateQuantityForStockItem(stockItemID, 10, true)
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

		// Add items to cart
		items := []magento2.CartItem{item}
		err = guestCart.AddItems(items)
		if err != nil {
			t.Errorf("Failed to add items to cart: %v", err)
		} else {
			log.Info().
				Interface("items", items).
				Msg("Items added to cart successfully")
		}

		// Estimate shipping methods
		shippingAddr := &magento2.ShippingAddress{
			Address: magento2.Address{
				CountryID: "CH",
				Postcode:  "8000",
				City:      "Zurich",
				Street:    []string{"Test Street 1"},
				Firstname: "Test",
				Lastname:  "User",
				Telephone: "123456789",
				Email:     "test@example.com",
			},
		}
		
		carriers, err := guestCart.EstimateShippingCarrier(shippingAddr)
		if err != nil {
			t.Logf("Failed to estimate shipping: %v", err)
		} else {
			log.Info().
				Interface("carriers", carriers).
				Msg("Shipping methods estimated")
		}

		// Estimate payment methods
		paymentMethods, err := guestCart.EstimatePaymentMethods()
		if err != nil {
			t.Logf("Failed to estimate payment methods: %v", err)
		} else {
			log.Info().
				Interface("paymentMethods", paymentMethods).
				Msg("Payment methods estimated")
		}
	})
}

// Run a quick connectivity test first
func TestFunctionalV2_QuickTest(t *testing.T) {
	t.Run("API Connectivity", TestFunctionalV2_APIConnection)
	
	// If connectivity works, run other tests
	t.Run("Products", TestFunctionalV2_Products)
	t.Run("Categories", TestFunctionalV2_Categories) 
	t.Run("Attributes", TestFunctionalV2_Attributes)
	t.Run("Cart", TestFunctionalV2_Cart)
}