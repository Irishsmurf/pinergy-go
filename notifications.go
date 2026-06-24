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

// UpdateNotificationPreferences updates the account's notification channel
// preferences and returns the resulting state. The notification cache is
// invalidated after a successful update.
func (c *Client) UpdateNotificationPreferences(ctx context.Context, sms, email, phone bool) (*NotificationResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}
	reqBody := UpdateNotificationPreferencesRequest{
		SMS:   sms,
		Email: email,
		Phone: phone,
	}
	var out NotificationResponse
	if err := c.post(ctx, "/api/updatenotif/", reqBody, &out); err != nil {
		return nil, err
	}
	c.cache.Invalidate("/api/getnotif/")
	return &out, nil
}
