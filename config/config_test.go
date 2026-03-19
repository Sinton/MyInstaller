package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configContent := `packages:
  - name: Test Package
    install-command: npm install -g test-package
    version-check-command: npm view test-package version
    local-version-command: test-package --version
`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	// Load the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	// Verify the config
	if cfg == nil {
		t.Fatal("Config is nil")
	}
	
	if len(cfg.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(cfg.Packages))
	}
	
	pkg := cfg.Packages[0]
	if pkg.Name != "Test Package" {
		t.Errorf("Expected name 'Test Package', got '%s'", pkg.Name)
	}
	if pkg.InstallCommand != "npm install -g test-package" {
		t.Errorf("Expected install_command 'npm install -g test-package', got '%s'", pkg.InstallCommand)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	
	invalidContent := `packages:
  - name: test
    invalid yaml content [[[
`
	
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestLoadConfig_EmptyPath_CurrentDirectory(t *testing.T) {
	// Save current directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)
	
	// Create a temporary directory with config.yaml
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	
	configContent := `packages:
  - name: Test Package
    install-command: npm install -g test-package
    version-check-command: npm view test-package version
    local-version-command: test-package --version
`
	
	if err := os.WriteFile("config.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config.yaml: %v", err)
	}
	
	// Load config with empty path (should find config.yaml in current directory)
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	if cfg == nil {
		t.Fatal("Config is nil")
	}
	
	if len(cfg.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(cfg.Packages))
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "pkg1",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "pkg1 --version",
			},
			{
				Name:                "pkg2",
				InstallCommand:      "npm install -g pkg2",
				VersionCheckCommand: "npm view pkg2 version",
				LocalVersionCommand: "pkg2 --version",
			},
		},
	}
	
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestValidate_EmptyPackages(t *testing.T) {
	cfg := &Config{
		Packages: []Package{},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty packages list, got nil")
	}
	
	expectedMsg := "packages list cannot be empty"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidate_EmptyName(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "pkg1 --version",
			},
		},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}
	
	expectedMsg := "has empty 'name' field"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidate_EmptyInstallCommand(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "pkg1",
				InstallCommand:      "",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "pkg1 --version",
			},
		},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty install-command, got nil")
	}

	expectedMsg := "has empty 'install-command' field"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidate_EmptyVersionCheckCommand(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "pkg1",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "",
				LocalVersionCommand: "pkg1 --version",
			},
		},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty version-check-command, got nil")
	}

	expectedMsg := "has empty 'version-check-command' field"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidate_EmptyLocalVersionCommand(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "pkg1",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "",
			},
		},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty local-version-command, got nil")
	}

	expectedMsg := "has empty 'local-version-command' field"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

func TestValidate_DuplicateNames(t *testing.T) {
	cfg := &Config{
		Packages: []Package{
			{
				Name:                "pkg1",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "pkg1 --version",
			},
			{
				Name:                "pkg1",
				InstallCommand:      "npm install -g pkg1",
				VersionCheckCommand: "npm view pkg1 version",
				LocalVersionCommand: "pkg1 --version",
			},
		},
	}
	
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for duplicate packages, got nil")
	}
	
	expectedMsg := "duplicate package found"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message containing '%s', got: %v", expectedMsg, err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestLoadConfig_ValidFixture tests loading a valid config from testdata
func TestLoadConfig_ValidFixture(t *testing.T) {
	cfg, err := LoadConfig("../testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed with valid fixture: %v", err)
	}
	
	if cfg == nil {
		t.Fatal("Config is nil")
	}
	
	if len(cfg.Packages) != 2 {
		t.Fatalf("Expected 2 packages, got %d", len(cfg.Packages))
	}
	
	// Verify first package
	pkg1 := cfg.Packages[0]
	if pkg1.Name != "TypeScript" {
		t.Errorf("Expected name 'TypeScript', got '%s'", pkg1.Name)
	}
	
	// Verify second package
	pkg2 := cfg.Packages[1]
	if pkg2.Name != "pnpm Package Manager" {
		t.Errorf("Expected name 'pnpm Package Manager', got '%s'", pkg2.Name)
	}
}

// TestLoadConfig_InvalidYAMLSyntaxFixture tests loading invalid YAML from testdata
func TestLoadConfig_InvalidYAMLSyntaxFixture(t *testing.T) {
	_, err := LoadConfig("../testdata/invalid_yaml_syntax.yaml")
	if err == nil {
		t.Fatal("Expected error for invalid YAML syntax, got nil")
	}
	
	// Verify error message contains helpful suggestions
	if !contains(err.Error(), "YAML") {
		t.Errorf("Expected error message to mention YAML, got: %v", err)
	}
}

// TestLoadConfig_MalformedYAMLFixture tests loading malformed YAML (tabs instead of spaces)
func TestLoadConfig_MalformedYAMLFixture(t *testing.T) {
	_, err := LoadConfig("../testdata/malformed_yaml.yaml")
	if err == nil {
		t.Fatal("Expected error for malformed YAML, got nil")
	}
}

