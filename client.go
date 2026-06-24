package pinergy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"time"
)

// newRequest builds an *http.Request targeting c.baseURL+path. If body is
// non-nil it is JSON-encoded and set as the request body. The auth_token
// header is added when the client holds a token and authenticated is true
// (the default).
func (c *Client) newRequest(ctx context.Context, method, path string, body any, authenticated bool) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, &APIError{Code: ErrCodeUnknown, Message: "failed to encode request body", Err: err}
		}
		bodyReader = bytes.NewReader(b)
	}

	fullURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &APIError{Code: ErrCodeUnknown, Message: "failed to create request", Err: err}
	}

	if !c.allowInsecure && req.URL.Scheme == "http" && !isLoopback(req.URL.Hostname()) {
		return nil, &APIError{
			Code:    ErrCodeUnknown,
			Message: "refusing to send credentials over plaintext HTTP (use WithInsecureHTTP to override)",
		}
	}

	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if authenticated {
		c.mu.RLock()
		token := c.authToken
		c.mu.RUnlock()
		if token != "" {
			req.Header.Set("auth_token", token)
		}
	}

	return req, nil
}

// do dispatches a single HTTP request, waiting for a rate-limiter token first.
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &APIError{Code: ErrCodeContextDeadline, Message: "rate limiter wait exceeded deadline", Err: err}
		}
		return nil, &APIError{Code: ErrCodeContextCanceled, Message: "rate limiter wait canceled", Err: err}
	}
	return c.httpClient.Do(req)
}

// doWithRetry dispatches the request with automatic retry on transient errors.
// The request body (if any) is captured before the loop so it can be replayed.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Buffer the body so we can replay it on retries.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, &APIError{Code: ErrCodeUnknown, Message: "failed to buffer request body", Err: err}
		}
	}

	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		// Respect cancellation between attempts.
		if attempt > 0 {
			delay := backoffDuration(attempt-1, c.retryBaseDelay, c.retryMaxDelay)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return nil, &APIError{Code: ErrCodeContextDeadline, Err: ctx.Err()}
				}
				return nil, &APIError{Code: ErrCodeContextCanceled, Err: ctx.Err()}
			case <-timer.C:
			}
		}

		// Restore the body for this attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err = c.do(ctx, req)
		if !isRetryable(resp, err) {
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
			resp = nil
		}
	}

	return resp, err
}

// readAndClose reads the response body and closes it. If the body exceeds
// limit bytes an error is returned instead of silently truncating.
func readAndClose(resp *http.Response, limit int64) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response body exceeded %d byte limit", limit)
	}
	return data, nil
}

// decodeJSON unmarshals data into dst.
func decodeJSON(data []byte, dst any) error {
	if err := json.Unmarshal(data, dst); err != nil {
		return &APIError{Code: ErrCodeInvalidResponse, Message: "failed to decode response", Err: err}
	}
	return nil
}

// checkEnvelope inspects the success field of the raw JSON. On failure it
// returns an *APIError with the API's message and error code.
func checkEnvelope(data []byte, statusCode int) error {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return &APIError{
			Code:       ErrCodeInvalidResponse,
			StatusCode: statusCode,
			Message:    "could not decode response envelope",
			Err:        err,
		}
	}
	if !env.Success {
		code := httpStatusToErrCode(statusCode)
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("API returned success=false (error_code=%d)", env.ErrorCode)
		}
		return &APIError{Code: code, StatusCode: statusCode, Message: msg}
	}
	return nil
}

// fetch is the canonical helper used by every authenticated GET endpoint:
// check auth → check cache → HTTP → check envelope → cache & decode.
// On 401, it attempts re-authentication and retries once.
func (c *Client) fetch(ctx context.Context, path string, dst any) error {
	return c.fetchWithReauth(ctx, path, dst, true)
}

