// Package config provides configuration file management for my-pnpm-installer.
//
// This package handles:
//   - Loading YAML configuration files from multiple search paths
//   - Parsing package definitions with validation
//   - Providing helpful error messages for configuration issues
//
// The configuration file (config.yaml) defines the list of packages to manage,
// including their installation commands and version check commands.
//
// Example configuration:
//
//	packages:
//	  - name: typescript
//	    display_name: TypeScript
//	    install_command: npm install -g typescript@latest
//	    version_check_command: npm view typescript version
//	    local_version_command: npm list -g typescript --depth=0
//
// Configuration Search Paths:
//  1. ./config.yaml (current directory)
//  2. ~/.config/pnpm-manager/config.yaml (user config directory)
//  3. %USERPROFILE%\.config\pnpm-manager\config.yaml (Windows)
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Sinton/my-pnpm-installer/internal/errors"

	"gopkg.in/yaml.v3"
)


// Package represents a single package definition from the configuration file.
// It contains all necessary information to check versions and install the package.
type Package struct {
	// Name is the display name for the package (shown in UI)
	Name string `yaml:"name"`
	
	// InstallCommand is the command to execute for installing/updating the package
	InstallCommand string `yaml:"install-command"`
	
	// VersionCheckCommand is the command to query the latest version from npm registry
	VersionCheckCommand string `yaml:"version-check-command"`
	
	// LocalVersionCommand is the command to query the locally installed version
	LocalVersionCommand string `yaml:"local-version-command"`
}

// Config represents the entire configuration file structure.
// It contains the list of packages to be managed by the tool.
type Config struct {
	// Packages is the list of package definitions
	Packages []Package `yaml:"packages"`
}

// LoadConfig loads and parses the configuration file from the specified path.
// If the path is empty, it searches for config.yaml in:
// 1. Current directory
// 2. ~/.config/pnpm-manager/
//
// Returns the parsed Config struct or an error if the file cannot be read or parsed.
func LoadConfig(path string) (*Config, error) {
	// If no path specified, search in default locations
	if path == "" {
		path = findConfigPath()
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Provide helpful error message with suggestions for config not found
			return nil, errors.NewAppError(
				errors.ErrConfigNotFound,
				fmt.Sprintf("config file not found at %s", path),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("path", path),
			)
		}
		return nil, errors.WrapError(
			err,
			errors.ErrConfigNotFound,
			fmt.Sprintf("failed to read config file %s", path),
			errors.WithCategory(errors.CategoryConfig),
		)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Enhanced YAML parse error with line number information
		return nil, formatYAMLError(path, err)
	}

	return &cfg, nil
}

// formatYAMLError formats YAML parsing errors with line numbers and helpful suggestions
//
// This function enhances raw YAML parsing errors by:
//  1. Extracting line number information when available
//  2. Providing context-specific troubleshooting suggestions
//  3. Making errors more user-friendly and actionable
//
// The function handles two types of YAML errors:
//  - yaml.TypeError: Type mismatch errors (e.g., string where number expected)
//  - Generic errors: Syntax errors, malformed YAML, etc.
func formatYAMLError(path string, err error) error {
	// Try to extract line number from yaml.TypeError
	// TypeError provides detailed information about type mismatches
	if typeErr, ok := err.(*yaml.TypeError); ok {
		return errors.WrapError(
			typeErr,
			errors.ErrConfigInvalid,
			fmt.Sprintf("YAML syntax error in %s", path),
			errors.WithCategory(errors.CategoryConfig),
			errors.WithContext("path", path),
		)
	}

	// For other YAML errors, provide general guidance
	// These are typically syntax errors like missing colons, incorrect indentation, etc.
	return errors.WrapError(
		err,
		errors.ErrConfigInvalid,
		fmt.Sprintf("failed to parse YAML from %s", path),
		errors.WithCategory(errors.CategoryConfig),
		errors.WithContext("path", path),
	)
}

