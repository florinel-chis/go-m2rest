package magento2

import (
	"context" // Added context
	"fmt"
	"testing"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest" // Assuming this is the correct import path
	"github.com/rs/zerolog/log"
)

func setupAdvancedTestClient(t *testing.T) (*magento2.Client, *TestConfig) {
	// Same note as in functional_test.go regarding SetupTestClient and context for authentication.
	client, config, err := SetupTestClient()
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}
	return client, config
}

func TestAdvancedProducts_ConfigurableProduct(t *testing.T) {
	client, _ := setupAdvancedTestClient(t)
	ctx := context.Background() // Define context

	t.Run("Create Configurable Product with Variations", func(t *testing.T) {
		timestamp := time.Now().Unix()
		colorAttr := magento2.Attribute{
			AttributeCode:        fmt.Sprintf("test_color_%d", timestamp),
			FrontendInput:        "select",
			DefaultFrontendLabel: "Test Color",
			IsRequired:           false, Scope: "global", EntityTypeID: "4", IsUserDefined: true,
			Options: []magento2.Option{
				{Label: "Red", Value: "red", SortOrder: 1}, {Label: "Blue", Value: "blue", SortOrder: 2}, {Label: "Green", Value: "green", SortOrder: 3},
			},
		}
		sizeAttr := magento2.Attribute{
			AttributeCode:        fmt.Sprintf("test_size_%d", timestamp),
			FrontendInput:        "select",
			DefaultFrontendLabel: "Test Size",
			IsRequired:           false, Scope: "global", EntityTypeID: "4", IsUserDefined: true,
			Options: []magento2.Option{
				{Label: "Small", Value: "s", SortOrder: 1}, {Label: "Medium", Value: "m", SortOrder: 2}, {Label: "Large", Value: "l", SortOrder: 3},
			},
		}

		createdColorAttr, err := magento2.CreateAttribute(ctx, &colorAttr, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create color attribute: %v", err)
			return
		}
		log.Info().Str("code", createdColorAttr.Attribute.AttributeCode).Int("id", createdColorAttr.Attribute.AttributeID).Msg("Color attribute created")

		createdSizeAttr, err := magento2.CreateAttribute(ctx, &sizeAttr, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create size attribute: %v", err)
			return
		}
		log.Info().Str("code", createdSizeAttr.Attribute.AttributeCode).Int("id", createdSizeAttr.Attribute.AttributeID).Msg("Size attribute created")

		configurableSku := fmt.Sprintf("configurable-test-%d", timestamp)
		configurableProduct := magento2.Product{
			Sku: configurableSku, Name: fmt.Sprintf("Configurable Test Product %d", timestamp),
			AttributeSetID: 4, Price: 0, TypeID: "configurable", Status: 1, Visibility: 4, Weight: 1.0,
		}
		mConfigurable, err := magento2.CreateOrReplaceProduct(ctx, &configurableProduct, true, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create configurable product: %v", err)
			return
		}
		log.Info().Str("sku", mConfigurable.Product.Sku).Int("id", mConfigurable.Product.ID).Msg("Configurable product created")

		variations := []struct {color string; size  string; price float64}{
			{"red", "s", 25.00}, {"red", "m", 27.00}, {"blue", "s", 26.00}, {"blue", "l", 30.00},
		}
		var childProducts []*magento2.MProduct
		for i, variation := range variations {
			childSku := fmt.Sprintf("%s-%s-%s", configurableSku, variation.color, variation.size)
			var colorValue, sizeValue string
			for _, option := range createdColorAttr.Attribute.Options { if option.Label == variation.color || option.Value == variation.color { colorValue = option.Value; break } }
			for _, option := range createdSizeAttr.Attribute.Options { if option.Label == variation.size || option.Value == variation.size { sizeValue = option.Value; break } }
			
			childProduct := magento2.Product{
				Sku: childSku, Name: fmt.Sprintf("%s - %s %s", configurableProduct.Name, variation.color, variation.size),
				AttributeSetID: 4, Price: variation.price, TypeID: "simple", Status: 1, Visibility: 1, Weight: 1.0,
				CustomAttributes: []map[string]any{
					{"attribute_code": createdColorAttr.Attribute.AttributeCode, "value": colorValue},
					{"attribute_code": createdSizeAttr.Attribute.AttributeCode, "value": sizeValue},
				},
			}
			mChild, err := magento2.CreateOrReplaceProduct(ctx, &childProduct, true, client) // Pass context
			if err != nil {
				t.Errorf("Failed to create child product %d: %v", i, err)
				continue
			}
			if mChild.Product.ExtensionAttributes != nil {
				if stockData, ok := mChild.Product.ExtensionAttributes["stock_item"]; ok {
					if stockMap, ok := stockData.(map[string]any); ok {
						if itemID, ok := stockMap["item_id"]; ok {
							stockItemID := fmt.Sprintf("%v", itemID)
							err = mChild.UpdateQuantityForStockItem(ctx, stockItemID, 50, true) // Pass context
							if err != nil { t.Logf("Failed to set stock for child product: %v", err) }
						}
					}
				}
			}
			childProducts = append(childProducts, mChild)
			log.Info().Str("sku", mChild.Product.Sku).Float64("price", mChild.Product.Price).Msg("Child product created")
		}
		if len(childProducts) > 0 {
			t.Logf("Created configurable product with %d child products. Manual linking required in Magento admin.", len(childProducts))
		}
		t.Logf("Created configurable product with %d variations", len(childProducts))
	})
}

