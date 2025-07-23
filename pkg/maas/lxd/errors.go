package lxd

import (
	"fmt"

	"github.com/pkg/errors"
)

// LXDErrorType defines the type of LXD-related errors
type LXDErrorType string

const (
	// LXDErrorHostUnavailable indicates the selected LXD host is unavailable
	LXDErrorHostUnavailable LXDErrorType = "HostUnavailable"
	// LXDErrorInsufficientResources indicates insufficient resources for VM creation
	LXDErrorInsufficientResources LXDErrorType = "InsufficientResources"
	// LXDErrorProfileNotFound indicates the specified LXD profile was not found
	LXDErrorProfileNotFound LXDErrorType = "ProfileNotFound"
	// LXDErrorProjectNotFound indicates the specified LXD project was not found
	LXDErrorProjectNotFound LXDErrorType = "ProjectNotFound"
	// LXDErrorVMCreationFailed indicates VM composition failed
	LXDErrorVMCreationFailed LXDErrorType = "VMCreationFailed"
	// LXDErrorVMDeploymentFailed indicates VM deployment failed
	LXDErrorVMDeploymentFailed LXDErrorType = "VMDeploymentFailed"
	// LXDErrorNetworkConfiguration indicates network configuration issues
	LXDErrorNetworkConfiguration LXDErrorType = "NetworkConfiguration"
	// LXDErrorStorageConfiguration indicates storage configuration issues
	LXDErrorStorageConfiguration LXDErrorType = "StorageConfiguration"
)

// LXDError represents an LXD-specific error with detailed context
type LXDError struct {
	// Type is the category of the error
	Type LXDErrorType
	// Message is the human-readable error message
	Message string
	// HostID is the system ID of the involved LXD host (if applicable)
	HostID string
	// Details contains additional error context
	Details interface{}
	// Cause is the underlying error that caused this LXD error
	Cause error
}

// Error implements the error interface
func (e *LXDError) Error() string {
	if e.HostID != "" {
		return fmt.Sprintf("LXD %s [host:%s]: %s", e.Type, e.HostID, e.Message)
	}
	return fmt.Sprintf("LXD %s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error chain inspection
func (e *LXDError) Unwrap() error {
	return e.Cause
}

// NewLXDError creates a new LXD error with the specified type and message
func NewLXDError(errorType LXDErrorType, message string) *LXDError {
	return &LXDError{
		Type:    errorType,
		Message: message,
	}
}

// NewLXDErrorf creates a new LXD error with formatted message
func NewLXDErrorf(errorType LXDErrorType, format string, args ...interface{}) *LXDError {
	return &LXDError{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapLXDError wraps an existing error with LXD error context
func WrapLXDError(err error, errorType LXDErrorType, message string) *LXDError {
	return &LXDError{
		Type:    errorType,
		Message: message,
		Cause:   err,
	}
}

// WrapLXDErrorf wraps an existing error with formatted LXD error context
func WrapLXDErrorf(err error, errorType LXDErrorType, format string, args ...interface{}) *LXDError {
	return &LXDError{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WithHost adds host context to an LXD error
func (e *LXDError) WithHost(hostID string) *LXDError {
	e.HostID = hostID
	return e
}

// WithDetails adds additional details to an LXD error
func (e *LXDError) WithDetails(details interface{}) *LXDError {
	e.Details = details
	return e
}

// IsLXDError checks if an error is an LXD error of a specific type
func IsLXDError(err error, errorType LXDErrorType) bool {
	var lxdErr *LXDError
	if errors.As(err, &lxdErr) {
		return lxdErr.Type == errorType
	}
	return false
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	var lxdErr *LXDError
	if errors.As(err, &lxdErr) {
		switch lxdErr.Type {
		case LXDErrorHostUnavailable:
			return true
		case LXDErrorInsufficientResources:
			return true
		case LXDErrorVMCreationFailed:
			return true
		case LXDErrorVMDeploymentFailed:
			return true
		case LXDErrorNetworkConfiguration:
			return true
		default:
			return false
		}
	}
	return false
}

// GetErrorDetails extracts details from an LXD error
func GetErrorDetails(err error) interface{} {
	var lxdErr *LXDError
	if errors.As(err, &lxdErr) {
		return lxdErr.Details
	}
	return nil
}
