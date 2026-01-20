package steps

import (
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"

	"github.com/rs/zerolog/log"
)

func BuildAccumulatedContext(input agent.ExecutionInput) string {
	log.Debug().
		Int("accumulated_outputs", len(input.AccumulatedOutputs)).
		Int("previous_output_size", len(input.PreviousOutput)).
		Msg("[slide_generator] buildAccumulatedContext started")
	var contextParts []string

	for _, output := range input.AccumulatedOutputs {
		if len(output) > 0 {
			extracted := extractContextFromOutput(output)
			if extracted != "" {
				contextParts = append(contextParts, extracted)
			}
		}
	}

	if len(input.PreviousOutput) > 0 {
		extracted := extractContextFromOutput(input.PreviousOutput)
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

func extractContextFromOutput(output json.RawMessage) string {
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

func limitText(input string, max int) string {
	if max <= 0 || len(input) <= max {
		return input
	}
	if max <= 12 {
		return input[:max]
	}
	return input[:max-12] + "... [trimmed]"
}

// isTruncatedJSON checks if a JSON parsing error is due to truncation
func isTruncatedJSON(err error, payload string) bool {
	if err == nil {
		return false
	}
	// Empty responses are not "truncated" - they're empty
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return false
	}
	if strings.Contains(err.Error(), "unexpected end of JSON input") {
		return true
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return !strings.HasSuffix(trimmed, "}") && !strings.HasSuffix(trimmed, "]")
	}
	return false
}

// isEmptyResponse checks if the LLM returned an empty or whitespace-only response
func isEmptyResponse(payload string) bool {
	return strings.TrimSpace(payload) == ""
}
