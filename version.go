package pinergy

import (
	"context"
	"errors"
)

// GetVersion returns the minimum required and current app version information.
// This endpoint does not require authentication.
// Results are cached for 10 minutes by default.
func (c *Client) GetVersion(ctx context.Context) (*VersionResponse, error) {
	const path = "/version.json"

	if cached, ok := c.cache.Get(path); ok {
		var out VersionResponse
		if err := decodeJSON(cached, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}

	data, status, err := c.doSimpleGET(ctx, path)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return nil, &APIError{
			Code:       httpStatusToErrCode(status),
			StatusCode: status,
			Message:    "failed to fetch version",
		}
	}

	// /version.json does not use the success envelope.
	var out VersionResponse
	if err := decodeJSON(data, &out); err != nil {
		// Try the envelope check as fallback.
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Code == ErrCodeInvalidResponse {
			if envErr := checkEnvelope(data, status); envErr != nil {
				return nil, envErr
			}
		}
		return nil, err
	}

	c.cache.Set(path, path, data)
	return &out, nil
}
