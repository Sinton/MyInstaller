// Package testutil provides testing utilities for my-pnpm-installer.
//
// This package contains:
//   - Mock implementations of interfaces for unit testing
//   - Helper functions for common test operations
//   - Test data generators
package testutil

import (
	"context"
	"io"
	"strings"
)

// MockCommandExecutor is a mock implementation of utils.CommandExecutor
// for unit testing. It allows fine-grained control over command execution
// behavior in tests.
type MockCommandExecutor struct {
	// ExecuteFunc is called when Execute is invoked
	ExecuteFunc func(ctx context.Context, command string) (string, error)

	// ExecuteWithStreamFunc is called when ExecuteWithStream is invoked
	ExecuteWithStreamFunc func(ctx context.Context, command string) (io.ReadCloser, error)

	// ParseCommandFunc is called when ParseCommand is invoked
	ParseCommandFunc func(command string) (string, []string)

	// ValidateCommandFunc is called when ValidateCommand is invoked
	ValidateCommandFunc func(command string) error

	// Call tracking
	ExecuteCalls        []ExecuteCall
	ExecuteWithStreamCalls []ExecuteWithStreamCall
	ParseCommandCalls   []ParseCommandCall
	ValidateCommandCalls []ValidateCommandCall
}

// ExecuteCall records a call to Execute
type ExecuteCall struct {
	Ctx     context.Context
	Command string
}

// ExecuteWithStreamCall records a call to ExecuteWithStream
type ExecuteWithStreamCall struct {
	Ctx     context.Context
	Command string
}

// ParseCommandCall records a call to ParseCommand
type ParseCommandCall struct {
	Command string
}

// ValidateCommandCall records a call to ValidateCommand
type ValidateCommandCall struct {
	Command string
}

// Execute implements utils.CommandExecutor
func (m *MockCommandExecutor) Execute(ctx context.Context, command string) (string, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, ExecuteCall{Ctx: ctx, Command: command})

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, command)
	}
	return "", nil
}

// ExecuteWithStream implements utils.CommandExecutor
func (m *MockCommandExecutor) ExecuteWithStream(ctx context.Context, command string) (io.ReadCloser, error) {
	m.ExecuteWithStreamCalls = append(m.ExecuteWithStreamCalls, ExecuteWithStreamCall{Ctx: ctx, Command: command})

	if m.ExecuteWithStreamFunc != nil {
		return m.ExecuteWithStreamFunc(ctx, command)
	}
	return nil, nil
}

// ParseCommand implements utils.CommandExecutor
func (m *MockCommandExecutor) ParseCommand(command string) (string, []string) {
	m.ParseCommandCalls = append(m.ParseCommandCalls, ParseCommandCall{Command: command})

	if m.ParseCommandFunc != nil {
		return m.ParseCommandFunc(command)
	}
	// Default implementation
	return "sh", []string{"-c", command}
}

// ValidateCommand implements utils.CommandExecutor
func (m *MockCommandExecutor) ValidateCommand(command string) error {
	m.ValidateCommandCalls = append(m.ValidateCommandCalls, ValidateCommandCall{Command: command})

	if m.ValidateCommandFunc != nil {
		return m.ValidateCommandFunc(command)
	}
	// Default: allow all commands (for backward compatibility)
	return nil
}

// Reset clears all call records
func (m *MockCommandExecutor) Reset() {
	m.ExecuteCalls = nil
	m.ExecuteWithStreamCalls = nil
	m.ParseCommandCalls = nil
	m.ValidateCommandCalls = nil
}

// NewMockCommandExecutor creates a new MockCommandExecutor with optional configuration
func NewMockCommandExecutor(opts ...MockOption) *MockCommandExecutor {
	mock := &MockCommandExecutor{}
	for _, opt := range opts {
		opt(mock)
	}
	return mock
}

// MockOption configures a MockCommandExecutor
type MockOption func(*MockCommandExecutor)

// WithExecuteFunc sets the ExecuteFunc
func WithExecuteFunc(fn func(ctx context.Context, command string) (string, error)) MockOption {
	return func(m *MockCommandExecutor) {
		m.ExecuteFunc = fn
	}
}

// WithExecuteWithStreamFunc sets the ExecuteWithStreamFunc
func WithExecuteWithStreamFunc(fn func(ctx context.Context, command string) (io.ReadCloser, error)) MockOption {
	return func(m *MockCommandExecutor) {
		m.ExecuteWithStreamFunc = fn
	}
}

// WithParseCommandFunc sets the ParseCommandFunc
func WithParseCommandFunc(fn func(command string) (string, []string)) MockOption {
	return func(m *MockCommandExecutor) {
		m.ParseCommandFunc = fn
	}
}

// WithAlwaysSucceed configures the mock to always succeed
func WithAlwaysSucceed(output string) MockOption {
	return WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
		return output, nil
	})
}

// WithAlwaysFail configures the mock to always fail
func WithAlwaysFail(err error) MockOption {
	return WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
		return "", err
	})
}

// WithSequence configures the mock to return different results for successive calls
func WithSequence(results ...string) MockOption {
	return WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
		if len(results) == 0 {
			return "", nil
		}
		result := results[0]
		results = results[1:]
		return result, nil
	})
}

// stringReadCloser wraps a string to implement io.ReadCloser
type stringReadCloser struct {
	reader io.Reader
}

func (s *stringReadCloser) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *stringReadCloser) Close() error {
	return nil
}

// NewStringReadCloser creates an io.ReadCloser from a string
func NewStringReadCloser(s string) io.ReadCloser {
	return &stringReadCloser{reader: strings.NewReader(s)}
}

// MockError is a simple error type for testing
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}

// NewMockError creates a new MockError
func NewMockError(message string) error {
	return &MockError{Message: message}
}
