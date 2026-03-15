package pinergy

import "context"

// GetUsage returns aggregated energy usage grouped by day (last 7 days),
// week (last 8 weeks), and month (last 11 months).
// Results are cached for 5 minutes by default.
func (c *Client) GetUsage(ctx context.Context) (*UsageResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out UsageResponse
	if err := c.fetch(ctx, "/api/usage/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetLevelPayUsage returns half-hourly interval data for level-pay customers.
// Results are cached for 5 minutes by default.
//
// This endpoint is only relevant when [LoginResponse.IsLevelPay] is true.
// Its response does not use the standard success envelope.
func (c *Client) GetLevelPayUsage(ctx context.Context) (*LevelPayUsageResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out LevelPayUsageResponse
	if err := c.fetchDirect(ctx, "/api/levelpayusage/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
