// Package utils provides cross-platform utility functions for command execution
// and semantic version comparison.
//
// This package contains two main components:
//
// 1. CommandExecutor (command.go):
//   - Cross-platform command execution (Windows cmd.exe vs Unix sh)
//   - Command timeout handling (30-second default)
//   - Real-time output streaming for long-running commands
//   - Command validation for security (whitelist-based)
//
// 2. VersionComparator (version.go):
//   - Semantic version parsing from command output
//   - Version comparison using semver library
//   - Support for pre-release versions and build metadata
//
// The utilities are designed to be platform-agnostic and handle edge cases
// such as network timeouts, malformed version strings, and command failures.
//
// Example usage:
//
//	// Command execution
//	executor := utils.NewCommandExecutor()
//	output, err := executor.Execute("npm view typescript version")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Version comparison
//	comparator := utils.NewVersionComparator()
//	version, err := comparator.ParseVersion(output)
//	if comparator.IsNewerVersion("2.0.0", "1.5.0") {
//	    fmt.Println("Update available")
//	}
package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/Sinton/my-pnpm-installer/config"
)

// CommandExecutor provides cross-platform command execution capabilities
type CommandExecutor interface {
	// Execute executes a command and returns its output as a string.
	// The provided context controls the lifetime of the command.
	// If ctx has no deadline, a default 30-second timeout is applied.
	// The command is validated against a whitelist before execution.
	Execute(ctx context.Context, command string) (string, error)

	// ExecuteWithStream executes a command and returns a reader for real-time output.
	// The provided context controls the lifetime of the command.
	// The caller is responsible for closing the returned ReadCloser.
	// The command is validated against a whitelist before execution.
	ExecuteWithStream(ctx context.Context, command string) (io.ReadCloser, error)

	// ParseCommand parses a command string into executable name and arguments
	// for cross-platform compatibility (Windows cmd.exe vs Unix sh)
	ParseCommand(command string) (name string, args []string)

	// ValidateCommand validates a command against the security whitelist.
	// Returns nil if the command is allowed, or an error if it's blocked.
	ValidateCommand(command string) error
}

// commandExecutor is the default implementation of CommandExecutor
type commandExecutor struct {
	// allowedPrefixes defines the whitelist of allowed command prefixes
	// Commands must start with one of these prefixes to be executed
	allowedPrefixes []string
}

// CommandSecurityConfig holds security configuration for command execution
type CommandSecurityConfig struct {
	// AllowedPrefixes is a list of allowed command prefixes
	// Default: ["npm ", "pnpm ", "node ", "npx "]
	AllowedPrefixes []string
}

// NewCommandExecutor creates a new cross-platform command executor
// with default security settings
func NewCommandExecutor() CommandExecutor {
	prefixes := strings.Fields(config.DefaultCommandWhitelist)
	for i, prefix := range prefixes {
		// Add space suffix to each prefix for proper matching, e.g. "qwen "
		prefixes[i] = prefix + " "
	}
	return &commandExecutor{
		allowedPrefixes: prefixes,
	}
}

// NewCommandExecutorWithConfig creates a new command executor with custom security config
func NewCommandExecutorWithConfig(cfg CommandSecurityConfig) CommandExecutor {
	prefixes := cfg.AllowedPrefixes
	if len(prefixes) == 0 {
		// Use defaults if no prefixes specified
		prefixes = strings.Fields(config.DefaultCommandWhitelist)
		// Add space suffix to each prefix for proper matching
		for i, prefix := range prefixes {
			prefixes[i] = prefix + " "
		}
	}
	return &commandExecutor{
		allowedPrefixes: prefixes,
	}
}

// ParseCommand parses a command string into platform-appropriate executable and arguments
// Windows: returns ("cmd.exe", ["/C", command])
// Unix-like: returns ("sh", ["-c", command])
func (e *commandExecutor) ParseCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		// Windows: Use cmd.exe for compatibility
		return "cmd.exe", []string{"/C", command}
	}

	// Unix-like systems: Use sh
	return "sh", []string{"-c", command}
}

