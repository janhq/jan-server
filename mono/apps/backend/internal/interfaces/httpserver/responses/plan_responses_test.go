package responses

import (
	"encoding/json"
	"testing"
)

func TestSanitizePlannedParams_RemovesSchema(t *testing.T) {
	// Sample input with schema field
	input := json.RawMessage(`{
		"action": "generate_single_slide",
		"description": "Generate slide 1 content",
		"slide_index": 1,
		"schema": {
			"type": "object",
			"properties": {
				"large": "data"
			}
		}
	}`)

	result := sanitizePlannedParams(input)

	// Unmarshal result to verify schema is removed
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify schema field is removed
	if _, exists := resultMap["schema"]; exists {
		t.Error("Expected schema field to be removed, but it still exists")
	}

	// Verify other fields are preserved
	if resultMap["action"] != "generate_single_slide" {
		t.Error("Expected action field to be preserved")
	}
	if resultMap["description"] != "Generate slide 1 content" {
		t.Error("Expected description field to be preserved")
	}
	if resultMap["slide_index"] != float64(1) {
		t.Error("Expected slide_index field to be preserved")
	}
}

func TestSanitizePlannedParams_NoSchema(t *testing.T) {
	// Sample input without schema field
	input := json.RawMessage(`{
		"action": "test_action",
		"description": "Test description"
	}`)

	result := sanitizePlannedParams(input)

	// Result should be identical to input
	var inputMap, resultMap map[string]interface{}
	json.Unmarshal(input, &inputMap)
	json.Unmarshal(result, &resultMap)

	if len(resultMap) != len(inputMap) {
		t.Errorf("Expected %d fields, got %d", len(inputMap), len(resultMap))
	}
}

func TestSanitizePlannedParams_EmptyInput(t *testing.T) {
	input := json.RawMessage(``)
	result := sanitizePlannedParams(input)

	if len(result) != 0 {
		t.Error("Expected empty result for empty input")
	}
}

func TestSanitizePlannedParams_InvalidJSON(t *testing.T) {
	input := json.RawMessage(`not valid json`)
	result := sanitizePlannedParams(input)

	// Should return original input when JSON is invalid
	if string(result) != string(input) {
		t.Error("Expected original input to be returned for invalid JSON")
	}
}
