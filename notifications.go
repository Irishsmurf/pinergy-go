package pinergy

import "context"

// GetNotifications returns the current notification preferences for the account.
// Results are cached for 5 minutes by default.
func (c *Client) GetNotifications(ctx context.Context) (*NotificationResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	var out NotificationResponse
	if err := c.fetch(ctx, "/api/getnotif/", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