// ValidateCommand validates a command against the security whitelist.
// This is a critical security measure to prevent command injection attacks.
//
// Security Rules:
//  1. Command must start with an allowed prefix (npm, pnpm, node, npx)
//  2. Command cannot contain shell injection patterns (|, ;, &, $, `, etc.)
//  3. Command cannot contain file redirection operators (>, <, >>, etc.)
//
// Allowed prefixes can be customized via CommandSecurityConfig.
//
// Returns:
//   - nil if the command is safe and allowed
//   - CommandSecurityError if the command violates security rules
func (e *commandExecutor) ValidateCommand(command string) error {
	// Trim leading/trailing whitespace
	command = strings.TrimSpace(command)

	if command == "" {
		return &CommandSecurityError{
			Command: command,
			Reason:  "empty command is not allowed",
		}
	}

	// Check for dangerous shell injection patterns
	// These patterns could allow arbitrary command execution
	dangerousPatterns := []string{
		"|",  // Pipe
		";",  // Command separator
		"&",  // Background/AND
		"$",  // Variable expansion
		"`",  // Command substitution
		"(",  // Subshell
		")",  // Subshell
		"{",  // Command grouping
		"}",  // Command grouping
		"<",  // Input redirection
		">",  // Output redirection
		"\n", // Newline
		"\r", // Carriage return
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return &CommandSecurityError{
				Command: command,
				Reason:  fmt.Sprintf("command contains dangerous pattern: %q", pattern),
			}
		}
	}

	// Check if command starts with an allowed prefix
	// Add space to ensure we match complete words
	commandWithSpace := command + " "
	
	for _, prefix := range e.allowedPrefixes {
		if strings.HasPrefix(commandWithSpace, prefix) {
			return nil
		}
	}

	// Command doesn't match any allowed prefix
	return &CommandSecurityError{
		Command: command,
		Reason:  fmt.Sprintf("command must start with one of: %v", e.allowedPrefixes),
	}
}

// CommandSecurityError represents a command security validation failure
type CommandSecurityError struct {
	Command string
	Reason  string
}

func (e *CommandSecurityError) Error() string {
	return fmt.Sprintf("command security error: %s (command: %s)", e.Reason, e.Command)
}

// Execute executes a command with a 30-second timeout and returns its output
// Returns combined stdout and stderr output, trimmed of whitespace
//
// The function uses context.WithTimeout to enforce a 30-second deadline,
// preventing commands from hanging indefinitely due to network issues or
// unresponsive processes.
//
// Security: The command is validated against a whitelist before execution.
//
// Algorithm:
//  1. Validate command against security whitelist
//  2. Create context with 30-second timeout
//  3. Parse command for cross-platform execution
//  4. Execute command and capture combined output
//  5. Check for timeout and provide helpful error message
//  6. Return trimmed output or error
//
// Error handling:
//   - Security violation: Returns error indicating command is not allowed
//   - Timeout: Returns specific timeout error with troubleshooting suggestions
//   - Command failure: Returns error with command output for debugging
func (e *commandExecutor) Execute(ctx context.Context, command string) (string, error) {
	// Step 0: Validate command against security whitelist
	if err := e.ValidateCommand(command); err != nil {
		return "", err
	}

	// If the caller's context has no deadline, apply a default timeout
	// to prevent commands from hanging indefinitely.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.DefaultCommandTimeout)
		defer cancel()
	}

	// Parse command for cross-platform execution
	// Windows: cmd.exe /C <command>
	// Unix: sh -c <command>
	name, args := e.ParseCommand(command)

	// Create command with context for timeout support
	cmd := exec.CommandContext(ctx, name, args...)

	// CRITICAL: Detach stdin from the TUI's terminal.
	// When Bubble Tea is running, it puts the terminal in raw mode.
	// Child processes that inherit this raw-mode stdin can hang or behave
	// unexpectedly (e.g., pnpm may block waiting for input or get confused
	// by the terminal state). Setting stdin to /dev/null ensures child
	// processes run independently of the TUI's terminal.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}

	// Execute and capture combined stdout and stderr
	// CombinedOutput waits for the command to complete and returns all output
	output, err := cmd.CombinedOutput()

	// Check if the error was due to timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %v\n\n"+
			"Possible causes:\n"+
			"  1. Network connection is slow or unstable\n"+
			"  2. npm registry is not responding\n"+
			"  3. The command is taking longer than expected\n\n"+
			"Suggestions:\n"+
			"  - Check your internet connection\n"+
			"  - Try again later when network conditions improve\n"+
			"  - Verify npm registry is accessible: npm config get registry",
			config.CommandTimeoutErrorThreshold)
	}

	if err != nil {
		// Command failed for reasons other than timeout
		// Include the output in the error message for debugging
		return "", fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	// Return trimmed output (remove leading/trailing whitespace)
	return strings.TrimSpace(string(output)), nil
}

