# Rate Limiting

## Overview

The client enforces a token-bucket rate limiter using
[`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate).
Every request — including retried requests — waits for a token before being
dispatched.

## Default parameters

| Parameter | Default | Description |
|---|---|---|
| Rate | 2 req/s | Sustained request rate |
| Burst | 5 | Maximum burst before throttling begins |

A burst of 5 means up to 5 requests can proceed immediately at startup or after
a period of inactivity, before the 2 req/s limit takes effect.

## How it works

The rate limiter is shared across all goroutines using the same `*Client`.
When the token bucket is empty, the goroutine waits until a token becomes
available. The wait respects the caller's `context.Context`:

```
goroutine A calls GetBalance → waits for token → dispatches HTTP → returns
goroutine B calls GetUsage   → waits for token → dispatches HTTP → returns
```

If a context is canceled or times out while waiting for a token, the call
returns immediately with `ErrCodeContextCanceled` or `ErrCodeContextDeadline`.

## Configuring the rate limiter

```go
import "golang.org/x/time/rate"

client := pinergy.NewClient(
    // 1 request per second with a burst of 2.
    pinergy.WithRateLimit(rate.Limit(1), 2),
)
```

Setting the rate to `rate.Inf` disables throttling:

```go
pinergy.WithRateLimit(rate.Inf, 0)
```

## Interaction with caching

Cached responses skip the HTTP request entirely, so they do not consume a
rate-limiter token. If you expect high concurrent read traffic, keeping caching
enabled reduces the effective request rate significantly.

## Interaction with retries

Each retry attempt consumes a token. With the default 2 req/s rate and up to 3
attempts (1 initial + 2 retries), a worst-case failure costs 3 tokens plus the
back-off delays.
