// Package slide_generator contains slide generation planner/executor implementations.
package slide_generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners/slide_generator/schemas"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// SlideGeneratorPlanner creates execution plans for slide generation tasks.
type SlideGeneratorPlanner struct {
	planService     plan.Service
	artifactService artifact.Service
}

// SlideGeneratorConfig holds configuration for slide generation.
type SlideGeneratorConfig struct {
	NumSlides     int    `json:"num_slides"`
	Theme         string `json:"theme"`
	Format        string `json:"format"`         // pptx, pdf
	ResearchDepth string `json:"research_depth"` // minimal, standard, deep
	OptionsCount  int    `json:"options_count"`
}

// DefaultSlideGeneratorConfig returns sensible defaults.
func DefaultSlideGeneratorConfig() SlideGeneratorConfig {
	return SlideGeneratorConfig{
		NumSlides:     10,
		Theme:         "modern",
		Format:        "pptx",
		ResearchDepth: "standard",
		OptionsCount:  1,
	}
}

// NewSlideGeneratorPlanner creates a new slide generator planner.
func NewSlideGeneratorPlanner(planService plan.Service, artifactService artifact.Service) *SlideGeneratorPlanner {
	return &SlideGeneratorPlanner{
		planService:     planService,
		artifactService: artifactService,
	}
}

// Name returns the planner's unique identifier.
func (p *SlideGeneratorPlanner) Name() string {
	return string(plan.AgentTypeSlideGenerator)
}

// CanHandle determines if this planner can handle the given request.
func (p *SlideGeneratorPlanner) CanHandle(ctx context.Context, request *agent.PlanRequest) bool {
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
	return agentTypeStr == string(plan.AgentTypeSlideGenerator)
}

