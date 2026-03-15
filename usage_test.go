package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetUsage_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/usage_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/usage/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	usage, err := c.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(usage.Day) != 7 {
		t.Errorf("expected 7 day entries, got %d", len(usage.Day))
	}
	if len(usage.Week) != 8 {
		t.Errorf("expected 8 week entries, got %d", len(usage.Week))
	}
	if len(usage.Month) != 11 {
		t.Errorf("expected 11 month entries, got %d", len(usage.Month))
	}
	// Verify first day entry.
	if usage.Day[0].KWh != 2.45 {
		t.Errorf("Day[0].KWh = %v, want 2.45", usage.Day[0].KWh)
	}
	if usage.Day[0].Date.IsZero() {
		t.Error("expected Day[0].Date to be non-zero")
	}
}

func TestGetUsage_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetUsage(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}

func TestGetLevelPayUsage_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/levelpay_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/levelpayusage/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// levelpay endpoint does not use the success envelope
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetLevelPayUsage(context.Background())
	if err != nil {
		t.Fatalf("GetLevelPayUsage: %v", err)
	}
	if len(resp.UsageData.Daily.Labels) != 4 {
		t.Errorf("expected 4 labels, got %d", len(resp.UsageData.Daily.Labels))
	}
}