func TestAdvancedProducts_BundleProduct(t *testing.T) {
	client, _ := setupAdvancedTestClient(t)
	ctx := context.Background() // Define context

	t.Run("Create Bundle Product", func(t *testing.T) {
		timestamp := time.Now().Unix()
		bundleComponents := []struct {name  string; price float64}{
			{"Bundle Item A", 15.00}, {"Bundle Item B", 20.00}, {"Bundle Item C", 10.00},
		}
		var componentProducts []*magento2.MProduct
		for i, component := range bundleComponents {
			componentSku := fmt.Sprintf("bundle-component-%d-%d", timestamp, i)
			componentProduct := magento2.Product{
				Sku: componentSku, Name: fmt.Sprintf("%s %d", component.name, timestamp),
				AttributeSetID: 4, Price: component.price, TypeID: "simple", Status: 1, Visibility: 1, Weight: 0.5,
			}
			mComponent, err := magento2.CreateOrReplaceProduct(ctx, &componentProduct, true, client) // Pass context
			if err != nil {
				t.Errorf("Failed to create bundle component %d: %v", i, err)
				continue
			}
			if mComponent.Product.ExtensionAttributes != nil {
				if stockData, ok := mComponent.Product.ExtensionAttributes["stock_item"]; ok {
					if stockMap, ok := stockData.(map[string]any); ok {
						if itemID, ok := stockMap["item_id"]; ok {
							stockItemID := fmt.Sprintf("%v", itemID)
							err = mComponent.UpdateQuantityForStockItem(ctx, stockItemID, 100, true) // Pass context
							if err != nil { t.Logf("Failed to set stock for bundle component: %v", err) }
						}
					}
				}
			}
			componentProducts = append(componentProducts, mComponent)
			log.Info().Str("sku", mComponent.Product.Sku).Float64("price", mComponent.Product.Price).Msg("Bundle component created")
		}

		bundleSku := fmt.Sprintf("bundle-test-%d", timestamp)
		bundleProductDef := magento2.Product{ // Renamed variable to avoid conflict
			Sku: bundleSku, Name: fmt.Sprintf("Bundle Test Product %d", timestamp),
			AttributeSetID: 4, Price: 0, TypeID: "bundle", Status: 1, Visibility: 4, Weight: 0,
		}
		mBundle, err := magento2.CreateOrReplaceProduct(ctx, &bundleProductDef, true, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create bundle product: %v", err)
			return
		}
		log.Info().Str("sku", mBundle.Product.Sku).Int("id", mBundle.Product.ID).Msg("Bundle product created")
		t.Logf("Created bundle product with %d component products. Manual linking required in Magento admin.", len(componentProducts))
	})
}

func TestAdvancedProducts_VirtualProduct(t *testing.T) {
	client, _ := setupAdvancedTestClient(t)
	ctx := context.Background() // Define context

	t.Run("Create Virtual Product", func(t *testing.T) {
		timestamp := time.Now().Unix()
		virtualProductDef := magento2.Product{ // Renamed variable
			Sku: fmt.Sprintf("virtual-test-%d", timestamp), Name: fmt.Sprintf("Virtual Service %d", timestamp),
			AttributeSetID: 4, Price: 99.99, TypeID: "virtual", Status: 1, Visibility: 4,
		}
		mVirtual, err := magento2.CreateOrReplaceProduct(ctx, &virtualProductDef, true, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create virtual product: %v", err)
			return
		}
		log.Info().Str("sku", mVirtual.Product.Sku).Float64("price", mVirtual.Product.Price).Str("type", mVirtual.Product.TypeID).Msg("Virtual product created")
		t.Logf("Virtual product created successfully: %s", mVirtual.Product.Sku)
	})
}

