# go-m2rest Real-World Examples and Use Cases

This document provides practical examples and use cases for integrating the go-m2rest library into real-world e-commerce scenarios.

## 🛍️ E-commerce Integration Use Cases

### 1. Inventory Management Systems

**Scenario**: A warehouse management system needs to sync stock levels with Magento in real-time.

**Implementation**:
```go
package main

import (
    "log"
    "github.com/florinel-chis/go-m2rest"
)

func syncInventoryFromWarehouse(client *magento2.Client, warehouseItems []WarehouseItem) {
    for _, item := range warehouseItems {
        // Get product from Magento
        product, err := magento2.GetProductBySKU(item.SKU, client)
        if err != nil {
            log.Printf("SKU %s not found: %v", item.SKU, err)
            continue
        }
        
        // Update stock quantity
        if stockItem, ok := product.Product.ExtensionAttributes["stock_item"].(map[string]any); ok {
            if itemID, ok := stockItem["item_id"]; ok {
                err = product.UpdateQuantityForStockItem(
                    fmt.Sprintf("%v", itemID), 
                    item.Quantity, 
                    item.Quantity > 0,
                )
                if err != nil {
                    log.Printf("Failed to update stock for %s: %v", item.SKU, err)
                }
            }
        }
    }
}
```

**Benefits**: 
- Real-time inventory accuracy
- Prevent overselling
- Automated stock management

### 2. Product Information Management (PIM)

**Scenario**: Central product database needs to push updates to multiple Magento stores.

**Implementation**:
```go
func syncProductsFromPIM(client *magento2.Client, pimProducts []PIMProduct) error {
    for _, pimProduct := range pimProducts {
        product := magento2.Product{
            Sku:            pimProduct.SKU,
            Name:           pimProduct.Name,
            Price:          pimProduct.Price,
            Description:    pimProduct.Description,
            AttributeSetID: 4,
            Status:         1,
            Visibility:     4,
            TypeID:         "simple",
            Weight:         pimProduct.Weight,
            CustomAttributes: []map[string]any{
                {"attribute_code": "brand", "value": pimProduct.Brand},
                {"attribute_code": "color", "value": pimProduct.Color},
                {"attribute_code": "size", "value": pimProduct.Size},
            },
        }
        
        _, err := magento2.CreateOrReplaceProduct(&product, true, client)
        if err != nil {
            return fmt.Errorf("failed to sync product %s: %w", pimProduct.SKU, err)
        }
    }
    return nil
}
```

### 3. Marketplace Integration

**Scenario**: Sync products between Magento and marketplaces like Amazon or eBay.

**Implementation**:
```go
func exportToMarketplace(client *magento2.Client, categoryID int) ([]MarketplaceListing, error) {
    // Get products from specific category
    products := []magento2.Product{}
    // Fetch products by category (pseudo-code)
    
    listings := []MarketplaceListing{}
    for _, product := range products {
        listing := MarketplaceListing{
            Title:       product.Name,
            Price:       product.Price,
            SKU:         product.Sku,
            Description: product.Description,
            Quantity:    getProductStock(product),
            Images:      getProductImages(product),
        }
        listings = append(listings, listing)
    }
    
    return listings, nil
}
```

### 4. B2B Customer Portals

**Scenario**: Custom B2B portal with customer-specific pricing and catalog visibility.

**Implementation**:
```go
func getB2BCustomerCatalog(client *magento2.Client, customerToken string) ([]magento2.Product, error) {
    // Create authenticated client for customer
    b2bClient := client.WithCustomerToken(customerToken)
    
    // Get customer-specific products with negotiated prices
    // This would show only products assigned to the customer's group
    products, err := getCustomerProducts(b2bClient)
    if err != nil {
        return nil, err
    }
    
    // Apply customer-specific pricing rules
    for i := range products {
        products[i].Price = applyCustomerPricing(products[i], customerToken)
    }
    
    return products, nil
}
```

## 📊 Analytics & Reporting

### 5. Business Intelligence Dashboards

**Scenario**: Extract sales data for executive dashboards and KPI tracking.

