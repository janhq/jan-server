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

func (e *SlideCreatorExecutor) executeDeckTheme(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	brief, _ := params["brief"].(string)
	config, _ := params["config"].(map[string]interface{})
	themePref := formatThemePreferences(config)
	colorScheme := strings.TrimSpace(stringValue(config, "color_scheme"))
	outlineText := strings.TrimSpace(collectOutlineText(input))
	if outlineText != "" {
		outlineText = truncateWithSuffix(outlineText, slidePlanContextPerSlideLimit)
	}

	promptParts := []string{
		"Select a concise deck title and a high-contrast theme.",
		"Return ONLY JSON with this shape:",
		`{"title":"...","theme":{"primary_color":"#RRGGBB","accent_color":"#RRGGBB","background_color":"#RRGGBB","text_color":"#RRGGBB","font_family":"Segoe UI, Arial, Helvetica, sans-serif"}}`,
		"HARD RULES:",
		"- title length: 6-60 characters.",
		"- background must be very dark or very light.",
		"- text_color must contrast with background (near-white on dark, near-black on light).",
		"- Output JSON only. No commentary.",
	}

	if strings.TrimSpace(brief) != "" {
		promptParts = append(promptParts, "Brief:\n"+strings.TrimSpace(brief))
	}
	if themePref != "" {
		promptParts = append(promptParts, "Theme preferences:\n"+themePref)
	}
	if outlineText != "" {
		promptParts = append(promptParts, "Outline (summary):\n"+outlineText)
	}

	prompt := strings.Join(promptParts, "\n\n")

	log.Debug().
		Str("plan_id", planContextValue(input, "plan_id")).
		Str("prompt", sanitizeForLog(prompt)).
		Msg("[slide_creator] deck theme prompt")

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
		attemptStart := time.Now()
		content, err = e.generateWithStructuredOutput(ctx, "", prompt, model, DeckThemeSchema, 0.2)
		if err == nil {
			log.Info().
				Str("plan_id", planContextValue(input, "plan_id")).
				Int("attempt", attempt).
				Int64("duration_ms", time.Since(attemptStart).Milliseconds()).
				Msg("[slide_creator] deck theme structured output succeeded")
			lastErr = nil
			break
		}
		lastErr = err
		log.Warn().
			Err(err).
			Str("plan_id", planContextValue(input, "plan_id")).
			Int("attempt", attempt).
			Int64("duration_ms", time.Since(attemptStart).Milliseconds()).
			Msg("[slide_creator] deck theme structured output failed")
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			break
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

	var theme DeckTheme
	if err := json.Unmarshal([]byte(content), &theme); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("could not parse deck theme JSON: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}
	if strings.TrimSpace(theme.Title) == "" {
		theme.Title = fallbackDeckTitle(outlineText, brief)
	}
	theme.Title = trimToRunesNoEllipsis(strings.TrimSpace(theme.Title), 60)
	theme.Theme = normalizeTheme(theme.Theme)
	theme.Theme = applyColorScheme(theme.Theme, colorScheme, theme.Title)

	output := map[string]interface{}{
		"type":  "deck_theme",
		"title": theme.Title,
		"theme": theme.Theme,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}
