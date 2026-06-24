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

// TopUp initiates an instant top-up using a saved payment card. The ccToken
// is obtained from [CreditCard.Token] in the [LoginResponse]. On success
// the balance and active top-ups caches are invalidated.
func (c *Client) TopUp(ctx context.Context, amount int, ccToken string) (*TopUpResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	reqBody := TopUpRequest{
		Amount:  amount,
		CCToken: ccToken,
	}
	var out TopUpResponse
	if err := c.post(ctx, "/api/topup/", reqBody, &out); err != nil {
		return nil, err
	}
	c.cache.Invalidate("/api/balance/")
	c.cache.Invalidate("/api/activetopups/")
	return &out, nil
}
