package steps

import (
	"context"
	"encoding/json"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

func ExecuteToolCall(ctx context.Context, deps ExecutorDeps, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
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
	case "image_search_slide":
		return ExecuteSlideImageSearch(ctx, deps, params, input)
	case "upload_slide_spec":
		if deps.ExecuteUploadSlideSpec == nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "EXECUTOR_MISSING",
					Message:  "executeUploadSlideSpec not configured",
					Severity: status.ErrorSeverityFatal,
				},
			}, nil
		}
		return deps.ExecuteUploadSlideSpec(ctx, params, input)
	case "render_deck":
		if deps.ExecuteRenderScript == nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "EXECUTOR_MISSING",
					Message:  "executeRenderScript not configured",
					Severity: status.ErrorSeverityFatal,
				},
			}, nil
		}
		return deps.ExecuteRenderScript(ctx, params, input)
	default:
		return executeGenericToolCall(ctx, deps, step, params, input)
	}
}

func executeGenericToolCall(ctx context.Context, deps ExecutorDeps, step *plan.Step, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
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
	toolArgs, err := buildToolArguments(toolName, params, input, description)
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
	if deps.CallTool == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MCP_CLIENT_MISSING",
				Message:  "mcp client not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	result, err := deps.CallTool(ctx, callReq)
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
