package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	model := getModelFromContext(input)
	log.Debug().
		Str("model", model).
		Float64("temperature", deps.Temperature).
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

	var lastErr error
	var planResult schemas.SlidePlan
	var templateResult schemas.SlideTemplate
	var totalUsage *llm.Usage
	var retryErrors []string

	systemPromptPlan := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerPrompt)
	systemPromptTemplate := fmt.Sprintf("%s\n%s", sizeGuardPrompt, templatePrompt)
	planSchema := prepareSchema(schemas.SlidePlanSchema)
	templateSchema := prepareSchema(schemas.SlideTemplateSchema)

	appendUsage := func(usage *llm.Usage, stage string, attempt int) {
		if usage == nil {
			return
		}
		if totalUsage == nil {
			totalUsage = &llm.Usage{}
		}
		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.CompletionTokens += usage.CompletionTokens
		totalUsage.TotalTokens += usage.TotalTokens
		log.Info().
			Str("stage", stage).
			Int("attempt", attempt).
			Int("prompt_tokens", usage.PromptTokens).
			Int("completion_tokens", usage.CompletionTokens).
			Int("total_tokens", usage.TotalTokens).
			Msg("[slide_generator] plan_and_template LLM token usage")
	}

	// Step 1: generate plan
	for attempt := 1; attempt <= 5; attempt++ {
		contextLimit := 16000
		dataBankLimit := 6000
		if attempt == 2 {
			contextLimit = 8000
			dataBankLimit = 3000
		} else if attempt >= 3 {
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
		if attempt > 1 && len(retryErrors) > 0 {
			retryContext := strings.Join(retryErrors, "\n")
			userPrompt = fmt.Sprintf("%s\n\nPREVIOUS_ERRORS:\n%s", userPrompt, limitText(retryContext, 1200))
		}

		log.Debug().
			Int("attempt", attempt).
			Str("model", model).
			Int("context_limit", contextLimit).
			Int("databank_limit", dataBankLimit).
			Msg("[slide_generator] plan LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPromptPlan, userPrompt, model, planSchema, intPtr(planTemplateMaxTokens))
		if err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan LLM call failed: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[slide_generator] plan LLM call failed")
			continue
		}

		appendUsage(llmResult.Usage, "plan", attempt)

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] plan LLM response received")

		// Handle empty response - this is a different issue than truncation
		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response")
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan empty response", attempt))
			log.Warn().
				Int("attempt", attempt).
				Msg("[slide_generator] plan LLM returned empty response - may indicate rate limiting, content filter, or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &planResult); err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan parse error: %v", attempt, err))
			if isTruncatedJSON(err, result) {
				log.Warn().
					Int("attempt", attempt).
					Int("response_length", len(result)).
					Msg("[slide_generator] plan response appears truncated (incomplete JSON), retrying with smaller context")
			}
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse plan")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("[slide_generator] plan successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse plan after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Step 2: generate template
	retryErrors = nil
	planJSON, _ := json.Marshal(planResult)
	layoutIDs := extractSuggestedLayoutIDs(planResult.Slides)
	layoutIDsJSON, _ := json.Marshal(layoutIDs)
	for attempt := 1; attempt <= 5; attempt++ {
		contextLimit := 8000
		if attempt == 2 {
			contextLimit = 4000
		} else if attempt >= 3 {
			contextLimit = 2000
		}
		contextData := limitText(baseContext, contextLimit)
		userPrompt := fmt.Sprintf(
			"BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s\n\nPLAN (locked):\n%s\n\nREQUIRED_LAYOUT_IDS:\n%s\n\nASSETS AVAILABLE:\n%s",
			contextData,
			numSlides,
			theme,
			string(planJSON),
			string(layoutIDsJSON),
			string(assetsJSON),
		)
		if attempt > 1 && len(retryErrors) > 0 {
			retryContext := strings.Join(retryErrors, "\n")
			userPrompt = fmt.Sprintf("%s\n\nPREVIOUS_ERRORS:\n%s", userPrompt, limitText(retryContext, 1200))
		}

		log.Debug().
			Int("attempt", attempt).
			Str("model", model).
			Int("context_limit", contextLimit).
			Msg("[slide_generator] template LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPromptTemplate, userPrompt, model, templateSchema, intPtr(planTemplateMaxTokens))
		if err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template LLM call failed: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[slide_generator] template LLM call failed")
			continue
		}

		appendUsage(llmResult.Usage, "template", attempt)

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] template LLM response received")

		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response")
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template empty response", attempt))
			log.Warn().
				Int("attempt", attempt).
				Msg("[slide_generator] template LLM returned empty response - may indicate rate limiting, content filter, or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &templateResult); err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template parse error: %v", attempt, err))
			if isTruncatedJSON(err, result) {
				log.Warn().
					Int("attempt", attempt).
					Int("response_length", len(result)).
					Msg("[slide_generator] template response appears truncated (incomplete JSON), retrying with smaller context")
			}
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse template")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("[slide_generator] template successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse template after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	planAndTemplate := schemas.PlanAndTemplate{
		Plan:     planResult,
		Template: templateResult,
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

func ExecutePlanOnly(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executePlanOnly started")
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

	model := getModelFromContext(input)
	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerPrompt)

	if deps.GenerateWithStructuredOutputWithMaxTokensAndUsage == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := prepareSchema(schemas.SlidePlanSchema)
	var lastErr error
	var planResult schemas.SlidePlan
	var totalUsage *llm.Usage
	var retryErrors []string

	for attempt := 1; attempt <= 5; attempt++ {
		contextLimit := 16000
		dataBankLimit := 6000
		if attempt == 2 {
			contextLimit = 8000
			dataBankLimit = 3000
		} else if attempt >= 3 {
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
		if attempt > 1 && len(retryErrors) > 0 {
			retryContext := strings.Join(retryErrors, "\n")
			userPrompt = fmt.Sprintf("%s\n\nPREVIOUS_ERRORS:\n%s", userPrompt, limitText(retryContext, 1200))
		}

		log.Debug().
			Int("attempt", attempt).
			Str("model", model).
			Int("context_limit", contextLimit).
			Int("databank_limit", dataBankLimit).
			Msg("[slide_generator] plan LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPrompt, userPrompt, model, schema, intPtr(planTemplateMaxTokens))
		if err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan LLM call failed: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[slide_generator] plan LLM call failed")
			continue
		}

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
				Msg("[slide_generator] plan LLM token usage")
		}

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] plan LLM response received")

		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response")
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan empty response", attempt))
			log.Warn().
				Int("attempt", attempt).
				Msg("[slide_generator] plan LLM returned empty response - may indicate rate limiting, content filter, or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &planResult); err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d plan parse error: %v", attempt, err))
			if isTruncatedJSON(err, result) {
				log.Warn().
					Int("attempt", attempt).
					Int("response_length", len(result)).
					Msg("[slide_generator] plan response appears truncated (incomplete JSON), retrying with smaller context")
			}
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse plan")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("[slide_generator] plan successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse plan after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	output := map[string]interface{}{
		"type":               "plan",
		"plan":               planResult,
		"recommended_slides": planResult.RecommendedSlideCount,
	}
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
			Msg("[slide_generator] plan total token usage")
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executePlanOnly completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func ExecuteTemplateOnly(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeTemplateOnly started")
	baseContext := BuildAccumulatedContext(input)
	log.Debug().Int("context_length", len(baseContext)).Msg("[slide_generator] built accumulated context")

	config, _ := params["config"].(map[string]interface{})
	numSlides := 10
	if n, ok := config["num_slides"].(float64); ok {
		numSlides = int(n)
	}
	theme, _ := config["theme"].(string)

	if deps.CollectImageAssets == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXECUTOR_MISSING",
				Message:  "collectImageAssets not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	assets := limitImageAssets(deps.CollectImageAssets(input), 4)
	assetsJSON, _ := json.Marshal(assets)

	planResult, err := extractPlanFromOutputs(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_PLAN",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	model := getModelFromContext(input)
	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, templatePrompt)

	if deps.GenerateWithStructuredOutputWithMaxTokensAndUsage == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := prepareSchema(schemas.SlideTemplateSchema)
	var lastErr error
	var templateResult schemas.SlideTemplate
	var totalUsage *llm.Usage
	var retryErrors []string

	planJSON, _ := json.Marshal(planResult)
	layoutIDs := extractSuggestedLayoutIDs(planResult.Slides)
	layoutIDsJSON, _ := json.Marshal(layoutIDs)

	for attempt := 1; attempt <= 5; attempt++ {
		contextLimit := 8000
		if attempt == 2 {
			contextLimit = 4000
		} else if attempt >= 3 {
			contextLimit = 2000
		}
		contextData := limitText(baseContext, contextLimit)
		userPrompt := fmt.Sprintf(
			"BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s\n\nPLAN (locked):\n%s\n\nREQUIRED_LAYOUT_IDS:\n%s\n\nASSETS AVAILABLE:\n%s",
			contextData,
			numSlides,
			theme,
			string(planJSON),
			string(layoutIDsJSON),
			string(assetsJSON),
		)
		if attempt > 1 && len(retryErrors) > 0 {
			retryContext := strings.Join(retryErrors, "\n")
			userPrompt = fmt.Sprintf("%s\n\nPREVIOUS_ERRORS:\n%s", userPrompt, limitText(retryContext, 1200))
		}

		log.Debug().
			Int("attempt", attempt).
			Str("model", model).
			Int("context_limit", contextLimit).
			Msg("[slide_generator] template LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPrompt, userPrompt, model, schema, intPtr(planTemplateMaxTokens))
		if err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template LLM call failed: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[slide_generator] template LLM call failed")
			continue
		}

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
				Msg("[slide_generator] template LLM token usage")
		}

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] template LLM response received")

		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response")
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template empty response", attempt))
			log.Warn().
				Int("attempt", attempt).
				Msg("[slide_generator] template LLM returned empty response - may indicate rate limiting, content filter, or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &templateResult); err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d template parse error: %v", attempt, err))
			if isTruncatedJSON(err, result) {
				log.Warn().
					Int("attempt", attempt).
					Int("response_length", len(result)).
					Msg("[slide_generator] template response appears truncated (incomplete JSON), retrying with smaller context")
			}
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse template")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("[slide_generator] template successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse template after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	output := map[string]interface{}{
		"type":     "template",
		"template": templateResult,
	}
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
			Msg("[slide_generator] template total token usage")
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executeTemplateOnly completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func ExecuteAssemblePlanTemplate(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeAssemblePlanTemplate started")

	planResult, err := extractPlanFromOutputs(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_PLAN",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}
	templateResult, err := extractTemplateFromOutputs(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_TEMPLATE",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	output := map[string]interface{}{
		"type":               "plan_and_template",
		"plan":               planResult,
		"template":           templateResult,
		"recommended_slides": planResult.RecommendedSlideCount,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executeAssemblePlanTemplate completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func extractPlanFromOutputs(input agent.ExecutionInput) (*schemas.SlidePlan, error) {
	if input.PreviousOutput != nil {
		if planResult := parsePlanOutput(input.PreviousOutput); planResult != nil {
			return planResult, nil
		}
	}
	for _, output := range input.AccumulatedOutputs {
		if planResult := parsePlanOutput(output); planResult != nil {
			return planResult, nil
		}
	}
	return nil, fmt.Errorf("plan not found in previous outputs")
}

func extractTemplateFromOutputs(input agent.ExecutionInput) (*schemas.SlideTemplate, error) {
	if input.PreviousOutput != nil {
		if templateResult := parseTemplateOutput(input.PreviousOutput); templateResult != nil {
			return templateResult, nil
		}
	}
	for _, output := range input.AccumulatedOutputs {
		if templateResult := parseTemplateOutput(output); templateResult != nil {
			return templateResult, nil
		}
	}
	return nil, fmt.Errorf("template not found in previous outputs")
}

func parsePlanOutput(raw json.RawMessage) *schemas.SlidePlan {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if payloadType, _ := payload["type"].(string); payloadType != "plan" {
		return nil
	}
	if planData, ok := payload["plan"]; ok {
		encoded, _ := json.Marshal(planData)
		var plan schemas.SlidePlan
		if err := json.Unmarshal(encoded, &plan); err == nil {
			return &plan
		}
	}
	return nil
}

func parseTemplateOutput(raw json.RawMessage) *schemas.SlideTemplate {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if payloadType, _ := payload["type"].(string); payloadType != "template" {
		return nil
	}
	if templateData, ok := payload["template"]; ok {
		encoded, _ := json.Marshal(templateData)
		var tmpl schemas.SlideTemplate
		if err := json.Unmarshal(encoded, &tmpl); err == nil {
			return &tmpl
		}
	}
	return nil
}

func extractSuggestedLayoutIDs(slides []schemas.PlanEntry) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(slides))
	for _, slide := range slides {
		layout := strings.TrimSpace(slide.SuggestedLayout)
		if layout == "" || seen[layout] {
			continue
		}
		seen[layout] = true
		out = append(out, layout)
	}
	return out
}
