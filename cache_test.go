package pinergy

import (
	"sync"
	"testing"
	"time"
)

func TestTTLCache_GetSet(t *testing.T) {
	c := newTTLCache(nil)
	data := []byte(`{"balance":10.5}`)

	// Miss before set.
	if _, ok := c.Get("/api/balance/"); ok {
		t.Fatal("expected cache miss before set")
	}

	c.Set("/api/balance/", "/api/balance/", data)

	got, ok := c.Get("/api/balance/")
	if !ok {
		t.Fatal("expected cache hit after set")
	}
	if string(got) != string(data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestTTLCache_Expiry(t *testing.T) {
	c := newTTLCache(map[string]time.Duration{
		"/api/balance/": 10 * time.Millisecond,
	})
	c.Set("/api/balance/", "/api/balance/", []byte(`{}`))

	if _, ok := c.Get("/api/balance/"); !ok {
		t.Fatal("expected hit before expiry")
	}

	time.Sleep(20 * time.Millisecond)

	if _, ok := c.Get("/api/balance/"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestTTLCache_Invalidate(t *testing.T) {
	c := newTTLCache(nil)
	c.Set("/api/balance/", "/api/balance/", []byte(`{}`))
	c.Invalidate("/api/balance/")

	if _, ok := c.Get("/api/balance/"); ok {
		t.Fatal("expected miss after invalidation")
	}
}

func TestTTLCache_Flush(t *testing.T) {
	c := newTTLCache(nil)
	c.Set("/api/balance/", "/api/balance/", []byte(`{}`))
	c.Set("/api/usage/", "/api/usage/", []byte(`{}`))
	c.Flush()

	if _, ok := c.Get("/api/balance/"); ok {
		t.Error("expected miss after flush: /api/balance/")
	}
	if _, ok := c.Get("/api/usage/"); ok {
		t.Error("expected miss after flush: /api/usage/")
	}
}

func TestTTLCache_Disabled(t *testing.T) {
	c := newDisabledCache()
	c.Set("/api/balance/", "/api/balance/", []byte(`{}`))

	if _, ok := c.Get("/api/balance/"); ok {
		t.Fatal("expected miss on disabled cache")
	}
}

func TestTTLCache_NoTTLForEndpoint(t *testing.T) {
	c := newTTLCache(nil) // /api/login/ has no TTL
	c.Set("/api/login/", "/api/login/", []byte(`{}`))

	if _, ok := c.Get("/api/login/"); ok {
		t.Fatal("expected miss for unconfigured endpoint")
	}
}

func TestTTLCache_ConcurrentReadWrite(t *testing.T) {
	c := newTTLCache(nil)
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c.Set("/api/balance/", "/api/balance/", []byte(`{"balance":1.0}`))
		}()
		go func() {
			defer wg.Done()
			c.Get("/api/balance/")
		}()
	}

	wg.Wait()
}

func TestTTLCache_SetTTL(t *testing.T) {
	c := newTTLCache(nil)
	// Reduce the balance TTL to something tiny.
	c.SetTTL("/api/balance/", 5*time.Millisecond)
	c.Set("/api/balance/", "/api/balance/", []byte(`{}`))

	time.Sleep(10 * time.Millisecond)

	if _, ok := c.Get("/api/balance/"); ok {
		t.Fatal("expected miss after custom TTL expiry")
	}
}
