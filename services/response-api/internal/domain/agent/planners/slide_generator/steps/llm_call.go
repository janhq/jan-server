package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func ExecuteLLMCall(ctx context.Context, deps ExecutorDeps, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
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
	case "plan_only":
		return ExecutePlanOnly(ctx, deps, params, input)
	case "template_only":
		return ExecuteTemplateOnly(ctx, deps, params, input)
	case "assemble_plan_template":
		return ExecuteAssemblePlanTemplate(ctx, deps, params, input)
	case "plan_and_template":
		return ExecutePlanAndTemplate(ctx, deps, params, input)
	case "generate_single_slide":
		return ExecuteSingleSlide(ctx, deps, params, input)
	case "reasoning":
		return ExecuteReasoning(ctx, deps, params, input)
	case "data_bank":
		return ExecuteDataBank(ctx, deps, params, input)
	default:
		return ExecuteReasoning(ctx, deps, params, input)
	}
}
