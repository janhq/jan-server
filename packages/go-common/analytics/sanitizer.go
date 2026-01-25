package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// PIILevel defines the level of PII sanitization
type PIILevel string

const (
	// PIILevelNone redacts all user content
	PIILevelNone PIILevel = "none"
	// PIILevelHashed hashes PII with tenant salt
	PIILevelHashed PIILevel = "hashed"
	// PIILevelFull performs no sanitization
	PIILevelFull PIILevel = "full"
)

// Sanitizer handles PII detection and sanitization for analytics
type Sanitizer struct {
	level      PIILevel
	tenantSalt string

	// Regex patterns for PII detection
	emailPattern      *regexp.Regexp
	phonePattern      *regexp.Regexp
	ssnPattern        *regexp.Regexp
	creditCardPattern *regexp.Regexp
	ipv4Pattern       *regexp.Regexp
	ipv6Pattern       *regexp.Regexp
}

// NewSanitizer creates a new PII sanitizer with tenant-specific salt
func NewSanitizer(level PIILevel, tenantID string) *Sanitizer {
	return &Sanitizer{
		level:             level,
		tenantSalt:        tenantID,
		emailPattern:      regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phonePattern:      regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`),
		ssnPattern:        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		creditCardPattern: regexp.MustCompile(`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`),
		ipv4Pattern:       regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		ipv6Pattern:       regexp.MustCompile(`\b(?:[A-Fa-f0-9]{1,4}:){7}[A-Fa-f0-9]{1,4}\b`),
	}
}

// SanitizePrompt sanitizes a user prompt based on the configured PII level
func (s *Sanitizer) SanitizePrompt(input string) string {
	switch s.level {
	case PIILevelNone:
		return "[REDACTED]"
	case PIILevelHashed:
		return s.hashPII(input)
	case PIILevelFull:
		return input
	default:
		// Default to hashed for safety
		return s.hashPII(input)
	}
}

// hashPII detects and hashes PII in the input string
func (s *Sanitizer) hashPII(input string) string {
	result := input

	// Replace emails
	result = s.emailPattern.ReplaceAllStringFunc(result, func(match string) string {
		return fmt.Sprintf("[EMAIL:%s]", s.hash(match))
	})

	// Replace phone numbers
	result = s.phonePattern.ReplaceAllStringFunc(result, func(match string) string {
		return fmt.Sprintf("[PHONE:%s]", s.hash(match))
	})

	// Replace SSNs
	result = s.ssnPattern.ReplaceAllStringFunc(result, func(match string) string {
		return "[SSN:REDACTED]"
	})

	// Replace credit cards
	result = s.creditCardPattern.ReplaceAllStringFunc(result, func(match string) string {
		return "[CC:REDACTED]"
	})

	// Replace IPv4 addresses
	result = s.ipv4Pattern.ReplaceAllStringFunc(result, func(match string) string {
		return fmt.Sprintf("[IP:%s]", s.hash(match))
	})

	// Replace IPv6 addresses
	result = s.ipv6Pattern.ReplaceAllStringFunc(result, func(match string) string {
		return fmt.Sprintf("[IP:%s]", s.hash(match))
	})

	return result
}

// hash creates a SHA-256 hash with tenant salt
func (s *Sanitizer) hash(data string) string {
	h := sha256.New()
	h.Write([]byte(data + s.tenantSalt))
	hashBytes := hex.EncodeToString(h.Sum(nil))
	// Return first 8 chars for readability
	return hashBytes[:8]
}
