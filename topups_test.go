package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetActiveTopups_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/activetopups_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/activetopups/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetActiveTopups(context.Background())
	if err != nil {
		t.Fatalf("GetActiveTopups: %v", err)
	}
	if len(resp.Scheduled) != 1 {
		t.Errorf("expected 1 scheduled top-up, got %d", len(resp.Scheduled))
	}
	if resp.Scheduled[0].TopUpAmount != 150.0 {
		t.Errorf("TopUpAmount = %v, want 150.0", resp.Scheduled[0].TopUpAmount)
	}
	if resp.Scheduled[0].CurrentUser {
		t.Error("expected CurrentUser = false")
	}
}

func TestGetActiveTopups_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetActiveTopups(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}
