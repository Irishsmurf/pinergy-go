package pinergy

import (
	"strconv"
)

// ErrorCode identifies the category of a Pinergy API error.
type ErrorCode int

const (
	// ErrCodeUnknown is returned for unclassified errors.
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeUnauthorized is returned for HTTP 401 responses.
	ErrCodeUnauthorized
	// ErrCodeForbidden is returned for HTTP 403 responses.
	ErrCodeForbidden
	// ErrCodeNotFound is returned for HTTP 404 responses.
	ErrCodeNotFound
	// ErrCodeRateLimited is returned for HTTP 429 responses.
	ErrCodeRateLimited
	// ErrCodeServerError is returned for HTTP 5xx responses.
	ErrCodeServerError
	// ErrCodeInvalidResponse is returned when the response body cannot be decoded.
	ErrCodeInvalidResponse
	// ErrCodeContextCanceled is returned when the caller's context is canceled.
	ErrCodeContextCanceled
	// ErrCodeContextDeadline is returned when the caller's context deadline is exceeded.
	ErrCodeContextDeadline
	// ErrCodeNetworkError is returned for transient network-level errors.
	ErrCodeNetworkError
	// ErrCodeEmailNotFound is returned when CheckEmail finds the address is not registered.
	ErrCodeEmailNotFound
	// ErrCodeAuthRequired is returned when an authenticated endpoint is called before Login.
	ErrCodeAuthRequired
)

func (c ErrorCode) String() string {
	switch c {
	case ErrCodeUnknown:
		return "unknown"
	case ErrCodeUnauthorized:
		return "unauthorized"
	case ErrCodeForbidden:
		return "forbidden"
	case ErrCodeNotFound:
		return "not_found"
	case ErrCodeRateLimited:
		return "rate_limited"
	case ErrCodeServerError:
		return "server_error"
	case ErrCodeInvalidResponse:
		return "invalid_response"
	case ErrCodeContextCanceled:
		return "context_canceled"
	case ErrCodeContextDeadline:
		return "context_deadline"
	case ErrCodeNetworkError:
		return "network_error"
	case ErrCodeEmailNotFound:
		return "email_not_found"
	case ErrCodeAuthRequired:
		return "auth_required"
	default:
		// Using string concatenation and strconv.Itoa avoids fmt.Sprintf reflection overhead and allocations
		return "error_code(" + strconv.Itoa(int(c)) + ")"
	}
}

// APIError is returned for all Pinergy client errors.
// Use errors.As to extract it, or errors.Is against the sentinel variables.
type APIError struct {
	// Code identifies the error category.
	Code ErrorCode
	// StatusCode is the HTTP status code, or 0 if not applicable.
	StatusCode int
	// Message is a human-readable description.
	Message string
	// Err is the underlying wrapped error, if any.
	Err error
}

func (e *APIError) Error() string {
	// Using string concatenation and strconv.Itoa avoids fmt.Sprintf reflection overhead and allocations
	if e.StatusCode != 0 {
		return "pinergy: " + e.Code.String() + " (HTTP " + strconv.Itoa(e.StatusCode) + "): " + e.Message
	}
	if e.Message != "" {
		return "pinergy: " + e.Code.String() + ": " + e.Message
	}
	return "pinergy: " + e.Code.String()
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// Is allows errors.Is comparisons against sentinel APIError values.
// Two APIErrors are considered equal if their Code values match.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Sentinel errors for use with errors.Is.
var (
	// ErrUnauthorized is returned when the API rejects the auth token (HTTP 401).
	ErrUnauthorized = &APIError{Code: ErrCodeUnauthorized}
	// ErrAuthRequired is returned when Login has not been called before an authenticated endpoint.
	ErrAuthRequired = &APIError{Code: ErrCodeAuthRequired}
	// ErrEmailNotFound is returned when CheckEmail finds the address is not registered.
	ErrEmailNotFound = &APIError{Code: ErrCodeEmailNotFound}
	// ErrRateLimited is returned when the API returns HTTP 429.
	ErrRateLimited = &APIError{Code: ErrCodeRateLimited}
)
