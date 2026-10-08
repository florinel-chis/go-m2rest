// Types derived from the Magento 2.4.8 REST schema definitions; every json tag
// was checked against the constants and getters of the Api/Data interface
// cited on the type. Fields carry no omitempty: a 0 is data.

package magento2

import "encoding/json"

// Invoice is an invoice (GET /V1/invoices/{id}).
// Fields: module-sales/Api/Data/InvoiceInterface.php (Magento 2.4.8), schema definition sales-data-invoice-interface.
type Invoice struct {
	BaseCurrencyCode                        string           `json:"base_currency_code"`
	BaseDiscountAmount                      float64          `json:"base_discount_amount"`
	BaseGrandTotal                          float64          `json:"base_grand_total"`
	BaseDiscountTaxCompensationAmount       float64          `json:"base_discount_tax_compensation_amount"`
	BaseShippingAmount                      float64          `json:"base_shipping_amount"`
	BaseShippingDiscountTaxCompensationAmnt float64          `json:"base_shipping_discount_tax_compensation_amnt"`
	BaseShippingInclTax                     float64          `json:"base_shipping_incl_tax"`
	BaseShippingTaxAmount                   float64          `json:"base_shipping_tax_amount"`
	BaseSubtotal                            float64          `json:"base_subtotal"`
	BaseSubtotalInclTax                     float64          `json:"base_subtotal_incl_tax"`
	BaseTaxAmount                           float64          `json:"base_tax_amount"`
	BaseTotalRefunded                       float64          `json:"base_total_refunded"`
	BaseToGlobalRate                        float64          `json:"base_to_global_rate"`
	BaseToOrderRate                         float64          `json:"base_to_order_rate"`
	BillingAddressID                        int              `json:"billing_address_id"`
	CanVoidFlag                             int              `json:"can_void_flag"`
	CreatedAt                               string           `json:"created_at"`
	DiscountAmount                          float64          `json:"discount_amount"`
	DiscountDescription                     string           `json:"discount_description"`
	EmailSent                               int              `json:"email_sent"`
	EntityID                                int              `json:"entity_id"`
	GlobalCurrencyCode                      string           `json:"global_currency_code"`
	GrandTotal                              float64          `json:"grand_total"`
	DiscountTaxCompensationAmount           float64          `json:"discount_tax_compensation_amount"`
	IncrementID                             string           `json:"increment_id"`
	IsUsedForRefund                         int              `json:"is_used_for_refund"`
	OrderCurrencyCode                       string           `json:"order_currency_code"`
	OrderID                                 int              `json:"order_id"`
	ShippingAddressID                       int              `json:"shipping_address_id"`
	ShippingAmount                          float64          `json:"shipping_amount"`
	ShippingDiscountTaxCompensationAmount   float64          `json:"shipping_discount_tax_compensation_amount"`
	ShippingInclTax                         float64          `json:"shipping_incl_tax"`
	ShippingTaxAmount                       float64          `json:"shipping_tax_amount"`
	State                                   int              `json:"state"`
	StoreCurrencyCode                       string           `json:"store_currency_code"`
	StoreID                                 int              `json:"store_id"`
	StoreToBaseRate                         float64          `json:"store_to_base_rate"`
	StoreToOrderRate                        float64          `json:"store_to_order_rate"`
	Subtotal                                float64          `json:"subtotal"`
	SubtotalInclTax                         float64          `json:"subtotal_incl_tax"`
	TaxAmount                               float64          `json:"tax_amount"`
	TotalQty                                float64          `json:"total_qty"`
	TransactionID                           string           `json:"transaction_id"`
	UpdatedAt                               string           `json:"updated_at"`
	Items                                   []InvoiceItem    `json:"items"`
	Comments                                []InvoiceComment `json:"comments"`
	ExtensionAttributes                     json.RawMessage  `json:"extension_attributes"`
}

