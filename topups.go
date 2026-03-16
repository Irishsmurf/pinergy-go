package pinergy

import "context"

// GetActiveTopups returns the scheduled and automatic top-up configurations
// for the account.
// Results are cached for 2 minutes by default.
func (c *Client) GetActiveTopups(ctx context.Context) (*ActiveTopUpsResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out ActiveTopUpsResponse
	if err := c.fetch(ctx, "/api/activetopups/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
