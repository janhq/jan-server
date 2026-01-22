package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

const perSlideImageSearchDefaultNum = 6

func ExecuteSlideImageSearch(ctx context.Context, deps ExecutorDeps, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	slideIndex := parseIntParam(params, "slide_index")
	if slideIndex <= 0 {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_SLIDE_INDEX",
				Message:  "slide_index is required",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

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

	plan, err := extractPlanFromOutputs(input)
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
	arrayIndex := slideIndex - 1
	if arrayIndex < 0 || arrayIndex >= len(plan.Slides) {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PLAN_ENTRY_NOT_FOUND",
				Message:  fmt.Sprintf("no plan entry for slide %d", slideIndex),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	entry := plan.Slides[arrayIndex]
	if !shouldSearchImagesForSlide(entry.SuggestedLayout, entry.VisualIdeas) {
		return buildSkippedImageSearchResult(slideIndex, "layout_not_image"), nil
	}

	query := buildImageSearchQuery(plan.DeckTitle, entry)
	if query == "" {
		return buildSkippedImageSearchResult(slideIndex, "empty_query"), nil
	}

	num := parseIntParam(params, "num")
	if num <= 0 {
		num = perSlideImageSearchDefaultNum
	}

	args := map[string]interface{}{
		"q":   query,
		"num": num,
	}
	if gl, ok := params["gl"].(string); ok && strings.TrimSpace(gl) != "" {
		args["gl"] = strings.TrimSpace(gl)
	}
	if hl, ok := params["hl"].(string); ok && strings.TrimSpace(hl) != "" {
		args["hl"] = strings.TrimSpace(hl)
	}

	callReq := tool.CallRequest{
		Name:      "image_search",
		Arguments: args,
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
		callReq.UserID = input.PlanContext.UserID
	}

	log.Debug().Int("slide_index", slideIndex).Str("query", query).Msg("[slide_generator] image search per slide")
	result, err := deps.CallTool(ctx, callReq)
	if err != nil {
		log.Warn().Err(err).Int("slide_index", slideIndex).Msg("[slide_generator] image search failed")
		return buildSkippedImageSearchResult(slideIndex, "tool_call_failed"), nil
	}
	if result == nil {
		return buildSkippedImageSearchResult(slideIndex, "tool_empty"), nil
	}
	if result != nil && result.IsError {
		log.Warn().Int("slide_index", slideIndex).Msg("[slide_generator] image search returned error")
		return buildSkippedImageSearchResult(slideIndex, "tool_error"), nil
	}

	output := map[string]interface{}{
		"type":        "image_search_slide",
		"slide_index": slideIndex,
		"query":       query,
		"tool_name":   "image_search",
		"content":     result.Content,
		"is_error":    false,
	}
	outputBytes, _ := json.Marshal(output)
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func parseIntParam(params map[string]interface{}, key string) int {
	if params == nil {
		return 0
	}
	switch v := params[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	case string:
		if n := strings.TrimSpace(v); n != "" {
			if parsed, err := strconv.Atoi(n); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func shouldSearchImagesForSlide(layout string, visualIdeas []string) bool {
	layout = strings.TrimSpace(strings.ToUpper(layout))
	if strings.Contains(layout, "IMAGE") {
		return true
	}
	for _, idea := range visualIdeas {
		if strings.TrimSpace(idea) != "" {
			return true
		}
	}
	return false
}

func buildImageSearchQuery(deckTitle string, entry schemas.PlanEntry) string {
	parts := []string{}
	if title := strings.TrimSpace(deckTitle); title != "" {
		parts = append(parts, title)
	}
	if title := strings.TrimSpace(entry.Title); title != "" {
		parts = append(parts, title)
	}
	for _, point := range entry.KeyPoints {
		point = strings.TrimSpace(point)
		if point != "" {
			parts = append(parts, point)
		}
		if len(parts) >= 4 {
			break
		}
	}
	for _, idea := range entry.VisualIdeas {
		idea = strings.TrimSpace(idea)
		if idea != "" {
			parts = append(parts, idea)
		}
		if len(parts) >= 6 {
			break
		}
	}
	query := strings.Join(parts, " ")
	query = strings.TrimSpace(query)
	if len(query) > 200 {
		query = query[:200]
	}
	return query
}

func buildSkippedImageSearchResult(slideIndex int, reason string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"type":        "image_search_slide",
		"slide_index": slideIndex,
		"skipped":     true,
		"reason":      reason,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}
