package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// versionRegex is a pre-compiled regular expression for extracting semantic versions.
// It matches formats like: 1.2.3, v1.2.3, @scope/package@1.2.3, 1.2.3-alpha.1, 1.2.3+build.123
//
// Pattern breakdown:
//   - \d+\.\d+\.\d+           - Major.Minor.Patch (required)
//   - (?:-[a-zA-Z0-9.]+)?     - Optional pre-release (e.g., -alpha.1, -beta)
//   - (?:\+[a-zA-Z0-9.]+)?    - Optional build metadata (e.g., +build.123)
//
// Compiled once at package initialization for performance.
var versionRegex = regexp.MustCompile(`(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?(?:\+[a-zA-Z0-9.]+)?)`)

// VersionComparator provides version parsing and comparison capabilities
type VersionComparator interface {
	// ParseVersion extracts a semantic version from command output
	// Removes prefixes like "v", package names, and extra whitespace
	ParseVersion(output string) (string, error)

	// CompareVersions compares two version strings
	// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
	CompareVersions(v1, v2 string) (int, error)

	// IsNewerVersion returns true if v1 is newer than v2
	IsNewerVersion(v1, v2 string) bool
}

// versionComparator is the default implementation of VersionComparator
type versionComparator struct{}

// NewVersionComparator creates a new version comparator
func NewVersionComparator() VersionComparator {
	return &versionComparator{}
}

// ParseVersion extracts a semantic version from command output
// Handles various formats like: 1.2.3, v1.2.3, @scope/package@1.2.3
//
// The function uses a regex pattern to find version numbers in the output,
// which may contain additional text like package names, prefixes, or metadata.
//
// Supported formats:
//   - Basic: 1.2.3
//   - With 'v' prefix: v1.2.3
//   - Scoped packages: @scope/package@1.2.3
//   - Pre-release: 1.2.3-alpha.1, 1.2.3-beta.2
//   - Build metadata: 1.2.3+build.123, 1.2.3-alpha+build
//
// Algorithm:
//  1. Trim whitespace from output
//  2. Use pre-compiled regex to find first valid semver pattern
//  3. Return the matched version string
//  4. Return error if no valid version found
func (v *versionComparator) ParseVersion(output string) (string, error) {
	// Clean output - remove leading/trailing whitespace
	output = strings.TrimSpace(output)

	if output == "" {
		return "", fmt.Errorf("empty output - command did not return any version information\n\n"+
			"Suggestion: Verify the command is correct and the package exists")
	}

	// Extract version using pre-compiled regex (defined at package level)
	matches := versionRegex.FindStringSubmatch(output)

	if len(matches) < 2 {
		// No valid version found in output
		return "", fmt.Errorf("no valid semantic version found in output: '%s'\n\n"+
			"Expected format: X.Y.Z (e.g., 1.2.3, 2.0.0-beta.1)\n"+
			"Suggestion: Check if the command is returning the expected output format", output)
	}

	// Return the captured group (index 1) which contains the version string
	return matches[1], nil
}

// CompareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func (v *versionComparator) CompareVersions(v1, v2 string) (int, error) {
	ver1, err := semver.NewVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version v1 '%s': %w", v1, err)
	}

	ver2, err := semver.NewVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version v2 '%s': %w", v2, err)
	}

	return ver1.Compare(ver2), nil
}

// IsNewerVersion returns true if v1 is newer than v2
// Returns false if either version is invalid
func (v *versionComparator) IsNewerVersion(v1, v2 string) bool {
	result, err := v.CompareVersions(v1, v2)
	if err != nil {
		return false
	}
	return result > 0
}
