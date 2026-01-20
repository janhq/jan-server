package steps

import (
	"encoding/json"
	"strings"
)

func cloneSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	raw, _ := json.Marshal(schema)
	var cloned map[string]any
	_ = json.Unmarshal(raw, &cloned)
	return cloned
}

func truncateForLogString(data string, maxLen int) string {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen] + "..."
}

func extractJSONFromResponse(response string) string {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return trimmed
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}

	start = strings.Index(trimmed, "[")
	end = strings.LastIndex(trimmed, "]")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}

	return trimmed
}