**Implementation**:
```go
func extractSalesMetrics(client *magento2.Client, startDate, endDate time.Time) (*SalesMetrics, error) {
    metrics := &SalesMetrics{
        Period: fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
    }
    
    // Get orders within date range
    orders := []magento2.Order{}
    // Implementation to fetch orders by date range
    
    for _, order := range orders {
        metrics.TotalRevenue += order.GrandTotal
        metrics.OrderCount++
        metrics.AverageOrderValue = metrics.TotalRevenue / float64(metrics.OrderCount)
        
        // Track product performance
        for _, item := range order.Items {
            metrics.ProductsSold[item.SKU] += item.QtyOrdered
        }
    }
    
    return metrics, nil
}
```

### 6. Inventory Forecasting

**Scenario**: Predict stock needs based on sales velocity and seasonal trends.

**Implementation**:
```go
func calculateReorderPoints(client *magento2.Client) (map[string]ReorderPoint, error) {
    reorderPoints := make(map[string]ReorderPoint)
    
    // Analyze 30-day sales history
    salesHistory := getSalesHistory(client, 30)
    
    for sku, dailySales := range salesHistory {
        product, err := magento2.GetProductBySKU(sku, client)
        if err != nil {
            continue
        }
        
        avgDailySales := calculateAverage(dailySales)
        leadTimeDays := 7 // Supplier lead time
        safetyStock := avgDailySales * 3 // 3 days safety stock
        
        reorderPoints[sku] = ReorderPoint{
            SKU:          sku,
            ReorderAt:    int(avgDailySales*float64(leadTimeDays) + safetyStock),
            ReorderQty:   int(avgDailySales * 30), // 30-day supply
            CurrentStock: getCurrentStock(product),
        }
    }
    
    return reorderPoints, nil
}
```

## 🤖 Automation Use Cases

### 7. Automated Product Launches

**Scenario**: Schedule new product releases with timed visibility changes.

**Implementation**:
```go
func scheduleProductLaunch(client *magento2.Client, launches []ProductLaunch) {
    for _, launch := range launches {
        go func(l ProductLaunch) {
            // Wait until launch time
            time.Sleep(time.Until(l.LaunchTime))
            
            // Update product visibility
            product, err := magento2.GetProductBySKU(l.SKU, client)
            if err != nil {
                log.Printf("Failed to get product %s: %v", l.SKU, err)
                return
            }
            
            product.Product.Status = 1 // Enable
            product.Product.Visibility = 4 // Catalog, Search
            
            // Add launch price if specified
            if l.LaunchPrice > 0 {
                product.Product.Price = l.LaunchPrice
            }
            
            _, err = magento2.CreateOrReplaceProduct(&product.Product, true, client)
            if err != nil {
                log.Printf("Failed to launch product %s: %v", l.SKU, err)
            } else {
                log.Printf("Successfully launched product %s", l.SKU)
            }
        }(launch)
    }
}
```

### 8. Dynamic Pricing Engine

**Scenario**: Adjust prices based on competition, demand, or inventory levels.

**Implementation**:
```go
func applyDynamicPricing(client *magento2.Client) error {
    products, err := getAllProducts(client)
    if err != nil {
        return err
    }
    
    for _, product := range products {
        currentStock := getCurrentStock(product)
        originalPrice := product.Price
        newPrice := originalPrice
        
        // Low stock premium
        if currentStock < 10 {
            newPrice = originalPrice * 1.15 // 15% increase
        } else if currentStock < 5 {
            newPrice = originalPrice * 1.25 // 25% increase
        }
        
        // High stock discount
        if currentStock > 100 {
            newPrice = originalPrice * 0.90 // 10% discount
        }
        
        // Competitor pricing check
        competitorPrice := getCompetitorPrice(product.Sku)
        if competitorPrice > 0 && newPrice > competitorPrice {
            newPrice = competitorPrice * 0.99 // Beat competitor by 1%
        }
        
        if newPrice != originalPrice {
            product.Price = newPrice
            _, err := magento2.CreateOrReplaceProduct(&product, true, client)
            if err != nil {
                log.Printf("Failed to update price for %s: %v", product.Sku, err)
            }
        }
    }
    
    return nil
}
```

### 9. Seasonal Catalog Management

**Scenario**: Automatically enable/disable seasonal products based on date.

