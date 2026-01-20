package slide_generator

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func slideLayoutID(slide any) string {
	if slide == nil {
		return ""
	}
	slideMap, ok := slide.(map[string]interface{})
	if !ok {
		return ""
	}
	if layoutID, ok := slideMap["layoutId"].(string); ok {
		return strings.TrimSpace(layoutID)
	}
	return ""
}

func slideHasElementType(slide any, elementType string) bool {
	if slide == nil {
		return false
	}
	slideMap, ok := slide.(map[string]interface{})
	if !ok {
		return false
	}
	rawElements, ok := slideMap["elements"].([]interface{})
	if !ok {
		return false
	}
	for _, raw := range rawElements {
		el, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if t, ok := el["type"].(string); ok && t == elementType {
			return true
		}
	}
	return false
}

func ensureSlideOrderAndID(slide map[string]any, slideIndex int) {
	slide["order"] = slideIndex
	expectedID := fmt.Sprintf("slide_%d", slideIndex)
	if id, ok := slide["id"].(string); !ok || strings.TrimSpace(id) == "" || id != expectedID {
		slide["id"] = expectedID
	}
}

func ensureSlideUseComponents(slide map[string]any) {
	if _, ok := slide["useComponents"]; !ok {
		slide["useComponents"] = []any{}
	}
}

func templateLayoutIDs(layouts any) map[string]bool {
	result := map[string]bool{}
	if layoutsSlice, ok := layouts.([]any); ok {
		for _, layout := range layoutsSlice {
			if layoutMap, ok := layout.(map[string]any); ok {
				if id, ok := layoutMap["id"].(string); ok && strings.TrimSpace(id) != "" {
					result[strings.TrimSpace(id)] = true
				}
			}
		}
	}
	return result
}

func layoutIDMatchesSuggestedLayout(layoutID string, suggestedLayout string, layouts any) bool {
	if layoutID == "" || suggestedLayout == "" {
		return false
	}
	if strings.EqualFold(layoutID, suggestedLayout) {
		return true
	}
	normalized := normalizeLayoutToken(layoutID)
	if normalized == suggestedLayout {
		return true
	}

	nameMap := layoutNameByID(layouts)
	if name, ok := nameMap[layoutID]; ok {
		nameNormalized := normalizeLayoutToken(name)
		if nameNormalized == suggestedLayout {
			return true
		}
	}

	if alias, ok := layoutAliasMap()[normalized]; ok {
		if alias == suggestedLayout {
			log.Debug().
				Str("layout_id", layoutID).
				Str("suggested_layout", suggestedLayout).
				Msg("[slide_generator] layout matched via alias")
			return true
		}
		return false
	}

	return false
}

func layoutNameByID(layouts any) map[string]string {
	result := map[string]string{}
	if layoutsSlice, ok := layouts.([]any); ok {
		for _, layout := range layoutsSlice {
			if layoutMap, ok := layout.(map[string]any); ok {
				id, _ := layoutMap["id"].(string)
				name, _ := layoutMap["name"].(string)
				if id != "" && name != "" {
					result[id] = name
				}
			}
		}
	}
	return result
}

func normalizeLayoutToken(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "layout_")
	value = strings.TrimPrefix(value, "layout-")
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ToUpper(value)
	return value
}

func layoutAliasMap() map[string]string {
	return map[string]string{
		"TITLE_BULLETS":        "TITLE_AND_BULLETS",
		"TITLE_AND_BULLETS":    "TITLE_AND_BULLETS",
		"TITLE_TWO_COLUMNS":    "TITLE_TWO_COLUMNS",
		"TITLE_IMAGE":          "TITLE_IMAGE",
		"FULL_BLEED_IMAGE":     "FULL_BLEED_IMAGE",
		"SECTION_HEADER":       "SECTION_HEADER",
		"TABLE":                "TABLE",
		"CHART":                "CHART",
		"QUOTE":                "QUOTE",
		"TIMELINE":             "TIMELINE",
		"CLOSING":              "CLOSING",
		"APPENDIX":             "APPENDIX",
		"DASHBOARD_3KPI_2COL":  "DASHBOARD_3KPI_2COL",
		"CHART_AND_INSIGHTS":   "CHART_AND_INSIGHTS",
		"TABLE_AND_CALLOUTS":   "TABLE_AND_CALLOUTS",
		"TITLE":                "TITLE",
		"TITLE_SLIDE":          "TITLE",
		"TITLE_AND_BULLETS_V1": "TITLE_AND_BULLETS",
	}
}

func extractAssetIDs(assets []any) map[string]bool {
	ids := map[string]bool{}
	for _, asset := range assets {
		switch v := asset.(type) {
		case string:
			if v != "" {
				ids[v] = true
			}
		case map[string]any:
			if id, ok := v["id"].(string); ok && id != "" {
				ids[id] = true
			}
		}
	}
	return ids
}

func extractDatasetIDs(datasets []any) map[string]bool {
	ids := map[string]bool{}
	for _, dataset := range datasets {
		if v, ok := dataset.(map[string]any); ok {
			if id, ok := v["id"].(string); ok && id != "" {
				ids[id] = true
			}
		}
	}
	return ids
}

func validateChartDatasetRefs(slide map[string]any, datasetIDs map[string]bool) string {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if elemType, _ := elem["type"].(string); elemType != "chart" {
			continue
		}
		chart, _ := elem["chart"].(map[string]any)
		if chart == nil {
			continue
		}
		if ref, ok := chart["datasetRef"].(string); ok && ref != "" {
			if !datasetIDs[ref] {
				return ref
			}
		}
	}
	return ""
}

func validateImageAssetRefs(slide map[string]any, assetIDs map[string]bool) string {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if elemType, _ := elem["type"].(string); elemType != "image" {
			continue
		}
		image, _ := elem["image"].(map[string]any)
		if image == nil {
			continue
		}
		if ref, ok := image["ref"].(string); ok && ref != "" {
			if !assetIDs[ref] {
				return ref
			}
		}
	}
	return ""
}

