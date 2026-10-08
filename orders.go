package magento2

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

	err := apiClient.GetRouteAndDecode(endpoint, response, "get order by increment_id from remote")
	if err != nil {
		return nil, fmt.Errorf("error getting order by increment ID from remote: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, ErrNotFound
	}

	mOrder.Order.EntityID = response.Items[0].EntityID
	mOrder.Route = Orders + "/" + strconv.Itoa(mOrder.Order.EntityID)
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

	_, err := mo.APIClient.v1(context.Background(), http.MethodPost, Orders, payLoad, mo.Order, "update order entity on remote")
	return err
}

func (mo *MOrder) UpdateFromRemote() error {
	err := mo.APIClient.GetRouteAndDecode(mo.Route, mo.Order, "get detailed order object from magento2-api")
	if err != nil {
		return fmt.Errorf("error updating order details from remote: %w", err)
	}
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

	response := StatusHistory{}

	err := mo.APIClient.PostRouteAndDecode(endpoint, payLoad, &response, "add comment to order")
	if err != nil {
		return response, fmt.Errorf("error adding comment to order: %w", err)
	}
	return response, nil
}
