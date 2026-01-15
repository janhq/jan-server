// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/agent"
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
	Format        string `json:"format"`         // pptx, pdf, google_slides
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
	// Parse slide configuration from metadata
	config := p.parseConfig(request)

	// Determine steps based on research depth
	estimatedSteps := p.calculateEstimatedSteps(config)

	// Create the plan
	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeSlideGenerator,
		EstimatedSteps: estimatedSteps,
		Config: &plan.PlanConfig{
			MaxRetries:        3,
			TimeoutPerStep:    300000000000, // 5 minutes in nanoseconds
			EnableFallback:    true,
			UserApproval:      config.OptionsCount > 1, // Require approval if multiple options
			StreamProgress:    true,
			ArtifactRetention: "session",
		},
	})
	if err != nil {
		return nil, err
	}

	// Track current task sequence
	taskSequence := 1

	// ============================================
	// Task 0: User Selection (when multiple options)
	// ============================================
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
			MaxRetries:  1,
		})
		if err != nil {
			return nil, err
		}

		taskSequence++
	}

	// ============================================
	// Task 1: Research (if research_depth != minimal)
	// ============================================
	if config.ResearchDepth != "minimal" {
		researchTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeResearch,
			Title:       "Research",
			Description: strPtr("Gather information and context for the presentation"),
		})
		if err != nil {
			return nil, err
		}

		// Step 1: Primary search
		searchParams1, _ := json.Marshal(map[string]interface{}{
			"tool":        "google_search",
			"description": "Search for key topics related to the presentation",
			"q":           request.UserMessage, // Include user query for search
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

		// Additional research for deep mode
		if config.ResearchDepth == "deep" {
			searchParams2, _ := json.Marshal(map[string]interface{}{
				"tool":        "google_search",
				"description": "Secondary search for supporting data and examples",
				"q":           request.UserMessage + " examples data statistics", // Extended query
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
				MaxRetries:  2,
			})
			if err != nil {
				return nil, err
			}
		}

		taskSequence++
	}

	// ============================================
	// Task 2: Outline & Structure
	// ============================================
	outlineTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeValidation,
		Title:       "Outline",
		Description: strPtr("Create presentation outline and structure"),
	})
	if err != nil {
		return nil, err
	}

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
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	// ============================================
	// Task 3: Content Generation
	// ============================================
	contentTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Generate Content",
		Description: strPtr("Generate slide content based on outline"),
	})
	if err != nil {
		return nil, err
	}

	// Generate content for each major section (grouped for efficiency)
	contentParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_slides_content",
		"description": "Generate content for all slides",
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
			"format":     config.Format,
		},
	})
	_, err = p.planService.CreateStep(ctx, contentTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		InputParams: contentParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	// Generate structured slide data (JSON format for rendering)
	structureParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_slides_json",
		"description": "Convert content to structured slide JSON format",
		"config": map[string]interface{}{
			"format": config.Format,
		},
	})
	_, err = p.planService.CreateStep(ctx, contentTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeLLMCall,
		InputParams: structureParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	// ============================================
	// Task 4: Visual Enhancement (optional for rich themes)
	// ============================================
	if p.requiresVisualEnhancement(config.Theme) {
		visualTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeTransform,
			Title:       "Visual Enhancement",
			Description: strPtr("Add visual elements and polish design"),
		})
		if err != nil {
			return nil, err
		}

		// Search for relevant images
		imageSearchParams, _ := json.Marshal(map[string]interface{}{
			"tool":        "google_search",
			"description": "Search for relevant images and graphics",
			"search_type": "images",
		})
		_, err = p.planService.CreateStep(ctx, visualTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeToolCall,
			InputParams: imageSearchParams,
			MaxRetries:  2,
		})
		if err != nil {
			return nil, err
		}

		taskSequence++
	}

	// ============================================
	// Task 5: Skill Execution & Artifact Creation
	// ============================================
	finalTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Generate Presentation",
		Description: strPtr("Generate final presentation artifact"),
	})
	if err != nil {
		return nil, err
	}

	// Execute skill to generate PPTX
	compileParams, _ := json.Marshal(map[string]interface{}{
		"skill_type": "slides",
		"options": map[string]interface{}{
			"format": config.Format,
			"theme":  config.Theme,
			"title":  request.UserMessage,
		},
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeSkillExecute,
		InputParams: compileParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	// Store artifact
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
		Sequence:    2,
		Action:      plan.ActionTypeArtifactCreate,
		InputParams: artifactParams,
		MaxRetries:  1,
	})
	if err != nil {
		return nil, err
	}

	// Reload plan with tasks
	planWithDetails, err := p.planService.GetPlanWithDetails(ctx, createdPlan.ID)
	if err != nil {
		return nil, err
	}

	// Build result
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

	// Parse top-level metadata fields first.
	applySlideConfigFromMap(&config, request.Metadata)

	// Parse options from metadata (tool choice options) last to allow overrides.
	if options, ok := request.Metadata["options"].(map[string]interface{}); ok {
		applySlideConfigFromMap(&config, options)
	}

	// Validate and clamp values
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
	if numSlides, ok := values["num_slides"].(float64); ok {
		config.NumSlides = int(numSlides)
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
	if optionsCount, ok := values["options_count"].(float64); ok {
		config.OptionsCount = int(optionsCount)
	}
}

