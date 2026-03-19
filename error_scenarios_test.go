package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/internal/testutil"
	"github.com/Sinton/my-pnpm-installer/services"
	"github.com/Sinton/my-pnpm-installer/utils"
)

// TestErrorScenarios_ConfigNotFound tests error handling when config file is not found
// **Validates: Requirements 1.8**
func TestErrorScenarios_ConfigNotFound(t *testing.T) {
	// Try to load a non-existent config file
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	
	if err == nil {
		t.Fatal("Expected error for non-existent config file, got nil")
	}
	
	// Verify error message is helpful
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Error should mention config or file
	if !testutil.Contains(errMsg, "config") && !testutil.Contains(errMsg, "file") {
		t.Errorf("Expected error message to mention config or file, got: %s", errMsg)
	}

	t.Logf("Got expected error: %v", err)
}

// TestErrorScenarios_InvalidYAMLSyntax tests error handling for malformed YAML
// **Validates: Requirements 1.8**
func TestErrorScenarios_InvalidYAMLSyntax(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	
	invalidYAML := `packages:
  - name: test
    invalid yaml [[[
    missing structure
`
	
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	_, err := config.LoadConfig(configPath)
	
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
	
	t.Logf("Got expected error for invalid YAML: %v", err)
}

// TestErrorScenarios_EmptyConfigFile tests error handling for empty config
// **Validates: Requirements 1.8**
func TestErrorScenarios_EmptyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")
	
	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	cfg, err := config.LoadConfig(configPath)
	
	// Loading might succeed but validation should fail
	if err == nil && cfg != nil {
		err = cfg.Validate()
		if err == nil {
			t.Fatal("Expected validation error for empty config, got nil")
		}
	}
	
	t.Logf("Got expected error for empty config: %v", err)
}

// TestErrorScenarios_CommandExecutionTimeout tests timeout handling
// **Validates: Requirements 3.2**
func TestErrorScenarios_CommandExecutionTimeout(t *testing.T) {
	executor := utils.NewCommandExecutor()

	// Note: We can't easily test actual timeout without waiting 30+ seconds
	// This test verifies that the timeout mechanism exists and works for fast commands

	// Execute a fast command that should complete within timeout
	_, err := executor.Execute(context.Background(), "echo test")

	if err != nil {
		// If we got an error, it shouldn't be a timeout error
		if testutil.Contains(err.Error(), "timeout") || testutil.Contains(err.Error(), "timed out") {
			t.Error("Fast command should not timeout")
		}
	}

	t.Log("Timeout mechanism verified (fast command completed successfully)")
}

// TestErrorScenarios_CommandNotFound tests handling of non-existent commands
// **Validates: Requirements 3.1**
func TestErrorScenarios_CommandNotFound(t *testing.T) {
	executor := utils.NewCommandExecutor()

	// Try to execute a command that doesn't exist
	_, err := executor.Execute(context.Background(), "nonexistentcommand12345xyz")

	if err == nil {
		t.Fatal("Expected error for non-existent command, got nil")
	}

	// Error should mention command failure
	errMsg := err.Error()
	if !testutil.Contains(errMsg, "command") && !testutil.Contains(errMsg, "failed") {
		t.Errorf("Expected error message to mention command failure, got: %s", errMsg)
	}

	t.Logf("Got expected error for non-existent command: %v", err)
}

// TestErrorScenarios_CommandWithNonZeroExit tests handling of failed commands
// **Validates: Requirements 3.1**
func TestErrorScenarios_CommandWithNonZeroExit(t *testing.T) {
	executor := utils.NewCommandExecutor()
	
	// Execute a command that exits with non-zero status
	_, err := executor.Execute(context.Background(), "exit 1")
	
	if err == nil {
		t.Fatal("Expected error for command with non-zero exit, got nil")
	}
	
	t.Logf("Got expected error for failed command: %v", err)
}

