package slide_generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeToolCall started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse step parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse step parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "upload_slide_spec":
		return e.executeUploadSlideSpec(ctx, params, input)
	case "render_deck":
		return e.executeRenderScript(ctx, params, input)
	default:
		return e.executeGenericToolCall(ctx, step, params, input)
	}
}

func (e *SlideGeneratorExecutor) executeGenericToolCall(ctx context.Context, step *plan.Step, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	toolName, _ := params["tool"].(string)
	log.Debug().Str("tool_name", toolName).Interface("params", params).Msg("[slide_generator] executeGenericToolCall started")
	if toolName == "" {
		log.Error().Msg("[slide_generator] no tool specified")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_TOOL",
				Message:  "no tool specified",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	description, _ := params["description"].(string)
	toolArgs, err := e.buildToolArguments(toolName, params, input, description)
	if err != nil {
		if isNonCriticalToolForSlides(toolName) {
			return buildSkippedToolResultForSlides(toolName, err.Error(), "invalid_arguments"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_ARGUMENTS",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	callReq := tool.CallRequest{
		Name:      toolName,
		Arguments: toolArgs,
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	log.Debug().Str("tool_name", toolName).Interface("arguments", toolArgs).Msg("[slide_generator] calling tool")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		log.Error().Err(err).Str("tool_name", toolName).Msg("[slide_generator] tool call failed")
		if isNonCriticalToolForSlides(toolName) {
			return buildSkippedToolResultForSlides(toolName, err.Error(), "tool_call_failed"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "TOOL_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if result != nil && result.IsError && isNonCriticalToolForSlides(toolName) {
		return buildSkippedToolResultForSlides(toolName, "tool reported error", "tool_error"), nil
	}

	outputBytes, _ := json.Marshal(result)
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeLLMCall started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse LLM call parameters")
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
	case "plan_and_template":
		return e.executePlanAndTemplate(ctx, params, input)
	case "generate_single_slide":
		return e.executeSingleSlide(ctx, params, input)
	case "reasoning":
		return e.executeReasoning(ctx, params, input)
	case "data_bank":
		return e.executeDataBank(ctx, params, input)
	default:
		return e.executeReasoning(ctx, params, input)
	}
}

func (e *SlideGeneratorExecutor) executeReasoning(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	description, _ := params["description"].(string)
	contextData := e.buildAccumulatedContext(input)
	prompt := fmt.Sprintf(
		"Analyze and plan the slide structure. %s\n\nResearch findings:\n%s\n\nExtract concrete data for any requested tables (column headers + row entries) and include them in the outline.\nProvide a clear, concise outline for the presentation.\nReturn plain text only.",
		description,
		contextData,
	)

	model := e.getModelFromContext(input)
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

	response, err := e.generateWithModel(ctx, prompt, model)
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
		Msg("[slide_generator] reasoning completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeDataBank(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeDataBank started")
	contextData := e.buildAccumulatedContext(input)
	assets := limitImageAssets(e.collectImageAssets(input), 4)
	assetsJSON, _ := json.Marshal(assets)

	systemPrompt := dataBankPrompt
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nASSETS AVAILABLE:\n%s", contextData, string(assetsJSON))

	model := e.getModelFromContext(input)
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

	schema := cloneSchema(schemas.DataBankSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var dataBank schemas.DataBank

	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			schemaJSON, _ := json.MarshalIndent(schemas.DataBankSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Msg("[slide_generator] data_bank LLM call failed")
			continue
		}

		if err := json.Unmarshal([]byte(result), &dataBank); err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Msg("[slide_generator] failed to parse data_bank result")
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
		Msg("[slide_generator] executeDataBank completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executePlanAndTemplate(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executePlanAndTemplate started")
	contextData := e.buildAccumulatedContext(input)
	log.Debug().Int("context_length", len(contextData)).Msg("[slide_generator] built accumulated context")

	config, _ := params["config"].(map[string]interface{})
	numSlides := 10
	if n, ok := config["num_slides"].(float64); ok {
		numSlides = int(n)
	}
	theme, _ := config["theme"].(string)

	assets := limitImageAssets(e.collectImageAssets(input), 4)
	assetsJSON, _ := json.Marshal(assets)
	dataBankText := e.collectDataBankText(input)
	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerAndTemplatePrompt)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s", contextData, numSlides, theme, string(assetsJSON), dataBankText)

	model := e.getModelFromContext(input)
	log.Debug().
		Str("model", model).
		Float64("temperature", e.temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("user_prompt_length", len(userPrompt)).
		Str("user_prompt_preview", truncateForLogString(userPrompt, 300)).
		Msg("[slide_generator] plan_and_template prompt prepared")
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

	schema := cloneSchema(schemas.PlanAndTemplateSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var planAndTemplate schemas.PlanAndTemplate

	// Enhanced retry strategy:
	// Attempt 1-2: Use response_format with structured output (OpenAI enforced schema)
	// Attempt 3: Fallback to schema in prompt (if structured output fails)
	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			log.Debug().
				Int("attempt", attempt).
				Str("model", model).
				Str("method", "structured_output").
				Msg("[slide_generator] plan_and_template LLM call started")
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Fallback: append schema to prompt
			log.Info().
				Int("attempt", attempt).
				Str("model", model).
				Str("method", "schema_in_prompt").
				Msg("[slide_generator] plan_and_template using fallback method (schema in prompt)")
			schemaJSON, _ := json.MarshalIndent(schemas.PlanAndTemplateSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				// Extract JSON from potential markdown code blocks
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Bool("structured_output", useStructuredOutput).
				Msg("[slide_generator] plan_and_template LLM call failed")
			continue
		}

		log.Debug().
			Int("attempt", attempt).
			Bool("structured_output", useStructuredOutput).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] plan_and_template LLM response received")

		if err := json.Unmarshal([]byte(result), &planAndTemplate); err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Bool("structured_output", useStructuredOutput).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse plan+template")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Bool("structured_output", useStructuredOutput).
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

	normalizePlanIndices(&planAndTemplate.Plan)
	normalizeTemplateComponents(&planAndTemplate.Template)
	normalizeTemplateLayouts(&planAndTemplate.Plan, &planAndTemplate.Template)

	output := map[string]interface{}{
		"type":               "plan_and_template",
		"plan":               planAndTemplate.Plan,
		"template":           planAndTemplate.Template,
		"recommended_slides": planAndTemplate.Plan.RecommendedSlideCount,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executePlanAndTemplate completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeSingleSlide(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	slideIndex := 0
	if raw, ok := params["slide_index"].(float64); ok {
		slideIndex = int(raw)
	}
	log.Debug().Int("slide_index", slideIndex).Msg("[slide_generator] executeSingleSlide started")
	if slideIndex <= 0 {
		log.Error().Int("slide_index", slideIndex).Msg("[slide_generator] invalid slide index")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_SLIDE_INDEX",
				Message:  "slide_index is required",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	planAndTemplate := e.extractPlanAndTemplate(input)
	if planAndTemplate == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_CONTEXT",
				Message:  "Plan and template not found in previous outputs",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Use 1-based slideIndex to access 0-based array
	// (LLM may use 0-based or 1-based Index field, so use array position instead)
	arrayIndex := slideIndex - 1
	if arrayIndex < 0 || arrayIndex >= len(planAndTemplate.Plan.Slides) {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PLAN_ENTRY_NOT_FOUND",
				Message:  fmt.Sprintf("No plan entry for slide %d (array index %d out of range, plan has %d slides)", slideIndex, arrayIndex, len(planAndTemplate.Plan.Slides)),
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	planEntry := &planAndTemplate.Plan.Slides[arrayIndex]

	contextData := e.buildAccumulatedContext(input)
	assets := limitImageAssets(e.collectImageAssets(input), 4)
	assetsJSON, _ := json.Marshal(assets)
	dataBankText := e.collectDataBankText(input)
	templateJSON, _ := json.Marshal(planAndTemplate.Template)
	planEntryJSON, _ := json.Marshal(planEntry)
	themeJSON, _ := json.Marshal(planAndTemplate.Template.Theme)

	systemPrompt := slideWriterPrompt(slideIndex)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nLOCKED TEMPLATE:\n%s\n\nTHEME:\n%s\n\nPLAN ENTRY (slide %d):\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s", contextData, string(templateJSON), string(themeJSON), slideIndex, string(planEntryJSON), string(assetsJSON), dataBankText)

	model := e.getModelFromContext(input)
	log.Debug().
		Int("slide_index", slideIndex).
		Str("model", model).
		Float64("temperature", e.temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("user_prompt_length", len(userPrompt)).
		Str("user_prompt_preview", truncateForLogString(userPrompt, 300)).
		Msg("[slide_generator] slide prompt prepared")
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

	schema := cloneSchema(schemas.SlideGenResultSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var slideResult schemas.SlideGenResult

	// Enhanced retry strategy:
	// Attempt 1-2: Use response_format with structured output (OpenAI enforced schema)
	// Attempt 3: Fallback to schema in prompt (if structured output fails)
	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			log.Debug().
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("model", model).
				Str("method", "structured_output").
				Msg("[slide_generator] slide LLM call started")
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Fallback: append schema to prompt
			log.Info().
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("model", model).
				Str("method", "schema_in_prompt").
				Msg("[slide_generator] slide using fallback method (schema in prompt)")
			schemaJSON, _ := json.MarshalIndent(schemas.SlideGenResultSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				// Extract JSON from potential markdown code blocks
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Bool("structured_output", useStructuredOutput).
				Msg("[slide_generator] slide LLM call failed")
			continue
		}

		log.Debug().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Bool("structured_output", useStructuredOutput).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] slide LLM response received")

		if err := json.Unmarshal([]byte(result), &slideResult); err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Bool("structured_output", useStructuredOutput).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse slide result")
			continue
		}

		slideMap, ok := slideResult.Slide.(map[string]any)
		if !ok {
			lastErr = fmt.Errorf("slide result is not an object")
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] slide result has invalid type")
			continue
		}

		ensureSlideOrderAndID(slideMap, slideIndex)
		ensureSlideUseComponents(slideMap)

		layoutIDs := templateLayoutIDs(planAndTemplate.Template.Layouts)
		layoutID := slideLayoutID(slideResult.Slide)
		if layoutID == "" || !layoutIDs[layoutID] {
			lastErr = fmt.Errorf("layoutId %q not found in template layouts", layoutID)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("layout_id", layoutID).
				Msg("[slide_generator] slide layout missing from template")
			continue
		}

		assetIDs := extractAssetIDs(slideResult.Requires.Assets)
		datasetIDs := extractDatasetIDs(slideResult.Requires.Datasets)

		if missing := validateChartDatasetRefs(slideMap, datasetIDs); missing != "" {
			lastErr = fmt.Errorf("chart datasetRef %q missing from requires.datasets", missing)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] chart datasetRef missing")
			continue
		}

		if missing := validateImageAssetRefs(slideMap, assetIDs); missing != "" {
			lastErr = fmt.Errorf("image ref %q missing from requires.assets", missing)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] image asset ref missing")
			continue
		}

		if suggestedLayout := strings.TrimSpace(planEntry.SuggestedLayout); suggestedLayout != "" {
			if !layoutIDMatchesSuggestedLayout(layoutID, suggestedLayout, planAndTemplate.Template.Layouts) {
				lastErr = fmt.Errorf("layoutId %q does not match suggestedLayout %q", layoutID, suggestedLayout)
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Str("layout_id", layoutID).
					Str("suggested_layout", suggestedLayout).
					Msg("[slide_generator] slide layout mismatch")
				continue
			}
			if suggestedLayout == "TABLE" && !slideHasElementType(slideResult.Slide, "table") {
				lastErr = fmt.Errorf("missing table element for TABLE layout")
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Msg("[slide_generator] required table element missing")
				continue
			}
		}

		log.Info().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Bool("structured_output", useStructuredOutput).
			Msg("[slide_generator] slide successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse slide %d after retries: %v", slideIndex, lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().
		Int("slide_index", slideIndex).
		Interface("slide", slideResult.Slide).
		Interface("requires", slideResult.Requires).
		Msg("[slide_generator] parsed slide result")

	output := map[string]interface{}{
		"type":        "slide_result",
		"slide_index": slideIndex,
		"slide":       slideResult.Slide,
		"requires":    slideResult.Requires,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Int("slide_index", slideIndex).Msg("[slide_generator] executeSingleSlide completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}


func (e *SlideGeneratorExecutor) buildAccumulatedContext(input agent.ExecutionInput) string {
	log.Debug().
		Int("accumulated_outputs", len(input.AccumulatedOutputs)).
		Int("previous_output_size", len(input.PreviousOutput)).
		Msg("[slide_generator] buildAccumulatedContext started")
	var contextParts []string

	for _, output := range input.AccumulatedOutputs {
		if len(output) > 0 {
			extracted := e.extractContextFromOutput(output)
			if extracted != "" {
				contextParts = append(contextParts, extracted)
			}
		}
	}

	if len(input.PreviousOutput) > 0 {
		extracted := e.extractContextFromOutput(input.PreviousOutput)
		if extracted != "" {
			contextParts = append(contextParts, extracted)
		}
	}

	if len(contextParts) == 0 {
		log.Debug().Msg("[slide_generator] no context available")
		return "[No previous context available]"
	}

	result := strings.Join(contextParts, "\n\n---\n\n")
	log.Debug().
		Int("context_parts", len(contextParts)).
		Int("context_length", len(result)).
		Msg("[slide_generator] buildAccumulatedContext completed")
	return result
}

func (e *SlideGeneratorExecutor) extractContextFromOutput(output json.RawMessage) string {
	if len(output) == 0 {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		rawStr := string(output)
		if len(rawStr) > 10000 {
			return rawStr[:10000] + "... [truncated]"
		}
		return rawStr
	}

	if content, ok := data["content"].(string); ok && content != "" {
		return content
	}
	if text, ok := data["text"].(string); ok && text != "" {
		return text
	}

	if toolName, ok := data["tool_name"].(string); ok && toolName != "" {
		if content, ok := data["content"].([]interface{}); ok {
			texts := []string{}
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			if len(texts) > 0 {
				return fmt.Sprintf("[%s result]: %s", toolName, strings.Join(texts, "\n"))
			}
		}
	}

	rawStr := string(output)
	if len(rawStr) > 10000 {
		return rawStr[:10000] + "... [truncated]"
	}
	return rawStr
}