// calculateEstimatedSteps returns the expected number of steps based on configuration.
func (p *SlideGeneratorPlanner) calculateEstimatedSteps(config SlideGeneratorConfig) int {
	steps := 0

	if config.OptionsCount > 1 {
		steps++
	}

	// Research steps
	switch config.ResearchDepth {
	case "minimal":
		steps += 0
	case "standard":
		steps += 1 // 1 search
	case "deep":
		steps += 3 // 2 searches + 1 scrape
	}

	// Outline step
	steps += 1

	// Content generation steps
	steps += 2 // content + structure

	// Visual enhancement (for non-minimal themes)
	if p.requiresVisualEnhancement(config.Theme) {
		steps += 1
	}

	// Finalization steps
	steps += 2 // compile + artifact

	return steps
}

// requiresVisualEnhancement checks if the theme needs image search.
func (p *SlideGeneratorPlanner) requiresVisualEnhancement(theme string) bool {
	// Themes that benefit from image search
	enhancedThemes := []string{"modern", "corporate", "creative", "infographic"}
	themeLower := strings.ToLower(theme)
	for _, t := range enhancedThemes {
		if strings.Contains(themeLower, t) {
			return true
		}
	}
	return false
}

// Verify interface compliance at compile time
var _ agent.Planner = (*SlideGeneratorPlanner)(nil)

// SlideGeneratorExecutor executes steps for slide generation plans.
type SlideGeneratorExecutor struct {
	mcpClient       MCPClient
	llmProvider     LLMProvider
	artifactService artifact.Service
	mediaClient     *media.Client
	skillExecutor   *SkillExecutor
}

// NewSlideGeneratorExecutor creates a new slide generator executor.
func NewSlideGeneratorExecutor(mcpClient MCPClient, llmProvider LLMProvider, artifactService artifact.Service, mediaClient *media.Client, skillExecutor *SkillExecutor) *SlideGeneratorExecutor {
	return &SlideGeneratorExecutor{
		mcpClient:       mcpClient,
		llmProvider:     llmProvider,
		artifactService: artifactService,
		mediaClient:     mediaClient,
		skillExecutor:   skillExecutor,
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
	// Slide generation steps are generally not reversible
	// Artifact creation could be rolled back by deleting the artifact
	if step.Action == plan.ActionTypeArtifactCreate {
		// Could implement artifact deletion here if needed
		return nil
	}
	return nil
}

// Execute runs a single step and returns the result.
func (e *SlideGeneratorExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
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
		return &agent.ExecutionResult{
			Status: status.StatusCompleted,
			Output: nil,
		}, nil
	}
}

