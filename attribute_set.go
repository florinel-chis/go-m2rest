package magento2

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

	payLoad := createAttributeSetPayload{
		AttributeSet: a,
		SkeletonID:   skeletonID,
	}

	_, err := apiClient.v1(context.Background(), http.MethodPost, productsAttributeSet, payLoad, mAttributeSet.AttributeSet, "create attribute-set")
	mAttributeSet.Route = productsAttributeSet + "/" + strconv.Itoa(mAttributeSet.AttributeSet.AttributeSetID)
	if err != nil {
		return mAttributeSet, err
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

	response := &attributeSetSearchQueryResponse{}
	if _, err := apiClient.v1(context.Background(), http.MethodGet, endpoint, nil, response, "get attribute-set by name from remote"); err != nil {
		return nil, err
	}

	if len(response.AttributeSets) == 0 {
		return nil, ErrNotFound
	}

	mAttributeSet.AttributeSet = &response.AttributeSets[0]
	mAttributeSet.Route = productsAttributeSet + "/" + strconv.Itoa(mAttributeSet.AttributeSet.AttributeSetID)

	err := mAttributeSet.UpdateAttributeSetFromRemote()
	if err != nil {
		return mAttributeSet, fmt.Errorf("error updating attribute set from remote after getting by name: %w", err)
	}

	return mAttributeSet, nil
}

func (mas *MAttributeSet) UpdateAttributeSetOnRemote() error {
	_, err := mas.APIClient.v1(context.Background(), http.MethodPut, mas.Route, mas.AttributeSet, mas.AttributeSet, "update remote attribute-set from local")
	return err
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

func (mas *MAttributeSet) updateAttributeSetDetails() error {
	_, err := mas.APIClient.v1(context.Background(), http.MethodGet, mas.Route, nil, mas.AttributeSet, "get details for attribute-set from remote")
	return err
}

func (mas *MAttributeSet) updateAttributes() error {
	attributesRoute := mas.Route + "/" + productsAttributeSetAttributesRelative
	_, err := mas.APIClient.v1(context.Background(), http.MethodGet, attributesRoute, nil, mas.AttributeSetAttributes, "get attributes for attribute-set from remote")
	return err
}

func (mas *MAttributeSet) updateGroups() error {
	searchQuery := BuildSearchQuery("attribute_set_id", strconv.Itoa(mas.AttributeSet.AttributeSetID), "in")
	endpoint := productsAttributeSetGroupsList + "?" + searchQuery

	response := &groupSearchQueryResponse{}
	if _, err := mas.APIClient.v1(context.Background(), http.MethodGet, endpoint, nil, response, "get groups for attribute-set from remote"); err != nil {
		return err
	}

	mas.AttributeSetGroups = response.Groups
	return nil
}

func (mas *MAttributeSet) AssignAttribute(attributeGroupID, sortOrder int, attributeCode string) error {
	payLoad := assignAttributePayload{
		AttributeSetID:      mas.AttributeSet.AttributeSetID,
		AttributeSetGroupID: attributeGroupID,
		AttributeCode:       attributeCode,
		SortOrder:           sortOrder,
	}

	if _, err := mas.APIClient.v1(context.Background(), http.MethodPost, productsAttributeSetAttributes, payLoad, nil, "assign attribute to attribute-set"); err != nil {
		return err
	}

	err := mas.UpdateAttributeSetFromRemote()
	if err != nil {
		return fmt.Errorf("error updating attribute set from remote after assigning attribute: %w", err)
	}
	return nil
}

func (mas *MAttributeSet) CreateGroup(groupName string) error {
	payLoad := createGroupPayload{
		Group: Group{
			AttributeGroupName: groupName,
			AttributeSetID:     mas.AttributeSet.AttributeSetID,
		},
	}

	if _, err := mas.APIClient.v1(context.Background(), http.MethodPost, productsAttributeSetGroups, payLoad, nil, "create group on attribute-set"); err != nil {
		return err
	}

	err := mas.UpdateAttributeSetFromRemote()
	if err != nil {
		return fmt.Errorf("error updating attribute set from remote after creating group: %w", err)
	}
	return nil
}
