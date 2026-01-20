package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func ExecutePlanAndTemplate(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executePlanAndTemplate started")
	baseContext := BuildAccumulatedContext(input)
	log.Debug().Int("context_length", len(baseContext)).Msg("[slide_generator] built accumulated context")

	config, _ := params["config"].(map[string]interface{})
	numSlides := 10
	if n, ok := config["num_slides"].(float64); ok {
		numSlides = int(n)
	}
	theme, _ := config["theme"].(string)

	if deps.CollectImageAssets == nil || deps.CollectDataBankText == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXECUTOR_MISSING",
				Message:  "collectImageAssets/collectDataBankText not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	assets := limitImageAssets(deps.CollectImageAssets(input), 4)
	assetsJSON, _ := json.Marshal(assets)
	baseDataBank := deps.CollectDataBankText(input)
	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerAndTemplatePrompt)

	model := getModelFromContext(input)
	log.Debug().
		Str("model", model).
		Float64("temperature", deps.Temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("context_length", len(baseContext)).
		Int("data_bank_length", len(baseDataBank)).
		Int("assets_count", len(assets)).
		Msg("[slide_generator] plan_and_template prompt prepared")
	if deps.GenerateWithStructuredOutputWithMaxTokensAndUsage == nil || deps.GenerateWithSystemPromptWithMaxTokens == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := cloneSchema(schemas.PlanAndTemplateSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var planAndTemplate schemas.PlanAndTemplate
	var totalUsage *llm.Usage

	// Retry strategy: All attempts use structured output with progressively smaller context
	for attempt := 1; attempt <= 3; attempt++ {
		contextLimit := 16000
		dataBankLimit := 6000
		if attempt == 2 {
			contextLimit = 8000
			dataBankLimit = 3000
		} else if attempt == 3 {
			contextLimit = 4000
			dataBankLimit = 1500
		}
		contextData := limitText(baseContext, contextLimit)
		dataBankText := limitText(baseDataBank, dataBankLimit)
		userPrompt := fmt.Sprintf(
			"BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s",
			contextData,
			numSlides,
			theme,
			string(assetsJSON),
			dataBankText,
		)

		log.Debug().
			Int("attempt", attempt).
			Str("model", model).
			Int("context_limit", contextLimit).
			Int("databank_limit", dataBankLimit).
			Msg("[slide_generator] plan_and_template LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPrompt, userPrompt, model, schema, intPtr(planTemplateMaxTokens))
		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[slide_generator] plan_and_template LLM call failed")
			continue
		}

		// Track token usage
		if llmResult.Usage != nil {
			if totalUsage == nil {
				totalUsage = &llm.Usage{}
			}
			totalUsage.PromptTokens += llmResult.Usage.PromptTokens
			totalUsage.CompletionTokens += llmResult.Usage.CompletionTokens
			totalUsage.TotalTokens += llmResult.Usage.TotalTokens
			log.Info().
				Int("attempt", attempt).
				Int("prompt_tokens", llmResult.Usage.PromptTokens).
				Int("completion_tokens", llmResult.Usage.CompletionTokens).
				Int("total_tokens", llmResult.Usage.TotalTokens).
				Msg("[slide_generator] plan_and_template LLM token usage")
		}

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] plan_and_template LLM response received")

		// Handle empty response - this is a different issue than truncation
		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response")
			log.Warn().
				Int("attempt", attempt).
				Msg("[slide_generator] plan_and_template LLM returned empty response - may indicate rate limiting, content filter, or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &planAndTemplate); err != nil {
			lastErr = err
			if isTruncatedJSON(err, result) {
				log.Warn().
					Int("attempt", attempt).
					Int("response_length", len(result)).
					Msg("[slide_generator] plan_and_template response appears truncated (incomplete JSON), retrying with smaller context")
			}
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Str("response_preview", truncateForLogString(result, 300)).
				Str("response_full", result).
				Msg("[slide_generator] failed to parse plan+template")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("[slide_generator] plan_and_template successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse plan+template after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().
		Int("slide_count", len(planAndTemplate.Plan.Slides)).
		Int("recommended_slides", planAndTemplate.Plan.RecommendedSlideCount).
		Interface("template", planAndTemplate.Template).
		Msg("[slide_generator] parsed plan and template")

	if deps.NormalizePlanIndices != nil {
		deps.NormalizePlanIndices(&planAndTemplate.Plan)
	}
	if deps.NormalizeTemplateComponents != nil {
		deps.NormalizeTemplateComponents(&planAndTemplate.Template)
	}
	if deps.NormalizeTemplateLayouts != nil {
		deps.NormalizeTemplateLayouts(&planAndTemplate.Plan, &planAndTemplate.Template)
	}

	output := map[string]interface{}{
		"type":               "plan_and_template",
		"plan":               planAndTemplate.Plan,
		"template":           planAndTemplate.Template,
		"recommended_slides": planAndTemplate.Plan.RecommendedSlideCount,
	}
	// Include token usage in output if available
	if totalUsage != nil {
		output["token_usage"] = map[string]int{
			"prompt_tokens":     totalUsage.PromptTokens,
			"completion_tokens": totalUsage.CompletionTokens,
			"total_tokens":      totalUsage.TotalTokens,
		}
		log.Info().
			Int("total_prompt_tokens", totalUsage.PromptTokens).
			Int("total_completion_tokens", totalUsage.CompletionTokens).
			Int("total_tokens", totalUsage.TotalTokens).
			Msg("[slide_generator] plan_and_template total token usage")
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executePlanAndTemplate completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}