// ExecuteWithStream executes a command and returns a reader for streaming output.
// The returned ReadCloser combines both stdout and stderr concurrently using
// an io.Pipe, so neither stream blocks the other.
// The caller must close the returned ReadCloser when done.
//
// Security: The command is validated against a whitelist before execution.
//
// This function is designed for long-running commands (like package installations)
// where we want to display output in real-time rather than waiting for completion.
//
// Usage pattern:
//
//	reader, err := executor.ExecuteWithStream("npm install -g typescript")
//	if err != nil {
//	    return err
//	}
//	defer reader.Close()
//
//	scanner := bufio.NewScanner(reader)
//	for scanner.Scan() {
//	    fmt.Println(scanner.Text())
//	}
func (e *commandExecutor) ExecuteWithStream(ctx context.Context, command string) (io.ReadCloser, error) {
	// Step 0: Validate command against security whitelist
	if err := e.ValidateCommand(command); err != nil {
		return nil, err
	}

	// Parse command for cross-platform execution
	name, args := e.ParseCommand(command)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, name, args...)

	// CRITICAL: Detach stdin from the TUI's terminal.
	// When Bubble Tea is running, it puts the terminal in raw mode.
	// Child processes that inherit this raw-mode stdin can hang or behave
	// unexpectedly (e.g., pnpm may block waiting for input or get confused
	// by the terminal state). Setting stdin to /dev/null ensures child
	// processes run independently of the TUI's terminal.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}

	// Get stdout pipe for reading command output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Get stderr pipe for reading error output
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command execution
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Use io.Pipe to merge stdout and stderr concurrently.
	// Unlike io.MultiReader (which reads sequentially), this approach
	// uses goroutines so neither stream blocks the other.
	pr, pw := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(pw, stdout)
	}()

	go func() {
		defer wg.Done()
		io.Copy(pw, stderr)
	}()

	// Close the pipe writer once both streams are fully consumed
	go func() {
		wg.Wait()
		pw.Close()
	}()

	return &streamReader{
		reader: pr,
		cmd:    cmd,
	}, nil
}

// streamReader wraps an io.Reader and ensures the command is properly cleaned up.
// It implements io.ReadCloser to provide a unified interface for reading command output
// and waiting for command completion.
//
// The Close() method is critical because it:
//  1. Waits for the command to finish execution
//  2. Collects the exit status
//  3. Returns any errors that occurred during execution
//
// This ensures that even if the caller stops reading early, the command process
// is properly terminated and resources are released.
type streamReader struct {
	reader io.Reader   // Combined stdout and stderr reader
	cmd    *exec.Cmd   // The running command
}

// Read implements io.Reader
// Delegates to the underlying combined reader (stdout + stderr)
func (sr *streamReader) Read(p []byte) (n int, err error) {
	return sr.reader.Read(p)
}

// Close implements io.Closer and waits for the command to finish
// This method blocks until the command completes and returns any execution errors
func (sr *streamReader) Close() error {
	// Wait for command to complete
	// This is important to:
	//   1. Collect the exit status
	//   2. Release system resources
	//   3. Ensure child processes are terminated
	if err := sr.cmd.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
