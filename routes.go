package magento2

// Route is one REST route this package calls: the HTTP method, Magento's
// path template relative to /rest (parameters in braces, as in
// etc/webapi.xml with ":name" written "{name}"), and Type, a pointer to the
// zero value of the Go type its response decodes into (nil when the
// response is not decoded). Definition optionally pins the schema
// definition Type corresponds to; when empty, tools derive it from the
// route's 200 response in /rest/all/schema.
type Route struct {
	Method     string
	Template   string
	Type       any
	Definition string
}

// Routes returns a copy of the route table, for tools such as the drift
// check (cmd/m2drift) that compare the client with a store's schema.
func Routes() []Route {
	out := make([]Route, len(routes))
	copy(out, routes)
	return out
}

// routes lists every route a function of this package calls. Every
// template is cited from the Magento 2.4.8 module-*/etc/webapi.xml that
// declares it.
var routes = []Route{
	// module-catalog/etc/webapi.xml
	{Method: "GET", Template: "/V1/products", Type: &ProductListResponse{}},                                    // :30 GetProductsPage, IterateProducts
	{Method: "POST", Template: "/V1/products", Type: &Product{}},                                               // :12 CreateOrReplaceProduct
	{Method: "GET", Template: "/V1/products/{sku}", Type: &Product{}},                                          // :36 GetProductBySKU
	{Method: "GET", Template: "/V1/products/attributes", Type: &AttributeListResponse{}},                       // :60 GetAttributesPage, IterateAttributes
	{Method: "POST", Template: "/V1/products/attributes", Type: &Attribute{}},                                  // :84 CreateAttribute
	{Method: "GET", Template: "/V1/products/attributes/{attributeCode}", Type: &Attribute{}},                   // :48 GetAttributeByAttributeCode
	{Method: "PUT", Template: "/V1/products/attributes/{attributeCode}", Type: &Attribute{}},                   // :90 MAttribute.UpdateAttributeOnRemote
	{Method: "POST", Template: "/V1/products/attributes/{attributeCode}/options", Type: new(string)},           // :192 MAttribute.AddOption
	{Method: "GET", Template: "/V1/products/attribute-sets/sets/list", Type: &attributeSetListResponse{}},      // :114 GetAttributeSetsList, GetAttributeSetByName
	{Method: "POST", Template: "/V1/products/attribute-sets", Type: &AttributeSet{}},                           // :132 CreateAttributeSet
	{Method: "GET", Template: "/V1/products/attribute-sets/{attributeSetId}", Type: &AttributeSet{}},           // :120 MAttributeSet.UpdateAttributeSetFromRemote
	{Method: "PUT", Template: "/V1/products/attribute-sets/{attributeSetId}", Type: &AttributeSet{}},           // :138 MAttributeSet.UpdateAttributeSetOnRemote
	{Method: "GET", Template: "/V1/products/attribute-sets/{attributeSetId}/attributes", Type: &[]Attribute{}}, // :144 GetAttributeSetAttributes
	{Method: "POST", Template: "/V1/products/attribute-sets/attributes", Type: new(int)},                       // :150 MAttributeSet.AssignAttribute
	{Method: "GET", Template: "/V1/products/attribute-sets/groups/list", Type: &groupSearchQueryResponse{}},    // :162 MAttributeSet groups
	{Method: "POST", Template: "/V1/products/attribute-sets/groups", Type: &Group{}},                           // :168 MAttributeSet.CreateGroup
	{Method: "GET", Template: "/V1/categories", Type: &CategoryTreeNode{}},                                     // :357 GetCategoryTree
	{Method: "POST", Template: "/V1/categories", Type: &Category{}},                                            // :351 CreateCategory
	{Method: "GET", Template: "/V1/categories/list", Type: &categorySearchQueryResponse{}},                     // :375 GetCategoryByName
	{Method: "GET", Template: "/V1/categories/{categoryId}", Type: &Category{}},                                // :345 MCategory.UpdateCategoryFromRemote
	{Method: "GET", Template: "/V1/categories/{categoryId}/products", Type: &[]ProductLink{}},                  // :459 MCategory.UpdateCategoryProductsFromRemote
	{Method: "PUT", Template: "/V1/categories/{categoryId}/products", Type: new(bool)},                         // :471 MCategory.AssignProductByProductLink

	// module-catalog-inventory/etc/webapi.xml
	{Method: "PUT", Template: "/V1/products/{productSku}/stockItems/{itemId}", Type: new(int)}, // :16 MProduct.UpdateQuantityForStockItem
	{Method: "GET", Template: routeStockItem, Type: &StockItem{}},                              // :10 GetStockItem
	{Method: "GET", Template: routeLowStockItems, Type: &StockItemListResponse{}},              // :22 GetLowStockItems

	// module-configurable-product/etc/webapi.xml
	{Method: "POST", Template: "/V1/configurable-products/{sku}/options", Type: new(int)},       // :46 SetOptionForExistingConfigurableProduct
	{Method: "GET", Template: "/V1/configurable-products/{sku}/options/all", Type: &[]Option{}}, // :40 MConfigurableProduct.UpdateOptionsFromRemote
	{Method: "PUT", Template: "/V1/configurable-products/{sku}/options/{id}", Type: new(int)},   // :52 MConfigurableProduct.UpdateOptionByID
	{Method: "POST", Template: "/V1/configurable-products/{sku}/child", Type: new(bool)},        // :28 MConfigurableProduct.AddChildBySKU

	// module-quote/etc/webapi.xml (cart routes; see GetCartsPage on per-cart GETs)
	{Method: "GET", Template: routeCartsSearch, Type: &CartListResponse{}},                               // :19 GetCartsPage, IterateCarts
	{Method: "POST", Template: "/V1/guest-carts", Type: new(string)},                                     // :91 NewGuestCartFromAPIClient
	{Method: "GET", Template: "/V1/guest-carts/{cartId}", Type: &Cart{}},                                 // :84 MCart.UpdateFromRemote
	{Method: "POST", Template: "/V1/guest-carts/{cartId}/items", Type: &CartItem{}},                      // :209 MCart.AddItems
	{Method: "DELETE", Template: "/V1/guest-carts/{cartId}/items/{itemId}", Type: new(bool)},             // :221 MCart.DeleteItem
	{Method: "POST", Template: "/V1/guest-carts/{cartId}/estimate-shipping-methods", Type: &[]Carrier{}}, // :169 MCart.EstimateShippingCarrier
	{Method: "GET", Template: "/V1/guest-carts/{cartId}/payment-methods", Type: &[]PaymentMethod{}},      // :299 MCart.EstimatePaymentMethods
	{Method: "PUT", Template: "/V1/guest-carts/{cartId}/order", Type: new(string)},                       // :106 MCart.CreateOrder
	{Method: "POST", Template: "/V1/carts/mine", Type: new(int)},                                         // :45 NewCustomerCartFromAPIClient
	{Method: "GET", Template: "/V1/carts/mine", Type: &Cart{}},                                           // :54 MCart.UpdateFromRemote
	{Method: "POST", Template: "/V1/carts/mine/items", Type: &CartItem{}},                                // :238 MCart.AddItems
	{Method: "DELETE", Template: "/V1/carts/mine/items/{itemId}", Type: new(bool)},                       // :256 MCart.DeleteItem
	{Method: "POST", Template: "/V1/carts/mine/estimate-shipping-methods", Type: &[]Carrier{}},           // :143 MCart.EstimateShippingCarrier
	{Method: "GET", Template: "/V1/carts/mine/payment-methods", Type: &[]PaymentMethod{}},                // :325 MCart.EstimatePaymentMethods
	{Method: "PUT", Template: "/V1/carts/mine/order", Type: new(int)},                                    // :72 MCart.CreateOrder

	// module-checkout/etc/webapi.xml
	{Method: "POST", Template: "/V1/guest-carts/{cartId}/shipping-information", Type: nil}, // :12 MCart.AddShippingInformation
	{Method: "POST", Template: "/V1/carts/mine/shipping-information", Type: nil},           // :20 MCart.AddShippingInformation

	// module-sales/etc/webapi.xml
	{Method: "GET", Template: routeOrders, Type: &OrderListResponse{}},             // :16 GetOrdersPage, IterateOrders, GetOrderByIncrementID
	{Method: "GET", Template: routeOrder, Type: &Order{}},                          // :10 GetOrder, MOrder.UpdateFromRemote
	{Method: "POST", Template: routeOrders, Type: &Order{}},                        // :256 MOrder.UpdateEntity
	{Method: "POST", Template: "/V1/orders/{id}/comments", Type: &StatusHistory{}}, // :52 MOrder.AddComment
	{Method: "GET", Template: routeInvoices, Type: &InvoiceListResponse{}},         // :94 GetInvoicesPage, IterateInvoices
	{Method: "GET", Template: routeInvoice, Type: &Invoice{}},                      // :88 GetInvoice
	{Method: "GET", Template: routeCreditMemos, Type: &CreditMemoListResponse{}},   // :148 GetCreditMemosPage, IterateCreditMemos
	{Method: "GET", Template: routeCreditMemo, Type: &CreditMemo{}},                // :154 GetCreditMemo
	{Method: "GET", Template: routeShipments, Type: &ShipmentListResponse{}},       // :202 GetShipmentsPage, IterateShipments
	{Method: "GET", Template: routeShipment, Type: &Shipment{}},                    // :196 GetShipment

	// module-customer/etc/webapi.xml
	{Method: "GET", Template: routeCustomersSearch, Type: &CustomerListResponse{}}, // :167 GetCustomersPage, IterateCustomers
	{Method: "GET", Template: routeCustomer, Type: &Customer{}},                    // :119 GetCustomer

	// module-inventory-api/etc/webapi.xml, module-inventory-sales-api/etc/webapi.xml
	{Method: "GET", Template: routeSourceItems, Type: &SourceItemListResponse{}}, // inventory-api :92 GetSourceItemsPage, IterateSourceItems
	{Method: "GET", Template: routeSources, Type: &SourceListResponse{}},         // inventory-api :11 GetSourcesPage, IterateSources
	{Method: "GET", Template: routeStocks, Type: &StockListResponse{}},           // inventory-api :42 GetStocksPage, IterateStocks
	{Method: "GET", Template: routeSalableQty, Type: new(float64)},               // inventory-sales-api :10 GetSalableQuantity

	// module-store/etc/webapi.xml
	{Method: "GET", Template: "/V1/store/storeViews", Type: &[]StoreView{}},   // :11 GetStoreViews
	{Method: "GET", Template: "/V1/store/websites", Type: &[]Website{}},       // :27 GetWebsites
	{Method: "GET", Template: routeStoreConfigs, Type: &[]StoreConfigEntry{}}, // :35 GetStoreConfigs
	{Method: "GET", Template: routeStoreGroups, Type: &[]StoreGroup{}},        // :19 GetStoreGroups

	// module-integration/etc/webapi.xml
	{Method: "POST", Template: "/V1/integration/admin/token", Type: new(string)},    // :10 NewAPIClientFromAuthentication
	{Method: "POST", Template: "/V1/integration/customer/token", Type: new(string)}, // :16 NewAPIClientFromAuthentication

	// module-webapi Controller/Rest/SchemaRequestProcessor (not in webapi.xml)
	{Method: "GET", Template: routeSchema, Type: &Schema{}}, // GetSchema (store code "all")
}
