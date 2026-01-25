package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_creator/schemas"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

const dataBankPrompt = `
You are the Data Bank Extractor for a slide-deck generation system.
Given BRIEF, ASSETS, and RESEARCH, extract concrete facts and chart-ready datasets.

OUTPUT FORMAT (STRICT):
- Return ONLY valid JSON that matches the provided schema.
- Do NOT wrap in markdown, code fences, or commentary.
- Do NOT include any extra keys outside the schema.

RULES:
- Facts must be atomic, sourced, and include a date when available.
- Datasets must be ready to use in charts (labels + numeric series).
- Use ONLY data that appears in the provided research context.
- If data is missing, leave the dataset list empty instead of inventing values.
`

const (
	dataBankImageAssetLimit            = 2
	dataBankImageAssetLimitSingleSlide = 1
)

func (e *SlideCreatorExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(input.StepParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_creator] failed to parse LLM call parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	if requiresUser, ok := params["requires_user"].(bool); ok && requiresUser {
		prompt, _ := params["prompt"].(string)
		optionsCount := 0
		if rawCount, ok := params["options_count"].(float64); ok {
			optionsCount = int(rawCount)
		}

		options := make([]string, 0, optionsCount)
		for i := 0; i < optionsCount; i++ {
			options = append(options, fmt.Sprintf("option_%d", i+1))
		}

		outputBytes, _ := json.Marshal(map[string]interface{}{
			"status":        "waiting_for_user",
			"prompt":        prompt,
			"options":       options,
			"options_count": optionsCount,
		})

		return &agent.ExecutionResult{
			Status:       status.StatusCompleted,
			Output:       outputBytes,
			RequiresUser: true,
			UserPrompt:   &prompt,
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "select_templates":
		return e.executeSelectTemplates(ctx, params, input)
	case "deck_theme":
		return e.executeDeckTheme(ctx, params, input)
	case "slide_plan":
		return e.executeSlidePlan(ctx, params, input)
	case "slide_plan_slide":
		return e.executeSlidePlanSlide(ctx, params, input)
	case "reasoning":
		return e.executeOutlineReasoning(ctx, params, input)
	case "data_bank":
		return e.executeDataBank(ctx, params, input)
	default:
		return e.executeOutlineReasoning(ctx, params, input)
	}
}

func (e *SlideCreatorExecutor) executeOutlineReasoning(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	description, _ := params["description"].(string)
	brief, _ := params["brief"].(string)
	config, _ := params["config"].(map[string]interface{})
	numSlides, _ := parseIntFromInterface(config["num_slides"])
	contextData := buildAccumulatedContext(input)
	if strings.TrimSpace(contextData) == "" {
		contextData = "[No previous context available]"
	}
	if numSlides == 1 && len(contextData) > slidePlanContextPerSlideLimit {
		contextData = truncateWithSuffix(contextData, slidePlanContextPerSlideLimit)
	}
	singleSlideNote := ""
	if numSlides == 1 {
		singleSlideNote = "\n\nFor single-slide requests, avoid listing full URLs; summarize sources by publisher name only."
	}
	prompt := fmt.Sprintf(
		`You are creating a detailed presentation outline. %s

USER REQUEST:
%s

RESEARCH DATA:
%s%s

OUTLINE REQUIREMENTS (CRITICAL - must follow):
1. RICH DATA: Every slide MUST include specific statistics, numbers, percentages, or metrics. NO vague statements.
   - BAD: "Sales increased significantly"
   - GOOD: "Sales grew 47%% from $2.3M to $3.4M in Q3 2024"

2. DETAILED BULLETS: Each bullet point should be 2-3 sentences with supporting evidence.
   - BAD: "Lions hunt at night"
   - GOOD: "Lions hunt primarily between 8PM-6AM (78%% of kills), using darkness to offset their 30%% slower sprint speed vs prey. Studies show success rates drop to 17%% in daylight vs 32%% at night."

3. DATA VISUALIZATIONS: Include at least 2-3 slides with:
   - Charts: Specify type (bar/line/pie), exact categories and values
     Format: "CHART: [type] - [title] - Categories: [X, Y, Z] - Values: [10, 20, 30]"
   - Tables: Include full data with headers and rows
     Format: "TABLE: [title] | Col1 | Col2 | Col3 | Row1: val1, val2, val3 | Row2: ..."

4. IMAGE SUGGESTIONS: For each slide that needs visuals, suggest:
   - "IMAGE: [description of ideal image, specific subject, composition]"

5. KEY METRICS: Include a "KEY STATS" section on appropriate slides:
   - Format: "STATS: [Metric1: Value1] | [Metric2: Value2] | [Metric3: Value3]"

6. STRUCTURE: For each slide include:
   - Clear title (action-oriented, not generic)
   - 4-6 detailed bullet points with data
   - Visual element suggestion (chart/table/image/stats)

OUTPUT: Plain text outline with slide headers and detailed content. Be comprehensive and data-rich.`,
		description,
		brief,
		contextData,
		singleSlideNote,
	)

	log.Debug().
		Str("plan_id", planContextValue(input, "plan_id")).
		Str("prompt", sanitizeForLog(prompt)).
		Msg("[slide_creator] outline prompt")

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

	response, err := e.generateWithModel(ctx, prompt, model, 0.3)
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

	output := map[string]interface{}{
		"type":        "llm_response",
		"action":      "reasoning",
		"description": description,
		"content":     response,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().
		Int("response_length", len(response)).
		Msg("[slide_creator] reasoning completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideCreatorExecutor) executeDataBank(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_creator] executeDataBank started")
	outlineText := strings.TrimSpace(collectOutlineText(input))
	if outlineText != "" && !outlineNeedsDataBank(outlineText) {
		outputBytes, _ := json.Marshal(map[string]interface{}{
			"type":    "data_bank",
			"skipped": true,
			"reason":  "outline_no_data_markers",
		})
		return &agent.ExecutionResult{
			Status: status.StatusCompleted,
			Output: outputBytes,
		}, nil
	}
	config, _ := params["config"].(map[string]interface{})
	numSlides, _ := parseIntFromInterface(config["num_slides"])
	contextData := buildAccumulatedContext(input)
	if numSlides == 1 && len(contextData) > slidePlanContextPerSlideLimit {
		contextData = truncateWithSuffix(contextData, slidePlanContextPerSlideLimit)
	}
	brief, _ := params["brief"].(string)
	assetLimit := dataBankImageAssetLimit
	if numSlides == 1 {
		assetLimit = dataBankImageAssetLimitSingleSlide
	}
	assets := limitImageAssets(collectImageAssets(input), assetLimit)
	assetsJSON, _ := json.Marshal(compactImageAssetsForPrompt(assets))

	systemPrompt := dataBankPrompt
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nRESEARCH:\n%s\n\nASSETS AVAILABLE:\n%s", brief, contextData, string(assetsJSON))

	log.Debug().
		Str("plan_id", planContextValue(input, "plan_id")).
		Str("system_prompt", sanitizeForLog(systemPrompt)).
		Str("user_prompt", sanitizeForLog(userPrompt)).
		Msg("[slide_creator] data bank prompt")

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

	schema := schemas.DataBankSchema

	var lastErr error
	var dataBank schemas.DataBank
	for attempt := 1; attempt <= 3; attempt++ {
		result, err := e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema, 0.1)
		if err != nil {
			lastErr = err
			continue
		}
		if err := json.Unmarshal([]byte(result), &dataBank); err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse data bank after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentBytes, _ := json.Marshal(dataBank)
	output := map[string]interface{}{
		"type":    "data_bank",
		"data":    dataBank,
		"content": string(contentBytes),
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().
		Int("facts", len(dataBank.Facts)).
		Int("datasets", len(dataBank.Datasets)).
		Msg("[slide_creator] executeDataBank completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}
