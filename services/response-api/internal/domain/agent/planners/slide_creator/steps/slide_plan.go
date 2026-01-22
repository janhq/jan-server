package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func (e *SlideCreatorExecutor) executeSlidePlan(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	config, _ := params["config"].(map[string]interface{})
	brief := strings.TrimSpace(stringValue(params, "brief"))
	numSlides, ok := parseIntFromInterface(config["num_slides"])
	if !ok || numSlides <= 0 {
		numSlides = 10
	}
	themePref := strings.TrimSpace(stringValue(config, "theme"))

	contextData := buildAccumulatedContext(input)
	assets := collectImageAssets(input)
	assetsJSON, _ := json.Marshal(assets)
	dataBank := collectDataBankText(input)

	briefParts := []string{}
	if brief != "" {
		briefParts = append(briefParts, "Brief:\n"+brief)
	}
	if themePref != "" {
		briefParts = append(briefParts, "Preferred theme:\n"+themePref)
	}
	if contextData != "" {
		briefParts = append(briefParts, "Research context:\n"+contextData)
	}
	if dataBank != "" {
		briefParts = append(briefParts, "Data bank:\n"+dataBank)
	}
	if len(assets) > 0 {
		briefParts = append(briefParts, "Image assets:\n"+string(assetsJSON))
	}

	planPrompt := fmt.Sprintf(`You are a slide planning assistant.

Return ONLY JSON with this shape:
{
  "title": "...",
  "theme": {
    "primary_color": "#RRGGBB",
    "accent_color": "#RRGGBB",
    "background_color": "#RRGGBB",
    "text_color": "#RRGGBB",
    "font_family": "Segoe UI, Arial, Helvetica, sans-serif"
  },
  "slides": [
    {
      "id": 1,
      "layout": "split|bullets|hero|title|table|chart",
      "title": "...",
      "subtitle": "...",
      "bullets": ["..."],
      "images": [{"src":"https://...","alt":"...","caption":"..."}],
      "table": {"title": "...", "columns": ["..."], "rows": [["..."]], "notes": "optional"},
      "chart": {"type": "bar", "title": "...", "categories": ["..."], "series": [{"name": "...", "values": [1,2]}], "notes": "optional"},
      "notes": "optional"
    }
  ]
}

HARD RULES (must follow):
- Use exactly %d slides.
- Fit-safety (prevent overlap):
  - title: 6-60 characters (aim 1 line, max 2 lines).
  - subtitle: 0-110 characters (max 2 lines).
  - bullets: 3-6 items, each 25-75 characters, no wrapping paragraphs.
  - table: 3-6 columns, 3-9 rows, keep cells <= 24 chars.
  - chart: 3-6 categories, 1-3 series, use bar|line|pie.
  - Avoid colons + long clauses; prefer short phrases.
- Theme safety (prevent low contrast):
  - Choose a background that is either VERY dark or VERY light (avoid mid-tone grays/blues).
    - Good dark examples: #0B1220, #0F172A, #111827
    - Good light examples: #FFFFFF, #F8FAFC, #F1F5F9
  - Choose text_color for high contrast against background:
    - If background is dark: text_color must be near-white (e.g. #F8FAFC or #FFFFFF).
    - If background is light: text_color must be near-black (e.g. #0F172A or #111827).
  - Target contrast: body text >= 4.5:1 (WCAG AA), large titles >= 3:1.
  - Do NOT use gray-on-gray or low-saturation combinations for text vs background.
- Keep theme consistent across all slides.
- Use image URLs only (no base64).
- When picking from provided assets, prefer imageUrl and only use thumbnailUrl if imageUrl is missing.
- Omit empty optional fields.
- Output JSON only (no markdown, no commentary).

%s
`, numSlides, strings.Join(briefParts, "\n\n"))

	model := getModelFromContext(input)
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	var content string
	var err error
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		content, err = e.generateWithStructuredOutput(ctx, "", planPrompt, model, DeckPlanSchema, 0.2)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		log.Warn().Err(err).Int("attempt", attempt).Msg("[slide_creator] plan structured output failed")
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "LLM_CALL_FAILED",
					Message:  fmt.Sprintf("LLM generation failed: %v", err),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	}

	if lastErr != nil {
		log.Warn().Err(lastErr).Msg("[slide_creator] plan structured output failed after retries, falling back to free-form")
		content, err = e.generateWithSystemPrompt(ctx, "", planPrompt, model, 0.2)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "LLM_CALL_FAILED",
					Message:  fmt.Sprintf("LLM generation failed: %v", err),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
		content = extractJSONFromResponse(content)
	}

	plan, err := parseDeckPlan(strings.TrimSpace(content))
	if err != nil {
		fixPrompt := fmt.Sprintf(`Fix the following JSON to be valid AND to conform exactly to the required shape and HARD RULES.

HARD RULES:
- Output ONLY a single JSON object.
- Ensure exactly %d slides.
- Enforce Fit-safety limits: title 6-60 chars; subtitle <= 110 chars; bullets 3-6 and each 25-75 chars; table 3-6 columns and 3-9 rows.
- Enforce Theme safety: background very dark OR very light; text_color must be near-white on dark, near-black on light; target contrast >= 4.5:1.
- Do NOT add markdown or explanations.

BAD_JSON:
%s
`, numSlides, content)

		fixed, fixErr := e.generateWithSystemPrompt(ctx, "", fixPrompt, model, 0.0)
		if fixErr != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "PARSE_ERROR",
					Message:  fmt.Sprintf("invalid JSON from plan (and repair failed): %v", fixErr),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
		fixed = extractJSONFromResponse(fixed)
		plan, err = parseDeckPlan(strings.TrimSpace(fixed))
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "PARSE_ERROR",
					Message:  fmt.Sprintf("could not parse plan JSON: %v", err),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	}

	contentBytes, _ := json.Marshal(plan)
	output := map[string]interface{}{
		"type":    "slide_plan",
		"plan":    plan,
		"content": string(contentBytes),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func extractJSONFromResponse(response string) string {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}
	return strings.TrimSpace(response)
}
