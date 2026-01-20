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

func ExecuteSingleSlide(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
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

	if deps.ExtractPlanAndTemplate == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "EXECUTOR_MISSING",
				Message:  "extractPlanAndTemplate not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	planAndTemplate := deps.ExtractPlanAndTemplate(input)
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

	contextData := limitText(BuildAccumulatedContext(input), 8000)
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
	dataBankText := limitText(deps.CollectDataBankText(input), 3000)

	// Reduce prompt size: avoid sending the full template object. Provide only the
	// locked layout id, its slot ids, and a slim theme digest.
	suggestedLayout := strings.TrimSpace(planEntry.SuggestedLayout)
	slotIDs := extractLayoutSlotIDs(planAndTemplate.Template.Layouts, suggestedLayout)
	componentIDs := extractComponentIDs(planAndTemplate.Template.Components)
	lockedLayoutJSON, _ := json.Marshal(map[string]any{
		"layoutId": suggestedLayout,
		"slotIds":  slotIDs,
	})
	componentIDsJSON, _ := json.Marshal(componentIDs)
	themeDigestJSON, _ := json.Marshal(buildThemeDigest(planAndTemplate.Template.Theme))
	planEntryJSON, _ := json.Marshal(planEntry)

	systemPrompt := slideWriterPrompt(slideIndex)
	userPrompt := fmt.Sprintf(
		"BRIEF:\n%s\n\nLOCKED_LAYOUT (use this exact layoutId + slotIds):\n%s\n\nAVAILABLE_COMPONENT_IDS (useComponents):\n%s\n\nTHEME_DIGEST:\n%s\n\nPLAN ENTRY (slide %d):\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s",
		contextData,
		string(lockedLayoutJSON),
		string(componentIDsJSON),
		string(themeDigestJSON),
		slideIndex,
		string(planEntryJSON),
		string(assetsJSON),
		dataBankText,
	)

	model := getModelFromContext(input)
	log.Debug().
		Int("slide_index", slideIndex).
		Str("model", model).
		Float64("temperature", deps.Temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("user_prompt_length", len(userPrompt)).
		Str("user_prompt_preview", truncateForLogString(userPrompt, 300)).
		Msg("[slide_generator] slide prompt prepared")
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

	schema := prepareSlideSchema(schemas.SlideGenResultSchema, suggestedLayout, slotIDs)

	var lastErr error
	var slideResult schemas.SlideGenResult
	var totalUsage *llm.Usage
	var retryErrors []string

	// All attempts use structured output
	for attempt := 1; attempt <= 5; attempt++ {
		retryPrompt := userPrompt
		if attempt > 1 && len(retryErrors) > 0 {
			retryContext := strings.Join(retryErrors, "\n")
			retryPrompt = fmt.Sprintf("%s\n\nPREVIOUS_ERRORS:\n%s", userPrompt, limitText(retryContext, 1200))
		}
		log.Debug().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Str("model", model).
			Msg("[slide_generator] slide LLM call started")

		llmResult, err := deps.GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx, systemPrompt, retryPrompt, model, schema, intPtr(slideMaxTokens))
		if err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d LLM call failed: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] slide LLM call failed")
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
				Int("slide_index", slideIndex).
				Int("prompt_tokens", llmResult.Usage.PromptTokens).
				Int("completion_tokens", llmResult.Usage.CompletionTokens).
				Int("total_tokens", llmResult.Usage.TotalTokens).
				Msg("[slide_generator] slide LLM token usage")
		}

		result := llmResult.Content
		log.Debug().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] slide LLM response received")

		// Handle empty response
		if isEmptyResponse(result) {
			lastErr = fmt.Errorf("LLM returned empty response for slide %d", slideIndex)
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d empty response", attempt))
			log.Warn().
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] slide LLM returned empty response - may indicate rate limiting or model issue")
			continue
		}

		if err := json.Unmarshal([]byte(result), &slideResult); err != nil {
			lastErr = err
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d parse error: %v", attempt, err))
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse slide result")
			continue
		}

		slideMap, ok := slideResult.Slide.(map[string]any)
		if !ok {
			lastErr = fmt.Errorf("slide result is not an object")
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d invalid slide object", attempt))
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] slide result has invalid type")
			continue
		}

		if deps.EnsureSlideOrderAndID != nil {
			deps.EnsureSlideOrderAndID(slideMap, slideIndex)
		}
		if deps.EnsureSlideUseComponents != nil {
			deps.EnsureSlideUseComponents(slideMap)
		}

		layoutIDs := map[string]bool{}
		if deps.TemplateLayoutIDs != nil {
			layoutIDs = deps.TemplateLayoutIDs(planAndTemplate.Template.Layouts)
		}
		layoutID := ""
		if deps.SlideLayoutID != nil {
			layoutID = deps.SlideLayoutID(slideResult.Slide)
		}
		if layoutID == "" || !layoutIDs[layoutID] {
			lastErr = fmt.Errorf("layoutId %q not found in template layouts", layoutID)
			retryErrors = append(retryErrors, fmt.Sprintf("attempt %d missing layoutId %q", attempt, layoutID))
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("layout_id", layoutID).
				Msg("[slide_generator] slide layout missing from template")
			continue
		}

		assetIDs := map[string]bool{}
		if deps.ExtractAssetIDs != nil {
			assetIDs = deps.ExtractAssetIDs(slideResult.Requires.Assets)
		}
		// P1 fix: Also include global locked assets from CollectImageAssets
		// This allows slides to reference assets from image_search without duplicating into requires.assets
		for _, a := range assets {
			if id, ok := a["id"].(string); ok && id != "" {
				assetIDs[id] = true
			}
		}

		datasetIDs := map[string]bool{}
		if deps.ExtractDatasetIDs != nil {
			datasetIDs = deps.ExtractDatasetIDs(slideResult.Requires.Datasets)
		}
		// P1 fix: Also include global datasets from DataBank
		// This allows slides to reference datasets from data bank without duplicating into requires.datasets
		if deps.CollectDataBankDatasets != nil {
			bankDatasets := deps.CollectDataBankDatasets(input)
			for _, ds := range bankDatasets {
				if dsMap, ok := ds.(map[string]any); ok {
					if id, ok := dsMap["id"].(string); ok && id != "" {
						datasetIDs[id] = true
					}
				}
			}
		}

		if deps.ValidateChartDatasetRefs != nil {
			if missing := deps.ValidateChartDatasetRefs(slideMap, datasetIDs); missing != "" {
				lastErr = fmt.Errorf("chart datasetRef %q missing from requires.datasets", missing)
				retryErrors = append(retryErrors, fmt.Sprintf("attempt %d missing datasetRef %q", attempt, missing))
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Msg("[slide_generator] chart datasetRef missing")
				continue
			}
		}

		if deps.ValidateImageAssetRefs != nil {
			if missing := deps.ValidateImageAssetRefs(slideMap, assetIDs); missing != "" {
				lastErr = fmt.Errorf("image ref %q missing from requires.assets", missing)
				retryErrors = append(retryErrors, fmt.Sprintf("attempt %d missing image ref %q", attempt, missing))
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Msg("[slide_generator] image asset ref missing")
				continue
			}
		}

		if suggestedLayout := strings.TrimSpace(planEntry.SuggestedLayout); suggestedLayout != "" {
			if deps.LayoutIDMatchesSuggestedLayout != nil && !deps.LayoutIDMatchesSuggestedLayout(layoutID, suggestedLayout, planAndTemplate.Template.Layouts) {
				lastErr = fmt.Errorf("layoutId %q does not match suggestedLayout %q", layoutID, suggestedLayout)
				retryErrors = append(retryErrors, fmt.Sprintf("attempt %d layout mismatch: %s vs %s", attempt, layoutID, suggestedLayout))
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Str("layout_id", layoutID).
					Str("suggested_layout", suggestedLayout).
					Msg("[slide_generator] slide layout mismatch")
				continue
			}
			if suggestedLayout == "TABLE" && deps.SlideHasElementType != nil && !deps.SlideHasElementType(slideResult.Slide, "table") {
				lastErr = fmt.Errorf("missing table element for TABLE layout")
				retryErrors = append(retryErrors, fmt.Sprintf("attempt %d missing table element for TABLE layout", attempt))
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Msg("[slide_generator] required table element missing")
				continue
			}
		}

		// P2 fix: Richness score validation - ensure slides have enough content
		if !isContentSparse(layoutID) {
			richScore := calculateRichnessScore(slideMap, layoutID)
			minScore := getMinRichnessScore(layoutID)
			if richScore < minScore {
				lastErr = fmt.Errorf("slide is too sparse (richness score %d, minimum %d required)", richScore, minScore)
				retryErrors = append(retryErrors, fmt.Sprintf("attempt %d slide too sparse: %d < %d", attempt, richScore, minScore))
				log.Warn().
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Str("layout_id", layoutID).
					Int("richness_score", richScore).
					Int("min_score", minScore).
					Msg("[slide_generator] slide content too sparse, retrying")
				continue
			}
		}

		log.Info().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
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
	// Include token usage in output if available
	if totalUsage != nil {
		output["token_usage"] = map[string]int{
			"prompt_tokens":     totalUsage.PromptTokens,
			"completion_tokens": totalUsage.CompletionTokens,
			"total_tokens":      totalUsage.TotalTokens,
		}
		log.Info().
			Int("slide_index", slideIndex).
			Int("total_prompt_tokens", totalUsage.PromptTokens).
			Int("total_completion_tokens", totalUsage.CompletionTokens).
			Int("total_tokens", totalUsage.TotalTokens).
			Msg("[slide_generator] slide total token usage")
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Int("slide_index", slideIndex).Msg("[slide_generator] executeSingleSlide completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

// P2 fix: Richness scoring to prevent sparse slides

// isContentSparse returns true for layout types that are intentionally sparse
func isContentSparse(layoutID string) bool {
	switch layoutID {
	case "TITLE", "SECTION_HEADER", "CLOSING":
		return true
	default:
		return false
	}
}

// calculateRichnessScore calculates a richness score for a slide based on its elements
func calculateRichnessScore(slide map[string]any, layoutID string) int {
	score := 0
	elements, _ := slide["elements"].([]any)

	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		elemType, _ := elem["type"].(string)
		slotID, _ := elem["slotId"].(string)

		// Skip header/footer elements in scoring
		if slotID == "header" || slotID == "footer" {
			continue
		}

		switch elemType {
		case "text":
			text, _ := elem["text"].(map[string]any)
			if text != nil {
				content, _ := text["content"].(string)
				// Score based on content length
				if len(content) > 100 {
					score += 2
				} else if len(content) > 20 {
					score += 1
				}
			}
		case "chart":
			score += 3 // Charts are high-value content
		case "table":
			score += 3 // Tables are high-value content
		case "image":
			score += 2 // Images add visual content
		case "shape":
			score += 1 // Shapes are lower value
		}
	}

	return score
}

// getMinRichnessScore returns the minimum richness score for a layout type
func getMinRichnessScore(layoutID string) int {
	switch layoutID {
	case "DASHBOARD_3KPI_2COL":
		return 6 // KPIs + chart/table
	case "CHART_AND_INSIGHTS", "TABLE_AND_CALLOUTS":
		return 4 // Chart/table + insights
	case "CHART", "TABLE":
		return 3 // Just the main element
	default:
		return 2 // At least some content
	}
}