**Implementation**:
```go
func manageSeasonalProducts(client *magento2.Client) {
    seasonalRules := []SeasonalRule{
        {
            Category:  "summer-collection",
            StartDate: time.Date(time.Now().Year(), 6, 1, 0, 0, 0, 0, time.UTC),
            EndDate:   time.Date(time.Now().Year(), 8, 31, 23, 59, 59, 0, time.UTC),
        },
        {
            Category:  "winter-collection",
            StartDate: time.Date(time.Now().Year(), 11, 1, 0, 0, 0, 0, time.UTC),
            EndDate:   time.Date(time.Now().Year()+1, 2, 28, 23, 59, 59, 0, time.UTC),
        },
    }
    
    now := time.Now()
    for _, rule := range seasonalRules {
        products := getProductsByCategory(client, rule.Category)
        
        for _, product := range products {
            shouldBeActive := now.After(rule.StartDate) && now.Before(rule.EndDate)
            
            if shouldBeActive && product.Status != 1 {
                product.Status = 1 // Enable
                product.Visibility = 4
                magento2.CreateOrReplaceProduct(&product, true, client)
            } else if !shouldBeActive && product.Status == 1 {
                product.Status = 2 // Disable
                product.Visibility = 1
                magento2.CreateOrReplaceProduct(&product, true, client)
            }
        }
    }
}
```

## 🔄 Integration Scenarios

### 10. ERP Synchronization

**Scenario**: Two-way sync between ERP system (SAP/Oracle) and Magento.

**Implementation**:
```go
func syncWithERP(client *magento2.Client, erpClient *ERPClient) error {
    // Sync products from ERP to Magento
    erpProducts, err := erpClient.GetProducts()
    if err != nil {
        return err
    }
    
    for _, erpProduct := range erpProducts {
        magentoProduct := magento2.Product{
            Sku:            erpProduct.ItemCode,
            Name:           erpProduct.Description,
            Price:          erpProduct.ListPrice,
            Cost:           erpProduct.StandardCost,
            Weight:         erpProduct.Weight,
            AttributeSetID: 4,
            Status:         1,
            TypeID:         "simple",
        }
        
        _, err := magento2.CreateOrReplaceProduct(&magentoProduct, true, client)
        if err != nil {
            log.Printf("Failed to sync product %s: %v", erpProduct.ItemCode, err)
        }
    }
    
    // Sync orders from Magento to ERP
    pendingOrders := getPendingOrders(client)
    for _, order := range pendingOrders {
        erpOrder := convertToERPOrder(order)
        err := erpClient.CreateSalesOrder(erpOrder)
        if err != nil {
            log.Printf("Failed to sync order %s: %v", order.IncrementID, err)
        } else {
            // Update order status in Magento
            order.State = "processing"
            updateOrder(client, order)
        }
    }
    
    return nil
}
```

### 11. CRM Integration

**Scenario**: Sync customer data and order history with CRM systems.

**Implementation**:
```go
func syncCustomersToCRM(client *magento2.Client, crmClient *CRMClient) error {
    // Get recent orders
    recentOrders := getRecentOrders(client, 24*time.Hour)
    
    for _, order := range recentOrders {
        // Create or update CRM contact
        contact := CRMContact{
            Email:     order.CustomerEmail,
            FirstName: order.CustomerFirstname,
            LastName:  order.CustomerLastname,
            Phone:     order.BillingAddress.Telephone,
            
            CustomFields: map[string]any{
                "last_order_date":   order.CreatedAt,
                "last_order_amount": order.GrandTotal,
                "total_spent":       calculateCustomerTotalSpent(client, order.CustomerEmail),
                "order_count":       getCustomerOrderCount(client, order.CustomerEmail),
            },
        }
        
        err := crmClient.UpsertContact(contact)
        if err != nil {
            log.Printf("Failed to sync customer %s: %v", contact.Email, err)
        }
        
        // Log order as CRM activity
        activity := CRMActivity{
            ContactEmail: order.CustomerEmail,
            Type:         "purchase",
            Description:  fmt.Sprintf("Order #%s - $%.2f", order.IncrementID, order.GrandTotal),
            Date:         order.CreatedAt,
        }
        crmClient.CreateActivity(activity)
    }
    
    return nil
}
```

### 12. Shipping & Fulfillment Integration

**Scenario**: Integrate with 3PL or shipping providers for order fulfillment.

