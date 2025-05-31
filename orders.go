package magento2

import (
	"context" // Added context
	"fmt"

	"github.com/rs/zerolog/log"
)

// Assuming OrderComments is defined elsewhere (e.g., order_routes.go)
// var Orders = "/orders" // Already defined in cart.go's subtask, ensure single definition
var (
	OrderComments = "comments" // Relative path for order comments
)


type MOrder struct {
	Route     string
	Order     *Order
	APIClient *Client
}

// GetOrderByIncrementID now accepts context.Context
func GetOrderByIncrementID(ctx context.Context, id string, apiClient *Client) (*MOrder, error) {
	mOrder := &MOrder{
		Route:     "", // Will be set after entity_id is found and order updated
		Order:     &Order{},
		APIClient: apiClient,
	}

	searchCriteria := []SearchQueryCriteria{ // Assuming SearchQueryCriteria, FilterFields, Filter are defined
		{
			Fields: []FilterFields{
				{
					Field:         Filter{FilterGroups: 2, Filters: 0, FilterFor: "increment_id"},
					Value:         Filter{FilterGroups: 2, Filters: 0, FilterFor: id},
					ConditionType: Filter{FilterGroups: 2, Filters: 0, FilterFor: "eq"},
				},
			},
		},
	}
	additionalQuery := Fields{Key: "fields", Value: "items[entity_id]"} // Assuming Fields struct

	searchQueryStr := BuildFlexibleSearchQuery(searchCriteria, additionalQuery) // Assuming BuildFlexibleSearchQuery

	type searchResponse struct {
		Items []struct {
			EntityID int `json:"entity_id"`
		}
	}
	response := &searchResponse{Items: []struct {EntityID int `json:"entity_id"`}{}}
	endpoint := Orders + "?" + searchQueryStr

	log.Debug().Str("incrementID", id).Str("endpoint", endpoint).Msg("Getting order by increment ID with context")

	err := apiClient.GetRouteAndDecode(ctx, endpoint, response, "get order by increment_id from remote")
	if err != nil {
		return nil, err // Error already wrapped
	}

	if len(response.Items) == 0 {
		log.Warn().Str("incrementID", id).Msg("Order not found by increment ID")
		return nil, ErrNotFound
	}

	mOrder.Order.EntityID = response.Items[0].EntityID
	// Construct the route for this specific order
	mOrder.Route = fmt.Sprintf("%s/%d", Orders, mOrder.Order.EntityID)

	err = mOrder.UpdateFromRemote(ctx) // Pass context
	if err != nil {
		return mOrder, fmt.Errorf("error updating order from remote after getting by increment ID: %w", err)
	}
	return mOrder, nil
}

// UpdateEntity now accepts context.Context
func (mo *MOrder) UpdateEntity(ctx context.Context, order *Order) error {
	type updateOrderEntityPayload struct {
		Entity Order `json:"entity"`
	}
	order.EntityID = mo.Order.EntityID // Ensure correct ID
	payLoad := updateOrderEntityPayload{Entity: *order}

	log.Debug().Int("orderID", mo.Order.EntityID).Str("endpoint", Orders).Interface("payload", payLoad).Msg("Updating order entity with context")

	// This is a POST request to the base Orders endpoint (Magento's way of updating orders)
	// Pass mo.Order as target to update its fields from response if API returns the full order.
	err := mo.APIClient.PostRouteAndDecode(ctx, Orders, payLoad, mo.Order, "update order entity on remote")
	if err != nil {
		return err
	}
	return nil
}

// UpdateFromRemote now accepts context.Context
func (mo *MOrder) UpdateFromRemote(ctx context.Context) error {
	if mo.Route == "" && mo.Order != nil && mo.Order.EntityID != 0 { // Defensive: ensure route is set
		mo.Route = fmt.Sprintf("%s/%d", Orders, mo.Order.EntityID)
	} else if mo.Route == "" {
		return fmt.Errorf("order route is not set, cannot update from remote")
	}

	log.Debug().Str("route", mo.Route).Msg("Updating order details from remote with context")
	err := mo.APIClient.GetRouteAndDecode(ctx, mo.Route, mo.Order, "get detailed order object from magento2-api")
	if err != nil {
		// GetRouteAndDecode already logs and wraps, but specific log here is fine too.
		log.Error().Err(err).Str("route", mo.Route).Msg("Error response from GetRouteAndDecode for order update")
		return err // Error already wrapped
	}
	log.Debug().Interface("order", mo.Order).Msg("Order details updated from remote successfully")
	return nil
}

// AddComment now accepts context.Context
func (mo *MOrder) AddComment(ctx context.Context, comment *StatusHistory) (StatusHistory, error) {
	if mo.Route == "" && mo.Order != nil && mo.Order.EntityID != 0 { // Defensive: ensure route is set
		mo.Route = fmt.Sprintf("%s/%d", Orders, mo.Order.EntityID)
	} else if mo.Route == "" {
		return StatusHistory{}, fmt.Errorf("order route is not set, cannot add comment")
	}
	endpoint := mo.Route + "/" + OrderComments

	type PayLoad struct { // Renamed from StatusHistory to avoid conflict if StatusHistory is also a type name
		StatusHistoryPayload StatusHistory `json:"statusHistory"`
	}
	payLoad := &PayLoad{StatusHistoryPayload: *comment}
	response := StatusHistory{} // To capture the response of the created comment

	log.Debug().Int("orderID", mo.Order.EntityID).Str("endpoint", endpoint).Interface("payload", payLoad).Interface("comment", comment).Msg("Adding comment to order with context")

	err := mo.APIClient.PostRouteAndDecode(ctx, endpoint, payLoad, &response, "add comment to order")
	if err != nil {
		// PostRouteAndDecode logs and wraps.
		return response, err // Return potentially partially filled response and error
	}
	log.Debug().Interface("commentResponse", response).Msg("Comment added to order successfully")
	return response, nil
}
