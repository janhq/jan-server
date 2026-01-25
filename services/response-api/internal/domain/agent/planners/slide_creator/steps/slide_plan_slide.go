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
	block, blockFound := outlineBlockForSlide(outlineText, slideIndex)

	// Extract structured content from outline block
	var slideContent OutlineSlideContent
	if blockFound {
		slideContent = ExtractSlideContent(block)
	}

	// Format structured content for prompt - this preserves all ideas from outline
	structuredContent := ""
	if blockFound {
		structuredContent = FormatSlideContentForPrompt(slideContent)
	}

	// Fallback: if no structured content, use truncated outline
	if structuredContent == "" && outlineText != "" {
		// Keep context focused - limit to 2000 chars for single slide
		maxLen := 2000
		if len(outlineText) > maxLen {
			structuredContent = outlineText[:maxLen] + "..."
		} else {
			structuredContent = outlineText
		}
	}

	outlineURLs := extractOutlineURLs(structuredContent)
	if len(outlineURLs) == 0 && blockFound {
		outlineURLs = extractOutlineURLs(slideContent.RawText)
	}

	// Get data bank only if this slide needs data (table/chart)
	dataBank := ""
	if slideContent.HasDataNeeds {
		dataBank = collectDataBankText(input)
		// Limit data bank size to keep context focused
		if len(dataBank) > 1500 {
			dataBank = dataBank[:1500] + "..."
		}
	}

	deckTitle := extractDeckTitle(input)
	if strings.TrimSpace(deckTitle) == "" {
		deckTitle = fallbackDeckTitle(outlineText, brief)
	}

	// Build focused prompt - keep it concise
	promptParts := []string{
		fmt.Sprintf("Create slide %d plan. Return ONLY valid JSON.", slideIndex),
	}

	// Add JSON schema (compact)
	promptParts = append(promptParts, `Schema: {"slide":{"id":N,"layout":"split|bullets|hero|table|chart","title":"...","subtitle":"...","bullets":["..."],"table":{"columns":["..."],"rows":[["..."]]},"chart":{"type":"bar|line|pie","categories":["..."],"series":[{"name":"...","values":[N]}]}},"image_required":bool,"image_query":"..."}`)

	// Add essential rules (condensed)
	rules := []string{
		fmt.Sprintf("slide.id=%d", slideIndex),
		"title: 6-60 chars",
		"bullets: 3-6 items when used",
		"table OR chart, not both",
		"layout must match content type",
	}

	// Add table/chart instruction if outline indicates need
	if slideContent.TableData != nil {
		rules = append(rules, "MUST include table with the provided data")
		rules = append(rules, "layout MUST be 'table'")
	}
	if slideContent.ChartHint != nil {
		rules = append(rules, fmt.Sprintf("MUST include %s chart with ALL data points from outline", slideContent.ChartHint.Type))
		rules = append(rules, "layout MUST be 'chart'")
	}

	promptParts = append(promptParts, "Rules: "+strings.Join(rules, "; "))

	// Add context - keep each section brief
	if deckTitle != "" {
		promptParts = append(promptParts, "Deck: "+trimToRunesNoEllipsis(deckTitle, 60))
	}

	// Include slide title from outline if available
	if blockFound && block.Title != "" {
		promptParts = append(promptParts, "Slide title: "+block.Title)
	}

	// Include structured content - this contains all ideas from outline
	if structuredContent != "" {
		promptParts = append(promptParts, "Content:\n"+structuredContent)
	}

	// Include pre-extracted table data if found in outline
	if slideContent.TableData != nil {
		tableJSON, _ := json.Marshal(slideContent.TableData)
		promptParts = append(promptParts, "Table data from outline:\n"+string(tableJSON))
	}

	// Include pre-extracted chart data if found in outline
	if slideContent.ChartHint != nil && len(slideContent.ChartHint.Values) > 0 {
		chartData := map[string]interface{}{
			"type":       slideContent.ChartHint.Type,
			"title":      slideContent.ChartHint.Title,
			"categories": slideContent.ChartHint.Categories,
			"series": []map[string]interface{}{
				{
					"name":   "Data",
					"values": slideContent.ChartHint.Values,
				},
			},
		}
		chartJSON, _ := json.Marshal(chartData)
		promptParts = append(promptParts, "Chart data from outline (MUST use these exact values):\n"+string(chartJSON))
	}

	// Include data bank if needed (already size-limited above)
	if dataBank != "" {
		promptParts = append(promptParts, "Data bank:\n"+dataBank)
	}

	// Theme preferences (brief)
	if themePref != "" && len(themePref) <= 200 {
		promptParts = append(promptParts, "Theme: "+themePref)
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

	// Validate and filter images
	if len(draft.Slide.Images) > 0 {
		// First, validate all image URLs
		draft.Slide.Images = FilterValidSlideImages(draft.Slide.Images)

		// If outline has URLs, only allow those specific URLs
		if len(outlineURLs) > 0 {
			allowed := map[string]struct{}{}
			for _, url := range outlineURLs {
				// Validate outline URLs too
				if result := ValidateImageURL(url); result.IsValid {
					allowed[url] = struct{}{}
				}
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
	}

	// If no valid outline URLs, clear LLM-generated images (will be filled by image search)
	if len(outlineURLs) == 0 {
		draft.Slide.Images = nil
	}

	if draft.ImageRequired == nil {
		required := slideNeedsImage(draft.Slide)
		if !required && blockFound && outlineBlockNeedsImage(block) {
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
