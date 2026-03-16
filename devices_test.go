package pinergy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateDeviceToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/updatedevicetoken/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	injectToken(c, "tok")

	err := c.UpdateDeviceToken(context.Background(), "", "android", "Android SDK: 33 (13)")
	if err != nil {
		t.Fatalf("UpdateDeviceToken: %v", err)
	}
}

func TestUpdateDeviceToken_AuthRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.UpdateDeviceToken(context.Background(), "", "android", "")
	if !errors.Is(err, ErrAuthRequired) {
		t.Errorf("expected ErrAuthRequired, got %v", err)
	}
}
