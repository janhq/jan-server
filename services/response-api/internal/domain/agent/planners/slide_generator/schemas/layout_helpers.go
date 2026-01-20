package schemas

import (
	"strings"
)

// InjectLayoutIDsIntoSchema modifies a schema to include available layoutIds as enum.
// This ensures the LLM can only choose from valid layouts.
func InjectLayoutIDsIntoSchema(schema map[string]any, layoutIDs []string) map[string]any {
	if schema == nil || len(layoutIDs) == 0 {
		return schema
	}

	// Deep copy to avoid mutating the original
	schemaCopy := deepCopySchema(schema)

	// Find and update layoutId property
	if props, ok := schemaCopy["properties"].(map[string]any); ok {
		// Case 1: Direct layoutId property (SlideMetadataSchema)
		if layoutIDProp, ok := props["layoutId"].(map[string]any); ok {
			layoutIDProp["enum"] = layoutIDs
		}

		// Case 2: Nested in slide property (SlideGenResultSchema)
		if slideProp, ok := props["slide"].(map[string]any); ok {
			if slideProps, ok := slideProp["properties"].(map[string]any); ok {
				if layoutIDProp, ok := slideProps["layoutId"].(map[string]any); ok {
					layoutIDProp["enum"] = layoutIDs
				}
			}
		}

		// Case 3: Nested in element property (for future element-level schemas)
		if elemProp, ok := props["element"].(map[string]any); ok {
			if elemProps, ok := elemProp["properties"].(map[string]any); ok {
				if layoutIDProp, ok := elemProps["layoutId"].(map[string]any); ok {
					layoutIDProp["enum"] = layoutIDs
				}
			}
		}
	}

	return schemaCopy
}

// FormatAvailableLayoutsForPrompt formats layout IDs into a prompt-friendly string.
func FormatAvailableLayoutsForPrompt(layoutIDs []string) string {
	if len(layoutIDs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## AVAILABLE LAYOUTS (MUST USE ONLY THESE):\n")
	for _, id := range layoutIDs {
		sb.WriteString("- ")
		sb.WriteString(id)
		sb.WriteString("\n")
	}
	sb.WriteString("\nYou MUST use exactly one of the above layoutIds. Do not invent or use any other layout names.\n")
	return sb.String()
}

// FindClosestLayoutID finds the closest matching layout from available layouts.
// Used as fallback when LLM generates an invalid layoutId.
func FindClosestLayoutID(requestedLayout string, availableLayouts []string) string {
	if len(availableLayouts) == 0 {
		return ""
	}

	requested := normalizeLayoutName(requestedLayout)

	// Exact match first
	for _, layout := range availableLayouts {
		if strings.EqualFold(layout, requestedLayout) {
			return layout
		}
	}

	// Normalized match
	for _, layout := range availableLayouts {
		if normalizeLayoutName(layout) == requested {
			return layout
		}
	}

	// Partial match (contains)
	for _, layout := range availableLayouts {
		normalized := normalizeLayoutName(layout)
		if strings.Contains(normalized, requested) || strings.Contains(requested, normalized) {
			return layout
		}
	}

	// Semantic match based on keywords
	keywordMap := map[string][]string{
		"title":          {"TITLE", "SECTION_HEADER"},
		"bullets":        {"TITLE_AND_BULLETS", "TITLE_TWO_COLUMNS"},
		"chart":          {"CHART", "CHART_AND_INSIGHTS", "DASHBOARD_3KPI_2COL"},
		"image":          {"TITLE_IMAGE", "FULL_BLEED_IMAGE"},
		"table":          {"TABLE", "TABLE_AND_CALLOUTS"},
		"quote":          {"QUOTE"},
		"timeline":       {"TIMELINE"},
		"closing":        {"CLOSING"},
		"two_columns":    {"TITLE_TWO_COLUMNS", "CHART_AND_INSIGHTS"},
		"dashboard":      {"DASHBOARD_3KPI_2COL"},
		"insights":       {"CHART_AND_INSIGHTS", "TABLE_AND_CALLOUTS"},
		"section_header": {"SECTION_HEADER"},
		"appendix":       {"APPENDIX"},
	}

	// Check if requested contains any keyword
	for keyword, candidates := range keywordMap {
		if strings.Contains(requested, keyword) {
			// Find first matching candidate in available layouts
			for _, candidate := range candidates {
				for _, layout := range availableLayouts {
					if strings.EqualFold(layout, candidate) {
						return layout
					}
				}
			}
		}
	}

	// Default fallback - return first available layout
	return availableLayouts[0]
}

// normalizeLayoutName normalizes layout name for comparison.
func normalizeLayoutName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, " ", "")
	return name
}

// deepCopySchema creates a deep copy of a schema map.
func deepCopySchema(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[k] = deepCopySchema(val)
		case []any:
			dst[k] = deepCopySlice(val)
		default:
			dst[k] = v
		}
	}
	return dst
}

// deepCopySlice creates a deep copy of a slice.
func deepCopySlice(src []any) []any {
	if src == nil {
		return nil
	}
	dst := make([]any, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[i] = deepCopySchema(val)
		case []any:
			dst[i] = deepCopySlice(val)
		default:
			dst[i] = v
		}
	}
	return dst
}

// ExtractLayoutIDsFromTemplate extracts layout IDs from a template structure.
func ExtractLayoutIDsFromTemplate(template any) []string {
	var layoutIDs []string

	// Handle map with "layouts" key
	if templateMap, ok := template.(map[string]any); ok {
		if layouts, ok := templateMap["layouts"]; ok {
			layoutIDs = extractLayoutIDsFromSlice(layouts)
		}
	} else if layoutsSlice, ok := template.([]any); ok {
		// Handle direct slice of layouts
		layoutIDs = extractLayoutIDsFromSlice(layoutsSlice)
	}

	return layoutIDs
}

// extractLayoutIDsFromSlice extracts IDs from a slice of layout objects.
func extractLayoutIDsFromSlice(layouts any) []string {
	var ids []string
	if layoutsSlice, ok := layouts.([]any); ok {
		for _, layout := range layoutsSlice {
			if layoutMap, ok := layout.(map[string]any); ok {
				if id, ok := layoutMap["id"].(string); ok && strings.TrimSpace(id) != "" {
					ids = append(ids, strings.TrimSpace(id))
				}
			}
		}
	}
	return ids
}
