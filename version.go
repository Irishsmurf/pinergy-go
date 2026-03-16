package pinergy

import "context"

// GetVersion returns the minimum required and current app version information.
// This endpoint does not require authentication.
// Results are cached for 10 minutes by default.
func (c *Client) GetVersion(ctx context.Context) (*VersionResponse, error) {
	var out VersionResponse
	if err := c.fetchDirect(ctx, "/version.json", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