// InvoiceItem is an invoice line.
// Fields: module-sales/Api/Data/InvoiceItemInterface.php (Magento 2.4.8), schema definition sales-data-invoice-item-interface.
type InvoiceItem struct {
	AdditionalData                    string          `json:"additional_data"`
	BaseCost                          float64         `json:"base_cost"`
	BaseDiscountAmount                float64         `json:"base_discount_amount"`
	BaseDiscountTaxCompensationAmount float64         `json:"base_discount_tax_compensation_amount"`
	BasePrice                         float64         `json:"base_price"`
	BasePriceInclTax                  float64         `json:"base_price_incl_tax"`
	BaseRowTotal                      float64         `json:"base_row_total"`
	BaseRowTotalInclTax               float64         `json:"base_row_total_incl_tax"`
	BaseTaxAmount                     float64         `json:"base_tax_amount"`
	Description                       string          `json:"description"`
	DiscountAmount                    float64         `json:"discount_amount"`
	EntityID                          int             `json:"entity_id"`
	DiscountTaxCompensationAmount     float64         `json:"discount_tax_compensation_amount"`
	Name                              string          `json:"name"`
	ParentID                          int             `json:"parent_id"`
	Price                             float64         `json:"price"`
	PriceInclTax                      float64         `json:"price_incl_tax"`
	ProductID                         int             `json:"product_id"`
	RowTotal                          float64         `json:"row_total"`
	RowTotalInclTax                   float64         `json:"row_total_incl_tax"`
	Sku                               string          `json:"sku"`
	TaxAmount                         float64         `json:"tax_amount"`
	ExtensionAttributes               json.RawMessage `json:"extension_attributes"`
	OrderItemID                       int             `json:"order_item_id"`
	Qty                               float64         `json:"qty"`
}