func TestAdvancedProducts_GroupedProduct(t *testing.T) {
	client, _ := setupAdvancedTestClient(t)
	ctx := context.Background() // Define context

	t.Run("Create Grouped Product", func(t *testing.T) {
		timestamp := time.Now().Unix()
		groupComponents := []struct {name  string; price float64}{
			{"Group Item 1", 12.00}, {"Group Item 2", 18.00}, {"Group Item 3", 24.00},
		}
		var componentProducts []*magento2.MProduct // Renamed from groupedProducts
		for i, component := range groupComponents {
			componentSku := fmt.Sprintf("group-item-%d-%d", timestamp, i)
			componentProduct := magento2.Product{
				Sku: componentSku, Name: fmt.Sprintf("%s %d", component.name, timestamp),
				AttributeSetID: 4, Price: component.price, TypeID: "simple", Status: 1, Visibility: 4, Weight: 1.0,
			}
			mComponent, err := magento2.CreateOrReplaceProduct(ctx, &componentProduct, true, client) // Pass context
			if err != nil {
				t.Errorf("Failed to create group component %d: %v", i, err)
				continue
			}
			if mComponent.Product.ExtensionAttributes != nil {
				if stockData, ok := mComponent.Product.ExtensionAttributes["stock_item"]; ok {
					if stockMap, ok := stockData.(map[string]any); ok {
						if itemID, ok := stockMap["item_id"]; ok {
							stockItemID := fmt.Sprintf("%v", itemID)
							err = mComponent.UpdateQuantityForStockItem(ctx, stockItemID, 75, true) // Pass context
							if err != nil { t.Logf("Failed to set stock for group component: %v", err) }
						}
					}
				}
			}
			componentProducts = append(componentProducts, mComponent)
			log.Info().Str("sku", mComponent.Product.Sku).Float64("price", mComponent.Product.Price).Msg("Group component created")
		}

		groupSku := fmt.Sprintf("grouped-test-%d", timestamp)
		groupedProductDef := magento2.Product{ // Renamed variable
			Sku: groupSku, Name: fmt.Sprintf("Grouped Test Product %d", timestamp),
			AttributeSetID: 4, Price: 0, TypeID: "grouped", Status: 1, Visibility: 4, Weight: 0,
		}
		mGroup, err := magento2.CreateOrReplaceProduct(ctx, &groupedProductDef, true, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create grouped product: %v", err)
			return
		}
		log.Info().Str("sku", mGroup.Product.Sku).Int("id", mGroup.Product.ID).Msg("Grouped product created")
		t.Logf("Created grouped product with %d associated products. Manual linking required in Magento admin.", len(componentProducts))
	})
}

func TestAdvancedProducts_AttributeOptions(t *testing.T) {
	client, _ := setupAdvancedTestClient(t)
	ctx := context.Background() // Define context

	t.Run("Create Attribute with Options", func(t *testing.T) {
		timestamp := time.Now().Unix()
		dropdownAttrDef := magento2.Attribute{ // Renamed variable
			AttributeCode: fmt.Sprintf("test_dropdown_%d", timestamp), FrontendInput: "select", DefaultFrontendLabel: "Test Dropdown",
			IsRequired: false, Scope: "global", EntityTypeID: "4", IsUserDefined: true,
		}
		createdAttr, err := magento2.CreateAttribute(ctx, &dropdownAttrDef, client) // Pass context
		if err != nil {
			t.Errorf("Failed to create dropdown attribute: %v", err)
			return
		}
		log.Info().Str("code", createdAttr.Attribute.AttributeCode).Int("id", createdAttr.Attribute.AttributeID).Msg("Dropdown attribute created")

		options := []magento2.Option{
			{Label: "Option 1", Value: "opt1", SortOrder: 1}, {Label: "Option 2", Value: "opt2", SortOrder: 2}, {Label: "Option 3", Value: "opt3", SortOrder: 3},
		}
		for i, option := range options {
			result, err := createdAttr.AddOption(ctx, option) // Pass context
			if err != nil {
				t.Errorf("Failed to add option %d (%s): %v", i, option.Label, err)
			} else {
				log.Info().Str("option", option.Label).Str("result", result).Msg("Option added to attribute")
			}
		}
	})
}

func TestAdvancedProducts_All(t *testing.T) {
	// Each sub-test function will define its own client and context.
	t.Run("ConfigurableProduct", TestAdvancedProducts_ConfigurableProduct)
	t.Run("BundleProduct", TestAdvancedProducts_BundleProduct)
	t.Run("VirtualProduct", TestAdvancedProducts_VirtualProduct)
	t.Run("GroupedProduct", TestAdvancedProducts_GroupedProduct)
	t.Run("AttributeOptions", TestAdvancedProducts_AttributeOptions)
}