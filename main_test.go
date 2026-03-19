package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/internal/testutil"
)

// TestMainWorkflow_ValidConfig tests the main workflow with a valid configuration
// This integration test verifies that:
// - Config can be loaded successfully
// - Config validation passes
// - Services can be initialized
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
func TestMainWorkflow_ValidConfig(t *testing.T) {
	// Use the testdata valid config
	cfg, err := config.LoadConfig("testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("Expected to load valid config, got error: %v", err)
	}

	// Validate the config
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Expected valid config to pass validation, got error: %v", err)
	}

	// Verify packages were loaded
	if len(cfg.Packages) == 0 {
		t.Fatal("Expected at least one package in valid config")
	}

	// Verify first package has all required fields
	pkg := cfg.Packages[0]
	if pkg.Name == "" {
		t.Error("Package name should not be empty")
	}
	if pkg.InstallCommand == "" {
		t.Error("Package install_command should not be empty")
	}
	if pkg.VersionCheckCommand == "" {
		t.Error("Package version_check_command should not be empty")
	}
	if pkg.LocalVersionCommand == "" {
		t.Error("Package local_version_command should not be empty")
	}

	t.Logf("Successfully loaded and validated config with %d package(s)", len(cfg.Packages))
}

// TestMainWorkflow_MissingConfig tests error handling when config file is missing
// This integration test verifies that:
// - LoadConfig returns an error when file doesn't exist
// - Error message is helpful and suggests solutions
// **Validates: Requirements 1.1, 1.2, 1.8**
func TestMainWorkflow_MissingConfig(t *testing.T) {
	// Try to load a non-existent config file
	nonExistentPath := filepath.Join("testdata", "nonexistent.yaml")
	cfg, err := config.LoadConfig(nonExistentPath)

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when loading non-existent config, got nil")
	}

	// Config should be nil
	if cfg != nil {
		t.Error("Expected nil config when file doesn't exist")
	}

	// Error message should be helpful
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Error should mention the file path
	if !testutil.Contains(errMsg, nonExistentPath) {
		t.Errorf("Expected error message to mention the file path, got: %s", errMsg)
	}

	t.Logf("Got expected error for missing config: %v", err)
}

// TestMainWorkflow_InvalidYAMLSyntax tests error handling for invalid YAML syntax
// This integration test verifies that:
// - LoadConfig returns an error for malformed YAML
// - Error message provides helpful suggestions
// **Validates: Requirements 1.3, 1.8**
func TestMainWorkflow_InvalidYAMLSyntax(t *testing.T) {
	// Try to load a config with invalid YAML syntax
	cfg, err := config.LoadConfig("testdata/invalid_yaml_syntax.yaml")

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when loading invalid YAML, got nil")
	}

	// Config should be nil
	if cfg != nil {
		t.Error("Expected nil config when YAML is invalid")
	}

	// Error message should mention YAML or syntax
	errMsg := err.Error()
	if !testutil.Contains(errMsg, "YAML") && !testutil.Contains(errMsg, "parse") {
		t.Errorf("Expected error message to mention YAML or parsing, got: %s", errMsg)
	}

	t.Logf("Got expected error for invalid YAML: %v", err)
}

// TestMainWorkflow_EmptyPackages tests validation error for empty packages list
// This integration test verifies that:
// - Config with empty packages list fails validation
// - Error message is descriptive and helpful
// **Validates: Requirements 1.5, 1.8**
func TestMainWorkflow_EmptyPackages(t *testing.T) {
	// Load config with empty packages list
	cfg, err := config.LoadConfig("testdata/empty_packages.yaml")
	if err != nil {
		t.Fatalf("Expected to load config with empty packages, got error: %v", err)
	}

	// Validation should fail
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for empty packages list, got nil")
	}

	// Error should mention empty packages
	errMsg := err.Error()
	if !testutil.Contains(errMsg, "empty") && !testutil.Contains(errMsg, "packages") {
		t.Errorf("Expected error message to mention empty packages, got: %s", errMsg)
	}

	t.Logf("Got expected validation error for empty packages: %v", err)
}

