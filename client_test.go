package pinergy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient creates a Client pointing at srv.URL with the cache and
// retry delays configured for fast unit tests.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(3),
		WithRetryDelays(1*time.Millisecond, 10*time.Millisecond),
	)
}

// injectToken sets the auth token directly, bypassing the Login flow.
func injectToken(c *Client, token string) {
	c.mu.Lock()
	c.authToken = token
	c.mu.Unlock()
}

// TestIsRetryable verifies the retryability classification.
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		err  error
		want bool
	}{
		{"nil resp nil err", nil, nil, false},
		{"200 OK", &http.Response{StatusCode: 200}, nil, false},
		{"400 Bad Request", &http.Response{StatusCode: 400}, nil, false},
		{"401 Unauthorized", &http.Response{StatusCode: 401}, nil, false},
		{"500 Internal Server Error", &http.Response{StatusCode: 500}, nil, true},
		{"503 Service Unavailable", &http.Response{StatusCode: 503}, nil, true},
		{"context canceled", nil, context.Canceled, false},
		{"context deadline exceeded", nil, context.DeadlineExceeded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.resp, tt.err)
			if got != tt.want {
				t.Errorf("isRetryable = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsRetryable_NetError verifies that net.Error triggers a retry.
func TestIsRetryable_NetError(t *testing.T) {
	var netErr net.Error = &mockNetError{temporary: true}
	if !isRetryable(nil, netErr) {
		t.Error("expected net.Error to be retryable")
	}
}

type mockNetError struct{ temporary bool }

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return false }
func (e *mockNetError) Temporary() bool { return e.temporary }

// TestBackoffDuration verifies the back-off properties.
func TestBackoffDuration(t *testing.T) {
	base := 100 * time.Millisecond
	max := 5 * time.Second

	prev := time.Duration(0)
	for i := 0; i < 5; i++ {
		d := backoffDuration(i, base, max)
		if d > max {
			t.Errorf("attempt %d: delay %v exceeds max %v", i, d, max)
		}
		if d < base {
			t.Errorf("attempt %d: delay %v less than base %v (jitter issue)", i, d, base)
		}
		_ = prev
		prev = d
	}
}

// TestBackoffDuration_Cap verifies the cap is enforced.
func TestBackoffDuration_Cap(t *testing.T) {
	base := 1 * time.Second
	max := 2 * time.Second
	for i := 0; i < 10; i++ {
		d := backoffDuration(i, base, max)
		if d > max {
			t.Errorf("attempt %d: delay %v exceeds cap %v", i, d, max)
		}
	}
}

// TestDoWithRetry_RetriesOn503 verifies that the client retries up to maxRetries
// times on 5xx responses before returning a successful response.
func TestDoWithRetry_RetriesOn503(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"balance":10.5,"top_up_in_days":5,"pending_top_up":false,"pending_top_up_by":"","last_top_up_time":"1772182668","last_top_up_amount":50.0,"credit_low":false,"emergency_credit":false,"power_off":false,"last_reading":"1773532800"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestDoWithRetry_NoRetryOn401 verifies that 4xx responses are not retried.
func TestDoWithRetry_NoRetryOn401(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"error_code":1,"message":"invalid token"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "bad-token")

	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt on 401, got %d", attempts)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeUnauthorized {
		t.Errorf("expected ErrCodeUnauthorized, got %v", apiErr.Code)
	}
}

// TestContextCancellation verifies that a canceled context is respected.
func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.GetBalance(ctx)
	if err == nil {
		t.Fatal("expected error with short timeout")
	}
}

// TestRequireAuth verifies that authenticated endpoints return ErrAuthRequired
// when no token is set.
func TestRequireAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected HTTP call — auth guard should have fired first")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// No token set.

	_, err := c.GetBalance(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}

// TestNoRedirect verifies that the default HTTP client does not follow redirects,
// preventing auth header leakage to third-party domains.
func TestNoRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/balance/" {
			http.Redirect(w, r, "https://evil.example.com/steal", http.StatusFound)
			return
		}
		t.Error("unexpected follow-up request")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error when server redirects")
	}
}

func BenchmarkBackoffDuration(b *testing.B) {
	base := 500 * time.Millisecond
	max := 10 * time.Second
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backoffDuration(i%10, base, max)
	}
}

