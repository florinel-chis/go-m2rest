package magento2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type MAttribute struct {
	Route     string
	Attribute *Attribute
	APIClient *Client
}

func CreateAttribute(a *Attribute, apiClient *Client) (*MAttribute, error) {
	mAttribute := &MAttribute{
		Attribute: &Attribute{},
		APIClient: apiClient,
	}

	payLoad := createAttributePayload{
		Attribute: *a,
	}

	_, err := apiClient.v1(context.Background(), http.MethodPost, productsAttribute, payLoad, mAttribute.Attribute, "create attribute")
	mAttribute.Route = productsAttribute + "/" + url.PathEscape(mAttribute.Attribute.AttributeCode)
	if err != nil {
		return mAttribute, err
	}

	return mAttribute, nil
}

func GetAttributeByAttributeCode(attributeCode string, apiClient *Client) (*MAttribute, error) {
	mAttribute := &MAttribute{
		Route:     productsAttribute + "/" + url.PathEscape(attributeCode),
		Attribute: &Attribute{},
		APIClient: apiClient,
	}

	err := mAttribute.UpdateAttributeFromRemote()
	if err != nil {
		return nil, fmt.Errorf("error updating attribute from remote when getting by code: %w", err)
	}

	return mAttribute, nil
}

func (mas *MAttribute) UpdateAttributeOnRemote() error {
	_, err := mas.APIClient.v1(context.Background(), http.MethodPut, mas.Route, mas.Attribute, mas.Attribute, "update remote attribute from local")
	return err
}

func (mas *MAttribute) UpdateAttributeFromRemote() error {
	_, err := mas.APIClient.v1(context.Background(), http.MethodGet, mas.Route, nil, mas.Attribute, "update local attribute from remote")
	return err
}

func (mas *MAttribute) AddOption(option Option) (string, error) {
	endpoint := mas.Route + "/" + productsAttributeOptions

	payLoad := addOptionPayload{
		Option: option,
	}

	body, err := mas.APIClient.v1(context.Background(), http.MethodPost, endpoint, payLoad, nil, "assign option to attribute")
	if err != nil {
		return "", err
	}

	optionValue := mayTrimSurroundingQuotes(string(body))
	optionValue = strings.TrimPrefix(optionValue, "id_")

	err = mas.UpdateAttributeFromRemote()
	if err != nil {
		return "", fmt.Errorf("error updating attribute from remote after adding option: %w", err)
	}

	return optionValue, nil
}
