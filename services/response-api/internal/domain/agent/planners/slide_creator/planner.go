// Package slide_creator contains slide creator planner/executor implementations.
package slide_creator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_creator/schemas"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// SlideCreatorPlanner creates execution plans for slide creation tasks (HTML -> PPTX flow).
type SlideCreatorPlanner struct {
	planService     plan.Service
	artifactService artifact.Service
}

// SlideCreatorConfig holds configuration for slide creation.
type SlideCreatorConfig struct {
	NumSlides       int    `json:"num_slides"`
	Theme           string `json:"theme"`
	ColorScheme     string `json:"color_scheme"`
	Style           string `json:"style"`
	Format          string `json:"format"`         // pptx
	ResearchDepth   string `json:"research_depth"` // minimal, standard, deep
	OptionsCount    int    `json:"options_count"`
	TemplateDir     string `json:"template_dir"`
	TemplateCatalog string `json:"template_catalog"`
	TemplateID      int    `json:"template_id"`
	Tone            string `json:"tone"`
	BodyMode        string `json:"body_mode"` // template|llm
	Debug           bool   `json:"debug"`
}

// DefaultSlideCreatorConfig returns sensible defaults.
func DefaultSlideCreatorConfig() SlideCreatorConfig {
	return SlideCreatorConfig{
		NumSlides:     10,
		Theme:         "modern",
		Format:        "pptx",
		ResearchDepth: "standard",
		OptionsCount:  1,
		BodyMode:      "template",
	}
}

// NewSlideCreatorPlanner creates a new slide creator planner.
func NewSlideCreatorPlanner(planService plan.Service, artifactService artifact.Service) *SlideCreatorPlanner {
	return &SlideCreatorPlanner{
		planService:     planService,
		artifactService: artifactService,
	}
}

// Name returns the planner's unique identifier.
func (p *SlideCreatorPlanner) Name() string {
	return string(plan.AgentTypeSlideCreator)
}

// CanHandle determines if this planner can handle the given request.
func (p *SlideCreatorPlanner) CanHandle(ctx context.Context, request *agent.PlanRequest) bool {
	if request.Metadata == nil {
		return false
	}
	agentType, ok := request.Metadata["agent_type"]
	if !ok {
		return false
	}
	agentTypeStr, ok := agentType.(string)
	if !ok {
		return false
	}
	return agentTypeStr == string(plan.AgentTypeSlideCreator)
}

