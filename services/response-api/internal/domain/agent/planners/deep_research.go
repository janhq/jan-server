// Package planners contains agent planner implementations.
package planners

import (
	"context"
	"encoding/json"
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
	mcpClient   MCPClient
	llmProvider LLMProvider
}

// MCPClient interface for tool execution - matches tool.MCPClient
type MCPClient interface {
	CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
}

// LLMProvider interface for LLM calls to fix code.
type LLMProvider interface {
	FixCode(ctx context.Context, code string, errorMsg string, language string) (string, error)
}

// MaxInstallRetries is the maximum number of package install retry attempts.
const MaxInstallRetries = 3

// MaxCodeFixRetries is the maximum number of LLM code fix retry attempts.
const MaxCodeFixRetries = 3

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

	// For code execution tools, ensure we have code to execute
	isCodeExecTool := toolName == "aio_code_execute" || toolName == "aio_shell_exec"
	if isCodeExecTool {
		// Priority: currentCode (from retry) > params["code"] > PreviousOutput
		if currentCode != nil {
			params["code"] = *currentCode
		} else if _, hasCode := params["code"].(string); !hasCode {
			// Try to extract code from previous step's output (LLM code generation step)
			if code := e.extractCodeFromPreviousOutput(input.PreviousOutput); code != "" {
				params["code"] = code
				log.Debug().
					Str("tool", toolName).
					Int("code_len", len(code)).
					Msg("Extracted code from previous step output")
			}
		}

		// Validate we have code
		if code, ok := params["code"].(string); !ok || code == "" {
			return &agent.ExecutionResult{
				Status: status.StatusFailed,
				Error:  &agent.ExecutionError{Message: "no code provided for execution", Severity: status.ErrorSeverityFatal},
			}, nil
		} else {
			params["code"] = agent.NormalizeSandboxFilePaths(code)
		}
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

	// Check for errors in code execution results
	// The tool might return is_error=false but still have errors in the content
	hasError := result.IsError || (isCodeExecTool && e.hasErrorInResult(result))

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
					Msg("Auto-installing missing package and retrying code execution")

				// Install the missing package (don't record as step output)
				installResult, installErr := e.installPackage(ctx, packageName, input)
				if installErr == nil && !installResult.IsError {
					// Package installed successfully, retry the original tool call
					newInstalledPackages := append(installedPackages, packageName)
					return e.executeToolCallWithRetry(ctx, step, input, installRetryCount+1, newInstalledPackages, currentCode, codeFixRetryCount)
				}

				log.Warn().
					Str("package", packageName).
					Msg("Package installation failed, trying LLM code fix")
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
					Msg("Attempting LLM-based code fix")

				fixedCode, fixErr := e.llmProvider.FixCode(ctx, originalCode, errorText, language)
				if fixErr == nil && fixedCode != "" && fixedCode != originalCode {
					log.Info().
						Int("original_len", len(originalCode)).
						Int("fixed_len", len(fixedCode)).
						Str("error_type", errorType).
						Msg("LLM generated fixed code, retrying execution")

					return e.executeToolCallWithRetry(ctx, step, input, installRetryCount, installedPackages, &fixedCode, codeFixRetryCount+1)
				}

				if fixErr != nil {
					log.Warn().
						Err(fixErr).
						Int("retry", codeFixRetryCount+1).
						Msg("LLM code fix attempt failed")
				} else if fixedCode == originalCode {
					log.Warn().
						Int("retry", codeFixRetryCount+1).
						Msg("LLM returned same code, no fix applied")
				}
			}
		} else if e.llmProvider == nil {
			log.Warn().Msg("LLM provider not configured, cannot attempt code fix")
		} else {
			log.Warn().
				Int("code_fix_retries", codeFixRetryCount).
				Int("max_retries", MaxCodeFixRetries).
				Msg("Code fix retry limit reached")
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
		return ""
	}

	// First, try to parse as JSON object
	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err == nil {
		// Check for direct "code" field
		if code, ok := parsed["code"].(string); ok && code != "" {
			return code
		}

		// Check for "content" field (common in LLM responses)
		if content, ok := parsed["content"].(string); ok && content != "" {
			if code := extractCodeBlock(content); code != "" {
				return code
			}
		}

		// Check for "text" field
		if text, ok := parsed["text"].(string); ok && text != "" {
			if code := extractCodeBlock(text); code != "" {
				return code
			}
		}

		// Check for nested "choices" (OpenAI format)
		if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok && content != "" {
						if code := extractCodeBlock(content); code != "" {
							return code
						}
					}
				}
			}
		}
	}

	// Try as raw text
	rawText := string(output)
	if code := extractCodeBlock(rawText); code != "" {
		return code
	}

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
