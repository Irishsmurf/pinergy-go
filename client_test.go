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
