// Package planners contains agent planner implementations.
package planners

import (
	"context"
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
	// Task 5: Finalization & Artifact Creation
	// ============================================
	finalTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Finalize",
		Description: strPtr("Create final presentation artifact"),
	})
	if err != nil {
		return nil, err
	}

	// Compile final presentation
	compileParams, _ := json.Marshal(map[string]interface{}{
		"action":      "compile_presentation",
		"description": "Compile slides into final presentation format",
		"config": map[string]interface{}{
			"format": config.Format,
			"theme":  config.Theme,
		},
	})
	_, err = p.planService.CreateStep(ctx, finalTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
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

	// Parse options from metadata
	if options, ok := request.Metadata["options"].(map[string]interface{}); ok {
		if numSlides, ok := options["num_slides"].(float64); ok {
			config.NumSlides = int(numSlides)
		}
		if theme, ok := options["theme"].(string); ok {
			config.Theme = theme
		}
		if format, ok := options["format"].(string); ok {
			config.Format = format
		}
		if researchDepth, ok := options["research_depth"].(string); ok {
			config.ResearchDepth = researchDepth
		}
		if optionsCount, ok := options["options_count"].(float64); ok {
			config.OptionsCount = int(optionsCount)
		}
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
}

// NewSlideGeneratorExecutor creates a new slide generator executor.
func NewSlideGeneratorExecutor(mcpClient MCPClient, llmProvider LLMProvider, artifactService artifact.Service, mediaClient *media.Client) *SlideGeneratorExecutor {
	return &SlideGeneratorExecutor{
		mcpClient:       mcpClient,
		llmProvider:     llmProvider,
		artifactService: artifactService,
		mediaClient:     mediaClient,
	}
}

// CanExecute checks if this executor can handle the given action type.
func (e *SlideGeneratorExecutor) CanExecute(action plan.ActionType) bool {
	switch action {
	case plan.ActionTypeToolCall, plan.ActionTypeLLMCall, plan.ActionTypeArtifactCreate:
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

	// Build tool arguments from metadata
	arguments := make(map[string]interface{})
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			arguments[k] = v
		}
	}

	// Extract context info
	requestID := ""
	conversationID := ""
	if input.PlanContext != nil {
		requestID = input.PlanContext.ResponseID
		conversationID = input.PlanContext.ConversationID
	}

	// Execute tool
	result, err := e.mcpClient.CallTool(ctx, tool.CallRequest{
		Name:           toolName,
		Arguments:      arguments,
		RequestID:      requestID,
		ConversationID: conversationID,
	})

	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "TOOL_ERROR",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	outputBytes, _ := json.Marshal(result)
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func (e *SlideGeneratorExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	// LLM calls are handled by the orchestrator in the response service
	// This just returns a placeholder - actual LLM execution happens upstream
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: nil,
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

	// Build MediaArtifact for the response - use media ID if uploaded, otherwise artifact ID
	artifactID := createdArtifact.ID
	if mediaID != "" {
		artifactID = mediaID
	}

	// If no download URL from media-api, generate one for the artifact endpoint
	if downloadURL == "" {
		// This will be replaced with actual base URL in production
		downloadURL = fmt.Sprintf("/v1/artifacts/%s/download", createdArtifact.ID)
	}

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

func resolveArtifactContentType(artifactType string, format string) artifact.ContentType {
	switch artifactType {
	case "report":
		return artifact.ContentTypeResearch
	case "document":
		return artifact.ContentTypeDocument
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
	case "markdown":
		return "content.md"
	default:
		if format == "pdf" {
			return "presentation.pdf"
		}
		return "presentation.pptx"
	}
}

// Helper function
func strPtrS(s string) *string {
	return &s
}

// Verify interface compliance at compile time
var _ agent.Executor = (*SlideGeneratorExecutor)(nil)