**Implementation**:
```go
func processFulfillment(client *magento2.Client, fulfillmentClient *FulfillmentClient) {
    // Get orders ready for fulfillment
    readyOrders := getOrdersByStatus(client, "processing")
    
    for _, order := range readyOrders {
        // Create fulfillment request
        fulfillmentReq := FulfillmentRequest{
            OrderID:      order.IncrementID,
            CustomerName: fmt.Sprintf("%s %s", order.CustomerFirstname, order.CustomerLastname),
            ShipTo:       convertAddress(order.ShippingAddress),
            Items:        convertOrderItems(order.Items),
            ShipMethod:   order.ShippingMethod,
        }
        
        // Send to fulfillment center
        trackingInfo, err := fulfillmentClient.CreateShipment(fulfillmentReq)
        if err != nil {
            log.Printf("Failed to create shipment for order %s: %v", order.IncrementID, err)
            continue
        }
        
        // Update order with tracking information
        shipment := magento2.Shipment{
            OrderID: order.EntityID,
            Tracks: []magento2.Track{
                {
                    TrackNumber: trackingInfo.TrackingNumber,
                    Title:       trackingInfo.Carrier,
                    CarrierCode: trackingInfo.CarrierCode,
                },
            },
        }
        
        err = createShipment(client, shipment)
        if err != nil {
            log.Printf("Failed to update tracking for order %s: %v", order.IncrementID, err)
        }
    }
}
```

## 🛒 Customer Experience

### 13. Mobile App Backend

**Scenario**: Native mobile app using Magento as backend.

**Implementation**:
```go
// Mobile API wrapper
type MobileAPI struct {
    client *magento2.Client
}

func (api *MobileAPI) GetFeaturedProducts() ([]MobileProduct, error) {
    // Get featured products from specific category
    products := getProductsByCategory(api.client, "featured")
    
    mobileProducts := []MobileProduct{}
    for _, p := range products {
        mobileProducts = append(mobileProducts, MobileProduct{
            ID:          p.ID,
            SKU:         p.Sku,
            Name:        p.Name,
            Price:       p.Price,
            Image:       getPrimaryImage(p),
            InStock:     isInStock(p),
            Rating:      getProductRating(p),
            ReviewCount: getReviewCount(p),
        })
    }
    
    return mobileProducts, nil
}

func (api *MobileAPI) AddToCart(customerToken string, sku string, qty int) error {
    // Create customer cart
    cart, err := magento2.NewCustomerCartFromAPIClient(api.client, customerToken)
    if err != nil {
        return err
    }
    
    // Add item
    item := magento2.CartItem{
        Sku:     sku,
        Qty:     qty,
        QuoteID: cart.QuoteID,
    }
    
    return cart.AddItems([]magento2.CartItem{item})
}
```

### 14. Progressive Web App (PWA) Backend

**Scenario**: Headless commerce with modern frontend framework.

**Implementation**:
```go
// GraphQL-like resolver for PWA
func resolveCategoryPage(client *magento2.Client, categoryID int, filters map[string]any) (*CategoryPageData, error) {
    // Get category info
    category, err := magento2.GetCategoryByID(categoryID, client)
    if err != nil {
        return nil, err
    }
    
    // Get products with filters
    products := getFilteredProducts(client, categoryID, filters)
    
    // Get facets for filtering
    facets := getFacets(client, categoryID)
    
    return &CategoryPageData{
        Category: category,
        Products: products,
        Facets:   facets,
        Total:    len(products),
    }, nil
}
```

### 15. Kiosk/POS Systems

**Scenario**: In-store kiosks or point-of-sale systems connected to Magento.

**Implementation**:
```go
type POSSystem struct {
    client   *magento2.Client
    storeID  string
    terminal string
}

func (pos *POSSystem) ProcessSale(items []POSItem, payment PaymentInfo) (*Receipt, error) {
    // Create order
    order := buildPOSOrder(items, payment, pos.storeID, pos.terminal)
    
    // Check inventory
    for _, item := range items {
        available, err := checkLocalInventory(pos.client, item.SKU, pos.storeID)
        if err != nil || available < item.Quantity {
            return nil, fmt.Errorf("insufficient inventory for %s", item.SKU)
        }
    }
    
    // Process payment
    paymentResult, err := processPayment(payment)
    if err != nil {
        return nil, err
    }
    
    // Create order in Magento
    magentoOrder, err := createPOSOrder(pos.client, order, paymentResult)
    if err != nil {
        // Reverse payment if order creation fails
        reversePayment(paymentResult)
        return nil, err
    }
    
    // Update local inventory
    updateLocalInventory(pos.client, items, pos.storeID)
    
    return generateReceipt(magentoOrder), nil
}
```

