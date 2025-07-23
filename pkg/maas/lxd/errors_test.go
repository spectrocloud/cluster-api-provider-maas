package lxd

import (
	"errors"
	"testing"
)

func TestLXDError_Error(t *testing.T) {
	tests := []struct {
		name     string
		lxdError *LXDError
		expected string
	}{
		{
			name: "error without host ID",
			lxdError: &LXDError{
				Type:    LXDErrorVMCreationFailed,
				Message: "failed to create VM",
			},
			expected: "LXD VMCreationFailed: failed to create VM",
		},
		{
			name: "error with host ID",
			lxdError: &LXDError{
				Type:    LXDErrorHostUnavailable,
				Message: "host is not responding",
				HostID:  "host-123",
			},
			expected: "LXD HostUnavailable [host:host-123]: host is not responding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lxdError.Error(); got != tt.expected {
				t.Errorf("LXDError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLXDError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	lxdErr := WrapLXDError(originalErr, LXDErrorVMCreationFailed, "VM creation failed")

	if unwrapped := lxdErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("LXDError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewLXDErrorf(t *testing.T) {
	err := NewLXDErrorf(LXDErrorInsufficientResources, "need %d cores but only %d available", 4, 2)
	expected := "LXD InsufficientResources: need 4 cores but only 2 available"

	if got := err.Error(); got != expected {
		t.Errorf("NewLXDErrorf() = %v, want %v", got, expected)
	}
}

func TestWithHost(t *testing.T) {
	err := NewLXDError(LXDErrorProfileNotFound, "profile not found").WithHost("host-456")
	expected := "LXD ProfileNotFound [host:host-456]: profile not found"

	if got := err.Error(); got != expected {
		t.Errorf("WithHost() = %v, want %v", got, expected)
	}
}

func TestWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"requested": 8,
		"available": 4,
	}
	err := NewLXDError(LXDErrorInsufficientResources, "not enough CPU").WithDetails(details)

	if err.Details == nil {
		t.Errorf("WithDetails() details = nil, want non-nil")
	}
}

func TestIsLXDError(t *testing.T) {
	lxdErr := NewLXDError(LXDErrorHostUnavailable, "host unavailable")
	regularErr := errors.New("regular error")

	tests := []struct {
		name      string
		err       error
		errorType LXDErrorType
		expected  bool
	}{
		{
			name:      "matching LXD error type",
			err:       lxdErr,
			errorType: LXDErrorHostUnavailable,
			expected:  true,
		},
		{
			name:      "non-matching LXD error type",
			err:       lxdErr,
			errorType: LXDErrorVMCreationFailed,
			expected:  false,
		},
		{
			name:      "regular error",
			err:       regularErr,
			errorType: LXDErrorHostUnavailable,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLXDError(tt.err, tt.errorType); got != tt.expected {
				t.Errorf("IsLXDError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "retryable - host unavailable",
			err:       NewLXDError(LXDErrorHostUnavailable, "host unavailable"),
			retryable: true,
		},
		{
			name:      "retryable - insufficient resources",
			err:       NewLXDError(LXDErrorInsufficientResources, "not enough resources"),
			retryable: true,
		},
		{
			name:      "retryable - VM creation failed",
			err:       NewLXDError(LXDErrorVMCreationFailed, "VM creation failed"),
			retryable: true,
		},
		{
			name:      "not retryable - profile not found",
			err:       NewLXDError(LXDErrorProfileNotFound, "profile not found"),
			retryable: false,
		},
		{
			name:      "not retryable - project not found",
			err:       NewLXDError(LXDErrorProjectNotFound, "project not found"),
			retryable: false,
		},
		{
			name:      "not retryable - regular error",
			err:       errors.New("regular error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.retryable {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestGetErrorDetails(t *testing.T) {
	details := map[string]int{"cores": 4}
	lxdErr := NewLXDError(LXDErrorInsufficientResources, "not enough cores").WithDetails(details)
	regularErr := errors.New("regular error")

	t.Run("LXD error with details", func(t *testing.T) {
		got := GetErrorDetails(lxdErr)
		if got == nil {
			t.Errorf("GetErrorDetails() = nil, want non-nil details")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		got := GetErrorDetails(regularErr)
		if got != nil {
			t.Errorf("GetErrorDetails() = %v, want nil", got)
		}
	})
}
