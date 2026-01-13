package search

import (
	"fmt"
	"strings"

	domainsearch "jan-server/services/mcp-tools/internal/domain/search"

	"github.com/rs/zerolog/log"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// ValidateSearchResponse checks if a search response is valid and has meaningful content
func ValidateSearchResponse(resp *domainsearch.SearchResponse, minResults int) error {
	if resp == nil {
		return ValidationError{Field: "response", Message: "response is nil"}
	}

	if resp.Organic == nil {
		log.Warn().Msg("search response has nil organic results")
		resp.Organic = []map[string]any{}
	}

	if len(resp.Organic) == 0 {
		return ValidationError{Field: "organic", Message: "no results returned"}
	}

	if minResults > 0 && len(resp.Organic) < minResults {
		log.Warn().
			Int("expected_min", minResults).
			Int("actual", len(resp.Organic)).
			Msg("fewer results than expected, but not failing")
	}

	// Validate that results have required fields
	validResults := 0
	for idx, result := range resp.Organic {
		if result == nil {
			log.Warn().Int("index", idx).Msg("nil result in organic array")
			continue
		}

		// Check for essential fields
		hasTitle := hasNonEmptyString(result, "title")
		hasLink := hasNonEmptyString(result, "link")

		if !hasTitle || !hasLink {
			log.Warn().
				Int("index", idx).
				Bool("has_title", hasTitle).
				Bool("has_link", hasLink).
				Msg("result missing essential fields")
			continue
		}

		validResults++
	}

	if validResults == 0 {
		return ValidationError{Field: "organic", Message: "no valid results with title and link"}
	}

	return nil
}

// ValidateFetchResponse checks if a scrape response has meaningful content
func ValidateFetchResponse(resp *domainsearch.FetchWebpageResponse, minLength int) error {
	if resp == nil {
		return ValidationError{Field: "response", Message: "response is nil"}
	}

	text := strings.TrimSpace(resp.Text)
	if text == "" {
		return ValidationError{Field: "text", Message: "empty text content"}
	}

	if minLength > 0 && len(text) < minLength {
		return ValidationError{
			Field:   "text",
			Message: fmt.Sprintf("text too short: %d chars (min: %d)", len(text), minLength),
		}
	}

	return nil
}

// EnrichEmptyResponse adds helpful context when responses are empty
func EnrichEmptyResponse(resp *domainsearch.SearchResponse, query string, reason string) *domainsearch.SearchResponse {
	if resp == nil {
		resp = &domainsearch.SearchResponse{}
	}

	if resp.Organic == nil || len(resp.Organic) == 0 {
		resp.Organic = []map[string]any{
			{
				"title":   fmt.Sprintf("No results found for: %s", query),
				"link":    fmt.Sprintf("https://google.com/search?q=%s", strings.ReplaceAll(query, " ", "+")),
				"snippet": fmt.Sprintf("The search returned no results. Reason: %s. Try refining your query or checking connectivity.", reason),
				"source":  "empty_fallback",
			},
		}
	}

	if resp.SearchParameters == nil {
		resp.SearchParameters = map[string]any{}
	}
	resp.SearchParameters["empty_result_reason"] = reason
	resp.SearchParameters["has_results"] = len(resp.Organic) > 0

	return resp
}

// EnrichEmptyFetch adds helpful context when scrape response is empty
func EnrichEmptyFetch(resp *domainsearch.FetchWebpageResponse, url string, reason string) *domainsearch.FetchWebpageResponse {
	if resp == nil {
		resp = &domainsearch.FetchWebpageResponse{}
	}

	if strings.TrimSpace(resp.Text) == "" {
		resp.Text = fmt.Sprintf("Failed to fetch content from %s. Reason: %s", url, reason)
	}

	if resp.Metadata == nil {
		resp.Metadata = map[string]any{}
	}
	resp.Metadata["empty_result_reason"] = reason
	resp.Metadata["has_content"] = strings.TrimSpace(resp.Text) != ""

	return resp
}

// hasNonEmptyString checks if a map has a non-empty string value for a key
func hasNonEmptyString(m map[string]any, key string) bool {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return strings.TrimSpace(str) != ""
		}
	}
	return false
}
