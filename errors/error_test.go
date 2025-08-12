package errors

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	err := New(ErrCodeInvalidRequest, "test error")

	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("expected code %d, got %d", ErrCodeInvalidRequest, err.Code)
	}

	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got '%s'", err.Message)
	}

	if err.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestErrorWithDetail(t *testing.T) {
	err := New(ErrCodeValidationFailed, "validation failed")
	err.WithDetail("field", "username").WithDetail("reason", "required")

	if err.Details["field"] != "username" {
		t.Errorf("expected detail field 'username', got '%v'", err.Details["field"])
	}

	if err.Details["reason"] != "required" {
		t.Errorf("expected detail reason 'required', got '%v'", err.Details["reason"])
	}
}

func TestErrorWithCause(t *testing.T) {
	originalErr := errors.New("original error")
	err := New(ErrCodeInternalError, "wrapped error").WithCause(originalErr)

	if err.Cause != originalErr {
		t.Error("cause should be set to original error")
	}

	if !errors.Is(err, originalErr) {
		t.Error("errors.Is should return true for wrapped error")
	}
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "error without cause",
			err:      New(ErrCodeTimeout, "request timeout"),
			expected: "[3] request timeout",
		},
		{
			name:     "error with cause",
			err:      New(ErrCodeServiceUnavailable, "service down").WithCause(errors.New("connection refused")),
			expected: "[2] service down: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	err1 := New(ErrCodeInvalidRequest, "invalid")
	err2 := New(ErrCodeInvalidRequest, "different message")

	if !err1.Is(err2) {
		t.Error("errors with same code should be considered equal")
	}

	err3 := New(ErrCodeTimeout, "timeout")
	if err1.Is(err3) {
		t.Error("errors with different codes should not be considered equal")
	}
}

func TestMultiError(t *testing.T) {
	multiErr := NewMultiError(nil)

	if multiErr.HasErrors() {
		t.Error("new multi error should not have errors")
	}

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	multiErr.Add(err1)
	multiErr.Add(err2)

	if !multiErr.HasErrors() {
		t.Error("multi error should have errors after adding")
	}

	if len(multiErr.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(multiErr.Errors))
	}

	expectedMsg := "2 errors: [error 1 error 2]"
	if multiErr.Error() != expectedMsg {
		t.Errorf("expected '%s', got '%s'", expectedMsg, multiErr.Error())
	}
}

func TestMultiErrorEmpty(t *testing.T) {
	multiErr := NewMultiError(nil)

	if multiErr.Error() != "no errors" {
		t.Errorf("expected 'no errors', got '%s'", multiErr.Error())
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		code ErrorCode
	}{
		{"ErrInvalidRequest", ErrInvalidRequest, ErrCodeInvalidRequest},
		{"ErrServiceUnavailable", ErrServiceUnavailable, ErrCodeServiceUnavailable},
		{"ErrTimeout", ErrTimeout, ErrCodeTimeout},
		{"ErrInternalError", ErrInternalError, ErrCodeInternalError},
		{"ErrUnauthorized", ErrUnauthorized, ErrCodeUnauthorized},
		{"ErrForbidden", ErrForbidden, ErrCodeForbidden},
		{"ErrNotFound", ErrNotFound, ErrCodeNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, tt.err.Code)
			}
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original")
	err := New(ErrCodeInternalError, "wrapped").WithCause(originalErr)

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Error("Unwrap should return the original error")
	}
}
