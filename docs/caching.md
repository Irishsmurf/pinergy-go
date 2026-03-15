# Caching

## Overview

The client includes an in-memory TTL cache that stores raw API response bytes.
When a cached response is available and not yet expired, the library returns it
immediately without making an HTTP request.

Caching is enabled by default with per-endpoint TTLs chosen to balance
freshness against API load.

## Default TTLs

| Endpoint | Default TTL | Rationale |
|---|---|---|
| `/api/balance/` | 60 seconds | Balances change only on top-up events |
| `/api/usage/` | 5 minutes | Daily/weekly/monthly data updates slowly |
| `/api/levelpayusage/` | 5 minutes | Same cadence as usage |
| `/api/compare/` | 15 minutes | Comparison is computed periodically |
| `/api/configinfo/` | 30 minutes | Top-up config is nearly static |
| `/api/defaultsinfo/` | 30 minutes | House/heating reference, essentially static |
| `/api/activetopups/` | 2 minutes | May change if the user reconfigures |
| `/api/getnotif/` | 5 minutes | Notification prefs, stable |
| `/version.json` | 10 minutes | App version rarely changes |

## Implementation notes

The cache stores the raw JSON bytes returned by the API rather than decoded
structs. This approach:

- Avoids a generic cache that would require reflection or type assertions
- Keeps decoding close to the caller (one `json.Unmarshal` per cache hit)
- Means the cache is type-agnostic and shared across all endpoints

## Customising TTLs

Override the TTL for a specific endpoint:

```go
client := pinergy.NewClient(
    // Refresh balance every 30 seconds instead of 60.
    pinergy.WithCacheTTL("/api/balance/", 30*time.Second),
    // Disable caching for config info (always fresh).
    pinergy.WithCacheTTL("/api/configinfo/", 0),
)
```

A TTL of `0` disables caching for that endpoint.

## Disabling the cache entirely

```go
client := pinergy.NewClient(pinergy.WithCacheDisabled())
```

Every call will make a live HTTP request. Useful for testing or applications
where data currency is more important than request volume.

## Manual invalidation

After a significant state change (e.g. topping up credit), force a fresh read:

```go
// Invalidate only the balance cache entry.
client.CacheInvalidate("/api/balance/")

// Clear everything.
client.CacheFlush()
```

## Concurrency

The cache uses a `sync.RWMutex` internally. Multiple goroutines can read from
the cache simultaneously, and writes are serialised. The cache is fully safe
for concurrent use.
