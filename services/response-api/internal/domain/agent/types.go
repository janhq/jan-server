// Package agent defines interfaces and types for agent execution.
package agent

import (
	"context"
	"encoding/json"

	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
)

// Planner defines the interface for agents that can create execution plans.
type Planner interface {
	// Name returns the unique identifier for this agent.
	Name() string

	// CanHandle determines if this agent can handle the given request.
	CanHandle(ctx context.Context, request *PlanRequest) bool

	// CreatePlan analyzes the request and creates an execution plan.
	CreatePlan(ctx context.Context, request *PlanRequest) (*PlanResult, error)
}

// Executor defines the interface for executing plan steps.
type Executor interface {
	// Execute runs a single step and returns the result.
	Execute(ctx context.Context, step *plan.Step, input ExecutionInput) (*ExecutionResult, error)

	// CanExecute checks if this executor can handle the given action type.
	CanExecute(action plan.ActionType) bool

	// Rollback attempts to undo a step's effects (optional).
	Rollback(ctx context.Context, step *plan.Step) error
}

// PlanRequest contains the input for plan creation.
type PlanRequest struct {
	ResponseID          string                 `json:"response_id"`
	ConversationID      string                 `json:"conversation_id"`
	UserMessage         string                 `json:"user_message"`
	SystemPrompt        *string                `json:"system_prompt,omitempty"`
	Model               string                 `json:"model"`
	Temperature         *float64               `json:"temperature,omitempty"`
	MaxTokens           *int                   `json:"max_tokens,omitempty"`
	Tools               []ToolDefinition       `json:"tools,omitempty"`
	ConversationHistory []ConversationItem     `json:"conversation_history,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// ToolDefinition describes an available tool.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ConversationItem represents a message in the conversation history.
type ConversationItem struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// PlanResult contains the output of plan creation.
type PlanResult struct {
	Plan             *plan.Plan   `json:"plan"`
	Tasks            []*plan.Task `json:"tasks"`
	InitialSteps     []*plan.Step `json:"initial_steps,omitempty"`
	RequiresApproval bool         `json:"requires_approval"`
	ApprovalPrompt   *string      `json:"approval_prompt,omitempty"`
}

// ExecutionInput contains input data for step execution.
type ExecutionInput struct {
	StepParams     json.RawMessage        `json:"step_params"`
	PreviousOutput json.RawMessage        `json:"previous_output,omitempty"`
	PlanContext    *PlanContext           `json:"plan_context,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PlanContext provides context about the current plan execution.
type PlanContext struct {
	PlanID         string         `json:"plan_id"`
	TaskID         string         `json:"task_id"`
	ConversationID string         `json:"conversation_id"`
	ResponseID     string         `json:"response_id"`
	AgentType      plan.AgentType `json:"agent_type"`
	ArtifactIDs    []string       `json:"artifact_ids,omitempty"`
}

// ExecutionResult contains the output of step execution.
type ExecutionResult struct {
	Status       status.Status     `json:"status"`
	Output       json.RawMessage   `json:"output,omitempty"`
	Error        *ExecutionError   `json:"error,omitempty"`
	ArtifactID   *string           `json:"artifact_id,omitempty"`
	Details      []plan.StepDetail `json:"details,omitempty"`
	NextSteps    []*plan.Step      `json:"next_steps,omitempty"`
	RequiresUser bool              `json:"requires_user"`
	UserPrompt   *string           `json:"user_prompt,omitempty"`
}

// ExecutionError contains error details from execution.
type ExecutionError struct {
	Code     string               `json:"code"`
	Message  string               `json:"message"`
	Severity status.ErrorSeverity `json:"severity"`
	Details  json.RawMessage      `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	return "[" + e.Code + "] " + e.Message
}

// IsRetryable returns true if the error can be retried.
func (e *ExecutionError) IsRetryable() bool {
	return e.Severity.IsRetryable()
}

// IsFatal returns true if the error should fail the plan.
func (e *ExecutionError) IsFatal() bool {
	return e.Severity.IsFatal()
}
