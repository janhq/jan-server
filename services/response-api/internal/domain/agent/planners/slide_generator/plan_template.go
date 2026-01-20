package slide_generator

import (
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) extractPlanAndTemplate(input agent.ExecutionInput) *schemas.PlanAndTemplate {
	log.Debug().Msg("[slide_generator] extractPlanAndTemplate started")
	candidates := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	if len(input.PreviousOutput) > 0 {
		candidates = append(candidates, input.PreviousOutput)
	}
	candidates = append(candidates, input.AccumulatedOutputs...)
	log.Debug().Int("candidates_count", len(candidates)).Msg("[slide_generator] searching for plan and template")

	for i := len(candidates) - 1; i >= 0; i-- {
		var payload map[string]interface{}
		if err := json.Unmarshal(candidates[i], &payload); err != nil {
			continue
		}
		if payloadType, _ := payload["type"].(string); payloadType == "plan_and_template" {
			planBytes, _ := json.Marshal(payload)
			var planAndTemplate schemas.PlanAndTemplate
			if err := json.Unmarshal(planBytes, &planAndTemplate); err == nil {
				log.Debug().
					Int("slide_count", len(planAndTemplate.Plan.Slides)).
					Str("deck_title", planAndTemplate.Plan.DeckTitle).
					Msg("[slide_generator] found plan and template from type=plan_and_template")
				return &planAndTemplate
			}
		}
		if planRaw, ok := payload["plan"]; ok {
			if templateRaw, ok := payload["template"]; ok {
				merged := map[string]interface{}{"plan": planRaw, "template": templateRaw}
				planBytes, _ := json.Marshal(merged)
				var planAndTemplate schemas.PlanAndTemplate
				if err := json.Unmarshal(planBytes, &planAndTemplate); err == nil {
					log.Debug().
						Int("slide_count", len(planAndTemplate.Plan.Slides)).
						Str("deck_title", planAndTemplate.Plan.DeckTitle).
						Msg("[slide_generator] found plan and template from separate keys")
					return &planAndTemplate
				}
			}
		}
	}
	log.Warn().Msg("[slide_generator] plan and template not found in any output")
	return nil
}

func normalizePlanIndices(plan *schemas.SlidePlan) {
	if plan == nil {
		return
	}
	log.Debug().Int("slides", len(plan.Slides)).Msg("[slide_generator] normalizing plan indices")
	for i := range plan.Slides {
		plan.Slides[i].Index = i + 1
	}
	plan.RecommendedSlideCount = len(plan.Slides)
}

func normalizeTemplateComponents(template *schemas.SlideTemplate) {
	if template == nil {
		return
	}
	components, ok := template.Components.([]any)
	if !ok {
		return
	}
	normalized := make([]any, 0, len(components))
	for _, compAny := range components {
		comp, ok := compAny.(map[string]any)
		if !ok {
			continue
		}
		if _, hasElements := comp["elements"]; hasElements {
			normalized = append(normalized, comp)
			continue
		}
		compID, _ := comp["id"].(string)
		compType, _ := comp["type"].(string)
		rect, _ := comp["rect"].(map[string]any)
		style, _ := comp["style"].(map[string]any)

		if compType == "" {
			compType = "text"
		}

		element := map[string]any{
			"id":   compID,
			"type": compType,
			"rect": rect,
		}

		switch compType {
		case "text":
			content := ""
			if compID == "header" {
				content = "{{title}}"
			} else if compID == "footer" {
				content = "{page}/{total_pages}"
			}
			element["text"] = map[string]any{
				"content": content,
				"runs":    []any{},
				"autoFit": "shrink",
				"style":   style,
			}
		case "shape":
			element["shape"] = map[string]any{
				"kind":  "rect",
				"style": map[string]any{"fill": map[string]any{"type": "solid", "color": "#FFFFFF"}},
			}
		case "image":
			if image, ok := comp["image"].(map[string]any); ok {
				element["image"] = image
			}
		}

		normalized = append(normalized, map[string]any{
			"id":       compID,
			"elements": []any{element},
		})
	}
	template.Components = normalized
	log.Debug().Int("components", len(normalized)).Msg("[slide_generator] normalized template components")
}

