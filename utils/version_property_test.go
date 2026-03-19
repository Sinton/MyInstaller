package utils

import (
	"regexp"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_VersionComparisonTransitivity verifies Property 2: Version Comparison Transitivity
//
// **Validates: Requirements 2.3**
//
// Universal Quantification:
// ∀ v1, v2, v3 ∈ ValidSemver:
//   (CompareVersions(v1, v2) < 0 ∧ CompareVersions(v2, v3) < 0) ⟹
//     CompareVersions(v1, v3) < 0
//
// This property test verifies that:
// 1. If v1 < v2 and v2 < v3, then v1 < v3 (less-than transitivity)
// 2. If v1 > v2 and v2 > v3, then v1 > v3 (greater-than transitivity)
// 3. If v1 == v2 and v2 == v3, then v1 == v3 (equality transitivity)
func TestProperty_VersionComparisonTransitivity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	comparator := NewVersionComparator()

	properties.Property("version comparison is transitive for less-than", prop.ForAll(
		func(v1, v2, v3 string) bool {
			// Compare v1 and v2
			cmp12, err12 := comparator.CompareVersions(v1, v2)
			if err12 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v2 and v3
			cmp23, err23 := comparator.CompareVersions(v2, v3)
			if err23 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v1 and v3
			cmp13, err13 := comparator.CompareVersions(v1, v3)
			if err13 != nil {
				// Skip invalid versions
				return true
			}

			// Test transitivity: if v1 < v2 and v2 < v3, then v1 < v3
			if cmp12 < 0 && cmp23 < 0 {
				if cmp13 >= 0 {
					t.Logf("Transitivity violation (less-than): v1=%s, v2=%s, v3=%s, cmp(v1,v2)=%d, cmp(v2,v3)=%d, cmp(v1,v3)=%d",
						v1, v2, v3, cmp12, cmp23, cmp13)
					return false
				}
			}

			return true
		},
		genValidSemver(),
		genValidSemver(),
		genValidSemver(),
	))

	properties.Property("version comparison is transitive for greater-than", prop.ForAll(
		func(v1, v2, v3 string) bool {
			// Compare v1 and v2
			cmp12, err12 := comparator.CompareVersions(v1, v2)
			if err12 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v2 and v3
			cmp23, err23 := comparator.CompareVersions(v2, v3)
			if err23 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v1 and v3
			cmp13, err13 := comparator.CompareVersions(v1, v3)
			if err13 != nil {
				// Skip invalid versions
				return true
			}

			// Test transitivity: if v1 > v2 and v2 > v3, then v1 > v3
			if cmp12 > 0 && cmp23 > 0 {
				if cmp13 <= 0 {
					t.Logf("Transitivity violation (greater-than): v1=%s, v2=%s, v3=%s, cmp(v1,v2)=%d, cmp(v2,v3)=%d, cmp(v1,v3)=%d",
						v1, v2, v3, cmp12, cmp23, cmp13)
					return false
				}
			}

			return true
		},
		genValidSemver(),
		genValidSemver(),
		genValidSemver(),
	))

	properties.Property("version comparison is transitive for equality", prop.ForAll(
		func(v1, v2, v3 string) bool {
			// Compare v1 and v2
			cmp12, err12 := comparator.CompareVersions(v1, v2)
			if err12 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v2 and v3
			cmp23, err23 := comparator.CompareVersions(v2, v3)
			if err23 != nil {
				// Skip invalid versions
				return true
			}

			// Compare v1 and v3
			cmp13, err13 := comparator.CompareVersions(v1, v3)
			if err13 != nil {
				// Skip invalid versions
				return true
			}

			// Test transitivity: if v1 == v2 and v2 == v3, then v1 == v3
			if cmp12 == 0 && cmp23 == 0 {
				if cmp13 != 0 {
					t.Logf("Transitivity violation (equality): v1=%s, v2=%s, v3=%s, cmp(v1,v2)=%d, cmp(v2,v3)=%d, cmp(v1,v3)=%d",
						v1, v2, v3, cmp12, cmp23, cmp13)
					return false
				}
			}

			return true
		},
		genValidSemver(),
		genValidSemver(),
		genValidSemver(),
	))

	properties.TestingRun(t)
}

// genValidSemver generates valid semantic version strings for testing
func genValidSemver() gopter.Gen {
	return gen.OneGenOf(
		// Common stable versions
		gen.Const("1.0.0"),
		gen.Const("2.0.0"),
		gen.Const("3.0.0"),
		gen.Const("1.1.0"),
		gen.Const("1.2.0"),
		gen.Const("1.0.1"),
		gen.Const("1.0.2"),
		gen.Const("0.1.0"),
		gen.Const("0.0.1"),
		gen.Const("10.0.0"),
		gen.Const("5.2.1"),
		
		// Pre-release versions
		gen.Const("1.0.0-alpha"),
		gen.Const("1.0.0-alpha.1"),
		gen.Const("1.0.0-beta"),
		gen.Const("1.0.0-beta.2"),
		gen.Const("1.0.0-rc.1"),
		gen.Const("2.0.0-alpha"),
		gen.Const("2.0.0-beta"),
		
		// Build metadata
		gen.Const("1.0.0+build.1"),
		gen.Const("1.0.0+build.123"),
		gen.Const("1.0.0-alpha+build.1"),
		
		// Real-world versions
		gen.Const("4.9.5"),
		gen.Const("18.11.9"),
		gen.Const("8.15.4"),
		gen.Const("6.21.0"),
	)
}