## 📈 Marketing & Sales

### 16. Bulk Promotional Updates

**Scenario**: Apply sale prices to hundreds of products for promotional campaigns.

**Implementation**:
```go
func applyPromotion(client *magento2.Client, promo Promotion) error {
    affectedProducts := []string{}
    
    // Get products by criteria
    var products []magento2.Product
    switch promo.Type {
    case "category":
        products = getProductsByCategory(client, promo.CategoryID)
    case "brand":
        products = getProductsByAttribute(client, "brand", promo.BrandName)
    case "sku_list":
        products = getProductsBySKUs(client, promo.SKUs)
    }
    
    // Apply promotion
    for _, product := range products {
        originalPrice := product.Price
        
        switch promo.DiscountType {
        case "percentage":
            product.SpecialPrice = originalPrice * (1 - promo.DiscountValue/100)
        case "fixed":
            product.SpecialPrice = originalPrice - promo.DiscountValue
        }
        
        product.SpecialPriceFrom = promo.StartDate
        product.SpecialPriceTo = promo.EndDate
        
        _, err := magento2.CreateOrReplaceProduct(&product, true, client)
        if err != nil {
            log.Printf("Failed to apply promo to %s: %v", product.Sku, err)
        } else {
            affectedProducts = append(affectedProducts, product.Sku)
        }
    }
    
    log.Printf("Applied promotion to %d products", len(affectedProducts))
    return nil
}
```

### 17. Product Bundle Creator

**Scenario**: Dynamically create product bundles based on buying patterns.

**Implementation**:
```go
func createAutoBundles(client *magento2.Client) error {
    // Analyze frequently bought together
    patterns := analyzeBuyingPatterns(client, 30) // Last 30 days
    
    for _, pattern := range patterns {
        if pattern.Frequency < 10 {
            continue // Skip infrequent combinations
        }
        
        // Create bundle product
        bundleSKU := fmt.Sprintf("auto-bundle-%s", generateBundleID(pattern.Products))
        
        bundle := magento2.Product{
            Sku:            bundleSKU,
            Name:           generateBundleName(pattern.Products),
            TypeID:         "bundle",
            Status:         1,
            Visibility:     4,
            AttributeSetID: 4,
            Price:          calculateBundlePrice(pattern.Products, 0.9), // 10% discount
        }
        
        mBundle, err := magento2.CreateOrReplaceProduct(&bundle, true, client)
        if err != nil {
            log.Printf("Failed to create bundle: %v", err)
            continue
        }
        
        // Add bundle options (requires additional API calls)
        addBundleOptions(client, mBundle.Product.ID, pattern.Products)
    }
    
    return nil
}
```

### 18. Affiliate/Dropship Management

**Scenario**: Manage products from multiple suppliers with different commission rates.

**Implementation**:
```go
type DropshipManager struct {
    client    *magento2.Client
    suppliers map[string]Supplier
}

func (dm *DropshipManager) ImportSupplierCatalog(supplierID string) error {
    supplier := dm.suppliers[supplierID]
    products, err := supplier.GetCatalog()
    if err != nil {
        return err
    }
    
    for _, sp := range products {
        // Calculate markup
        ourPrice := sp.WholesalePrice * (1 + supplier.MarkupPercent/100)
        
        product := magento2.Product{
            Sku:            fmt.Sprintf("%s-%s", supplier.Prefix, sp.SKU),
            Name:           sp.Name,
            Price:          ourPrice,
            Cost:           sp.WholesalePrice,
            Weight:         sp.Weight,
            Description:    sp.Description,
            AttributeSetID: 4,
            Status:         1,
            TypeID:         "simple",
            CustomAttributes: []map[string]any{
                {"attribute_code": "supplier_id", "value": supplierID},
                {"attribute_code": "supplier_sku", "value": sp.SKU},
                {"attribute_code": "commission_rate", "value": supplier.CommissionRate},
            },
        }
        
        _, err := magento2.CreateOrReplaceProduct(&product, true, dm.client)
        if err != nil {
            log.Printf("Failed to import %s: %v", sp.SKU, err)
        }
    }
    
    return nil
}
```

## 🔧 Operations & Maintenance

### 19. Bulk Product Enrichment

