package steps

import (
	"context"
	"encoding/json"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

const perSlideImageSearchDefaultNum = 6

func (e *SlideCreatorExecutor) executeSlideImageSearch(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	slideIndex, _ := parseIntFromInterface(params["slide_index"])
	if slideIndex <= 0 {
		return buildSkippedImageSearchResult(slideIndex, "invalid_slide_index"), nil
	}

	plan, err := extractDeckPlanFromOutputs(input)
	if err != nil {
		return buildSkippedImageSearchResult(slideIndex, "plan_missing"), nil
	}
	arrayIndex := slideIndex - 1
	if arrayIndex < 0 || arrayIndex >= len(plan.Slides) {
		return buildSkippedImageSearchResult(slideIndex, "slide_out_of_range"), nil
	}

	slide := plan.Slides[arrayIndex]
	if slideHasImage(slide) {
		return buildSkippedImageSearchResult(slideIndex, "already_has_image"), nil
	}
	if !slideNeedsImage(slide) {
		return buildSkippedImageSearchResult(slideIndex, "layout_not_image"), nil
	}

	query := buildSlideImageQuery(plan.Title, slide)
	if query == "" {
		return buildSkippedImageSearchResult(slideIndex, "empty_query"), nil
	}

	num := perSlideImageSearchDefaultNum
	if parsed, ok := parseIntFromInterface(params["num"]); ok && parsed > 0 {
		num = parsed
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
		Name:      "image_search",
		Arguments: args,
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
		callReq.UserID = input.PlanContext.UserID
	}

	log.Debug().Int("slide_index", slideIndex).Str("query", query).Msg("[slide_creator] image search per slide")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		log.Warn().Err(err).Int("slide_index", slideIndex).Msg("[slide_creator] image search failed")
		return buildSkippedImageSearchResult(slideIndex, "tool_call_failed"), nil
	}
	if result == nil {
		return buildSkippedImageSearchResult(slideIndex, "tool_empty"), nil
	}
	if result.IsError {
		log.Warn().Int("slide_index", slideIndex).Msg("[slide_creator] image search returned error")
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

func buildSlideImageQuery(deckTitle string, slide SlidePlan) string {
	parts := []string{}
	if title := strings.TrimSpace(deckTitle); title != "" {
		parts = append(parts, title)
	}
	if title := strings.TrimSpace(slide.Title); title != "" {
		parts = append(parts, title)
	}
	if subtitle := strings.TrimSpace(slide.Subtitle); subtitle != "" {
		parts = append(parts, subtitle)
	}
	for _, bullet := range slide.Bullets {
		bullet = strings.TrimSpace(bullet)
		if bullet != "" {
			parts = append(parts, bullet)
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

func slideNeedsImage(slide SlidePlan) bool {
	layout := strings.ToLower(strings.TrimSpace(slide.Layout))
	if layout == "" {
		layout = chooseLayout(slide)
	}
	if strings.Contains(layout, "image") {
		return true
	}
	return layout == "split" || layout == "hero"
}

func slideHasImage(slide SlidePlan) bool {
	for _, img := range slide.Images {
		if strings.TrimSpace(img.Src) != "" {
			return true
		}
	}
	return false
}

func mergeSlideImagesFromSearch(plan DeckPlan, input agent.ExecutionInput) DeckPlan {
	perSlide := collectPerSlideImageAssets(input)
	if len(perSlide) == 0 {
		return plan
	}

	for i := range plan.Slides {
		slideIndex := i + 1
		slide := &plan.Slides[i]
		if slideHasImage(*slide) || !slideNeedsImage(*slide) {
			continue
		}
		assets := perSlide[slideIndex]
		if len(assets) == 0 {
			continue
		}
		image := slideImageFromAsset(assets[0], slide.Title)
		if strings.TrimSpace(image.Src) == "" {
			continue
		}
		slide.Images = []SlideImage{image}
	}

	return plan
}

func collectPerSlideImageAssets(input agent.ExecutionInput) map[int][]map[string]any {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	results := map[int][]map[string]any{}
	for _, output := range outputs {
		if len(output) == 0 {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(output, &payload); err != nil {
			continue
		}
		if payloadType, _ := payload["type"].(string); payloadType != "image_search_slide" {
			continue
		}
		slideIndex, _ := parseIntFromInterface(payload["slide_index"])
		if slideIndex <= 0 {
			continue
		}
		assets := extractImageAssetsFromOutput(output)
		if len(assets) == 0 {
			continue
		}
		results[slideIndex] = assets
	}

	return results
}

func slideImageFromAsset(asset map[string]any, fallbackAlt string) SlideImage {
	src := assetImageURL(asset)
	if strings.TrimSpace(src) == "" {
		return SlideImage{}
	}

	title := strings.TrimSpace(firstString(asset, "title", "altText", "alt"))
	alt := title
	if alt == "" {
		alt = strings.TrimSpace(fallbackAlt)
	}
	image := SlideImage{
		Src: src,
		Alt: alt,
	}
	if title != "" {
		image.Caption = title
	}
	return image
}

func assetImageURL(asset map[string]any) string {
	if asset == nil {
		return ""
	}
	if img, ok := asset["imageUrl"].(string); ok && strings.TrimSpace(img) != "" {
		return img
	}
	if thumb, ok := asset["thumbnailUrl"].(string); ok && strings.TrimSpace(thumb) != "" {
		return thumb
	}
	if source, ok := asset["source"].(map[string]any); ok {
		if url, ok := source["url"].(string); ok && strings.TrimSpace(url) != "" {
			return url
		}
	}
	return ""
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
