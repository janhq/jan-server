package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

// Endpoint represents a single backend URL for a provider.
type Endpoint struct {
	URL string `json:"url"`
	// Weight is reserved for future weighted round-robin routing.
	// Currently ignored by routers; all endpoints are treated equally.
	Weight int `json:"weight,omitempty"`
	// Healthy indicates endpoint availability. Read-only: set by health checker only.
	Healthy bool `json:"healthy,omitempty"`
	// Priority is reserved for future priority-based routing (lower = higher priority).
	// Currently ignored by routers.
	Priority int `json:"priority,omitempty"`
}

// EndpointList manages multiple endpoints for a provider.
type EndpointList []Endpoint

// ParseEndpoints builds an EndpointList from the provided string.
// Supported formats:
//   - Comma-separated URLs: "http://a:8101/v1, http://b:8101/v1"
//   - JSON array: [{"url":"http://a:8101/v1"},{"url":"http://b:8101/v1"}]
func ParseEndpoints(input string) (EndpointList, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	if strings.HasPrefix(input, "[") {
		return parseEndpointsJSON(input)
	}

	return parseEndpointsCSV(input)
}

func parseEndpointsJSON(input string) (EndpointList, error) {
	var raw []Endpoint
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON endpoint array: %w", err)
	}

	result := make(EndpointList, 0, len(raw))
	for _, ep := range raw {
		normalized, err := normalizeAndValidateURL(ep.URL)
		if err != nil {
			log.Warn().Str("url", ep.URL).Err(err).Msg("skipping invalid endpoint URL")
			continue
		}
		ep.URL = normalized
		if ep.Weight <= 0 {
			ep.Weight = 1
		}
		ep.Healthy = true
		result = append(result, ep)
	}
	return result, nil
}

func parseEndpointsCSV(input string) (EndpointList, error) {
	parts := strings.Split(input, ",")
	result := make(EndpointList, 0, len(parts))

	for _, part := range parts {
		normalized, err := normalizeAndValidateURL(part)
		if err != nil {
			if strings.TrimSpace(part) != "" {
				log.Warn().Str("url", part).Err(err).Msg("skipping invalid endpoint URL")
			}
			continue
		}
		result = append(result, Endpoint{
			URL:     normalized,
			Weight:  1,
			Healthy: true,
		})
	}
	return result, nil
}

// normalizeAndValidateURL trims, validates, and normalizes a URL.
// Requires http/https scheme and a host; removes trailing slash.
func normalizeAndValidateURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL must have http or https scheme: %s", rawURL)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("URL must have a host: %s", rawURL)
	}

	return strings.TrimSuffix(parsed.String(), "/"), nil
}

// GetHealthy returns only healthy endpoints.
func (el EndpointList) GetHealthy() EndpointList {
	if len(el) == 0 {
		return nil
	}
	healthy := make(EndpointList, 0, len(el))
	for _, ep := range el {
		if ep.Healthy {
			healthy = append(healthy, ep)
		}
	}
	return healthy
}

// URLs returns the URL strings from the list.
func (el EndpointList) URLs() []string {
	urls := make([]string, len(el))
	for i, ep := range el {
		urls[i] = ep.URL
	}
	return urls
}
