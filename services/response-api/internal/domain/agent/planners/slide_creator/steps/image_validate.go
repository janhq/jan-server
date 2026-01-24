package steps

import (
	"net/url"
	"strings"
)

// blockedImageDomains contains domains known to block hotlinking or have issues.
var blockedImageDomains = map[string]bool{
	"facebook.com":      true,
	"fbcdn.net":         true,
	"instagram.com":     true,
	"twitter.com":       true,
	"x.com":             true,
	"linkedin.com":      true,
	"licdn.com":         true,
	"pinterest.com":     true,
	"pinimg.com":        true,
	"tiktok.com":        true,
	"snapchat.com":      true,
	"reddit.com":        true,
	"redd.it":           true,
	"discord.com":       true,
	"discordapp.com":    true,
	"whatsapp.com":      true,
	"telegram.org":      true,
	"gstatic.com":       true, // Google's CDN often has auth issues
	"googleusercontent": true, // Can have short-lived URLs
}

// ValidImageExtensions contains common image file extensions.
var validImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
	".bmp":  true,
	".ico":  true,
}

// ImageValidationResult contains the result of image URL validation.
type ImageValidationResult struct {
	IsValid     bool
	URL         string
	ErrorReason string
	IsWarning   bool
	WarningMsg  string
}

// ValidateImageURL checks if an image URL is valid and usable.
func ValidateImageURL(imageURL string) ImageValidationResult {
	imageURL = strings.TrimSpace(imageURL)

	// Empty URL
	if imageURL == "" {
		return ImageValidationResult{
			IsValid:     false,
			ErrorReason: "empty_url",
		}
	}

	// Parse URL
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return ImageValidationResult{
			IsValid:     false,
			URL:         imageURL,
			ErrorReason: "invalid_url_format",
		}
	}

	// Must be HTTP or HTTPS
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ImageValidationResult{
			IsValid:     false,
			URL:         imageURL,
			ErrorReason: "invalid_scheme",
		}
	}

	// Must have a host
	if parsed.Host == "" {
		return ImageValidationResult{
			IsValid:     false,
			URL:         imageURL,
			ErrorReason: "missing_host",
		}
	}

	// Check for blocked domains
	host := strings.ToLower(parsed.Host)
	for blockedDomain := range blockedImageDomains {
		if strings.Contains(host, blockedDomain) {
			return ImageValidationResult{
				IsValid:     false,
				URL:         imageURL,
				ErrorReason: "blocked_domain",
			}
		}
	}

	// Check for data URLs (base64)
	if strings.HasPrefix(strings.ToLower(imageURL), "data:") {
		return ImageValidationResult{
			IsValid:     false,
			URL:         imageURL,
			ErrorReason: "data_url_not_supported",
		}
	}

	// Check URL length (very long URLs are often problematic)
	if len(imageURL) > 2048 {
		return ImageValidationResult{
			IsValid:     false,
			URL:         imageURL,
			ErrorReason: "url_too_long",
		}
	}

	// Check for placeholder or test images
	lowerURL := strings.ToLower(imageURL)
	placeholderPatterns := []string{
		"placeholder",
		"example.com",
		"example.org",
		"test.com",
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"1x1",
		"pixel.gif",
		"spacer.gif",
		"blank.gif",
	}
	for _, pattern := range placeholderPatterns {
		if strings.Contains(lowerURL, pattern) {
			return ImageValidationResult{
				IsValid:     false,
				URL:         imageURL,
				ErrorReason: "placeholder_image",
			}
		}
	}

	// Success with optional warnings
	result := ImageValidationResult{
		IsValid: true,
		URL:     imageURL,
	}

	// Add warning for HTTP (not HTTPS)
	if scheme == "http" {
		result.IsWarning = true
		result.WarningMsg = "http_not_secure"
	}

	// Add warning if URL doesn't look like an image path
	path := strings.ToLower(parsed.Path)
	hasImageExtension := false
	for ext := range validImageExtensions {
		if strings.HasSuffix(path, ext) {
			hasImageExtension = true
			break
		}
	}

	// Check for common image CDN patterns
	imagePatterns := []string{
		"/image",
		"/img",
		"/photo",
		"/media",
		"/upload",
		"/cdn",
		"/static",
		"/assets",
	}
	looksLikeImage := hasImageExtension
	if !looksLikeImage {
		for _, pattern := range imagePatterns {
			if strings.Contains(path, pattern) {
				looksLikeImage = true
				break
			}
		}
	}

	if !looksLikeImage && !result.IsWarning {
		result.IsWarning = true
		result.WarningMsg = "may_not_be_image"
	}

	return result
}

// FilterValidImages filters a list of image assets, keeping only valid ones.
func FilterValidImages(assets []map[string]any) []map[string]any {
	valid := make([]map[string]any, 0, len(assets))

	for _, asset := range assets {
		imageURL := assetImageURL(asset)
		result := ValidateImageURL(imageURL)
		if result.IsValid {
			valid = append(valid, asset)
		}
	}

	return valid
}

// FilterValidSlideImages filters SlideImage list, keeping only valid ones.
func FilterValidSlideImages(images []SlideImage) []SlideImage {
	valid := make([]SlideImage, 0, len(images))

	for _, img := range images {
		result := ValidateImageURL(img.Src)
		if result.IsValid {
			valid = append(valid, img)
		}
	}

	return valid
}

// SanitizeImageURL cleans up an image URL.
func SanitizeImageURL(imageURL string) string {
	imageURL = strings.TrimSpace(imageURL)

	// Remove common tracking parameters
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return imageURL
	}

	// Clean up query parameters - remove tracking params
	query := parsed.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"fbclid", "gclid", "ref", "source", "tracking_id",
	}
	for _, param := range trackingParams {
		query.Del(param)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

// PreferredImageURL returns the best available image URL from an asset.
// Prefers full-size imageUrl over thumbnailUrl for final rendering.
func PreferredImageURL(asset map[string]any) string {
	// For rendering, prefer full-size image
	if img, ok := asset["imageUrl"].(string); ok && strings.TrimSpace(img) != "" {
		result := ValidateImageURL(img)
		if result.IsValid {
			return SanitizeImageURL(img)
		}
	}

	// Fall back to thumbnail
	if thumb, ok := asset["thumbnailUrl"].(string); ok && strings.TrimSpace(thumb) != "" {
		result := ValidateImageURL(thumb)
		if result.IsValid {
			return SanitizeImageURL(thumb)
		}
	}

	// Try source URL
	if source, ok := asset["source"].(map[string]any); ok {
		if url, ok := source["url"].(string); ok && strings.TrimSpace(url) != "" {
			result := ValidateImageURL(url)
			if result.IsValid {
				return SanitizeImageURL(url)
			}
		}
	}

	return ""
}
