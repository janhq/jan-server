// Package plan defines plan-related domain entities and services.
package plan

import (
	"encoding/json"
	"time"

	"jan-server/services/response-api/internal/domain/status"
)

// Plan represents a multi-step execution plan created by an agent.
type Plan struct {
	ID              string        `json:"id"`
	ResponseID      string        `json:"response_id"`
	Status          status.Status `json:"status"`
	Progress        float64       `json:"progress"` // 0-100 percentage
	AgentType       AgentType     `json:"agent_type"`
	PlanningConfig  PlanConfig    `json:"planning_config,omitempty"`
	EstimatedSteps  int           `json:"estimated_steps"`
	CompletedSteps  int           `json:"completed_steps"`
	CurrentTaskID   *string       `json:"current_task_id,omitempty"`
	FinalArtifactID *string       `json:"final_artifact_id,omitempty"`
	UserSelection   *string       `json:"user_selection,omitempty"` // JSON: user's choices
	ErrorMessage    *string       `json:"error_message,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Tasks []Task `json:"tasks,omitempty"`
}

// AgentType identifies the type of agent handling the plan.
type AgentType string

const (
	AgentTypeSlideGenerator AgentType = "slide_generator"
	AgentTypeDeepResearch   AgentType = "deep_research"
	AgentTypeCustom         AgentType = "custom"
)

// String returns the string representation of the agent type.
func (a AgentType) String() string {
	return string(a)
}

// PlanConfig contains configuration for plan execution.
type PlanConfig struct {
	MaxRetries        int           `json:"max_retries"`
	TimeoutPerStep    time.Duration `json:"timeout_per_step"`
	EnableFallback    bool          `json:"enable_fallback"`
	UserApproval      bool          `json:"user_approval"`
	StreamProgress    bool          `json:"stream_progress"`
	ArtifactRetention string        `json:"artifact_retention"` // "ephemeral", "session", "permanent"
}

// DefaultPlanConfig returns the default configuration for a plan.
func DefaultPlanConfig() PlanConfig {
	return PlanConfig{
		MaxRetries:        3,
		TimeoutPerStep:    5 * time.Minute,
		EnableFallback:    true,
		UserApproval:      false,
		StreamProgress:    true,
		ArtifactRetention: "session",
	}
}

// Task represents a group of related steps in a plan.
type Task struct {
	ID           string        `json:"id"`
	PlanID       string        `json:"plan_id"`
	Sequence     int           `json:"sequence"`
	TaskType     TaskType      `json:"task_type"`
	Status       status.Status `json:"status"`
	Title        string        `json:"title"`
	Description  *string       `json:"description,omitempty"`
	ErrorMessage *string       `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Steps []Step `json:"steps,omitempty"`
}

// TaskType identifies the category of task.
type TaskType string

const (
	TaskTypeResearch     TaskType = "research"
	TaskTypeGeneration   TaskType = "generation"
	TaskTypeValidation   TaskType = "validation"
	TaskTypeTransform    TaskType = "transform"
	TaskTypeUserInput    TaskType = "user_input"
	TaskTypeFinalization TaskType = "finalization"
	TaskTypeExecution    TaskType = "execution"
)

// String returns the string representation of the task type.
func (t TaskType) String() string {
	return string(t)
}

// Step represents an individual action in a task.
type Step struct {
	ID            string               `json:"id"`
	TaskID        string               `json:"task_id"`
	Sequence      int                  `json:"sequence"`
	Action        ActionType           `json:"action"`
	Status        status.Status        `json:"status"`
	InputParams   json.RawMessage      `json:"input_params,omitempty"`
	OutputData    json.RawMessage      `json:"output_data,omitempty"`
	RetryCount    int                  `json:"retry_count"`
	MaxRetries    int                  `json:"max_retries"`
	ErrorMessage  *string              `json:"error_message,omitempty"`
	ErrorSeverity status.ErrorSeverity `json:"error_severity,omitempty"`
	DurationMs    *int64               `json:"duration_ms,omitempty"`
	StartedAt     *time.Time           `json:"started_at,omitempty"`
	CompletedAt   *time.Time           `json:"completed_at,omitempty"`

	// Relations (loaded conditionally)
	Details []StepDetail `json:"details,omitempty"`
}

// ActionType identifies the action performed by a step.
type ActionType string

const (
	ActionTypeLLMCall        ActionType = "llm_call"
	ActionTypeToolCall       ActionType = "tool_call"
	ActionTypeArtifactCreate ActionType = "artifact_create"
	ActionTypeArtifactUpdate ActionType = "artifact_update"
	ActionTypeUserPrompt     ActionType = "user_prompt"
	ActionTypeValidation     ActionType = "validation"
	ActionTypeTransform      ActionType = "transform"
)

// String returns the string representation of the action type.
func (a ActionType) String() string {
	return string(a)
}

// StepDetail links a step to its execution artifacts.
type StepDetail struct {
	ID                 string          `json:"id"`
	StepID             string          `json:"step_id"`
	DetailType         DetailType      `json:"detail_type"`
	ConversationItemID *string         `json:"conversation_item_id,omitempty"`
	ToolCallID         *string         `json:"tool_call_id,omitempty"`
	ToolExecutionID    *string         `json:"tool_execution_id,omitempty"`
	ArtifactID         *string         `json:"artifact_id,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// DetailType identifies the type of step detail.
type DetailType string

const (
	DetailTypeMessage      DetailType = "message"
	DetailTypeToolCall     DetailType = "tool_call"
	DetailTypeToolResult   DetailType = "tool_result"
	DetailTypeArtifact     DetailType = "artifact"
	DetailTypeUserResponse DetailType = "user_response"
)

// String returns the string representation of the detail type.
func (d DetailType) String() string {
	return string(d)
}

// PlanProgress holds aggregated progress information.
type PlanProgress struct {
	PlanID         string        `json:"plan_id"`
	Status         status.Status `json:"status"`
	Progress       float64       `json:"progress"`
	EstimatedSteps int           `json:"estimated_steps"`
	CompletedSteps int           `json:"completed_steps"`
	CurrentTask    *TaskProgress `json:"current_task,omitempty"`
	FailedSteps    int           `json:"failed_steps"`
}

// TaskProgress holds progress for a single task.
type TaskProgress struct {
	TaskID      string        `json:"task_id"`
	Title       string        `json:"title"`
	Status      status.Status `json:"status"`
	TotalSteps  int           `json:"total_steps"`
	CurrentStep int           `json:"current_step"`
}

// CanRetry checks if the step can be retried.
func (s *Step) CanRetry() bool {
	return s.RetryCount < s.MaxRetries && s.ErrorSeverity.IsRetryable()
}

// IncrementRetry increments the retry count.
func (s *Step) IncrementRetry() {
	s.RetryCount++
}

// UpdateProgress recalculates the plan's progress based on completed steps.
func (p *Plan) UpdateProgress() {
	if p.EstimatedSteps == 0 {
		p.Progress = 0
		return
	}
	p.Progress = float64(p.CompletedSteps) / float64(p.EstimatedSteps) * 100
	if p.Progress > 100 {
		p.Progress = 100
	}
}

// IsCompleted returns true if the plan is in a terminal completed state.
func (p *Plan) IsCompleted() bool {
	return p.Status == status.StatusCompleted
}

// IsFailed returns true if the plan is in a failed state.
func (p *Plan) IsFailed() bool {
	return p.Status == status.StatusFailed
}
