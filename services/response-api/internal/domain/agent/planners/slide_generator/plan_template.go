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
	left := 36.0
	top := 36.0
	right := 36.0
	bottom := 36.0
	usableW := 960.0 - left - right
	usableH := 540.0 - top - bottom

	titleRect := map[string]any{"x": left, "y": 72.0, "w": usableW, "h": 48.0}
	bodyRect := map[string]any{"x": left, "y": 140.0, "w": usableW, "h": 320.0}
	headerRect := map[string]any{"x": left, "y": top, "w": 700.0, "h": 20.0}
	footerRect := map[string]any{"x": left, "y": 540.0 - bottom - 18.0, "w": usableW, "h": 18.0}

	slots := []any{}
	switch layoutID {
	case "TITLE":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 200.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "subtitle", "rect": map[string]any{"x": 120.0, "y": 290.0, "w": 720.0, "h": 60.0}},
		}
	case "SECTION_HEADER":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 220.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "subtitle", "rect": map[string]any{"x": 120.0, "y": 300.0, "w": 720.0, "h": 50.0}},
		}
	case "TITLE_TWO_COLUMNS":
		colW := (usableW - 24.0) / 2
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "left", "rect": map[string]any{"x": left, "y": 140.0, "w": colW, "h": 320.0}},
			map[string]any{"id": "right", "rect": map[string]any{"x": left + colW + 24.0, "y": 140.0, "w": colW, "h": 320.0}},
		}
	case "TITLE_IMAGE":
		imageW := 320.0
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": map[string]any{"x": left, "y": 140.0, "w": usableW - imageW - 24.0, "h": 320.0}},
			map[string]any{"id": "image", "rect": map[string]any{"x": left + usableW - imageW, "y": 140.0, "w": imageW, "h": 320.0}},
		}
	case "FULL_BLEED_IMAGE":
		slots = []any{
			map[string]any{"id": "image", "rect": map[string]any{"x": 0.0, "y": 0.0, "w": 960.0, "h": 540.0}},
			map[string]any{"id": "title", "rect": map[string]any{"x": left, "y": 72.0, "w": usableW, "h": 60.0}},
		}
	case "CHART":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "chart", "rect": bodyRect},
		}
	case "TABLE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "table", "rect": bodyRect},
		}
	case "QUOTE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "quote", "rect": bodyRect},
		}
	case "TIMELINE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "timeline", "rect": bodyRect},
		}
	case "CLOSING":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 200.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "body", "rect": map[string]any{"x": 120.0, "y": 290.0, "w": 720.0, "h": 120.0}},
		}
	case "APPENDIX":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": bodyRect},
		}
	case "DASHBOARD_3KPI_2COL":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "kpi1", "rect": map[string]any{"x": left, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "kpi2", "rect": map[string]any{"x": left + 300.0, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "kpi3", "rect": map[string]any{"x": left + 600.0, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "chart_left", "rect": map[string]any{"x": left, "y": 224.0, "w": 548.0, "h": 260.0}},
			map[string]any{"id": "table_right", "rect": map[string]any{"x": left + 564.0, "y": 224.0, "w": 324.0, "h": 260.0}},
		}
	case "CHART_AND_INSIGHTS":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "chart_left", "rect": map[string]any{"x": left, "y": 140.0, "w": 560.0, "h": 320.0}},
			map[string]any{"id": "insights_right", "rect": map[string]any{"x": left + 580.0, "y": 140.0, "w": 308.0, "h": 320.0}},
		}
	case "TABLE_AND_CALLOUTS":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "table_left", "rect": map[string]any{"x": left, "y": 140.0, "w": 560.0, "h": 320.0}},
			map[string]any{"id": "callouts_right", "rect": map[string]any{"x": left + 580.0, "y": 140.0, "w": 308.0, "h": 320.0}},
		}
	default:
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": bodyRect},
		}
	}

	if len(slots) == 0 {
		slots = []any{map[string]any{"id": "body", "rect": map[string]any{"x": left, "y": top, "w": usableW, "h": usableH}}}
	}

	return map[string]any{
		"id":       layoutID,
		"name":     layoutID,
		"masterId": "master_default",
		"slots":    slots,
	}
}

