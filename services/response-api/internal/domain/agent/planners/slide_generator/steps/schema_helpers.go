package steps

import (
	"strings"

	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
)

// prepareSchema clones + normalizes a schema for structured output, then minifies
// it to reduce token overhead.
//
// NOTE: Normalization happens first to satisfy strict structured-output
// requirements (e.g., all properties required + additionalProperties=false).
func prepareSchema(base map[string]any) map[string]any {
	schema := cloneSchema(base)
	schemas.NormalizeSchemaForStructuredOutput(schema)
	schemas.MinifySchemaForLLM(schema)
	return schema
}

// prepareSlideSchema creates a smaller, per-slide schema by applying runtime
// constraints (layoutId enum + slotId enum) before normalization + minification.
func prepareSlideSchema(base map[string]any, layoutID string, slotIDs []string) map[string]any {
	schema := cloneSchema(base)
	layoutID = strings.TrimSpace(layoutID)
	if layoutID != "" {
		setSlideLayoutIDEnum(schema, layoutID)
	}
	if len(slotIDs) > 0 {
		setElementSlotIDEnum(schema, slotIDs)
	}
	schemas.NormalizeSchemaForStructuredOutput(schema)
	schemas.MinifySchemaForLLM(schema)
	return schema
}

func setSlideLayoutIDEnum(schema map[string]any, layoutID string) {
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return
	}
	slide, _ := props["slide"].(map[string]any)
	if slide == nil {
		return
	}
	slideProps, _ := slide["properties"].(map[string]any)
	if slideProps == nil {
		return
	}
	layoutSchema, _ := slideProps["layoutId"].(map[string]any)
	if layoutSchema == nil {
		return
	}
	layoutSchema["enum"] = []string{layoutID}
}

func setElementSlotIDEnum(schema map[string]any, slotIDs []string) {
	slotIDs = uniqueStrings(slotIDs)
	if len(slotIDs) == 0 {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return
	}
	slide, _ := props["slide"].(map[string]any)
	if slide == nil {
		return
	}
	slideProps, _ := slide["properties"].(map[string]any)
	if slideProps == nil {
		return
	}
	elements, _ := slideProps["elements"].(map[string]any)
	if elements == nil {
		return
	}
	items, _ := elements["items"].(map[string]any)
	if items == nil {
		return
	}
	itemProps, _ := items["properties"].(map[string]any)
	if itemProps == nil {
		return
	}
	slotSchema, _ := itemProps["slotId"].(map[string]any)
	if slotSchema == nil {
		return
	}
	slotSchema["enum"] = slotIDs
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
