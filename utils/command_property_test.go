package utils

import (
	"runtime"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_CrossPlatformCommandParsing verifies Property 6: Cross-Platform Command Parsing
//
// **Validates: Requirements 6.1**
//
// Universal Quantification:
// ∀ cmd ∈ String:
//   (name, args) = ParseCommand(cmd) ⟹
//     (runtime.GOOS = "windows" ⟹ name = "cmd.exe" ∧ args[0] = "/C") ∧
//     (runtime.GOOS ≠ "windows" ⟹ name = "sh" ∧ args[0] = "-c")
//
// This property test verifies that:
// 1. Commands are parsed correctly on different platforms (Windows vs Unix)
// 2. The parsed command structure is consistent and correct
// 3. The original command string is preserved in the arguments
func TestProperty_CrossPlatformCommandParsing(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	executor := NewCommandExecutor()

	properties.Property("ParseCommand returns correct shell for current platform", prop.ForAll(
		func(command string) bool {
			// Skip empty commands as they're not valid
			if command == "" {
				return true
			}

			name, args := executor.ParseCommand(command)

			// Verify structure based on current platform
			if runtime.GOOS == "windows" {
				// Windows: should use cmd.exe with /C flag
				if name != "cmd.exe" {
					t.Logf("Windows: expected name 'cmd.exe', got '%s'", name)
					return false
				}
				if len(args) < 2 {
					t.Logf("Windows: expected at least 2 args, got %d", len(args))
					return false
				}
				if args[0] != "/C" {
					t.Logf("Windows: expected first arg '/C', got '%s'", args[0])
					return false
				}
			} else {
				// Unix-like: should use sh with -c flag
				if name != "sh" {
					t.Logf("Unix: expected name 'sh', got '%s'", name)
					return false
				}
				if len(args) < 2 {
					t.Logf("Unix: expected at least 2 args, got %d", len(args))
					return false
				}
				if args[0] != "-c" {
					t.Logf("Unix: expected first arg '-c', got '%s'", args[0])
					return false
				}
			}

			// Verify the original command is preserved in args[1]
			if args[1] != command {
				t.Logf("Command not preserved: expected '%s', got '%s'", command, args[1])
				return false
			}

			return true
		},
		genCommandString(),
	))

	properties.Property("ParseCommand is deterministic", prop.ForAll(
		func(command string) bool {
			// Skip empty commands
			if command == "" {
				return true
			}

			// Parse the same command twice
			name1, args1 := executor.ParseCommand(command)
			name2, args2 := executor.ParseCommand(command)

			// Results should be identical
			if name1 != name2 {
				t.Logf("Non-deterministic name: '%s' vs '%s'", name1, name2)
				return false
			}

			if len(args1) != len(args2) {
				t.Logf("Non-deterministic args length: %d vs %d", len(args1), len(args2))
				return false
			}

			for i := range args1 {
				if args1[i] != args2[i] {
					t.Logf("Non-deterministic args[%d]: '%s' vs '%s'", i, args1[i], args2[i])
					return false
				}
			}

			return true
		},
		genCommandString(),
	))

	properties.Property("ParseCommand handles special characters correctly", prop.ForAll(
		func(command string) bool {
			// Skip empty commands
			if command == "" {
				return true
			}

			_, args := executor.ParseCommand(command)

			// The command should be passed as-is to the shell
			// The shell will handle special characters, quotes, etc.
			// We just verify the structure is correct
			if len(args) < 2 {
				t.Logf("Expected at least 2 args, got %d", len(args))
				return false
			}

			// Verify the command is preserved exactly
			if args[1] != command {
				t.Logf("Command modified: expected '%s', got '%s'", command, args[1])
				return false
			}

			return true
		},
		genCommandWithSpecialChars(),
	))

	properties.TestingRun(t)
}

// genCommandString generates realistic command strings for testing
func genCommandString() gopter.Gen {
	return gen.OneGenOf(
		// Simple commands
		gen.Const("echo hello"),
		gen.Const("npm --version"),
		gen.Const("node -v"),
		gen.Const("pnpm list -g"),
		
		// Commands with arguments
		gen.Const("npm view typescript version"),
		gen.Const("npm install -g typescript@latest"),
		gen.Const("npm list -g typescript --depth=0"),
		
		// Commands with pipes (Unix)
		gen.Const("npm view typescript version | grep -v beta"),
		gen.Const("echo test | cat"),
		
		// Commands with redirects
		gen.Const("echo test > output.txt"),
		gen.Const("npm list 2>&1"),
		
		// Commands with environment variables
		gen.Const("NODE_ENV=production npm start"),
		gen.Const("PATH=/usr/bin npm install"),
		
		// Commands with multiple statements
		gen.Const("echo line1 && echo line2"),
		gen.Const("cd /tmp && ls"),
		
		// Complex real-world commands
		gen.Const("npm config get registry"),
		gen.Const("pnpm add -g @typescript-eslint/parser"),
		gen.Const("npm view @types/node version"),
		
		// Generate random alphanumeric commands
		gen.AlphaString().Map(func(s string) string {
			if s == "" {
				return "echo test"
			}
			return "echo " + s
		}),
	)
}

// genCommandWithSpecialChars generates commands with special characters
// to test that they are properly preserved and passed to the shell
func genCommandWithSpecialChars() gopter.Gen {
	return gen.OneGenOf(
		// Commands with quotes
		gen.Const("echo 'hello world'"),
		gen.Const(`echo "hello world"`),
		gen.Const("echo \"test with \\\"quotes\\\"\""),
		
		// Commands with special shell characters
		gen.Const("echo $PATH"),
		gen.Const("echo $(date)"),
		gen.Const("echo `date`"),
		gen.Const("echo test; echo test2"),
		gen.Const("echo test | grep test"),
		gen.Const("echo test > /dev/null"),
		gen.Const("echo test 2>&1"),
		
		// Commands with spaces and tabs
		gen.Const("echo    multiple   spaces"),
		gen.Const("echo\ttab\tseparated"),
		
		// Commands with newlines (escaped)
		gen.Const("echo line1\\nline2"),
		
		// Commands with backslashes
		gen.Const("echo C:\\\\Windows\\\\System32"),
		gen.Const("echo /usr/local/bin"),
		
		// Commands with special characters
		gen.Const("echo !@#$%^&*()"),
		gen.Const("echo test=value"),
		gen.Const("echo [test]"),
		gen.Const("echo {test}"),
	)
}
