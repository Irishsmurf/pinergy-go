package pinergy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetVersion_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/version_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// version.json does not require auth.
		if tok := r.Header.Get("auth_token"); tok != "" {
			t.Errorf("unexpected auth_token header on unauthenticated endpoint: %q", tok)
		}
		w.Write(data)
	}))
	defer srv.Close()

	// No token set — version endpoint is public.
	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled())

	resp, err := c.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if resp.MinVersion != "3.0.0" {
		t.Errorf("MinVersion = %q, want 3.0.0", resp.MinVersion)
	}
	if resp.CurrentVersion != "3.5.2" {
		t.Errorf("CurrentVersion = %q, want 3.5.2", resp.CurrentVersion)
	}
}

func TestGetVersion_UsesCache(t *testing.T) {
	data, _ := os.ReadFile("testdata/version_response.json")
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write(data)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(1))

	c.GetVersion(context.Background())
	c.GetVersion(context.Background())

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache hit on 2nd), got %d", callCount)
	}
}

func TestGetVersion_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCacheDisabled(), WithMaxRetries(1))

	_, err := c.GetVersion(context.Background())
	if err == nil {
		t.Fatal("expected error on 404")
	}
}
