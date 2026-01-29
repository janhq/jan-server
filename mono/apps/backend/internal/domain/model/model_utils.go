package model

import (
	"strings"

	"github.com/shopspring/decimal"
)

// we can reuse these utility functions in both model_catalog and provider_model
func extractDefaultParameters(value any) map[string]*decimal.Decimal {
	result := map[string]*decimal.Decimal{}
	params, ok := value.(map[string]any)
	if !ok {
		return result
	}
	for key, raw := range params {
		if raw == nil {
			result[key] = nil
			continue
		}
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				result[key] = nil
				continue
			}
			if d, err := decimal.NewFromString(v); err == nil {
				val := d
				result[key] = &val
			}
		case float64:
			d := decimal.NewFromFloat(v)
			result[key] = &d
		case float32:
			d := decimal.NewFromFloat32(v)
			result[key] = &d
		default:
			// ignore unsupported types
		}
	}
	return result
}

func extractStringSlice(value any) []string {
	list := []string{}
	switch arr := value.(type) {
	case []any:
		for _, item := range arr {
			if str, ok := item.(string); ok {
				list = append(list, strings.TrimSpace(str))
			}
		}
	case []string:
		for _, item := range arr {
			list = append(list, strings.TrimSpace(item))
		}
	}
	return list
}

func extractStringSliceFromMap(raw map[string]any, path ...string) []string {
	current := any(raw)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return extractStringSlice(current)
}

func getString(raw map[string]any, key string) (string, bool) {
	if raw == nil {
		return "", false
	}
	if value, ok := raw[key]; ok {
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str), true
		}
	}
	return "", false
}

func copyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	dest := make(map[string]any, len(source))
	for k, v := range source {
		dest[k] = v
	}
	return dest
}

func floatFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, false
		}
		if parsed, err := decimal.NewFromString(v); err == nil {
			return parsed.InexactFloat64(), true
		}
	}
	return 0, false
}

func containsString(list []string, target string) bool {
	target = strings.ToLower(target)
	for _, item := range list {
		if strings.ToLower(item) == target {
			return true
		}
	}
	return false
}

func normalizeURL(baseURL string) string {
	normalized := strings.TrimSpace(baseURL)
	normalized = strings.TrimRight(normalized, "/")
	return normalized
}

// catalogPublicID builds a public ID for a model catalog
func catalogPublicID(kind ProviderKind, modelID, canonicalSlug string) string {
	rawModelKey := canonicalSlug
	if rawModelKey == "" {
		rawModelKey = modelID
	}
	return NormalizeModelKey(kind, rawModelKey)
}

// detectEmbeddingSupport checks multiple sources to determine if model supports embeddings
func detectEmbeddingSupport(modelID string, raw map[string]any) bool {
	// Check model ID
	lowerID := strings.ToLower(modelID)
	if strings.Contains(lowerID, "embed") {
		return true
	}

	// Check architecture output modalities
	outputModalities := extractStringSliceFromMap(raw, "architecture", "output_modalities")
	if containsString(outputModalities, "embedding") {
		return true
	}

	// Check model type/category in raw metadata
	if modelType, ok := getString(raw, "type"); ok {
		if strings.Contains(strings.ToLower(modelType), "embed") {
			return true
		}
	}

	// Check category field
	if category, ok := getString(raw, "category"); ok {
		if strings.Contains(strings.ToLower(category), "embed") {
			return true
		}
	}

	return false
}

// extractFamily attempts to identify model family from model ID
func extractFamily(modelID string) string {
	lowerID := strings.ToLower(modelID)

	// Known model families with patterns
	families := map[string][]string{
		"gpt-4":      {"gpt-4", "gpt4"},
		"gpt-3.5":    {"gpt-3.5", "gpt35"},
		"claude-3":   {"claude-3"},
		"claude-3.5": {"claude-3.5"},
		"llama-3":    {"llama-3", "llama3"},
		"llama-2":    {"llama-2", "llama2"},
		"gemini":     {"gemini"},
		"mistral":    {"mistral"},
		"mixtral":    {"mixtral"},
		"phi":        {"phi-"},
		"qwen":       {"qwen"},
	}

	for family, patterns := range families {
		for _, pattern := range patterns {
			if strings.Contains(lowerID, pattern) {
				return family
			}
		}
	}

	// Fallback to delimiter-based extraction
	if idx := strings.Index(modelID, "/"); idx > 0 {
		return strings.TrimSpace(modelID[:idx])
	}

	return ""
}