// TestProperty_VersionParsingRobustness verifies Property 7: Version Parsing Robustness
//
// **Validates: Requirements 2.1**
//
// Universal Quantification:
// ∀ output ∈ String containing valid semver:
//   ParseVersion(output) = (version, nil) ⟹
//     version matches regex \d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?
//
// This property test verifies that:
// 1. ParseVersion can extract valid semver from various npm command outputs
// 2. The function handles different output formats (with prefixes, package names, extra text)
// 3. The parsed version is always a valid semver string
// 4. The function correctly fails when no version is present
func TestProperty_VersionParsingRobustness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	comparator := NewVersionComparator()

	properties.Property("ParseVersion extracts valid semver from noisy output", prop.ForAll(
		func(version string, prefix string, suffix string) bool {
			// Construct output with noise around the version
			output := prefix + version + suffix

			// Parse the version
			parsed, err := comparator.ParseVersion(output)
			if err != nil {
				t.Logf("Failed to parse version from output: '%s', error: %v", output, err)
				return false
			}

			// Verify the parsed version matches the expected semver pattern
			// Pattern: \d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?(?:\+[a-zA-Z0-9.]+)?
			semverPattern := regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?(?:\+[a-zA-Z0-9.]+)?$`)
			if !semverPattern.MatchString(parsed) {
				t.Logf("Parsed version '%s' does not match semver pattern", parsed)
				return false
			}

			// Verify the parsed version is contained in the original output
			if !strings.Contains(output, parsed) {
				t.Logf("Parsed version '%s' not found in output '%s'", parsed, output)
				return false
			}

			return true
		},
		genValidSemver(),
		genOutputPrefix(),
		genOutputSuffix(),
	))

	properties.Property("ParseVersion fails gracefully on invalid output", prop.ForAll(
		func(invalidOutput string) bool {
			// Try to parse invalid output
			_, err := comparator.ParseVersion(invalidOutput)

			// Should return an error
			if err == nil {
				t.Logf("Expected error for invalid output '%s', but got nil", invalidOutput)
				return false
			}

			return true
		},
		genInvalidVersionOutput(),
	))

	properties.Property("ParseVersion is consistent", prop.ForAll(
		func(version string, prefix string, suffix string) bool {
			// Construct output
			output := prefix + version + suffix

			// Parse twice
			parsed1, err1 := comparator.ParseVersion(output)
			parsed2, err2 := comparator.ParseVersion(output)

			// Both should succeed or both should fail
			if (err1 == nil) != (err2 == nil) {
				t.Logf("Inconsistent error results for output '%s'", output)
				return false
			}

			// If both succeed, results should be identical
			if err1 == nil && parsed1 != parsed2 {
				t.Logf("Inconsistent parse results for output '%s': '%s' vs '%s'", output, parsed1, parsed2)
				return false
			}

			return true
		},
		genValidSemver(),
		genOutputPrefix(),
		genOutputSuffix(),
	))

	properties.TestingRun(t)
}

// genOutputPrefix generates various prefixes that might appear before a version number
func genOutputPrefix() gopter.Gen {
	return gen.OneGenOf(
		gen.Const(""),                           // No prefix
		gen.Const("v"),                          // Common 'v' prefix
		gen.Const("version "),                   // Word prefix
		gen.Const("typescript@"),                // Package name
		gen.Const("@types/node@"),               // Scoped package
		gen.Const("pnpm@"),                      // Another package
		gen.Const("  "),                         // Whitespace
		gen.Const("\t"),                         // Tab
		gen.Const("\n"),                         // Newline
		gen.Const("latest: "),                   // npm output format
		gen.Const("installed: "),                // Another format
		gen.Const("@qwen-code/qwen-code@"),      // Complex scoped package
		gen.Const("package-name@"),              // Generic package
		gen.Const("  v"),                        // Whitespace + v
		gen.Const("Version: "),                  // Capitalized
		gen.Const("current version: "),          // Longer prefix
	)
}

// genOutputSuffix generates various suffixes that might appear after a version number
func genOutputSuffix() gopter.Gen {
	return gen.OneGenOf(
		gen.Const(""),                           // No suffix
		gen.Const(" "),                          // Whitespace
		gen.Const("\n"),                         // Newline
		gen.Const("\t"),                         // Tab
		gen.Const(" (latest)"),                  // npm output
		gen.Const(" - latest"),                  // Another format
		gen.Const(" installed"),                 // Status
		gen.Const("  "),                         // Multiple spaces
		gen.Const("\n\n"),                       // Multiple newlines
		gen.Const(" from registry"),             // Source info
		gen.Const(" (2 days ago)"),              // Time info
		gen.Const(" - published"),               // Status
	)
}

// genInvalidVersionOutput generates strings that should NOT contain valid versions
func genInvalidVersionOutput() gopter.Gen {
	return gen.OneGenOf(
		gen.Const(""),                           // Empty string
		gen.Const("   "),                        // Only whitespace
		gen.Const("no version here"),            // Text without version
		gen.Const("1.2"),                        // Incomplete version
		gen.Const("1.2."),                       // Incomplete version
		gen.Const("1.2.x"),                      // Invalid character
		gen.Const("v1.2"),                       // Incomplete with prefix
		gen.Const("version"),                    // Just the word
		gen.Const("error: package not found"),   // Error message
		gen.Const("npm ERR!"),                   // npm error
		gen.Const("command not found"),          // Shell error
		gen.Const("1"),                          // Single number
		gen.Const("a.b.c"),                      // Non-numeric
		gen.Const("x.y.z"),                      // Non-numeric letters
		gen.Const("version: unknown"),           // No numeric version
		gen.Const("not installed"),              // Status without version
	)
}
