package magento2

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type MOrder struct {
	Route     string
	Order     *Order
	APIClient *Client
}

func GetOrderByIncrementID(id string, apiClient *Client) (*MOrder, error) {
	mOrder := &MOrder{
		Route:     "",
		Order:     &Order{},
		APIClient: apiClient,
	}

	// ?searchCriteria[filter_groups][2][filters][0][field]=increment_id
	// &searchCriteria[filter_groups][2][filters][0][value]=INCREMENT_ID_HERE
	// &searchCriteria[filter_groups][2][filters][0][condition_type]=eq
	// &fields=items[entity_id]
	searchCriteria := []SearchQueryCriteria{
		{
			Fields: []FilterFields{
				{
					Field: Filter{
						FilterGroups: 2,
						Filters:      0,
						FilterFor:    "increment_id",
					},
					Value: Filter{
						FilterGroups: 2,
						Filters:      0,
						FilterFor:    id,
					},
					ConditionType: Filter{
						FilterGroups: 2,
						Filters:      0,
						FilterFor:    "eq",
					},
				},
			},
		},
	}

	additionalQuery := Fields{
		Key:   "fields",
		Value: "items[entity_id]",
	}

	searchQuery := BuildFlexibleSearchQuery(searchCriteria, additionalQuery)

	type searchResponse struct {
		Items []struct {
			EntityID int `json:"entity_id"`
		}
	}

	response := &searchResponse{
		Items: []struct {
			EntityID int `json:"entity_id"`
		}{},
	}

	endpoint := Orders + "?" + searchQuery

	log.Debug().
		Str("incrementID", id).
		Str("endpoint", endpoint).
		Msg("Getting order by increment ID")

	err := apiClient.GetRouteAndDecode(endpoint, response, "get order by increment_id from remote")
	if err != nil {
		return nil, fmt.Errorf("error getting order by increment ID from remote: %w", err)
	}

	if len(response.Items) == 0 {
		log.Warn().Str("incrementID", id).Msg("Order not found by increment ID")
		return nil, ErrNotFound
	}

	mOrder.Order.EntityID = response.Items[0].EntityID
	err = mOrder.UpdateFromRemote()
	if err != nil {
		return mOrder, fmt.Errorf("error updating order from remote after getting by increment ID: %w", err)
	}

	return mOrder, nil
}

func (mo *MOrder) UpdateEntity(order *Order) error {
	type updateOrderEntityPayload struct {
		Entity Order `json:"entity"`
	}

	order.EntityID = mo.Order.EntityID

	payLoad := updateOrderEntityPayload{
		Entity: *order,
	}

	log.Debug().
		Int("orderID", mo.Order.EntityID).
		Str("endpoint", Orders).
		Interface("payload", payLoad).
		Msg("Updating order entity")

	resp, err := mo.APIClient.HTTPClient.R().SetResult(mo.Order).SetBody(payLoad).Post(Orders)

	if err != nil {
		log.Error().Err(err).Msg("Error updating order entity")
		return fmt.Errorf("error updating order entity: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Order entity updated response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "update order entity on remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mo *MOrder) UpdateFromRemote() error {
	log.Debug().
		Str("route", mo.Route).
		Msg("Updating order details from remote")

	err := mo.APIClient.GetRouteAndDecode(mo.Route, mo.Order, "get detailed order object from magento2-api")
	if err != nil {
		log.Error().Err(err).Msg("Error updating order details from remote")
		return fmt.Errorf("error updating order details from remote: %w", err)
	}
	log.Debug().Interface("order", mo.Order).Msg("Order details updated from remote successfully")
	return nil
}

func (mo *MOrder) AddComment(comment *StatusHistory) (StatusHistory, error) {
	endpoint := mo.Route + "/" + OrderComments

	type PayLoad struct {
		StatusHistory StatusHistory `json:"statusHistory"`
	}

	payLoad := &PayLoad{
		StatusHistory: *comment,
	}

	log.Debug().
		Int("orderID", mo.Order.EntityID).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Interface("comment", comment). // Log the comment itself
		Msg("Adding comment to order")

	response := StatusHistory{}

	err := mo.APIClient.PostRouteAndDecode(endpoint, payLoad, &response, "add comment to order")
	if err != nil {
		log.Error().Err(err).Msg("Error adding comment to order")
		return response, fmt.Errorf("error adding comment to order: %w", err)
	}
	log.Debug().Interface("commentResponse", response).Msg("Comment added to order successfully")
	return response, nil
}