func (c *Client) fetchWithReauth(ctx context.Context, path string, dst any, allowReauth bool) error {
	if cached, ok := c.cache.Get(path); ok {
		return decodeJSON(cached, dst)
	}

	c.mu.RLock()
	tokenUsed := c.authToken
	c.mu.RUnlock()

	req, err := c.newRequest(ctx, http.MethodGet, path, nil, true)
	if err != nil {
		return err
	}

	resp, err := c.doWithRetry(ctx, req) //nolint:bodyclose // closed by readAndClose
	if err != nil {
		return classifyNetError(err)
	}

	data, err := readAndClose(resp, c.maxResponseBytes)
	if err != nil {
		return &APIError{Code: ErrCodeNetworkError, Message: "failed to read response body", Err: err}
	}

	if envErr := checkEnvelope(data, resp.StatusCode); envErr != nil {
		if allowReauth && errors.Is(envErr, ErrUnauthorized) && c.reauthenticate(ctx, tokenUsed) {
			return c.fetchWithReauth(ctx, path, dst, false)
		}
		return envErr
	}

	c.cache.Set(path, path, data)
	return decodeJSON(data, dst)
}

// fetchDirect fetches path, caches the raw bytes, and decodes into dst
// WITHOUT checking the success envelope. Used for endpoints that return a
// different JSON structure (e.g. /api/levelpayusage/).
func (c *Client) fetchDirect(ctx context.Context, path string, dst any, authenticated bool) error {
	if cached, ok := c.cache.Get(path); ok {
		return decodeJSON(cached, dst)
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil, authenticated)
	if err != nil {
		return err
	}

	resp, err := c.doWithRetry(ctx, req) //nolint:bodyclose // closed by readAndClose
	if err != nil {
		return classifyNetError(err)
	}

	data, err := readAndClose(resp, c.maxResponseBytes)
	if err != nil {
		return &APIError{Code: ErrCodeNetworkError, Message: "failed to read response body", Err: err}
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			Code:       httpStatusToErrCode(resp.StatusCode),
			StatusCode: resp.StatusCode,
			Message:    "request failed",
		}
	}

	c.cache.Set(path, path, data)
	return decodeJSON(data, dst)
}

// doSimpleGET performs an optionally non-authed GET without caching and returns
// the raw body bytes.
func (c *Client) doSimpleGET(ctx context.Context, path string, authenticated bool, mods ...func(*http.Request)) ([]byte, int, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil, authenticated)
	if err != nil {
		return nil, 0, err
	}
	for _, m := range mods {
		m(req)
	}
	resp, err := c.doWithRetry(ctx, req) //nolint:bodyclose // closed by readAndClose
	if err != nil {
		return nil, 0, classifyNetError(err)
	}
	data, err := readAndClose(resp, c.maxResponseBytes)
	if err != nil {
		return nil, resp.StatusCode, &APIError{Code: ErrCodeNetworkError, Err: err}
	}
	return data, resp.StatusCode, nil
}

// post marshals body, POSTs to path, and decodes the response into dst.
// The response is not cached. On 401, it attempts re-authentication and
// retries once.
func (c *Client) post(ctx context.Context, path string, body, dst any) error {
	return c.postWithReauth(ctx, path, body, dst, true)
}

func (c *Client) postWithReauth(ctx context.Context, path string, body, dst any, allowReauth bool) error {
	c.mu.RLock()
	tokenUsed := c.authToken
	c.mu.RUnlock()

	req, err := c.newRequest(ctx, http.MethodPost, path, body, true)
	if err != nil {
		return err
	}
	resp, err := c.doWithRetry(ctx, req) //nolint:bodyclose // closed by readAndClose
	if err != nil {
		return classifyNetError(err)
	}
	data, err := readAndClose(resp, c.maxResponseBytes)
	if err != nil {
		return &APIError{Code: ErrCodeNetworkError, Message: "failed to read response body", Err: err}
	}
	if envErr := checkEnvelope(data, resp.StatusCode); envErr != nil {
		if allowReauth && errors.Is(envErr, ErrUnauthorized) && c.reauthenticate(ctx, tokenUsed) {
			return c.postWithReauth(ctx, path, body, dst, false)
		}
		return envErr
	}
	if dst != nil {
		return decodeJSON(data, dst)
	}
	return nil
}

