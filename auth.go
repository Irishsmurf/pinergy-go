package pinergy

import (
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 is mandated by the Pinergy API, not used for security
	"encoding/hex"
	"errors"
	"net/http"
)

// hashPassword returns the lowercase hex-encoded SHA-1 digest of password.
// The Pinergy API requires this transformation before login; the plaintext
// password is never transmitted.
func hashPassword(password string) string {
	// sha1.Sum is faster and allocates less than sha1.New() + Write() + Sum()
	sum := sha1.Sum([]byte(password)) //nolint:gosec
	return hex.EncodeToString(sum[:])
}

// CheckEmail checks whether email is registered in the Pinergy system.
// This endpoint does not require authentication and is typically called
// before presenting the login screen.
//
// Returns [ErrEmailNotFound] if the address is not registered.
func (c *Client) CheckEmail(ctx context.Context, email string) error {
	data, status, err := c.doSimpleGET(ctx, "/api/checkemail",
		func(r *http.Request) { r.Header.Set("email_address", email) },
	)
	if err != nil {
		return err
	}
	if err := checkEnvelope(data, status); err != nil {
		// Map a failure response to ErrEmailNotFound.
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return &APIError{
				Code:       ErrCodeEmailNotFound,
				StatusCode: status,
				Message:    apiErr.Message,
				Err:        apiErr,
			}
		}
		return err
	}
	return nil
}

// Login authenticates the client with the given credentials. The password
// is SHA-1 hashed internally before transmission.
//
// On success the auth token is stored and attached to all subsequent
// authenticated requests. Login is safe to call concurrently but a
// subsequent call replaces the stored token.
func (c *Client) Login(ctx context.Context, email, password string) error {
	reqBody := LoginRequest{
		Email:       email,
		Password:    hashPassword(password),
		DeviceToken: "", // headless — no FCM token needed
	}

	var resp LoginResponse
	if err := c.post(ctx, "/api/login/", reqBody, &resp); err != nil {
		return err
	}

	c.mu.Lock()
	c.authToken = resp.AuthToken
	c.isLevelPay = resp.IsLevelPay
	c.mu.Unlock()

	return nil
}

// IsAuthenticated reports whether the client currently holds a valid auth token.
// A true return does not guarantee the token is still accepted by the server.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authToken != ""
}

// Logout clears the stored auth token. Subsequent calls to authenticated
// endpoints will return [ErrAuthRequired].
func (c *Client) Logout() {
	c.mu.Lock()
	c.authToken = ""
	c.isLevelPay = false
	c.mu.Unlock()
	c.cache.Flush()
}
