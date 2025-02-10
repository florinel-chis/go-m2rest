package magento2

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

type MCart struct {
	Route     string
	QuoteID   string
	Cart      *Cart
	APIClient *Client
}

func NewGuestCartFromAPIClient(apiClient *Client) (*MCart, error) {
	mCart := &MCart{
		Cart:      &Cart{},
		APIClient: apiClient,
	}

	err := mCart.initializeGuestCart()
	if err != nil {
		return nil, fmt.Errorf("error initializing guest cart: %w", err)
	}
	return mCart, nil
}

func NewCustomerCartFromAPIClient(apiClient *Client) (*MCart, error) {
	mCart := &MCart{
		Cart:      &Cart{},
		APIClient: apiClient,
	}

	err := mCart.initializeCustomerCart()
	if err != nil {
		return nil, fmt.Errorf("error initializing customer cart: %w", err)
	}
	return mCart, nil
}

func (cart *MCart) initializeGuestCart() error {
	endpoint := guestCart
	apiClient := cart.APIClient

	httpClient := apiClient.HTTPClient
	log.Debug().Str("endpoint", endpoint).Msg("Initializing guest cart")
	resp, err := httpClient.R().Post(endpoint)

	if err != nil {
		return fmt.Errorf("error initializing guest cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "initialize cart for guest")
	if httpErr != nil {
		return httpErr
	}

	quoteID := mayTrimSurroundingQuotes(resp.String())

	cart.Route = guestCart + "/" + quoteID
	cart.QuoteID = quoteID
	cart.APIClient = apiClient

	log.Debug().Str("quoteID", quoteID).Msg("Guest cart initialized successfully, updating from remote")
	err = cart.UpdateFromRemote()
	if err != nil {
		return fmt.Errorf("error updating guest cart from remote after initialization: %w", err)
	}
	return nil
}

func (cart *MCart) initializeCustomerCart() error {
	endpoint := customerCart
	apiClient := cart.APIClient

	httpClient := apiClient.HTTPClient
	log.Debug().Str("endpoint", endpoint).Msg("Initializing customer cart")
	resp, err := httpClient.R().Post(endpoint)

	if err != nil {
		return fmt.Errorf("error initializing customer cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "initialize cart for customer")
	if httpErr != nil {
		return httpErr
	}

	quoteID := mayTrimSurroundingQuotes(resp.String())

	cart.Route = customerCart
	cart.QuoteID = quoteID
	cart.APIClient = apiClient

	log.Debug().Str("quoteID", quoteID).Msg("Customer cart initialized successfully, updating from remote")
	err = cart.UpdateFromRemote()
	if err != nil {
		return fmt.Errorf("error updating customer cart from remote after initialization: %w", err)
	}
	return nil
}

func (cart *MCart) UpdateFromRemote() error {
	httpClient := cart.APIClient.HTTPClient
	log.Debug().Str("route", cart.Route).Msg("Updating cart from remote")

	resp, err := httpClient.R().SetResult(cart.Cart).Get(cart.Route)

	if err != nil {
		return fmt.Errorf("error updating cart from remote: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get detailed cart object from magento2-api")
	if httpErr != nil {
		return httpErr
	}

	log.Debug().Interface("cart", cart.Cart).Msg("Cart updated from remote successfully")
	return nil
}

func (cart *MCart) AddItems(items []CartItem) error {
	endpoint := cart.Route + cartItems
	httpClient := cart.APIClient.HTTPClient

	type PayLoad struct {
		CartItem CartItem `json:"cartItem"`
	}

	for _, item := range items {
		item.QuoteID = cart.QuoteID
		payLoad := &PayLoad{
			CartItem: item,
		}

		log.Debug().
			Str("endpoint", endpoint).
			Interface("payload", payLoad).
			Msg("Adding item to cart")

		resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)

		httpErr := mayReturnErrorForHTTPResponse(resp, fmt.Sprintf("add item '%+v' to cart", item))
		if httpErr != nil && errors.Is(httpErr, ErrNotFound) {
			customErr := &ItemNotFoundError{ItemID: item.ItemID}
			return customErr
		} else if httpErr != nil {
			return httpErr
		}
		if err != nil {
			return fmt.Errorf("error adding item to cart: %w", err)
		}

		cart.Cart.Items = append(cart.Cart.Items, item)
		log.Debug().Interface("item", item).Msg("Item added to cart successfully")
	}

	log.Debug().Msg("All items added to cart successfully")
	return nil
}

func (cart *MCart) EstimateShippingCarrier(addr *ShippingAddress) ([]Carrier, error) {
	endpoint := cart.Route + cartShippingCosts
	httpClient := cart.APIClient.HTTPClient

	type PayLoad struct {
		Address ShippingAddress `json:"address"`
	}

	payLoad := &PayLoad{
		Address: *addr,
	}

	shippingCarrier := &[]Carrier{}

	log.Debug().
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Estimating shipping carrier for cart")

	resp, err := httpClient.R().SetBody(*payLoad).SetResult(shippingCarrier).Post(endpoint)

	if err != nil {
		log.Error().Err(err).Msg("Error estimating shipping carrier")
		return *shippingCarrier, fmt.Errorf("error estimating shipping carrier: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Shipping carrier estimation response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "estimate shipping carrier for cart")
	if httpErr != nil {
		return *shippingCarrier, httpErr
	}

	return *shippingCarrier, nil
}

func (cart *MCart) AddShippingInformation(addrInfo *AddressInformation) error {
	endpoint := cart.Route + cartShippingInformation
	httpClient := cart.APIClient.HTTPClient

	type PayLoad struct {
		AddressInformation AddressInformation `json:"addressInformation"`
	}

	payLoad := &PayLoad{
		AddressInformation: *addrInfo,
	}

	log.Debug().
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Adding shipping information to cart")

	resp, err := httpClient.R().SetBody(*payLoad).Post(endpoint)

	if err != nil {
		log.Error().Err(err).Msg("Error adding shipping information")
		return fmt.Errorf("error adding shipping information to cart: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Shipping information added response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "add shipping information to cart")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (cart *MCart) EstimatePaymentMethods() ([]PaymentMethod, error) {
	endpoint := cart.Route + cartPaymentMethods

	paymentMethods := &[]PaymentMethod{}

	log.Debug().Str("endpoint", endpoint).Msg("Estimating payment methods for cart")
	err := cart.APIClient.GetRouteAndDecode(endpoint, paymentMethods, "estimate payment methods for cart")
	if err != nil {
		return *paymentMethods, fmt.Errorf("error estimating payment methods: %w", err)
	}
	log.Debug().Interface("paymentMethods", paymentMethods).Msg("Payment methods estimated successfully")
	return *paymentMethods, nil
}

func (cart *MCart) CreateOrder(paymentMethod PaymentMethod) (*MOrder, error) {
	endpoint := cart.Route + cartPlaceOrder
	httpClient := cart.APIClient.HTTPClient

	type PayLoad struct {
		PaymentMethod PaymentMethodCode `json:"paymentMethod"`
	}

	payLoad := &PayLoad{
		PaymentMethod: PaymentMethodCode{
			Method: paymentMethod.Code,
		},
	}

	log.Debug().
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Creating order for cart")

	resp, err := httpClient.R().SetBody(payLoad).Put(endpoint)

	if err != nil {
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
		Route: Orders + "/" + orderIDString,
		Order: &Order{
			EntityID: orderIDInt,
		},
		APIClient: cart.APIClient,
	}, nil
}

func (cart *MCart) DeleteItem(itemID int) error {
	endpoint := cart.Route + cartItems + "/" + strconv.Itoa(itemID)
	httpClient := cart.APIClient.HTTPClient

	log.Debug().Str("endpoint", endpoint).Int("itemID", itemID).Msg("Deleting item from cart")
	resp, err := httpClient.R().Delete(endpoint)

	if err != nil {
		return fmt.Errorf("error deleting item from cart: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, fmt.Sprintf("delete itemID '%d'", itemID))
	if httpErr != nil {
		return httpErr
	}

	log.Debug().Int("itemID", itemID).Msg("Item deleted from cart successfully")
	return nil
}

func (cart *MCart) DeleteAllItems() error {
	err := cart.UpdateFromRemote()
	if err != nil {
		return fmt.Errorf("error updating cart before deleting all items: %w", err)
	}

	for i := range cart.Cart.Items {
		err = cart.DeleteItem(cart.Cart.Items[i].ItemID)
		if err != nil {
			return fmt.Errorf("error deleting item during delete all items: %w", err)
		}
	}

	log.Debug().Msg("All items deleted from cart successfully")
	return nil
}