func (e *SlideGeneratorExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse step parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	toolName, _ := params["tool"].(string)
	if toolName == "" {
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

	// Build actual tool arguments (strip metadata fields, validate per-tool requirements)
	toolArgs, err := e.buildToolArguments(toolName, params, input, description)
	if err != nil {
		// For non-critical tools like search/scrape, return skipped result instead of failing
		if isNonCriticalToolForSlides(toolName) {
			log.Warn().
				Str("tool", toolName).
				Err(err).
				Msg("Non-critical tool argument validation failed, skipping")
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

	// Extract context info
	requestID := ""
	conversationID := ""
	if input.PlanContext != nil {
		requestID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
	}

	log.Info().
		Str("tool", toolName).
		Interface("arguments", toolArgs).
		Str("step_id", step.ID).
		Msg("Executing tool call")

	// Execute tool via MCP client
	result, err := e.mcpClient.CallTool(ctx, tool.CallRequest{
		Name:           toolName,
		Arguments:      toolArgs,
		RequestID:      requestID,
		ConversationID: conversationID,
	})

	if err != nil {
		log.Error().
			Err(err).
			Str("tool", toolName).
			Interface("arguments", toolArgs).
			Str("step_id", step.ID).
			Msg("Tool call failed")

		// For non-critical tools, return skipped result instead of failing
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

	log.Info().
		Str("tool", toolName).
		Str("step_id", step.ID).
		Bool("is_error", result.IsError).
		Msg("Tool call completed")

	// Handle tool errors for non-critical tools gracefully
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
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	action, _ := params["action"].(string)
	description, _ := params["description"].(string)

	// Build context from accumulated outputs
	contextData := e.buildAccumulatedContext(input)

	// Build prompt based on action type
	var prompt string
	switch action {
	case "reasoning":
		prompt = fmt.Sprintf("Analyze and plan the slide structure. %s\n\nResearch findings: %s\n\nProvide a clear outline for the presentation.",
			description, contextData)
	case "generate_slides_content":
		config, _ := params["config"].(map[string]interface{})
		numSlides := 10
		if n, ok := config["num_slides"].(float64); ok {
			numSlides = int(n)
		}
		theme, _ := config["theme"].(string)
		prompt = fmt.Sprintf("Generate content for a %d-slide presentation with theme '%s'. %s\n\nResearch context: %s\n\nProvide detailed content for each slide with titles and bullet points.",
			numSlides, theme, description, contextData)
	case "generate_slides_json":
		prompt = fmt.Sprintf("Convert the slide content into structured JSON format suitable for presentation generation. %s\n\nContent to structure: %s",
			description, contextData)
	default:
		prompt = fmt.Sprintf("%s\n\nContext: %s", description, contextData)
	}

	// Get model from plan context
	model := ""
	if input.PlanContext != nil && input.PlanContext.Model != "" {
		model = input.PlanContext.Model
	}

	log.Info().
		Str("action", action).
		Str("step_id", step.ID).
		Str("model", model).
		Msg("Executing LLM call for slide generation")

	// Call LLM provider
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

	// Build output
	output := map[string]interface{}{
		"type":        "llm_response",
		"action":      action,
		"description": description,
		"content":     response,
	}
	outputBytes, _ := json.Marshal(output)

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeArtifactCreation(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "PARSE_ERROR",
				Message:  "failed to parse artifact parameters",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	// Extract artifact configuration
	config, _ := params["config"].(map[string]interface{})
	format, _ := config["format"].(string)
	if format == "" {
		format = "pptx"
	}
	artifactType, _ := params["artifact_type"].(string)

	if input.PreviousOutput != nil {
		var skillOutput SkillExecuteOutput
		if err := json.Unmarshal(input.PreviousOutput, &skillOutput); err == nil && skillOutput.Success {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadSkillArtifact(ctx, step, input, skillOutput, artifactType, retentionPolicy)
		}
	}

	// Get content from previous step output
	var content string
	if input.PreviousOutput != nil {
		var prevOutput map[string]interface{}
		if err := json.Unmarshal(input.PreviousOutput, &prevOutput); err == nil {
			if c, ok := prevOutput["content"].(string); ok {
				content = c
			}
		}
		// If not a map, try string directly
		if content == "" {
			content = string(input.PreviousOutput)
		}
	}

	if content == "" {
		content = "Artifact content unavailable."
	}

	// Get context info
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

	// Try to upload to media-api for persistent storage
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

	// Create artifact record (with inline content as fallback, or storage path if uploaded)
	var contentPtr *string
	if storagePath == nil {
		// Fallback to inline content if media upload failed or client unavailable
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

	artifactID := createdArtifact.ID
	// Always expose the response-api artifact endpoint for downloads.
	downloadURL = fmt.Sprintf("/v1/artifacts/%s/download", createdArtifact.ID)

	// Create StepOutput with proper Artifact field for ExtractArtifacts to find
	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          artifactID,
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

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    fileName,
			DownloadURL: fmt.Sprintf("/v1/artifacts/%s/download", createdArtifact.ID),
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
	command := fmt.Sprintf("base64 -w0 %q", path)

	callReq := tool.CallRequest{
		Name: "aio_shell_exec",
		Arguments: map[string]interface{}{
			"command": command,
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("shell exec failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("shell exec returned nil result")
	}
	if result.IsError {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = firstTextContent(result.Content)
		}
		return nil, fmt.Errorf("shell exec error: %s", errMsg)
	}

	rawText := firstTextContent(result.Content)
	if rawText == "" {
		return nil, fmt.Errorf("shell exec returned empty content")
	}

	var shellResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(rawText), &shellResult); err != nil {
		cleanedB64 := cleanBase64String(rawText)
		return base64.StdEncoding.DecodeString(cleanedB64)
	}

	if shellResult.ExitCode != 0 {
		return nil, fmt.Errorf("base64 command failed (exit %d): %s", shellResult.ExitCode, shellResult.Stderr)
	}

	cleanedB64 := cleanBase64String(shellResult.Stdout)
	decoded, err := base64.StdEncoding.DecodeString(cleanedB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	return decoded, nil
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
// This ensures LLM calls have full context from research and other preceding steps.
func (e *SlideGeneratorExecutor) buildAccumulatedContext(input agent.ExecutionInput) string {
	var contextParts []string

	// Add accumulated outputs from previous tasks
	for _, output := range input.AccumulatedOutputs {
		if len(output) > 0 {
			extracted := e.extractContextFromOutput(output)
			if extracted != "" {
				contextParts = append(contextParts, extracted)
			}
		}
	}

	// Add current task's previous output (if any)
	if len(input.PreviousOutput) > 0 {
		extracted := e.extractContextFromOutput(input.PreviousOutput)
		if extracted != "" {
			contextParts = append(contextParts, extracted)
		}
	}

	if len(contextParts) == 0 {
		return "[No previous context available]"
	}

	return strings.Join(contextParts, "\n\n---\n\n")
}

// extractContextFromOutput extracts meaningful text content from a step output.
func (e *SlideGeneratorExecutor) extractContextFromOutput(output json.RawMessage) string {
	if len(output) == 0 {
		return ""
	}

	// Try to parse as structured output
	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err == nil {
		// Check for tool result format with "content" array
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

		// Check for LLM response format with "content" string
		if content, ok := parsed["content"].(string); ok && content != "" {
			return content
		}

		// Check for "text" field directly
		if text, ok := parsed["text"].(string); ok && text != "" {
			return text
		}

		// Check for tool output format
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

	// Fallback: return raw output if it's not too long
	rawStr := string(output)
	if len(rawStr) > 10000 {
		return rawStr[:10000] + "... [truncated]"
	}
	return rawStr
}

// buildToolArguments constructs proper tool arguments from step params and context.
// This mirrors the logic from DeepResearchExecutor for consistency.
func (e *SlideGeneratorExecutor) buildToolArguments(toolName string, params map[string]interface{}, input agent.ExecutionInput, description string) (map[string]interface{}, error) {
	switch toolName {
	case "google_search":
		// Priority: params["q"] > params["query"] > description > error
		query := ""
		if q, ok := params["q"].(string); ok && q != "" {
			query = q
		} else if q, ok := params["query"].(string); ok && q != "" {
			// Backward compatibility
			query = q
		} else if description != "" {
			query = description
		}

		if query == "" {
			return nil, fmt.Errorf("no search query provided")
		}
		return map[string]interface{}{
			"q": query,
		}, nil

	case "scrape":
		// Extract URLs from previous search results or params
		urls := e.extractURLsFromPreviousOutput(input.PreviousOutput)
		if len(urls) == 0 {
			// Check if explicitly provided
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
		// MCP scrape tool expects single url parameter
		return map[string]interface{}{
			"url": urls[0],
		}, nil

	case "aio_code_execute":
		var code string
		if codeParam, ok := params["code"].(string); ok {
			code = codeParam
		}
		if code == "" {
			return nil, fmt.Errorf("no code provided for execution")
		}

		code = agent.NormalizeSandboxFilePaths(code)
		language, _ := params["language"].(string)
		if language == "" {
			language = "python"
		}

		return map[string]interface{}{
			"language": language,
			"code":     code,
		}, nil

	default:
		// Generic tool: remove metadata fields
		toolArgs := make(map[string]interface{})
		for k, v := range params {
			// Exclude known metadata fields
			if k != "tool" && k != "description" {
				toolArgs[k] = v
			}
		}
		return toolArgs, nil
	}
}

// extractURLsFromPreviousOutput extracts URLs from search results.
func (e *SlideGeneratorExecutor) extractURLsFromPreviousOutput(previousOutput json.RawMessage) []string {
	if len(previousOutput) == 0 {
		return nil
	}

	var output map[string]interface{}
	if err := json.Unmarshal(previousOutput, &output); err != nil {
		return nil
	}

	var urls []string

	// Check for content array (MCP tool result format)
	if content, ok := output["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					// Parse the text as JSON to extract results
					var textData map[string]interface{}
					if err := json.Unmarshal([]byte(text), &textData); err == nil {
						urls = append(urls, extractURLsFromData(textData)...)
					}
				}
			}
		}
	}

	// Direct extraction from various result formats
	urls = append(urls, extractURLsFromData(output)...)

	return urls
}

// extractURLsFromData extracts URLs from various search result formats.
func extractURLsFromData(data map[string]interface{}) []string {
	var urls []string

	// Serper format: organic[].link
	if organic, ok := data["organic"].([]interface{}); ok {
		for _, item := range organic {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if link, ok := itemMap["link"].(string); ok && link != "" {
					urls = append(urls, link)
				}
			}
		}
	}

	// Generic results format: results[].url or results[].link
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

// isNonCriticalToolForSlides determines if a tool failure should not fail the overall step.
func isNonCriticalToolForSlides(toolName string) bool {
	// Search and scrape are optional for slide generation - we can fall back to LLM knowledge
	nonCriticalTools := map[string]bool{
		"google_search": true,
		"scrape":        true,
	}
	return nonCriticalTools[toolName]
}

// buildSkippedToolResultForSlides creates a result indicating a tool was skipped.
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

// Helper function
func strPtrS(s string) *string {
	return &s
}

// Verify interface compliance at compile time
var _ agent.Executor = (*SlideGeneratorExecutor)(nil)
