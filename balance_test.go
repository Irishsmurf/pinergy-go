package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestGetBalance_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/balance_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("auth_token") == "" {
			t.Error("expected auth_token header on balance request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "test-token")

	bal, err := c.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.Balance != 16.38 {
		t.Errorf("Balance = %v, want 16.38", bal.Balance)
	}
	if bal.TopUpInDays != 6 {
		t.Errorf("TopUpInDays = %v, want 6", bal.TopUpInDays)
	}
	if bal.CreditLow {
		t.Error("expected CreditLow = false")
	}
	// Verify UnixTime parsing.
	if bal.LastTopUpTime.IsZero() {
		t.Error("expected LastTopUpTime to be non-zero")
	}
}

func TestGetBalance_UsesCache(t *testing.T) {
	data, _ := os.ReadFile("testdata/balance_response.json")
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write(data)
	}))
	defer srv.Close()

	// Use a real cache (not disabled) so caching is exercised.
	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(1))
	injectToken(c, "test-token")

	if _, err := c.GetBalance(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := c.GetBalance(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache hit on 2nd), got %d", callCount)
	}
}

func TestGetBalance_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected HTTP call — auth guard should have fired")
	}))
	defer srv.Close()

	c := newTestClient(t, srv) // no token

	_, err := c.GetBalance(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}

func TestGetBalance_RetryOn503(t *testing.T) {
	data, _ := os.ReadFile("testdata/balance_response.json")
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write(data)
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

func TestGetBalance_UnauthorizedNoRetry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":false,"error_code":1,"message":"session expired"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "expired-token")

	_, err := c.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt on API failure (not 5xx), got %d", attempts)
	}
}

func TestGetBalance_CacheInvalidate(t *testing.T) {
	data, _ := os.ReadFile("testdata/balance_response.json")
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithMaxRetries(1),
		WithCacheTTL("/api/balance/", 10*time.Second),
	)
	injectToken(c, "tok")

	c.GetBalance(context.Background())
	c.CacheInvalidate("/api/balance/")
	c.GetBalance(context.Background())

	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls after invalidation, got %d", callCount)
	}
}
