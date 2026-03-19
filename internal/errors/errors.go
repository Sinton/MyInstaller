// Package errors provides application-level error types for my-pnpm-installer.
//
// This package defines:
//   - ErrorCode: Typed error codes for categorizing errors
//   - AppError: Application error with code, message, and context
//   - ErrorCategory: Error categories for grouping related errors
//
// Example usage:
//
//	err := &AppError{
//	    Code:    ErrConfigNotFound,
//	    Message: "configuration file not found",
//	    Cause:   os.ErrNotExist,
//	    Context: map[string]interface{}{"path": "/etc/config.yaml"},
//	}
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode is a typed error code for categorizing application errors
type ErrorCode string

// Error implements the error interface for ErrorCode
func (e ErrorCode) Error() string {
	return string(e)
}

// Application Error Codes
const (
	// Config Errors
	ErrConfigNotFound     ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrConfigValidation   ErrorCode = "CONFIG_VALIDATION"

	// Command Execution Errors
	ErrCommandExecution   ErrorCode = "COMMAND_EXECUTION"
	ErrCommandTimeout     ErrorCode = "COMMAND_TIMEOUT"
	ErrCommandNotFound    ErrorCode = "COMMAND_NOT_FOUND"
	ErrCommandSecurity    ErrorCode = "COMMAND_SECURITY"

	// Version Errors
	ErrVersionParse       ErrorCode = "VERSION_PARSE"
	ErrVersionCompare     ErrorCode = "VERSION_COMPARE"

	// Package Errors
	ErrPackageNotFound    ErrorCode = "PACKAGE_NOT_FOUND"
	ErrPackageInstall     ErrorCode = "PACKAGE_INSTALL"
	ErrPackageCheck       ErrorCode = "PACKAGE_CHECK"

	// Network Errors
	ErrNetworkTimeout     ErrorCode = "NETWORK_TIMEOUT"
	ErrNetworkUnavailable ErrorCode = "NETWORK_UNAVAILABLE"

	// UI Errors
	ErrUIRender           ErrorCode = "UI_RENDER"
	ErrUITerminal         ErrorCode = "UI_TERMINAL"
)

// ErrorCategory represents a category of errors for grouping
type ErrorCategory string

const (
	CategoryConfig    ErrorCategory = "CONFIG"
	CategoryCommand   ErrorCategory = "COMMAND"
	CategoryVersion   ErrorCategory = "VERSION"
	CategoryPackage   ErrorCategory = "PACKAGE"
	CategoryNetwork   ErrorCategory = "NETWORK"
	CategoryUI        ErrorCategory = "UI"
	CategoryInternal  ErrorCategory = "INTERNAL"
)

// AppError represents an application-level error with additional context
type AppError struct {
	// Code is the error code for programmatic handling
	Code ErrorCode

	// Message is a human-readable error message
	Message string

	// Cause is the underlying error that caused this error (if any)
	Cause error

	// Category is the error category for grouping
	Category ErrorCategory

	// Context provides additional key-value context about the error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)

	if len(e.Context) > 0 {
		msg += fmt.Sprintf(" (%v)", e.Context)
	}

	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}

	return msg
}

// Unwrap returns the underlying cause (for errors.Is/As support)
func (e *AppError) Unwrap() error {
	return e.Cause
}

// AppErrorOption is a functional option for configuring AppError
type AppErrorOption func(*AppError)

// WithCause sets the underlying cause of the error
func WithCause(cause error) AppErrorOption {
	return func(e *AppError) {
		e.Cause = cause
	}
}

// WithContext adds a key-value pair to the error context
func WithContext(key string, value interface{}) AppErrorOption {
	return func(e *AppError) {
		if e.Context == nil {
			e.Context = make(map[string]interface{})
		}
		e.Context[key] = value
	}
}

// WithCategory sets the error category
func WithCategory(category ErrorCategory) AppErrorOption {
	return func(e *AppError) {
		e.Category = category
	}
}

// NewAppError creates a new AppError with the given code and message
func NewAppError(code ErrorCode, message string, opts ...AppErrorOption) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(err)
	}

	return err
}

// WrapError wraps an existing error with additional context
func WrapError(err error, code ErrorCode, message string, opts ...AppErrorOption) *AppError {
	opts = append(opts, WithCause(err))
	return NewAppError(code, message, opts...)
}

// IsAppError checks if an error is an AppError with the given code
func IsAppError(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode extracts the ErrorCode from an AppError if present
func GetErrorCode(err error) (ErrorCode, bool) {
	if err == nil {
		return "", false
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code, true
	}
	return "", false
}
