package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// GenerateRandomString generates a random hex string of specified length
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// SanitizeString removes leading/trailing whitespace and normalizes spaces
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}

// IsEmpty checks if a string is empty or only whitespace
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// CoalesceString returns the first non-empty string
func CoalesceString(strings ...string) string {
	for _, s := range strings {
		if !IsEmpty(s) {
			return s
		}
	}
	return ""
}

// StringPtr returns a pointer to the string
func StringPtr(s string) *string {
	return &s
}

// StringValue returns the value of a string pointer or empty string if nil
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
