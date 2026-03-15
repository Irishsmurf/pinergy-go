package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetCompare_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/compare_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/compare/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetCompare(context.Background())
	if err != nil {
		t.Fatalf("GetCompare: %v", err)
	}
	if !resp.Day.Available {
		t.Error("expected Day.Available = true")
	}
	if resp.Day.Euro.UsersHome != 2.17 {
		t.Errorf("Day.Euro.UsersHome = %v, want 2.17", resp.Day.Euro.UsersHome)
	}
	if resp.Week.KWh.AverageHome != 58.32 {
		t.Errorf("Week.KWh.AverageHome = %v, want 58.32", resp.Week.KWh.AverageHome)
	}
}

func TestGetCompare_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetCompare(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}