**Scenario**: Add missing attributes or improve product descriptions using AI.

**Implementation**:
```go
func enrichProductDescriptions(client *magento2.Client, aiClient *AIClient) error {
    // Get products with poor descriptions
    products := getProductsNeedingEnrichment(client)
    
    for _, product := range products {
        // Generate enhanced description using AI
        enhanced, err := aiClient.GenerateProductDescription(
            product.Name,
            product.ShortDescription,
            product.CustomAttributes,
        )
        if err != nil {
            log.Printf("Failed to generate description for %s: %v", product.Sku, err)
            continue
        }
        
        // Update product
        product.Description = enhanced.LongDescription
        product.ShortDescription = enhanced.ShortDescription
        product.MetaDescription = enhanced.MetaDescription
        product.MetaKeywords = enhanced.Keywords
        
        // Add missing attributes
        for _, attr := range enhanced.SuggestedAttributes {
            product.CustomAttributes = append(product.CustomAttributes, map[string]any{
                "attribute_code": attr.Code,
                "value":          attr.Value,
            })
        }
        
        _, err = magento2.CreateOrReplaceProduct(&product, true, client)
        if err != nil {
            log.Printf("Failed to update %s: %v", product.Sku, err)
        }
    }
    
    return nil
}
```

### 20. Catalog Cleanup

**Scenario**: Remove discontinued products and clean up obsolete data.

**Implementation**:
```go
func cleanupCatalog(client *magento2.Client) error {
    // Find products to clean up
    candidatesForRemoval := []magento2.Product{}
    
    allProducts := getAllProducts(client)
    for _, product := range allProducts {
        // Check various criteria
        shouldRemove := false
        
        // No sales in 12 months
        if lastSaleDate := getLastSaleDate(product.Sku); time.Since(lastSaleDate) > 365*24*time.Hour {
            shouldRemove = true
        }
        
        // Out of stock for 6 months
        if stockHistory := getStockHistory(product.Sku); hasBeenOutOfStock(stockHistory, 180) {
            shouldRemove = true
        }
        
        // Discontinued by manufacturer
        if isDiscontinued(product) {
            shouldRemove = true
        }
        
        if shouldRemove {
            candidatesForRemoval = append(candidatesForRemoval, product)
        }
    }
    
    // Archive before removing
    archiveProducts(candidatesForRemoval)
    
    // Disable products (safer than deleting)
    for _, product := range candidatesForRemoval {
        product.Status = 2 // Disabled
        product.Visibility = 1 // Not visible
        _, err := magento2.CreateOrReplaceProduct(&product, true, client)
        if err != nil {
            log.Printf("Failed to disable %s: %v", product.Sku, err)
        }
    }
    
    log.Printf("Cleaned up %d products", len(candidatesForRemoval))
    return nil
}
```

### 21. Multi-Store Management

**Scenario**: Manage products across multiple store views with different currencies and languages.

**Implementation**:
```go
type MultiStoreManager struct {
    stores map[string]*magento2.Client // Store code -> Client
}

func (msm *MultiStoreManager) ReplicateProduct(masterSKU string, targetStores []string) error {
    // Get product from master store
    masterClient := msm.stores["default"]
    masterProduct, err := magento2.GetProductBySKU(masterSKU, masterClient)
    if err != nil {
        return err
    }
    
    // Replicate to target stores
    for _, storeCode := range targetStores {
        storeClient := msm.stores[storeCode]
        storeProduct := masterProduct.Product
        
        // Adjust for store-specific settings
        switch storeCode {
        case "uk_store":
            storeProduct.Price = convertCurrency(storeProduct.Price, "USD", "GBP")
            storeProduct.Name = localizeProductName(storeProduct.Name, "en_GB")
        case "de_store":
            storeProduct.Price = convertCurrency(storeProduct.Price, "USD", "EUR")
            storeProduct.Name = localizeProductName(storeProduct.Name, "de_DE")
            storeProduct.Description = translateDescription(storeProduct.Description, "de_DE")
        }
        
        _, err := magento2.CreateOrReplaceProduct(&storeProduct, true, storeClient)
        if err != nil {
            log.Printf("Failed to replicate to %s: %v", storeCode, err)
        }
    }
    
    return nil
}
```

## 🎯 Specialized Use Cases

### 22. Subscription Commerce

**Scenario**: Handle recurring orders for subscription-based products.