// InvoiceComment is an invoice comment.
// Fields: module-sales/Api/Data/InvoiceCommentInterface.php (Magento 2.4.8), schema definition sales-data-invoice-comment-interface.
type InvoiceComment struct {
	IsCustomerNotified  int             `json:"is_customer_notified"`
	ParentID            int             `json:"parent_id"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
	Comment             string          `json:"comment"`
	IsVisibleOnFront    int             `json:"is_visible_on_front"`
	CreatedAt           string          `json:"created_at"`
	EntityID            int             `json:"entity_id"`
}

// CreditMemo is a credit memo (GET /V1/creditmemo/{id}).
// Fields: module-sales/Api/Data/CreditmemoInterface.php (Magento 2.4.8), schema definition sales-data-creditmemo-interface.
type CreditMemo struct {
	Adjustment                              float64             `json:"adjustment"`
	AdjustmentNegative                      float64             `json:"adjustment_negative"`
	AdjustmentPositive                      float64             `json:"adjustment_positive"`
	BaseAdjustment                          float64             `json:"base_adjustment"`
	BaseAdjustmentNegative                  float64             `json:"base_adjustment_negative"`
	BaseAdjustmentPositive                  float64             `json:"base_adjustment_positive"`
	BaseCurrencyCode                        string              `json:"base_currency_code"`
	BaseDiscountAmount                      float64             `json:"base_discount_amount"`
	BaseGrandTotal                          float64             `json:"base_grand_total"`
	BaseDiscountTaxCompensationAmount       float64             `json:"base_discount_tax_compensation_amount"`
	BaseShippingAmount                      float64             `json:"base_shipping_amount"`
	BaseShippingDiscountTaxCompensationAmnt float64             `json:"base_shipping_discount_tax_compensation_amnt"`
	BaseShippingInclTax                     float64             `json:"base_shipping_incl_tax"`
	BaseShippingTaxAmount                   float64             `json:"base_shipping_tax_amount"`
	BaseSubtotal                            float64             `json:"base_subtotal"`
	BaseSubtotalInclTax                     float64             `json:"base_subtotal_incl_tax"`
	BaseTaxAmount                           float64             `json:"base_tax_amount"`
	BaseToGlobalRate                        float64             `json:"base_to_global_rate"`
	BaseToOrderRate                         float64             `json:"base_to_order_rate"`
	BillingAddressID                        int                 `json:"billing_address_id"`
	CreatedAt                               string              `json:"created_at"`
	CreditmemoStatus                        int                 `json:"creditmemo_status"`
	DiscountAmount                          float64             `json:"discount_amount"`
	DiscountDescription                     string              `json:"discount_description"`
	EmailSent                               int                 `json:"email_sent"`
	EntityID                                int                 `json:"entity_id"`
	GlobalCurrencyCode                      string              `json:"global_currency_code"`
	GrandTotal                              float64             `json:"grand_total"`
	DiscountTaxCompensationAmount           float64             `json:"discount_tax_compensation_amount"`
	IncrementID                             string              `json:"increment_id"`
	InvoiceID                               int                 `json:"invoice_id"`
	OrderCurrencyCode                       string              `json:"order_currency_code"`
	OrderID                                 int                 `json:"order_id"`
	ShippingAddressID                       int                 `json:"shipping_address_id"`
	ShippingAmount                          float64             `json:"shipping_amount"`
	ShippingDiscountTaxCompensationAmount   float64             `json:"shipping_discount_tax_compensation_amount"`
	ShippingInclTax                         float64             `json:"shipping_incl_tax"`
	ShippingTaxAmount                       float64             `json:"shipping_tax_amount"`
	State                                   int                 `json:"state"`
	StoreCurrencyCode                       string              `json:"store_currency_code"`
	StoreID                                 int                 `json:"store_id"`
	StoreToBaseRate                         float64             `json:"store_to_base_rate"`
	StoreToOrderRate                        float64             `json:"store_to_order_rate"`
	Subtotal                                float64             `json:"subtotal"`
	SubtotalInclTax                         float64             `json:"subtotal_incl_tax"`
	TaxAmount                               float64             `json:"tax_amount"`
	TransactionID                           string              `json:"transaction_id"`
	UpdatedAt                               string              `json:"updated_at"`
	Items                                   []CreditMemoItem    `json:"items"`
	Comments                                []CreditMemoComment `json:"comments"`
	ExtensionAttributes                     json.RawMessage     `json:"extension_attributes"`
}

// CreditMemoItem is a credit memo line.
// Fields: module-sales/Api/Data/CreditmemoItemInterface.php (Magento 2.4.8), schema definition sales-data-creditmemo-item-interface.
type CreditMemoItem struct {
	AdditionalData                    string          `json:"additional_data"`
	BaseCost                          float64         `json:"base_cost"`
	BaseDiscountAmount                float64         `json:"base_discount_amount"`
	BaseDiscountTaxCompensationAmount float64         `json:"base_discount_tax_compensation_amount"`
	BasePrice                         float64         `json:"base_price"`
	BasePriceInclTax                  float64         `json:"base_price_incl_tax"`
	BaseRowTotal                      float64         `json:"base_row_total"`
	BaseRowTotalInclTax               float64         `json:"base_row_total_incl_tax"`
	BaseTaxAmount                     float64         `json:"base_tax_amount"`
	BaseWeeeTaxAppliedAmount          float64         `json:"base_weee_tax_applied_amount"`
	BaseWeeeTaxAppliedRowAmnt         float64         `json:"base_weee_tax_applied_row_amnt"`
	BaseWeeeTaxDisposition            float64         `json:"base_weee_tax_disposition"`
	BaseWeeeTaxRowDisposition         float64         `json:"base_weee_tax_row_disposition"`
	Description                       string          `json:"description"`
	DiscountAmount                    float64         `json:"discount_amount"`
	EntityID                          int             `json:"entity_id"`
	DiscountTaxCompensationAmount     float64         `json:"discount_tax_compensation_amount"`
	Name                              string          `json:"name"`
	OrderItemID                       int             `json:"order_item_id"`
	ParentID                          int             `json:"parent_id"`
	Price                             float64         `json:"price"`
	PriceInclTax                      float64         `json:"price_incl_tax"`
	ProductID                         int             `json:"product_id"`
	Qty                               float64         `json:"qty"`
	RowTotal                          float64         `json:"row_total"`
	RowTotalInclTax                   float64         `json:"row_total_incl_tax"`
	Sku                               string          `json:"sku"`
	TaxAmount                         float64         `json:"tax_amount"`
	WeeeTaxApplied                    string          `json:"weee_tax_applied"`
	WeeeTaxAppliedAmount              float64         `json:"weee_tax_applied_amount"`
	WeeeTaxAppliedRowAmount           float64         `json:"weee_tax_applied_row_amount"`
	WeeeTaxDisposition                float64         `json:"weee_tax_disposition"`
	WeeeTaxRowDisposition             float64         `json:"weee_tax_row_disposition"`
	ExtensionAttributes               json.RawMessage `json:"extension_attributes"`
}

// CreditMemoComment is a credit memo comment.
// Fields: module-sales/Api/Data/CreditmemoCommentInterface.php (Magento 2.4.8), schema definition sales-data-creditmemo-comment-interface.
type CreditMemoComment struct {
	Comment             string          `json:"comment"`
	CreatedAt           string          `json:"created_at"`
	EntityID            int             `json:"entity_id"`
	IsCustomerNotified  int             `json:"is_customer_notified"`
	IsVisibleOnFront    int             `json:"is_visible_on_front"`
	ParentID            int             `json:"parent_id"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
}

