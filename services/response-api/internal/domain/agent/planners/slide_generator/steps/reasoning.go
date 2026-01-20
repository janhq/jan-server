package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func ExecuteReasoning(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	description, _ := params["description"].(string)
	contextData := BuildAccumulatedContext(input)
	prompt := fmt.Sprintf(
		"Analyze and plan the slide structure. %s\n\nResearch findings:\n%s\n\nExtract concrete data for any requested tables (column headers + row entries) and include them in the outline.\nProvide a clear, concise outline for the presentation.\nReturn plain text only.",
		description,
		contextData,
	)

	model := getModelFromContext(input)
	if deps.GenerateWithModel == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	response, err := deps.GenerateWithModel(ctx, prompt, model)
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
