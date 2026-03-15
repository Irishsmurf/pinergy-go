package pinergy

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		err  *APIError
		want string
	}{
		{
			&APIError{Code: ErrCodeUnauthorized, StatusCode: 401, Message: "bad token"},
			"pinergy: unauthorized (HTTP 401): bad token",
		},
		{
			&APIError{Code: ErrCodeAuthRequired, Message: "call Login first"},
			"pinergy: auth_required: call Login first",
		},
		{
			&APIError{Code: ErrCodeNetworkError},
			"pinergy: network_error",
		},
	}
	for _, tt := range tests {
		got := tt.err.Error()
		if got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}

func TestAPIError_Is(t *testing.T) {
	err := &APIError{Code: ErrCodeUnauthorized, StatusCode: 401, Message: "expired"}

	if !errors.Is(err, ErrUnauthorized) {
		t.Error("expected errors.Is(err, ErrUnauthorized) = true")
	}
	if errors.Is(err, ErrAuthRequired) {
		t.Error("expected errors.Is(err, ErrAuthRequired) = false")
	}
}

func TestAPIError_As(t *testing.T) {
	var apiErr *APIError
	err := &APIError{Code: ErrCodeServerError, StatusCode: 500, Message: "internal error"}

	if !errors.As(err, &apiErr) {
		t.Fatal("expected errors.As to succeed")
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	outer := &APIError{Code: ErrCodeUnknown, Err: inner}

	if !errors.Is(outer, inner) {
		t.Error("expected errors.Is to unwrap to inner error")
	}
}

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrCodeUnknown, "unknown"},
		{ErrCodeUnauthorized, "unauthorized"},
		{ErrCodeAuthRequired, "auth_required"},
		{ErrCodeEmailNotFound, "email_not_found"},
		{ErrorCode(999), "error_code(999)"},
	}
	for _, tt := range tests {
		got := tt.code.String()
		if got != tt.want {
			t.Errorf("ErrorCode(%d).String() = %q, want %q", int(tt.code), got, tt.want)
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify each sentinel can be compared with errors.Is.
	sentinels := []*APIError{ErrUnauthorized, ErrAuthRequired, ErrEmailNotFound, ErrRateLimited}
	for _, s := range sentinels {
		same := &APIError{Code: s.Code}
		if !errors.Is(same, s) {
			t.Errorf("errors.Is(%v, %v) = false, want true", same, s)
		}
		diff := &APIError{Code: s.Code + 100}
		if errors.Is(diff, s) {
			t.Errorf("errors.Is(%v, %v) = true, want false", diff, s)
		}
	}
}