func normalizeTemplateLayouts(plan *schemas.SlidePlan, template *schemas.SlideTemplate) {
	if template == nil {
		return
	}
	layoutsSlice, ok := template.Layouts.([]any)
	if !ok {
		layoutsSlice = []any{}
	}
	log.Debug().
		Int("plan_slides", len(plan.Slides)).
		Int("layouts_before", len(layoutsSlice)).
		Msg("[slide_generator] normalizing template layouts")

	allowed := layoutEnumSet()
	usedIDs := map[string]bool{}
	normalizedLayouts := make([]any, 0, len(layoutsSlice))

	for _, layoutAny := range layoutsSlice {
		layout, ok := layoutAny.(map[string]any)
		if !ok {
			continue
		}
		rawID, _ := layout["id"].(string)
		rawName, _ := layout["name"].(string)
		candidateID := normalizeLayoutToken(rawID)
		if candidateID == "" {
			candidateID = normalizeLayoutToken(rawName)
		}
		if allowed[candidateID] && !usedIDs[candidateID] {
			layout["id"] = candidateID
			usedIDs[candidateID] = true
			normalizedLayouts = append(normalizedLayouts, layout)
			continue
		}
		if rawID != "" && !usedIDs[rawID] {
			usedIDs[rawID] = true
		} else if rawID == "" {
			rawID = fmt.Sprintf("CUSTOM_%d", len(usedIDs)+1)
			layout["id"] = rawID
			usedIDs[rawID] = true
		}
		normalizedLayouts = append(normalizedLayouts, layout)
	}

	for _, entry := range plan.Slides {
		layoutID := strings.TrimSpace(entry.SuggestedLayout)
		if layoutID == "" {
			continue
		}
		if usedIDs[layoutID] {
			continue
		}
		normalizedLayouts = append(normalizedLayouts, defaultLayoutForType(layoutID))
		usedIDs[layoutID] = true
	}

	template.Layouts = normalizedLayouts
	log.Debug().Int("layouts_after", len(normalizedLayouts)).Msg("[slide_generator] normalized template layouts")
}

func layoutEnumSet() map[string]bool {
	return map[string]bool{
		"TITLE":               true,
		"SECTION_HEADER":      true,
		"TITLE_AND_BULLETS":   true,
		"TITLE_TWO_COLUMNS":   true,
		"TITLE_IMAGE":         true,
		"FULL_BLEED_IMAGE":    true,
		"CHART":               true,
		"TABLE":               true,
		"QUOTE":               true,
		"TIMELINE":            true,
		"CLOSING":             true,
		"APPENDIX":            true,
		"DASHBOARD_3KPI_2COL": true,
		"CHART_AND_INSIGHTS":  true,
		"TABLE_AND_CALLOUTS":  true,
	}
}

