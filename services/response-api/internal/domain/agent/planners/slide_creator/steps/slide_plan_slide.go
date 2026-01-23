package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

const slidePlanSlideAttemptTimeout = 90 * time.Second

func (e *SlideCreatorExecutor) executeSlidePlanSlide(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	slideIndex, _ := parseIntFromInterface(params["slide_index"])
	if slideIndex <= 0 {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_SLIDE_INDEX",
				Message:  "slide_index must be >= 1",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	brief, _ := params["brief"].(string)
	config, _ := params["config"].(map[string]interface{})
	themePref := formatThemePreferences(config)
	maxAttempts := 3
	if parsed, ok := parseIntFromInterface(params["max_retries"]); ok && parsed > 0 {
		maxAttempts = parsed
	}
	attemptTimeout := slidePlanSlideAttemptTimeout
	if parsed, ok := parseIntFromInterface(params["attempt_timeout_ms"]); ok && parsed > 0 {
		attemptTimeout = time.Duration(parsed) * time.Millisecond
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining < attemptTimeout {
			attemptTimeout = remaining
		}
	}

	outlineText := strings.TrimSpace(collectOutlineText(input))
	block, ok := outlineBlockForSlide(outlineText, slideIndex)
	blockText := ""
	if ok {
		blockText = outlineBlockText(block)
	}
	if blockText == "" {
		blockText = truncateWithSuffix(outlineText, slidePlanContextPerSlideLimit)
	}
	outlineURLs := extractOutlineURLs(blockText)

	dataBank := ""
	if ok && outlineBlockNeedsDataBank(block) {
		dataBank = collectDataBankText(input)
	} else if !ok && outlineNeedsDataBank(outlineText) {
		dataBank = collectDataBankText(input)
	}

	deckTitle := extractDeckTitle(input)
	if strings.TrimSpace(deckTitle) == "" {
		deckTitle = fallbackDeckTitle(outlineText, brief)
	}

	promptParts := []string{
		fmt.Sprintf("Draft a single slide plan for slide %d.", slideIndex),
		"Return ONLY JSON with this shape:",
		`{"slide":{"id":1,"layout":"split|bullets|hero|table|chart","title":"...","subtitle":"...","bullets":["..."],"table":{"title":"...","columns":["..."],"rows":[["..."]],"notes":"optional"},"chart":{"type":"bar|line|pie","title":"...","categories":["..."],"series":[{"name":"...","values":[1,2]}],"notes":"optional"},"notes":"optional"},"image_required":true,"image_query":"..."}`,
		"HARD RULES:",
		"- slide.id must match the slide index.",
		"- title: 6-60 characters.",
		"- bullets: 3-6 items, 25-75 chars each when used.",
		"- table: 3-6 columns, 3-9 rows when used.",
		"- chart: 3-6 categories, 1-3 series when used.",
		"- Include at most one of table or chart (not both).",
		"- layout MUST match the primary content block:",
		"  - if chart is present -> layout = \"chart\"",
		"  - if table is present -> layout = \"table\"",
		"  - if image + bullets -> layout = \"split\"",
		"  - if image only -> layout = \"hero\"",
		"  - otherwise -> layout = \"bullets\"",
		"- If outline requests a table or chart (or mentions dataset/graph or pipe-delimited rows), include it and set layout accordingly.",
		"- If outline requests an image or visual, set image_required true and provide image_query.",
		"- Do NOT include images unless the outline provides a URL.",
		"- Bullets must be plain text (no markdown or quotes).",
		"- image_required true only if needed or requested.",
		"- Output JSON only, no commentary.",
	}

	if deckTitle != "" {
		promptParts = append(promptParts, "Deck title:\n"+deckTitle)
	}
	if brief != "" {
		promptParts = append(promptParts, "Brief:\n"+strings.TrimSpace(brief))
	}
	if themePref != "" {
		promptParts = append(promptParts, "Theme preferences:\n"+themePref)
	}
	if ok && block.Title != "" {
		promptParts = append(promptParts, "Slide title from outline:\n"+block.Title)
	}
	if blockText != "" {
		promptParts = append(promptParts, "Slide outline:\n"+blockText)
	}
	if dataBank != "" {
		promptParts = append(promptParts, "Data bank:\n"+dataBank)
	}

	prompt := strings.Join(promptParts, "\n\n")

	log.Debug().
		Str("plan_id", planContextValue(input, "plan_id")).
		Int("slide_index", slideIndex).
		Str("prompt", sanitizeForLog(prompt)).
		Msg("[slide_creator] slide plan per-slide prompt")

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
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptStart := time.Now()
		attemptCtx := ctx
		var cancel context.CancelFunc
		if attemptTimeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, attemptTimeout)
		}
		content, err = e.generateWithStructuredOutput(attemptCtx, "", prompt, model, SlidePlanSchema, 0.2)
		if cancel != nil {
			cancel()
		}
		if err == nil {
			log.Info().
				Str("plan_id", planContextValue(input, "plan_id")).
				Int("slide_index", slideIndex).
				Int("attempt", attempt).
				Int64("duration_ms", time.Since(attemptStart).Milliseconds()).
				Msg("[slide_creator] slide plan structured output succeeded")
			lastErr = nil
			break
		}
		lastErr = err
		log.Warn().
			Err(err).
			Str("plan_id", planContextValue(input, "plan_id")).
			Int("slide_index", slideIndex).
			Int("attempt", attempt).
			Int64("duration_ms", time.Since(attemptStart).Milliseconds()).
			Msg("[slide_creator] slide plan structured output failed")
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			if ctx.Err() != nil {
				break
			}
		}
	}
	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_CALL_FAILED",
				Message:  fmt.Sprintf("LLM generation failed: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	draft, err := parseSlidePlanDraft(content)
	if err != nil || validateSlidePlan(draft.Slide) != nil {
		log.Warn().
			Err(err).
			Int("slide_index", slideIndex).
			Msg("[slide_creator] slide plan parse failed, attempting repair")
		fixPrompt := fmt.Sprintf(`Fix the following JSON to be valid AND to conform exactly to the required shape and HARD RULES.

HARD RULES:
- Output ONLY a single JSON object.
- slide.id must be %d.
- title: 6-60 characters.
- bullets: 3-6 items, 25-75 chars each when used.
- table: 3-6 columns, 3-9 rows when used.
- chart: 3-6 categories, 1-3 series when used.
- Ensure the slide has bullets OR table OR chart.
- layout must match the primary content block:
  - chart -> layout "chart"
  - table -> layout "table"
  - image + bullets -> layout "split"
  - image only -> layout "hero"
  - otherwise -> layout "bullets"
- Output JSON only, no commentary.

BAD_JSON:
%s
`, slideIndex, content)
		fixed, fixErr := e.generateWithSystemPrompt(ctx, "", fixPrompt, model, 0.0)
		if fixErr != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "PARSE_ERROR",
					Message:  fmt.Sprintf("invalid slide plan (and repair failed): %v", fixErr),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
		fixed = extractJSONFromResponse(fixed)
		draft, err = parseSlidePlanDraft(fixed)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "PARSE_ERROR",
					Message:  fmt.Sprintf("could not parse slide plan JSON: %v", err),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	}

	draft.SlideIndex = slideIndex
	draft.Slide.ID = slideIndex
	draft.Slide.Title = trimToRunesNoEllipsis(strings.TrimSpace(draft.Slide.Title), 60)
	if strings.TrimSpace(draft.Slide.Title) == "" {
		draft.Slide.Title = fmt.Sprintf("Slide %d", slideIndex)
	}
	draft.Slide.Subtitle = strings.TrimSpace(draft.Slide.Subtitle)
	draft.Slide.Notes = strings.TrimSpace(draft.Slide.Notes)
	draft.Slide.Bullets = clampBullets(draft.Slide.Bullets, 6)
	if len(draft.Slide.Images) > 0 && len(outlineURLs) > 0 {
		allowed := map[string]struct{}{}
		for _, url := range outlineURLs {
			allowed[url] = struct{}{}
		}
		filtered := make([]SlideImage, 0, len(draft.Slide.Images))
		for _, img := range draft.Slide.Images {
			src := strings.TrimSpace(img.Src)
			if _, ok := allowed[src]; ok {
				filtered = append(filtered, img)
			}
		}
		draft.Slide.Images = filtered
	}
	if len(outlineURLs) == 0 {
		draft.Slide.Images = nil
	}

	if draft.ImageRequired == nil {
		required := slideNeedsImage(draft.Slide)
		if !required && ok && outlineBlockNeedsImage(block) {
			required = true
		}
		draft.ImageRequired = &required
	}
	if strings.TrimSpace(draft.ImageQuery) == "" && draft.ImageRequired != nil && *draft.ImageRequired {
		draft.ImageQuery = buildSlideImageQuery(deckTitle, draft.Slide)
	}

	output := map[string]interface{}{
		"type":           "slide_plan_slide",
		"slide_index":    draft.SlideIndex,
		"slide":          draft.Slide,
		"image_query":    strings.TrimSpace(draft.ImageQuery),
		"image_required": draft.ImageRequired,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func parseSlidePlanDraft(content string) (SlidePlanDraft, error) {
	var draft SlidePlanDraft
	if err := json.Unmarshal([]byte(content), &draft); err == nil {
		return draft, nil
	}
	if extracted := extractFirstJSONObject(content); extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &draft); err == nil {
			return draft, nil
		}
	}
	return SlidePlanDraft{}, fmt.Errorf("could not parse slide plan draft JSON")
}