// TestBackoffDuration_Overflow verifies that large attempts do not cause integer overflow.
func TestBackoffDuration_Overflow(t *testing.T) {
	base := 100 * time.Millisecond
	max := 5 * time.Second

	d := backoffDuration(100, base, max)
	if d <= 0 {
		t.Errorf("attempt 100 resulted in invalid delay %v", d)
	}
	if d > max {
		t.Errorf("attempt 100 delay %v exceeds max %v", d, max)
	}
}

// TestBackoffDuration_BaseDelayOverflow verifies overflow protection handles
// the case where baseDelay << attempt overflows time.Duration.
func TestBackoffDuration_BaseDelayOverflow(t *testing.T) {
	base := 1 * time.Second
	max := 5 * time.Second

	d := backoffDuration(34, base, max) // 1 second << 34 overflows int64
	if d <= 0 {
		t.Errorf("attempt 34 with 1s base delay resulted in invalid delay %v", d)
	}
	if d > max {
		t.Errorf("attempt 34 delay %v exceeds max %v", d, max)
	}
}

// TestHTTPSEnforcement verifies that plaintext HTTP to non-loopback hosts is
// rejected unless WithInsecureHTTP is set.
func TestHTTPSEnforcement_RejectsPlaintextHTTP(t *testing.T) {
	c := NewClient(WithBaseURL("http://api.example.com"), WithCacheDisabled())
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error for plaintext HTTP to non-loopback host")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
}

func TestHTTPSEnforcement_AllowsLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"balance":1.0}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("loopback HTTP should be allowed: %v", err)
	}
}

func TestHTTPSEnforcement_AllowsInsecureOpt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"balance":1.0}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithInsecureHTTP(), WithCacheDisabled())
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("insecure opt-in should allow HTTP: %v", err)
	}
}