func defaultLayoutForType(layoutID string) map[string]any {
	// P2 fix: Use grid specs instead of absolute rect values for consistent layout
	// Grid-based positioning allows the theme to control column widths and gutters
	// Format: {"id": "slotName", "grid": {"col": 1, "span": 12}, "y": 140, "h": 320}

	top := 36.0
	bottom := 36.0

	// Header and footer use absolute positioning (outside main content area)
	headerSlot := map[string]any{"id": "header", "grid": map[string]any{"col": 1, "span": 9}, "y": top, "h": 20.0}
	footerSlot := map[string]any{"id": "footer", "grid": map[string]any{"col": 1, "span": 12}, "y": 540.0 - bottom - 18.0, "h": 18.0}
	titleSlot := map[string]any{"id": "title", "grid": map[string]any{"col": 1, "span": 12}, "y": 72.0, "h": 48.0}
	bodySlot := map[string]any{"id": "body", "grid": map[string]any{"col": 1, "span": 12}, "y": 140.0, "h": 320.0}

	slots := []any{}
	switch layoutID {
	case "TITLE":
		slots = []any{
			map[string]any{"id": "title", "grid": map[string]any{"col": 2, "span": 10}, "y": 200.0, "h": 80.0},
			map[string]any{"id": "subtitle", "grid": map[string]any{"col": 2, "span": 10}, "y": 290.0, "h": 60.0},
		}
	case "SECTION_HEADER":
		slots = []any{
			map[string]any{"id": "title", "grid": map[string]any{"col": 2, "span": 10}, "y": 220.0, "h": 80.0},
			map[string]any{"id": "subtitle", "grid": map[string]any{"col": 2, "span": 10}, "y": 300.0, "h": 50.0},
		}
	case "TITLE_AND_BULLETS":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			bodySlot,
		}
	case "TITLE_TWO_COLUMNS":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "left", "grid": map[string]any{"col": 1, "span": 6}, "y": 140.0, "h": 320.0},
			map[string]any{"id": "right", "grid": map[string]any{"col": 7, "span": 6}, "y": 140.0, "h": 320.0},
		}
	case "TITLE_IMAGE":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "body", "grid": map[string]any{"col": 1, "span": 8}, "y": 140.0, "h": 320.0},
			map[string]any{"id": "image", "grid": map[string]any{"col": 9, "span": 4}, "y": 140.0, "h": 320.0},
		}
	case "FULL_BLEED_IMAGE":
		// Full bleed uses absolute rect for full canvas coverage
		slots = []any{
			map[string]any{"id": "image", "rect": map[string]any{"x": 0.0, "y": 0.0, "w": 960.0, "h": 540.0}},
			map[string]any{"id": "title", "grid": map[string]any{"col": 1, "span": 12}, "y": 72.0, "h": 60.0},
		}
	case "CHART":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "chart", "grid": map[string]any{"col": 1, "span": 12}, "y": 140.0, "h": 320.0},
		}
	case "TABLE":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "table", "grid": map[string]any{"col": 1, "span": 12}, "y": 140.0, "h": 320.0},
		}
	case "QUOTE":
		slots = []any{
			headerSlot,
			footerSlot,
			map[string]any{"id": "quote", "grid": map[string]any{"col": 2, "span": 10}, "y": 140.0, "h": 320.0},
		}
	case "TIMELINE":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "timeline", "grid": map[string]any{"col": 1, "span": 12}, "y": 140.0, "h": 320.0},
		}
	case "CLOSING":
		slots = []any{
			map[string]any{"id": "title", "grid": map[string]any{"col": 2, "span": 10}, "y": 200.0, "h": 80.0},
			map[string]any{"id": "body", "grid": map[string]any{"col": 2, "span": 10}, "y": 290.0, "h": 120.0},
		}
	case "APPENDIX":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			bodySlot,
		}
	case "DASHBOARD_3KPI_2COL":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "kpi1", "grid": map[string]any{"col": 1, "span": 4}, "y": 128.0, "h": 80.0},
			map[string]any{"id": "kpi2", "grid": map[string]any{"col": 5, "span": 4}, "y": 128.0, "h": 80.0},
			map[string]any{"id": "kpi3", "grid": map[string]any{"col": 9, "span": 4}, "y": 128.0, "h": 80.0},
			map[string]any{"id": "chart_left", "grid": map[string]any{"col": 1, "span": 7}, "y": 224.0, "h": 260.0},
			map[string]any{"id": "table_right", "grid": map[string]any{"col": 8, "span": 5}, "y": 224.0, "h": 260.0},
		}
	case "CHART_AND_INSIGHTS":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "chart_left", "grid": map[string]any{"col": 1, "span": 7}, "y": 140.0, "h": 320.0},
			map[string]any{"id": "insights_right", "grid": map[string]any{"col": 8, "span": 5}, "y": 140.0, "h": 320.0},
		}
	case "TABLE_AND_CALLOUTS":
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			map[string]any{"id": "table_left", "grid": map[string]any{"col": 1, "span": 7}, "y": 140.0, "h": 320.0},
			map[string]any{"id": "callouts_right", "grid": map[string]any{"col": 8, "span": 5}, "y": 140.0, "h": 320.0},
		}
	default:
		slots = []any{
			headerSlot,
			footerSlot,
			titleSlot,
			bodySlot,
		}
	}

	if len(slots) == 0 {
		slots = []any{map[string]any{"id": "body", "grid": map[string]any{"col": 1, "span": 12}, "y": top, "h": 468.0}}
	}

	return map[string]any{
		"id":       layoutID,
		"name":     layoutID,
		"masterId": "master_default",
		"slots":    slots,
	}
}
