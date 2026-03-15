# pinergy-go

An idiomatic Go client library for the [Pinergy](https://www.pinergy.ie) electricity API.

> **Disclaimer:** This library is based on reverse-engineered Android app traffic and is not officially supported by Pinergy. Use at your own risk.

[![CI](https://github.com/Irishsmurf/pinergy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/Irishsmurf/pinergy-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Irishsmurf/pinergy-go.svg)](https://pkg.go.dev/github.com/Irishsmurf/pinergy-go)

---

## Features

- **Full API coverage** — balance, usage, comparison, top-ups, config, notifications, and version
- **Rate limiting** — token-bucket limiter (2 req/s, burst 5) to avoid hammering the upstream API
- **Automatic retries** — exponential back-off with jitter on transient 5xx and network errors
- **In-memory caching** — per-endpoint TTL cache (60s–30min) reduces unnecessary API calls
- **Context-aware** — every method accepts a `context.Context`; cancellation propagates cleanly
- **Thread-safe** — a single `*Client` can be shared across goroutines after login
- **Typed errors** — `*APIError` with `errors.Is` / `errors.As` support and an `ErrorCode` enum

---

## Installation

```bash
go get github.com/Irishsmurf/pinergy-go
```

Requires Go 1.22 or later.

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    pinergy "github.com/Irishsmurf/pinergy-go"
)

func main() {
    client := pinergy.NewClient()

    ctx := context.Background()

    if err := client.Login(ctx, "user@example.com", "yourpassword"); err != nil {
        log.Fatal(err)
    }

    bal, err := client.GetBalance(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Balance: €%.2f  (%.0f days remaining)\n", bal.Balance, float64(bal.TopUpInDays))
}
```

---

## Available Methods

| Method | Description | Cached | Auth required |
|---|---|---|---|
| `CheckEmail(ctx, email)` | Check if an email is registered | No | No |
| `Login(ctx, email, password)` | Authenticate and store the auth token | No | No |
| `IsAuthenticated()` | Report whether a token is stored | — | — |
| `Logout()` | Clear the stored token and flush cache | — | — |
| `GetBalance(ctx)` | Current credit balance and top-up info | 60s | Yes |
| `GetUsage(ctx)` | Daily/weekly/monthly energy usage | 5m | Yes |
| `GetLevelPayUsage(ctx)` | Half-hourly interval data (level-pay) | 5m | Yes |
| `GetCompare(ctx)` | Usage vs. similar homes | 15m | Yes |
| `GetConfigInfo(ctx)` | Valid top-up amounts and thresholds | 30m | Yes |
| `GetDefaultsInfo(ctx)` | House/heating type reference data | 30m | Yes |
| `GetActiveTopups(ctx)` | Scheduled and auto top-up config | 2m | Yes |
| `GetNotifications(ctx)` | Notification preferences | 5m | Yes |
| `UpdateDeviceToken(ctx, token, type, osVer)` | Register FCM push token | No | Yes |
| `GetVersion(ctx)` | Minimum/current app version | 10m | No |
| `CacheFlush()` | Clear all cached responses | — | — |
| `CacheInvalidate(endpoint)` | Clear cache for one endpoint | — | — |

---

## Configuration

Pass `Option` values to `NewClient`:

```go
client := pinergy.NewClient(
    pinergy.WithTimeout(15 * time.Second),
    pinergy.WithRateLimit(1, 3),                              // 1 req/s, burst 3
    pinergy.WithMaxRetries(5),
    pinergy.WithRetryDelays(200*time.Millisecond, 5*time.Second),
    pinergy.WithCacheTTL("/api/balance/", 30*time.Second),    // override one endpoint TTL
    // pinergy.WithCacheDisabled(),                           // disable caching entirely
)
```

| Option | Default | Description |
|---|---|---|
| `WithBaseURL(u)` | `https://api.pinergy.ie` | Override the API base URL |
| `WithHTTPClient(c)` | internal | Replace the underlying `*http.Client` |
| `WithTimeout(d)` | 30s | HTTP request timeout |
| `WithRateLimit(r, burst)` | 2 req/s, burst 5 | Token-bucket rate limiter |
| `WithMaxRetries(n)` | 3 | Total attempts (1 initial + n−1 retries) |
| `WithRetryDelays(base, max)` | 500ms, 10s | Exponential back-off base and cap |
| `WithCacheTTL(endpoint, ttl)` | varies | Override TTL for one endpoint |
| `WithCacheDisabled()` | enabled | Disable all caching |

---

## Error Handling

All errors are returned as `*APIError` values:

```go
import "errors"

bal, err := client.GetBalance(ctx)
if err != nil {
    // Quick check with sentinel errors:
    if errors.Is(err, pinergy.ErrAuthRequired) {
        log.Fatal("call Login first")
    }
    if errors.Is(err, pinergy.ErrUnauthorized) {
        log.Fatal("auth token rejected — re-login")
    }

    // Inspect the full error:
    var apiErr *pinergy.APIError
    if errors.As(err, &apiErr) {
        log.Printf("code=%s http=%d msg=%s", apiErr.Code, apiErr.StatusCode, apiErr.Message)
    }
    log.Fatal(err)
}
```

### Sentinel errors

| Variable | When returned |
|---|---|
| `ErrAuthRequired` | Authenticated endpoint called before `Login` |
| `ErrUnauthorized` | API rejected the auth token (HTTP 401) |
| `ErrEmailNotFound` | `CheckEmail`: address is not registered |
| `ErrRateLimited` | API returned HTTP 429 |

### `ErrorCode` values

`ErrCodeUnknown`, `ErrCodeUnauthorized`, `ErrCodeForbidden`, `ErrCodeNotFound`,
`ErrCodeRateLimited`, `ErrCodeServerError`, `ErrCodeInvalidResponse`,
`ErrCodeContextCanceled`, `ErrCodeContextDeadline`, `ErrCodeNetworkError`,
`ErrCodeEmailNotFound`, `ErrCodeAuthRequired`

See [docs/error-handling.md](docs/error-handling.md) for full details.

---

## Caching

Responses are cached in memory with per-endpoint TTLs. This avoids hammering the API when the same data is needed frequently (e.g. polling balance in a dashboard).

```go
// Force a fresh balance read after a top-up:
client.CacheInvalidate("/api/balance/")

// Or clear everything:
client.CacheFlush()
```

See [docs/caching.md](docs/caching.md) for TTL details and customisation.

---

## Rate Limiting

The client applies a token-bucket rate limiter (2 req/s sustained, burst of 5) before every request. If the bucket is empty, the caller's goroutine waits. If the `context.Context` is canceled or times out while waiting, the call returns immediately.

See [docs/rate-limiting.md](docs/rate-limiting.md).

---

## Testing

```bash
# Unit tests (always safe to run, no network required)
make test

# Unit tests with coverage report
make coverage

# Lint
make lint

# Integration tests (requires a real Pinergy account)
PINERGY_EMAIL=user@example.com PINERGY_PASSWORD=secret make integration
```

### Integration tests

Integration tests are behind the `integration` build tag and will be skipped
unless `PINERGY_EMAIL` and `PINERGY_PASSWORD` are set. They validate response
shapes (not specific values) against the live API.

---

## GitHub Actions

Two workflows are included:

- **`ci.yml`** — runs on every push and PR: lint + unit tests (Go 1.22, 1.23, and latest stable)
- **`integration.yml`** — runs on manual dispatch, or automatically when the `PINERGY_INTEGRATION_ENABLED` repository variable is set to `true` (uses `PINERGY_EMAIL` and `PINERGY_PASSWORD` secrets)

See `.github/workflows/` for configuration.

---

## Examples

- [`examples/basic/`](examples/basic/) — login, print balance, exit
- [`examples/monitor/`](examples/monitor/) — poll balance every 5 minutes

---

## Documentation

- [Authentication](docs/authentication.md)
- [Caching](docs/caching.md)
- [Error Handling](docs/error-handling.md)
- [Rate Limiting](docs/rate-limiting.md)

---

## Contributing

Pull requests are welcome. Please:

1. Add tests for new functionality
2. Run `make lint` and `make test` before submitting
3. Keep commit messages descriptive

---

## License

MIT — see [LICENSE](LICENSE).
