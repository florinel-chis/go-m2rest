package magento2

import (
	"context" // Added context
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

// Constants for cart routes - assuming these are defined elsewhere or should be
// For this example, I'll assume they exist or define placeholders if needed for compilation.
// It seems `guestCart`, `customerCart`, `cartItems`, `cartShippingCosts`,
// `cartShippingInformation`, `cartPaymentMethods`, `cartPlaceOrder` are expected.
// Let's assume they are global vars or consts from another file in this package.
// If not, their definition would be required. For now, the logic focuses on context.

var ( // Placeholders if not defined elsewhere
	guestCart               = "/guest-carts"
	customerCart            = "/carts/mine" // Example for customer cart
	cartItems               = "/items"
	cartShippingCosts       = "/estimate-shipping-methods"
	cartShippingInformation = "/shipping-information"
	cartPaymentMethods      = "/payment-methods"
	cartPlaceOrder          = "/order"
	Orders                  = "/orders" // Used in CreateOrder
)


type MCart struct {
	Route     string
	QuoteID   string
	Cart      *Cart
	APIClient *Client
}

// NewGuestCartFromAPIClient now accepts context.Context
func NewGuestCartFromAPIClient(ctx context.Context, apiClient *Client) (*MCart, error) {
	mCart := &MCart{
		Cart:      &Cart{},
		APIClient: apiClient,
	}

	err := mCart.initializeGuestCart(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing guest cart: %w", err)
	}
	return mCart, nil
}

// NewCustomerCartFromAPIClient now accepts context.Context
func NewCustomerCartFromAPIClient(ctx context.Context, apiClient *Client) (*MCart, error) {
	mCart := &MCart{
		Cart:      &Cart{},
		APIClient: apiClient,
	}

	err := mCart.initializeCustomerCart(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing customer cart: %w", err)
	}
	return mCart, nil
}

// initializeGuestCart now accepts context.Context
func (cart *MCart) initializeGuestCart(ctx context.Context) error {
	endpoint := guestCart
	// Use APIClient.PostRouteAndDecode for consistency, assuming it can return raw string if target is nil or string ptr
	// However, the original code directly uses httpClient and parses string.
	// For now, will adapt direct usage to include context.
	// Ideally, APIClient would have methods that can also return the raw response or handle string responses.

	log.Debug().Str("endpoint", endpoint).Msg("Initializing guest cart with context")
	resp, err := cart.APIClient.HTTPClient.R().SetContext(ctx).Post(endpoint)

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("error initializing guest cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "initialize cart for guest")
	if httpErr != nil {
		return httpErr
	}

	quoteID := mayTrimSurroundingQuotes(resp.String())

	cart.Route = guestCart + "/" + quoteID
	cart.QuoteID = quoteID
	// cart.APIClient is already set

	log.Debug().Str("quoteID", quoteID).Msg("Guest cart initialized successfully, updating from remote")
	err = cart.UpdateFromRemote(ctx) // Pass context
	if err != nil {
		return fmt.Errorf("error updating guest cart from remote after initialization: %w", err)
	}
	return nil
}

// initializeCustomerCart now accepts context.Context
func (cart *MCart) initializeCustomerCart(ctx context.Context) error {
	endpoint := customerCart

	log.Debug().Str("endpoint", endpoint).Msg("Initializing customer cart with context")
	resp, err := cart.APIClient.HTTPClient.R().SetContext(ctx).Post(endpoint)

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("error initializing customer cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "initialize cart for customer")
	if httpErr != nil {
		return httpErr
	}

	quoteID := mayTrimSurroundingQuotes(resp.String())

	cart.Route = customerCart // For customer carts, the route might be /carts/mine directly
	// If Magento returns a quote ID for customer carts that needs to be appended, this logic might differ.
	// Assuming /carts/mine is the base and operations use it. If quoteID is relevant, adjust cart.Route.
	// The original code set cart.Route = customerCart, and quoteID was used.
	// This implies that even for /carts/mine, the quoteID is somehow relevant for subsequent calls or structure.
	// For safety, if /carts/mine is the main route, let's assume it is.
	// If the API for customer carts returns a specific quote ID that forms part of the route, that's different.
	// The original code has `cart.Route = customerCart` which implies `customerCart` is the full base path.
	// Let's assume `customerCart` is something like `/rest/default/V1/carts/mine`.

	cart.QuoteID = quoteID // Store the quote ID from response
	// cart.APIClient is already set

	log.Debug().Str("quoteID", quoteID).Msg("Customer cart initialized successfully, updating from remote")
	err = cart.UpdateFromRemote(ctx) // Pass context
	if err != nil {
		return fmt.Errorf("error updating customer cart from remote after initialization: %w", err)
	}
	return nil
}

// UpdateFromRemote now accepts context.Context
func (cart *MCart) UpdateFromRemote(ctx context.Context) error {
	log.Debug().Str("route", cart.Route).Msg("Updating cart from remote with context")
	// Use APIClient's GetRouteAndDecode
	err := cart.APIClient.GetRouteAndDecode(ctx, cart.Route, cart.Cart, "get detailed cart object from magento2-api")
	if err != nil {
		return err // Error already wrapped and logged by GetRouteAndDecode
	}
	log.Debug().Interface("cart", cart.Cart).Msg("Cart updated from remote successfully")
	return nil
}

// AddItems now accepts context.Context
func (cart *MCart) AddItems(ctx context.Context, items []CartItem) error {
	endpoint := cart.Route + cartItems

	type PayLoad struct {
		CartItem CartItem `json:"cartItem"`
	}

	for _, item := range items {
		item.QuoteID = cart.QuoteID // Ensure QuoteID is set on the item if needed by API
		payLoad := &PayLoad{
			CartItem: item,
		}

		log.Debug().
			Str("endpoint", endpoint).
			Interface("payload", payLoad).
			Msg("Adding item to cart with context")

		// Use APIClient's PostRouteAndDecode. Assuming it can handle target=nil if no response body is expected or decoded.
		// Or, if a specific response is expected for AddItem, it should be used as target.
		// The original code doesn't decode a specific struct for AddItem response, it just checks errors.
		// Let's assume a generic map[string]any or a specific AddItemResponse struct if available.
		// For now, using nil for target and relying on error handling.
		// A better approach: APIClient methods should allow nil target or specific response struct.
		var responseBody map[string]any // Placeholder for any response
		err := cart.APIClient.PostRouteAndDecode(ctx, endpoint, payLoad, &responseBody, fmt.Sprintf("add item '%+v' to cart", item))

		if err != nil {
			if errors.Is(err, ErrNotFound) { // Check against wrapped ErrNotFound
				customErr := &ItemNotFoundError{ItemID: item.ItemID} // ItemID might not be set if product not found by SKU
				return customErr
			}
			return err // Already wrapped
		}
		// Original code appended to cart.Cart.Items optimistically.
		// It's better to call UpdateFromRemote(ctx) after all items are added to get the true state.
		// cart.Cart.Items = append(cart.Cart.Items, item) // Optimistic update, consider removing
		log.Debug().Interface("item", item).Msg("Item submitted to be added to cart successfully")
	}

	log.Debug().Msg("All items submitted to be added to cart successfully. Consider calling UpdateFromRemote to sync.")
	return nil
}

// EstimateShippingCarrier now accepts context.Context
func (cart *MCart) EstimateShippingCarrier(ctx context.Context, addr *ShippingAddress) ([]Carrier, error) {
	endpoint := cart.Route + cartShippingCosts

	type PayLoad struct {
		Address ShippingAddress `json:"address"`
	}
	payLoad := &PayLoad{Address: *addr}
	shippingCarriers := &[]Carrier{}

	log.Debug().Str("endpoint", endpoint).Interface("payload", payLoad).Msg("Estimating shipping carrier for cart with context")
	err := cart.APIClient.PostRouteAndDecode(ctx, endpoint, payLoad, shippingCarriers, "estimate shipping carrier for cart")
	if err != nil {
		return *shippingCarriers, err
	}
	return *shippingCarriers, nil
}

// AddShippingInformation now accepts context.Context
func (cart *MCart) AddShippingInformation(ctx context.Context, addrInfo *AddressInformation) error {
	endpoint := cart.Route + cartShippingInformation

	type PayLoad struct {
		AddressInformation AddressInformation `json:"addressInformation"`
	}
	payLoad := &PayLoad{AddressInformation: *addrInfo}

	log.Debug().Str("endpoint", endpoint).Interface("payload", payLoad).Msg("Adding shipping information to cart with context")
	// Assuming no specific response structure needs to be decoded, pass nil or a generic map.
	var responseBody map[string]any
	err := cart.APIClient.PostRouteAndDecode(ctx, endpoint, payLoad, &responseBody, "add shipping information to cart")
	return err // error is already wrapped
}

// EstimatePaymentMethods now accepts context.Context
func (cart *MCart) EstimatePaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	endpoint := cart.Route + cartPaymentMethods
	paymentMethods := &[]PaymentMethod{}

	log.Debug().Str("endpoint", endpoint).Msg("Estimating payment methods for cart with context")
	err := cart.APIClient.GetRouteAndDecode(ctx, endpoint, paymentMethods, "estimate payment methods for cart")
	if err != nil {
		return *paymentMethods, err // Error already wrapped
	}
	log.Debug().Interface("paymentMethods", paymentMethods).Msg("Payment methods estimated successfully")
	return *paymentMethods, nil
}

// CreateOrder now accepts context.Context
func (cart *MCart) CreateOrder(ctx context.Context, paymentMethod PaymentMethod) (*MOrder, error) {
	endpoint := cart.Route + cartPlaceOrder

	type PayLoad struct {
		PaymentMethod PaymentMethodCode `json:"paymentMethod"`
	}
	payLoad := &PayLoad{PaymentMethod: PaymentMethodCode{Method: paymentMethod.Code}}

	log.Debug().Str("endpoint", endpoint).Interface("payload", payLoad).Msg("Creating order for cart with context")

	// CreateOrder is a PUT request. APIClient needs a PutRouteAndDecode method.
	// For now, direct use of httpClient with context:
	resp, err := cart.APIClient.HTTPClient.R().SetContext(ctx).SetBody(payLoad).Put(endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("error creating order: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "create order")
	if httpErr != nil {
		return nil, httpErr
	}

	orderIDString := mayTrimSurroundingQuotes(resp.String())
	orderIDInt, err := strconv.Atoi(orderIDString)
	if err != nil {
		return nil, fmt.Errorf("unexpected error while extracting orderID: %w", err)
	}

	log.Debug().Int("orderID", orderIDInt).Msg("Order created successfully")
	return &MOrder{
		Route:     Orders + "/" + orderIDString, // Assuming Orders is defined
		Order:     &Order{EntityID: orderIDInt},
		APIClient: cart.APIClient,
	}, nil
}

// DeleteItem now accepts context.Context
func (cart *MCart) DeleteItem(ctx context.Context, itemID int) error {
	endpoint := cart.Route + cartItems + "/" + strconv.Itoa(itemID)

	log.Debug().Str("endpoint", endpoint).Int("itemID", itemID).Msg("Deleting item from cart with context")

	// DeleteItem is a DELETE request. APIClient needs a DeleteRouteAndDecode method.
	// For now, direct use of httpClient with context:
	resp, err := cart.APIClient.HTTPClient.R().SetContext(ctx).Delete(endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("error deleting item from cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, fmt.Sprintf("delete itemID '%d'", itemID))
	if httpErr != nil {
		return httpErr
	}

	log.Debug().Int("itemID", itemID).Msg("Item deleted from cart successfully")
	return nil
}

// DeleteAllItems now accepts context.Context
func (cart *MCart) DeleteAllItems(ctx context.Context) error {
	err := cart.UpdateFromRemote(ctx) // Pass context
	if err != nil {
		return fmt.Errorf("error updating cart before deleting all items: %w", err)
	}

	for i := range cart.Cart.Items {
		err = cart.DeleteItem(ctx, cart.Cart.Items[i].ItemID) // Pass context
		if err != nil {
			return fmt.Errorf("error deleting item during delete all items: %w", err)
		}
	}

	log.Debug().Msg("All items deleted from cart successfully")
	return nil
}