// isRetryable reports whether the response/error warrants a retry.
func isRetryable(resp *http.Response, err error) bool {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		var netErr net.Error
		return errors.As(err, &netErr)
	}
	return resp != nil && resp.StatusCode >= 500
}

// backoffDuration returns the delay for the given 0-indexed retry attempt
// using exponential back-off with full jitter, capped at maxDelay.
func backoffDuration(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Use bitwise shift for power-of-2 to avoid floating point overhead.
	// Cap attempt to avoid shifting out of bounds.
	delay := baseDelay
	if attempt < 63 {
		delay <<= attempt
	} else {
		delay = maxDelay // Shift would overflow int64, cap it.
	}

	// If the shifted value overflowed and became negative, or exceeded maxDelay, cap it.
	if delay <= 0 || delay > maxDelay {
		delay = maxDelay
	}

	var jitter time.Duration
	if baseDelay > 0 {
		jitter = time.Duration(rand.Int64N(int64(baseDelay))) //nolint:gosec // jitter does not need crypto/rand
	}
	delay += jitter
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// classifyNetError wraps a raw error from the HTTP stack into an *APIError.
func classifyNetError(err error) *APIError {
	if errors.Is(err, context.Canceled) {
		return &APIError{Code: ErrCodeContextCanceled, Err: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &APIError{Code: ErrCodeContextDeadline, Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &APIError{Code: ErrCodeNetworkError, Err: err}
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return &APIError{Code: ErrCodeUnknown, Err: err}
}

// httpStatusToErrCode maps HTTP status codes to ErrorCode values.
func httpStatusToErrCode(status int) ErrorCode {
	switch {
	case status == http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case status == http.StatusForbidden:
		return ErrCodeForbidden
	case status == http.StatusNotFound:
		return ErrCodeNotFound
	case status == http.StatusTooManyRequests:
		return ErrCodeRateLimited
	case status >= 500:
		return ErrCodeServerError
	default:
		return ErrCodeUnknown
	}
}

// reauthenticate attempts to log in again using stored credentials. It returns
// true if re-authentication succeeded. The caller must hold no locks.
// Concurrent callers are coordinated via reauthChan so that only one login
// request is in flight at a time; others wait for the result.
func (c *Client) reauthenticate(ctx context.Context, staleToken string) bool {
	c.mu.Lock()
	for c.reauthChan != nil {
		ch := c.reauthChan
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return false
		case <-ch:
		}
		c.mu.Lock()
	}

	if c.authToken != staleToken {
		success := c.authToken != ""
		c.mu.Unlock()
		return success
	}

	email := c.email
	pwHash := c.passwordHash
	if pwHash == "" {
		c.mu.Unlock()
		return false
	}

	ch := make(chan struct{})
	c.reauthChan = ch
	c.authToken = ""
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.reauthChan = nil
		close(ch)
		c.mu.Unlock()
	}()

	reqBody := LoginRequest{
		Email:    email,
		Password: pwHash,
	}
	var resp LoginResponse
	if err := c.postWithReauth(ctx, "/api/login/", reqBody, &resp, false); err != nil {
		return false
	}

	c.mu.Lock()
	if c.email == email && c.passwordHash == pwHash {
		c.authToken = resp.AuthToken
		c.isLevelPay = resp.IsLevelPay
	}
	c.mu.Unlock()
	return true
}

// requireAuth returns ErrAuthRequired if the client has no auth token.
func (c *Client) requireAuth() error {
	c.mu.RLock()
	token := c.authToken
	c.mu.RUnlock()
	if token == "" {
		return ErrAuthRequired
	}
	return nil
}

// isLoopback reports whether host is a loopback address (IPv4 127.0.0.0/8,
// IPv6 ::1, or the literal name "localhost").
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
