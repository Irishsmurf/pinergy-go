package pinergy

import "context"

// GetConfigInfo returns the valid top-up amounts and credit alert thresholds
// configured for the account.
// Results are cached for 30 minutes by default.
func (c *Client) GetConfigInfo(ctx context.Context) (*ConfigInfoResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out ConfigInfoResponse
	if err := c.fetch(ctx, "/api/configinfo/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDefaultsInfo returns reference data for house types and heating types
// used when updating the account profile.
// Results are cached for 30 minutes by default.
func (c *Client) GetDefaultsInfo(ctx context.Context) (*DefaultsInfoResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out DefaultsInfoResponse
	if err := c.fetch(ctx, "/api/defaultsinfo/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