**Implementation**:
```go
type SubscriptionManager struct {
    client *magento2.Client
}

func (sm *SubscriptionManager) ProcessSubscriptions() error {
    // Get active subscriptions due for renewal
    dueSubscriptions := getSubscriptionsDueToday()
    
    for _, sub := range dueSubscriptions {
        // Create order from subscription
        order := magento2.Order{
            CustomerEmail:     sub.CustomerEmail,
            CustomerFirstname: sub.CustomerFirstname,
            CustomerLastname:  sub.CustomerLastname,
            Items: []magento2.OrderItem{
                {
                    Sku:   sub.ProductSKU,
                    Qty:   sub.Quantity,
                    Price: sub.Price,
                },
            },
            BillingAddress:  sub.BillingAddress,
            ShippingAddress: sub.ShippingAddress,
            PaymentMethod:   sub.PaymentMethod,
        }
        
        // Process payment
        paymentResult, err := processSubscriptionPayment(sub)
        if err != nil {
            // Handle failed payment
            notifyCustomerPaymentFailed(sub)
            updateSubscriptionStatus(sub, "payment_failed")
            continue
        }
        
        // Create order
        magentoOrder, err := createSubscriptionOrder(sm.client, order, paymentResult)
        if err != nil {
            log.Printf("Failed to create order for subscription %s: %v", sub.ID, err)
            continue
        }
        
        // Update next renewal date
        sub.NextRenewalDate = sub.NextRenewalDate.AddDate(0, 0, sub.FrequencyDays)
        updateSubscription(sub)
        
        // Send confirmation
        sendSubscriptionOrderConfirmation(sub, magentoOrder)
    }
    
    return nil
}
```

### 23. Custom Product Configurators

**Scenario**: Complex product customization for made-to-order items.

**Implementation**:
```go
func createCustomizedProduct(client *magento2.Client, config ProductConfiguration) (*magento2.Product, error) {
    // Base product
    baseSKU := fmt.Sprintf("custom-%s-%d", config.BaseProduct, time.Now().Unix())
    
    product := magento2.Product{
        Sku:            baseSKU,
        Name:           fmt.Sprintf("Custom %s", config.ProductName),
        TypeID:         "simple",
        Price:          calculateCustomPrice(config),
        Status:         1,
        Visibility:     1, // Not visible in catalog
        AttributeSetID: 4,
        CustomAttributes: []map[string]any{
            {"attribute_code": "is_custom", "value": "1"},
            {"attribute_code": "base_product", "value": config.BaseProduct},
        },
    }
    
    // Add customization options
    for _, option := range config.Options {
        product.CustomAttributes = append(product.CustomAttributes, map[string]any{
            "attribute_code": fmt.Sprintf("custom_%s", option.Name),
            "value":          option.Value,
        })
        
        // Add to product name for clarity
        product.Name += fmt.Sprintf(" - %s: %s", option.Name, option.Value)
    }
    
    // Add custom options for reordering
    product.Options = []magento2.ProductOption{}
    for _, option := range config.Options {
        product.Options = append(product.Options, magento2.ProductOption{
            Title:    option.Name,
            Type:     "field",
            Required: true,
            Price:    option.PriceModifier,
        })
    }
    
    mProduct, err := magento2.CreateOrReplaceProduct(&product, true, client)
    if err != nil {
        return nil, err
    }
    
    return &mProduct.Product, nil
}
```

### 24. Digital Asset Management

**Scenario**: Sync product images and videos from a DAM system.

**Implementation**:
```go
func syncProductMedia(client *magento2.Client, damClient *DAMClient) error {
    // Get products needing media updates
    products := getProductsNeedingMedia(client)
    
    for _, product := range products {
        // Get media from DAM
        assets, err := damClient.GetProductAssets(product.Sku)
        if err != nil {
            log.Printf("Failed to get assets for %s: %v", product.Sku, err)
            continue
        }
        
        // Clear existing media
        product.MediaGalleryEntries = []magento2.MediaGalleryEntry{}
        
        // Add new media
        for i, asset := range assets {
            mediaEntry := magento2.MediaGalleryEntry{
                MediaType: asset.Type, // image or video
                Label:     asset.Label,
                Position:  i,
                Disabled:  false,
                File:      asset.URL,
            }
            
            // Set first image as main
            if i == 0 && asset.Type == "image" {
                mediaEntry.Types = []string{"image", "small_image", "thumbnail"}
            }
            
            product.MediaGalleryEntries = append(product.MediaGalleryEntries, mediaEntry)
        }
        
        _, err = magento2.CreateOrReplaceProduct(&product, true, client)
        if err != nil {
            log.Printf("Failed to update media for %s: %v", product.Sku, err)
        }
    }
    
    return nil
}
```

