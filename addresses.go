package magento2

// OrderAddress is a sales order address: Order.BillingAddress and the
// address of a shipping assignment
// (module-sales/Api/Data/OrderAddressInterface.php, schema definition
// sales-data-order-address-interface). Quote (cart) addresses are Address,
// BillingAddress and ShippingAddress; customer addresses are
// CustomerAddress.
type OrderAddress struct {
	EntityID            int            `json:"entity_id,omitempty"`
	ParentID            int            `json:"parent_id,omitempty"`
	AddressType         string         `json:"address_type,omitempty"`
	CustomerAddressID   int            `json:"customer_address_id,omitempty"`
	CustomerID          int            `json:"customer_id,omitempty"`
	Firstname           string         `json:"firstname,omitempty"`
	Middlename          string         `json:"middlename,omitempty"`
	Lastname            string         `json:"lastname,omitempty"`
	Prefix              string         `json:"prefix,omitempty"`
	Suffix              string         `json:"suffix,omitempty"`
	Company             string         `json:"company,omitempty"`
	Street              []string       `json:"street,omitempty"`
	City                string         `json:"city,omitempty"`
	Region              string         `json:"region,omitempty"`
	RegionCode          string         `json:"region_code,omitempty"`
	RegionID            int            `json:"region_id,omitempty"`
	Postcode            string         `json:"postcode,omitempty"`
	CountryID           string         `json:"country_id,omitempty"`
	Telephone           string         `json:"telephone,omitempty"`
	Fax                 string         `json:"fax,omitempty"`
	Email               string         `json:"email,omitempty"`
	VatID               string         `json:"vat_id,omitempty"`
	VatIsValid          int            `json:"vat_is_valid,omitempty"`
	VatRequestID        string         `json:"vat_request_id,omitempty"`
	VatRequestDate      string         `json:"vat_request_date,omitempty"`
	VatRequestSuccess   int            `json:"vat_request_success,omitempty"`
	ExtensionAttributes map[string]any `json:"extension_attributes,omitempty"`
}

// CustomerAddress is an address in a customer's address book: the entries
// of Customer.Addresses (module-customer/Api/Data/AddressInterface.php,
// schema definition customer-data-address-interface). Region is an object
// here, and DefaultBilling/DefaultShipping are booleans.
type CustomerAddress struct {
	ID                  int              `json:"id,omitempty"`
	CustomerID          int              `json:"customer_id,omitempty"`
	Region              *Region          `json:"region,omitempty"`
	RegionID            int              `json:"region_id,omitempty"`
	CountryID           string           `json:"country_id,omitempty"`
	Street              []string         `json:"street,omitempty"`
	Company             string           `json:"company,omitempty"`
	Telephone           string           `json:"telephone,omitempty"`
	Fax                 string           `json:"fax,omitempty"`
	Postcode            string           `json:"postcode,omitempty"`
	City                string           `json:"city,omitempty"`
	Firstname           string           `json:"firstname,omitempty"`
	Lastname            string           `json:"lastname,omitempty"`
	Middlename          string           `json:"middlename,omitempty"`
	Prefix              string           `json:"prefix,omitempty"`
	Suffix              string           `json:"suffix,omitempty"`
	VatID               string           `json:"vat_id,omitempty"`
	DefaultShipping     bool             `json:"default_shipping,omitempty"`
	DefaultBilling      bool             `json:"default_billing,omitempty"`
	ExtensionAttributes map[string]any   `json:"extension_attributes,omitempty"`
	CustomAttributes    []map[string]any `json:"custom_attributes,omitempty"`
}
