package stringutils

import (
	"regexp"
	"strings"
	"unicode"
)

// Pre-compiled regex patterns for better performance
var (
	urlPattern          = regexp.MustCompile(`(?i)(https?://|ftp://|www\.)[^\s]+`)
	markdownLinkPattern = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
	emailPattern        = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	multiSpacePattern   = regexp.MustCompile(`\s+`)
)

// SanitizeTitleContent removes URLs, special characters, and cleans up the content for use as a title
func SanitizeTitleContent(content string) string {
	// Remove URLs (http, https, ftp, and www patterns)
	content = urlPattern.ReplaceAllString(content, "")

	// Remove markdown links [text](url)
	content = markdownLinkPattern.ReplaceAllString(content, "$1")

	// Remove email addresses
	content = emailPattern.ReplaceAllString(content, "")

	// Remove special characters but keep basic punctuation (.,!?-') and unicode letters/numbers
	var result strings.Builder
	for _, r := range content {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) ||
			r == '.' || r == ',' || r == '!' || r == '?' || r == '-' || r == '\'' {
			result.WriteRune(r)
		}
	}
	content = result.String()

	// Replace multiple spaces with single space (after special char removal)
	content = multiSpacePattern.ReplaceAllString(content, " ")

	// Trim whitespace and trailing punctuation for cleaner titles
	content = strings.TrimSpace(content)
	content = strings.TrimRight(content, " .,!?-'")

	return content
}

// TruncateTitle truncates a title to a maximum length, breaking at word boundaries
func TruncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}

	// Reserve space for ellipsis so the final string never exceeds maxLen
	ellipsis := "..."
	contentLimit := maxLen - len(ellipsis)
	if contentLimit < 0 {
		contentLimit = 0
	}

	truncated := title[:contentLimit]
	minLen := contentLimit / 2 // At least half the max content length

	// Prefer to cut on a word boundary when possible
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > minLen {
		truncated = strings.TrimRight(truncated[:lastSpace], " ")
	}

	return truncated + ellipsis
}

// GenerateTitle creates a clean, truncated title from content
func GenerateTitle(content string, maxLen int) string {
	sanitized := SanitizeTitleContent(content)
	if sanitized == "" {
		return ""
	}
	return TruncateTitle(sanitized, maxLen)
}