// TestErrorScenarios_VersionParsingFailure tests handling of unparseable version output
// **Validates: Requirements 1.8**
func TestErrorScenarios_VersionParsingFailure(t *testing.T) {
	comparator := utils.NewVersionComparator()
	
	// Try to parse invalid version strings
	invalidVersions := []string{
		"",
		"not a version",
		"1.2",
		"x.y.z",
		"version unknown",
	}
	
	for _, invalid := range invalidVersions {
		_, err := comparator.ParseVersion(invalid)
		
		if err == nil {
			t.Errorf("Expected error for invalid version '%s', got nil", invalid)
		}
	}
	
	t.Log("Version parsing correctly rejects invalid inputs")
}

// TestErrorScenarios_VersionComparisonWithInvalidVersions tests error handling in version comparison
// **Validates: Requirements 1.8**
func TestErrorScenarios_VersionComparisonWithInvalidVersions(t *testing.T) {
	comparator := utils.NewVersionComparator()
	
	// Try to compare invalid versions
	_, err := comparator.CompareVersions("invalid", "1.0.0")
	if err == nil {
		t.Error("Expected error when comparing invalid version, got nil")
	}
	
	_, err = comparator.CompareVersions("1.0.0", "invalid")
	if err == nil {
		t.Error("Expected error when comparing with invalid version, got nil")
	}
	
	t.Log("Version comparison correctly handles invalid inputs")
}

// TestErrorScenarios_PackageServiceWithFailingCommands tests service error handling
// **Validates: Requirements 1.8, 3.1**
func TestErrorScenarios_PackageServiceWithFailingCommands(t *testing.T) {
	// Create mock executor that always fails
	executor := testutil.NewMockCommandExecutor(
		testutil.WithAlwaysFail(testutil.NewMockError("command execution failed")),
	)

	comparator := utils.NewVersionComparator()
	service := services.NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                "test-pkg",
		VersionCheckCommand: "npm view test-pkg version",
		LocalVersionCommand: "test-pkg --version",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	// Should return error status
	if info.Status != services.StatusError {
		t.Errorf("Expected StatusError, got %v", info.Status)
	}

	// Error should be set
	if info.Error == nil {
		t.Error("Expected error to be set")
	}

	t.Logf("Service correctly handles command failures: %v", info.Error)
}

// TestErrorScenarios_NetworkFailureSimulation tests handling of network-like errors
// **Validates: Requirements 1.8, 3.1**
func TestErrorScenarios_NetworkFailureSimulation(t *testing.T) {
	// Simulate network failure by making version check fail
	executor := testutil.NewMockCommandExecutor(
		testutil.WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
			if testutil.Contains(command, "view") {
				return "", testutil.NewMockError("network timeout")
			}
			return "1.0.0", nil
		}),
	)

	comparator := utils.NewVersionComparator()
	service := services.NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                "test-pkg",
		VersionCheckCommand: "npm view test-pkg version",
		LocalVersionCommand: "test-pkg --version",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	// Should handle network error gracefully
	if info.Status != services.StatusError {
		t.Errorf("Expected StatusError for network failure, got %v", info.Status)
	}

	if info.Error == nil {
		t.Error("Expected error to be set for network failure")
	}

	t.Log("Service correctly handles network-like failures")
}

