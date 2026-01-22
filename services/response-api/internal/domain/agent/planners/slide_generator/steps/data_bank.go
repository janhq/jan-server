package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

const dataBankImageAssetLimit = 2

func ExecuteDataBank(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeDataBank started")
	contextData := BuildAccumulatedContext(input)
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
	assets := limitImageAssets(deps.CollectImageAssets(input), dataBankImageAssetLimit)
	assetsJSON, _ := json.Marshal(compactImageAssetsForPrompt(assets))

	systemPrompt := dataBankPrompt
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nASSETS AVAILABLE:\n%s", contextData, string(assetsJSON))

	model := getModelFromContext(input)
	if deps.GenerateWithStructuredOutput == nil || deps.GenerateWithSystemPrompt == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := prepareSchema(schemas.DataBankSchema)

	var lastErr error
	var dataBank schemas.DataBank

	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			result, err = deps.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Final fallback (non-structured): avoid embedding the full JSON Schema in the prompt
			// to reduce token overhead and improve compliance.
			shape := `{"facts":[{"claim":"...","value":"...","unit":"...","sourceUrl":"https://...","date":"YYYY-MM-DD"}],"datasets":[{"id":"dataset_id","kind":"bar|line|pie","data":{"labels":["..."],"series":[{"name":"...","values":[1,2,3]}]},"sourceNote":"..."}]}`
			enhancedUserPrompt := fmt.Sprintf(
				"%s\n\nIMPORTANT:\n- Return ONLY a single JSON object (no markdown, no commentary).\n- Match this JSON SHAPE (example placeholders):\n%s\n- If you cannot extract reliable facts/datasets from the provided context, return empty arrays for facts and datasets.",
				userPrompt,
				shape,
			)
			result, err = deps.GenerateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
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
