package idgen

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// GenerateSecureID generates a cryptographically secure ID with the given prefix and length.
// This mirrors the llm-api format for consistency across services.
// Format: prefix_alphanumeric (e.g., conv_abc123xyz456, resp_def789ghi012)
func GenerateSecureID(prefix string, length int) (string, error) {
	// Use larger byte array for better entropy
	bytes := make([]byte, length*2) // Use more bytes to ensure we have enough entropy
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Generate alphanumeric string (numbers and lowercase letters only)
	const charset = "0123456789abcdefghijklmnopqrstuvwxyz"
	encoded := make([]byte, length)
	for i := 0; i < length; i++ {
		encoded[i] = charset[bytes[i]%36] // 36 = len(charset)
	}

	return fmt.Sprintf("%s_%s", prefix, string(encoded)), nil
}

// MustGenerateSecureID generates a cryptographically secure ID with the given prefix and length.
// It panics if generation fails (should never happen with crypto/rand).
func MustGenerateSecureID(prefix string, length int) string {
	id, err := GenerateSecureID(prefix, length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate secure ID: %v", err))
	}
	return id
}

// ValidateIDFormat validates that an ID has the expected format (prefix_alphanumeric).
func ValidateIDFormat(id, expectedPrefix string) bool {
	if !strings.HasPrefix(id, expectedPrefix+"_") {
		return false
	}

	// Extract the suffix after the prefix and underscore
	suffix := id[len(expectedPrefix)+1:]

	// Check that suffix is not empty and contains only valid characters
	if len(suffix) == 0 {
		return false
	}

	// Validate characters (numbers and lowercase letters only: 0-9, a-z)
	for _, char := range suffix {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}