// TestValidate_EmptyPackagesFixture tests validation with empty packages list from testdata
func TestValidate_EmptyPackagesFixture(t *testing.T) {
	cfg, err := LoadConfig("../testdata/empty_packages.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for empty packages list, got nil")
	}
	
	if !contains(err.Error(), "packages list cannot be empty") {
		t.Errorf("Expected error about empty packages, got: %v", err)
	}
}

// TestValidate_MissingFieldsFixture tests validation with missing required fields from testdata
func TestValidate_MissingFieldsFixture(t *testing.T) {
	cfg, err := LoadConfig("../testdata/missing_fields.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing fields, got nil")
	}
	
	// Should fail on one of the missing required fields
	hasExpectedError := contains(err.Error(), "install-command") ||
		contains(err.Error(), "version-check-command") ||
		contains(err.Error(), "local-version-command")
	
	if !hasExpectedError {
		t.Errorf("Expected error about missing required fields, got: %v", err)
	}
}

// TestValidate_DuplicateNamesFixture tests validation with duplicate package names from testdata
func TestValidate_DuplicateNamesFixture(t *testing.T) {
	cfg, err := LoadConfig("../testdata/duplicate_names.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for duplicate packages, got nil")
	}
	
	if !contains(err.Error(), "duplicate package found") {
		t.Errorf("Expected error about duplicate packages, got: %v", err)
	}
	if !contains(err.Error(), "TypeScript") {
		t.Errorf("Expected error to mention 'TypeScript', got: %v", err)
	}
}

// TestLoadConfig_MultiplePackages tests loading config with multiple packages
func TestLoadConfig_MultiplePackages(t *testing.T) {
	cfg, err := LoadConfig("../testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	if len(cfg.Packages) != 2 {
		t.Fatalf("Expected 2 packages, got %d", len(cfg.Packages))
	}
	
	// Verify all packages have required fields
	for i, pkg := range cfg.Packages {
		if pkg.Name == "" {
			t.Errorf("Package %d has empty name", i)
		}
		if pkg.InstallCommand == "" {
			t.Errorf("Package %d has empty install_command", i)
		}
		if pkg.VersionCheckCommand == "" {
			t.Errorf("Package %d has empty version_check_command", i)
		}
		if pkg.LocalVersionCommand == "" {
			t.Errorf("Package %d has empty local_version_command", i)
		}
	}
}

// TestValidate_AllFieldsPresent tests validation passes when all fields are present
func TestValidate_AllFieldsPresent(t *testing.T) {
	cfg, err := LoadConfig("../testdata/valid_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass for valid config, got error: %v", err)
	}
}

// TestLoadConfig_ErrorMessages tests that error messages are helpful
func TestLoadConfig_ErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedInMsg  string
	}{
		{
			name:          "File not found",
			path:          "/nonexistent/config.yaml",
			expectedInMsg: "config file not found",
		},
		{
			name:          "Invalid YAML",
			path:          "../testdata/invalid_yaml_syntax.yaml",
			expectedInMsg: "YAML",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.path)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			
			if !contains(err.Error(), tt.expectedInMsg) {
				t.Errorf("Expected error message to contain '%s', got: %v", tt.expectedInMsg, err)
			}
		})
	}
}

// TestValidate_EdgeCases tests various edge cases in validation
func TestValidate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Single valid package",
			config: &Config{
				Packages: []Package{
					{
						Name:                "test",
						InstallCommand:      "npm install -g test",
						VersionCheckCommand: "npm view test version",
						LocalVersionCommand: "test --version",
					},
				},
			},
			shouldError: false,
		},
		{
			name: "Package with whitespace in name",
			config: &Config{
				Packages: []Package{
					{
						Name:                "test package",
						InstallCommand:      "npm install -g test",
						VersionCheckCommand: "npm view test version",
						LocalVersionCommand: "test --version",
					},
				},
			},
			shouldError: false, // Whitespace is allowed in names
		},
		{
			name: "Package with special characters",
			config: &Config{
				Packages: []Package{
					{
						Name:                "@scope/package",
						InstallCommand:      "npm install -g @scope/package",
						VersionCheckCommand: "npm view @scope/package version",
						LocalVersionCommand: "@scope/package --version",
					},
				},
			},
			shouldError: false, // Special characters are allowed
		},
		{
			name: "Multiple packages with all fields",
			config: &Config{
				Packages: []Package{
					{
						Name:                "pkg1",
						InstallCommand:      "npm install -g pkg1",
						VersionCheckCommand: "npm view pkg1 version",
						LocalVersionCommand: "pkg1 --version",
					},
					{
						Name:                "pkg2",
						InstallCommand:      "npm install -g pkg2",
						VersionCheckCommand: "npm view pkg2 version",
						LocalVersionCommand: "pkg2 --version",
					},
					{
						Name:                "pkg3",
						InstallCommand:      "npm install -g pkg3",
						VersionCheckCommand: "npm view pkg3 version",
						LocalVersionCommand: "pkg3 --version",
					},
				},
			},
			shouldError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.shouldError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			
			if tt.shouldError && err != nil && tt.errorMsg != "" {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// Helper function removed - using strings.Contains instead
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
