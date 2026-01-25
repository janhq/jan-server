package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

func (e *SlideCreatorExecutor) executeTransform(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(input.StepParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_creator] failed to parse transform parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "normalize_plan":
		return e.executeNormalizePlan(ctx, params, input)
	case "merge_slide_plans":
		return e.executeMergeSlidePlans(ctx, params, input)
	case "render_slides":
		return e.executeRenderSlides(ctx, params, input)
	case "write_outputs":
		return e.executeWriteOutputs(ctx, params, input)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func (e *SlideCreatorExecutor) executeNormalizePlan(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	planData, err := extractDeckPlanFromOutputs(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "PLAN_MISSING", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	config, _ := params["config"].(map[string]interface{})
	numSlides, ok := parseIntFromInterface(config["num_slides"])
	if !ok || numSlides <= 0 {
		numSlides = len(planData.Slides)
	}

	enriched := mergeSlideImagesFromSearch(*planData, input)
	normalized := normalizeDeckPlan(enriched, numSlides)
	applyOutlineFallbacks(&normalized, collectOutlineText(input))
	contentBytes, _ := json.Marshal(normalized)
	output := map[string]interface{}{
		"type":    "normalized_plan",
		"plan":    normalized,
		"content": string(contentBytes),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideCreatorExecutor) executeMergeSlidePlans(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	config, _ := params["config"].(map[string]interface{})
	numSlides, ok := parseIntFromInterface(config["num_slides"])
	if !ok || numSlides <= 0 {
		numSlides = 0
	}
	brief := strings.TrimSpace(stringValue(params, "brief"))
	outlineText := strings.TrimSpace(collectOutlineText(input))

	deckTheme, _ := extractDeckTheme(input)
	title := strings.TrimSpace(deckTheme.Title)
	if title == "" {
		title = fallbackDeckTitle(outlineText, brief)
	}

	theme := normalizeTheme(deckTheme.Theme)
	drafts := collectSlidePlanDrafts(input)

	if numSlides <= 0 {
		numSlides = len(drafts)
	}
	if numSlides <= 0 {
		numSlides = 1
	}

	slides := make([]SlidePlan, 0, numSlides)
	for i := 1; i <= numSlides; i++ {
		if draft, ok := drafts[i]; ok {
			slide := draft.Slide
			slide.ID = i
			slides = append(slides, slide)
			continue
		}
		slides = append(slides, SlidePlan{ID: i, Title: fmt.Sprintf("Slide %d", i)})
	}

	plan := DeckPlan{
		Title:  title,
		Theme:  theme,
		Slides: slides,
	}
	applyOutlineFallbacks(&plan, outlineText)

	contentBytes, _ := json.Marshal(plan)
	output := map[string]interface{}{
		"type":    "slide_plan",
		"plan":    plan,
		"content": string(contentBytes),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideCreatorExecutor) executeRenderSlides(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	planData, err := extractNormalizedPlan(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "PLAN_MISSING", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	selection := extractTemplateSelection(input)
	templateFS, templateDir, fallbackFS, fallbackDir, err := resolveTemplateSources(selection)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "TEMPLATE_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	config, _ := params["config"].(map[string]interface{})
	bodyMode := strings.ToLower(strings.TrimSpace(stringValue(config, "body_mode")))
	if bodyMode == "" {
		bodyMode = "template"
	}

	outDir := outputDirForPlan(input)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "OUTPUT_DIR_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	rendered := make([]map[string]interface{}, 0, len(planData.Slides))
	for i, slide := range planData.Slides {
		layout := slide.Layout
		bodyHTML := ""
		if bodyMode == "llm" {
			b, err := e.generateSlideBody(ctx, input, *planData, slide)
			if err != nil {
				return &agent.ExecutionResult{
					Status: status.StatusFailed,
					Error:  &agent.ExecutionError{Code: "BODY_HTML_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
				}, nil
			}
			bodyHTML = b
			layout = "llm"
		}

		data := buildTemplateData(*planData, slide, i+1, len(planData.Slides), bodyHTML)
		fullHTML, err := renderSlideFromTemplate(layout, data, templateFS, templateDir, fallbackFS, fallbackDir)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error:  &agent.ExecutionError{Code: "RENDER_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
			}, nil
		}
		if err := writeSlideHTML(outDir, slide.ID, fullHTML); err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error:  &agent.ExecutionError{Code: "WRITE_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
			}, nil
		}
		rendered = append(rendered, map[string]interface{}{
			"id":     slide.ID,
			"layout": layout,
			"path":   filepath.Join(outDir, fmt.Sprintf("slide-%d.html", slide.ID)),
		})
	}

	output := map[string]interface{}{
		"type":       "rendered_slides",
		"output_dir": outDir,
		"slides":     rendered,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideCreatorExecutor) executeWriteOutputs(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	planData, err := extractNormalizedPlan(input)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "PLAN_MISSING", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	outDir := extractOutputDir(input)
	if outDir == "" {
		outDir = outputDirForPlan(input)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "OUTPUT_DIR_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	if err := writeIndexHTML(outDir, *planData); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "WRITE_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	config, _ := params["config"].(map[string]interface{})
	debug, _ := parseBoolFromInterface(config["debug"])
	if debug {
		planJSON, _ := json.Marshal(planData)
		_ = writeFile(filepath.Join(outDir, "plan.json"), planJSON)
	}

	charts := buildChartsExport(*planData)
	if len(charts.Slides) > 0 {
		chartsJSON, _ := json.Marshal(charts)
		_ = writeFile(filepath.Join(outDir, "charts.json"), chartsJSON)
	}

	images := ImagesExport{Slides: make([]SlideImages, 0, len(planData.Slides))}
	for _, slide := range planData.Slides {
		if len(slide.Images) == 0 {
			continue
		}
		images.Slides = append(images.Slides, SlideImages{
			ID:     slide.ID,
			Images: slide.Images,
		})
	}
	if len(images.Slides) > 0 {
		imagesJSON, _ := json.Marshal(images)
		_ = writeFile(filepath.Join(outDir, "images.json"), imagesJSON)
	}

	files := []string{"index.html"}
	if debug {
		files = append(files, "plan.json")
	}
	if len(charts.Slides) > 0 {
		files = append(files, "charts.json")
	}
	if len(images.Slides) > 0 {
		files = append(files, "images.json")
	}

	output := map[string]interface{}{
		"type":       "html_outputs",
		"output_dir": outDir,
		"files":      files,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}