// TestMaxResponseBytes verifies that oversized responses return an explicit
// error rather than silently truncating.
func TestMaxResponseBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"balance":1.0}`))
		padding := make([]byte, 1024)
		for i := range padding {
			padding[i] = ' '
		}
		for i := 0; i < 200; i++ {
			w.Write(padding)
		}
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(1),
		WithMaxResponseBytes(512),
	)
	injectToken(c, "tok")

	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error when response exceeds max size")
	}
}

func TestMaxResponseBytes_WithinLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"balance":1.0}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(1),
		WithMaxResponseBytes(512),
	)
	injectToken(c, "tok")

	bal, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("expected success for response within limit: %v", err)
	}
	if bal.Balance != 1.0 {
		t.Errorf("Balance = %v, want 1.0", bal.Balance)
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"::1", true},
		{"0.0.0.0", false},
		{"api.pinergy.ie", false},
		{"192.168.1.1", false},
	}
	for _, tt := range tests {
		if got := isLoopback(tt.host); got != tt.want {
			t.Errorf("isLoopback(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// classifyNetError
// ---------------------------------------------------------------------------

func TestClassifyNetError_ContextCanceled(t *testing.T) {
	apiErr := classifyNetError(context.Canceled)
	if apiErr.Code != ErrCodeContextCanceled {
		t.Errorf("Code = %v, want ErrCodeContextCanceled", apiErr.Code)
	}
	if !errors.Is(apiErr, context.Canceled) {
		t.Error("expected wrapped error to match context.Canceled")
	}
}

func TestClassifyNetError_ContextDeadline(t *testing.T) {
	apiErr := classifyNetError(context.DeadlineExceeded)
	if apiErr.Code != ErrCodeContextDeadline {
		t.Errorf("Code = %v, want ErrCodeContextDeadline", apiErr.Code)
	}
}

func TestClassifyNetError_NetError(t *testing.T) {
	var netErr net.Error = &mockNetError{temporary: true}
	apiErr := classifyNetError(netErr)
	if apiErr.Code != ErrCodeNetworkError {
		t.Errorf("Code = %v, want ErrCodeNetworkError", apiErr.Code)
	}
}

func TestClassifyNetError_DNSError(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "api.pinergy.ie"}
	apiErr := classifyNetError(dnsErr)
	if apiErr.Code != ErrCodeNetworkError {
		t.Errorf("Code = %v, want ErrCodeNetworkError", apiErr.Code)
	}
}

func TestClassifyNetError_APIError(t *testing.T) {
	inner := &APIError{Code: ErrCodeRateLimited, Message: "too many requests"}
	apiErr := classifyNetError(inner)
	if apiErr.Code != ErrCodeRateLimited {
		t.Errorf("Code = %v, want ErrCodeRateLimited", apiErr.Code)
	}
	if apiErr != inner {
		t.Error("expected classifyNetError to return the original *APIError")
	}
}

func TestClassifyNetError_UnknownError(t *testing.T) {
	apiErr := classifyNetError(errors.New("something unexpected"))
	if apiErr.Code != ErrCodeUnknown {
		t.Errorf("Code = %v, want ErrCodeUnknown", apiErr.Code)
	}
}

// ---------------------------------------------------------------------------
// httpStatusToErrCode
// ---------------------------------------------------------------------------

func TestHTTPStatusToErrCode(t *testing.T) {
	tests := []struct {
		status int
		want   ErrorCode
	}{
		{http.StatusUnauthorized, ErrCodeUnauthorized},
		{http.StatusForbidden, ErrCodeForbidden},
		{http.StatusNotFound, ErrCodeNotFound},
		{http.StatusTooManyRequests, ErrCodeRateLimited},
		{http.StatusInternalServerError, ErrCodeServerError},
		{http.StatusBadGateway, ErrCodeServerError},
		{http.StatusServiceUnavailable, ErrCodeServerError},
		{http.StatusBadRequest, ErrCodeUnknown},
		{http.StatusOK, ErrCodeUnknown},
	}
	for _, tt := range tests {
		got := httpStatusToErrCode(tt.status)
		if got != tt.want {
			t.Errorf("httpStatusToErrCode(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// decodeJSON
// ---------------------------------------------------------------------------

func TestDecodeJSON_MalformedJSON(t *testing.T) {
	var dst BalanceResponse
	err := decodeJSON([]byte(`{not json`), &dst)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeInvalidResponse {
		t.Errorf("Code = %v, want ErrCodeInvalidResponse", apiErr.Code)
	}
}

func TestDecodeJSON_EmptyInput(t *testing.T) {
	var dst BalanceResponse
	err := decodeJSON([]byte(""), &dst)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestDecodeJSON_Success(t *testing.T) {
	var dst BalanceResponse
	err := decodeJSON([]byte(`{"success":true,"balance":42.5}`), &dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Balance != 42.5 {
		t.Errorf("Balance = %v, want 42.5", dst.Balance)
	}
}

// ---------------------------------------------------------------------------
// do() — rate limiter integration
// ---------------------------------------------------------------------------

func TestDo_ContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithRateLimit(0.001, 0),
	)
	injectToken(c, "tok")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req, _ := c.newRequest(ctx, http.MethodGet, "/api/balance/", nil, true)
	resp, err := c.do(ctx, req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected error when rate limiter deadline is exceeded")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeContextDeadline && apiErr.Code != ErrCodeContextCanceled {
		t.Errorf("Code = %v, want ErrCodeContextDeadline or ErrCodeContextCanceled", apiErr.Code)
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithRateLimit(0.001, 0),
	)
	injectToken(c, "tok")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := c.newRequest(ctx, http.MethodGet, "/api/balance/", nil, true)
	resp, err := c.do(ctx, req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected error when context is canceled")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeContextCanceled {
		t.Errorf("Code = %v, want ErrCodeContextCanceled", apiErr.Code)
	}
}

// ---------------------------------------------------------------------------
// Client options
// ---------------------------------------------------------------------------

func TestWithHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"balance":1.0}`))
	}))
	defer srv.Close()

	custom := &http.Client{Timeout: 5 * time.Second}
	c := NewClient(
		WithBaseURL(srv.URL),
		WithHTTPClient(custom),
		WithCacheDisabled(),
	)
	injectToken(c, "tok")

	if c.httpClient != custom {
		t.Error("expected custom HTTP client to be used")
	}

	bal, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance with custom client: %v", err)
	}
	if bal.Balance != 1.0 {
		t.Errorf("Balance = %v, want 1.0", bal.Balance)
	}
}

func TestWithTimeout(t *testing.T) {
	c := NewClient(WithTimeout(42 * time.Second))
	if c.httpClient.Timeout != 42*time.Second {
		t.Errorf("Timeout = %v, want 42s", c.httpClient.Timeout)
	}
}