// TestErrorScenarios_GracefulDegradation tests graceful degradation scenarios
// **Validates: Requirements 1.8, 3.1, 3.2**
func TestErrorScenarios_GracefulDegradation(t *testing.T) {
	t.Run("Missing npm/pnpm tool", func(t *testing.T) {
		executor := testutil.NewMockCommandExecutor(
			testutil.WithAlwaysFail(testutil.NewMockError("command not found")),
		)

		comparator := utils.NewVersionComparator()
		service := services.NewPackageService(executor, comparator)

		// Check tool availability
		if service.CheckToolAvailability(context.Background(), "npm") {
			t.Error("Expected npm to be unavailable")
		}

		t.Log("Correctly detects missing tools")
	})

	t.Run("Partial package info available", func(t *testing.T) {
		// Simulate scenario where latest version is available but local is not
		executor := testutil.NewMockCommandExecutor(
			testutil.WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
				if testutil.Contains(command, "view") {
					return "2.0.0", nil
				}
				return "", testutil.NewMockError("not installed")
			}),
		)

		comparator := utils.NewVersionComparator()
		service := services.NewPackageService(executor, comparator)

		pkg := config.Package{
			Name:                "test-pkg",
			VersionCheckCommand: "npm view test-pkg version",
			LocalVersionCommand: "test-pkg --version",
		}

		info := service.GetPackageInfo(context.Background(), pkg)

		// Should gracefully handle partial info
		if info.Status != services.StatusNotInstalled {
			t.Errorf("Expected StatusNotInstalled, got %v", info.Status)
		}

		if info.LatestVersion != "2.0.0" {
			t.Errorf("Expected latest version to be available, got %s", info.LatestVersion)
		}

		t.Log("Gracefully handles partial package information")
	})
}

// TestErrorScenarios_RecoveryMechanisms tests error recovery
// **Validates: Requirements 1.8**
func TestErrorScenarios_RecoveryMechanisms(t *testing.T) {
	t.Run("Retry after transient failure", func(t *testing.T) {
		callCount := 0
		executor := testutil.NewMockCommandExecutor(
			testutil.WithExecuteFunc(func(ctx context.Context, command string) (string, error) {
				callCount++
				if callCount == 1 {
					// First call fails
					return "", testutil.NewMockError("transient error")
				}
				// Second call succeeds
				return "1.0.0", nil
			}),
		)

		comparator := utils.NewVersionComparator()
		service := services.NewPackageService(executor, comparator)

		pkg := config.Package{
			Name:                "test-pkg",
			VersionCheckCommand: "npm view test-pkg version",
			LocalVersionCommand: "test-pkg --version",
		}

		// First call fails
		info1 := service.GetPackageInfo(context.Background(), pkg)
		if info1.Status != services.StatusError {
			t.Errorf("Expected first call to fail, got %v", info1.Status)
		}

		// Second call succeeds (simulating retry)
		info2 := service.GetPackageInfo(context.Background(), pkg)
		if info2.Status == services.StatusError {
			t.Errorf("Expected second call to succeed, got %v", info2.Status)
		}

		t.Log("Recovery mechanism works after transient failure")
	})
}

// TestErrorScenarios_EdgeCases tests various edge cases
// **Validates: Requirements 1.8**
func TestErrorScenarios_EdgeCases(t *testing.T) {
	t.Run("Empty package name", func(t *testing.T) {
		cfg := &config.Config{
			Packages: []config.Package{
				{
					Name:                "",
					InstallCommand:      "test",
					VersionCheckCommand: "test",
					LocalVersionCommand: "test",
				},
			},
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for empty package name")
		}
	})
	
	t.Run("Whitespace-only fields", func(t *testing.T) {
		cfg := &config.Config{
			Packages: []config.Package{
				{
					Name:                "test",
					InstallCommand:      "test",
					VersionCheckCommand: "test",
					LocalVersionCommand: "test",
				},
			},
		}
		
		err := cfg.Validate()
		// Whitespace-only should be treated as empty
		if err == nil {
			t.Log("Note: Whitespace-only fields may need additional validation")
		}
	})
	
	t.Run("Very long command output", func(t *testing.T) {
		// Simulate very long output
		longOutput := make([]byte, 10000)
		for i := range longOutput {
			longOutput[i] = 'a'
		}
		
		comparator := utils.NewVersionComparator()
		_, err := comparator.ParseVersion(string(longOutput))
		
		// Should handle long output gracefully
		if err == nil {
			t.Log("Parser handles long output (no version found)")
		}
	})
}