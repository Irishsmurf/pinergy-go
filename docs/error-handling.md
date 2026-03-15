# Error Handling

## The `*APIError` type

All errors returned by this library are either `*APIError` values or wrap one.
`*APIError` implements `error`, `Unwrap() error`, and a custom `Is` method for
use with `errors.Is`.

```go
type APIError struct {
    Code       ErrorCode // category of the error
    StatusCode int       // HTTP status code, or 0 if not applicable
    Message    string    // human-readable description
    Err        error     // wrapped underlying error, if any
}
```

## Checking errors

### Quick check with sentinel errors

```go
bal, err := client.GetBalance(ctx)
if err != nil {
    switch {
    case errors.Is(err, pinergy.ErrAuthRequired):
        // Call Login() first.
    case errors.Is(err, pinergy.ErrUnauthorized):
        // Token expired — re-login.
    case errors.Is(err, pinergy.ErrRateLimited):
        // Slow down.
    default:
        log.Printf("unexpected error: %v", err)
    }
}
```

### Full inspection with errors.As

```go
var apiErr *pinergy.APIError
if errors.As(err, &apiErr) {
    log.Printf("error code: %s", apiErr.Code)
    if apiErr.StatusCode != 0 {
        log.Printf("HTTP status: %d", apiErr.StatusCode)
    }
    log.Printf("message: %s", apiErr.Message)
}
```

## Sentinel errors

| Variable | `ErrorCode` | When returned |
|---|---|---|
| `ErrAuthRequired` | `ErrCodeAuthRequired` | Authenticated endpoint called before `Login` |
| `ErrUnauthorized` | `ErrCodeUnauthorized` | API rejected the auth token (HTTP 401) |
| `ErrEmailNotFound` | `ErrCodeEmailNotFound` | `CheckEmail`: address is not registered |
| `ErrRateLimited` | `ErrCodeRateLimited` | API returned HTTP 429 |

## `ErrorCode` reference

| Code | Value | Description |
|---|---|---|
| `ErrCodeUnknown` | 0 | Unclassified error |
| `ErrCodeUnauthorized` | 1 | HTTP 401 |
| `ErrCodeForbidden` | 2 | HTTP 403 |
| `ErrCodeNotFound` | 3 | HTTP 404 |
| `ErrCodeRateLimited` | 4 | HTTP 429 |
| `ErrCodeServerError` | 5 | HTTP 5xx |
| `ErrCodeInvalidResponse` | 6 | JSON decode failure |
| `ErrCodeContextCanceled` | 7 | `context.Canceled` |
| `ErrCodeContextDeadline` | 8 | `context.DeadlineExceeded` |
| `ErrCodeNetworkError` | 9 | Transient `net.Error` |
| `ErrCodeEmailNotFound` | 10 | `CheckEmail`: not registered |
| `ErrCodeAuthRequired` | 11 | No auth token stored |

## Which errors does the library retry?

The library automatically retries on `ErrCodeServerError` and `ErrCodeNetworkError`.
It **never** retries on 4xx responses, context errors, or `ErrCodeInvalidResponse`.

If you need custom retry logic on top of the library's built-in behaviour, wrap
your calls:

```go
for {
    bal, err := client.GetBalance(ctx)
    if err == nil {
        break
    }
    if errors.Is(err, pinergy.ErrUnauthorized) {
        client.Login(ctx, email, password) // refresh token
        continue
    }
    log.Fatal(err)
}
```

## Error wrapping

`*APIError` implements `Unwrap() error`, so the full Go error chain is
preserved. If a network error or context error is the root cause, you can
reach it with `errors.Unwrap` or `errors.As`.
