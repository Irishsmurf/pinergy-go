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

// UpdateDeviceToken registers or updates the Firebase Cloud Messaging (FCM)
// token used for push notifications. For headless or server-side clients,
// pass an empty string for deviceToken.
//
// deviceType should be "android" or "ios".
// osVersion is a free-form string, e.g. "Android SDK: 33 (13)".
func (c *Client) UpdateDeviceToken(ctx context.Context, deviceToken, deviceType, osVersion string) error {
	if err := c.requireAuth(); err != nil {
		return err
	}
	body := UpdateDeviceTokenRequest{
		DeviceToken: deviceToken,
		DeviceType:  deviceType,
		OSVersion:   osVersion,
	}
	return c.post(ctx, "/api/updatedevicetoken/", body, nil)
}