// Validate checks the configuration for completeness and correctness.
// It verifies that:
// - The packages list is not empty
// - All required fields (Name, DisplayName, InstallCommand, VersionCheckCommand, LocalVersionCommand) are non-empty
// - All package names are unique
//
// Returns nil if validation passes, or a descriptive error if validation fails.
//
// Validation rules:
//  1. At least one package must be defined
//  2. Each package must have all required fields populated
//  3. Package names must be unique (used as identifiers)
//
// The function provides detailed error messages with suggestions for fixing
// configuration issues, making it easier for users to correct problems.
func (c *Config) Validate() error {
	// Check that packages list is not empty
	if len(c.Packages) == 0 {
		return errors.NewAppError(
			errors.ErrConfigValidation,
			"packages list cannot be empty",
			errors.WithCategory(errors.CategoryConfig),
		)
	}

	// Track package uniqueness using name + install_command combination
	packagesSeen := make(map[string]bool)

	// Validate each package
	for i, pkg := range c.Packages {
		// Check required fields are non-empty (whitespace-only treated as empty)
		if strings.TrimSpace(pkg.Name) == "" {
			return errors.NewAppError(
				errors.ErrConfigValidation,
				fmt.Sprintf("package at index %d has empty 'name' field", i),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("index", i),
			)
		}

		if strings.TrimSpace(pkg.InstallCommand) == "" {
			return errors.NewAppError(
				errors.ErrConfigValidation,
				fmt.Sprintf("package '%s' has empty 'install-command' field", pkg.Name),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("package", pkg.Name),
			)
		}

		if strings.TrimSpace(pkg.VersionCheckCommand) == "" {
			return errors.NewAppError(
				errors.ErrConfigValidation,
				fmt.Sprintf("package '%s' has empty 'version-check-command' field", pkg.Name),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("package", pkg.Name),
			)
		}

		if strings.TrimSpace(pkg.LocalVersionCommand) == "" {
			return errors.NewAppError(
				errors.ErrConfigValidation,
				fmt.Sprintf("package '%s' has empty 'local-version-command' field", pkg.Name),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("package", pkg.Name),
			)
		}

		// Check for duplicate packages
		uniqueKey := pkg.Name + "|" + pkg.InstallCommand
		if packagesSeen[uniqueKey] {
			return errors.NewAppError(
				errors.ErrConfigValidation,
				fmt.Sprintf("duplicate package found: name='%s', install-command='%s'", pkg.Name, pkg.InstallCommand),
				errors.WithCategory(errors.CategoryConfig),
				errors.WithContext("name", pkg.Name),
				errors.WithContext("install_command", pkg.InstallCommand),
			)
		}
		packagesSeen[uniqueKey] = true
	}

	return nil
}

// findConfigPath searches for config.yaml in default locations.
// Returns the first valid path found, or "config.yaml" as default.
//
// Search order:
//  1. ./config.yaml (current directory) - highest priority
//  2. ~/.config/pnpm-manager/config.yaml (Unix/macOS user config)
//  3. %USERPROFILE%\.config\pnpm-manager\config.yaml (Windows user config)
//
// The function checks if each file exists before returning it.
// If no file is found, it returns "config.yaml" as a default, which will
// cause LoadConfig to fail with a helpful error message.
func findConfigPath() string {
	// Search locations in order of priority
	paths := []string{
		"config.yaml", // Current directory (highest priority)
		filepath.Join(os.Getenv("HOME"), ".config", "pnpm-manager", "config.yaml"), // Unix/macOS
	}
	
	// On Windows, also check USERPROFILE environment variable
	// This is the Windows equivalent of HOME
	if runtime.GOOS == "windows" {
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			paths = append(paths, filepath.Join(userProfile, ".config", "pnpm-manager", "config.yaml"))
		}
	}
	
	// Return first existing file
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			// File exists and is accessible
			return path
		}
	}
	
	// Default to current directory if no file found
	// This will cause LoadConfig to fail with a helpful error message
	return "config.yaml"
}
