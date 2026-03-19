package testutil

import "strings"

// Contains checks if a string contains a substring (case-sensitive)
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsIgnoreCase checks if a string contains a substring (case-insensitive)
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Ptr returns a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}
