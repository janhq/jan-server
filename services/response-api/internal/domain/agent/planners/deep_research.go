// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"encoding/json"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
)

// DeepResearchPlanner creates execution plans for deep research tasks.
type DeepResearchPlanner struct {
	planService plan.Service
}

// NewDeepResearchPlanner creates a new deep research planner.
func NewDeepResearchPlanner(planService plan.Service) *DeepResearchPlanner {
	return &DeepResearchPlanner{
		planService: planService,
	}
}

// Name returns the planner's unique identifier.
func (p *DeepResearchPlanner) Name() string {
	return string(plan.AgentTypeDeepResearch)
}

// CanHandle determines if this planner can handle the given request.
func (p *DeepResearchPlanner) CanHandle(ctx context.Context, request *agent.PlanRequest) bool {
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
func (p *DeepResearchPlanner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	// Check if the request involves code execution
	requiresCodeExecution := p.detectCodeExecutionNeed(request)

	// Determine estimated steps based on whether code execution is needed
	estimatedSteps := 8
	if requiresCodeExecution {
		estimatedSteps = 10 // Add 2 more steps for code execution
	}

	// Create the plan
	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
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

	// Create research steps
	searchParams1, _ := json.Marshal(map[string]interface{}{
		"tool":        "google_search",
		"description": "Primary search query",
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

	searchParams2, _ := json.Marshal(map[string]interface{}{
		"tool":        "google_search",
		"description": "Secondary search query for additional context",
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
		"description": "Extract content from top search results",
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

	synthesisParams, _ := json.Marshal(map[string]interface{}{
		"action":      "reasoning",
		"description": "Cross-reference claims and identify key themes",
	})
	_, err = p.planService.CreateStep(ctx, synthesisTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
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
		codeTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
			Sequence:    taskSequence,
			TaskType:    plan.TaskTypeExecution,
			Title:       "Code Execution",
			Description: strPtr("Generate and execute code to demonstrate concepts"),
		})
		if err != nil {
			return nil, err
		}

		// Step 1: Generate code based on research findings
		codeGenParams, _ := json.Marshal(map[string]interface{}{
			"action":      "generate_code",
			"description": "Generate Python code based on research findings",
		})
		_, err = p.planService.CreateStep(ctx, codeTask.ID, plan.CreateStepParams{
			Sequence:    1,
			Action:      plan.ActionTypeLLMCall,
			InputParams: codeGenParams,
			MaxRetries:  2,
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
			InputParams: codeExecParams,
			MaxRetries:  2,
		})
		if err != nil {
			return nil, err
		}

		taskSequence++
	}

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

	generateParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_content",
		"description": "Write comprehensive analysis",
	})
	_, err = p.planService.CreateStep(ctx, reportTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		InputParams: generateParams,
		MaxRetries:  2,
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

	return result, nil
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// detectCodeExecutionNeed checks if the request involves code generation/execution
func (p *DeepResearchPlanner) detectCodeExecutionNeed(request *agent.PlanRequest) bool {
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

// Verify interface compliance at compile time
var _ agent.Planner = (*DeepResearchPlanner)(nil)

// DeepResearchExecutor executes steps for deep research plans.
type DeepResearchExecutor struct {
	mcpClient MCPClient
}

// MCPClient interface for tool execution - matches tool.MCPClient
type MCPClient interface {
	CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
}

// NewDeepResearchExecutor creates a new deep research executor.
func NewDeepResearchExecutor(mcpClient MCPClient) *DeepResearchExecutor {
	return &DeepResearchExecutor{
		mcpClient: mcpClient,
	}
}

// Execute runs a single step and returns the result.
func (e *DeepResearchExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	switch step.Action {
	case plan.ActionTypeToolCall:
		return e.executeToolCall(ctx, step, input)
	case plan.ActionTypeLLMCall:
		return e.executeLLMCall(ctx, step, input)
	default:
		return &agent.ExecutionResult{
			Status: status.StatusCompleted,
			Output: nil,
		}, nil
	}
}

func (e *DeepResearchExecutor) executeToolCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	toolName, _ := params["tool"].(string)
	if toolName == "" {
		toolName = "google_search"
	}

	// Build the CallRequest for MCP client
	callReq := tool.CallRequest{
		Name:      toolName,
		Arguments: params,
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	// Execute the tool via MCP client
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	output, _ := json.Marshal(result)
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}, nil
}

func (e *DeepResearchExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	// For LLM calls, we defer to the main orchestrator
	// This is a placeholder that will be filled by the orchestrator
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: nil,
	}, nil
}

// CanExecute checks if this executor can handle the given action type.
func (e *DeepResearchExecutor) CanExecute(action plan.ActionType) bool {
	switch action {
	case plan.ActionTypeToolCall, plan.ActionTypeLLMCall:
		return true
	default:
		return false
	}
}

// Rollback attempts to undo a step's effects (optional).
func (e *DeepResearchExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	// Deep research steps are generally not rollback-able
	return nil
}

// Verify interface compliance at compile time
var _ agent.Executor = (*DeepResearchExecutor)(nil)
