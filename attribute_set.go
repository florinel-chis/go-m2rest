package magento2

import (
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

type MAttributeSet struct {
	Route                  string
	AttributeSet           *AttributeSet
	AttributeSetGroups     []Group
	AttributeSetAttributes *[]Attribute
	APIClient              *Client
}

func CreateAttributeSet(a AttributeSet, skeletonID int, apiClient *Client) (*MAttributeSet, error) {
	mAttributeSet := &MAttributeSet{
		AttributeSet:           &AttributeSet{},
		AttributeSetAttributes: &[]Attribute{},
		APIClient:              apiClient,
	}
	endpoint := productsAttributeSet
	httpClient := apiClient.HTTPClient

	payLoad := createAttributeSetPayload{
		AttributeSet: a,
		SkeletonID:   skeletonID,
	}

	log.Debug().
		Interface("payload", payLoad).
		Int("skeletonID", skeletonID).
		Str("endpoint", endpoint).
		Msg("Creating attribute set")

	resp, err := httpClient.R().SetBody(payLoad).SetResult(mAttributeSet.AttributeSet).Post(endpoint)
	mAttributeSet.Route = productsAttributeSet + "/" + strconv.Itoa(mAttributeSet.AttributeSet.AttributeSetID)

	if err != nil {
		return mAttributeSet, fmt.Errorf("error creating attribute set: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "create attribute-set")
	if httpErr != nil {
		return mAttributeSet, httpErr
	}

	err = mAttributeSet.UpdateAttributeSetFromRemote()
	if err != nil {
		return mAttributeSet, fmt.Errorf("error updating attribute set from remote after creation: %w", err)
	}

	return mAttributeSet, nil
}

func GetAttributeSetByName(name string, apiClient *Client) (*MAttributeSet, error) {
	mAttributeSet := &MAttributeSet{
		AttributeSet:           &AttributeSet{},
		AttributeSetAttributes: &[]Attribute{},
		APIClient:              apiClient,
	}
	searchQuery := BuildSearchQuery("attribute_set_name", name, "in")
	endpoint := productsAttributeSetList + "?" + searchQuery
	httpClient := apiClient.HTTPClient

	response := &attributeSetSearchQueryResponse{}

	log.Debug().
		Str("name", name).
		Str("endpoint", endpoint).
		Msg("Getting attribute set by name")

	resp, err := httpClient.R().SetResult(response).Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting attribute set by name: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get attribute-set by name from remote")
	if httpErr != nil {
		return nil, httpErr
	}

	if len(response.AttributeSets) == 0 {
		log.Warn().Str("name", name).Msg("Attribute set not found by name")
		return nil, ErrNotFound
	}

	mAttributeSet.AttributeSet = &response.AttributeSets[0]
	mAttributeSet.Route = productsAttributeSet + "/" + strconv.Itoa(mAttributeSet.AttributeSet.AttributeSetID)

	err = mAttributeSet.UpdateAttributeSetFromRemote()
	if err != nil {
		return mAttributeSet, fmt.Errorf("error updating attribute set from remote after getting by name: %w", err)
	}
	httpErr = mayReturnErrorForHTTPResponse(resp, "get detailed attribute-set by name from remote")
	if httpErr != nil {
		return mAttributeSet, httpErr
	}

	return mAttributeSet, nil
}

func (mas *MAttributeSet) UpdateAttributeSetOnRemote() error {
	log.Debug().
		Str("route", mas.Route).
		Interface("attributeSet", mas.AttributeSet).
		Msg("Updating attribute set on remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(mas.AttributeSet).SetBody(mas.AttributeSet).Put(mas.Route)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute set on remote")
		return fmt.Errorf("error updating attribute set on remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute set updated response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "update remote attribute-set from local")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mas *MAttributeSet) UpdateAttributeSetFromRemote() error {
	err := mas.updateAttributeSetDetails()
	if err != nil {
		return fmt.Errorf("error updating attribute set details: %w", err)
	}

	err = mas.updateGroups()
	if err != nil {
		return fmt.Errorf("error updating attribute set groups: %w", err)
	}

	err = mas.updateAttributes()
	if err != nil {
		return fmt.Errorf("error updating attribute set attributes: %w", err)
	}

	return nil
}

// updateAttributeSetDetails - Renamed to be more specific about what is being updated (details)
func (mas *MAttributeSet) updateAttributeSetDetails() error {
	log.Debug().
		Str("route", mas.Route).
		Msg("Updating attribute set details from remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(mas.AttributeSet).Get(mas.Route)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute set details from remote")
		return fmt.Errorf("error getting attribute set details from remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute set details updated response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "get details for attribute-set from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mas *MAttributeSet) updateAttributes() error {
	attributesRoute := mas.Route + "/" + productsAttributeSetAttributesRelative
	log.Debug().
		Str("route", attributesRoute).
		Msg("Updating attribute set attributes from remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(mas.AttributeSetAttributes).Get(attributesRoute)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute set attributes from remote")
		return fmt.Errorf("error getting attribute set attributes from remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute set attributes updated response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "get attributes for attribute-set from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mas *MAttributeSet) updateGroups() error {
	searchQuery := BuildSearchQuery("attribute_set_id", strconv.Itoa(mas.AttributeSet.AttributeSetID), "in")
	endpoint := productsAttributeSetGroupsList + "?" + searchQuery

	response := &groupSearchQueryResponse{}

	log.Debug().
		Str("endpoint", endpoint).
		Msg("Updating attribute set groups from remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(response).Get(endpoint)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute set groups from remote")
		return fmt.Errorf("error getting attribute set groups from remote: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "get groups for attribute-set from remote")
	if httpErr != nil {
		return httpErr
	}

	mas.AttributeSetGroups = response.Groups
	log.Debug().Interface("groups", mas.AttributeSetGroups).Msg("Attribute set groups updated successfully")

	return nil
}

func (mas *MAttributeSet) AssignAttribute(attributeGroupID, sortOrder int, attributeCode string) error {
	endpoint := productsAttributeSetAttributes
	httpClient := mas.APIClient.HTTPClient

	payLoad := assignAttributePayload{
		AttributeSetID:      mas.AttributeSet.AttributeSetID,
		AttributeSetGroupID: attributeGroupID,
		AttributeCode:       attributeCode,
		SortOrder:           sortOrder,
	}

	log.Debug().
		Str("attributeCode", attributeCode).
		Int("attributeSetID", mas.AttributeSet.AttributeSetID).
		Int("groupID", attributeGroupID).
		Int("sortOrder", sortOrder).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Assigning attribute to attribute set")

	resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)
	if err != nil {
		log.Error().Err(err).Msg("Error assigning attribute to attribute set")
		return fmt.Errorf("error assigning attribute to attribute set: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "assign attribute to attribute-set")
	if httpErr != nil {
		return httpErr
	}

	log.Debug().Msg("Attribute assigned successfully, updating attribute set from remote")
	err = mas.UpdateAttributeSetFromRemote()
	if err != nil {
		return fmt.Errorf("error updating attribute set from remote after assigning attribute: %w", err)
	}
	return nil
}

func (mas *MAttributeSet) CreateGroup(groupName string) error {
	endpoint := productsAttributeSetGroups
	httpClient := mas.APIClient.HTTPClient

	payLoad := createGroupPayload{
		Group: Group{
			AttributeGroupName: groupName,
			AttributeSetID:     mas.AttributeSet.AttributeSetID,
		},
	}

	log.Debug().
		Str("groupName", groupName).
		Int("attributeSetID", mas.AttributeSet.AttributeSetID).
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Msg("Creating attribute group for attribute set")

	resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)
	if err != nil {
		log.Error().Err(err).Msg("Error creating attribute group for attribute set")
		return fmt.Errorf("error creating group on attribute-set: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "create group on attribute-set")
	if httpErr != nil {
		return httpErr
	}

	log.Debug().Msg("Attribute group created successfully, updating attribute set from remote")
	err = mas.UpdateAttributeSetFromRemote()
	if err != nil {
		return fmt.Errorf("error updating attribute set from remote after creating group: %w", err)
	}
	return nil
}

// --- Helper Functions (Potentially in a separate util file) ---

// BuildSearchQuery is assumed to be defined elsewhere and is not modified as part of the logging refactor.
// mayReturnErrorForHTTPResponse is assumed to be defined elsewhere.
// You should refactor mayReturnErrorForHTTPResponse to use zerolog if it also does logging.

// Example of how mayReturnErrorForHTTPResponse might be refactored to use zerolog:
/*
func mayReturnErrorForHTTPResponse(resp *resty.Response, operation string) error {
	if resp.IsError() {
		log.Error().
			Int("status_code", resp.StatusCode()).
			Str("operation", operation).
			Str("body", resp.String()).
			Msg("HTTP error during operation")
		return fmt.Errorf("http error during %s, status code: %d, body: %s", operation, resp.StatusCode(), resp.String())
	}
	return nil
}
*/
