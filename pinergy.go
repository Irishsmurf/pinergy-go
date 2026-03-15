// Package pinergy provides an idiomatic Go client for the Pinergy electricity
// API (https://api.pinergy.ie). The client handles authentication, rate
// limiting, exponential-backoff retries, and in-memory TTL caching so that
// callers receive current data without hammering the upstream service.
//
// # Quick start
//
//	client := pinergy.NewClient()
//
//	ctx := context.Background()
//	if err := client.Login(ctx, "user@example.com", "password"); err != nil {
//	    log.Fatal(err)
//	}
//
//	bal, err := client.GetBalance(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Balance: €%.2f\n", bal.Balance)
//
// # Authentication
//
// Call [Client.Login] once. The returned auth token is stored internally and
// attached as an auth_token header on all subsequent authenticated requests.
// The client is safe for concurrent use from multiple goroutines after Login.
//
// # Rate limiting
//
// The client enforces a token-bucket rate limit (default 2 req/s, burst 5)
// using [golang.org/x/time/rate]. Every request waits for a token before
// being dispatched. The wait respects the caller's [context.Context], so
// cancellation propagates correctly.
//
// # Retries
//
// Transient failures (HTTP 5xx, network errors) are retried automatically
// with exponential back-off and jitter. 4xx responses are never retried.
//
// # Caching
//
// GET responses are cached in memory with per-endpoint TTLs. Caching reduces
// API calls when the same data is requested frequently. Use [WithCacheDisabled]
// to opt out, or [WithCacheTTL] to adjust individual TTLs.
package pinergy

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the Pinergy API base URL.
	DefaultBaseURL = "https://api.pinergy.ie"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultRateLimit is the default sustained request rate (requests per second).
	DefaultRateLimit = rate.Limit(2)

	// DefaultBurst is the default burst size for the rate limiter.
	DefaultBurst = 5

	// DefaultMaxRetries is the default number of total attempts (1 initial + N-1 retries).
	DefaultMaxRetries = 3

	// DefaultRetryBaseDelay is the base delay for exponential back-off.
	DefaultRetryBaseDelay = 500 * time.Millisecond

	// DefaultRetryMaxDelay caps the back-off delay.
	DefaultRetryMaxDelay = 10 * time.Second

	// userAgent mimics the official Android client so the API responds normally.
	userAgent = "okhttp/5.1.0"
)

// Option configures a [Client]. Options are applied in order by [NewClient].
type Option func(*Client)

// Client is the Pinergy API client. It is safe for concurrent use from
// multiple goroutines after a successful [Client.Login].
type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      *ttlCache

	maxRetries     int
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration

	mu         sync.RWMutex
	authToken  string
	isLevelPay bool // set by Login; used to route usage calls
}

// NewClient creates a new [Client] with the given options applied over
// the default configuration.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		limiter:        rate.NewLimiter(DefaultRateLimit, DefaultBurst),
		cache:          newTTLCache(nil),
		maxRetries:     DefaultMaxRetries,
		retryBaseDelay: DefaultRetryBaseDelay,
		retryMaxDelay:  DefaultRetryMaxDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithBaseURL overrides the API base URL. Useful for testing or staging
// environments.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient replaces the underlying [http.Client]. The provided client's
// timeout takes precedence over [WithTimeout].
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithRateLimit configures the token-bucket rate limiter.
// r is the sustained rate in requests per second; burst is the maximum
// number of requests allowed to proceed without waiting.
func WithRateLimit(r rate.Limit, burst int) Option {
	return func(c *Client) { c.limiter = rate.NewLimiter(r, burst) }
}

// WithMaxRetries sets the maximum number of attempts for retryable requests.
// A value of 1 means no retries (only the initial attempt).
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

// WithRetryDelays sets the base and maximum delay for exponential back-off.
func WithRetryDelays(base, max time.Duration) Option {
	return func(c *Client) {
		c.retryBaseDelay = base
		c.retryMaxDelay = max
	}
}

// WithCacheTTL overrides the cache TTL for a specific endpoint path
// (e.g. "/api/balance/"). A TTL of 0 disables caching for that endpoint.
func WithCacheTTL(endpoint string, ttl time.Duration) Option {
	return func(c *Client) { c.cache.SetTTL(endpoint, ttl) }
}

// WithCacheDisabled disables the in-memory response cache entirely.
// Every call will hit the upstream API.
func WithCacheDisabled() Option {
	return func(c *Client) { c.cache = newDisabledCache() }
}

// CacheFlush clears all cached responses. Useful after a top-up to force
// a fresh balance read.
func (c *Client) CacheFlush() { c.cache.Flush() }

// CacheInvalidate removes the cached response for a specific endpoint.
func (c *Client) CacheInvalidate(endpoint string) { c.cache.Invalidate(endpoint) }
