package magento2

import "context"

const (
	storeStoreViews = "/store/storeViews"
	storeWebsites   = "/store/websites"
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
