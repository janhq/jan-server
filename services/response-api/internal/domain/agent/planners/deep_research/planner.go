package deepresearch

import (
	"context"
	"encoding/json"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// Planner creates execution plans for deep research tasks.
type Planner struct {
	planService plan.Service
}

// NewPlanner creates a new deep research planner.
func NewPlanner(planService plan.Service) *Planner {
	return &Planner{
		planService: planService,
	}
}

// Name returns the planner's unique identifier.
func (p *Planner) Name() string {
	return string(plan.AgentTypeDeepResearch)
}

// CanHandle determines if this planner can handle the given request.
func (p *Planner) CanHandle(ctx context.Context, request *agent.PlanRequest) bool {
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
	return agentTypeStr == string(plan.AgentTypeDeepResearch)
}

// CreatePlan analyzes the request and creates an execution plan.
func (p *Planner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[deep_research] CreatePlan started")
	// Check if the request involves code execution
	requiresCodeExecution := p.detectCodeExecutionNeed(request)
	log.Debug().Bool("requires_code_execution", requiresCodeExecution).Msg("[deep_research] detected code execution need")

	// Determine estimated steps based on whether code execution is needed
	estimatedSteps := 9
	if requiresCodeExecution {
		estimatedSteps = 11 // Add 2 more steps for code execution
	}
	log.Debug().Int("estimated_steps", estimatedSteps).Msg("[deep_research] calculated estimated steps")

	// Create the plan
	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeDeepResearch,
		EstimatedSteps: estimatedSteps,
		Config: &plan.PlanConfig{
			MaxRetries:        3,
			TimeoutPerStep:    300000000000, // 5 minutes in nanoseconds
			EnableFallback:    true,
			UserApproval:      false,
			StreamProgress:    true,
			ArtifactRetention: "session",
		},
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("[deep_research] creating research task")
	// Create Task 1: Research
	researchTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    1,
		TaskType:    plan.TaskTypeResearch,
		Title:       "Research",
		Description: strPtr("Search and gather information from multiple sources"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", researchTask.ID).Msg("[deep_research] research task created")

	// Create research steps with actual user query
	// Extract the user's question/query from the request
	userQuery := request.UserMessage
	if userQuery == "" {
		userQuery = "research query" // Fallback
	}

	searchParams1, _ := json.Marshal(map[string]interface{}{
		"tool":        "google_search",
		"q":           userQuery,
		"description": "Primary search query for main question",
	})
	_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeToolCall,
		Title:       "Search primary sources",
		Description: strPtr("Search the web for the main research question"),
		InputParams: searchParams1,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	// For the second search, use a variant/expansion of the original query
	secondQuery := userQuery + " detailed explanation examples"
	searchParams2, _ := json.Marshal(map[string]interface{}{
		"tool":        "google_search",
		"q":           secondQuery,
		"description": "Secondary search query for additional context",
	})
	_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeToolCall,
		Title:       "Search supporting sources",
		Description: strPtr("Find additional context, explanations, and examples"),
		InputParams: searchParams2,
		MaxRetries:  3,
	})
	if err != nil {
		return nil, err
	}

	scrapeParams, _ := json.Marshal(map[string]interface{}{
		"tool":        "scrape",
		"description": "Extract content from top search results",
	})
	_, err = p.planService.CreateStep(ctx, researchTask.ID, plan.CreateStepParams{
		Sequence:    3,
		Action:      plan.ActionTypeToolCall,
		Title:       "Scrape top sources",
		Description: strPtr("Extract content from the most relevant sources"),
		InputParams: scrapeParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("[deep_research] creating synthesis task")
	// Create Task 2: Synthesis
	synthesisTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    2,
		TaskType:    plan.TaskTypeValidation,
		Title:       "Synthesis",
		Description: strPtr("Cross-reference and synthesize findings"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", synthesisTask.ID).Msg("[deep_research] synthesis task created")

	synthesisParams, _ := json.Marshal(map[string]interface{}{
		"action":      "reasoning",
		"description": "Cross-reference claims and identify key themes",
	})
	_, err = p.planService.CreateStep(ctx, synthesisTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Synthesize findings",
		Description: strPtr("Cross-reference claims and identify key themes"),
		InputParams: synthesisParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	// Track task sequence for dynamic task ordering
	taskSequence := 3

	// Create Task 3: Code Execution (conditional - only if code is requested)
	if requiresCodeExecution {
		log.Debug().Msg("[deep_research] creating code execution task")
		codeTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeExecution,
			Title:       "Code Execution",
			Description: strPtr("Generate and execute code to demonstrate concepts"),
		})
		if err != nil {
			return nil, err
		}
		log.Debug().Str("task_id", codeTask.ID).Msg("[deep_research] code execution task created")

		// Step 1: Generate code based on research findings
		codeGenParams, _ := json.Marshal(map[string]interface{}{
			"action":      "generate_code",
			"description": "Generate Python code based on research findings",
		})
		_, err = p.planService.CreateStep(ctx, codeTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeLLMCall,
			Title:       "Generate code",
			Description: strPtr("Generate Python code based on research findings"),
			InputParams: codeGenParams,
			MaxRetries:  5, // Increased from 2 to handle LLM empty responses
		})
		if err != nil {
			return nil, err
		}

		// Step 2: Execute the generated code using aio_code_execute
		codeExecParams, _ := json.Marshal(map[string]interface{}{
			"tool":        "aio_code_execute",
			"language":    "python",
			"description": "Execute the generated Python code in sandbox",
		})
		_, err = p.planService.CreateStep(ctx, codeTask.ID, plan.CreateStepParams{
			Sequence:    2,
			Action:      plan.ActionTypeToolCall,
			Title:       "Run code",
			Description: strPtr("Execute the generated Python code in the sandbox"),
			InputParams: codeExecParams,
			MaxRetries:  2,
		})
		if err != nil {
			return nil, err
		}

		taskSequence++
	}

	log.Debug().Int("task_sequence", taskSequence).Msg("[deep_research] creating report generation task")
	// Create Task 4 (or 3): Report Generation
	reportTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    taskSequence,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Report",
		Description: strPtr("Generate comprehensive analysis with citations"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", reportTask.ID).Msg("[deep_research] report task created")

	generateParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_content",
		"description": "Write comprehensive analysis",
	})
	_, err = p.planService.CreateStep(ctx, reportTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		Title:       "Write report",
		Description: strPtr("Generate comprehensive analysis with citations"),
		InputParams: generateParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_artifact",
		"description":   "Store report as an artifact",
		"artifact_type": "report",
		"config": map[string]interface{}{
			"format":           "markdown",
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, reportTask.ID, plan.CreateStepParams{
		Sequence:    2,
		Action:      plan.ActionTypeArtifactCreate,
		Title:       "Save report",
		Description: strPtr("Store the report as a downloadable artifact"),
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
		RequiresApproval: false,
	}

	for i := range planWithDetails.Tasks {
		result.Tasks[i] = &planWithDetails.Tasks[i]
	}

	log.Info().
		Str("plan_id", createdPlan.ID).
		Str("response_id", request.ResponseID).
		Int("num_tasks", len(result.Tasks)).
		Bool("requires_code_execution", requiresCodeExecution).
		Int("estimated_steps", estimatedSteps).
		Msg("[deep_research] created deep research plan")

	return result, nil
}

// detectCodeExecutionNeed checks if the request involves code generation/execution.
func (p *Planner) detectCodeExecutionNeed(request *agent.PlanRequest) bool {
	// Check metadata for explicit code execution flag
	if request.Metadata != nil {
		if codeExec, ok := request.Metadata["require_code_execution"].(bool); ok && codeExec {
			return true
		}
	}

	// Check the input for code-related keywords
	input := strings.ToLower(request.UserMessage)
	codeKeywords := []string{
		"python", "script", "code", "program", "execute", "run",
		"aio_code_execute", "aio_shell_exec", "implementation",
		"demonstrate", "example code", "working example",
		"analysis script", "data analysis", "visualization",
	}

	for _, keyword := range codeKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}

	return false
}

// Verify interface compliance at compile time.
var _ agent.Planner = (*Planner)(nil)
