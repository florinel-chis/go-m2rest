package magento2

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
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
	endpoint := productsAttribute
	httpClient := apiClient.HTTPClient

	payLoad := createAttributePayload{
		Attribute: *a,
	}

	log.Debug().
		Interface("payload", payLoad).
		Str("endpoint", endpoint).
		Msg("Creating attribute")

	resp, err := httpClient.R().SetBody(payLoad).SetResult(mAttribute.Attribute).Post(endpoint)
	mAttribute.Route = productsAttribute + "/" + mAttribute.Attribute.AttributeCode

	if err != nil {
		log.Error().Err(err).Msg("Error creating attribute")
		return mAttribute, fmt.Errorf("error creating attribute: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute creation response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "create attribute")
	if httpErr != nil {
		return mAttribute, httpErr
	}

	return mAttribute, nil
}

func GetAttributeByAttributeCode(attributeCode string, apiClient *Client) (*MAttribute, error) {
	mAttributeSet := &MAttribute{ // Note: variable name was mAttributeSet, corrected to mAttribute for consistency
		Route:     fmt.Sprintf("%s/%s", productsAttribute, attributeCode),
		Attribute: &Attribute{},
		APIClient: apiClient,
	}

	log.Debug().
		Str("attributeCode", attributeCode).
		Str("route", mAttributeSet.Route). // Added route to debug log
		Msg("Getting attribute by attribute code")

	err := mAttributeSet.UpdateAttributeFromRemote()
	if err != nil {
		return nil, fmt.Errorf("error updating attribute from remote when getting by code: %w", err)
	}

	return mAttributeSet, nil
}

func (mas *MAttribute) UpdateAttributeOnRemote() error {
	log.Debug().
		Str("route", mas.Route).
		Interface("attribute", mas.Attribute).
		Msg("Updating attribute on remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(mas.Attribute).SetBody(mas.Attribute).Put(mas.Route)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute on remote")
		return fmt.Errorf("error updating attribute on remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute update response from remote")

	httpErr := mayReturnErrorForHTTPResponse(resp, "update remote attribute from local")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mas *MAttribute) UpdateAttributeFromRemote() error {
	log.Debug().
		Str("route", mas.Route).
		Msg("Updating attribute from remote")

	resp, err := mas.APIClient.HTTPClient.R().SetResult(mas.Attribute).Get(mas.Route)
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute from remote")
		return fmt.Errorf("error updating attribute from remote: %w", err)
	}

	log.Debug().
		Int("status", resp.StatusCode()).
		Str("body", resp.String()).
		Msg("Attribute update from remote response")

	httpErr := mayReturnErrorForHTTPResponse(resp, "update local attribute from remote")
	if httpErr != nil {
		return httpErr
	}
	return nil
}

func (mas *MAttribute) AddOption(option Option) (string, error) {
	endpoint := mas.Route + "/" + productsAttributeOptions
	httpClient := mas.APIClient.HTTPClient

	payLoad := addOptionPayload{
		Option: option,
	}

	log.Debug().
		Str("endpoint", endpoint).
		Interface("payload", payLoad).
		Interface("option", option). // Added logging for the option itself
		Msg("Adding option to attribute")

	resp, err := httpClient.R().SetBody(payLoad).Post(endpoint)
	if err != nil {
		log.Error().Err(err).Msg("Error adding option to attribute")
		return "", fmt.Errorf("error assigning option to attribute: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "assign option to attribute")
	if httpErr != nil {
		return "", httpErr
	}

	optionValue := mayTrimSurroundingQuotes(resp.String())
	optionValue = strings.TrimPrefix(optionValue, "id_")

	log.Debug().
		Str("optionValue", optionValue).
		Msg("Option added successfully, updating attribute from remote")

	err = mas.UpdateAttributeFromRemote()
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute from remote after adding option")
		return "", fmt.Errorf("error updating attribute from remote after adding option: %w", err)
	}

	return optionValue, nil
}
