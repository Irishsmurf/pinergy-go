package pinergy

import "context"

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
