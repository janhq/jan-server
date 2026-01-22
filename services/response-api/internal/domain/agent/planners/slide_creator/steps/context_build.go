package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"

	"github.com/rs/zerolog/log"
)

const (
	defaultContextPartLimit       = 10000
	slidePlanContextPerSlideLimit = 3000
	slidePlanContextMaxTotal      = 12000
	slidePlanContextPartLimit     = 3000
)

type contextBuildOptions struct {
	maxTotalChars     int
	maxPartChars      int
	excludedToolNames map[string]struct{}
	excludedTypes     map[string]struct{}
	excludePayload    func(map[string]interface{}) bool
}

func buildSlidePlanContext(input agent.ExecutionInput, numSlides int) string {
	maxTotal := slidePlanContextPerSlideLimit
	if numSlides > 0 {
		maxTotal = slidePlanContextPerSlideLimit * numSlides
	}
	if maxTotal > slidePlanContextMaxTotal {
		maxTotal = slidePlanContextMaxTotal
	}
	return buildAccumulatedContextWithOptions(input, contextBuildOptions{
		maxTotalChars: maxTotal,
		maxPartChars:  slidePlanContextPartLimit,
		excludedToolNames: toKeySet(
			"google_search",
			"image_search",
			"image_search_slide",
			"scrape",
		),
		excludedTypes: toKeySet(
			"google_search",
			"image_search",
			"image_search_slide",
			"scrape",
			"data_bank",
		),
		excludePayload: excludeOutlinePayload,
	})
}

func buildAccumulatedContext(input agent.ExecutionInput) string {
	return buildAccumulatedContextWithOptions(input, contextBuildOptions{
		maxPartChars: defaultContextPartLimit,
		excludedToolNames: toKeySet(
			"image_search",
			"image_search_slide",
		),
		excludedTypes: toKeySet(
			"image_search",
			"image_search_slide",
		),
	})
}

func buildAccumulatedContextWithOptions(input agent.ExecutionInput, opts contextBuildOptions) string {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		if len(outputs) == 0 || !bytes.Equal(outputs[len(outputs)-1], input.PreviousOutput) {
			outputs = append(outputs, input.PreviousOutput)
		}
	}

	contextParts := []string{}
	for _, output := range outputs {
		if len(output) == 0 {
			continue
		}
		extracted := extractContextFromOutputWithOptions(output, opts)
		if strings.TrimSpace(extracted) == "" {
			continue
		}
		if opts.maxPartChars > 0 {
			extracted = truncateWithSuffix(extracted, opts.maxPartChars)
		}
		contextParts = append(contextParts, extracted)
	}

	if len(contextParts) == 0 {
		return ""
	}
	if opts.maxTotalChars > 0 {
		contextParts = trimContextPartsToLimit(contextParts, opts.maxTotalChars)
	}

	return strings.Join(contextParts, "\n\n---\n\n")
}

func extractContextFromOutputWithOptions(output json.RawMessage, opts contextBuildOptions) string {
	if len(output) == 0 {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return string(output)
	}
	if shouldSkipContextPayload(data, opts) {
		return ""
	}
	if opts.excludePayload != nil && opts.excludePayload(data) {
		return ""
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
					if text, ok := itemMap["text"].(string); ok && text != "" {
						texts = append(texts, text)
					}
				}
			}
			if len(texts) > 0 {
				return fmt.Sprintf("[%s result]: %s", toolName, strings.Join(texts, "\n"))
			}
		}
	}

	return string(output)
}

func shouldSkipContextPayload(data map[string]interface{}, opts contextBuildOptions) bool {
	if data == nil {
		return false
	}
	if len(opts.excludedTypes) > 0 {
		if payloadType, ok := data["type"].(string); ok {
			if _, excluded := opts.excludedTypes[normalizeKey(payloadType)]; excluded {
				return true
			}
		}
	}
	if len(opts.excludedToolNames) > 0 {
		if toolName, ok := data["tool_name"].(string); ok {
			if _, excluded := opts.excludedToolNames[normalizeKey(toolName)]; excluded {
				return true
			}
		}
		if toolName, ok := data["tool"].(string); ok {
			if _, excluded := opts.excludedToolNames[normalizeKey(toolName)]; excluded {
				return true
			}
		}
	}
	return false
}

func trimContextPartsToLimit(parts []string, limit int) []string {
	if limit <= 0 {
		return parts
	}

	total := 0
	kept := make([]string, 0, len(parts))
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		partLen := len(part)
		if total+partLen <= limit {
			kept = append(kept, part)
			total += partLen
			continue
		}

		remaining := limit - total
		if remaining <= 0 {
			break
		}
		truncated := truncateWithSuffix(part, remaining)
		if strings.TrimSpace(truncated) != "" {
			kept = append(kept, truncated)
		}
		break
	}

	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}

	return kept
}

func truncateWithSuffix(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	const suffix = "... [truncated]"
	if limit <= len(suffix) {
		return text[:limit]
	}
	return text[:limit-len(suffix)] + suffix
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func toKeySet(values ...string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := normalizeKey(value)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func collectOutlineText(input agent.ExecutionInput) string {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	for i := len(outputs) - 1; i >= 0; i-- {
		if len(outputs[i]) == 0 {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if !isOutlinePayload(payload) {
			continue
		}
		if content, ok := payload["content"].(string); ok && strings.TrimSpace(content) != "" {
			log.Debug().Int("content_length", len(content)).Msg("[slide_creator] outline text collected")
			return content
		}
	}
	return ""
}

func excludeOutlinePayload(payload map[string]interface{}) bool {
	return isOutlinePayload(payload)
}

func isOutlinePayload(payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}
	payloadType, _ := payload["type"].(string)
	if normalizeKey(payloadType) != "llm_response" {
		return false
	}
	action, _ := payload["action"].(string)
	if normalizeKey(action) == "reasoning" {
		return true
	}
	description, _ := payload["description"].(string)
	return strings.Contains(strings.ToLower(description), "outline")
}
