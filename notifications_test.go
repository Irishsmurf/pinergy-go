package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetNotifications_Success(t *testing.T) {
	data, _ := os.ReadFile("testdata/getnotif_response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/getnotif/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(data)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	resp, err := c.GetNotifications(context.Background())
	if err != nil {
		t.Fatalf("GetNotifications: %v", err)
	}
	if resp.SMS {
		t.Error("expected SMS = false")
	}
	if !resp.Email {
		t.Error("expected Email = true")
	}
	if !resp.Phone {
		t.Error("expected Phone = true")
	}
}

func TestGetNotifications_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetNotifications(context.Background())
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}