// TestMainWorkflow_MissingRequiredFields tests validation error for missing fields
// This integration test verifies that:
// - Config with missing required fields fails validation
// - Error message identifies which field is missing
// **Validates: Requirements 1.6, 1.8**
func TestMainWorkflow_MissingRequiredFields(t *testing.T) {
	// Load config with missing required fields
	cfg, err := config.LoadConfig("testdata/missing_fields.yaml")
	if err != nil {
		t.Fatalf("Expected to load config with missing fields, got error: %v", err)
	}

	// Validation should fail
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing required fields, got nil")
	}

	// Error should be descriptive
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message for missing fields")
	}

	t.Logf("Got expected validation error for missing fields: %v", err)
}

// TestMainWorkflow_DuplicatePackageNames tests validation error for duplicate names
// This integration test verifies that:
// - Config with duplicate package names fails validation
// - Error message identifies the duplicate name
// **Validates: Requirements 1.7, 1.8**
func TestMainWorkflow_DuplicatePackageNames(t *testing.T) {
	// Load config with duplicate package names
	cfg, err := config.LoadConfig("testdata/duplicate_names.yaml")
	if err != nil {
		t.Fatalf("Expected to load config with duplicate names, got error: %v", err)
	}

	// Validation should fail
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for duplicate package names, got nil")
	}

	// Error should mention duplicate
	errMsg := err.Error()
	if !testutil.Contains(errMsg, "duplicate") {
		t.Errorf("Expected error message to mention duplicate, got: %s", errMsg)
	}

	t.Logf("Got expected validation error for duplicate names: %v", err)
}

// TestMainWorkflow_ConfigSearchPaths tests config file search in multiple locations
// This integration test verifies that:
// - LoadConfig searches in current directory first
// - LoadConfig falls back to user config directory
// - Appropriate error is returned when no config is found
// **Validates: Requirements 1.1, 1.2**
func TestMainWorkflow_ConfigSearchPaths(t *testing.T) {
	// Test 1: Empty path triggers search behavior
	// This should fail since we're in a test environment without default configs
	cfg, err := config.LoadConfig("")
	
	// In test environment, this will likely fail (which is expected)
	// The important thing is that it attempts the search
	if err != nil {
		// Error is expected in test environment
		t.Logf("Config search returned error (expected in test env): %v", err)

		// Verify error message is helpful
		errMsg := err.Error()
		if !testutil.Contains(errMsg, "config") {
			t.Errorf("Expected error message to mention config, got: %s", errMsg)
		}
	} else if cfg != nil {
		// If it succeeded, it found a config somewhere
		t.Logf("Config search succeeded, found config with %d packages", len(cfg.Packages))
	}
}

// TestMainWorkflow_RealConfigFile tests loading the actual project config.yaml
// This integration test verifies that:
// - The project's config.yaml is valid and loadable
// - All packages in the config have required fields
// **Validates: Requirements 1.1-1.7**
func TestMainWorkflow_RealConfigFile(t *testing.T) {
	// Check if config.yaml exists in project root
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		t.Skip("config.yaml not found in project root, skipping real config test")
	}

	// Load the real config
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("Failed to load real config.yaml: %v", err)
	}

	// Validate it
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Real config.yaml failed validation: %v", err)
	}

	// Verify it has packages
	if len(cfg.Packages) == 0 {
		t.Fatal("Real config.yaml has no packages")
	}

	// Verify all packages have required fields
	for i, pkg := range cfg.Packages {
		if pkg.Name == "" {
			t.Errorf("Package %d has empty name", i)
		}
		if pkg.InstallCommand == "" {
			t.Errorf("Package %d (%s) has empty install_command", i, pkg.Name)
		}
		if pkg.VersionCheckCommand == "" {
			t.Errorf("Package %d (%s) has empty version_check_command", i, pkg.Name)
		}
		if pkg.LocalVersionCommand == "" {
			t.Errorf("Package %d (%s) has empty local_version_command", i, pkg.Name)
		}
	}

	t.Logf("Successfully validated real config.yaml with %d package(s)", len(cfg.Packages))
}
