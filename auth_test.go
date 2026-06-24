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

func TestHashPassword(t *testing.T) {
	// Known SHA-1 vectors.
	tests := []struct {
		input string
		want  string
	}{
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"password", "5baa61e4c9b93f3f0682250b6cf8331b7ee68fd8"},
		{"hunter2", "f3bbbd66a63d4bf1747940578ec3d0103530e21d"},
		{"P@ssw0rd!", "076d3e6c4b9f654b5b220b9045b7458ab6b4cbc6"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := hashPassword(tt.input)
			if got != tt.want {
				t.Errorf("hashPassword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/login_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/login/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	if err := c.Login(context.Background(), "user@example.com", "password"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if !c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() = true after Login")
	}

	c.mu.RLock()
	token := c.authToken
	c.mu.RUnlock()
	if token != "TESTAUTHTOKENABCDEF123456" {
		t.Errorf("unexpected auth token: %q", token)
	}
}

func TestLogin_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // API returns 200 with success=false
		w.Write([]byte(`{"success":false,"error_code":1,"message":"invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	err := c.Login(context.Background(), "bad@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error on failed login")
	}
	if c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() = false after failed login")
	}
}

func TestLogout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	injectToken(c, "sometoken")

	if !c.IsAuthenticated() {
		t.Fatal("expected authenticated before Logout")
	}
	c.Logout()
	if c.IsAuthenticated() {
		t.Error("expected unauthenticated after Logout")
	}
}

func TestCheckEmail_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("email_address") == "" {
			t.Error("expected email_address header")
		}
		w.Write([]byte(`{"success":true,"message":"","error_code":0}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithRetryDelays(1*time.Millisecond, 5*time.Millisecond),
	)
	if err := c.CheckEmail(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("CheckEmail: %v", err)
	}
}

func TestCheckEmail_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"message":"email not registered","error_code":1}`))
	}))
	defer srv.Close()

	c := NewClient(
		WithBaseURL(srv.URL),
		WithCacheDisabled(),
		WithRetryDelays(1*time.Millisecond, 5*time.Millisecond),
	)
	err := c.CheckEmail(context.Background(), "unknown@example.com")
	if err == nil {
		t.Fatal("expected error for unregistered email")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeEmailNotFound {
		t.Errorf("expected ErrCodeEmailNotFound, got %v", apiErr.Code)
	}
	if !errors.Is(err, ErrEmailNotFound) {
		t.Error("expected errors.Is(err, ErrEmailNotFound) to be true")
	}
}

func TestCheckEmail_RejectsCRLF(t *testing.T) {
	c := NewClient(WithCacheDisabled())
	for _, email := range []string{"user@example.com\r\nEvil: header", "user\n@example.com", "a\rb"} {
		err := c.CheckEmail(context.Background(), email)
		if err == nil {
			t.Errorf("expected error for email %q", email)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Errorf("expected *APIError for %q, got %T", email, err)
		}
	}
}

func TestLogin_StoresIsLevelPay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"auth_token":"tok123","is_level_pay":true,"user":{},"house":{},"credit_cards":[]}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	if err := c.Login(context.Background(), "u@e.com", "p"); err != nil {
		t.Fatal(err)
	}

	if !c.IsLevelPay() {
		t.Error("expected IsLevelPay() = true after Login with is_level_pay:true")
	}
}

func TestLoginFull_ReturnsResponse(t *testing.T) {
	data, _ := os.ReadFile("testdata/login_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	resp, err := c.LoginFull(context.Background(), "user@example.com", "password")
	if err != nil {
		t.Fatalf("LoginFull: %v", err)
	}

	if !c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() = true after LoginFull")
	}
	if resp.AuthToken != "TESTAUTHTOKENABCDEF123456" {
		t.Errorf("AuthToken = %q, want TESTAUTHTOKENABCDEF123456", resp.AuthToken)
	}
	if resp.User.Name == "" {
		t.Error("expected non-empty User.Name in response")
	}
	if resp.PremisesNumber == "" {
		t.Error("expected non-empty PremisesNumber in response")
	}
}

func TestLoginFull_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false,"error_code":1,"message":"invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	resp, err := c.LoginFull(context.Background(), "bad@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error on failed login")
	}
	if resp != nil {
		t.Error("expected nil response on failure")
	}
}

func TestIsLevelPay_FalseBeforeLogin(t *testing.T) {
	c := NewClient()
	if c.IsLevelPay() {
		t.Error("expected IsLevelPay() = false before login")
	}
}

func TestIsLevelPay_ResetByLogout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"auth_token":"tok","is_level_pay":true,"user":{},"house":{},"credit_cards":[]}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())
	c.Login(context.Background(), "u@e.com", "p")

	if !c.IsLevelPay() {
		t.Fatal("expected IsLevelPay() = true after login")
	}
	c.Logout()
	if c.IsLevelPay() {
		t.Error("expected IsLevelPay() = false after Logout")
	}
}
