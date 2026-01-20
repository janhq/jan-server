package steps

import "strings"

// extractLayoutSlotIDs returns the slot ids for the layout with the given id.
// The template layouts are expected to be a []any of maps with a "id" string
// and a "slots" array of objects with "id".
func extractLayoutSlotIDs(layouts any, layoutID string) []string {
	layoutID = strings.TrimSpace(layoutID)
	if layoutID == "" {
		return nil
	}
	layoutsSlice, ok := layouts.([]any)
	if !ok {
		return nil
	}
	for _, layoutAny := range layoutsSlice {
		layout, ok := layoutAny.(map[string]any)
		if !ok {
			continue
		}
		id, _ := layout["id"].(string)
		if strings.TrimSpace(id) != layoutID {
			continue
		}
		slots, ok := layout["slots"].([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(slots))
		for _, slotAny := range slots {
			slot, ok := slotAny.(map[string]any)
			if !ok {
				continue
			}
			sid, _ := slot["id"].(string)
			sid = strings.TrimSpace(sid)
			if sid != "" {
				out = append(out, sid)
			}
		}
		return out
	}
	return nil
}

// extractComponentIDs extracts component ids from template.components.
func extractComponentIDs(components any) []string {
	componentsSlice, ok := components.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(componentsSlice))
	for _, compAny := range componentsSlice {
		comp, ok := compAny.(map[string]any)
		if !ok {
			continue
		}
		id, _ := comp["id"].(string)
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

// buildThemeDigest returns a reduced theme object suitable for prompts.
// It intentionally strips large/less useful parts (e.g., detailed grid specs)
// to save tokens.
func buildThemeDigest(theme any) any {
	root, ok := theme.(map[string]any)
	if !ok || root == nil {
		return theme
	}

	digest := map[string]any{}
	// Keep the most useful fields for writing styles.
	for _, key := range []string{"colors", "typography", "defaults", "canvas"} {
		if v, ok := root[key]; ok {
			digest[key] = v
		}
	}
	// If digest ended up empty, fall back to original theme.
	if len(digest) == 0 {
		return theme
	}
	return digest
}
