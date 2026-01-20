// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	sliderenderer "jan-server/services/response-api/assets/slide_renderer"
	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/schemas"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/media"

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
	})
	_, err = p.planService.CreateStep(ctx, outlineTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeToolCall,
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

	return steps
}

// Verify interface compliance at compile time
var _ agent.Planner = (*SlideGeneratorPlanner)(nil)

// SlideGeneratorExecutor executes steps for slide generation plans.
type SlideGeneratorExecutor struct {
	mcpClient          MCPClient
	llmProvider        LLMProvider
	artifactService    artifact.Service
	mediaClient        *media.Client
	skillExecutor      *SkillExecutor
	aioClient          *agent.AIOSandboxClient // Direct AIO sandbox client (bypasses MCP)
	aioBaseURL         string
	rendererScriptPath string
	rendererEnabled    bool
	temperature        float64 // LLM temperature for slide generation (default: 0.2)
}

type llmProviderWithTemperature interface {
	GenerateWithModelWithTemperature(ctx context.Context, prompt string, model string, temperature float64) (string, error)
	GenerateWithSystemPromptWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64) (string, error)
	GenerateWithStructuredOutputWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64) (string, error)
}

// NewSlideGeneratorExecutor creates a new slide generator executor.
func NewSlideGeneratorExecutor(mcpClient MCPClient, llmProvider LLMProvider, artifactService artifact.Service, mediaClient *media.Client, skillExecutor *SkillExecutor, cfg *config.Config) *SlideGeneratorExecutor {
	aioBaseURL := ""
	rendererScriptPath := ""
	rendererEnabled := true
	if cfg != nil {
		aioBaseURL = strings.TrimSpace(cfg.AIOURL)
		rendererScriptPath = strings.TrimSpace(cfg.SlideRendererScript)
		rendererEnabled = cfg.SlideRendererEnabled
	}

	// Initialize direct AIO sandbox client (bypasses unstable MCP layer)
	var aioClient *agent.AIOSandboxClient
	if aioBaseURL != "" {
		aioClient = agent.NewAIOSandboxClient(aioBaseURL, log.Logger)
		log.Info().Str("aio_url", aioBaseURL).Msg("[slide_generator] Initialized direct AIO sandbox client")
	}

	return &SlideGeneratorExecutor{
		mcpClient:          mcpClient,
		llmProvider:        llmProvider,
		artifactService:    artifactService,
		mediaClient:        mediaClient,
		skillExecutor:      skillExecutor,
		aioClient:          aioClient,
		aioBaseURL:         aioBaseURL,
		rendererScriptPath: rendererScriptPath,
		rendererEnabled:    rendererEnabled,
		temperature:        0.2, // Low temperature for deterministic, structured output
	}
}

func (e *SlideGeneratorExecutor) generateWithModel(ctx context.Context, prompt string, model string) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithModelWithTemperature(ctx, prompt, model, e.temperature)
	}
	return e.llmProvider.GenerateWithModel(ctx, prompt, model)
}

func (e *SlideGeneratorExecutor) generateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithSystemPromptWithTemperature(ctx, systemPrompt, userPrompt, model, e.temperature)
	}
	return e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, userPrompt, model)
}

func (e *SlideGeneratorExecutor) generateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error) {
	if provider, ok := e.llmProvider.(llmProviderWithTemperature); ok {
		return provider.GenerateWithStructuredOutputWithTemperature(ctx, systemPrompt, userPrompt, model, schema, e.temperature)
	}
	return e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
}

// CanExecute checks if this executor can handle the given action type.
func (e *SlideGeneratorExecutor) CanExecute(action plan.ActionType) bool {
	switch action {
	case plan.ActionTypeToolCall, plan.ActionTypeLLMCall, plan.ActionTypeSkillExecute, plan.ActionTypeArtifactCreate:
		return true
	default:
		return false
	}
}

// Rollback attempts to undo a step's effects.
func (e *SlideGeneratorExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	if step.Action == plan.ActionTypeArtifactCreate {
		return nil
	}
	return nil
}

// Execute runs a single step and returns the result.
func (e *SlideGeneratorExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Str("action", string(step.Action)).Int("sequence", step.Sequence).Msg("[slide_generator] Execute started")
	switch step.Action {
	case plan.ActionTypeToolCall:
		return e.executeToolCall(ctx, step, input)
	case plan.ActionTypeLLMCall:
		return e.executeLLMCall(ctx, step, input)
	case plan.ActionTypeSkillExecute:
		if e.skillExecutor == nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "SKILL_EXECUTOR_MISSING",
					Message:  "skill executor not configured",
					Severity: status.ErrorSeverityFatal,
				},
			}, nil
		}
		return e.skillExecutor.Execute(ctx, step, input)
	case plan.ActionTypeArtifactCreate:
		return e.executeArtifactCreation(ctx, step, input)
	default:
		return &agent.ExecutionResult{Status: status.StatusCompleted}, nil
	}
}

func (e *SlideGeneratorExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeToolCall started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse step parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse step parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "upload_slide_spec":
		return e.executeUploadSlideSpec(ctx, params, input)
	case "render_deck":
		return e.executeRenderScript(ctx, params, input)
	default:
		return e.executeGenericToolCall(ctx, step, params, input)
	}
}

