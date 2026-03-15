package pinergy

import "context"

// GetBalance returns the current credit balance and related information.
// Results are cached for 60 seconds by default.
//
// Call [Client.CacheInvalidate]("/api/balance/") or [Client.CacheFlush]
// after a top-up to force a fresh read.
func (c *Client) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out BalanceResponse
	if err := c.fetch(ctx, "/api/balance/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