// CreatePlan analyzes the request and creates an execution plan for slide creation.
func (p *SlideCreatorPlanner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[slide_creator] CreatePlan started")
	config := p.parseConfig(request)
	log.Debug().Interface("config", config).Msg("[slide_creator] parsed config")
	estimatedSteps := p.calculateEstimatedSteps(config)
	log.Debug().Int("estimated_steps", estimatedSteps).Msg("[slide_creator] calculated estimated steps")

	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeSlideCreator,
		EstimatedSteps: estimatedSteps,
		Config: &plan.PlanConfig{
			MaxRetries:        5,
			TimeoutPerStep:    300000000000, // 5 minutes in nanoseconds
			EnableFallback:    true,
			UserApproval:      config.OptionsCount > 1,
			StreamProgress:    true,
			ArtifactRetention: "session",
		},
	})
	if err != nil {
		return nil, err
	}

	taskSequence := 1

	if config.OptionsCount > 1 {
		selectionTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeValidation,
			Title:       "User Selection",
			Description: strPtr("Wait for user to select a presentation option"),
		})
		if err != nil {
			return nil, err
		}

		selectionParams, _ := json.Marshal(map[string]interface{}{
			"action":         "request_selection",
			"requires_user":  true,
			"prompt":         "Select a presentation option to continue",
			"options_count":  config.OptionsCount,
			"selection_type": "option",
		})
		_, err = p.planService.CreateStep(ctx, selectionTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeLLMCall,
			Title:       "Request User Selection",
			Description: strPtr("Wait for user to select a presentation option"),
			InputParams: selectionParams,
			MaxRetries:  3,
		})
		if err != nil {
			return nil, err
		}

		taskSequence++
	}

	if config.ResearchDepth != "minimal" {
		log.Debug().Str("research_depth", config.ResearchDepth).Msg("[slide_creator] creating research task")
		researchTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeResearch,
			Title:       "Research",
			Description: strPtr("Collect background and supporting facts for the presentation"),
		})
		if err != nil {
			return nil, err
		}
		log.Debug().Str("task_id", researchTask.ID).Msg("[slide_creator] research task created")

		searchParams1, _ := json.Marshal(map[string]interface{}{
			"tool":        "google_search",
			"description": "Find key ideas for the topic",
			"q":           request.UserMessage,
		})
		_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeToolCall,
			Title:       "Topic Research",
			Description: strPtr("Find key ideas for the topic"),
			InputParams: searchParams1,
			MaxRetries:  3,
		})
		if err != nil {
			return nil, err
		}

		if config.ResearchDepth == "deep" {
			searchParams2, _ := json.Marshal(map[string]interface{}{
				"tool":        "google_search",
				"description": "Find supporting data and examples",
				"q":           request.UserMessage + " examples data statistics",
			})
			_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
				Sequence:    2,
				Action:      plan.ActionTypeToolCall,
				Title:       "Supporting Research",
				Description: strPtr("Find supporting data and examples"),
				InputParams: searchParams2,
				MaxRetries:  3,
			})
			if err != nil {
				return nil, err
			}

			scrapeParams, _ := json.Marshal(map[string]interface{}{
				"tool":        "scrape",
				"description": "Capture key details from top sources",
			})
			_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
				Sequence:    3,
				Action:      plan.ActionTypeToolCall,
				Title:       "Capture Source Details",
				Description: strPtr("Capture key details from top sources"),
				InputParams: scrapeParams,
				MaxRetries:  3,
			})
			if err != nil {
				return nil, err
			}
		}

		taskSequence++
	}

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_creator] creating outline task")
	outlineTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeValidation,
		Title:       "Outline & Structure",
		Description: strPtr("Define the storyline and slide flow"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", outlineTask.ID).Msg("[slide_creator] outline task created")

	outlineParams, _ := json.Marshal(map[string]interface{}{
		"action":      "reasoning",
		"description": "Draft the slide outline and flow",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
		},
	})
	_, err = p.planService.CreateStep(ctx, outlineTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Draft Outline",
		Description: strPtr("Draft the slide outline and flow"),
		InputParams: outlineParams,
		MaxRetries:  5,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_creator] creating data bank task")
	dataBankTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Key Facts",
		Description: strPtr("Extract facts, figures, and data for the presentation"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", dataBankTask.ID).Msg("[slide_creator] data bank task created")

	dataBankParams, _ := json.Marshal(map[string]interface{}{
		"action":      "data_bank",
		"description": "Extract facts, figures, and datasets",
		"brief":       request.UserMessage,
		"schema":      schemas.DataBankSchema,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
		},
	})
	_, err = p.planService.CreateStep(ctx, dataBankTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Extract Key Facts",
		Description: strPtr("Extract facts, figures, and datasets"),
		InputParams: dataBankParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_creator] creating slide drafting task")
	htmlTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Slide Drafting",
		Description: strPtr("Select a style, plan slides, and prepare drafts"),
	})
	if err != nil {
		return nil, err
	}

	stepSequence := 1
	selectTemplateParams, _ := json.Marshal(map[string]interface{}{
		"action":      "select_templates",
		"description": "Select a slide style and layout set",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"template_dir":     config.TemplateDir,
			"template_catalog": config.TemplateCatalog,
			"template_id":      config.TemplateID,
			"tone":             config.Tone,
			"color_scheme":     config.ColorScheme,
			"style":            config.Style,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Select Slide Style",
		Description: strPtr("Select a slide style and layout set"),
		InputParams: selectTemplateParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	themeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "deck_theme",
		"description": "Select deck title and theme colors",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"theme":        config.Theme,
			"color_scheme": config.ColorScheme,
			"style":        config.Style,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Select Deck Theme",
		Description: strPtr("Select deck title and theme colors"),
		InputParams: themeParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	for i := 1; i <= config.NumSlides; i++ {
		planParams, _ := json.Marshal(map[string]interface{}{
			"action":      "slide_plan_slide",
			"description": fmt.Sprintf("Draft slide plan for slide %d", i),
			"brief":       request.UserMessage,
			"slide_index": i,
			"max_retries": 3,
			"config": map[string]interface{}{
				"num_slides":    config.NumSlides,
				"theme":         config.Theme,
				"color_scheme":  config.ColorScheme,
				"style":         config.Style,
			},
		})
		_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
			Sequence:    stepSequence,
			Action:      plan.ActionTypeLLMCall,
			Title:       fmt.Sprintf("Draft Slide Plan for Slide %d", i),
			Description: strPtr(fmt.Sprintf("Draft slide plan for slide %d", i)),
			InputParams: planParams,
			MaxRetries:  3,
		})
		if err != nil {
			return nil, err
		}
		stepSequence++

		stepTitle := fmt.Sprintf("Find Images for Slide %d", i)
		stepDescription := fmt.Sprintf("Find images for slide %d", i)
		perSlideNum := 6
		if config.NumSlides == 1 {
			perSlideNum = 4
		}
		imageParams, _ := json.Marshal(map[string]interface{}{
			"action":      "image_search_slide",
			"description": stepDescription,
			"slide_index": i,
			"num":         perSlideNum,
			"color_scheme": config.ColorScheme,
			"style":        config.Style,
		})
		_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
			Sequence:    stepSequence,
			Action:      plan.ActionTypeToolCall,
			Title:       stepTitle,
			Description: strPtr(stepDescription),
			InputParams: imageParams,
			MaxRetries:  2,
		})
		if err != nil {
			return nil, err
		}
		stepSequence++
	}

	mergeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "merge_slide_plans",
		"description": "Combine per-slide plans into a single deck plan",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeTransform,
		Title:       "Merge Slide Plans",
		Description: strPtr("Combine per-slide plans into a single deck plan"),
		InputParams: mergeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	normalizeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "normalize_plan",
		"description": "Refine the plan for layout consistency",
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeTransform,
		Title:       "Refine Slide Plan",
		Description: strPtr("Refine the plan for layout consistency"),
		InputParams: normalizeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	renderParams, _ := json.Marshal(map[string]interface{}{
		"action":      "render_slides",
		"description": "Compose slide drafts from the plan",
		"config": map[string]interface{}{
			"body_mode": config.BodyMode,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeTransform,
		Title:       "Compose Slides",
		Description: strPtr("Compose slide drafts from the plan"),
		InputParams: renderParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	writeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "write_outputs",
		"description": "Save slide drafts and supporting files",
		"config": map[string]interface{}{
			"debug": config.Debug,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeTransform,
		Title:       "Save Draft Files",
		Description: strPtr("Save slide drafts and supporting files"),
		InputParams: writeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}
	stepSequence++

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_html_artifact",
		"description":   "Save the slide draft package",
		"artifact_type": "slides_html",
		"config": map[string]interface{}{
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    stepSequence,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Save Draft Package",
		Description: strPtr("Save the slide draft package"),
		InputParams: artifactParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	exportTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Export Presentation",
		Description: strPtr("Export and save the final presentation"),
	})
	if err != nil {
		return nil, err
	}

	exportParams, _ := json.Marshal(map[string]interface{}{
		"action":      "export_pptx_dom",
		"description": "Export the final presentation file",
		"mode":        "dom",
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeToolCall,
		Title:       "Export Presentation",
		Description: strPtr("Export the final presentation file"),
		InputParams: exportParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	pptxArtifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_pptx_artifact",
		"description":   "Save the presentation for download",
		"artifact_type": "slides",
		"config": map[string]interface{}{
			"format":           config.Format,
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Save Presentation",
		Description: strPtr("Save the presentation for download"),
		InputParams: pptxArtifactParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	imageArtifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_slide_images",
		"description":   "Save slide preview images",
		"artifact_type": "slide_images",
		"config": map[string]interface{}{
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    3,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Save Slide Previews",
		Description: strPtr("Save slide preview images"),
		InputParams: imageArtifactParams,
		MaxRetries:  1,
	})
	if err != nil {
		return nil, err
	}

	planWithDetails, err := p.planService.GetPlanWithDetails(ctx, createdPlan.ID)
	if err != nil {
		return nil, err
	}

	result := &agent.PlanResult{
		Plan:             planWithDetails,
		Tasks:            make([]*plan.Task, len(planWithDetails.Tasks)),
		RequiresApproval: config.OptionsCount > 1,
	}

	for i := range planWithDetails.Tasks {
		result.Tasks[i] = &planWithDetails.Tasks[i]
	}

	log.Info().
		Str("plan_id", createdPlan.ID).
		Str("response_id", request.ResponseID).
		Int("num_slides", config.NumSlides).
		Str("theme", config.Theme).
		Str("color_scheme", config.ColorScheme).
		Str("style", config.Style).
		Str("format", config.Format).
		Str("research_depth", config.ResearchDepth).
		Int("estimated_steps", estimatedSteps).
		Msg("created slide creator plan")

	return result, nil
}

// parseConfig extracts slide configuration from the request metadata.
func (p *SlideCreatorPlanner) parseConfig(request *agent.PlanRequest) SlideCreatorConfig {
	config := DefaultSlideCreatorConfig()

	if request.Metadata == nil {
		return config
	}

	applySlideCreatorConfigFromMap(&config, request.Metadata)

	if options, ok := request.Metadata["options"].(map[string]interface{}); ok {
		applySlideCreatorConfigFromMap(&config, options)
	}

	if config.NumSlides < 1 {
		config.NumSlides = 1
	}
	if config.NumSlides > 50 {
		config.NumSlides = 50
	}
	if config.OptionsCount < 1 {
		config.OptionsCount = 1
	}
	if config.OptionsCount > 5 {
		config.OptionsCount = 5
	}
	if strings.TrimSpace(config.BodyMode) == "" {
		config.BodyMode = "template"
	}

	log.Debug().
		Int("num_slides", config.NumSlides).
		Str("theme", config.Theme).
		Str("color_scheme", config.ColorScheme).
		Str("style", config.Style).
		Str("format", config.Format).
		Str("research_depth", config.ResearchDepth).
		Int("options_count", config.OptionsCount).
		Str("template_dir", config.TemplateDir).
		Int("template_id", config.TemplateID).
		Str("template_catalog", config.TemplateCatalog).
		Str("tone", config.Tone).
		Str("body_mode", config.BodyMode).
		Bool("debug", config.Debug).
		Msg("[slide_creator] parsed plan config")

	return config
}

func applySlideCreatorConfigFromMap(config *SlideCreatorConfig, values map[string]interface{}) {
	if values == nil {
		return
	}
	if numSlides, ok := parseIntFromInterface(values["num_slides"]); ok {
		config.NumSlides = numSlides
	}
	if theme, ok := values["theme"].(string); ok {
		config.Theme = theme
	}
	if colorScheme, ok := values["color_scheme"].(string); ok {
		config.ColorScheme = colorScheme
	}
	if style, ok := values["style"].(string); ok {
		config.Style = style
	}
	if format, ok := values["format"].(string); ok {
		config.Format = format
	}
	if researchDepth, ok := values["research_depth"].(string); ok {
		config.ResearchDepth = researchDepth
	}
	if optionsCount, ok := parseIntFromInterface(values["options_count"]); ok {
		config.OptionsCount = optionsCount
	}
	if templateDir, ok := values["template_dir"].(string); ok {
		config.TemplateDir = templateDir
	}
	if templateCatalog, ok := values["template_catalog"].(string); ok {
		config.TemplateCatalog = templateCatalog
	}
	if templateID, ok := parseIntFromInterface(values["template_id"]); ok {
		config.TemplateID = templateID
	}
	if tone, ok := values["tone"].(string); ok {
		config.Tone = tone
	}
	if bodyMode, ok := values["body_mode"].(string); ok {
		config.BodyMode = bodyMode
	}
	if debug, ok := parseBoolFromInterface(values["debug"]); ok {
		config.Debug = debug
	}
}

func parseIntFromInterface(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n, true
		}
	}
	return 0, false
}

func parseBoolFromInterface(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	}
	return false, false
}

// calculateEstimatedSteps returns the expected number of steps based on configuration.
func (p *SlideCreatorPlanner) calculateEstimatedSteps(config SlideCreatorConfig) int {
	steps := 0

	if config.OptionsCount > 1 {
		steps++
	}

	switch config.ResearchDepth {
	case "minimal":
		steps += 0
	case "standard":
		steps += 1
	case "deep":
		steps += 3
	}

	steps += 1                          // outline
	steps += 1                          // data bank
	steps += 7 + (config.NumSlides * 2) // html generation: template + theme + per-slide plan + per-slide search + merge + normalize + render + write + artifact
	steps += 3                          // pptx export + artifacts

	log.Debug().
		Int("num_slides", config.NumSlides).
		Str("research_depth", config.ResearchDepth).
		Int("estimated_steps", steps).
		Msg("[slide_creator] estimated steps calculated")

	return steps
}

// Verify interface compliance at compile time
var _ agent.Planner = (*SlideCreatorPlanner)(nil)

func strPtr(s string) *string {
	return &s
}