func (e *SlideGeneratorExecutor) executeGenericToolCall(ctx context.Context, step *plan.Step, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	toolName, _ := params["tool"].(string)
	log.Debug().Str("tool_name", toolName).Interface("params", params).Msg("[slide_generator] executeGenericToolCall started")
	if toolName == "" {
		log.Error().Msg("[slide_generator] no tool specified")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_TOOL",
				Message:  "no tool specified",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	description, _ := params["description"].(string)
	toolArgs, err := e.buildToolArguments(toolName, params, input, description)
	if err != nil {
		if isNonCriticalToolForSlides(toolName) {
			return buildSkippedToolResultForSlides(toolName, err.Error(), "invalid_arguments"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_ARGUMENTS",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	callReq := tool.CallRequest{
		Name:      toolName,
		Arguments: toolArgs,
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	log.Debug().Str("tool_name", toolName).Interface("arguments", toolArgs).Msg("[slide_generator] calling tool")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		log.Error().Err(err).Str("tool_name", toolName).Msg("[slide_generator] tool call failed")
		if isNonCriticalToolForSlides(toolName) {
			return buildSkippedToolResultForSlides(toolName, err.Error(), "tool_call_failed"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "TOOL_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if result != nil && result.IsError && isNonCriticalToolForSlides(toolName) {
		return buildSkippedToolResultForSlides(toolName, "tool reported error", "tool_error"), nil
	}

	outputBytes, _ := json.Marshal(result)
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeLLMCall started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse LLM call parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	if requiresUser, ok := params["requires_user"].(bool); ok && requiresUser {
		prompt, _ := params["prompt"].(string)
		optionsCount := 0
		if rawCount, ok := params["options_count"].(float64); ok {
			optionsCount = int(rawCount)
		}

		options := make([]string, 0, optionsCount)
		for i := 0; i < optionsCount; i++ {
			options = append(options, fmt.Sprintf("option_%d", i+1))
		}

		outputBytes, _ := json.Marshal(map[string]interface{}{
			"status":        "waiting_for_user",
			"prompt":        prompt,
			"options":       options,
			"options_count": optionsCount,
		})

		return &agent.ExecutionResult{
			Status:       status.StatusCompleted,
			Output:       outputBytes,
			RequiresUser: true,
			UserPrompt:   &prompt,
		}, nil
	}

	action, _ := params["action"].(string)
	switch action {
	case "plan_and_template":
		return e.executePlanAndTemplate(ctx, params, input)
	case "generate_single_slide":
		return e.executeSingleSlide(ctx, params, input)
	case "reasoning":
		return e.executeReasoning(ctx, params, input)
	case "data_bank":
		return e.executeDataBank(ctx, params, input)
	default:
		return e.executeReasoning(ctx, params, input)
	}
}

func (e *SlideGeneratorExecutor) executeReasoning(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	description, _ := params["description"].(string)
	contextData := e.buildAccumulatedContext(input)
	prompt := fmt.Sprintf(
		"Analyze and plan the slide structure. %s\n\nResearch findings:\n%s\n\nExtract concrete data for any requested tables (column headers + row entries) and include them in the outline.\nProvide a clear, concise outline for the presentation.\nReturn plain text only.",
		description,
		contextData,
	)

	model := e.getModelFromContext(input)
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	response, err := e.generateWithModel(ctx, prompt, model)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_CALL_FAILED",
				Message:  fmt.Sprintf("LLM generation failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	output := map[string]interface{}{
		"type":        "llm_response",
		"action":      "reasoning",
		"description": description,
		"content":     response,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeDataBank(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeDataBank started")
	contextData := e.buildAccumulatedContext(input)
	assets := e.collectImageAssets(input)
	assetsJSON, _ := json.Marshal(assets)

	systemPrompt := dataBankPrompt
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nASSETS AVAILABLE:\n%s", contextData, string(assetsJSON))

	model := e.getModelFromContext(input)
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := cloneSchema(schemas.DataBankSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var dataBank schemas.DataBank

	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			schemaJSON, _ := json.MarshalIndent(schemas.DataBankSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Msg("[slide_generator] data_bank LLM call failed")
			continue
		}

		if err := json.Unmarshal([]byte(result), &dataBank); err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Msg("[slide_generator] failed to parse data_bank result")
			continue
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse data bank after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentBytes, _ := json.Marshal(dataBank)
	output := map[string]interface{}{
		"type":    "data_bank",
		"data":    dataBank,
		"content": string(contentBytes),
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executeDataBank completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executePlanAndTemplate(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executePlanAndTemplate started")
	contextData := e.buildAccumulatedContext(input)
	log.Debug().Int("context_length", len(contextData)).Msg("[slide_generator] built accumulated context")

	config, _ := params["config"].(map[string]interface{})
	numSlides := 10
	if n, ok := config["num_slides"].(float64); ok {
		numSlides = int(n)
	}
	theme, _ := config["theme"].(string)

	assets := e.collectImageAssets(input)
	assetsJSON, _ := json.Marshal(assets)
	dataBankText := e.collectDataBankText(input)
	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerAndTemplatePrompt)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s", contextData, numSlides, theme, string(assetsJSON), dataBankText)

	model := e.getModelFromContext(input)
	log.Debug().
		Str("model", model).
		Float64("temperature", e.temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("user_prompt_length", len(userPrompt)).
		Str("user_prompt_preview", truncateForLogString(userPrompt, 300)).
		Msg("[slide_generator] plan_and_template prompt prepared")
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := cloneSchema(schemas.PlanAndTemplateSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var planAndTemplate schemas.PlanAndTemplate

	// Enhanced retry strategy:
	// Attempt 1-2: Use response_format with structured output (OpenAI enforced schema)
	// Attempt 3: Fallback to schema in prompt (if structured output fails)
	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			log.Debug().
				Int("attempt", attempt).
				Str("model", model).
				Str("method", "structured_output").
				Msg("[slide_generator] plan_and_template LLM call started")
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Fallback: append schema to prompt
			log.Info().
				Int("attempt", attempt).
				Str("model", model).
				Str("method", "schema_in_prompt").
				Msg("[slide_generator] plan_and_template using fallback method (schema in prompt)")
			schemaJSON, _ := json.MarshalIndent(schemas.PlanAndTemplateSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				// Extract JSON from potential markdown code blocks
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Bool("structured_output", useStructuredOutput).
				Msg("[slide_generator] plan_and_template LLM call failed")
			continue
		}

		log.Debug().
			Int("attempt", attempt).
			Bool("structured_output", useStructuredOutput).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] plan_and_template LLM response received")

		if err := json.Unmarshal([]byte(result), &planAndTemplate); err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Bool("structured_output", useStructuredOutput).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse plan+template")
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Bool("structured_output", useStructuredOutput).
			Msg("[slide_generator] plan_and_template successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse plan+template after retries: %v", lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().
		Int("slide_count", len(planAndTemplate.Plan.Slides)).
		Int("recommended_slides", planAndTemplate.Plan.RecommendedSlideCount).
		Interface("template", planAndTemplate.Template).
		Msg("[slide_generator] parsed plan and template")

	normalizePlanIndices(&planAndTemplate.Plan)
	normalizeTemplateComponents(&planAndTemplate.Template)
	normalizeTemplateLayouts(&planAndTemplate.Plan, &planAndTemplate.Template)

	output := map[string]interface{}{
		"type":               "plan_and_template",
		"plan":               planAndTemplate.Plan,
		"template":           planAndTemplate.Template,
		"recommended_slides": planAndTemplate.Plan.RecommendedSlideCount,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executePlanAndTemplate completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeSingleSlide(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	slideIndex := 0
	if raw, ok := params["slide_index"].(float64); ok {
		slideIndex = int(raw)
	}
	log.Debug().Int("slide_index", slideIndex).Msg("[slide_generator] executeSingleSlide started")
	if slideIndex <= 0 {
		log.Error().Int("slide_index", slideIndex).Msg("[slide_generator] invalid slide index")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "INVALID_SLIDE_INDEX",
				Message:  "slide_index is required",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	planAndTemplate := e.extractPlanAndTemplate(input)
	if planAndTemplate == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_CONTEXT",
				Message:  "Plan and template not found in previous outputs",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Use 1-based slideIndex to access 0-based array
	// (LLM may use 0-based or 1-based Index field, so use array position instead)
	arrayIndex := slideIndex - 1
	if arrayIndex < 0 || arrayIndex >= len(planAndTemplate.Plan.Slides) {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PLAN_ENTRY_NOT_FOUND",
				Message:  fmt.Sprintf("No plan entry for slide %d (array index %d out of range, plan has %d slides)", slideIndex, arrayIndex, len(planAndTemplate.Plan.Slides)),
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	planEntry := &planAndTemplate.Plan.Slides[arrayIndex]

	contextData := e.buildAccumulatedContext(input)
	assets := e.collectImageAssets(input)
	assetsJSON, _ := json.Marshal(assets)
	dataBankText := e.collectDataBankText(input)
	templateJSON, _ := json.Marshal(planAndTemplate.Template)
	planEntryJSON, _ := json.Marshal(planEntry)
	themeJSON, _ := json.Marshal(planAndTemplate.Template.Theme)

	systemPrompt := slideWriterPrompt(slideIndex)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nLOCKED TEMPLATE:\n%s\n\nTHEME:\n%s\n\nPLAN ENTRY (slide %d):\n%s\n\nASSETS AVAILABLE:\n%s\n\nDATA BANK:\n%s", contextData, string(templateJSON), string(themeJSON), slideIndex, string(planEntryJSON), string(assetsJSON), dataBankText)

	model := e.getModelFromContext(input)
	log.Debug().
		Int("slide_index", slideIndex).
		Str("model", model).
		Float64("temperature", e.temperature).
		Int("system_prompt_length", len(systemPrompt)).
		Int("user_prompt_length", len(userPrompt)).
		Str("user_prompt_preview", truncateForLogString(userPrompt, 300)).
		Msg("[slide_generator] slide prompt prepared")
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "LLM_PROVIDER_MISSING",
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	schema := cloneSchema(schemas.SlideGenResultSchema)
	schemas.NormalizeSchemaForStructuredOutput(schema)

	var lastErr error
	var slideResult schemas.SlideGenResult

	// Enhanced retry strategy:
	// Attempt 1-2: Use response_format with structured output (OpenAI enforced schema)
	// Attempt 3: Fallback to schema in prompt (if structured output fails)
	for attempt := 1; attempt <= 3; attempt++ {
		useStructuredOutput := attempt <= 2
		var result string
		var err error

		if useStructuredOutput {
			log.Debug().
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("model", model).
				Str("method", "structured_output").
				Msg("[slide_generator] slide LLM call started")
			result, err = e.generateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Fallback: append schema to prompt
			log.Info().
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("model", model).
				Str("method", "schema_in_prompt").
				Msg("[slide_generator] slide using fallback method (schema in prompt)")
			schemaJSON, _ := json.MarshalIndent(schemas.SlideGenResultSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.generateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
			if err == nil {
				// Extract JSON from potential markdown code blocks
				result = extractJSONFromResponse(result)
			}
		}

		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Bool("structured_output", useStructuredOutput).
				Msg("[slide_generator] slide LLM call failed")
			continue
		}

		log.Debug().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Bool("structured_output", useStructuredOutput).
			Int("response_length", len(result)).
			Str("response_preview", truncateForLogString(result, 300)).
			Msg("[slide_generator] slide LLM response received")

		if err := json.Unmarshal([]byte(result), &slideResult); err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Bool("structured_output", useStructuredOutput).
				Str("response_preview", truncateForLogString(result, 300)).
				Msg("[slide_generator] failed to parse slide result")
			continue
		}

		slideMap, ok := slideResult.Slide.(map[string]any)
		if !ok {
			lastErr = fmt.Errorf("slide result is not an object")
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] slide result has invalid type")
			continue
		}

		ensureSlideOrderAndID(slideMap, slideIndex)
		ensureSlideUseComponents(slideMap)

		layoutIDs := templateLayoutIDs(planAndTemplate.Template.Layouts)
		layoutID := slideLayoutID(slideResult.Slide)
		if layoutID == "" || !layoutIDs[layoutID] {
			lastErr = fmt.Errorf("layoutId %q not found in template layouts", layoutID)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Str("layout_id", layoutID).
				Msg("[slide_generator] slide layout missing from template")
			continue
		}

		assetIDs := extractAssetIDs(slideResult.Requires.Assets)
		datasetIDs := extractDatasetIDs(slideResult.Requires.Datasets)

		if missing := validateChartDatasetRefs(slideMap, datasetIDs); missing != "" {
			lastErr = fmt.Errorf("chart datasetRef %q missing from requires.datasets", missing)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] chart datasetRef missing")
			continue
		}

		if missing := validateImageAssetRefs(slideMap, assetIDs); missing != "" {
			lastErr = fmt.Errorf("image ref %q missing from requires.assets", missing)
			log.Warn().
				Err(lastErr).
				Int("attempt", attempt).
				Int("slide_index", slideIndex).
				Msg("[slide_generator] image asset ref missing")
			continue
		}

		if suggestedLayout := strings.TrimSpace(planEntry.SuggestedLayout); suggestedLayout != "" {
			if !layoutIDMatchesSuggestedLayout(layoutID, suggestedLayout, planAndTemplate.Template.Layouts) {
				lastErr = fmt.Errorf("layoutId %q does not match suggestedLayout %q", layoutID, suggestedLayout)
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Str("layout_id", layoutID).
					Str("suggested_layout", suggestedLayout).
					Msg("[slide_generator] slide layout mismatch")
				continue
			}
			if suggestedLayout == "TABLE" && !slideHasElementType(slideResult.Slide, "table") {
				lastErr = fmt.Errorf("missing table element for TABLE layout")
				log.Warn().
					Err(lastErr).
					Int("attempt", attempt).
					Int("slide_index", slideIndex).
					Msg("[slide_generator] required table element missing")
				continue
			}
		}

		log.Info().
			Int("attempt", attempt).
			Int("slide_index", slideIndex).
			Bool("structured_output", useStructuredOutput).
			Msg("[slide_generator] slide successfully parsed")
		lastErr = nil
		break
	}

	if lastErr != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  fmt.Sprintf("Failed to parse slide %d after retries: %v", slideIndex, lastErr),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().
		Int("slide_index", slideIndex).
		Interface("slide", slideResult.Slide).
		Interface("requires", slideResult.Requires).
		Msg("[slide_generator] parsed slide result")

	output := map[string]interface{}{
		"type":        "slide_result",
		"slide_index": slideIndex,
		"slide":       slideResult.Slide,
		"requires":    slideResult.Requires,
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Int("slide_index", slideIndex).Msg("[slide_generator] executeSingleSlide completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func slideLayoutID(slide any) string {
	if slide == nil {
		return ""
	}
	slideMap, ok := slide.(map[string]interface{})
	if !ok {
		return ""
	}
	if layoutID, ok := slideMap["layoutId"].(string); ok {
		return strings.TrimSpace(layoutID)
	}
	return ""
}

func slideHasElementType(slide any, elementType string) bool {
	if slide == nil {
		return false
	}
	slideMap, ok := slide.(map[string]interface{})
	if !ok {
		return false
	}
	rawElements, ok := slideMap["elements"].([]interface{})
	if !ok {
		return false
	}
	for _, raw := range rawElements {
		el, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if t, ok := el["type"].(string); ok && t == elementType {
			return true
		}
	}
	return false
}

func ensureSlideOrderAndID(slide map[string]any, slideIndex int) {
	slide["order"] = slideIndex
	expectedID := fmt.Sprintf("slide_%d", slideIndex)
	if id, ok := slide["id"].(string); !ok || strings.TrimSpace(id) == "" || id != expectedID {
		slide["id"] = expectedID
	}
}

func ensureSlideUseComponents(slide map[string]any) {
	if _, ok := slide["useComponents"]; !ok {
		slide["useComponents"] = []any{}
	}
}

func templateLayoutIDs(layouts any) map[string]bool {
	result := map[string]bool{}
	if layoutsSlice, ok := layouts.([]any); ok {
		for _, layout := range layoutsSlice {
			if layoutMap, ok := layout.(map[string]any); ok {
				if id, ok := layoutMap["id"].(string); ok && strings.TrimSpace(id) != "" {
					result[strings.TrimSpace(id)] = true
				}
			}
		}
	}
	return result
}

func layoutIDMatchesSuggestedLayout(layoutID string, suggestedLayout string, layouts any) bool {
	if layoutID == "" || suggestedLayout == "" {
		return false
	}
	if strings.EqualFold(layoutID, suggestedLayout) {
		return true
	}
	normalized := normalizeLayoutToken(layoutID)
	if normalized == suggestedLayout {
		return true
	}

	nameMap := layoutNameByID(layouts)
	if name, ok := nameMap[layoutID]; ok {
		nameNormalized := normalizeLayoutToken(name)
		if nameNormalized == suggestedLayout {
			return true
		}
	}

	if alias, ok := layoutAliasMap()[normalized]; ok {
		return alias == suggestedLayout
	}

	return false
}

func layoutNameByID(layouts any) map[string]string {
	result := map[string]string{}
	if layoutsSlice, ok := layouts.([]any); ok {
		for _, layout := range layoutsSlice {
			if layoutMap, ok := layout.(map[string]any); ok {
				id, _ := layoutMap["id"].(string)
				name, _ := layoutMap["name"].(string)
				if id != "" && name != "" {
					result[id] = name
				}
			}
		}
	}
	return result
}

func normalizeLayoutToken(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "layout_")
	value = strings.TrimPrefix(value, "layout-")
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ToUpper(value)
	return value
}

func layoutAliasMap() map[string]string {
	return map[string]string{
		"TITLE_BULLETS":        "TITLE_AND_BULLETS",
		"TITLE_AND_BULLETS":    "TITLE_AND_BULLETS",
		"TITLE_TWO_COLUMNS":    "TITLE_TWO_COLUMNS",
		"TITLE_IMAGE":          "TITLE_IMAGE",
		"FULL_BLEED_IMAGE":     "FULL_BLEED_IMAGE",
		"SECTION_HEADER":       "SECTION_HEADER",
		"TABLE":                "TABLE",
		"CHART":                "CHART",
		"QUOTE":                "QUOTE",
		"TIMELINE":             "TIMELINE",
		"CLOSING":              "CLOSING",
		"APPENDIX":             "APPENDIX",
		"DASHBOARD_3KPI_2COL":  "DASHBOARD_3KPI_2COL",
		"CHART_AND_INSIGHTS":   "CHART_AND_INSIGHTS",
		"TABLE_AND_CALLOUTS":   "TABLE_AND_CALLOUTS",
		"TITLE":                "TITLE",
		"TITLE_SLIDE":          "TITLE",
		"TITLE_AND_BULLETS_V1": "TITLE_AND_BULLETS",
	}
}

func extractAssetIDs(assets []any) map[string]bool {
	ids := map[string]bool{}
	for _, asset := range assets {
		switch v := asset.(type) {
		case string:
			if v != "" {
				ids[v] = true
			}
		case map[string]any:
			if id, ok := v["id"].(string); ok && id != "" {
				ids[id] = true
			}
		}
	}
	return ids
}

func extractDatasetIDs(datasets []any) map[string]bool {
	ids := map[string]bool{}
	for _, dataset := range datasets {
		if v, ok := dataset.(map[string]any); ok {
			if id, ok := v["id"].(string); ok && id != "" {
				ids[id] = true
			}
		}
	}
	return ids
}

func validateChartDatasetRefs(slide map[string]any, datasetIDs map[string]bool) string {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if elemType, _ := elem["type"].(string); elemType != "chart" {
			continue
		}
		chart, _ := elem["chart"].(map[string]any)
		if chart == nil {
			continue
		}
		if ref, ok := chart["datasetRef"].(string); ok && ref != "" {
			if !datasetIDs[ref] {
				return ref
			}
		}
	}
	return ""
}

func validateImageAssetRefs(slide map[string]any, assetIDs map[string]bool) string {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if elemType, _ := elem["type"].(string); elemType != "image" {
			continue
		}
		image, _ := elem["image"].(map[string]any)
		if image == nil {
			continue
		}
		if ref, ok := image["ref"].(string); ok && ref != "" {
			if !assetIDs[ref] {
				return ref
			}
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) executeUploadSlideSpec(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Msg("[slide_generator] executeUploadSlideSpec started")
	deckJSON, err := e.assembleDeck(input)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to assemble deck")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ASSEMBLY_ERROR",
				Message:  fmt.Sprintf("Failed to assemble deck: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	targetPath, _ := params["target_path"].(string)
	if targetPath == "" {
		responseID := "slide_spec"
		if input.PlanContext != nil && input.PlanContext.ResponseID != "" {
			responseID = input.PlanContext.ResponseID
		}
		targetPath = fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", responseID)
	}

	code := fmt.Sprintf(`import json

spec = json.loads(%s)

with open(%q, "w", encoding="utf-8") as f:
    json.dump(spec, f, indent=2)

print(json.dumps({"success": True, "path": %q, "size": len(json.dumps(spec))}))
`, strconv.Quote(string(deckJSON)), targetPath, targetPath)

	callReq := tool.CallRequest{
		Name: "aio_code_execute",
		Arguments: map[string]interface{}{
			"language": "python",
			"code":     code,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	log.Debug().Str("target_path", targetPath).Int("deck_size", len(deckJSON)).Msg("[slide_generator] uploading slide spec")
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil || (result != nil && result.IsError) {
		log.Error().Err(err).Bool("is_error", result != nil && result.IsError).Msg("[slide_generator] failed to upload slide spec")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "UPLOAD_ERROR",
				Message:  "Failed to upload slide spec to sandbox",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	log.Debug().Str("target_path", targetPath).Int("size", len(deckJSON)).Msg("[slide_generator] slide spec uploaded successfully")

	output := map[string]interface{}{
		"type": "file_uploaded",
		"path": targetPath,
		"size": len(deckJSON),
		"json": string(deckJSON), // Include the DeckSpec JSON for the next step
	}
	outputBytes, _ := json.Marshal(output)
	log.Debug().Msg("[slide_generator] executeUploadSlideSpec completed")

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeRenderScript(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Info().Bool("renderer_enabled", e.rendererEnabled).Msg("[slide_generator] executeRenderScript started")

	if !e.rendererEnabled {
		log.Warn().Msg("[slide_generator] slide renderer is disabled")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_DISABLED",
				Message:  "slide renderer is disabled",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Check if AIO client is available
	if e.aioClient == nil {
		log.Error().Msg("[slide_generator] AIO sandbox client not initialized (AIO_URL not configured)")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "AIO_NOT_CONFIGURED",
				Message:  "AIO_URL environment variable not set",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	renderScriptContent := e.loadRenderDeckScript()
	if strings.TrimSpace(renderScriptContent) == "" {
		log.Error().Msg("[slide_generator] render_deck.py script is empty")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_SCRIPT_MISSING",
				Message:  "render_deck.py content is empty",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Get DeckSpec JSON from previous step
	var deckSpecJSON string
	if input.PreviousOutput != nil {
		var prevOutput map[string]interface{}
		if err := json.Unmarshal(input.PreviousOutput, &prevOutput); err == nil {
			// Try to get the JSON directly
			if jsonStr, ok := prevOutput["json"].(string); ok {
				deckSpecJSON = jsonStr
			} else if data, ok := prevOutput["data"]; ok {
				// Try to marshal the data object
				if jsonBytes, err := json.Marshal(data); err == nil {
					deckSpecJSON = string(jsonBytes)
				}
			}
		}
	}

	if deckSpecJSON == "" {
		log.Error().Msg("[slide_generator] No DeckSpec JSON from previous step")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MISSING_INPUT",
				Message:  "No DeckSpec JSON from previous step",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Get output path from params or use response ID
	outputPath := "/home/gem/output.pptx"
	if params != nil {
		if path, ok := params["output_path"].(string); ok && path != "" {
			outputPath = path
		}
	}

	log.Info().
		Int("deck_spec_size", len(deckSpecJSON)).
		Int("render_script_size", len(renderScriptContent)).
		Str("output_path", outputPath).
		Msg("[slide_generator] Starting PPTX rendering via direct AIO sandbox")

	// Use direct AIO sandbox client (bypasses unstable MCP layer)
	// This executes everything in a single sandbox call with clear step-by-step logging
	pptxData, err := e.aioClient.RenderSlidesPPTX(ctx, deckSpecJSON, renderScriptContent, outputPath)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] PPTX rendering failed")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "RENDER_ERROR",
				Message:  fmt.Sprintf("Render failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Encode to base64 for artifact storage
	base64Content := base64.StdEncoding.EncodeToString(pptxData)

	log.Info().
		Int("pptx_size", len(pptxData)).
		Int("base64_length", len(base64Content)).
		Msg("[slide_generator] ✓ PPTX rendering completed successfully")

	output := map[string]interface{}{
		"type":      "render_output",
		"base64":    base64Content,
		"filename":  "presentation.pptx",
		"mime_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"size":      len(pptxData),
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func truncateForLogString(data string, maxLen int) string {
	if data == "" {
		return ""
	}
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen] + "..."
}

func (e *SlideGeneratorExecutor) executeArtifactCreation(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	log.Debug().Str("step_id", step.ID).Msg("[slide_generator] executeArtifactCreation started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to parse artifact parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse artifact parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	config, _ := params["config"].(map[string]interface{})
	format, _ := config["format"].(string)
	if format == "" {
		format = "pptx"
	}
	artifactType, _ := params["artifact_type"].(string)
	isSlideArtifact := artifactType == "slides" || format == "pptx" || format == "pdf"

	if isSlideArtifact && input.PreviousOutput != nil {
		var skillOutput SkillExecuteOutput
		if err := json.Unmarshal(input.PreviousOutput, &skillOutput); err == nil && skillOutput.Success {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadSkillArtifact(ctx, step, input, skillOutput, artifactType, retentionPolicy)
		}
		if renderOutput := extractRenderOutput(input.PreviousOutput); renderOutput != nil {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadRenderedArtifact(ctx, step, input, renderOutput, artifactType, retentionPolicy)
		}
	}

	content := ""
	if input.PreviousOutput != nil {
		var prevOutput map[string]interface{}
		if err := json.Unmarshal(input.PreviousOutput, &prevOutput); err == nil {
			if c, ok := prevOutput["content"].(string); ok {
				content = c
			}
		}
		if content == "" {
			content = string(input.PreviousOutput)
		}
	}
	if content == "" {
		content = "Artifact content unavailable."
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	retentionPolicy, _ := config["retention_policy"].(string)
	if retentionPolicy == "" {
		retentionPolicy = "session"
	}

	contentType := resolveArtifactContentType(artifactType, format)
	title := resolveArtifactTitle(artifactType)
	filename := resolveArtifactFilename(artifactType, format)

	var storagePath *string
	var downloadURL string
	var mediaID string

	if e.mediaClient != nil {
		mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
			Content:        []byte(content),
			Filename:       filename,
			ContentType:    contentType.MimeTypeFor(),
			ConversationID: conversationID,
			ResponseID:     responseID,
			UserID:         userID,
		})
		if err != nil {
			log.Warn().Err(err).Str("response_id", responseID).Msg("failed to upload artifact to media-api, falling back to inline storage")
		} else {
			storagePath = &mediaArtifact.DownloadURL
			downloadURL = mediaArtifact.DownloadURL
			mediaID = mediaArtifact.ID
			log.Debug().
				Str("media_id", mediaID).
				Str("download_url", downloadURL).
				Str("response_id", responseID).
				Msg("artifact uploaded to media-api")
		}
	}

	var contentPtr *string
	if storagePath == nil {
		contentPtr = &content
	}

	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		Title:           title,
		Content:         contentPtr,
		StoragePath:     storagePath,
		SizeBytes:       int64(len(content)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ARTIFACT_ERROR",
				Message:  fmt.Sprintf("failed to create artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Use media URL if available, otherwise fall back to artifact API path
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}
	log.Debug().
		Str("artifact_id", createdArtifact.ID).
		Str("download_url", downloadURL).
		Int64("size", int64(len(content))).
		Str("content_type", string(contentType)).
		Msg("[slide_generator] artifact created successfully")
	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    filename,
			DownloadURL: downloadURL,
			Size:        int64(len(content)),
			ContentType: contentType.MimeTypeFor(),
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *SlideGeneratorExecutor) uploadSkillArtifact(ctx context.Context, step *plan.Step, input agent.ExecutionInput, skillOutput SkillExecuteOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", skillOutput.FileName).
		Str("mime_type", skillOutput.MimeType).
		Msg("[slide_generator] uploadSkillArtifact started")
	if e.mediaClient == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MEDIA_CLIENT_MISSING",
				Message:  "media client not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	var decoded []byte
	var err error
	if strings.TrimSpace(skillOutput.FileContentBase64) != "" {
		decoded, err = base64.StdEncoding.DecodeString(skillOutput.FileContentBase64)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "FILE_DECODE_ERROR",
					Message:  "failed to decode skill file content",
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	} else if strings.TrimSpace(skillOutput.OutputPath) != "" {
		decoded, err = e.readBinaryFileFromSandbox(ctx, skillOutput.OutputPath, input)
		if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "FILE_READ_ERROR",
					Message:  err.Error(),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	} else {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "no skill output file available for upload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	fileName := strings.TrimSpace(skillOutput.FileName)
	if fileName == "" {
		fileName = resolveArtifactFilename(artifactType, "")
	}
	mimeType := strings.TrimSpace(skillOutput.MimeType)
	if mimeType == "" {
		mimeType = resolveArtifactContentType(artifactType, "").MimeTypeFor()
	}

	mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
		Content:        decoded,
		Filename:       fileName,
		ContentType:    mimeType,
		ConversationID: conversationID,
		ResponseID:     responseID,
		UserID:         userID,
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "UPLOAD_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentType := resolveArtifactContentType(artifactType, "")
	title := resolveArtifactTitle(artifactType)
	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           title,
		StoragePath:     &mediaArtifact.DownloadURL,
		SizeBytes:       int64(len(decoded)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ARTIFACT_ERROR",
				Message:  fmt.Sprintf("failed to create artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Use media URL if available, otherwise fall back to artifact API path
	downloadURL := mediaArtifact.DownloadURL
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    fileName,
			DownloadURL: downloadURL,
			Size:        int64(len(decoded)),
			ContentType: mimeType,
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *SlideGeneratorExecutor) uploadRenderedArtifact(ctx context.Context, step *plan.Step, input agent.ExecutionInput, renderOutput *slideRenderOutput, artifactType string, retentionPolicy string) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("artifact_type", artifactType).
		Str("filename", renderOutput.FileName).
		Int("size", renderOutput.Size).
		Msg("[slide_generator] uploadRenderedArtifact started")
	if e.mediaClient == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "MEDIA_CLIENT_MISSING",
				Message:  "media client not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}
	if renderOutput == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "no render output available for upload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	if strings.TrimSpace(renderOutput.Base64) == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_MISSING",
				Message:  "render output missing base64 payload",
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(renderOutput.Base64)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "FILE_READ_ERROR",
				Message:  fmt.Sprintf("base64 decode failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	responseID := ""
	conversationID := ""
	userID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
		userID = input.PlanContext.UserID
	}

	fileName := strings.TrimSpace(renderOutput.FileName)
	if fileName == "" {
		fileName = resolveArtifactFilename(artifactType, "")
	}
	mimeType := strings.TrimSpace(renderOutput.MimeType)
	if mimeType == "" {
		mimeType = resolveArtifactContentType(artifactType, "").MimeTypeFor()
	}

	mediaArtifact, err := e.mediaClient.UploadArtifact(ctx, &media.UploadRequest{
		Content:        decoded,
		Filename:       fileName,
		ContentType:    mimeType,
		ConversationID: conversationID,
		ResponseID:     responseID,
		UserID:         userID,
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "UPLOAD_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	contentType := resolveArtifactContentType(artifactType, "")
	title := resolveArtifactTitle(artifactType)
	createdArtifact, err := e.artifactService.Create(ctx, artifact.CreateParams{
		ResponseID:      responseID,
		ContentType:     contentType,
		MimeType:        &mimeType,
		Title:           title,
		StoragePath:     &mediaArtifact.DownloadURL,
		SizeBytes:       int64(len(decoded)),
		RetentionPolicy: artifact.RetentionPolicy(retentionPolicy),
	})
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "ARTIFACT_ERROR",
				Message:  fmt.Sprintf("failed to create artifact: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Use media URL if we uploaded to media-api
	downloadURL := mediaArtifact.DownloadURL
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)
	}

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    fileName,
			DownloadURL: downloadURL,
			Size:        int64(len(decoded)),
			ContentType: mimeType,
		},
	}
	outputBytes, _ := json.Marshal(stepOutput)

	return &agent.ExecutionResult{
		Status:     status.StatusCompleted,
		Output:     outputBytes,
		ArtifactID: &createdArtifact.ID,
	}, nil
}

func (e *SlideGeneratorExecutor) readBinaryFileFromSandbox(ctx context.Context, path string, input agent.ExecutionInput) ([]byte, error) {
	if strings.TrimSpace(e.aioBaseURL) != "" {
		if payload, err := downloadAIOFile(ctx, e.aioBaseURL, path); err == nil {
			return payload, nil
		}
	}

	code := fmt.Sprintf(`import base64
import json
import os

path = %q
if not os.path.exists(path):
    print(json.dumps({"error": "file not found", "path": path}))
else:
    with open(path, "rb") as f:
        data = f.read()
    encoded = base64.b64encode(data).decode("ascii")
    print(json.dumps({"base64": encoded, "size": len(data)}))
`, path)

	callReq := tool.CallRequest{
		Name: "aio_code_execute",
		Arguments: map[string]interface{}{
			"language": "python",
			"code":     code,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("code execute failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("code execute returned nil result")
	}
	if result.IsError {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = firstTextContent(result.Content)
		}
		return nil, fmt.Errorf("code execute error: %s", errMsg)
	}

	rawText := firstTextContent(result.Content)
	if rawText == "" {
		return nil, fmt.Errorf("code execute returned empty content")
	}

	var execResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(rawText), &execResult); err != nil {
		return nil, fmt.Errorf("failed to parse code execute result: %w", err)
	}

	if execResult.Status != "" && execResult.Status != "ok" {
		return nil, fmt.Errorf("code execute status: %s, stderr: %s", execResult.Status, execResult.Stderr)
	}

	stdout := strings.TrimSpace(execResult.Stdout)
	if stdout == "" {
		return nil, fmt.Errorf("code execute returned empty stdout")
	}

	var fileResult struct {
		Base64 string `json:"base64"`
		Size   int    `json:"size"`
		Error  string `json:"error"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &fileResult); err != nil {
		return nil, fmt.Errorf("failed to parse file read result: %w, stdout: %s", err, stdout)
	}

	if fileResult.Error != "" {
		return nil, fmt.Errorf("file read error: %s (path: %s)", fileResult.Error, fileResult.Path)
	}
	if fileResult.Base64 == "" {
		return nil, fmt.Errorf("file read returned empty base64")
	}

	decoded, err := base64.StdEncoding.DecodeString(fileResult.Base64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	return decoded, nil
}

func downloadAIOFile(ctx context.Context, baseURL string, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty path")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	escaped := url.QueryEscape(path)
	reqURL := baseURL + "/v1/file/download?path=" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func resolveArtifactContentType(artifactType string, format string) artifact.ContentType {
	switch artifactType {
	case "report":
		return artifact.ContentTypeResearch
	case "document":
		return artifact.ContentTypeDocument
	case "spreadsheet":
		return artifact.ContentTypeTable
	case "markdown":
		return artifact.ContentTypeMarkdown
	default:
		if format == "markdown" {
			return artifact.ContentTypeMarkdown
		}
		return artifact.ContentTypeSlides
	}
}

func resolveArtifactTitle(artifactType string) string {
	switch artifactType {
	case "report":
		return "Research Report"
	case "document":
		return "Document"
	case "spreadsheet":
		return "Spreadsheet"
	default:
		return "Presentation"
	}
}

func resolveArtifactFilename(artifactType string, format string) string {
	switch artifactType {
	case "report":
		return "research_report.md"
	case "document":
		if format == "pdf" {
			return "document.pdf"
		}
		return "document.md"
	case "spreadsheet":
		return "spreadsheet.xlsx"
	case "markdown":
		return "content.md"
	default:
		if format == "pdf" {
			return "presentation.pdf"
		}
		return "presentation.pptx"
	}
}

// buildAccumulatedContext combines outputs from all previous tasks into a single context string.
func (e *SlideGeneratorExecutor) buildAccumulatedContext(input agent.ExecutionInput) string {
	log.Debug().
		Int("accumulated_outputs", len(input.AccumulatedOutputs)).
		Int("previous_output_size", len(input.PreviousOutput)).
		Msg("[slide_generator] buildAccumulatedContext started")
	var contextParts []string

	for _, output := range input.AccumulatedOutputs {
		if len(output) > 0 {
			extracted := e.extractContextFromOutput(output)
			if extracted != "" {
				contextParts = append(contextParts, extracted)
			}
		}
	}

	if len(input.PreviousOutput) > 0 {
		extracted := e.extractContextFromOutput(input.PreviousOutput)
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

func (e *SlideGeneratorExecutor) extractContextFromOutput(output json.RawMessage) string {
	if len(output) == 0 {
		return ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err == nil {
		if payloadType, _ := parsed["type"].(string); payloadType == "data_bank" {
			if content, ok := parsed["content"].(string); ok && content != "" {
				return fmt.Sprintf("[data_bank]: %s", content)
			}
			if data, ok := parsed["data"]; ok {
				if raw, err := json.Marshal(data); err == nil {
					return fmt.Sprintf("[data_bank]: %s", string(raw))
				}
			}
		}
		if content, ok := parsed["content"].([]interface{}); ok {
			var texts []string
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok && text != "" {
						texts = append(texts, text)
					}
				}
			}
			if len(texts) > 0 {
				return strings.Join(texts, "\n")
			}
		}

		if content, ok := parsed["content"].(string); ok && content != "" {
			return content
		}

		if text, ok := parsed["text"].(string); ok && text != "" {
			return text
		}

		if toolName, ok := parsed["tool_name"].(string); ok {
			if content, ok := parsed["content"].([]interface{}); ok {
				var texts []string
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
	}

	rawStr := string(output)
	if len(rawStr) > 10000 {
		return rawStr[:10000] + "... [truncated]"
	}
	return rawStr
}

func (e *SlideGeneratorExecutor) buildToolArguments(toolName string, params map[string]interface{}, input agent.ExecutionInput, description string) (map[string]interface{}, error) {
	switch toolName {
	case "google_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}

		if query == "" {
			return nil, fmt.Errorf("no search query provided")
		}
		return map[string]interface{}{"q": query}, nil

	case "image_search":
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			query = q
		} else if description != "" {
			query = description
		}

		if query == "" {
			return nil, fmt.Errorf("no image search query provided")
		}

		args := map[string]interface{}{"q": query}
		// Optional: set number of results
		if num, ok := params["num"].(float64); ok && num > 0 {
			args["num"] = int(num)
		} else {
			args["num"] = 10 // Default to 10 images
		}
		// Optional: geographic location
		if gl, ok := params["gl"].(string); ok && gl != "" {
			args["gl"] = gl
		}
		// Optional: language
		if hl, ok := params["hl"].(string); ok && hl != "" {
			args["hl"] = hl
		}
		return args, nil

	case "scrape":
		urls := e.extractURLsFromPreviousOutput(input.PreviousOutput)
		if len(urls) == 0 {
			if urlParam, ok := params["url"].(string); ok {
				urls = []string{urlParam}
			} else if urlsParam, ok := params["urls"].([]interface{}); ok {
				for _, u := range urlsParam {
					if urlStr, ok := u.(string); ok {
						urls = append(urls, urlStr)
					}
				}
			}
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("no URLs available to scrape from previous search results")
		}
		return map[string]interface{}{"url": urls[0]}, nil

	default:
		toolArgs := make(map[string]interface{})
		for k, v := range params {
			if k != "tool" && k != "description" {
				toolArgs[k] = v
			}
		}
		return toolArgs, nil
	}
}

func (e *SlideGeneratorExecutor) extractURLsFromPreviousOutput(previousOutput json.RawMessage) []string {
	if len(previousOutput) == 0 {
		return nil
	}

	var output map[string]interface{}
	if err := json.Unmarshal(previousOutput, &output); err != nil {
		return nil
	}

	var urls []string
	if content, ok := output["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					var textData map[string]interface{}
					if err := json.Unmarshal([]byte(text), &textData); err == nil {
						urls = append(urls, extractURLsFromData(textData)...)
					}
				}
			}
		}
	}

	urls = append(urls, extractURLsFromData(output)...)
	return urls
}

func extractURLsFromData(data map[string]interface{}) []string {
	var urls []string

	if organic, ok := data["organic"].([]interface{}); ok {
		for _, item := range organic {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	if results, ok := data["results"].([]interface{}); ok {
		for _, item := range results {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if url, ok := itemMap["source_url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if url, ok := itemMap["url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	return urls
}

func isNonCriticalToolForSlides(toolName string) bool {
	nonCriticalTools := map[string]bool{
		"google_search": true,
		"image_search":  true,
		"scrape":        true,
	}
	return nonCriticalTools[toolName]
}

func buildSkippedToolResultForSlides(toolName string, reason string, code string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"skipped": true,
		"tool":    toolName,
		"reason":  reason,
		"code":    code,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}

func (e *SlideGeneratorExecutor) getModelFromContext(input agent.ExecutionInput) string {
	if input.PlanContext != nil && input.PlanContext.Model != "" {
		return input.PlanContext.Model
	}
	return ""
}

func (e *SlideGeneratorExecutor) extractPlanAndTemplate(input agent.ExecutionInput) *schemas.PlanAndTemplate {
	log.Debug().Msg("[slide_generator] extractPlanAndTemplate started")
	candidates := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	if len(input.PreviousOutput) > 0 {
		candidates = append(candidates, input.PreviousOutput)
	}
	candidates = append(candidates, input.AccumulatedOutputs...)
	log.Debug().Int("candidates_count", len(candidates)).Msg("[slide_generator] searching for plan and template")

	for i := len(candidates) - 1; i >= 0; i-- {
		var payload map[string]interface{}
		if err := json.Unmarshal(candidates[i], &payload); err != nil {
			continue
		}
		if payloadType, _ := payload["type"].(string); payloadType == "plan_and_template" {
			planBytes, _ := json.Marshal(payload)
			var planAndTemplate schemas.PlanAndTemplate
			if err := json.Unmarshal(planBytes, &planAndTemplate); err == nil {
				log.Debug().
					Int("slide_count", len(planAndTemplate.Plan.Slides)).
					Str("deck_title", planAndTemplate.Plan.DeckTitle).
					Msg("[slide_generator] found plan and template from type=plan_and_template")
				return &planAndTemplate
			}
		}
		if planRaw, ok := payload["plan"]; ok {
			if templateRaw, ok := payload["template"]; ok {
				merged := map[string]interface{}{"plan": planRaw, "template": templateRaw}
				planBytes, _ := json.Marshal(merged)
				var planAndTemplate schemas.PlanAndTemplate
				if err := json.Unmarshal(planBytes, &planAndTemplate); err == nil {
					log.Debug().
						Int("slide_count", len(planAndTemplate.Plan.Slides)).
						Str("deck_title", planAndTemplate.Plan.DeckTitle).
						Msg("[slide_generator] found plan and template from separate keys")
					return &planAndTemplate
				}
			}
		}
	}
	log.Warn().Msg("[slide_generator] plan and template not found in any output")
	return nil
}

func normalizePlanIndices(plan *schemas.SlidePlan) {
	if plan == nil {
		return
	}
	for i := range plan.Slides {
		plan.Slides[i].Index = i + 1
	}
	plan.RecommendedSlideCount = len(plan.Slides)
}

func normalizeTemplateComponents(template *schemas.SlideTemplate) {
	if template == nil {
		return
	}
	components, ok := template.Components.([]any)
	if !ok {
		return
	}
	normalized := make([]any, 0, len(components))
	for _, compAny := range components {
		comp, ok := compAny.(map[string]any)
		if !ok {
			continue
		}
		if _, hasElements := comp["elements"]; hasElements {
			normalized = append(normalized, comp)
			continue
		}
		compID, _ := comp["id"].(string)
		compType, _ := comp["type"].(string)
		rect, _ := comp["rect"].(map[string]any)
		style, _ := comp["style"].(map[string]any)

		if compType == "" {
			compType = "text"
		}

		element := map[string]any{
			"id":   compID,
			"type": compType,
			"rect": rect,
		}

		switch compType {
		case "text":
			content := ""
			if compID == "header" {
				content = "{{title}}"
			} else if compID == "footer" {
				content = "{page}/{total_pages}"
			}
			element["text"] = map[string]any{
				"content": content,
				"runs":    []any{},
				"autoFit": "shrink",
				"style":   style,
			}
		case "shape":
			element["shape"] = map[string]any{
				"kind":  "rect",
				"style": map[string]any{"fill": map[string]any{"type": "solid", "color": "#FFFFFF"}},
			}
		case "image":
			if image, ok := comp["image"].(map[string]any); ok {
				element["image"] = image
			}
		}

		normalized = append(normalized, map[string]any{
			"id":       compID,
			"elements": []any{element},
		})
	}
	template.Components = normalized
}

func normalizeTemplateLayouts(plan *schemas.SlidePlan, template *schemas.SlideTemplate) {
	if template == nil {
		return
	}
	layoutsSlice, ok := template.Layouts.([]any)
	if !ok {
		layoutsSlice = []any{}
	}

	allowed := layoutEnumSet()
	usedIDs := map[string]bool{}
	normalizedLayouts := make([]any, 0, len(layoutsSlice))

	for _, layoutAny := range layoutsSlice {
		layout, ok := layoutAny.(map[string]any)
		if !ok {
			continue
		}
		rawID, _ := layout["id"].(string)
		rawName, _ := layout["name"].(string)
		candidateID := normalizeLayoutToken(rawID)
		if candidateID == "" {
			candidateID = normalizeLayoutToken(rawName)
		}
		if allowed[candidateID] && !usedIDs[candidateID] {
			layout["id"] = candidateID
			usedIDs[candidateID] = true
			normalizedLayouts = append(normalizedLayouts, layout)
			continue
		}
		if rawID != "" && !usedIDs[rawID] {
			usedIDs[rawID] = true
		} else if rawID == "" {
			rawID = fmt.Sprintf("CUSTOM_%d", len(usedIDs)+1)
			layout["id"] = rawID
			usedIDs[rawID] = true
		}
		normalizedLayouts = append(normalizedLayouts, layout)
	}

	for _, entry := range plan.Slides {
		layoutID := strings.TrimSpace(entry.SuggestedLayout)
		if layoutID == "" {
			continue
		}
		if usedIDs[layoutID] {
			continue
		}
		normalizedLayouts = append(normalizedLayouts, defaultLayoutForType(layoutID))
		usedIDs[layoutID] = true
	}

	template.Layouts = normalizedLayouts
}

func layoutEnumSet() map[string]bool {
	return map[string]bool{
		"TITLE":               true,
		"SECTION_HEADER":      true,
		"TITLE_AND_BULLETS":   true,
		"TITLE_TWO_COLUMNS":   true,
		"TITLE_IMAGE":         true,
		"FULL_BLEED_IMAGE":    true,
		"CHART":               true,
		"TABLE":               true,
		"QUOTE":               true,
		"TIMELINE":            true,
		"CLOSING":             true,
		"APPENDIX":            true,
		"DASHBOARD_3KPI_2COL": true,
		"CHART_AND_INSIGHTS":  true,
		"TABLE_AND_CALLOUTS":  true,
	}
}

func defaultLayoutForType(layoutID string) map[string]any {
	left := 36.0
	top := 36.0
	right := 36.0
	bottom := 36.0
	usableW := 960.0 - left - right
	usableH := 540.0 - top - bottom

	titleRect := map[string]any{"x": left, "y": 72.0, "w": usableW, "h": 48.0}
	bodyRect := map[string]any{"x": left, "y": 140.0, "w": usableW, "h": 320.0}
	headerRect := map[string]any{"x": left, "y": top, "w": 700.0, "h": 20.0}
	footerRect := map[string]any{"x": left, "y": 540.0 - bottom - 18.0, "w": usableW, "h": 18.0}

	slots := []any{}
	switch layoutID {
	case "TITLE":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 200.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "subtitle", "rect": map[string]any{"x": 120.0, "y": 290.0, "w": 720.0, "h": 60.0}},
		}
	case "SECTION_HEADER":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 220.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "subtitle", "rect": map[string]any{"x": 120.0, "y": 300.0, "w": 720.0, "h": 50.0}},
		}
	case "TITLE_TWO_COLUMNS":
		colW := (usableW - 24.0) / 2
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "left", "rect": map[string]any{"x": left, "y": 140.0, "w": colW, "h": 320.0}},
			map[string]any{"id": "right", "rect": map[string]any{"x": left + colW + 24.0, "y": 140.0, "w": colW, "h": 320.0}},
		}
	case "TITLE_IMAGE":
		imageW := 320.0
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": map[string]any{"x": left, "y": 140.0, "w": usableW - imageW - 24.0, "h": 320.0}},
			map[string]any{"id": "image", "rect": map[string]any{"x": left + usableW - imageW, "y": 140.0, "w": imageW, "h": 320.0}},
		}
	case "FULL_BLEED_IMAGE":
		slots = []any{
			map[string]any{"id": "image", "rect": map[string]any{"x": 0.0, "y": 0.0, "w": 960.0, "h": 540.0}},
			map[string]any{"id": "title", "rect": map[string]any{"x": left, "y": 72.0, "w": usableW, "h": 60.0}},
		}
	case "CHART":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "chart", "rect": bodyRect},
		}
	case "TABLE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "table", "rect": bodyRect},
		}
	case "QUOTE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "quote", "rect": bodyRect},
		}
	case "TIMELINE":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "timeline", "rect": bodyRect},
		}
	case "CLOSING":
		slots = []any{
			map[string]any{"id": "title", "rect": map[string]any{"x": 120.0, "y": 200.0, "w": 720.0, "h": 80.0}},
			map[string]any{"id": "body", "rect": map[string]any{"x": 120.0, "y": 290.0, "w": 720.0, "h": 120.0}},
		}
	case "APPENDIX":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": bodyRect},
		}
	case "DASHBOARD_3KPI_2COL":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "kpi1", "rect": map[string]any{"x": left, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "kpi2", "rect": map[string]any{"x": left + 300.0, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "kpi3", "rect": map[string]any{"x": left + 600.0, "y": 128.0, "w": 288.0, "h": 80.0}},
			map[string]any{"id": "chart_left", "rect": map[string]any{"x": left, "y": 224.0, "w": 548.0, "h": 260.0}},
			map[string]any{"id": "table_right", "rect": map[string]any{"x": left + 564.0, "y": 224.0, "w": 324.0, "h": 260.0}},
		}
	case "CHART_AND_INSIGHTS":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "chart_left", "rect": map[string]any{"x": left, "y": 140.0, "w": 560.0, "h": 320.0}},
			map[string]any{"id": "insights_right", "rect": map[string]any{"x": left + 580.0, "y": 140.0, "w": 308.0, "h": 320.0}},
		}
	case "TABLE_AND_CALLOUTS":
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "table_left", "rect": map[string]any{"x": left, "y": 140.0, "w": 560.0, "h": 320.0}},
			map[string]any{"id": "callouts_right", "rect": map[string]any{"x": left + 580.0, "y": 140.0, "w": 308.0, "h": 320.0}},
		}
	default:
		slots = []any{
			map[string]any{"id": "header", "rect": headerRect},
			map[string]any{"id": "footer", "rect": footerRect},
			map[string]any{"id": "title", "rect": titleRect},
			map[string]any{"id": "body", "rect": bodyRect},
		}
	}

	if len(slots) == 0 {
		slots = []any{map[string]any{"id": "body", "rect": map[string]any{"x": left, "y": top, "w": usableW, "h": usableH}}}
	}

	return map[string]any{
		"id":       layoutID,
		"name":     layoutID,
		"masterId": "master_default",
		"slots":    slots,
	}
}

func (e *SlideGeneratorExecutor) collectDataBankText(input agent.ExecutionInput) string {
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
		if payloadType, _ := payload["type"].(string); payloadType == "data_bank" {
			if content, ok := payload["content"].(string); ok && content != "" {
				return content
			}
			if data, ok := payload["data"]; ok {
				if raw, err := json.Marshal(data); err == nil {
					return string(raw)
				}
			}
		}
	}
	return "[No data bank available]"
}

func (e *SlideGeneratorExecutor) collectDataBankDatasets(input agent.ExecutionInput) []any {
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
		if payloadType, _ := payload["type"].(string); payloadType == "data_bank" {
			if data, ok := payload["data"].(map[string]any); ok {
				if datasets, ok := data["datasets"].([]any); ok {
					return datasets
				}
			}
		}
	}
	return nil
}

func (e *SlideGeneratorExecutor) collectImageAssets(input agent.ExecutionInput) []map[string]any {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	assetsByID := map[string]map[string]any{}
	for _, output := range outputs {
		for _, asset := range extractImageAssetsFromOutput(output) {
			id, _ := asset["id"].(string)
			if id == "" {
				continue
			}
			if _, exists := assetsByID[id]; !exists {
				assetsByID[id] = asset
			}
		}
	}

	assets := make([]map[string]any, 0, len(assetsByID))
	for _, asset := range assetsByID {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool {
		return fmt.Sprint(assets[i]["id"]) < fmt.Sprint(assets[j]["id"])
	})
	return assets
}

func extractImageAssetsFromOutput(output json.RawMessage) []map[string]any {
	if len(output) == 0 {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil
	}

	results := []map[string]any{}
	if content, ok := parsed["content"].([]any); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]any); ok {
				if text, ok := itemMap["text"].(string); ok && text != "" {
					var nested map[string]any
					if err := json.Unmarshal([]byte(text), &nested); err == nil {
						results = append(results, extractImageAssetsFromMap(nested)...)
					}
				}
			}
		}
	}

	results = append(results, extractImageAssetsFromMap(parsed)...)
	return results
}

func extractImageAssetsFromMap(data map[string]any) []map[string]any {
	results := []map[string]any{}
	for _, key := range []string{"images", "results", "items", "data"} {
		if arr, ok := data[key].([]any); ok {
			results = append(results, extractImageAssetsFromArray(arr)...)
		}
	}
	for _, value := range data {
		switch typed := value.(type) {
		case map[string]any:
			results = append(results, extractImageAssetsFromMap(typed)...)
		case []any:
			results = append(results, extractImageAssetsFromArray(typed)...)
		}
	}
	return results
}

func extractImageAssetsFromArray(arr []any) []map[string]any {
	results := []map[string]any{}
	for _, item := range arr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if asset := assetFromImageResult(itemMap); asset != nil {
			results = append(results, asset)
		}
	}
	return results
}

func assetFromImageResult(item map[string]any) map[string]any {
	urlStr := firstString(item, "imageUrl", "image_url", "url", "link", "thumbnail", "thumbnailUrl", "source_url")
	if urlStr == "" {
		return nil
	}
	parsed, _ := url.Parse(urlStr)
	host := ""
	if parsed != nil {
		host = parsed.Host
	}
	altText := firstString(item, "title", "alt", "altText", "snippet", "description")
	if altText == "" {
		altText = host
	}
	license := firstString(item, "license")
	attribution := firstString(item, "attribution", "source")
	if attribution == "" {
		attribution = host
	}
	id := assetIDFromURL(urlStr)
	return map[string]any{
		"id":   id,
		"kind": "image",
		"source": map[string]any{
			"type": "url",
			"url":  urlStr,
		},
		"altText":     altText,
		"license":     license,
		"attribution": attribution,
	}
}

func assetIDFromURL(urlStr string) string {
	hasher := sha1.New()
	hasher.Write([]byte(urlStr))
	return "img_" + hex.EncodeToString(hasher.Sum(nil))[:12]
}

func firstString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := item[key].(string); ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) assembleDeck(input agent.ExecutionInput) ([]byte, error) {
	log.Debug().Int("accumulated_outputs", len(input.AccumulatedOutputs)).Msg("[slide_generator] assembleDeck started")
	planAndTemplate := e.extractPlanAndTemplate(input)
	if planAndTemplate == nil {
		log.Error().Msg("[slide_generator] plan and template not found")
		return nil, fmt.Errorf("plan and template not found")
	}
	log.Debug().Int("plan_slides", len(planAndTemplate.Plan.Slides)).Msg("[slide_generator] extracted plan and template")

	slidesByIndex := map[int]any{}
	var allAssets []any
	var allDatasets []any
	lockedAssets := e.collectImageAssets(input)
	for _, asset := range lockedAssets {
		allAssets = append(allAssets, asset)
	}
	allDatasets = append(allDatasets, e.collectDataBankDatasets(input)...)

	for _, output := range input.AccumulatedOutputs {
		var parsed map[string]interface{}
		if err := json.Unmarshal(output, &parsed); err != nil {
			continue
		}
		if parsed["type"] == "slide_result" {
			if slideIndexRaw, ok := parsed["slide_index"].(float64); ok {
				if slide, ok := parsed["slide"]; ok {
					slidesByIndex[int(slideIndexRaw)] = slide
				}
				if requires, ok := parsed["requires"].(map[string]interface{}); ok {
					if assets, ok := requires["assets"].([]any); ok {
						allAssets = append(allAssets, assets...)
					}
					// Safely extract datasets - ignore invalid ones
					if datasets, ok := requires["datasets"].([]any); ok {
						for _, ds := range datasets {
							if ds != nil {
								allDatasets = append(allDatasets, ds)
							}
						}
					}
				}
			}
		}
	}

	expected := len(planAndTemplate.Plan.Slides)
	orderedSlides := make([]any, 0, expected)
	if expected > 0 {
		for i := 1; i <= expected; i++ {
			slide, ok := slidesByIndex[i]
			if !ok {
				return nil, fmt.Errorf("missing slide %d during assembly", i)
			}
			orderedSlides = append(orderedSlides, slide)
		}
	} else {
		indices := make([]int, 0, len(slidesByIndex))
		for idx := range slidesByIndex {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		for _, idx := range indices {
			orderedSlides = append(orderedSlides, slidesByIndex[idx])
		}
	}

	metadata := map[string]any{
		"title":    planAndTemplate.Plan.DeckTitle,
		"language": "en",
		"audience": planAndTemplate.Plan.Audience,
		"purpose":  planAndTemplate.Plan.Purpose,
		"tone":     planAndTemplate.Plan.Tone,
	}

	if tmplMeta, ok := planAndTemplate.Template.Metadata.(map[string]any); ok {
		if _, hasLang := tmplMeta["language"]; hasLang {
			for key, value := range tmplMeta {
				metadata[key] = value
			}
			metadata["title"] = planAndTemplate.Plan.DeckTitle
			if planAndTemplate.Plan.Audience != "" {
				metadata["audience"] = planAndTemplate.Plan.Audience
			}
			if planAndTemplate.Plan.Purpose != "" {
				metadata["purpose"] = planAndTemplate.Plan.Purpose
			}
			if planAndTemplate.Plan.Tone != "" {
				metadata["tone"] = planAndTemplate.Plan.Tone
			}
		}
	}

	assets, err := e.normalizeAssets(allAssets)
	if err != nil {
		return nil, err
	}
	data, err := e.normalizeDatasets(allDatasets)
	if err != nil {
		return nil, err
	}

	deck := map[string]any{
		"version":    planAndTemplate.Template.Version,
		"metadata":   metadata,
		"theme":      planAndTemplate.Template.Theme,
		"masters":    planAndTemplate.Template.Masters,
		"layouts":    planAndTemplate.Template.Layouts,
		"components": planAndTemplate.Template.Components,
		"slides":     orderedSlides,
		"assets":     assets,
		"data":       data,
		"validation": map[string]any{"rules": []any{}},
		"export":     planAndTemplate.Template.Export,
	}

	fixedDeck := e.fixCommonSchemaIssues(deck)
	if err := validateDeck(fixedDeck, expected); err != nil {
		return nil, err
	}
	deckJSON, err := json.Marshal(fixedDeck)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to marshal deck")
		return nil, err
	}
	log.Debug().
		Int("slides_count", len(orderedSlides)).
		Int("assets_count", len(allAssets)).
		Int("datasets_count", len(allDatasets)).
		Int("deck_json_size", len(deckJSON)).
		Msg("[slide_generator] assembleDeck completed")
	return deckJSON, nil
}

func (e *SlideGeneratorExecutor) normalizeAssets(allAssets []any) (map[string]any, error) {
	normalizedImages := []any{}
	seen := map[string]bool{}
	for _, asset := range allAssets {
		if asset == nil {
			continue
		}
		switch v := asset.(type) {
		case map[string]any:
			id, _ := v["id"].(string)
			if id == "" {
				id = fmt.Sprintf("asset_%d", len(normalizedImages)+1)
				v["id"] = id
			}
			if seen[id] {
				continue
			}
			seen[id] = true

			if _, ok := v["kind"].(string); !ok {
				v["kind"] = "image"
			}

			source, _ := v["source"].(map[string]any)
			if source == nil {
				source = map[string]any{}
			}

			if _, hasType := source["type"]; !hasType {
				if urlStr, ok := source["url"].(string); ok && urlStr != "" {
					source["type"] = "url"
				} else if filePath, ok := source["filePath"].(string); ok && filePath != "" {
					source["type"] = "file"
				} else if base64Str, ok := source["base64"].(string); ok && base64Str != "" {
					source["type"] = "base64"
				} else if urlStr, ok := v["url"].(string); ok && urlStr != "" {
					source["type"] = "url"
					source["url"] = urlStr
				} else if filePath, ok := v["filePath"].(string); ok && filePath != "" {
					source["type"] = "file"
					source["filePath"] = filePath
				} else if base64Str, ok := v["base64"].(string); ok && base64Str != "" {
					source["type"] = "base64"
					source["base64"] = base64Str
				}
			}

			sourceType, _ := source["type"].(string)
			switch sourceType {
			case "url":
				if urlStr, ok := source["url"].(string); !ok || urlStr == "" {
					return nil, fmt.Errorf("asset %s missing source.url", id)
				}
			case "file":
				if pathStr, ok := source["filePath"].(string); !ok || pathStr == "" {
					return nil, fmt.Errorf("asset %s missing source.filePath", id)
				}
			case "base64":
				if base64Str, ok := source["base64"].(string); !ok || base64Str == "" {
					return nil, fmt.Errorf("asset %s missing source.base64", id)
				}
			default:
				return nil, fmt.Errorf("asset %s has unsupported source type %q", id, sourceType)
			}

			v["source"] = source
			if _, hasAltText := v["altText"]; !hasAltText {
				v["altText"] = id
			}
			if _, hasLicense := v["license"]; !hasLicense {
				v["license"] = nil
			}
			if _, hasAttribution := v["attribution"]; !hasAttribution {
				v["attribution"] = nil
			}
			normalizedImages = append(normalizedImages, v)
		case string:
			return nil, fmt.Errorf("asset %q must be an object, not a string", v)
		default:
			return nil, fmt.Errorf("unsupported asset type %T", asset)
		}
	}

	return map[string]any{"images": normalizedImages}, nil
}

func (e *SlideGeneratorExecutor) normalizeDatasets(allDatasets []any) (map[string]any, error) {
	normalizedDatasets := []any{}
	seen := map[string]bool{}
	for i, dataset := range allDatasets {
		if dataset == nil {
			return nil, fmt.Errorf("dataset %d is nil", i)
		}
		switch v := dataset.(type) {
		case string:
			return nil, fmt.Errorf("dataset %q must be an object, not a string", v)
		case map[string]any:
			id, _ := v["id"].(string)
			if id == "" {
				return nil, fmt.Errorf("dataset missing id at index %d", i)
			}
			if seen[id] {
				continue
			}
			seen[id] = true

			kind, _ := v["kind"].(string)
			if kind == "" {
				kind = "series"
				v["kind"] = "series"
			}
			if kind != "series" {
				return nil, fmt.Errorf("dataset %s has unsupported kind %q", id, kind)
			}

			dataMap, _ := v["data"].(map[string]any)
			if dataMap == nil {
				return nil, fmt.Errorf("dataset %s missing data", id)
			}

			labels, _ := dataMap["labels"].([]any)
			if len(labels) == 0 {
				return nil, fmt.Errorf("dataset %s missing labels", id)
			}
			series, _ := dataMap["series"].([]any)
			if len(series) == 0 {
				return nil, fmt.Errorf("dataset %s missing series", id)
			}
			for _, seriesAny := range series {
				seriesMap, ok := seriesAny.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("dataset %s has invalid series entry", id)
				}
				values, _ := seriesMap["values"].([]any)
				if len(values) != len(labels) {
					return nil, fmt.Errorf("dataset %s has mismatched labels/values length", id)
				}
			}
			v["data"] = dataMap
			if _, hasSourceNote := v["sourceNote"]; !hasSourceNote {
				v["sourceNote"] = nil
			}
			normalizedDatasets = append(normalizedDatasets, v)
		default:
			return nil, fmt.Errorf("invalid dataset type %T", dataset)
		}
	}

	return map[string]any{"datasets": normalizedDatasets}, nil
}

func (e *SlideGeneratorExecutor) fixCommonSchemaIssues(deck map[string]any) map[string]any {
	allowedTextStyleProps := map[string]bool{
		"fontFamily": true, "fontSize": true, "bold": true, "italic": true,
		"underline": true, "color": true, "align": true, "valign": true,
		"lineHeight": true, "letterSpacing": true, "bullet": true,
	}

	allowedShapeStyleProps := map[string]bool{
		"fill": true, "stroke": true, "cornerRadius": true, "shadow": true,
	}

	layoutSlots := buildLayoutSlotMap(deck["layouts"])
	componentIDs := buildComponentIDSet(deck["components"])
	theme, _ := deck["theme"].(map[string]any)
	safeMargins := extractSafeMargins(theme)

	if slides, ok := deck["slides"].([]any); ok {
		for _, slideAny := range slides {
			slide, ok := slideAny.(map[string]any)
			if !ok {
				continue
			}
			ensureSlideUseComponents(slide)
			layoutID, _ := slide["layoutId"].(string)
			slotsForLayout := layoutSlots[layoutID]

			if isContentSlide(layoutID) {
				if componentIDs["header"] {
					appendComponentID(slide, "header")
				}
				if componentIDs["footer"] {
					appendComponentID(slide, "footer")
				}
				if !componentIDs["header"] && !hasSlotOrElement(slide, "header") {
					addHeaderElement(slide, slotsForLayout, theme, safeMargins)
				}
				if !componentIDs["footer"] && !hasSlotOrElement(slide, "footer") {
					addFooterElement(slide, slotsForLayout, theme, safeMargins)
				}
			}

			if elements, ok := slide["elements"].([]any); ok {
				for _, elemAny := range elements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
					}
					if slotID, ok := elem["slotId"].(string); ok && slotID != "" {
						if _, hasRect := elem["rect"].(map[string]any); !hasRect {
							if slot := slotsForLayout[slotID]; slot != nil {
								if rect, ok := rectFromSlot(slot, theme); ok {
									elem["rect"] = rect
								}
							}
						}
					}
					if text, ok := elem["text"].(map[string]any); ok {
						if content, ok := text["content"].(string); ok {
							if strings.Contains(content, "|") && !strings.Contains(content, "\n") {
								text["content"] = strings.ReplaceAll(content, "|", "\n")
							}
						}
						if v, ok := text["autoFit"]; !ok || v == nil {
							text["autoFit"] = "shrink"
						}
						if style, ok := text["style"].(map[string]any); ok {
							for _, k := range []string{"fontFamily", "fontSize", "bold", "italic", "underline", "color", "align", "valign", "lineHeight", "letterSpacing", "bullet"} {
								if _, has := style[k]; !has {
									style[k] = nil
								}
							}
							if b, ok := style["bullet"].(map[string]any); ok {
								for _, k := range []string{"enabled", "indent", "hanging"} {
									if _, has := b[k]; !has {
										b[k] = nil
									}
								}
							}
							fixStyleProperties(style, allowedTextStyleProps)
						}
					}

					if shape, ok := elem["shape"].(map[string]any); ok {
						allowedKinds := map[string]bool{"rect": true, "line": true, "arrow": true, "triangle": true, "diamond": true}
						if kind, ok := shape["kind"].(string); ok {
							if !allowedKinds[kind] {
								shape["kind"] = "rect"
							}
						} else {
							shape["kind"] = "rect"
						}

						style, _ := shape["style"].(map[string]any)
						if style == nil {
							style = map[string]any{}
							shape["style"] = style
						}
						for _, k := range []string{"fill", "stroke", "cornerRadius", "shadow"} {
							if _, has := style[k]; !has {
								style[k] = nil
							}
						}
						style["cornerRadius"] = nil
						style["shadow"] = nil

						fixStyleProperties(style, allowedShapeStyleProps)
					}
				}
			}
		}
	}

	return deck
}

func fixStyleProperties(style map[string]any, allowed map[string]bool) {
	for key := range style {
		if !allowed[key] {
			delete(style, key)
		}
	}
}

func buildLayoutSlotMap(layouts any) map[string]map[string]map[string]any {
	result := map[string]map[string]map[string]any{}
	layoutSlice, ok := layouts.([]any)
	if !ok {
		return result
	}
	for _, layoutAny := range layoutSlice {
		layout, ok := layoutAny.(map[string]any)
		if !ok {
			continue
		}
		layoutID, _ := layout["id"].(string)
		if layoutID == "" {
			continue
		}
		slots := map[string]map[string]any{}
		if slotList, ok := layout["slots"].([]any); ok {
			for _, slotAny := range slotList {
				if slotMap, ok := slotAny.(map[string]any); ok {
					if slotID, ok := slotMap["id"].(string); ok && slotID != "" {
						slots[slotID] = slotMap
					}
				}
			}
		}
		result[layoutID] = slots
	}
	return result
}

func buildComponentIDSet(components any) map[string]bool {
	result := map[string]bool{}
	componentsSlice, ok := components.([]any)
	if !ok {
		return result
	}
	for _, compAny := range componentsSlice {
		comp, ok := compAny.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := comp["id"].(string); ok && id != "" {
			result[id] = true
		}
	}
	return result
}

func appendComponentID(slide map[string]any, componentID string) {
	useComponents, _ := slide["useComponents"].([]any)
	if useComponents == nil {
		useComponents = []any{}
	}
	for _, existing := range useComponents {
		if existingID, ok := existing.(string); ok && existingID == componentID {
			return
		}
	}
	slide["useComponents"] = append(useComponents, componentID)
}

func isContentSlide(layoutID string) bool {
	switch strings.TrimSpace(layoutID) {
	case "TITLE", "SECTION_HEADER", "CLOSING":
		return false
	default:
		return true
	}
}

func hasSlotOrElement(slide map[string]any, target string) bool {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if slotID, ok := elem["slotId"].(string); ok && slotID == target {
			return true
		}
		if id, ok := elem["id"].(string); ok && strings.Contains(strings.ToLower(id), target) {
			return true
		}
	}
	return false
}

func addHeaderElement(slide map[string]any, slots map[string]map[string]any, theme map[string]any, safeMargins map[string]float64) {
	headerText := ""
	if title, ok := slide["title"].(string); ok {
		headerText = title
	}
	elem := map[string]any{
		"id":   fmt.Sprintf("%v_header", slide["id"]),
		"type": "text",
		"text": map[string]any{
			"content": headerText,
			"runs":    []any{},
			"autoFit": "shrink",
			"style": map[string]any{
				"fontSize": 12,
				"color":    themeMutedText(theme),
				"align":    "left",
			},
		},
	}
	if slot, ok := slots["header"]; ok {
		elem["slotId"] = "header"
		if rect, ok := rectFromSlot(slot, theme); ok {
			elem["rect"] = rect
		}
	} else {
		elem["rect"] = map[string]any{
			"x": safeMargins["left"],
			"y": safeMargins["top"],
			"w": 700.0,
			"h": 20.0,
		}
	}
	appendSlideElement(slide, elem)
}

func addFooterElement(slide map[string]any, slots map[string]map[string]any, theme map[string]any, safeMargins map[string]float64) {
	elem := map[string]any{
		"id":   fmt.Sprintf("%v_footer", slide["id"]),
		"type": "text",
		"text": map[string]any{
			"content": "{page}/{total_pages}",
			"runs":    []any{},
			"autoFit": "shrink",
			"style": map[string]any{
				"fontSize": 10,
				"color":    themeMutedText(theme),
				"align":    "right",
			},
		},
	}
	if slot, ok := slots["footer"]; ok {
		elem["slotId"] = "footer"
		if rect, ok := rectFromSlot(slot, theme); ok {
			elem["rect"] = rect
		}
	} else {
		elem["rect"] = map[string]any{
			"x": safeMargins["left"],
			"y": 540.0 - safeMargins["bottom"] - 18.0,
			"w": 960.0 - safeMargins["left"] - safeMargins["right"],
			"h": 18.0,
		}
	}
	appendSlideElement(slide, elem)
}

func appendSlideElement(slide map[string]any, elem map[string]any) {
	elements, _ := slide["elements"].([]any)
	slide["elements"] = append(elements, elem)
}

func themeMutedText(theme map[string]any) string {
	if theme == nil {
		return "#6B7280"
	}
	if colors, ok := theme["colors"].(map[string]any); ok {
		if semantic, ok := colors["semantic"].(map[string]any); ok {
			if muted, ok := semantic["mutedText"].(string); ok && muted != "" {
				return muted
			}
		}
	}
	return "#6B7280"
}

func extractSafeMargins(theme map[string]any) map[string]float64 {
	margins := map[string]float64{"top": 36, "right": 36, "bottom": 36, "left": 36}
	if theme == nil {
		return margins
	}
	canvas, _ := theme["canvas"].(map[string]any)
	if canvas == nil {
		return margins
	}
	safe, _ := canvas["safeMargins"].(map[string]any)
	if safe == nil {
		return margins
	}
	if v, ok := safe["top"].(float64); ok {
		margins["top"] = v
	}
	if v, ok := safe["right"].(float64); ok {
		margins["right"] = v
	}
	if v, ok := safe["bottom"].(float64); ok {
		margins["bottom"] = v
	}
	if v, ok := safe["left"].(float64); ok {
		margins["left"] = v
	}
	return margins
}

func rectFromSlot(slot map[string]any, theme map[string]any) (map[string]any, bool) {
	if slot == nil {
		return nil, false
	}
	if theme == nil {
		theme = map[string]any{}
	}
	if rect, ok := slot["rect"].(map[string]any); ok {
		return rect, true
	}
	gridSpec, _ := slot["grid"].(map[string]any)
	if gridSpec == nil {
		return nil, false
	}
	col := int(asFloat(gridSpec["col"]))
	span := int(asFloat(gridSpec["span"]))
	if col <= 0 || span <= 0 {
		return nil, false
	}

	themeGrid, _ := theme["grid"].(map[string]any)
	columns := int(asFloat(themeGrid["columns"]))
	if columns <= 0 {
		columns = 12
	}
	gutter := asFloat(themeGrid["gutter"])
	baseline := asFloat(themeGrid["baseline"])
	if baseline <= 0 {
		baseline = 8
	}
	snapOn := true
	if snapRaw, ok := themeGrid["snap"].(bool); ok {
		snapOn = snapRaw
	}

	margins := extractSafeMargins(theme)
	usableW := 960.0 - margins["left"] - margins["right"]
	colW := (usableW - gutter*float64(columns-1)) / float64(columns)
	x := margins["left"] + float64(col-1)*(colW+gutter)
	w := float64(span)*colW + float64(span-1)*gutter
	y := asFloat(slot["y"])
	h := asFloat(slot["h"])
	if snapOn {
		y = math.Round(y/baseline) * baseline
		h = math.Round(h/baseline) * baseline
	}
	return map[string]any{"x": x, "y": y, "w": w, "h": h}, true
}

func asFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
}

func validateDeck(deck map[string]any, expectedSlides int) error {
	var issues []string
	slides, _ := deck["slides"].([]any)
	if expectedSlides > 0 && len(slides) != expectedSlides {
		issues = append(issues, fmt.Sprintf("expected %d slides, got %d", expectedSlides, len(slides)))
	}

	layoutIDs := templateLayoutIDs(deck["layouts"])
	componentIDs := buildComponentIDSet(deck["components"])
	theme, _ := deck["theme"].(map[string]any)
	safeMargins := extractSafeMargins(theme)
	layoutSlots := buildLayoutSlotMap(deck["layouts"])

	assetIDs := map[string]bool{}
	if assets, ok := deck["assets"].(map[string]any); ok {
		if images, ok := assets["images"].([]any); ok {
			for _, imgAny := range images {
				if img, ok := imgAny.(map[string]any); ok {
					if id, ok := img["id"].(string); ok && id != "" {
						assetIDs[id] = true
					}
				}
			}
		}
	}

	datasetIDs := map[string]bool{}
	if data, ok := deck["data"].(map[string]any); ok {
		if datasets, ok := data["datasets"].([]any); ok {
			for _, dsAny := range datasets {
				if ds, ok := dsAny.(map[string]any); ok {
					if id, ok := ds["id"].(string); ok && id != "" {
						datasetIDs[id] = true
					}
				}
			}
		}
	}

	orderSeen := map[int]bool{}
	idSeen := map[string]bool{}

	for idx, slideAny := range slides {
		slide, ok := slideAny.(map[string]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("slide %d is not an object", idx+1))
			continue
		}
		order, _ := parseIntFromInterface(slide["order"])
		if order <= 0 {
			issues = append(issues, fmt.Sprintf("slide %d has invalid order", idx+1))
		} else {
			orderSeen[order] = true
		}
		if id, ok := slide["id"].(string); ok && id != "" {
			if idSeen[id] {
				issues = append(issues, fmt.Sprintf("duplicate slide id %s", id))
			}
			idSeen[id] = true
		} else {
			issues = append(issues, fmt.Sprintf("slide %d missing id", idx+1))
		}

		layoutID, _ := slide["layoutId"].(string)
		if layoutID == "" || !layoutIDs[layoutID] {
			issues = append(issues, fmt.Sprintf("slide %d has unknown layoutId %q", idx+1, layoutID))
		}

		if isContentSlide(layoutID) {
			useComponents, _ := slide["useComponents"].([]any)
			hasHeaderComponent := false
			hasFooterComponent := false
			for _, comp := range useComponents {
				if compID, ok := comp.(string); ok {
					if compID == "header" {
						hasHeaderComponent = true
					}
					if compID == "footer" {
						hasFooterComponent = true
					}
				}
			}
			if componentIDs["header"] && !hasHeaderComponent && !hasSlotOrElement(slide, "header") {
				issues = append(issues, fmt.Sprintf("slide %d missing header", idx+1))
			}
			if componentIDs["footer"] && !hasFooterComponent && !hasSlotOrElement(slide, "footer") {
				issues = append(issues, fmt.Sprintf("slide %d missing footer", idx+1))
			}
			if !componentIDs["header"] && !hasSlotOrElement(slide, "header") {
				issues = append(issues, fmt.Sprintf("slide %d missing header", idx+1))
			}
			if !componentIDs["footer"] && !hasSlotOrElement(slide, "footer") {
				issues = append(issues, fmt.Sprintf("slide %d missing footer", idx+1))
			}
		}

		elements, _ := slide["elements"].([]any)
		for _, elemAny := range elements {
			elem, ok := elemAny.(map[string]any)
			if !ok {
				continue
			}
			if slotID, ok := elem["slotId"].(string); ok && slotID != "" {
				if slotsForLayout, ok := layoutSlots[layoutID]; ok {
					if _, exists := slotsForLayout[slotID]; !exists {
						issues = append(issues, fmt.Sprintf("slide %d uses unknown slotId %q", idx+1, slotID))
					}
				}
			}
			if rect, ok := elem["rect"].(map[string]any); ok {
				x := asFloat(rect["x"])
				y := asFloat(rect["y"])
				w := asFloat(rect["w"])
				h := asFloat(rect["h"])
				if x < 0 || y < 0 || x+w > 960 || y+h > 540 {
					issues = append(issues, fmt.Sprintf("slide %d element %v out of bounds", idx+1, elem["id"]))
				}
				if elemType, _ := elem["type"].(string); elemType == "text" {
					if x < safeMargins["left"] || y < safeMargins["top"] || x+w > 960-safeMargins["right"] || y+h > 540-safeMargins["bottom"] {
						issues = append(issues, fmt.Sprintf("slide %d text element %v violates safe margins", idx+1, elem["id"]))
					}
				}
			}
			if elemType, _ := elem["type"].(string); elemType == "chart" {
				chart, _ := elem["chart"].(map[string]any)
				if chart != nil {
					if ref, ok := chart["datasetRef"].(string); ok && ref != "" && !datasetIDs[ref] {
						issues = append(issues, fmt.Sprintf("slide %d chart datasetRef %q missing", idx+1, ref))
					}
				}
			}
			if elemType, _ := elem["type"].(string); elemType == "image" {
				image, _ := elem["image"].(map[string]any)
				if image != nil {
					if ref, ok := image["ref"].(string); ok && ref != "" && !assetIDs[ref] {
						issues = append(issues, fmt.Sprintf("slide %d image ref %q missing", idx+1, ref))
					}
				}
			}
		}
	}

	for i := 1; i <= len(slides); i++ {
		if !orderSeen[i] {
			issues = append(issues, fmt.Sprintf("missing slide order %d", i))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("deck validation failed: %s", strings.Join(issues, "; "))
	}
	return nil
}

func (e *SlideGeneratorExecutor) loadRenderDeckScript() string {
	if e.rendererScriptPath != "" {
		if payload, err := os.ReadFile(e.rendererScriptPath); err == nil {
			return string(payload)
		}
		log.Warn().
			Str("path", e.rendererScriptPath).
			Msg("Failed to read SLIDE_RENDERER_SCRIPT, falling back to embedded script")
	}
	return sliderenderer.RenderDeckScript
}

func cloneSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	payload, _ := json.Marshal(schema)
	var clone map[string]any
	_ = json.Unmarshal(payload, &clone)
	return clone
}

// extractJSONFromResponse extracts JSON from a response that may be wrapped in markdown code blocks
func extractJSONFromResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code block markers if present
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}

	return strings.TrimSpace(response)
}

type slideRenderOutput struct {
	FileName string `json:"filename"`
	MimeType string `json:"mime_type"`
	Base64   string `json:"base64"`
	Size     int    `json:"size"`
}

func extractRenderOutput(previousOutput json.RawMessage) *slideRenderOutput {
	if len(previousOutput) == 0 {
		return nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(previousOutput, &parsed); err != nil {
		return nil
	}
	if parsed["type"] != "render_output" {
		return nil
	}
	payload, _ := json.Marshal(parsed)
	var output slideRenderOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		return nil
	}
	return &output
}

// Verify interface compliance at compile time
var _ agent.Executor = (*SlideGeneratorExecutor)(nil)
