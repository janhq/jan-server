// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jan-server/services/response-api/internal/config"
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

//go:embed templates/slide_template.json
var slideTemplateFS embed.FS

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

	templateJSON, err := loadEmbeddedSlideTemplate()
	if err != nil {
		return nil, err
	}

	// Create the plan
	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeSlideGenerator,
		EstimatedSteps: estimatedSteps,
		Config: &plan.PlanConfig{
			MaxRetries:        5,
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
			MaxRetries:  3,
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
			Description: strPtr("Gather information and context for the topic of the presentation"),
		})
		if err != nil {
			return nil, err
		}

		// Step 1: Primary search
		searchParams1, _ := json.Marshal(map[string]interface{}{
			"tool":        "google_search",
			"description": "Search for key ideas related to the topic for the presentation",
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
				MaxRetries:  3,
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
		MaxRetries:  5,
	})
	if err != nil {
		return nil, err
	}

	taskSequence++

	// ============================================
	// Task 3: Slide JSON Generation
	// ============================================
	contentTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Generate Slides JSON",
		Description: strPtr("Generate structured slide JSON based on the template"),
	})
	if err != nil {
		return nil, err
	}

	structureParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_slides_json",
		"description": "Generate structured slide JSON format",
		"template":    templateJSON,
		"config": map[string]interface{}{
			"num_slides": config.NumSlides,
			"theme":      config.Theme,
			"format":     config.Format,
		},
	})
	_, err = p.planService.CreateStep(ctx, contentTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		InputParams: structureParams,
		MaxRetries:  5,
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
			"q":           request.UserMessage + " images",
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

	// Execute slide rendering via code execution
	compileParams, _ := json.Marshal(map[string]interface{}{
		"tool":        "aio_code_execute",
		"language":    "python",
		"description": "Render PPTX from slide JSON",
		"action":      "render_slides_json",
		"spec_path":   fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", request.ResponseID),
		"script_path": fmt.Sprintf("/home/gem/slide_execs/slide_exec_%s.py", request.ResponseID),
		"output_path": fmt.Sprintf("/home/gem/slide_%s.pptx", request.ResponseID),
		"image_url":   "https://www.jan.ai/_next/static/media/cute-robot-flying.1479447f.png",
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeToolCall,
		InputParams: compileParams,
		MaxRetries:  3,
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
		MaxRetries:  3,
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
	steps += 1 // JSON generation

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

func loadEmbeddedSlideTemplate() (string, error) {
	data, err := slideTemplateFS.ReadFile("templates/slide_template.json")
	if err != nil {
		return "", fmt.Errorf("read slide template: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
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
	aioBaseURL      string
}

// NewSlideGeneratorExecutor creates a new slide generator executor.
func NewSlideGeneratorExecutor(mcpClient MCPClient, llmProvider LLMProvider, artifactService artifact.Service, mediaClient *media.Client, skillExecutor *SkillExecutor, cfg *config.Config) *SlideGeneratorExecutor {
	aioBaseURL := ""
	if cfg != nil {
		aioBaseURL = strings.TrimSpace(cfg.AIOURL)
	}
	return &SlideGeneratorExecutor{
		mcpClient:       mcpClient,
		llmProvider:     llmProvider,
		artifactService: artifactService,
		mediaClient:     mediaClient,
		skillExecutor:   skillExecutor,
		aioBaseURL:      aioBaseURL,
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
	log.Info().
		Str("tool", toolName).
		Str("step_id", step.ID).
		Msg("Preparing tool call arguments")
	toolArgs, err := e.buildToolArguments(toolName, params, input, description)
	if err != nil {
		if toolName == "aio_code_execute" {
			log.Warn().
				Str("tool", toolName).
				Err(err).
				Msg("Tool arguments failed, attempting LLM repair")
			if repairedArgs, repairErr := e.buildAioCodeArgsWithRepair(ctx, params, input); repairErr == nil {
				toolArgs = repairedArgs
				err = nil
				log.Info().
					Str("tool", toolName).
					Msg("Tool arguments repaired successfully")
			} else {
				log.Error().
					Str("tool", toolName).
					Err(repairErr).
					Msg("Tool argument repair failed")
			}
		}
	}
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
	var result *tool.Result
	if toolName == "aio_code_execute" {
		var lastErr error
		var lastResult *tool.Result
		for attempt := 0; attempt < 5; attempt++ {
			result, err = e.mcpClient.CallTool(ctx, tool.CallRequest{
				Name:           toolName,
				Arguments:      toolArgs,
				RequestID:      requestID,
				ConversationID: conversationID,
			})
			if result != nil {
				lastResult = result
			}
			if err == nil && result != nil && !result.IsError {
				lastErr = nil
				break
			}
			if err != nil {
				lastErr = err
			} else if result != nil && result.IsError {
				lastErr = fmt.Errorf(strings.TrimSpace(result.Error))
			}
			if attempt == 4 {
				break
			}

			repairedArgs, repairErr := e.buildAioCodeArgsWithRepair(ctx, params, input)
			if repairErr != nil {
				break
			}
			toolArgs = repairedArgs
		}
		if lastErr != nil {
			if lastResult != nil && toolName == "aio_code_execute" {
				logAioCodeExecuteResult(step.ID, lastResult)
			}
			var outputBytes []byte
			if lastResult != nil {
				outputBytes, _ = json.Marshal(lastResult)
			}
			log.Error().
				Err(lastErr).
				Str("tool", toolName).
				Interface("arguments", toolArgs).
				Str("step_id", step.ID).
				Msg("Tool call failed")
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Output: outputBytes,
				Error: &agent.ExecutionError{
					Code:     "TOOL_ERROR",
					Message:  lastErr.Error(),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
	} else {
		result, err = e.mcpClient.CallTool(ctx, tool.CallRequest{
			Name:           toolName,
			Arguments:      toolArgs,
			RequestID:      requestID,
			ConversationID: conversationID,
		})
	}

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
	if result != nil && result.IsError && toolName == "aio_code_execute" {
		logAioCodeExecuteResult(step.ID, result)
	}

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
		prompt = fmt.Sprintf(
			"Analyze and plan the slide structure. %s\n\nResearch findings:\n%s\n\n"+
				"Provide a clear, concise outline for the presentation.\n"+
				"Return plain text only.",
			description, contextData)
	case "generate_slides_json":
		config, _ := params["config"].(map[string]interface{})
		numSlides := 10
		if n, ok := config["num_slides"].(float64); ok {
			numSlides = int(n)
		}
		theme, _ := config["theme"].(string)
		template, _ := params["template"].(string)
		if strings.TrimSpace(template) == "" {
			if embedded, err := loadEmbeddedSlideTemplate(); err == nil {
				template = embedded
			}
		}
		prompt = fmt.Sprintf(
			"Generate an exact %d-slide JSON deck with theme '%s'. %s\n\n"+
				"Return a single JSON object only (no markdown, no backticks, no commentary).\n"+
				"Do not include comments (// or /* */) anywhere in the JSON.\n"+
				"Output must start with '{' and end with '}'.\n"+
				"Top-level keys must be exactly: deck, slides.\n"+
				"Do not use a top-level array. Do not add a 'presentation' wrapper.\n"+
				"Use only slide types shown in the template and keep the same key structure.\n"+
				"If the template has more slides than required, drop slides from the end.\n"+
				"If it has fewer slides, duplicate the last slide structure and update its content.\n"+
				"Each slide must include a 'type' plus all required keys for that type.\n"+
				"Do not include images: omit image, image_pos, logo, icons, and any image URLs.\n"+
				"If the content supports it, include at least one chart slide and one table slide.\n"+
				"Omit optional fields when not used.\n"+
				"Template:\n%s\n\nContext:\n%s",
			numSlides,
			theme,
			description,
			template,
			contextData,
		)
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

	if action == "generate_slides_json" {
		latest := response
		var lastErr error
		templateStr, _ := params["template"].(string)
		for attempt := 0; attempt < 5; attempt++ {
			if err := validateSlideTemplateSchema(latest, templateStr); err == nil {
				response = latest
				lastErr = nil
				break
			} else {
				lastErr = err
			}
			if e.llmProvider == nil {
				break
			}
			fixErrMsg := lastErr.Error()
			if strings.TrimSpace(templateStr) != "" {
				fixErrMsg = fmt.Sprintf("%s. Must match template: %s", fixErrMsg, templateStr)
			}
			fixed, fixErr := e.llmProvider.FixCode(ctx, latest, fixErrMsg, "json")
			if fixErr != nil || strings.TrimSpace(fixed) == "" || fixed == latest {
				break
			}
			latest = fixed
		}
		if lastErr != nil {
			if converted, convErr := convertPresentationToTemplate(latest, params); convErr == nil {
				response = converted
				lastErr = nil
			}
		}
		if lastErr != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "SLIDE_JSON_INVALID",
					Message:  lastErr.Error(),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
		}
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
		if renderOutput := extractSlideRenderOutput(input.PreviousOutput); renderOutput != nil {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadRenderedArtifact(ctx, step, input, renderOutput, artifactType, retentionPolicy)
		}
	}
	if isSlideArtifact {
		if renderOutput, err := e.renderSlidesFromSpec(ctx, input); err == nil && renderOutput != nil {
			retentionPolicy, _ := config["retention_policy"].(string)
			if retentionPolicy == "" {
				retentionPolicy = "session"
			}
			return e.uploadRenderedArtifact(ctx, step, input, renderOutput, artifactType, retentionPolicy)
		} else if err != nil {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error: &agent.ExecutionError{
					Code:     "SLIDE_RENDER_FAILED",
					Message:  fmt.Sprintf("render slides failed: %v", err),
					Severity: status.ErrorSeverityRetryable,
				},
			}, nil
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
	downloadURL = fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID)

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
			DownloadURL: fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID),
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

	stepOutput := &plan.StepOutput{
		Status:    "completed",
		Type:      "artifact_create",
		CreatedAt: time.Now(),
		Artifact: &plan.MediaArtifact{
			ID:          createdArtifact.ID,
			Type:        string(contentType),
			Filename:    fileName,
			DownloadURL: fmt.Sprintf("/responses/v1/artifacts/%s/download", createdArtifact.ID),
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
		} else {
			log.Debug().
				Err(err).
				Str("path", path).
				Str("base_url", e.aioBaseURL).
				Msg("AIO direct download failed; falling back to code execution")
		}
	}

	// Use Python to read and base64-encode the file, which is more reliable than shell base64 for large binary files
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

	// Parse the code execution result
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

	// Parse the Python script output
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
		action, _ := params["action"].(string)
		log.Info().
			Str("tool", "aio_code_execute").
			Str("action", action).
			Interface("params", params).
			Msg("Preparing aio_code_execute arguments")
		if action == "" {
			if _, hasSpec := params["spec_path"]; hasSpec {
				if _, hasOutput := params["output_path"]; hasOutput {
					action = "render_slides_json"
				}
			}
		}
		var code string
		if action == "render_slides_json" {
			slideSpec := e.extractSlideSpecCandidate(input)
			if strings.TrimSpace(slideSpec) == "" {
				log.Error().
					Str("tool", "aio_code_execute").
					Str("action", action).
					Msg("Slide spec missing for render")
				return nil, fmt.Errorf("missing slide JSON in previous output")
			}
			if err := validateSlideTemplateJSON(slideSpec); err != nil {
				if converted, convErr := convertPresentationToTemplate(slideSpec, nil); convErr == nil {
					slideSpec = converted
				} else {
					log.Error().
						Str("tool", "aio_code_execute").
						Str("action", action).
						Err(err).
						Msg("Slide spec failed validation for render")
					return nil, err
				}
			}
			log.Info().
				Str("tool", "aio_code_execute").
				Int("slide_spec_len", len(slideSpec)).
				Msg("Slide spec prepared for render")
			specPath, _ := params["spec_path"].(string)
			scriptPath, _ := params["script_path"].(string)
			outputPath, _ := params["output_path"].(string)
			imageURL, _ := params["image_url"].(string)
			var err error
			code, err = buildSlideRenderCode(slideSpec, specPath, scriptPath, outputPath, imageURL)
			if err != nil {
				return nil, err
			}
			log.Info().
				Str("tool", "aio_code_execute").
				Int("code_len", len(code)).
				Msg("Slide render code built")
		} else if _, hasSpec := params["spec_path"]; hasSpec {
			if _, hasOutput := params["output_path"]; hasOutput {
				slideSpec := e.extractSlideSpecCandidate(input)
				if strings.TrimSpace(slideSpec) == "" {
					return nil, fmt.Errorf("missing slide JSON in previous output")
				}
				if err := validateSlideTemplateJSON(slideSpec); err != nil {
					if converted, convErr := convertPresentationToTemplate(slideSpec, nil); convErr == nil {
						slideSpec = converted
					} else {
						return nil, err
					}
				}
				specPath, _ := params["spec_path"].(string)
				scriptPath, _ := params["script_path"].(string)
				outputPath, _ := params["output_path"].(string)
				imageURL, _ := params["image_url"].(string)
				var err error
				code, err = buildSlideRenderCode(slideSpec, specPath, scriptPath, outputPath, imageURL)
				if err != nil {
					return nil, err
				}
				log.Info().
					Str("tool", "aio_code_execute").
					Int("code_len", len(code)).
					Msg("Slide render code built")
			}
		} else if codeParam, ok := params["code"].(string); ok {
			code = codeParam
		}
		if strings.TrimSpace(code) == "" {
			// Fallback: try to build render code from any JSON output we can find.
			slideSpec := e.extractSlideSpecFromOutputs(input)
			if strings.TrimSpace(slideSpec) == "" {
				slideSpec = strings.TrimSpace(e.extractJSONFromOutputs(input))
			}
			if strings.TrimSpace(slideSpec) != "" {
				specPath, _ := params["spec_path"].(string)
				scriptPath, _ := params["script_path"].(string)
				outputPath, _ := params["output_path"].(string)
				imageURL, _ := params["image_url"].(string)
				if strings.TrimSpace(specPath) == "" || strings.TrimSpace(outputPath) == "" {
					responseID := ""
					if input.PlanContext != nil {
						responseID = input.PlanContext.ResponseID
					}
					if responseID == "" {
						responseID = fmt.Sprintf("slide_%d", time.Now().Unix())
					}
					if strings.TrimSpace(specPath) == "" {
						specPath = fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", responseID)
					}
					if strings.TrimSpace(scriptPath) == "" {
						scriptPath = fmt.Sprintf("/home/gem/slide_execs/slide_exec_%s.py", responseID)
					}
					if strings.TrimSpace(outputPath) == "" {
						outputPath = fmt.Sprintf("/home/gem/slide_%s.pptx", responseID)
					}
				}
				if err := validateSlideTemplateJSON(slideSpec); err != nil {
					if converted, convErr := convertPresentationToTemplate(slideSpec, nil); convErr == nil {
						slideSpec = converted
					}
				}
				if built, buildErr := buildSlideRenderCode(slideSpec, specPath, scriptPath, outputPath, imageURL); buildErr == nil {
					code = built
				}
			}
		}
		if strings.TrimSpace(code) == "" {
			log.Error().
				Str("tool", "aio_code_execute").
				Str("action", action).
				Interface("params", params).
				Msg("No code produced for aio_code_execute")
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

func (e *SlideGeneratorExecutor) buildAioCodeArgsWithRepair(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (map[string]interface{}, error) {
	if e.llmProvider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}

	action, _ := params["action"].(string)
	if action == "" {
		if _, hasSpec := params["spec_path"]; hasSpec {
			if _, hasOutput := params["output_path"]; hasOutput {
				action = "render_slides_json"
			}
		}
	}
	if action != "render_slides_json" {
		return nil, fmt.Errorf("unsupported repair action: %s", action)
	}

	candidate := strings.TrimSpace(e.extractSlideSpecCandidate(input))
	if candidate == "" {
		return nil, fmt.Errorf("missing slide JSON in previous output")
	}

	templateJSON, templateErr := loadEmbeddedSlideTemplate()
	if templateErr != nil {
		return nil, templateErr
	}

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if err := validateSlideTemplateJSON(candidate); err == nil {
			lastErr = nil
			break
		} else {
			lastErr = err
		}

		if converted, convErr := convertPresentationToTemplate(candidate, params); convErr == nil {
			candidate = converted
			lastErr = nil
			break
		}

		fixErrMsg := lastErr.Error()
		if strings.TrimSpace(templateJSON) != "" {
			fixErrMsg = fmt.Sprintf("%s. Must match template: %s", fixErrMsg, templateJSON)
		}
		fixed, fixErr := e.llmProvider.FixCode(ctx, candidate, fixErrMsg, "json")
		if fixErr != nil || strings.TrimSpace(fixed) == "" || fixed == candidate {
			break
		}
		candidate = fixed
	}

	if lastErr != nil {
		return nil, lastErr
	}

	specPath, _ := params["spec_path"].(string)
	scriptPath, _ := params["script_path"].(string)
	outputPath, _ := params["output_path"].(string)
	imageURL, _ := params["image_url"].(string)
	code, err := buildSlideRenderCode(candidate, specPath, scriptPath, outputPath, imageURL)
	if err != nil {
		return nil, err
	}

	language, _ := params["language"].(string)
	if language == "" {
		language = "python"
	}
	return map[string]interface{}{
		"language": language,
		"code":     agent.NormalizeSandboxFilePaths(code),
	}, nil
}

func (e *SlideGeneratorExecutor) extractJSONFromPreviousOutput(previousOutput json.RawMessage) string {
	if len(previousOutput) == 0 {
		return ""
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(previousOutput, &parsed); err != nil {
		return ""
	}
	if content, ok := parsed["content"].(string); ok {
		candidate := strings.TrimSpace(extractJSONFromMarkdown(content))
		if candidate != "" && json.Valid([]byte(candidate)) {
			return candidate
		}
		payload := extractJSONPayload(content)
		return strings.TrimSpace(payload)
	}
	if content, ok := parsed["content"]; ok {
		if raw, err := json.Marshal(content); err == nil {
			candidate := strings.TrimSpace(string(raw))
			if candidate != "" && json.Valid([]byte(candidate)) {
				return candidate
			}
			payload := extractJSONPayload(candidate)
			return strings.TrimSpace(payload)
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) extractJSONFromOutputs(input agent.ExecutionInput) string {
	if content := e.extractJSONFromPreviousOutput(input.PreviousOutput); strings.TrimSpace(content) != "" {
		if isSlideSpecJSON(content) {
			return content
		}
	}
	for i := len(input.AccumulatedOutputs) - 1; i >= 0; i-- {
		content := e.extractJSONFromPreviousOutput(input.AccumulatedOutputs[i])
		if strings.TrimSpace(content) != "" && isSlideSpecJSON(content) {
			return content
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) extractSlideSpecFromOutputs(input agent.ExecutionInput) string {
	candidates := make([]json.RawMessage, 0, 1+len(input.AccumulatedOutputs))
	if len(input.PreviousOutput) > 0 {
		candidates = append(candidates, input.PreviousOutput)
	}
	candidates = append(candidates, input.AccumulatedOutputs...)

	for i := len(candidates) - 1; i >= 0; i-- {
		content, action := extractOutputContentAndAction(candidates[i])
		if action == "generate_slides_json" {
			cleaned := strings.TrimSpace(extractJSONFromMarkdown(content))
			if isSlideSpecJSON(cleaned) {
				return cleaned
			}
			if payload := extractJSONPayload(content); payload != "" && isSlideSpecJSON(payload) {
				return strings.TrimSpace(payload)
			}
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) extractSlideSpecCandidate(input agent.ExecutionInput) string {
	if content := strings.TrimSpace(e.extractSlideSpecFromOutputs(input)); content != "" {
		return content
	}
	return ""
}

func extractOutputContentAndAction(output json.RawMessage) (string, string) {
	if len(output) == 0 {
		return "", ""
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return "", ""
	}
	action, _ := parsed["action"].(string)
	if content, ok := parsed["content"].(string); ok {
		return content, action
	}
	if content, ok := parsed["content"]; ok {
		if raw, err := json.Marshal(content); err == nil {
			return string(raw), action
		}
	}
	return "", action
}

func isSlideSpecJSON(content string) bool {
	cleaned := strings.TrimSpace(extractJSONFromMarkdown(content))
	if !json.Valid([]byte(cleaned)) {
		return false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return false
	}
	if _, ok := payload["deck"]; ok {
		if _, ok := payload["slides"]; ok {
			return true
		}
	}
	if _, ok := payload["presentation"]; ok {
		return true
	}
	if rawSlides, ok := payload["slides"]; ok {
		if slides, ok := rawSlides.([]interface{}); ok && len(slides) > 0 {
			return true
		}
	}
	return false
}

type slideRenderOutput struct {
	OutputPath string `json:"output_path"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	Base64     string `json:"base64,omitempty"`
}

func extractSlideRenderOutput(previousOutput json.RawMessage) *slideRenderOutput {
	if len(previousOutput) == 0 {
		return nil
	}
	var result tool.Result
	if err := json.Unmarshal(previousOutput, &result); err != nil {
		return nil
	}
	for _, item := range result.Content {
		if item.Text == "" {
			continue
		}
		if parsed := parseSlideRenderOutputFromText(item.Text); parsed != nil {
			return parsed
		}
	}
	return nil
}

type codeExecuteResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func logAioCodeExecuteResult(stepID string, result *tool.Result) {
	if result == nil {
		return
	}
	if strings.TrimSpace(result.Error) != "" {
		log.Error().
			Str("tool", "aio_code_execute").
			Str("step_id", stepID).
			Str("error", result.Error).
			Msg("AIO code execute tool error")
	}
	for _, item := range result.Content {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		log.Info().
			Str("tool", "aio_code_execute").
			Str("step_id", stepID).
			Str("raw", text).
			Msg("AIO code execute output")
		var execResult codeExecuteResult
		if err := json.Unmarshal([]byte(text), &execResult); err == nil {
			if strings.TrimSpace(execResult.Stdout) != "" {
				log.Info().
					Str("tool", "aio_code_execute").
					Str("step_id", stepID).
					Str("stdout", execResult.Stdout).
					Msg("AIO code execute stdout")
			}
			if strings.TrimSpace(execResult.Stderr) != "" {
				log.Error().
					Str("tool", "aio_code_execute").
					Str("step_id", stepID).
					Str("stderr", execResult.Stderr).
					Msg("AIO code execute stderr")
			}
		}
	}
}

func parseSlideRenderOutputFromText(text string) *slideRenderOutput {
	cleaned := strings.TrimSpace(extractJSONFromMarkdown(text))
	if cleaned == "" {
		return nil
	}
	if parsed := parseSlideRenderOutputJSON(cleaned); parsed != nil {
		return parsed
	}
	var execResult codeExecuteResult
	if err := json.Unmarshal([]byte(cleaned), &execResult); err == nil {
		stdoutJSON := extractJSONPayload(execResult.Stdout)
		if parsed := parseSlideRenderOutputJSON(stdoutJSON); parsed != nil {
			return parsed
		}
		stderrJSON := extractJSONPayload(execResult.Stderr)
		if parsed := parseSlideRenderOutputJSON(stderrJSON); parsed != nil {
			return parsed
		}
	}
	return nil
}

func parseSlideRenderOutputJSON(payload string) *slideRenderOutput {
	payload = strings.TrimSpace(payload)
	if payload == "" || !json.Valid([]byte(payload)) {
		return nil
	}
	var parsed slideRenderOutput
	if err := json.Unmarshal([]byte(payload), &parsed); err == nil && parsed.OutputPath != "" {
		return &parsed
	}
	return nil
}

func extractJSONPayload(text string) string {
	cleaned := strings.TrimSpace(stripJSONComments(text))
	if cleaned == "" {
		return ""
	}
	if json.Valid([]byte(cleaned)) {
		return cleaned
	}
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		candidate := strings.TrimSpace(cleaned[start : end+1])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}
	lastStart := strings.LastIndex(cleaned, "{")
	if lastStart >= 0 {
		candidate := strings.TrimSpace(cleaned[lastStart:])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}
	return ""
}

func (e *SlideGeneratorExecutor) renderSlidesFromSpec(ctx context.Context, input agent.ExecutionInput) (*slideRenderOutput, error) {
	if e.mcpClient == nil {
		return nil, fmt.Errorf("mcp client not configured")
	}
	slideSpec := e.extractSlideSpecCandidate(input)
	if strings.TrimSpace(slideSpec) == "" {
		return nil, fmt.Errorf("missing slide JSON in previous output")
	}
	templateJSON, tmplErr := loadEmbeddedSlideTemplate()
	if tmplErr != nil {
		return nil, tmplErr
	}
	if err := validateSlideTemplateSchema(slideSpec, templateJSON); err != nil {
		if converted, convErr := convertPresentationToTemplate(slideSpec, map[string]interface{}{"template": templateJSON}); convErr == nil {
			slideSpec = converted
		} else {
			return nil, err
		}
	}

	responseID := ""
	if input.PlanContext != nil {
		responseID = input.PlanContext.ResponseID
	}
	if responseID == "" {
		responseID = fmt.Sprintf("slide_%d", time.Now().Unix())
	}
	specPath := fmt.Sprintf("/home/gem/slide_specs/slide_spec_%s.json", responseID)
	scriptPath := fmt.Sprintf("/home/gem/slide_execs/slide_exec_%s.py", responseID)
	outputPath := fmt.Sprintf("/home/gem/slide_%s.pptx", responseID)
	code, err := buildSlideRenderCode(slideSpec, specPath, scriptPath, outputPath, "")
	if err != nil {
		return nil, err
	}

	args := map[string]interface{}{
		"language": "python",
		"code":     agent.NormalizeSandboxFilePaths(code),
	}
	result, err := e.mcpClient.CallTool(ctx, tool.CallRequest{
		Name:      "aio_code_execute",
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	toolText := ""
	for _, item := range result.Content {
		if item.Text != "" {
			if toolText == "" {
				toolText = item.Text
			} else {
				toolText = toolText + "\n" + item.Text
			}
			log.Info().
				Str("tool", "aio_code_execute").
				Str("stdout", item.Text).
				Msg("AIO code execute output")
			var execResult codeExecuteResult
			if err := json.Unmarshal([]byte(item.Text), &execResult); err == nil {
				if strings.TrimSpace(execResult.Stderr) != "" {
					log.Error().
						Str("tool", "aio_code_execute").
						Str("stderr", execResult.Stderr).
						Msg("AIO code execute stderr")
				}
			}
		}
	}
	if result.IsError {
		if toolText != "" {
			return nil, fmt.Errorf("aio_code_execute failed: %s", toolText)
		}
		if strings.TrimSpace(result.Error) != "" {
			return nil, fmt.Errorf("aio_code_execute failed: %s", result.Error)
		}
		return nil, fmt.Errorf("aio_code_execute failed")
	}
	outputBytes, _ := json.Marshal(result)
	renderOutput := extractSlideRenderOutput(outputBytes)
	if renderOutput == nil {
		if toolText != "" {
			return nil, fmt.Errorf("render output missing; last tool output: %s", toolText)
		}
		return nil, fmt.Errorf("render output missing")
	}
	return renderOutput, nil
}

func validateSlideTemplateJSON(content string) error {
	templateJSON, err := loadEmbeddedSlideTemplate()
	if err != nil {
		return err
	}
	return validateSlideTemplateSchema(content, templateJSON)
}

func convertPresentationToTemplate(content string, params map[string]interface{}) (string, error) {
	cleaned := strings.TrimSpace(extractJSONFromMarkdown(content))
	if !json.Valid([]byte(cleaned)) {
		return "", fmt.Errorf("invalid JSON output")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		var rawSlides []interface{}
		if err := json.Unmarshal([]byte(cleaned), &rawSlides); err == nil {
			return convertSlidesToTemplate(rawSlides, "", params)
		}
		return "", fmt.Errorf("json parse error: %w", err)
	}

	if _, hasDeck := payload["deck"]; hasDeck {
		if _, hasSlides := payload["slides"]; hasSlides {
			return cleaned, nil
		}
	}

	if rawSlides, ok := payload["slides"].([]interface{}); ok && len(rawSlides) > 0 {
		presentationTitle := ""
		if title, ok := payload["presentation_title"].(string); ok {
			presentationTitle = title
		} else if title, ok := payload["title"].(string); ok {
			presentationTitle = title
		}
		return convertSlidesToTemplate(rawSlides, presentationTitle, params)
	}

	presentation, ok := payload["presentation"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unsupported slide JSON shape")
	}

	templateJSON := ""
	if params != nil {
		if templateStr, ok := params["template"].(string); ok {
			templateJSON = strings.TrimSpace(templateStr)
		}
	}
	if templateJSON == "" {
		embedded, err := loadEmbeddedSlideTemplate()
		if err != nil {
			return "", err
		}
		templateJSON = embedded
	}
	if !json.Valid([]byte(templateJSON)) {
		return "", fmt.Errorf("invalid template JSON")
	}

	var templatePayload map[string]interface{}
	if err := json.Unmarshal([]byte(templateJSON), &templatePayload); err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	deck, _ := templatePayload["deck"].(map[string]interface{})

	rawSlides, _ := presentation["slides"].([]interface{})
	if len(rawSlides) == 0 {
		return "", fmt.Errorf("presentation slides are empty")
	}

	presentationTitle, _ := presentation["title"].(string)
	return convertSlidesToTemplateWithDeck(rawSlides, presentationTitle, params, deck)
}

func convertSlidesToTemplate(rawSlides []interface{}, presentationTitle string, params map[string]interface{}) (string, error) {
	templateJSON := ""
	if params != nil {
		if templateStr, ok := params["template"].(string); ok {
			templateJSON = strings.TrimSpace(templateStr)
		}
	}
	if templateJSON == "" {
		embedded, err := loadEmbeddedSlideTemplate()
		if err != nil {
			return "", err
		}
		templateJSON = embedded
	}
	if !json.Valid([]byte(templateJSON)) {
		return "", fmt.Errorf("invalid template JSON")
	}

	var templatePayload map[string]interface{}
	if err := json.Unmarshal([]byte(templateJSON), &templatePayload); err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	deck, _ := templatePayload["deck"].(map[string]interface{})
	return convertSlidesToTemplateWithDeck(rawSlides, presentationTitle, params, deck)
}

func convertSlidesToTemplateWithDeck(rawSlides []interface{}, presentationTitle string, params map[string]interface{}, deck map[string]interface{}) (string, error) {
	if len(rawSlides) == 0 {
		return "", fmt.Errorf("presentation slides are empty")
	}

	numSlides := len(rawSlides)
	if params != nil {
		if config, ok := params["config"].(map[string]interface{}); ok {
			if n, ok := config["num_slides"].(float64); ok && int(n) > 0 {
				numSlides = int(n)
			}
		}
	}

	convertedSlides := make([]interface{}, 0, len(rawSlides))
	for idx, raw := range rawSlides {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := item["title"].(string)
		if title == "" {
			title = fmt.Sprintf("Slide %d", idx+1)
		}
		if title == fmt.Sprintf("Slide %d", idx+1) && presentationTitle != "" {
			title = presentationTitle
		}

		bullets := make([]string, 0)
		if contentList, ok := item["content"].([]interface{}); ok {
			for _, entry := range contentList {
				bullets = append(bullets, fmt.Sprintf("%v", entry))
			}
		} else if contentStr, ok := item["content"].(string); ok && contentStr != "" {
			bullets = append(bullets, contentStr)
		}
		if len(bullets) == 0 {
			if focus, ok := item["contentFocus"].(string); ok && focus != "" {
				bullets = append(bullets, focus)
			}
		}
		if len(bullets) == 0 {
			if keyContent, ok := item["key_content"].([]interface{}); ok {
				for _, entry := range keyContent {
					bullets = append(bullets, fmt.Sprintf("%v", entry))
				}
			} else if keyContentStr, ok := item["key_content"].(string); ok && keyContentStr != "" {
				bullets = append(bullets, keyContentStr)
			}
		}
		if visual, ok := item["visual_suggestion"].(string); ok && visual != "" {
			bullets = append(bullets, "Visual: "+visual)
		}
		if len(bullets) == 0 {
			if rationale, ok := item["flow_rationale"].(string); ok && rationale != "" {
				bullets = append(bullets, rationale)
			}
		}
		if len(bullets) == 0 {
			if purpose, ok := item["purpose"].(string); ok && purpose != "" {
				bullets = append(bullets, purpose)
			}
		}
		if len(bullets) == 0 {
			bullets = append(bullets, "TBD")
		}

		slide := map[string]interface{}{
			"type":    "bullets",
			"title":   title,
			"bullets": bullets,
		}
		convertedSlides = append(convertedSlides, slide)
	}

	for len(convertedSlides) < numSlides {
		convertedSlides = append(convertedSlides, map[string]interface{}{
			"type":    "bullets",
			"title":   fmt.Sprintf("Additional Slide %d", len(convertedSlides)+1),
			"bullets": []string{"TBD"},
		})
	}
	if numSlides > 0 && len(convertedSlides) > numSlides {
		convertedSlides = convertedSlides[:numSlides]
	}

	convertedPayload := map[string]interface{}{
		"deck":   deck,
		"slides": convertedSlides,
	}
	convertedJSON, err := json.Marshal(convertedPayload)
	if err != nil {
		return "", fmt.Errorf("converted json marshal error: %w", err)
	}
	if err := validateSlideTemplateJSON(string(convertedJSON)); err != nil {
		return "", err
	}
	return string(convertedJSON), nil
}

type schemaNode struct {
	requiredKeys map[string]struct{}
	children     map[string]*schemaNode
	arrayItems   map[string]*schemaNode
}

var slideOptionalKeys = map[string]map[string]bool{
	"title": {
		"subtitle": true,
		"logo":     true,
	},
	"section": {
		"subtitle": true,
		"icons":    true,
	},
	"bullets": {
		"image":     true,
		"image_pos": true,
	},
	"chart": {
		"side_bullets": true,
	},
	"quote": {
		"author":    true,
		"image":     true,
		"image_pos": true,
	},
	"closing": {
		"logo": true,
	},
}

func validateSlideTemplateSchema(content string, templateJSON string) error {
	cleaned := strings.TrimSpace(extractJSONFromMarkdown(content))
	if !json.Valid([]byte(cleaned)) {
		return fmt.Errorf("invalid JSON output")
	}

	templateJSON = strings.TrimSpace(templateJSON)
	if templateJSON == "" {
		embedded, err := loadEmbeddedSlideTemplate()
		if err != nil {
			return err
		}
		templateJSON = embedded
	}
	if !json.Valid([]byte(templateJSON)) {
		return fmt.Errorf("invalid template JSON")
	}

	var templatePayload map[string]interface{}
	if err := json.Unmarshal([]byte(templateJSON), &templatePayload); err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		var rawSlides []interface{}
		if err := json.Unmarshal([]byte(cleaned), &rawSlides); err == nil {
			deck, ok := templatePayload["deck"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("template missing deck")
			}
			payload = map[string]interface{}{
				"deck":   deck,
				"slides": rawSlides,
			}
		} else {
			return fmt.Errorf("json parse error: %w", err)
		}
	}

	deck, ok := payload["deck"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing deck object in slide JSON")
	}
	slideSchemaMap, err := buildSlideSchemas(templatePayload)
	if err != nil {
		return err
	}

	if deckSchema, ok := slideSchemaMap["__deck__"]; ok {
		if err := validateWithSchema(deck, deckSchema, "deck"); err != nil {
			return err
		}
	}

	rawSlides, ok := payload["slides"]
	if !ok {
		return fmt.Errorf("missing slides array in slide JSON")
	}
	slides, ok := rawSlides.([]interface{})
	if !ok || len(slides) == 0 {
		return fmt.Errorf("slides must be a non-empty array")
	}

	for i, slide := range slides {
		obj, ok := slide.(map[string]interface{})
		if !ok {
			return fmt.Errorf("slide %d is not an object", i+1)
		}
		rawType, _ := obj["type"].(string)
		if rawType == "" {
			return fmt.Errorf("slide %d missing type", i+1)
		}
		schema, ok := slideSchemaMap[rawType]
		if !ok {
			return fmt.Errorf("slide %d has unsupported type %q", i+1, rawType)
		}
		if err := validateWithSchema(obj, schema, fmt.Sprintf("slides[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func buildSlideSchemas(templatePayload map[string]interface{}) (map[string]*schemaNode, error) {
	schemaMap := make(map[string]*schemaNode)

	templateDeck, ok := templatePayload["deck"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("template missing deck")
	}
	schemaMap["__deck__"] = buildSchemaFromTemplateMap(templateDeck, nil)

	templateSlides, ok := templatePayload["slides"].([]interface{})
	if !ok || len(templateSlides) == 0 {
		return nil, fmt.Errorf("template missing slides")
	}
	for _, entry := range templateSlides {
		slideTemplate, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		slideType, _ := slideTemplate["type"].(string)
		if slideType == "" {
			continue
		}
		optional := slideOptionalKeys[slideType]
		schema := buildSchemaFromTemplateMap(slideTemplate, optional)
		if slideType == "metrics" {
			if metricsSchema, ok := schema.arrayItems["metrics"]; ok {
				delete(metricsSchema.requiredKeys, "note")
			}
		}
		schemaMap[slideType] = schema
	}

	return schemaMap, nil
}

func buildSchemaFromTemplateMap(template map[string]interface{}, optional map[string]bool) *schemaNode {
	node := &schemaNode{
		requiredKeys: map[string]struct{}{},
		children:     map[string]*schemaNode{},
		arrayItems:   map[string]*schemaNode{},
	}
	for key, value := range template {
		if optional == nil || !optional[key] {
			node.requiredKeys[key] = struct{}{}
		}
		switch typed := value.(type) {
		case map[string]interface{}:
			node.children[key] = buildSchemaFromTemplateMap(typed, nil)
		case []interface{}:
			if len(typed) == 0 {
				continue
			}
			if itemMap, ok := typed[0].(map[string]interface{}); ok {
				node.arrayItems[key] = buildSchemaFromTemplateMap(itemMap, nil)
			}
		}
	}
	return node
}

func validateWithSchema(value map[string]interface{}, schema *schemaNode, path string) error {
	for key := range schema.requiredKeys {
		if _, ok := value[key]; !ok {
			return fmt.Errorf("%s missing required field %q", path, key)
		}
	}
	for key, child := range schema.children {
		childValue, ok := value[key]
		if !ok {
			continue
		}
		childMap, ok := childValue.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s.%s must be an object", path, key)
		}
		if err := validateWithSchema(childMap, child, path+"."+key); err != nil {
			return err
		}
	}
	for key, child := range schema.arrayItems {
		childValue, ok := value[key]
		if !ok {
			continue
		}
		childSlice, ok := childValue.([]interface{})
		if !ok {
			return fmt.Errorf("%s.%s must be an array", path, key)
		}
		for idx, entry := range childSlice {
			entryMap, ok := entry.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%s.%s[%d] must be an object", path, key, idx)
			}
			if err := validateWithSchema(entryMap, child, fmt.Sprintf("%s.%s[%d]", path, key, idx)); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildSlideRenderCode(slideSpec string, specPath string, scriptPath string, outputPath string, imageURL string) (string, error) {
	if !json.Valid([]byte(slideSpec)) {
		return "", fmt.Errorf("invalid slide JSON")
	}
	if strings.TrimSpace(specPath) == "" {
		return "", fmt.Errorf("missing spec_path for slide render")
	}
	if strings.TrimSpace(outputPath) == "" {
		return "", fmt.Errorf("missing output_path for slide render")
	}
	if strings.TrimSpace(imageURL) == "" {
		imageURL = "https://www.jan.ai/_next/static/media/cute-robot-flying.1479447f.png"
	}

pythonBodyTemplate := `import subprocess, sys
import os, json, base64
import urllib.request
import shutil
try:
    from pptx import Presentation
    from pptx.enum.shapes import MSO_SHAPE
    from pptx.enum.chart import XL_CHART_TYPE
    from pptx.chart.data import CategoryChartData
    from pptx.enum.text import PP_ALIGN
    from pptx.util import Inches, Pt
    from pptx.dml.color import RGBColor
except Exception:
    subprocess.check_call([sys.executable, "-m", "pip", "install", "python-pptx"])
    from pptx import Presentation
    from pptx.enum.shapes import MSO_SHAPE
    from pptx.enum.chart import XL_CHART_TYPE
    from pptx.chart.data import CategoryChartData
    from pptx.enum.text import PP_ALIGN
    from pptx.util import Inches, Pt
    from pptx.dml.color import RGBColor

assets_dir = "/home/gem/slide_assets"
os.makedirs(assets_dir, exist_ok=True)

logo_path = os.path.join(assets_dir, "jan_logo.png")
hero_path = os.path.join(assets_dir, "girl_sketch.png")

image_url = "%s"
try:
    urllib.request.urlretrieve(image_url, logo_path)
    shutil.copyfile(logo_path, hero_path)
except Exception:
    logo_path = ""
    hero_path = ""

spec_path = "%s"
with open(spec_path, "r", encoding="utf-8") as f:
    slide_spec = json.load(f)
deck = slide_spec.get("deck", {})
slides = slide_spec.get("slides", [])
file_name = os.path.basename("%s")

prs = Presentation()
size = deck.get("size", {})
width = size.get("width", 13.33)
height = size.get("height", 7.5)
prs.slide_width = Inches(float(width))
prs.slide_height = Inches(float(height))

navy = RGBColor(10, 18, 33)
teal = RGBColor(18, 112, 124)
cream = RGBColor(245, 241, 233)
ink = RGBColor(20, 24, 32)

def color_from_hex(value, fallback):
    if not value or not isinstance(value, str):
        return fallback
    value = value.lstrip("#")
    if len(value) != 6:
        return fallback
    try:
        r = int(value[0:2], 16)
        g = int(value[2:4], 16)
        b = int(value[4:6], 16)
        return RGBColor(r, g, b)
    except Exception:
        return fallback

theme = deck.get("theme", {})
title_bg = color_from_hex(theme.get("title_bg"), navy)
title_text = color_from_hex(theme.get("title_text"), cream)
header_bg = color_from_hex(theme.get("header_bg"), teal)
header_text = color_from_hex(theme.get("header_text"), cream)
body_bg = color_from_hex(theme.get("body_bg"), cream)
body_text = color_from_hex(theme.get("body_text"), ink)
closing_bg = color_from_hex(theme.get("closing_bg"), navy)
closing_text = color_from_hex(theme.get("closing_text"), cream)

asset_lookup = {}
if logo_path:
    asset_lookup["logo"] = logo_path
if hero_path:
    asset_lookup["hero"] = hero_path

def resolve_asset(value):
    if not value:
        return ""
    return asset_lookup.get(value, value)

def set_bg(slide, color):
    fill = slide.background.fill
    fill.solid()
    fill.fore_color.rgb = color

def add_header(slide, title, color_bg, color_text):
    bar = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, Inches(0), Inches(0), Inches(13.33), Inches(0.7))
    bar.fill.solid()
    bar.fill.fore_color.rgb = color_bg
    bar.line.fill.background()
    box = slide.shapes.add_textbox(Inches(0.6), Inches(0.1), Inches(11.5), Inches(0.5))
    tf = box.text_frame
    tf.clear()
    p = tf.paragraphs[0]
    run = p.add_run()
    run.text = title
    run.font.size = Pt(24)
    run.font.bold = True
    run.font.color.rgb = color_text

def add_textbox(slide, text, x, y, w, h, size, color, bold=False, align="left"):
    box = slide.shapes.add_textbox(Inches(x), Inches(y), Inches(w), Inches(h))
    tf = box.text_frame
    tf.clear()
    p = tf.paragraphs[0]
    run = p.add_run()
    run.text = text
    run.font.size = Pt(size)
    run.font.bold = bold
    run.font.color.rgb = color
    if align == "center":
        p.alignment = PP_ALIGN.CENTER
    elif align == "right":
        p.alignment = PP_ALIGN.RIGHT
    else:
        p.alignment = PP_ALIGN.LEFT
    return tf

def apply_text_color(paragraph, color):
    if paragraph.runs:
        for run in paragraph.runs:
            run.font.color.rgb = color
    else:
        paragraph.font.color.rgb = color

def apply_font_size(paragraph, size):
    if paragraph.runs:
        for run in paragraph.runs:
            run.font.size = size
    else:
        paragraph.font.size = size

def add_logo(slide, logo_key, default_x, default_y, default_w):
    logo_value = resolve_asset(logo_key)
    if not logo_value:
        return
    slide.shapes.add_picture(logo_value, Inches(default_x), Inches(default_y), width=Inches(default_w))

def add_bullets_box(slide, bullets, x, y, w, h, size, color):
    box = slide.shapes.add_textbox(Inches(x), Inches(y), Inches(w), Inches(h))
    tf = box.text_frame
    tf.clear()
    for idx, bullet in enumerate(bullets):
        # Handle both string bullets and object bullets with a "text" field
        if isinstance(bullet, dict):
            text = str(bullet.get("text", bullet.get("content", "")))
        else:
            text = str(bullet)
        text = text.replace("{file_name}", file_name)
        if idx == 0:
            tf.text = text
        else:
            p = tf.add_paragraph()
            p.text = text
            p.level = 0
    for paragraph in tf.paragraphs:
        apply_font_size(paragraph, Pt(size))
        apply_text_color(paragraph, color)
    return tf

def add_icon_row(slide, icons, x, y, size, gap, text_color):
    if not icons:
        return
    for idx, icon in enumerate(icons):
        icon_x = x + idx * (size + gap)
        icon_color = color_from_hex(icon.get("color"), header_bg)
        shape = slide.shapes.add_shape(
            MSO_SHAPE.OVAL, Inches(icon_x), Inches(y), Inches(size), Inches(size)
        )
        shape.fill.solid()
        shape.fill.fore_color.rgb = icon_color
        shape.line.fill.background()
        label = str(icon.get("text", ""))
        if label:
            tf = shape.text_frame
            tf.clear()
            p = tf.paragraphs[0]
            run = p.add_run()
            run.text = label
            run.font.size = Pt(12)
            run.font.bold = True
            run.font.color.rgb = text_color
            p.alignment = PP_ALIGN.CENTER

def add_table(slide, table_spec, text_color):
    headers = table_spec.get("headers", [])
    rows = table_spec.get("rows", [])
    cols = len(headers) if headers else (len(rows[0]) if rows else 0)
    if cols == 0:
        return
    row_count = len(rows) + 1 if headers else len(rows)
    x = float(table_spec.get("x", 0.8))
    y = float(table_spec.get("y", 1.6))
    w = float(table_spec.get("w", 11.5))
    h = float(table_spec.get("h", 4.6))
    table = slide.shapes.add_table(row_count, cols, Inches(x), Inches(y), Inches(w), Inches(h)).table
    start_row = 0
    if headers:
        for col_idx, header in enumerate(headers):
            cell = table.cell(0, col_idx)
            cell.text = str(header)
            for paragraph in cell.text_frame.paragraphs:
                apply_font_size(paragraph, Pt(14))
                apply_text_color(paragraph, text_color)
                paragraph.runs[0].font.bold = True
        start_row = 1
    for row_idx, row in enumerate(rows):
        for col_idx, value in enumerate(row):
            cell = table.cell(start_row + row_idx, col_idx)
            cell.text = str(value)
            for paragraph in cell.text_frame.paragraphs:
                apply_font_size(paragraph, Pt(12))
                apply_text_color(paragraph, text_color)

def add_chart(slide, chart_spec):
    chart_type_map = {
        "column": XL_CHART_TYPE.COLUMN_CLUSTERED,
        "bar": XL_CHART_TYPE.BAR_CLUSTERED,
        "line": XL_CHART_TYPE.LINE_MARKERS,
        "pie": XL_CHART_TYPE.PIE,
        "area": XL_CHART_TYPE.AREA,
    }
    chart_type = chart_type_map.get(str(chart_spec.get("type", "column")).lower(), XL_CHART_TYPE.COLUMN_CLUSTERED)
    categories = chart_spec.get("categories", [])
    series = chart_spec.get("series", [])
    if not categories or not series:
        return
    chart_data = CategoryChartData()
    chart_data.categories = categories
    for entry in series:
        chart_data.add_series(entry.get("name", "Series"), entry.get("values", []))
    x = float(chart_spec.get("x", 0.9))
    y = float(chart_spec.get("y", 1.6))
    w = float(chart_spec.get("w", 6.5))
    h = float(chart_spec.get("h", 4.2))
    chart = slide.shapes.add_chart(chart_type, Inches(x), Inches(y), Inches(w), Inches(h), chart_data).chart
    chart.has_legend = bool(chart_spec.get("show_legend", False))

def normalize_bullets(item):
    bullets = item.get("bullets")
    if isinstance(bullets, list) and bullets:
        return [str(b) for b in bullets]
    if isinstance(bullets, str) and bullets:
        return [bullets]
    content = item.get("content")
    if isinstance(content, list) and content:
        return [str(b) for b in content]
    if isinstance(content, str) and content:
        return [content]
    focus = item.get("contentFocus")
    if isinstance(focus, str) and focus:
        return [focus]
    purpose = item.get("purpose")
    if isinstance(purpose, str) and purpose:
        return [purpose]
    return []

blank_layout = prs.slide_layouts[6]

for item in slides:
    slide = None
    slide_type = str(item.get("type", "bullets")).lower()
    if slide_type == "title":
        title_layout = prs.slide_layouts[0]
        slide = prs.slides.add_slide(title_layout)
        set_bg(slide, color_from_hex(item.get("bg"), title_bg))
        slide.shapes.title.text = item.get("title", "")
        slide.placeholders[1].text = item.get("subtitle", "")
        title_para = slide.shapes.title.text_frame.paragraphs[0]
        subtitle_para = slide.placeholders[1].text_frame.paragraphs[0]
        apply_text_color(title_para, color_from_hex(item.get("title_text"), title_text))
        apply_text_color(subtitle_para, color_from_hex(item.get("subtitle_text"), title_text))
        add_logo(slide, item.get("logo", "logo"), 11.2, 0.2, 1.6)
        extra_lines = normalize_bullets(item)
        if extra_lines:
            add_textbox(
                slide,
                "\n".join(extra_lines),
                1.0,
                4.2,
                11.0,
                1.6,
                18,
                color_from_hex(item.get("text"), title_text),
            )
    elif slide_type == "section":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        add_textbox(
            slide,
            item.get("title", ""),
            1.0,
            2.5,
            11.3,
            1.0,
            48,
            color_from_hex(item.get("title_text"), body_text),
            bold=True,
            align="center",
        )
        subtitle = item.get("subtitle", "")
        if subtitle:
            add_textbox(
                slide,
                subtitle,
                2.0,
                3.6,
                9.3,
                0.6,
                22,
                color_from_hex(item.get("subtitle_text"), body_text),
                align="center",
            )
        extra_lines = normalize_bullets(item)
        if extra_lines:
            add_textbox(
                slide,
                "\n".join(extra_lines),
                1.0,
                4.4,
                11.3,
                1.3,
                18,
                color_from_hex(item.get("text"), body_text),
                align="center",
            )
    elif slide_type == "bullets":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        add_bullets_box(
            slide,
            item.get("bullets", []),
            float(item.get("bullets_x", 0.8)),
            float(item.get("bullets_y", 1.3)),
            float(item.get("bullets_w", 6.0)),
            float(item.get("bullets_h", 5.5)),
            int(item.get("bullets_size", 20)),
            color_from_hex(item.get("text"), body_text),
        )
        image_key = resolve_asset(item.get("image", ""))
        if image_key:
            image_pos = item.get("image_pos", {})
            image_x = float(image_pos.get("x", 7.2))
            image_y = float(image_pos.get("y", 1.4))
            image_w = float(image_pos.get("w", 5.4))
            image_h = image_pos.get("h")
            if image_h:
                slide.shapes.add_picture(image_key, Inches(image_x), Inches(image_y), Inches(image_w), Inches(float(image_h)))
            else:
                slide.shapes.add_picture(image_key, Inches(image_x), Inches(image_y), width=Inches(image_w))
    elif slide_type == "metrics":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        metrics = item.get("metrics", [])
        box_w = float(item.get("metric_w", 3.6))
        box_h = float(item.get("metric_h", 1.9))
        start_x = float(item.get("metric_x", 0.8))
        start_y = float(item.get("metric_y", 1.8))
        gap = float(item.get("metric_gap", 0.4))
        for idx, metric in enumerate(metrics):
            x = start_x + idx * (box_w + gap)
            shape = slide.shapes.add_shape(
                MSO_SHAPE.ROUNDED_RECTANGLE, Inches(x), Inches(start_y), Inches(box_w), Inches(box_h)
            )
            shape.fill.solid()
            shape.fill.fore_color.rgb = color_from_hex(metric.get("color"), header_bg)
            shape.line.fill.background()
            tf = shape.text_frame
            tf.clear()
            p_value = tf.paragraphs[0]
            run_value = p_value.add_run()
            run_value.text = str(metric.get("value", ""))
            run_value.font.size = Pt(28)
            run_value.font.bold = True
            run_value.font.color.rgb = color_from_hex(metric.get("text"), title_text)
            p_value.alignment = PP_ALIGN.CENTER
            p_label = tf.add_paragraph()
            p_label.text = str(metric.get("label", ""))
            p_label.alignment = PP_ALIGN.CENTER
            apply_font_size(p_label, Pt(14))
            apply_text_color(p_label, color_from_hex(metric.get("text"), title_text))
            note = str(metric.get("note", ""))
            if note:
                p_note = tf.add_paragraph()
                p_note.text = note
                p_note.alignment = PP_ALIGN.CENTER
                apply_font_size(p_note, Pt(12))
                apply_text_color(p_note, color_from_hex(metric.get("text"), title_text))
    elif slide_type == "chart":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        add_chart(slide, item.get("chart", {}))
        side_bullets = item.get("side_bullets", [])
        if side_bullets:
            add_bullets_box(
                slide,
                side_bullets,
                float(item.get("side_x", 7.2)),
                float(item.get("side_y", 1.6)),
                float(item.get("side_w", 5.2)),
                float(item.get("side_h", 4.2)),
                int(item.get("side_size", 16)),
                color_from_hex(item.get("text"), body_text),
            )
    elif slide_type == "table":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        add_table(slide, item.get("table", {}), color_from_hex(item.get("text"), body_text))
    elif slide_type == "comparison":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        left = item.get("left", {})
        right = item.get("right", {})
        add_textbox(
            slide,
            left.get("title", ""),
            0.9,
            1.4,
            5.6,
            0.4,
            18,
            color_from_hex(item.get("text"), body_text),
            bold=True,
        )
        add_bullets_box(
            slide,
            left.get("bullets", []),
            0.9,
            1.9,
            5.6,
            4.8,
            16,
            color_from_hex(item.get("text"), body_text),
        )
        add_textbox(
            slide,
            right.get("title", ""),
            6.8,
            1.4,
            5.6,
            0.4,
            18,
            color_from_hex(item.get("text"), body_text),
            bold=True,
        )
        add_bullets_box(
            slide,
            right.get("bullets", []),
            6.8,
            1.9,
            5.6,
            4.8,
            16,
            color_from_hex(item.get("text"), body_text),
        )
    elif slide_type == "timeline":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        timeline = item.get("timeline", {})
        items = timeline.get("items", [])
        start_x = float(timeline.get("x", 1.0))
        start_y = float(timeline.get("y", 3.6))
        width = float(timeline.get("w", 11.0))
        line_h = float(timeline.get("line_h", 0.05))
        dot_size = float(timeline.get("dot_size", 0.2))
        if items:
            line = slide.shapes.add_shape(
                MSO_SHAPE.RECTANGLE, Inches(start_x), Inches(start_y), Inches(width), Inches(line_h)
            )
            line.fill.solid()
            line.fill.fore_color.rgb = color_from_hex(timeline.get("line_color"), header_bg)
            line.line.fill.background()
            spacing = width / max(len(items) - 1, 1)
            for idx, entry in enumerate(items):
                dot_x = start_x + idx * spacing - (dot_size / 2)
                dot = slide.shapes.add_shape(
                    MSO_SHAPE.OVAL, Inches(dot_x), Inches(start_y - (dot_size / 2)), Inches(dot_size), Inches(dot_size)
                )
                dot.fill.solid()
                dot.fill.fore_color.rgb = color_from_hex(entry.get("color"), header_bg)
                dot.line.fill.background()
                label = str(entry.get("label", ""))
                if label:
                    add_textbox(
                        slide,
                        label,
                        dot_x - 0.6,
                        start_y + 0.3,
                        1.2,
                        0.4,
                        12,
                        color_from_hex(item.get("text"), body_text),
                        align="center",
                    )
    elif slide_type == "quote":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), body_bg))
        title_value = item.get("title", "")
        if title_value:
            add_header(
                slide,
                title_value,
                color_from_hex(item.get("header_bg"), header_bg),
                color_from_hex(item.get("header_text"), header_text),
            )
        quote_text = item.get("quote", "")
        if quote_text:
            add_textbox(
                slide,
                quote_text,
                0.9,
                1.6,
                8.0,
                2.2,
                26,
                color_from_hex(item.get("text"), body_text),
                bold=True,
            )
        author = item.get("author", "")
        if author:
            add_textbox(
                slide,
                author,
                0.9,
                3.9,
                6.0,
                0.4,
                16,
                color_from_hex(item.get("text"), body_text),
            )
        image_key = resolve_asset(item.get("image", ""))
        if image_key:
            image_pos = item.get("image_pos", {})
            image_x = float(image_pos.get("x", 9.4))
            image_y = float(image_pos.get("y", 1.6))
            image_w = float(image_pos.get("w", 3.2))
            slide.shapes.add_picture(image_key, Inches(image_x), Inches(image_y), width=Inches(image_w))
    elif slide_type == "closing":
        slide = prs.slides.add_slide(blank_layout)
        set_bg(slide, color_from_hex(item.get("bg"), closing_bg))
        header_value = item.get("header", "")
        if header_value:
            add_header(
                slide,
                header_value,
                color_from_hex(item.get("header_bg"), closing_bg),
                color_from_hex(item.get("header_text"), closing_text),
            )
        add_textbox(
            slide,
            item.get("body", ""),
            1.0,
            2.2,
            9.5,
            2.0,
            36,
            color_from_hex(item.get("text"), closing_text),
            bold=True,
        )
        add_logo(slide, item.get("logo", "logo"), 10.8, 5.8, 2.0)

    if slide is not None:
        icons = item.get("icons", [])
        if icons:
            add_icon_row(
                slide,
                icons,
                float(item.get("icons_x", 10.4)),
                float(item.get("icons_y", 6.6)),
                float(item.get("icons_size", 0.3)),
                float(item.get("icons_gap", 0.12)),
                color_from_hex(item.get("icons_text"), title_text),
            )

prs.save(r"%s")
with open(r"%s", "rb") as f:
    encoded = base64.b64encode(f.read()).decode("ascii")
print(json.dumps({
    "output_path": r"%s",
    "file_name": file_name,
    "mime_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    "base64": encoded
}))
`

	codeBody := fmt.Sprintf(pythonBodyTemplate, imageURL, specPath, outputPath, outputPath, outputPath, outputPath)
	specB64 := base64.StdEncoding.EncodeToString([]byte(slideSpec))
	scriptB64 := base64.StdEncoding.EncodeToString([]byte(codeBody))
	pythonExecTemplate := `import base64, os

spec_b64 = "%s"
script_b64 = "%s"
spec_path = "%s"
script_path = "%s"

os.makedirs(os.path.dirname(spec_path), exist_ok=True)
os.makedirs(os.path.dirname(script_path), exist_ok=True)

spec_payload = base64.b64decode(spec_b64).decode("utf-8")
with open(spec_path, "w", encoding="utf-8") as f:
    f.write(spec_payload)

with open(script_path, "wb") as f:
    f.write(base64.b64decode(script_b64))

%s
`
	return fmt.Sprintf(pythonExecTemplate, specB64, scriptB64, specPath, scriptPath, codeBody), nil
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