func TestWithRateLimit(t *testing.T) {
	c := NewClient(WithRateLimit(10, 20))
	if c.limiter.Limit() != 10 {
		t.Errorf("Limit = %v, want 10", c.limiter.Limit())
	}
	if c.limiter.Burst() != 20 {
		t.Errorf("Burst = %v, want 20", c.limiter.Burst())
	}
}

func TestCacheFlush(t *testing.T) {
	data := []byte(`{"success":true,"balance":1.0}`)
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(1))
	injectToken(c, "tok")

	c.GetBalance(context.Background())
	c.CacheFlush()
	c.GetBalance(context.Background())

	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls after CacheFlush, got %d", callCount)
	}
}

// ---------------------------------------------------------------------------
// checkEnvelope
// ---------------------------------------------------------------------------

func TestCheckEnvelope_MalformedJSON(t *testing.T) {
	err := checkEnvelope([]byte(`not json`), 200)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeInvalidResponse {
		t.Errorf("Code = %v, want ErrCodeInvalidResponse", apiErr.Code)
	}
}

func TestCheckEnvelope_SuccessFalseNoMessage(t *testing.T) {
	err := checkEnvelope([]byte(`{"success":false,"error_code":42}`), 403)
	if err == nil {
		t.Fatal("expected error for success=false")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeForbidden {
		t.Errorf("Code = %v, want ErrCodeForbidden", apiErr.Code)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// fetchDirect — error paths
// ---------------------------------------------------------------------------

func TestFetchDirect_ForbiddenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`forbidden`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(1),
		WithRetryDelays(1*time.Millisecond, 5*time.Millisecond),
	)
	injectToken(c, "tok")

	var dst LevelPayUsageResponse
	err := c.fetchDirect(context.Background(), "/api/levelpayusage/", &dst, true)
	if err == nil {
		t.Fatal("expected error on 403 response")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeForbidden {
		t.Errorf("Code = %v, want ErrCodeForbidden", apiErr.Code)
	}
}

func TestFetchDirect_429RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`rate limited`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(1),
		WithRetryDelays(1*time.Millisecond, 5*time.Millisecond),
	)
	injectToken(c, "tok")

	var dst LevelPayUsageResponse
	err := c.fetchDirect(context.Background(), "/api/levelpayusage/", &dst, true)
	if err == nil {
		t.Fatal("expected error on 429 response")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeRateLimited {
		t.Errorf("Code = %v, want ErrCodeRateLimited", apiErr.Code)
	}
}

// ---------------------------------------------------------------------------
// doWithRetry — context cancellation between retries
// ---------------------------------------------------------------------------

func TestDoWithRetry_ContextCanceledBetweenRetries(t *testing.T) {
	attempts := make(chan struct{}, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts <- struct{}{}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithMaxRetries(5),
		WithRetryDelays(100*time.Millisecond, 200*time.Millisecond),
	)
	injectToken(c, "tok")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.GetBalance(ctx)
	if err == nil {
		t.Fatal("expected error when context expires during retries")
	}
	if len(attempts) > 2 {
		t.Errorf("expected at most 2 attempts before context expiry, got %d", len(attempts))
	}
}

// ---------------------------------------------------------------------------
// newRequest — body encoding errors
// ---------------------------------------------------------------------------

func TestNewRequest_InvalidBody(t *testing.T) {
	c := NewClient(WithBaseURL("http://localhost:9999"))

	_, err := c.newRequest(context.Background(), http.MethodPost, "/api/test", make(chan int), true)
	if err == nil {
		t.Fatal("expected error when body cannot be JSON-encoded")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// postWithReauth — reauth failure
// ---------------------------------------------------------------------------

func TestPostWithReauth_ReauthFails(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/api/topup/":
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success":false,"error_code":1,"message":"invalid token"}`))
		case "/api/login/":
			w.Write([]byte(`{"success":false,"error_code":1,"message":"invalid credentials"}`))
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.mu.Lock()
	c.authToken = "stale"
	c.email = "u@e.com"
	c.passwordHash = "hash"
	c.mu.Unlock()

	_, err := c.TopUp(context.Background(), 20, "cc_tok")
	if err == nil {
		t.Fatal("expected error when reauth fails")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// readAndClose — response body size limit
// ---------------------------------------------------------------------------

func TestReadAndClose_ExceedsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // readAndClose closes the body
	if err != nil {
		t.Fatal(err)
	}
	_, err = readAndClose(resp, 512)
	if err == nil {
		t.Fatal("expected error when response exceeds limit")
	}
}
