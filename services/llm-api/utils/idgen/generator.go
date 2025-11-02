package idgen

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateSecureID generates a cryptographically secure ID with the given prefix and length
// This is a pure utility function that only handles the crypto and formatting logic
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

// ValidateIDFormat validates that an ID has the expected format (prefix_alphanumeric)
// This is a pure utility function that only handles format validation
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

// HashKey256 generates HMAC-SHA256 hash of a key with a secret
func HashKey256(key string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
