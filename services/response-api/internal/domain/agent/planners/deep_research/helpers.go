package deepresearch

import (
	"encoding/json"
	"regexp"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
)

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// mustMarshal marshals to JSON, returning empty object on error.
func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// truncateForLog truncates a byte slice for safe logging.
func truncateForLog(data json.RawMessage, maxLen int) string {
	if len(data) == 0 {
		return ""
	}
	s := string(data)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getErrorType extracts a simple error type from the error text.
func getErrorType(errorText string) string {
	if strings.Contains(errorText, "SyntaxError") {
		return "SyntaxError"
	}
	if strings.Contains(errorText, "ModuleNotFoundError") {
		return "ModuleNotFoundError"
	}
	if strings.Contains(errorText, "ImportError") {
		return "ImportError"
	}
	if strings.Contains(errorText, "NameError") {
		return "NameError"
	}
	if strings.Contains(errorText, "TypeError") {
		return "TypeError"
	}
	if strings.Contains(errorText, "ValueError") {
		return "ValueError"
	}
	if strings.Contains(errorText, "AttributeError") {
		return "AttributeError"
	}
	return "RuntimeError"
}

// isNonCriticalTool returns true if the tool failure should not fail the step.
func isNonCriticalTool(toolName string) bool {
	switch toolName {
	case "google_search", "scrape":
		return true
	default:
		return false
	}
}

// buildSkippedToolResult creates a result for a skipped non-critical tool.
func buildSkippedToolResult(toolName string, reason string, statusCode string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"type":    "tool_result",
		"tool":    toolName,
		"status":  "skipped",
		"reason":  reason,
		"code":    statusCode,
		"skipped": true,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}

// hasErrorInResult checks if the tool result contains an error in its content,
// even if the IsError flag is false.
func hasErrorInResult(result *tool.Result) bool {
	if result == nil || len(result.Content) == 0 {
		return false
	}

	for _, content := range result.Content {
		if content.Type != "text" || content.Text == "" {
			continue
		}

		// Try to parse as JSON to check for error_details
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content.Text), &parsed); err == nil {
			// Check for error_details.error_name
			if errDetails, ok := parsed["error_details"].(map[string]interface{}); ok {
				if errName, ok := errDetails["error_name"].(string); ok && errName != "" {
					return true
				}
			}

			// Check for is_error field
			if isError, ok := parsed["is_error"].(bool); ok && isError {
				return true
			}
		}

		// Check for common Python error patterns in text
		errorPatterns := []string{
			"ModuleNotFoundError:",
			"ImportError:",
			"SyntaxError:",
			"TypeError:",
			"ValueError:",
			"NameError:",
			"AttributeError:",
			"IndexError:",
			"KeyError:",
			"FileNotFoundError:",
			"RuntimeError:",
			"ZeroDivisionError:",
			"RecursionError:",
			"MemoryError:",
			"Traceback (most recent call last)",
		}

		for _, pattern := range errorPatterns {
			if strings.Contains(content.Text, pattern) {
				return true
			}
		}
	}

	return false
}

// extractErrorText extracts the error message from a tool result.
func extractErrorText(result *tool.Result) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// extractCodeBlock extracts Python code from markdown code blocks.
func extractCodeBlock(text string) string {
	// Look for Python code block (```python ... ```)
	pythonBlockRegex := regexp.MustCompile("(?s)```python\\s*\n(.*?)```")
	if matches := pythonBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Look for generic code block (``` ... ```)
	genericBlockRegex := regexp.MustCompile("(?s)```\\s*\n(.*?)```")
	if matches := genericBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If the entire text looks like code (has def/import/class statements), return it
	codeIndicators := []string{"import ", "from ", "def ", "class ", "print(", "if __name__"}
	for _, indicator := range codeIndicators {
		if strings.Contains(text, indicator) {
			return strings.TrimSpace(text)
		}
	}

	return ""
}

// looksLikeCode checks if text appears to be Python code.
func looksLikeCode(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "```") {
		return true
	}
	codePattern := regexp.MustCompile(`(?m)^\s*(def|class|import|from)\b`)
	if codePattern.MatchString(trimmed) {
		return true
	}
	if strings.Contains(trimmed, "if __name__") {
		return true
	}
	if strings.Contains(trimmed, "return ") && strings.Contains(trimmed, "\n") {
		return true
	}
	if strings.Contains(trimmed, "print(") && strings.Contains(trimmed, "\n") {
		return true
	}
	return false
}
