package magento2

import (
	"context"
	"net/url"
)

const (
	storeStoreViews = "/store/storeViews" // GET module-store/etc/webapi.xml:11
	storeWebsites   = "/store/websites"   // GET module-store/etc/webapi.xml:27

	routeStoreConfigs = "/V1/store/storeConfigs" // GET module-store/etc/webapi.xml:35
	routeStoreGroups  = "/V1/store/storeGroups"  // GET module-store/etc/webapi.xml:19
)

// StoreView is a Magento store view (GET /store/storeViews).
type StoreView struct {
	ID           int    `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	WebsiteID    int    `json:"website_id"`
	StoreGroupID int    `json:"store_group_id"`
}

// Website is a Magento website (GET /store/websites).
type Website struct {
	ID             int    `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	DefaultGroupID int    `json:"default_group_id"`
}

// GetStoreViews returns all store views (GET /store/storeViews).
func GetStoreViews(ctx context.Context, c *Client) ([]StoreView, error) {
	var storeViews []StoreView
	if err := c.GetRouteAndDecodeCtx(ctx, storeStoreViews, &storeViews, "get store views"); err != nil {
		return nil, err
	}
	return storeViews, nil
}

// GetWebsites returns all websites (GET /store/websites).
func GetWebsites(ctx context.Context, c *Client) ([]Website, error) {
	var websites []Website
	if err := c.GetRouteAndDecodeCtx(ctx, storeWebsites, &websites, "get websites"); err != nil {
		return nil, err
	}
	return websites, nil
}

// GetStoreConfigs returns the configuration (locale, currencies, base URLs)
// of the given store views, or of all of them when storeCodes is empty
// (GET /V1/store/storeConfigs?storeCodes[]=...).
func GetStoreConfigs(ctx context.Context, c *Client, storeCodes ...string) ([]StoreConfigEntry, error) {
	var q url.Values
	if len(storeCodes) > 0 {
		q = url.Values{"storeCodes[]": storeCodes}
	}
	var out []StoreConfigEntry
	if err := c.DoJSON(ctx, Request{Path: routeStoreConfigs, Query: q}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetStoreGroups returns all store groups (GET /V1/store/storeGroups).
func GetStoreGroups(ctx context.Context, c *Client) ([]StoreGroup, error) {
	var out []StoreGroup
	if err := c.DoJSON(ctx, Request{Path: routeStoreGroups}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
