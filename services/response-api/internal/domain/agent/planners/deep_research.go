// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
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
	mcpClient   MCPClient
	llmProvider LLMProvider
}

// MCPClient interface for tool execution - matches tool.MCPClient
type MCPClient interface {
	CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
}

// LLMProvider interface for LLM calls to fix code and generate content.
type LLMProvider interface {
	FixCode(ctx context.Context, code string, errorMsg string, language string) (string, error)
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateWithModel(ctx context.Context, prompt string, model string) (string, error)
	GenerateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error)
	GenerateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error)
}

// MaxInstallRetries is the maximum number of package install retry attempts.
const MaxInstallRetries = 3

// MaxCodeFixRetries is the maximum number of LLM code fix retry attempts.
const MaxCodeFixRetries = 5

// NewDeepResearchExecutor creates a new deep research executor.
func NewDeepResearchExecutor(mcpClient MCPClient, llmProvider LLMProvider) *DeepResearchExecutor {
	return &DeepResearchExecutor{
		mcpClient:   mcpClient,
		llmProvider: llmProvider,
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
	return e.executeToolCallWithRetry(ctx, step, input, 0, nil, nil, 0)
}

// codeExecutionState tracks the state of code execution retries.
type codeExecutionState struct {
	originalCode      string
	currentCode       string
	installedPackages []string
	installRetryCount int
	codeFixRetryCount int
	executionErrors   []string
}

// executeToolCallWithRetry executes a tool call with automatic package installation and LLM code fix retry.
func (e *DeepResearchExecutor) executeToolCallWithRetry(ctx context.Context, step *plan.Step, input agent.ExecutionInput, installRetryCount int, installedPackages []string, currentCode *string, codeFixRetryCount int) (*agent.ExecutionResult, error) {
	log.Debug().
		Str("step_id", step.ID).
		Int("install_retry_count", installRetryCount).
		Int("code_fix_retry_count", codeFixRetryCount).
		Int("installed_packages_count", len(installedPackages)).
		Msg("[deep_research] executeToolCallWithRetry started")
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
		log.Error().Err(err).Msg("[deep_research] failed to parse step parameters")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	// Extract tool name from metadata
	toolName, _ := params["tool"].(string)
	if toolName == "" {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "missing_tool_name",
				Message:  "no tool name specified in params",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	description, _ := params["description"].(string)

	log.Debug().Str("tool_name", toolName).Str("description", description).Msg("[deep_research] building tool arguments")
	// Build actual tool arguments (strip metadata fields)
	toolArgs, err := e.buildToolArguments(toolName, params, input, description, currentCode)
	if err != nil {
		log.Error().Err(err).Str("tool_name", toolName).Msg("[deep_research] failed to build tool arguments")
		if isNonCriticalTool(toolName) {
			return buildSkippedToolResult(toolName, err.Error(), "invalid_arguments"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Code:     "invalid_tool_arguments",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Build the CallRequest for MCP client
	callReq := tool.CallRequest{
		Name:      toolName,
		Arguments: toolArgs, // Clean arguments without metadata
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	log.Info().
		Str("tool", toolName).
		Interface("arguments", toolArgs).
		Str("step_id", step.ID).
		Msg("[deep_research] executing tool call")

	// Check if this is a code execution tool
	isCodeExecTool := toolName == "aio_code_execute" || toolName == "aio_shell_exec"
	log.Debug().Bool("is_code_exec_tool", isCodeExecTool).Msg("[deep_research] tool type determined")

	// Execute the tool via MCP client
	result, err := e.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("tool", toolName).
			Interface("arguments", toolArgs).
			Str("step_id", step.ID).
			Msg("Tool call failed")

		if isNonCriticalTool(toolName) && !isCodeExecTool {
			return buildSkippedToolResult(toolName, err.Error(), "tool_call_failed"), nil
		}
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	log.Info().
		Str("tool", toolName).
		Str("step_id", step.ID).
		Bool("is_error", result.IsError).
		Msg("[deep_research] tool call completed")

	if isNonCriticalTool(toolName) && !isCodeExecTool && result != nil && result.IsError {
		log.Debug().Str("tool", toolName).Msg("[deep_research] skipping non-critical tool error")
		return buildSkippedToolResult(toolName, "tool reported error", "tool_error"), nil
	}

	// Check for errors in code execution results
	// The tool might return is_error=false but still have errors in the content
	hasError := result.IsError || (isCodeExecTool && e.hasErrorInResult(result))
	log.Debug().Bool("has_error", hasError).Bool("is_code_exec_tool", isCodeExecTool).Msg("[deep_research] error check completed")

	// Handle errors for code execution tools
	if hasError && isCodeExecTool {
		errorText := e.extractErrorText(result)
		codeErr := agent.ParseCodeExecutionError(errorText)

		// Strategy 1: Try installing missing packages first
		if agent.IsRetryableWithInstall(codeErr) && installRetryCount < MaxInstallRetries {
			packageName := agent.ResolvePackageName(codeErr.ModuleName)

			// Check if we already tried installing this package
			alreadyInstalled := false
			for _, pkg := range installedPackages {
				if pkg == packageName {
					alreadyInstalled = true
					break
				}
			}

			if !alreadyInstalled {
				log.Info().
					Str("module", codeErr.ModuleName).
					Str("package", packageName).
					Int("install_retry", installRetryCount+1).
					Msg("[deep_research] auto-installing missing package and retrying code execution")

				// Install the missing package (don't record as step output)
				installResult, installErr := e.installPackage(ctx, packageName, input)
				if installErr == nil && !installResult.IsError {
					log.Debug().Str("package", packageName).Msg("[deep_research] package installed successfully, retrying")
					// Package installed successfully, retry the original tool call
					newInstalledPackages := append(installedPackages, packageName)
					return e.executeToolCallWithRetry(ctx, step, input, installRetryCount+1, newInstalledPackages, currentCode, codeFixRetryCount)
				}

				log.Warn().
					Str("package", packageName).
					Msg("[deep_research] package installation failed, trying LLM code fix")
			}
		}

		// Strategy 2: Use LLM to fix the code
		// Always try LLM fix if package install didn't work or wasn't applicable
		if e.llmProvider != nil && codeFixRetryCount < MaxCodeFixRetries {
			originalCode := ""
			if currentCode != nil {
				originalCode = *currentCode
			} else if code, ok := params["code"].(string); ok {
				originalCode = code
			}

			if originalCode != "" {
				language, _ := params["language"].(string)
				if language == "" {
					language = "python"
				}

				errorType := getErrorType(errorText)
				log.Info().
					Str("tool", toolName).
					Int("code_fix_retry", codeFixRetryCount+1).
					Int("max_retries", MaxCodeFixRetries).
					Str("error_type", errorType).
					Int("original_code_length", len(originalCode)).
					Msg("[deep_research] attempting LLM-based code fix")

				fixedCode, fixErr := e.llmProvider.FixCode(ctx, originalCode, errorText, language)
				if fixErr == nil && fixedCode != "" && fixedCode != originalCode {
					log.Info().
						Int("original_len", len(originalCode)).
						Int("fixed_len", len(fixedCode)).
						Str("error_type", errorType).
						Msg("[deep_research] LLM generated fixed code, retrying execution")

					return e.executeToolCallWithRetry(ctx, step, input, installRetryCount, installedPackages, &fixedCode, codeFixRetryCount+1)
				}

				if fixErr != nil {
					log.Warn().
						Err(fixErr).
						Int("retry", codeFixRetryCount+1).
						Msg("[deep_research] LLM code fix attempt failed")
				} else if fixedCode == originalCode {
					log.Warn().
						Int("retry", codeFixRetryCount+1).
						Msg("[deep_research] LLM returned same code, no fix applied")
				}
			}
		} else if e.llmProvider == nil {
			log.Warn().Msg("[deep_research] LLM provider not configured, cannot attempt code fix")
		} else {
			log.Warn().
				Int("code_fix_retries", codeFixRetryCount).
				Int("max_retries", MaxCodeFixRetries).
				Msg("[deep_research] code fix retry limit reached")
		}
	}

	// Build final output with retry information
	outputMap := make(map[string]interface{})
	if err := json.Unmarshal(mustMarshal(result), &outputMap); err == nil {
		// Add retry metadata
		if len(installedPackages) > 0 {
			outputMap["installed_packages"] = installedPackages
		}
		if codeFixRetryCount > 0 {
			outputMap["code_fix_attempts"] = codeFixRetryCount
		}
		if installRetryCount > 0 {
			outputMap["install_attempts"] = installRetryCount
		}
	}

	output, _ := json.Marshal(outputMap)

	// Determine if execution failed (check both IsError flag and content)
	executionFailed := result.IsError || (isCodeExecTool && e.hasErrorInResult(result))
	if executionFailed {
		errorMsg := e.extractErrorText(result)

		// Determine severity based on retry exhaustion
		// If we've exhausted all inner retries (install + LLM fix), mark as fatal
		// to prevent redundant outer orchestrator retries
		severity := status.ErrorSeverityRetryable
		if isCodeExecTool && installRetryCount >= MaxInstallRetries && codeFixRetryCount >= MaxCodeFixRetries {
			log.Warn().
				Int("install_retries", installRetryCount).
				Int("code_fix_retries", codeFixRetryCount).
				Msg("All code execution retry strategies exhausted, marking as fatal")
			severity = status.ErrorSeverityFatal
		}

		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Output: output,
			Error:  &agent.ExecutionError{Message: errorMsg, Severity: severity},
		}, nil
	}
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}, nil
}

// getErrorType extracts a simple error type from the error text.
func getErrorType(errorText string) string {
	if strings.Contains(errorText, "SyntaxError") {
		return "SyntaxError"
	}
	if strings.Contains(errorText, "ModuleNotFoundError") {
		return "ModuleNotFoundError"
	}
	if strings.Contains(errorText, "ImportError") {
		return "ImportError"
	}
	if strings.Contains(errorText, "NameError") {
		return "NameError"
	}
	if strings.Contains(errorText, "TypeError") {
		return "TypeError"
	}
	if strings.Contains(errorText, "ValueError") {
		return "ValueError"
	}
	if strings.Contains(errorText, "AttributeError") {
		return "AttributeError"
	}
	return "RuntimeError"
}

// mustMarshal marshals to JSON, returning empty object on error.
func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// hasErrorInResult checks if the tool result contains an error in its content,
// even if the IsError flag is false. This handles cases where code execution
// returns success=true but error_details contains actual errors.
func (e *DeepResearchExecutor) hasErrorInResult(result *tool.Result) bool {
	if result == nil || len(result.Content) == 0 {
		return false
	}

	for _, content := range result.Content {
		if content.Type != "text" || content.Text == "" {
			continue
		}

		// Try to parse as JSON to check for error_details
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content.Text), &parsed); err == nil {
			// Check for error_details.error_name
			if errDetails, ok := parsed["error_details"].(map[string]interface{}); ok {
				if errName, ok := errDetails["error_name"].(string); ok && errName != "" {
					return true
				}
			}

			// Check for is_error field
			if isError, ok := parsed["is_error"].(bool); ok && isError {
				return true
			}
		}

		// Check for common Python error patterns in text
		errorPatterns := []string{
			"ModuleNotFoundError:",
			"ImportError:",
			"SyntaxError:",
			"TypeError:",
			"ValueError:",
			"NameError:",
			"AttributeError:",
			"IndexError:",
			"KeyError:",
			"FileNotFoundError:",
			"RuntimeError:",
			"ZeroDivisionError:",
			"RecursionError:",
			"MemoryError:",
			"Traceback (most recent call last)",
		}

		for _, pattern := range errorPatterns {
			if strings.Contains(content.Text, pattern) {
				return true
			}
		}
	}

	return false
}

// buildAccumulatedContext combines outputs from all previous tasks into a single context string.
// This ensures LLM calls have full context from research, synthesis, and other preceding steps.
func (e *DeepResearchExecutor) buildAccumulatedContext(input agent.ExecutionInput) string {
	var contextParts []string

	// Add accumulated outputs from previous tasks (research results, synthesis, etc.)
	for _, output := range input.AccumulatedOutputs {
		if len(output) > 0 {
			// Try to extract meaningful content from the output
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
func (e *DeepResearchExecutor) extractContextFromOutput(output json.RawMessage) string {
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

// extractErrorText extracts the error message from a tool result.
func (e *DeepResearchExecutor) extractErrorText(result *tool.Result) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// extractCodeFromPreviousOutput extracts Python code from the previous step's output.
// The previous step (LLM code generation) may output code in various formats:
// - As a JSON object with "code" field
// - As a JSON object with "content" field containing markdown with code blocks
// - As raw text with Python code blocks
func (e *DeepResearchExecutor) extractCodeFromPreviousOutput(output json.RawMessage) string {
	if len(output) == 0 {
		log.Debug().Msg("extractCodeFromPreviousOutput: empty input")
		return ""
	}

	// First, try to parse as JSON object
	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err == nil {
		// Check for direct "code" field
		if code, ok := parsed["code"].(string); ok && code != "" {
			log.Debug().Int("code_len", len(code)).Msg("Found code in 'code' field")
			return code
		}

		// Check for "content" field (common in LLM responses)
		if content, ok := parsed["content"].(string); ok && content != "" {
			if code := extractCodeBlock(content); code != "" {
				log.Debug().Int("code_len", len(code)).Msg("Extracted code from 'content' field")
				return code
			}
			// If no code block found but content looks like code, return it
			if looksLikeCode(content) {
				log.Debug().Int("content_len", len(content)).Msg("Content field looks like code, using it directly")
				return strings.TrimSpace(content)
			}
		}

		// Check for "text" field
		if text, ok := parsed["text"].(string); ok && text != "" {
			if code := extractCodeBlock(text); code != "" {
				log.Debug().Int("code_len", len(code)).Msg("Extracted code from 'text' field")
				return code
			}
			if looksLikeCode(text) {
				log.Debug().Int("text_len", len(text)).Msg("Text field looks like code, using it directly")
				return strings.TrimSpace(text)
			}
		}

		// Check for nested "choices" (OpenAI format)
		if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok && content != "" {
						if code := extractCodeBlock(content); code != "" {
							log.Debug().Int("code_len", len(code)).Msg("Extracted code from OpenAI choices format")
							return code
						}
					}
				}
			}
		}

		log.Debug().Msg("No code found in JSON structure, trying raw text extraction")
	}

	// Try as raw text
	rawText := string(output)
	if code := extractCodeBlock(rawText); code != "" {
		log.Debug().Int("code_len", len(code)).Msg("Extracted code from raw text")
		return code
	}

	// Last resort: if the entire raw text looks like code, use it
	if looksLikeCode(rawText) {
		log.Debug().Int("raw_len", len(rawText)).Msg("Raw text looks like code, using it directly")
		return strings.TrimSpace(rawText)
	}

	log.Warn().Str("output_preview", truncateForLog(output, 200)).Msg("Could not extract code from previous output")
	return ""
}

// extractCodeBlock extracts Python code from markdown code blocks.
func extractCodeBlock(text string) string {
	// Look for Python code block (```python ... ```)
	pythonBlockRegex := regexp.MustCompile("(?s)```python\\s*\n(.*?)```")
	if matches := pythonBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Look for generic code block (``` ... ```)
	genericBlockRegex := regexp.MustCompile("(?s)```\\s*\n(.*?)```")
	if matches := genericBlockRegex.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If the entire text looks like code (has def/import/class statements), return it
	codeIndicators := []string{"import ", "from ", "def ", "class ", "print(", "if __name__"}
	for _, indicator := range codeIndicators {
		if strings.Contains(text, indicator) {
			return strings.TrimSpace(text)
		}
	}

	return ""
}

func looksLikeCode(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "```") {
		return true
	}
	codePattern := regexp.MustCompile(`(?m)^\s*(def|class|import|from)\b`)
	if codePattern.MatchString(trimmed) {
		return true
	}
	if strings.Contains(trimmed, "if __name__") {
		return true
	}
	if strings.Contains(trimmed, "return ") && strings.Contains(trimmed, "\n") {
		return true
	}
	if strings.Contains(trimmed, "print(") && strings.Contains(trimmed, "\n") {
		return true
	}
	return false
}

// buildToolArguments constructs proper tool arguments from step params and context.
func (e *DeepResearchExecutor) buildToolArguments(toolName string, params map[string]interface{}, input agent.ExecutionInput, description string, currentCode *string) (map[string]interface{}, error) {
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
			// Return error instead of nil to properly fail the step
			return nil, fmt.Errorf("no URLs available to scrape from previous search results")
		}
		// MCP scrape tool expects single url parameter, so use first URL
		return map[string]interface{}{
			"url": urls[0],
		}, nil

	case "aio_code_execute":
		// Priority: currentCode (from retry) > params["code"] > PreviousOutput
		var code string
		if currentCode != nil {
			code = *currentCode
		} else if codeParam, ok := params["code"].(string); ok {
			code = codeParam
		} else {
			// Extract code from previous LLM output
			code = e.extractCodeFromPreviousOutput(input.PreviousOutput)
		}

		if code == "" {
			// Try to extract from accumulated outputs as last resort
			for _, accOutput := range input.AccumulatedOutputs {
				if extracted := e.extractCodeFromPreviousOutput(accOutput); extracted != "" {
					log.Info().
						Str("tool", "aio_code_execute").
						Int("code_len", len(extracted)).
						Msg("Found code in accumulated outputs")
					code = extracted
					break
				}
			}
		}

		if code == "" {
			log.Error().
				Str("tool", "aio_code_execute").
				Str("previous_output_preview", truncateForLog(input.PreviousOutput, 500)).
				Int("accumulated_outputs_count", len(input.AccumulatedOutputs)).
				Msg("No code provided for execution - LLM may have returned empty or invalid response")
			return nil, fmt.Errorf("no code provided for execution: LLM returned empty or invalid response. This step will be retried automatically")
		}

		code = agent.NormalizeSandboxFilePaths(code)
		language, _ := params["language"].(string)
		if language == "" {
			language = "python" // Default
		}

		return map[string]interface{}{
			"language": language,
			"code":     code,
		}, nil

	case "aio_shell_exec":
		var command string
		if currentCode != nil {
			command = *currentCode
		} else if cmdParam, ok := params["command"].(string); ok {
			command = cmdParam
		}
		if command == "" {
			return nil, fmt.Errorf("no command provided for shell execution")
		}
		return map[string]interface{}{
			"command": command,
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
func (e *DeepResearchExecutor) extractURLsFromPreviousOutput(previousOutput json.RawMessage) []string {
	if len(previousOutput) == 0 {
		return nil
	}

	var output map[string]interface{}
	if err := json.Unmarshal(previousOutput, &output); err != nil {
		return nil
	}

	// Extract URLs from search results
	var urls []string
	if results, ok := output["results"].([]interface{}); ok {
		for _, r := range results {
			if result, ok := r.(map[string]interface{}); ok {
				// Check both 'source_url' (actual field) and 'url' (backward compatibility)
				if url, ok := result["source_url"].(string); ok && url != "" {
					urls = append(urls, url)
				} else if url, ok := result["url"].(string); ok && url != "" {
					urls = append(urls, url)
				}
			}
		}
	}

	// Also check for direct content array with url field
	if content, ok := output["content"].([]interface{}); ok {
		for _, c := range content {
			if item, ok := c.(map[string]interface{}); ok {
				if url, ok := item["url"].(string); ok && url != "" {
					urls = append(urls, url)
				}
				if text, ok := item["text"].(string); ok && text != "" {
					var embedded map[string]interface{}
					if err := json.Unmarshal([]byte(text), &embedded); err == nil {
						if results, ok := embedded["results"].([]interface{}); ok {
							for _, r := range results {
								if result, ok := r.(map[string]interface{}); ok {
									if url, ok := result["source_url"].(string); ok && url != "" {
										urls = append(urls, url)
									} else if url, ok := result["url"].(string); ok && url != "" {
										urls = append(urls, url)
									}
								}
							}
						}
						if citations, ok := embedded["citations"].([]interface{}); ok {
							for _, citation := range citations {
								if url, ok := citation.(string); ok && url != "" {
									urls = append(urls, url)
								}
							}
						}
					}
				}
			}
		}
	}

	return urls
}

// installPackage calls aio_install_packages to install a missing package.
func (e *DeepResearchExecutor) installPackage(ctx context.Context, packageName string, input agent.ExecutionInput) (*tool.Result, error) {
	callReq := tool.CallRequest{
		Name: "aio_install_packages",
		Arguments: map[string]interface{}{
			"packages": []string{packageName},
		},
	}
	if input.PlanContext != nil {
		callReq.RequestID = input.PlanContext.ResponseID
		callReq.ConversationID = input.PlanContext.ConversationID
	}

	return e.mcpClient.CallTool(ctx, callReq)
}

func (e *DeepResearchExecutor) executeLLMCall(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(step.InputParams, &params); err != nil {
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
	description, _ := params["description"].(string)

	// Build context from accumulated outputs (all previous tasks) + current task's previous output
	// This ensures LLM calls have full context from research, synthesis, etc.
	contextData := e.buildAccumulatedContext(input)

	// Build prompt based on action type and accumulated context
	var prompt string
	switch action {
	case "reasoning":
		prompt = fmt.Sprintf("Analyze and synthesize the following research findings. %s\n\nPrevious findings: %s",
			description, contextData)
	case "generate_code":
		prompt = fmt.Sprintf("Generate Python code based on the research findings. %s\n\nResearch context: %s\n\nRequirements: Create simple, runnable code using only standard library or commonly available packages. Include comments. The code should directly address the user's request using the data from the research.",
			description, contextData)
	case "generate_content":
		prompt = fmt.Sprintf("Write a comprehensive analysis report. %s\n\nAll research data: %s\n\nFormat: Provide detailed analysis with citations and conclusions.",
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
		Msg("Executing LLM call")

	// Call LLM provider
	if e.llmProvider == nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Message:  "LLM provider not configured",
				Severity: status.ErrorSeverityFatal,
			},
		}, nil
	}

	response, err := e.llmProvider.GenerateWithModel(ctx, prompt, model)
	if err != nil {
		log.Error().
			Err(err).
			Str("action", action).
			Str("step_id", step.ID).
			Str("model", model).
			Msg("LLM generation failed")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Message:  fmt.Sprintf("LLM generation failed: %v", err),
				Severity: status.ErrorSeverityRetryable,
			},
		}, nil
	}

	// Log LLM response for tracking
	log.Info().
		Str("action", action).
		Str("step_id", step.ID).
		Str("model", model).
		Int("response_length", len(response)).
		Str("response_preview", truncateForLog(json.RawMessage(response), 300)).
		Msg("LLM response received")

	// Check if response is empty
	if strings.TrimSpace(response) == "" {
		log.Warn().
			Str("action", action).
			Str("step_id", step.ID).
			Str("model", model).
			Msg("LLM returned empty response")
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error: &agent.ExecutionError{
				Message:  "LLM returned empty response, please retry",
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

	outputBytes, err := json.Marshal(output)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: outputBytes,
	}, nil
}

func isNonCriticalTool(toolName string) bool {
	switch toolName {
	case "google_search", "scrape":
		return true
	default:
		return false
	}
}

func buildSkippedToolResult(toolName string, reason string, statusCode string) *agent.ExecutionResult {
	output, _ := json.Marshal(map[string]interface{}{
		"type":    "tool_result",
		"tool":    toolName,
		"status":  "skipped",
		"reason":  reason,
		"code":    statusCode,
		"skipped": true,
	})
	return &agent.ExecutionResult{
		Status: status.StatusCompleted,
		Output: output,
	}
}

// truncateForLog truncates a byte slice for safe logging
func truncateForLog(data json.RawMessage, maxLen int) string {
	if len(data) == 0 {
		return ""
	}
	s := string(data)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
