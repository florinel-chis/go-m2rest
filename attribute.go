package magento2

import (
	"context" // Added context
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// Assuming these constants are defined elsewhere
var (
	productsAttribute        = "/products/attributes"       // Used in CreateAttribute, GetAttributeByAttributeCode
	productsAttributeOptions = "options"                    // Used in AddOption
)
// Assuming payload structs like createAttributePayload, addOptionPayload are defined

type MAttribute struct {
	Route     string
	Attribute *Attribute
	APIClient *Client
}

// CreateAttribute now accepts context.Context
func CreateAttribute(ctx context.Context, a *Attribute, apiClient *Client) (*MAttribute, error) {
	mAttribute := &MAttribute{
		Attribute: &Attribute{},
		APIClient: apiClient,
	}
	endpoint := productsAttribute
	// httpClient := apiClient.HTTPClient // Not needed directly

	payLoad := createAttributePayload{Attribute: *a} // Assuming createAttributePayload defined

	log.Debug().Interface("payload", payLoad).Str("endpoint", endpoint).Msg("Creating attribute with context")

	err := apiClient.PostRouteAndDecode(ctx, endpoint, payLoad, mAttribute.Attribute, "create attribute")
	if err != nil {
		return mAttribute, err // Return mAttribute for inspection even on partial failure
	}
	mAttribute.Route = productsAttribute + "/" + mAttribute.Attribute.AttributeCode
	return mAttribute, nil
}

// GetAttributeByAttributeCode now accepts context.Context
func GetAttributeByAttributeCode(ctx context.Context, attributeCode string, apiClient *Client) (*MAttribute, error) {
	mAttribute := &MAttribute{ // Corrected variable name from mAttributeSet
		Route:     fmt.Sprintf("%s/%s", productsAttribute, attributeCode),
		Attribute: &Attribute{},
		APIClient: apiClient,
	}

	log.Debug().Str("attributeCode", attributeCode).Str("route", mAttribute.Route).Msg("Getting attribute by attribute code with context")

	err := mAttribute.UpdateAttributeFromRemote(ctx) // Pass context
	if err != nil {
		return nil, fmt.Errorf("error updating attribute from remote when getting by code: %w", err)
	}
	return mAttribute, nil
}

// UpdateAttributeOnRemote now accepts context.Context
func (mas *MAttribute) UpdateAttributeOnRemote(ctx context.Context) error {
	log.Debug().Str("route", mas.Route).Interface("attribute", mas.Attribute).Msg("Updating attribute on remote with context")

	// This is a PUT request. Using direct client.
	resp, err := mas.APIClient.HTTPClient.R().SetContext(ctx).SetResult(mas.Attribute).SetBody(mas.Attribute).Put(mas.Route)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Error().Err(err).Msg("Error updating attribute on remote")
		return fmt.Errorf("error updating attribute on remote: %w", err)
	}

	log.Debug().Int("status", resp.StatusCode()).Str("body", resp.String()).Msg("Attribute update response from remote")
	return mayReturnErrorForHTTPResponse(resp, "update remote attribute from local")
}

// UpdateAttributeFromRemote now accepts context.Context
func (mas *MAttribute) UpdateAttributeFromRemote(ctx context.Context) error {
	log.Debug().Str("route", mas.Route).Msg("Updating attribute from remote with context")

	err := mas.APIClient.GetRouteAndDecode(ctx, mas.Route, mas.Attribute, "update local attribute from remote")
	if err != nil {
		return err
	}
	return nil
}

// AddOption now accepts context.Context
func (mas *MAttribute) AddOption(ctx context.Context, option Option) (string, error) {
	endpoint := mas.Route + "/" + productsAttributeOptions
	// httpClient := mas.APIClient.HTTPClient // Not needed directly

	payLoad := addOptionPayload{Option: option} // Assuming addOptionPayload defined

	log.Debug().Str("endpoint", endpoint).Interface("payload", payLoad).Interface("option", option).Msg("Adding option to attribute with context")

	// This is a POST request. Use PostRouteAndDecode.
	// The original code expects a string response, PostRouteAndDecode needs a target struct/map.
	// Let's assume PostRouteAndDecode can handle a *string or a struct that captures the ID.
	// For now, if PostRouteAndDecode expects a struct, this might need a specific response type.
	// The original code takes resp.String(). Forcing this into PostRouteAndDecode is tricky.
	// Fallback to direct client usage for this specific case due to string response.

	resp, err := mas.APIClient.HTTPClient.R().SetContext(ctx).SetBody(payLoad).Post(endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		log.Error().Err(err).Msg("Error adding option to attribute")
		return "", fmt.Errorf("error assigning option to attribute: %w", err)
	}

	httpErr := mayReturnErrorForHTTPResponse(resp, "assign option to attribute")
	if httpErr != nil {
		return "", httpErr
	}

	optionValue := mayTrimSurroundingQuotes(resp.String())
	// The original code also had: optionValue = strings.TrimPrefix(optionValue, "id_")
	// This suggests the response is not JSON but a raw string like "id_123" or "123".
	// PostRouteAndDecode is not suitable for non-JSON responses. Direct usage is correct here.
	optionValue = strings.TrimPrefix(optionValue, "id_")


	log.Debug().Str("optionValue", optionValue).Msg("Option added successfully, updating attribute from remote")
	err = mas.UpdateAttributeFromRemote(ctx) // Pass context
	if err != nil {
		log.Error().Err(err).Msg("Error updating attribute from remote after adding option")
		// Return optionValue still, as the option might have been added even if subsequent update fails.
		return optionValue, fmt.Errorf("error updating attribute from remote after adding option: %w", err)
	}
	return optionValue, nil
}
