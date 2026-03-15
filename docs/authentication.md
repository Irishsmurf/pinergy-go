# Authentication

## Overview

Pinergy uses a custom token-based authentication scheme. After a successful
`Login`, the client receives an opaque `auth_token` string which must be sent
as a plain request header (`auth_token: <value>`) — not as a Bearer token — on
every subsequent authenticated request.

This library handles all of this transparently.

## The login flow

### 1. (Optional) Check email

The official Pinergy app calls `/api/checkemail` before rendering the login
screen to validate that the email address is registered.

```go
if err := client.CheckEmail(ctx, "user@example.com"); err != nil {
    if errors.Is(err, pinergy.ErrEmailNotFound) {
        fmt.Println("Email not registered")
    } else {
        log.Fatal(err)
    }
}
```

### 2. Login

`Login` sends the email and a **SHA-1 hash** of the plaintext password. The
library hashes the password internally — you always pass the plaintext.

```go
if err := client.Login(ctx, "user@example.com", "mypassword"); err != nil {
    log.Fatalf("login failed: %v", err)
}
```

On success, the auth token is stored inside the client and attached
automatically to every subsequent request.

### Why SHA-1?

The Pinergy API was designed to receive a SHA-1-hashed password. This is a
weakness in the API design (SHA-1 is not suitable for password storage), but
it is a fixed requirement that this library must follow. The library uses
Go's standard `crypto/sha1` package solely to comply with the API's protocol,
not for any security guarantee.

## Thread safety

`Login` acquires an exclusive write lock before storing the token, so it is
safe to call `Login` concurrently with other goroutines that are making API
calls. Subsequent calls replace the stored token atomically.

## Checking authentication status

```go
if client.IsAuthenticated() {
    fmt.Println("Ready to make API calls")
}
```

## Logout

`Logout` clears the stored token and flushes the response cache:

```go
client.Logout()
// client.IsAuthenticated() == false
```

Subsequent calls to authenticated endpoints return `pinergy.ErrAuthRequired`.

## Token lifetime

The token lifetime is not documented by Pinergy. In practice, tokens appear to
be session-lived. If the API returns HTTP 401 (surfaced as `pinergy.ErrUnauthorized`),
call `Login` again to obtain a fresh token.

```go
bal, err := client.GetBalance(ctx)
if errors.Is(err, pinergy.ErrUnauthorized) {
    if err := client.Login(ctx, email, password); err != nil {
        log.Fatal(err)
    }
    bal, err = client.GetBalance(ctx)
}
```
