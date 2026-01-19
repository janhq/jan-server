// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	steps += 1 // outline
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
	default:
		return e.executeReasoning(ctx, params, input)
	}
}

func (e *SlideGeneratorExecutor) executeReasoning(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	description, _ := params["description"].(string)
	contextData := e.buildAccumulatedContext(input)
	prompt := fmt.Sprintf(
		"Analyze and plan the slide structure. %s\n\nResearch findings:\n%s\n\nProvide a clear, concise outline for the presentation.\nReturn plain text only.",
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

	response, err := e.llmProvider.GenerateWithModel(ctx, prompt, model)
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

	systemPrompt := fmt.Sprintf("%s\n%s", sizeGuardPrompt, plannerAndTemplatePrompt)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nTARGET SLIDE COUNT:\n%d\n\nTHEME:\n%s", contextData, numSlides, theme)

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
			result, err = e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
		} else {
			// Fallback: append schema to prompt
			log.Info().
				Int("attempt", attempt).
				Str("model", model).
				Str("method", "schema_in_prompt").
				Msg("[slide_generator] plan_and_template using fallback method (schema in prompt)")
			schemaJSON, _ := json.MarshalIndent(schemas.PlanAndTemplateSchema, "", "  ")
			enhancedUserPrompt := fmt.Sprintf("%s\n\nIMPORTANT: You MUST respond with valid JSON that strictly adheres to this schema:\n```json\n%s\n```\n\nReturn ONLY the JSON object, no markdown code blocks, no explanations.", userPrompt, string(schemaJSON))
			result, err = e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
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
	templateJSON, _ := json.Marshal(planAndTemplate.Template)
	planEntryJSON, _ := json.Marshal(planEntry)
	themeJSON, _ := json.Marshal(planAndTemplate.Template.Theme)

	systemPrompt := slideWriterPrompt(slideIndex)
	userPrompt := fmt.Sprintf("BRIEF:\n%s\n\nLOCKED TEMPLATE:\n%s\n\nTHEME:\n%s\n\nPLAN ENTRY (slide %d):\n%s", contextData, string(templateJSON), string(themeJSON), slideIndex, string(planEntryJSON))

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
			result, err = e.llmProvider.GenerateWithStructuredOutput(ctx, systemPrompt, userPrompt, model, schema)
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
			result, err = e.llmProvider.GenerateWithSystemPrompt(ctx, systemPrompt, enhancedUserPrompt, model)
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

	orderedSlides := make([]any, 0, len(slidesByIndex))
	if len(planAndTemplate.Plan.Slides) > 0 {
		for _, entry := range planAndTemplate.Plan.Slides {
			if slide, ok := slidesByIndex[entry.Index]; ok {
				orderedSlides = append(orderedSlides, slide)
			}
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

	deck := map[string]any{
		"version":    planAndTemplate.Template.Version,
		"metadata":   metadata,
		"theme":      planAndTemplate.Template.Theme,
		"masters":    planAndTemplate.Template.Masters,
		"layouts":    planAndTemplate.Template.Layouts,
		"components": planAndTemplate.Template.Components,
		"slides":     orderedSlides,
		"assets":     e.normalizeAssets(allAssets),
		"data":       e.normalizeDatasets(allDatasets),
		"validation": map[string]any{"rules": []any{}},
		"export":     planAndTemplate.Template.Export,
	}

	fixedDeck := e.fixCommonSchemaIssues(deck)
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

func (e *SlideGeneratorExecutor) normalizeAssets(allAssets []any) map[string]any {
	normalizedImages := []any{}
	for _, asset := range allAssets {
		switch v := asset.(type) {
		case string:
			normalizedImages = append(normalizedImages, map[string]any{
				"id": v,
				"source": map[string]any{
					"type":        "generated",
					"url":         nil,
					"filePath":    nil,
					"prompt":      fmt.Sprintf("Image for %s", v),
					"license":     nil,
					"attribution": nil,
				},
				"altText": v,
				"crop":    nil,
			})
		case map[string]any:
			if _, hasID := v["id"]; !hasID {
				v["id"] = fmt.Sprintf("asset_%d", len(normalizedImages)+1)
			}
			source, _ := v["source"].(map[string]any)
			if source == nil {
				source = map[string]any{}
			}
			if _, hasType := source["type"]; !hasType {
				source["type"] = "generated"
			}
			if _, hasURL := source["url"]; !hasURL {
				source["url"] = nil
			}
			if _, hasFilePath := source["filePath"]; !hasFilePath {
				source["filePath"] = nil
			}
			if _, hasPrompt := source["prompt"]; !hasPrompt {
				source["prompt"] = nil
			}
			if _, hasLicense := source["license"]; !hasLicense {
				source["license"] = nil
			}
			if _, hasAttribution := source["attribution"]; !hasAttribution {
				source["attribution"] = nil
			}
			v["source"] = source
			if _, hasAltText := v["altText"]; !hasAltText {
				v["altText"] = nil
			}
			if _, hasCrop := v["crop"]; !hasCrop {
				v["crop"] = nil
			}
			normalizedImages = append(normalizedImages, v)
		default:
			normalizedImages = append(normalizedImages, map[string]any{
				"id": fmt.Sprintf("asset_%d", len(normalizedImages)+1),
				"source": map[string]any{
					"type":        "generated",
					"url":         nil,
					"filePath":    nil,
					"prompt":      nil,
					"license":     nil,
					"attribution": nil,
				},
				"altText": nil,
				"crop":    nil,
			})
		}
	}

	return map[string]any{"images": normalizedImages}
}

func (e *SlideGeneratorExecutor) normalizeDatasets(allDatasets []any) map[string]any {
	normalizedDatasets := []any{}
	for i, dataset := range allDatasets {
		// Use defer/recover to catch panics from invalid datasets
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Warn().
						Int("dataset_index", i).
						Interface("panic", r).
						Msg("[slide_generator] panic while normalizing dataset, skipping")
				}
			}()

			if dataset == nil {
				log.Warn().Int("dataset_index", i).Msg("[slide_generator] nil dataset, skipping")
				return
			}

			switch v := dataset.(type) {
			case string:
				// String reference - create placeholder (should be avoided with updated prompts)
				log.Warn().Str("dataset_ref", v).Msg("[slide_generator] received string dataset reference instead of complete object, using placeholder")
				normalizedDatasets = append(normalizedDatasets, map[string]any{
					"id":   v,
					"kind": "series",
					"data": map[string]any{
						"labels": []string{"Category A", "Category B", "Category C"},
						"series": []any{map[string]any{"name": "Placeholder Data", "values": []float64{10, 20, 30}}},
					},
					"sourceNote": fmt.Sprintf("WARNING: Placeholder data for %s - real data not provided", v),
				})
			case map[string]any:
				// Complete dataset object - validate and normalize
				if _, hasID := v["id"]; !hasID {
					v["id"] = fmt.Sprintf("dataset_%d", len(normalizedDatasets)+1)
				}
				if _, hasKind := v["kind"]; !hasKind {
					v["kind"] = "series"
				}
				if _, hasData := v["data"]; !hasData {
					// Missing data - create placeholder
					datasetID := "unknown"
					if id, ok := v["id"].(string); ok {
						datasetID = id
					}
					log.Warn().Str("dataset_id", datasetID).Msg("[slide_generator] dataset missing data field, using placeholder")
					v["data"] = map[string]any{
						"labels": []string{"A", "B", "C"},
						"series": []any{map[string]any{"name": "Data", "values": []float64{1, 2, 3}}},
					}
				}
				kind, _ := v["kind"].(string)
				dataMap, _ := v["data"].(map[string]any)
				if dataMap == nil {
					dataMap = map[string]any{}
				}
				if kind == "table" {
					if _, ok := dataMap["columns"]; !ok {
						dataMap["columns"] = []string{"placeholder"}
					}
					if _, ok := dataMap["rows"]; !ok {
						dataMap["rows"] = [][]string{}
					}
				} else {
					if _, ok := dataMap["labels"]; !ok {
						dataMap["labels"] = []string{"A", "B", "C"}
					}
					if _, ok := dataMap["series"]; !ok {
						dataMap["series"] = []any{map[string]any{"name": "Data", "values": []float64{1, 2, 3}}}
					}
				}
				v["data"] = dataMap
				if _, hasSourceNote := v["sourceNote"]; !hasSourceNote {
					v["sourceNote"] = nil
				}
				normalizedDatasets = append(normalizedDatasets, v)
			default:
				log.Warn().
					Int("dataset_index", i).
					Str("type", fmt.Sprintf("%T", dataset)).
					Msg("[slide_generator] invalid dataset type, skipping")
			}
		}()
	}

	return map[string]any{"datasets": normalizedDatasets}
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

	if slides, ok := deck["slides"].([]any); ok {
		for _, slideAny := range slides {
			slide, ok := slideAny.(map[string]any)
			if !ok {
				continue
			}
			if elements, ok := slide["elements"].([]any); ok {
				for _, elemAny := range elements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
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
