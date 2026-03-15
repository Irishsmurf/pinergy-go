package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetConfigInfo_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/configinfo_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetConfigInfo(context.Background())
	if err != nil {
		t.Fatalf("GetConfigInfo: %v", err)
	}
	if len(resp.TopUpAmounts) == 0 {
		t.Error("expected non-empty TopUpAmounts")
	}
	if resp.TopUpAmounts[0] != 10 {
		t.Errorf("TopUpAmounts[0] = %v, want 10", resp.TopUpAmounts[0])
	}
}

func TestGetConfigInfo_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetConfigInfo(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}

func TestGetDefaultsInfo_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/defaultsinfo_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/defaultsinfo/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetDefaultsInfo(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultsInfo: %v", err)
	}
	if len(resp.HouseTypes) != 5 {
		t.Errorf("expected 5 house types, got %d", len(resp.HouseTypes))
	}
	if resp.MaxBedrooms != 6 {
		t.Errorf("MaxBedrooms = %d, want 6", resp.MaxBedrooms)
	}
}

func TestGetDefaultsInfo_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetDefaultsInfo(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}