### 25. Compliance & Regulations

**Scenario**: Update products for regulatory compliance (GDPR, California Prop 65, etc.).

**Implementation**:
```go
func applyComplianceUpdates(client *magento2.Client, regulations []Regulation) error {
    for _, reg := range regulations {
        affectedProducts := getProductsByRegulation(client, reg)
        
        for _, product := range affectedProducts {
            switch reg.Type {
            case "prop65":
                // Add California Prop 65 warning
                product.CustomAttributes = append(product.CustomAttributes, map[string]any{
                    "attribute_code": "prop65_warning",
                    "value":          reg.WarningText,
                })
                
            case "gdpr":
                // Update privacy-related attributes
                product.CustomAttributes = append(product.CustomAttributes, map[string]any{
                    "attribute_code": "data_retention_period",
                    "value":          reg.DataRetentionDays,
                })
                
            case "energy_label":
                // Add energy efficiency information
                product.CustomAttributes = append(product.CustomAttributes, map[string]any{
                    "attribute_code": "energy_class",
                    "value":          calculateEnergyClass(product),
                })
            }
            
            // Add compliance badge to description
            product.Description += fmt.Sprintf("\n\n%s", reg.ComplianceBadgeHTML)
            
            _, err := magento2.CreateOrReplaceProduct(&product, true, client)
            if err != nil {
                log.Printf("Failed to update compliance for %s: %v", product.Sku, err)
            }
        }
    }
    
    return nil
}
```

## 💡 Complete Implementation Example

Here's a comprehensive example that combines multiple use cases into a daily automation routine:

```go
package main

import (
    "log"
    "time"
    "github.com/florinel-chis/go-m2rest"
)

func main() {
    // Initialize client
    storeConfig := &magento2.StoreConfig{
        Scheme:    "https",
        HostName:  "magento.local",
        StoreCode: "default",
    }
    
    client, err := magento2.NewAPIClientFromIntegration(
        storeConfig,
        "your_integration_token",
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Run daily automation
    runDailyAutomation(client)
}

func runDailyAutomation(client *magento2.Client) {
    log.Println("Starting daily automation...")
    
    // 1. Morning: Inventory sync
    log.Println("Syncing inventory...")
    syncInventoryFromWarehouse(client, getWarehouseInventory())
    
    // 2. Price adjustments
    log.Println("Applying dynamic pricing...")
    applyDynamicPricing(client)
    
    // 3. Launch scheduled products
    log.Println("Launching scheduled products...")
    launchScheduledProducts(client)
    
    // 4. Process subscriptions
    log.Println("Processing subscriptions...")
    processSubscriptions(client)
    
    // 5. Sync with external systems
    log.Println("Syncing with ERP...")
    syncWithERP(client, getERPClient())
    
    log.Println("Syncing with CRM...")
    syncCustomersToCRM(client, getCRMClient())
    
    // 6. Cleanup tasks
    log.Println("Cleaning expired promotions...")
    cleanupExpiredPromotions(client)
    
    // 7. Generate reports
    log.Println("Generating daily reports...")
    generateDailyReports(client)
    
    log.Println("Daily automation completed!")
}

// Helper function to handle errors consistently
func handleError(operation string, err error) {
    if err != nil {
        log.Printf("Error in %s: %v", operation, err)
        // Send alert to monitoring system
        sendAlert(operation, err)
    }
}
```

## Summary

These examples demonstrate the versatility of the go-m2rest library in real-world e-commerce scenarios. From simple inventory updates to complex multi-system integrations, the library provides the foundation for building robust, scalable solutions that enhance Magento 2's capabilities.

Key benefits across all use cases:
- **Automation**: Reduce manual work and human errors
- **Integration**: Connect Magento with other business systems
- **Scalability**: Handle large catalogs and high transaction volumes
- **Flexibility**: Adapt to changing business requirements
- **Performance**: Leverage Go's concurrency for efficient operations

For more detailed API documentation, refer to the [GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest).