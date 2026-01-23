package steps

import (
	"regexp"
	"strings"
)

var urlPattern = regexp.MustCompile(`https?://\S+`)

func stripURLsFromText(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	cleaned := urlPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}