// CreatePlan analyzes the request and creates an execution plan for slide generation.
func (p *SlideGeneratorPlanner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[slide_generator] CreatePlan started")
	config := p.parseConfig(request)
	log.Debug().Interface("config", config).Msg("[slide_generator] parsed config")
	estimatedSteps := p.calculateEstimatedSteps(config)
	log.Debug().Int("estimated_steps", estimatedSteps).Msg("[slide_generator] calculated estimated steps")

	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeSlideGenerator,
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
		log.Debug().Str("research_depth", config.ResearchDepth).Msg("[slide_generator] creating research task")
		researchTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeResearch,
			Title:       "Research",
			Description: strPtr("Gather information and context for the topic of the presentation"),
		})
		if err != nil {
			return nil, err
		}
		log.Debug().Str("task_id", researchTask.ID).Msg("[slide_generator] research task created")

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

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_generator] creating outline task")
	outlineTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeValidation,
		Title:       "Outline",
		Description: strPtr("Create presentation outline and structure"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", outlineTask.ID).Msg("[slide_generator] outline task created")

	outlineParams, _ := json.Marshal(map[string]interface{}{
		"action":      "reasoning",
		"description": "Plan slide structure, key messages, and flow",
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

	// Add image search step after outline reasoning to find relevant images for slides
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

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_generator] creating data bank task")
	dataBankTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Data Bank",
		Description: strPtr("Extract structured facts and datasets from research"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", dataBankTask.ID).Msg("[slide_generator] data bank task created")

	dataBankParams, _ := json.Marshal(map[string]interface{}{
		"action":      "data_bank",
		"description": "Extract facts and datasets using DataBankSchema",
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

	log.Debug().Int("task_sequence", taskSequence).Msg("[slide_generator] creating plan & template task")
	plannerTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Plan & Template",
		Description: strPtr("Create presentation plan and DeckSpec template structure"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", plannerTask.ID).Msg("[slide_generator] plan & template task created")

	plannerParams, _ := json.Marshal(map[string]interface{}{
		"action":      "plan_and_template",
		"description": "Generate slide plan and template using PlanAndTemplateSchema",
		"schema":      schemas.PlanAndTemplateSchema,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
		},
	})
	_, err = p.planService.CreateStep(ctx, plannerTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Plan & Template",
		Description: strPtr("Generate slide plan and template structure"),
		InputParams: plannerParams,
		MaxRetries:  5,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	log.Debug().Int("task_sequence", taskSequence).Int("num_slides", config.NumSlides).Msg("[slide_generator] creating slide generation task")
	slideGenTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Generate Slides",
		Description: strPtr(fmt.Sprintf("Generate %d slides in parallel", config.NumSlides)),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", slideGenTask.ID).Int("num_slides", config.NumSlides).Msg("[slide_generator] slide generation task created")

	for i := 1; i <= config.NumSlides; i++ {
		log.Debug().Int("slide_index", i).Msg("[slide_generator] creating step for slide")
		slideParams, _ := json.Marshal(map[string]interface{}{
			"action":      "generate_single_slide",
			"description": fmt.Sprintf("Generate slide %d content", i),
			"slide_index": i,
			"schema":      schemas.SlideGenResultSchema,
			"parallel":    true,
		})
		_, err = p.planService.CreateStep(ctx, slideGenTask.ID, plan.CreateStepParams{
			Sequence:    i,
			Action:      plan.ActionTypeLLMCall,
			Title:       fmt.Sprintf("Generate Slide %d", i),
			Description: strPtr(fmt.Sprintf("Generate slide %d content", i)),
			InputParams: slideParams,
			MaxRetries:  3,
		})
		if err != nil {
			return nil, err
		}
	}

	taskSequence++

	finalTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Generate Presentation",
		Description: strPtr("Render slide deck and store artifact"),
	})
	if err != nil {
		return nil, err
	}

	specPath := fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", request.ResponseID)
	uploadParams, _ := json.Marshal(map[string]interface{}{
		"action":      "upload_slide_spec",
		"description": "Upload slide JSON to sandbox",
		"tool":        "aio_code_execute",
		"language":    "python",
		"target_path": specPath,
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeToolCall,
		Title:       "Upload Slide Spec",
		Description: strPtr("Upload slide JSON to sandbox"),
		InputParams: uploadParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	outputPath := fmt.Sprintf("/home/gem/slide_%s.pptx", request.ResponseID)
	renderParams, _ := json.Marshal(map[string]interface{}{
		"action":      "render_deck",
		"description": "Render PPTX from slide JSON",
		"tool":        "aio_code_execute",
		"language":    "python",
		"output_path": outputPath,
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeToolCall,
		Title:       "Render Deck",
		Description: strPtr("Render PPTX from slide JSON"),
		InputParams: renderParams,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_artifact",
		"description":   "Store presentation as downloadable artifact",
		"artifact_type": "slides",
		"config": map[string]interface{}{
			"format":           config.Format,
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    3,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Store Artifact",
		Description: strPtr("Store presentation as downloadable artifact"),
		InputParams: artifactParams,
		MaxRetries:  3,
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
		Msg("created slide generation plan")

	return result, nil
}

// parseConfig extracts slide configuration from the request metadata.
func (p *SlideGeneratorPlanner) parseConfig(request *agent.PlanRequest) SlideGeneratorConfig {
	config := DefaultSlideGeneratorConfig()

	if request.Metadata == nil {
		return config
	}

	applySlideConfigFromMap(&config, request.Metadata)

	if options, ok := request.Metadata["options"].(map[string]interface{}); ok {
		applySlideConfigFromMap(&config, options)
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

	log.Debug().
		Int("num_slides", config.NumSlides).
		Str("theme", config.Theme).
		Str("format", config.Format).
		Str("research_depth", config.ResearchDepth).
		Int("options_count", config.OptionsCount).
		Msg("[slide_generator] parsed plan config")

	return config
}

func applySlideConfigFromMap(config *SlideGeneratorConfig, values map[string]interface{}) {
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

// calculateEstimatedSteps returns the expected number of steps based on configuration.
func (p *SlideGeneratorPlanner) calculateEstimatedSteps(config SlideGeneratorConfig) int {
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
	steps += 1 // plan & template
	steps += config.NumSlides
	steps += 1 // upload spec
	steps += 1 // render
	steps += 1 // artifact

	log.Debug().
		Int("num_slides", config.NumSlides).
		Str("research_depth", config.ResearchDepth).
		Int("estimated_steps", steps).
		Msg("[slide_generator] estimated steps calculated")

	return steps
}

// Verify interface compliance at compile time
var _ agent.Planner = (*SlideGeneratorPlanner)(nil)

func strPtr(s string) *string {
	return &s
}
