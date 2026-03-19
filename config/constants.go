// Package config provides configuration constants for my-pnpm-installer.
//
// This package defines:
//   - Application-wide constants for UI, timeouts, and limits
//   - Default values for configuration options
//   - Security-related constants
package config

import "time"

// UI Constants
const (
	// MinTerminalWidth is the minimum terminal width required for proper UI display
	MinTerminalWidth = 80

	// MinTerminalHeight is the minimum terminal height required for proper UI display
	MinTerminalHeight = 24

	// MaxLogLines is the maximum number of log lines to keep in memory
	// Older lines are discarded to prevent memory overflow
	MaxLogLines = 1000
)

// Command Execution Constants
const (
	// DefaultCommandTimeout is the default timeout for command execution
	DefaultCommandTimeout = 60 * time.Second

	// CommandTimeoutErrorThreshold is the threshold for considering a command timed out
	// (used for error message formatting)
	CommandTimeoutErrorThreshold = 60 * time.Second
)

// Cache Constants
const (
	// DefaultCacheTTL is the default time-to-live for cached package information
	DefaultCacheTTL = 5 * time.Minute

	// PrefetchDelay is the delay between prefetching package information
	PrefetchDelay = 500 * time.Millisecond
)

// Security Constants
const (
	// DefaultCommandWhitelist is the default list of allowed command prefixes
	// Commands must start with one of these prefixes to be executed
	DefaultCommandWhitelist = "npm pnpm node npx qwen claude gemini ccr ccusage"
)

// Buffer Sizes
const (
	// DefaultLogBufferSize is the default buffer size for log channels
	DefaultLogBufferSize = 100

	// DefaultErrorBufferSize is the default buffer size for error channels
	DefaultErrorBufferSize = 1
)

// Validation Constants
const (
	// MaxPackageNameLength is the maximum allowed length for package names
	MaxPackageNameLength = 256

	// MaxCommandLength is the maximum allowed length for install commands
	MaxCommandLength = 1024
)
