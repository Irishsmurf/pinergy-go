//go:build integration

package pinergy_test

import (
	"context"
	"os"
	"testing"
	"time"

	pinergy "github.com/Irishsmurf/pinergy-go"
)

// newIntegrationClient creates and authenticates a real API client using
// credentials from environment variables.
//
// Required environment variables:
//   - PINERGY_EMAIL
//   - PINERGY_PASSWORD
//
// Optional:
//   - PINERGY_BASE_URL (overrides the default API URL for testing against staging)
func newIntegrationClient(t *testing.T) *pinergy.Client {
	t.Helper()

	email := os.Getenv("PINERGY_EMAIL")
	password := os.Getenv("PINERGY_PASSWORD")
	if email == "" || password == "" {
		t.Skip("integration tests require PINERGY_EMAIL and PINERGY_PASSWORD environment variables")
	}

	opts := []pinergy.Option{
		pinergy.WithCacheDisabled(), // always fetch fresh data in integration tests
		pinergy.WithTimeout(30 * time.Second),
	}
	if base := os.Getenv("PINERGY_BASE_URL"); base != "" {
		opts = append(opts, pinergy.WithBaseURL(base))
	}

	client := pinergy.NewClient(opts...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Login(ctx, email, password); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if !client.IsAuthenticated() {
		t.Fatal("expected client to be authenticated after Login")
	}

	return client
}

func TestIntegration_Login(t *testing.T) {
	// newIntegrationClient already exercises Login; just ensure it succeeds.
	client := newIntegrationClient(t)
	if !client.IsAuthenticated() {
		t.Error("expected authenticated client")
	}
}

func TestIntegration_GetBalance(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	bal, err := client.GetBalance(ctx)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}

	// Validate shape, not specific values.
	if bal.Balance < 0 {
		t.Errorf("unexpected negative balance: %v", bal.Balance)
	}
	// LastReading should be a recent date (within the last year).
	if bal.LastReading.IsZero() {
		t.Error("expected non-zero LastReading")
	}
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	if bal.LastReading.Before(oneYearAgo) {
		t.Errorf("LastReading seems too old: %v", bal.LastReading)
	}
}

func TestIntegration_GetUsage(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	usage, err := client.GetUsage(ctx)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}

	if len(usage.Day) == 0 {
		t.Error("expected at least one day entry")
	}
	if len(usage.Week) == 0 {
		t.Error("expected at least one week entry")
	}
	if len(usage.Month) == 0 {
		t.Error("expected at least one month entry")
	}

	// Verify dates are parseable and non-zero.
	for i, entry := range usage.Day {
		if entry.Available && entry.Date.IsZero() {
			t.Errorf("Day[%d]: available but zero date", i)
		}
	}
}

func TestIntegration_GetCompare(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	resp, err := client.GetCompare(ctx)
	if err != nil {
		t.Fatalf("GetCompare: %v", err)
	}

	if resp.Day.Available && resp.Day.Euro.UsersHome < 0 {
		t.Errorf("Day.Euro.UsersHome is negative: %v", resp.Day.Euro.UsersHome)
	}
}

func TestIntegration_GetConfigInfo(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	resp, err := client.GetConfigInfo(ctx)
	if err != nil {
		t.Fatalf("GetConfigInfo: %v", err)
	}

	if len(resp.TopUpAmounts) == 0 {
		t.Error("expected non-empty TopUpAmounts")
	}
}

func TestIntegration_GetDefaultsInfo(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	resp, err := client.GetDefaultsInfo(ctx)
	if err != nil {
		t.Fatalf("GetDefaultsInfo: %v", err)
	}

	if len(resp.HouseTypes) == 0 {
		t.Error("expected non-empty HouseTypes")
	}
	if len(resp.HeatingTypes) == 0 {
		t.Error("expected non-empty HeatingTypes")
	}
	if resp.MaxBedrooms == 0 {
		t.Error("expected non-zero MaxBedrooms")
	}
}

func TestIntegration_GetActiveTopups(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	resp, err := client.GetActiveTopups(ctx)
	if err != nil {
		t.Fatalf("GetActiveTopups: %v", err)
	}

	// Scheduled may be empty; just ensure the response is valid.
	_ = resp.Scheduled
}

func TestIntegration_GetNotifications(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	_, err := client.GetNotifications(ctx)
	if err != nil {
		t.Fatalf("GetNotifications: %v", err)
	}
}

func TestIntegration_GetVersion(t *testing.T) {
	// GetVersion is unauthenticated; no login required.
	email := os.Getenv("PINERGY_EMAIL")
	if email == "" {
		t.Skip("integration tests require PINERGY_EMAIL")
	}

	opts := []pinergy.Option{pinergy.WithCacheDisabled()}
	if base := os.Getenv("PINERGY_BASE_URL"); base != "" {
		opts = append(opts, pinergy.WithBaseURL(base))
	}
	client := pinergy.NewClient(opts...)

	resp, err := client.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if resp.MinVersion == "" && resp.CurrentVersion == "" {
		t.Error("expected at least one version field to be non-empty")
	}
}

func TestIntegration_Logout(t *testing.T) {
	client := newIntegrationClient(t)

	if !client.IsAuthenticated() {
		t.Fatal("expected authenticated before Logout")
	}
	client.Logout()
	if client.IsAuthenticated() {
		t.Error("expected unauthenticated after Logout")
	}
}
