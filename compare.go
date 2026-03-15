package pinergy

import "context"

// GetCompare returns daily, weekly, and monthly energy usage for the user's
// home compared against similar homes on the Pinergy network.
// Results are cached for 15 minutes by default.
func (c *Client) GetCompare(ctx context.Context) (*CompareResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out CompareResponse
	if err := c.fetch(ctx, "/api/compare/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
