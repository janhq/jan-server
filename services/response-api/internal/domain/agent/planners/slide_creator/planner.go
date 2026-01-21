// Package slide_creator contains slide creator planner/executor implementations.
package slide_creator

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
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
			Description: strPtr("Gather information and context for the topic of the presentation"),
		})
		if err != nil {
			return nil, err
		}
		log.Debug().Str("task_id", researchTask.ID).Msg("[slide_creator] research task created")

		searchParams1, _ := json.Marshal(map[string]interface{}{
			"tool":        "google_search",
			"description": "Search for key ideas related to the topic for the presentation",
			"q":           request.UserMessage,
		})
		_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeToolCall,
			Title:       "Primary Research",
			Description: strPtr("Search for key ideas related to the topic for the presentation"),
			InputParams: searchParams1,
			MaxRetries:  3,
		})
		if err != nil {
			return nil, err
		}

		if config.ResearchDepth == "deep" {
			searchParams2, _ := json.Marshal(map[string]interface{}{
				"tool":        "google_search",
				"description": "Secondary search for supporting data and examples",
				"q":           request.UserMessage + " examples data statistics",
			})
			_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
				Sequence:    2,
				Action:      plan.ActionTypeToolCall,
				Title:       "Secondary Research",
				Description: strPtr("Secondary search for supporting data and examples"),
				InputParams: searchParams2,
				MaxRetries:  3,
			})
			if err != nil {
				return nil, err
			}

			scrapeParams, _ := json.Marshal(map[string]interface{}{
				"tool":        "scrape",
				"description": "Extract detailed content from top sources",
			})
			_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
				Sequence:    3,
				Action:      plan.ActionTypeToolCall,
				Title:       "Scrape Sources",
				Description: strPtr("Extract detailed content from top sources"),
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
		Title:       "Outline",
		Description: strPtr("Create presentation outline and structure"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", outlineTask.ID).Msg("[slide_creator] outline task created")

	outlineParams, _ := json.Marshal(map[string]interface{}{
		"action":      "reasoning",
		"description": "Plan slide structure, key messages, and flow",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
		},
	})
	_, err = p.planService.CreateStep(ctx, outlineTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Outline Reasoning",
		Description: strPtr("Plan slide structure, key messages, and flow"),
		InputParams: outlineParams,
		MaxRetries:  5,
	})
	if err != nil {
		return nil, err
	}

	imageSearchParams, _ := json.Marshal(map[string]interface{}{
		"tool":        "image_search",
		"description": "Search for relevant images to illustrate the presentation slides",
		"q":           request.UserMessage,
		"num":         8,
	})
	_, err = p.planService.CreateStep(ctx, outlineTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeToolCall,
		Title:       "Image Search",
		Description: strPtr("Search for relevant images to illustrate the presentation slides"),
		InputParams: imageSearchParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_creator] creating data bank task")
	dataBankTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Data Bank",
		Description: strPtr("Extract structured facts and datasets from research"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", dataBankTask.ID).Msg("[slide_creator] data bank task created")

	dataBankParams, _ := json.Marshal(map[string]interface{}{
		"action":      "data_bank",
		"description": "Extract facts and datasets using DataBankSchema",
		"brief":       request.UserMessage,
		"schema":      schemas.DataBankSchema,
	})
	_, err = p.planService.CreateStep(ctx, dataBankTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Extract Data Bank",
		Description: strPtr("Extract structured facts and datasets from research"),
		InputParams: dataBankParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_creator] creating HTML generation task")
	htmlTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "HTML Slide Generation",
		Description: strPtr("Select templates, generate plan, and render HTML slides"),
	})
	if err != nil {
		return nil, err
	}

	selectTemplateParams, _ := json.Marshal(map[string]interface{}{
		"action":      "select_templates",
		"description": "Select the best HTML templates for the deck",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"template_dir":     config.TemplateDir,
			"template_catalog": config.TemplateCatalog,
			"template_id":      config.TemplateID,
			"tone":             config.Tone,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Template Selection",
		Description: strPtr("Select the best HTML templates for the deck"),
		InputParams: selectTemplateParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	planParams, _ := json.Marshal(map[string]interface{}{
		"action":      "slide_plan",
		"description": "Generate slide plan JSON",
		"brief":       request.UserMessage,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Slide Plan",
		Description: strPtr("Generate slide plan JSON"),
		InputParams: planParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	normalizeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "normalize_plan",
		"description": "Normalize the slide plan for layout safety",
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    3,
		Action:      plan.ActionTypeTransform,
		Title:       "Normalize Plan",
		Description: strPtr("Normalize the slide plan for layout safety"),
		InputParams: normalizeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	renderParams, _ := json.Marshal(map[string]interface{}{
		"action":      "render_slides",
		"description": "Render HTML slides from the normalized plan",
		"config": map[string]interface{}{
			"body_mode": config.BodyMode,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    4,
		Action:      plan.ActionTypeTransform,
		Title:       "Render Slides",
		Description: strPtr("Render HTML slides from the normalized plan"),
		InputParams: renderParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	writeParams, _ := json.Marshal(map[string]interface{}{
		"action":      "write_outputs",
		"description": "Write HTML outputs and metadata",
		"config": map[string]interface{}{
			"debug": config.Debug,
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    5,
		Action:      plan.ActionTypeTransform,
		Title:       "Write Outputs",
		Description: strPtr("Write HTML outputs and metadata"),
		InputParams: writeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_html_artifact",
		"description":   "Store HTML slide bundle as an artifact",
		"artifact_type": "slides_html",
		"config": map[string]interface{}{
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, htmlTask.ID, plan.CreateStepParams{
		Sequence:    6,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Store HTML Artifact",
		Description: strPtr("Store HTML slide bundle as an artifact"),
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
		Title:       "PPTX Export",
		Description: strPtr("Export PPTX and store artifacts"),
	})
	if err != nil {
		return nil, err
	}

	exportParams, _ := json.Marshal(map[string]interface{}{
		"action":      "export_pptx_dom",
		"description": "Export PPTX from HTML slides using dom mode",
		"mode":        "dom",
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeToolCall,
		Title:       "Export PPTX",
		Description: strPtr("Export PPTX from HTML slides using dom mode"),
		InputParams: exportParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	pptxArtifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_pptx_artifact",
		"description":   "Store PPTX as downloadable artifact",
		"artifact_type": "slides",
		"config": map[string]interface{}{
			"format":           config.Format,
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Store PPTX Artifact",
		Description: strPtr("Store PPTX as downloadable artifact"),
		InputParams: pptxArtifactParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	imageArtifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_slide_images",
		"description":   "Store slide images as an artifact",
		"artifact_type": "slide_images",
		"config": map[string]interface{}{
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, exportTask.ID, plan.CreateStepParams{
		Sequence:    3,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Store Slide Images",
		Description: strPtr("Store slide images as an artifact"),
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

	steps += 2 // outline (reasoning + image_search)
	steps += 1 // data bank
	steps += 6 // html generation: select templates, plan, normalize, render, write, artifact
	steps += 3 // pptx export + artifacts

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
