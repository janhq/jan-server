package steps

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
)

func extractDeckPlanFromOutputs(input agent.ExecutionInput) (*DeckPlan, error) {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		payloadType, _ := payload["type"].(string)
		if payloadType != "slide_plan" && payloadType != "normalized_plan" {
			continue
		}
		if planAny, ok := payload["plan"]; ok {
			raw, _ := json.Marshal(planAny)
			var plan DeckPlan
			if err := json.Unmarshal(raw, &plan); err == nil {
				return &plan, nil
			}
		}
		if content, ok := payload["content"].(string); ok && strings.TrimSpace(content) != "" {
			if plan, err := parseDeckPlan(content); err == nil {
				return &plan, nil
			}
		}
	}
	return nil, fmt.Errorf("plan not found in outputs")
}

func extractNormalizedPlan(input agent.ExecutionInput) (*DeckPlan, error) {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		payloadType, _ := payload["type"].(string)
		if payloadType != "normalized_plan" {
			continue
		}
		if planAny, ok := payload["plan"]; ok {
			raw, _ := json.Marshal(planAny)
			var plan DeckPlan
			if err := json.Unmarshal(raw, &plan); err == nil {
				return &plan, nil
			}
		}
		if content, ok := payload["content"].(string); ok && strings.TrimSpace(content) != "" {
			if plan, err := parseDeckPlan(content); err == nil {
				return &plan, nil
			}
		}
	}

	return extractDeckPlanFromOutputs(input)
}

func extractTemplateSelection(input agent.ExecutionInput) TemplateSelection {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	for i := len(outputs) - 1; i >= 0; i-- {
		var payload TemplateSelection
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if payload.Type == "template_selection" {
			return payload
		}
	}

	return TemplateSelection{
		Type:        "template_selection",
		TemplateDir: "standards",
		Source:      "embedded",
	}
}

func resolveTemplateSources(selection TemplateSelection) (fs.FS, string, fs.FS, string, error) {
	if selection.TemplateDir == "" {
		selection.TemplateDir = "standards"
	}

	if selection.Source == "filesystem" || filepath.IsAbs(selection.TemplateDir) {
		templateFS := os.DirFS(selection.TemplateDir)
		templateDir := "."
		fallbackDir := strings.TrimSpace(selection.FallbackDir)
		if fallbackDir == "" {
			fallbackDir = filepath.Join(filepath.Dir(selection.TemplateDir), "standards")
		}
		if stat, err := os.Stat(fallbackDir); err == nil && stat.IsDir() {
			return templateFS, templateDir, os.DirFS(fallbackDir), ".", nil
		}
		return templateFS, templateDir, embeddedTemplatesRoot(), "standards", nil
	}

	return embeddedTemplatesRoot(), selection.TemplateDir, embeddedTemplatesRoot(), "standards", nil
}

func extractOutputDir(input agent.ExecutionInput) string {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if outDir, ok := payload["output_dir"].(string); ok && strings.TrimSpace(outDir) != "" {
			return outDir
		}
	}
	return ""
}

func outputDirForPlan(input agent.ExecutionInput) string {
	responseID := "slide_creator"
	if input.PlanContext != nil && strings.TrimSpace(input.PlanContext.ResponseID) != "" {
		responseID = input.PlanContext.ResponseID
	}
	return filepath.Join(os.TempDir(), "jan_slide_creator", responseID)
}
