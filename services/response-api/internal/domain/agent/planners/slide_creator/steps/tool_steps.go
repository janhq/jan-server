package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

func (e *SlideCreatorExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Str("action", string(step.Action)).Msg("[slide_creator] executeToolCall started")
	var params map[string]interface{}
	if err := json.Unmarshal(input.StepParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_creator] failed to parse tool call parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "export_pptx_dom":
		return e.executeExportPPTX(ctx, params, input)
	case "image_search_slide":
		return e.executeSlideImageSearch(ctx, params, input)
	default:
		return e.executeGenericToolCall(ctx, step, params, input)
	}
}

func (e *SlideCreatorExecutor) executeGenericToolCall(ctx context.Context, step *plan.Step, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	toolName, _ := params["tool"].(string)
	if strings.TrimSpace(toolName) == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_TOOL",
				Message:  "no tool specified",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	log.Debug().Str("step_id", step.ID).Str("tool_name", toolName).Msg("[slide_creator] executeGenericToolCall")
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

	if e.mcpClient == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MCP_CLIENT_MISSING",
				Message:  "mcp client not configured",
				Severity: status.ErrorSeverityFatal,
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
		callReq.UserID = input.PlanContext.UserID
	}

	log.Debug().Str("tool_name", toolName).Interface("arguments", toolArgs).Msg("[slide_creator] calling tool")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		log.Error().Err(err).Str("tool_name", toolName).Msg("[slide_creator] tool call failed")
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

func buildToolArguments(toolName string, params map[string]interface{}, input agent.ExecutionInput, description string) (map[string]interface{}, error) {
	switch toolName {
	case "google_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}
		if query == "" {
			return nil, fmt.Errorf("no search query provided")
		}
		return map[string]interface{}{"q": query}, nil

	case "image_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}
		if query == "" {
			return nil, fmt.Errorf("no image search query provided")
		}
		args := map[string]interface{}{"q": query}
		if num, ok := params["num"].(float64); ok && num > 0 {
			args["num"] = int(num)
		} else {
			args["num"] = 10
		}
		if gl, ok := params["gl"].(string); ok && gl != "" {
			args["gl"] = gl
		}
		if hl, ok := params["hl"].(string); ok && hl != "" {
			args["hl"] = hl
		}
		return args, nil

	case "scrape":
		urls := extractURLsFromPreviousOutput(input.PreviousOutput)
		if len(urls) == 0 {
			if urlParam, ok := params["url"].(string); ok {
				urls = []string{urlParam}
			} else if urlsParam, ok := params["urls"].([]interface{}); ok {
				for _, u := range urlsParam {
					if urlStr, ok := u.(string); ok {
						urls = append(urls, urlStr)
					}
				}
			}
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("no URLs available to scrape from previous search results")
		}
		return map[string]interface{}{"url": urls[0]}, nil

	default:
		toolArgs := make(map[string]interface{})
		for k, v := range params {
			if k != "tool" && k != "description" {
				toolArgs[k] = v
			}
		}
		return toolArgs, nil
	}
}

func extractURLsFromPreviousOutput(previousOutput json.RawMessage) []string {
	if len(previousOutput) == 0 {
		return nil
	}

	var output map[string]interface{}
	if err := json.Unmarshal(previousOutput, &output); err != nil {
		return nil
	}

	var urls []string
	if content, ok := output["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					var textData map[string]interface{}
					if err := json.Unmarshal([]byte(text), &textData); err == nil {
						urls = append(urls, extractURLsFromData(textData)...)
					}
				}
			}
		}
	}

	urls = append(urls, extractURLsFromData(output)...)
	return urls
}

func extractURLsFromData(data map[string]interface{}) []string {
	var urls []string

	if organic, ok := data["organic"].([]interface{}); ok {
		for _, item := range organic {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	if results, ok := data["results"].([]interface{}); ok {
		for _, item := range results {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if url, ok := itemMap["source_url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if url, ok := itemMap["url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	return urls
}

func isNonCriticalToolForSlides(toolName string) bool {
	nonCriticalTools := map[string]bool{
		"google_search": true,
		"image_search":  true,
		"scrape":        true,
	}
	return nonCriticalTools[toolName]
}

func buildSkippedToolResultForSlides(toolName string, reason string, code string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"type":    "tool_result",
		"tool":    toolName,
		"status":  "skipped",
		"reason":  reason,
		"code":    code,
		"skipped": true,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}