// Shipment is a shipment (GET /V1/shipment/{id}).
// Fields: module-sales/Api/Data/ShipmentInterface.php (Magento 2.4.8), schema definition sales-data-shipment-interface.
type Shipment struct {
	BillingAddressID    int               `json:"billing_address_id"`
	CreatedAt           string            `json:"created_at"`
	CustomerID          int               `json:"customer_id"`
	EmailSent           int               `json:"email_sent"`
	EntityID            int               `json:"entity_id"`
	IncrementID         string            `json:"increment_id"`
	OrderID             int               `json:"order_id"`
	Packages            json.RawMessage   `json:"packages"`
	ShipmentStatus      int               `json:"shipment_status"`
	ShippingAddressID   int               `json:"shipping_address_id"`
	ShippingLabel       string            `json:"shipping_label"`
	StoreID             int               `json:"store_id"`
	TotalQty            float64           `json:"total_qty"`
	TotalWeight         float64           `json:"total_weight"`
	UpdatedAt           string            `json:"updated_at"`
	Items               []ShipmentItem    `json:"items"`
	Tracks              []ShipmentTrack   `json:"tracks"`
	Comments            []ShipmentComment `json:"comments"`
	ExtensionAttributes json.RawMessage   `json:"extension_attributes"`
}

// ShipmentItem is a shipment line.
// Fields: module-sales/Api/Data/ShipmentItemInterface.php (Magento 2.4.8), schema definition sales-data-shipment-item-interface.
type ShipmentItem struct {
	AdditionalData      string          `json:"additional_data"`
	Description         string          `json:"description"`
	EntityID            int             `json:"entity_id"`
	Name                string          `json:"name"`
	ParentID            int             `json:"parent_id"`
	Price               float64         `json:"price"`
	ProductID           int             `json:"product_id"`
	RowTotal            float64         `json:"row_total"`
	Sku                 string          `json:"sku"`
	Weight              float64         `json:"weight"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
	OrderItemID         int             `json:"order_item_id"`
	Qty                 float64         `json:"qty"`
}

// ShipmentTrack is a shipment tracking number.
// Fields: module-sales/Api/Data/ShipmentTrackInterface.php (Magento 2.4.8), schema definition sales-data-shipment-track-interface.
type ShipmentTrack struct {
	OrderID             int             `json:"order_id"`
	CreatedAt           string          `json:"created_at"`
	EntityID            int             `json:"entity_id"`
	ParentID            int             `json:"parent_id"`
	UpdatedAt           string          `json:"updated_at"`
	Weight              float64         `json:"weight"`
	Qty                 float64         `json:"qty"`
	Description         string          `json:"description"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
	TrackNumber         string          `json:"track_number"`
	Title               string          `json:"title"`
	CarrierCode         string          `json:"carrier_code"`
}

// ShipmentComment is a shipment comment.
// Fields: module-sales/Api/Data/ShipmentCommentInterface.php (Magento 2.4.8), schema definition sales-data-shipment-comment-interface.
type ShipmentComment struct {
	IsCustomerNotified  int             `json:"is_customer_notified"`
	ParentID            int             `json:"parent_id"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
	Comment             string          `json:"comment"`
	IsVisibleOnFront    int             `json:"is_visible_on_front"`
	CreatedAt           string          `json:"created_at"`
	EntityID            int             `json:"entity_id"`
}

// StoreConfigEntry is the configuration of one store view (GET /V1/store/storeConfigs).
// Fields: module-store/Api/Data/StoreConfigInterface.php (Magento 2.4.8), schema definition store-data-store-config-interface.
type StoreConfigEntry struct {
	ID                         int             `json:"id"`
	Code                       string          `json:"code"`
	WebsiteID                  int             `json:"website_id"`
	Locale                     string          `json:"locale"`
	BaseCurrencyCode           string          `json:"base_currency_code"`
	DefaultDisplayCurrencyCode string          `json:"default_display_currency_code"`
	Timezone                   string          `json:"timezone"`
	WeightUnit                 string          `json:"weight_unit"`
	BaseURL                    string          `json:"base_url"`
	BaseLinkURL                string          `json:"base_link_url"`
	BaseStaticURL              string          `json:"base_static_url"`
	BaseMediaURL               string          `json:"base_media_url"`
	SecureBaseURL              string          `json:"secure_base_url"`
	SecureBaseLinkURL          string          `json:"secure_base_link_url"`
	SecureBaseStaticURL        string          `json:"secure_base_static_url"`
	SecureBaseMediaURL         string          `json:"secure_base_media_url"`
	ExtensionAttributes        json.RawMessage `json:"extension_attributes"`
}

// StoreGroup is a store group (GET /V1/store/storeGroups).
// Fields: module-store/Api/Data/GroupInterface.php (Magento 2.4.8), schema definition store-data-group-interface.
type StoreGroup struct {
	ID                  int             `json:"id"`
	WebsiteID           int             `json:"website_id"`
	RootCategoryID      int             `json:"root_category_id"`
	DefaultStoreID      int             `json:"default_store_id"`
	Name                string          `json:"name"`
	Code                string          `json:"code"`
	ExtensionAttributes json.RawMessage `json:"extension_attributes"`
}
