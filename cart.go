package magento2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	body, err := cart.APIClient.v1(context.Background(), http.MethodPost, guestCart, nil, nil, "initialize cart for guest")
	if err != nil {
		return err
	}

	quoteID := mayTrimSurroundingQuotes(string(body))
	cart.Route = guestCart + "/" + url.PathEscape(quoteID)
	cart.QuoteID = quoteID

	err = cart.UpdateFromRemote()
	if err != nil {
		return fmt.Errorf("error updating guest cart from remote after initialization: %w", err)
	}
	return nil
}

func (cart *MCart) initializeCustomerCart() error {
	// POST /carts/mine answers the quote id as a JSON number or string.
	body, err := cart.APIClient.v1(context.Background(), http.MethodPost, customerCart, nil, nil, "initialize cart for customer")
	if err != nil {
		return err
	}

	cart.Route = customerCart
	cart.QuoteID = mayTrimSurroundingQuotes(string(body))

	err = cart.UpdateFromRemote()
	if err != nil {
		return fmt.Errorf("error updating customer cart from remote after initialization: %w", err)
	}
	return nil
}

func (cart *MCart) UpdateFromRemote() error {
	_, err := cart.APIClient.v1(context.Background(), http.MethodGet, cart.Route, nil, cart.Cart, "get detailed cart object from magento2-api")
	return err
}

func (cart *MCart) AddItems(items []CartItem) error {
	endpoint := cart.Route + cartItems

	type PayLoad struct {
		CartItem CartItem `json:"cartItem"`
	}

	for _, item := range items {
		item.QuoteID = cart.QuoteID
		payLoad := &PayLoad{
			CartItem: item,
		}

		_, err := cart.APIClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, nil, fmt.Sprintf("add item '%+v' to cart", item))
		if errors.Is(err, ErrNotFound) {
			return &ItemNotFoundError{ItemID: item.ItemID}
		} else if err != nil {
			return err
		}

		cart.Cart.Items = append(cart.Cart.Items, item)
	}

	return nil
}

func (cart *MCart) EstimateShippingCarrier(addr *ShippingAddress) ([]Carrier, error) {
	endpoint := cart.Route + cartShippingCosts

	type PayLoad struct {
		Address ShippingAddress `json:"address"`
	}

	payLoad := &PayLoad{
		Address: *addr,
	}

	shippingCarrier := []Carrier{}
	_, err := cart.APIClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, &shippingCarrier, "estimate shipping carrier for cart")
	return shippingCarrier, err
}

func (cart *MCart) AddShippingInformation(addrInfo *AddressInformation) error {
	endpoint := cart.Route + cartShippingInformation

	type PayLoad struct {
		AddressInformation AddressInformation `json:"addressInformation"`
	}

	payLoad := &PayLoad{
		AddressInformation: *addrInfo,
	}

	_, err := cart.APIClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, nil, "add shipping information to cart")
	return err
}

func (cart *MCart) EstimatePaymentMethods() ([]PaymentMethod, error) {
	endpoint := cart.Route + cartPaymentMethods

	paymentMethods := &[]PaymentMethod{}

	err := cart.APIClient.GetRouteAndDecode(endpoint, paymentMethods, "estimate payment methods for cart")
	if err != nil {
		return *paymentMethods, fmt.Errorf("error estimating payment methods: %w", err)
	}
	return *paymentMethods, nil
}

func (cart *MCart) CreateOrder(paymentMethod PaymentMethod) (*MOrder, error) {
	endpoint := cart.Route + cartPlaceOrder

	type PayLoad struct {
		PaymentMethod PaymentMethodCode `json:"paymentMethod"`
	}

	payLoad := &PayLoad{
		PaymentMethod: PaymentMethodCode{
			Method: paymentMethod.Code,
		},
	}

	body, err := cart.APIClient.v1(context.Background(), http.MethodPut, endpoint, payLoad, nil, "create order")
	if err != nil {
		return nil, err
	}

	orderIDString := mayTrimSurroundingQuotes(string(body))
	orderIDInt, err := strconv.Atoi(orderIDString)
	if err != nil {
		return nil, fmt.Errorf("unexpected error while extracting orderID: %w", err)
	}

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
	_, err := cart.APIClient.v1(context.Background(), http.MethodDelete, endpoint, nil, nil, fmt.Sprintf("delete itemID '%d'", itemID))
	return err
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

	return nil
}
