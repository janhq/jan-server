package agent

import (
	"context"
	"encoding/json"
	"time"

	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
)

// Orchestrator coordinates plan execution across tasks and steps.
type Orchestrator interface {
	// StartPlan begins execution of a plan.
	StartPlan(ctx context.Context, planID string) error

	// ExecuteNextStep executes the next pending step in the plan.
	ExecuteNextStep(ctx context.Context, planID string) (*ExecutionResult, error)

	// ResumePlan resumes a paused or waiting plan.
	ResumePlan(ctx context.Context, planID string, userInput *UserInput) error

	// CancelPlan cancels an in-progress plan.
	CancelPlan(ctx context.Context, planID string, reason string) error

	// GetStatus returns the current execution status.
	GetStatus(ctx context.Context, planID string) (*OrchestratorStatus, error)
}

// UserInput represents user input to resume a waiting plan.
type UserInput struct {
	Selection string          `json:"selection,omitempty"`
	Approval  *bool           `json:"approval,omitempty"`
	Message   *string         `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// OrchestratorStatus contains the current execution state.
type OrchestratorStatus struct {
	PlanID      string         `json:"plan_id"`
	Status      status.Status  `json:"status"`
	Progress    float64        `json:"progress"`
	CurrentTask *TaskStatus    `json:"current_task,omitempty"`
	CurrentStep *StepStatus    `json:"current_step,omitempty"`
	LastError   *string        `json:"last_error,omitempty"`
	WaitingFor  *WaitingStatus `json:"waiting_for,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TaskStatus contains task execution state.
type TaskStatus struct {
	TaskID   string        `json:"task_id"`
	Title    string        `json:"title"`
	Status   status.Status `json:"status"`
	Sequence int           `json:"sequence"`
}

// StepStatus contains step execution state.
type StepStatus struct {
	StepID     string          `json:"step_id"`
	Action     plan.ActionType `json:"action"`
	Status     status.Status   `json:"status"`
	RetryCount int             `json:"retry_count"`
}

// WaitingStatus describes what the plan is waiting for.
type WaitingStatus struct {
	Type      WaitingType     `json:"type"`
	Prompt    string          `json:"prompt"`
	Options   []WaitingOption `json:"options,omitempty"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
}

// WaitingType identifies the type of wait.
type WaitingType string

const (
	WaitingTypeApproval  WaitingType = "approval"
	WaitingTypeSelection WaitingType = "selection"
	WaitingTypeInput     WaitingType = "input"
)

// WaitingOption represents a user selection option.
type WaitingOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// DefaultOrchestrator implements the Orchestrator interface.
type DefaultOrchestrator struct {
	registry    Registry
	planService plan.Service
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(registry Registry, planService plan.Service) Orchestrator {
	return &DefaultOrchestrator{
		registry:    registry,
		planService: planService,
	}
}

// StartPlan begins execution of a plan.
func (o *DefaultOrchestrator) StartPlan(ctx context.Context, planID string) error {
	p, err := o.planService.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	if p.Status != status.StatusPending {
		return status.ErrInvalidTransition
	}

	// Transition to in_progress
	if err := o.planService.UpdateStatus(ctx, planID, status.StatusInProgress, nil); err != nil {
		return err
	}

	// Start first task
	_, err = o.planService.StartNextTask(ctx, planID)
	return err
}

// ExecuteNextStep executes the next pending step in the plan.
func (o *DefaultOrchestrator) ExecuteNextStep(ctx context.Context, planID string) (*ExecutionResult, error) {
	p, err := o.planService.GetPlanWithDetails(ctx, planID)
	if err != nil {
		return nil, err
	}

	if p.Status.IsTerminal() {
		return nil, status.ErrInvalidTransition
	}

	// Find current task and step
	var currentTask *plan.Task
	var currentStep *plan.Step

	for _, task := range p.Tasks {
		if task.Status == status.StatusInProgress {
			currentTask = &task
			for _, step := range task.Steps {
				if step.Status == status.StatusPending || step.Status == status.StatusInProgress {
					currentStep = &step
					break
				}
			}
			break
		}
	}

	if currentTask == nil {
		// No active task, try to start next
		nextTask, err := o.planService.StartNextTask(ctx, planID)
		if err != nil {
			return nil, err
		}
		if nextTask == nil {
			// All tasks completed
			return o.completePlan(ctx, p)
		}
		currentTask = nextTask
	}

	if currentStep == nil {
		// No pending steps in current task, complete it and move to next
		if err := o.planService.CompleteTask(ctx, currentTask.ID); err != nil {
			return nil, err
		}
		return o.ExecuteNextStep(ctx, planID) // Recurse to find next task
	}

	// Execute the step
	return o.executeStep(ctx, p, currentTask, currentStep)
}

// executeStep executes a single step.
func (o *DefaultOrchestrator) executeStep(ctx context.Context, p *plan.Plan, task *plan.Task, step *plan.Step) (*ExecutionResult, error) {
	executor, ok := o.registry.GetExecutor(step.Action)
	if !ok {
		return nil, &ExecutionError{
			Code:     "executor_not_found",
			Message:  "no executor for action: " + step.Action.String(),
			Severity: status.ErrorSeverityFatal,
		}
	}

	// Mark step as in-progress
	if err := o.planService.StartStep(ctx, step.ID); err != nil {
		return nil, err
	}

	// Prepare input
	input := ExecutionInput{
		StepParams: step.InputParams,
		PlanContext: &PlanContext{
			PlanID:         p.ID,
			TaskID:         task.ID,
			ConversationID: "", // Would be populated from response
			ResponseID:     p.ResponseID,
			AgentType:      p.AgentType,
		},
	}

	// Execute
	result, err := executor.Execute(ctx, step, input)
	if err != nil {
		// Handle execution error
		execErr, ok := err.(*ExecutionError)
		if !ok {
			execErr = &ExecutionError{
				Code:     "execution_failed",
				Message:  err.Error(),
				Severity: status.ErrorSeverityRetryable,
			}
		}

		if err := o.planService.FailStep(ctx, step.ID, execErr.Message, execErr.Severity); err != nil {
			return nil, err
		}

		// Handle based on severity
		if execErr.Severity == status.ErrorSeverityRetryable && step.CanRetry() {
			_, retryErr := o.planService.RetryStep(ctx, step.ID)
			if retryErr != nil {
				return nil, retryErr
			}
			return o.ExecuteNextStep(ctx, p.ID) // Retry immediately
		}

		if execErr.IsFatal() {
			errMsg := execErr.Message
			if err := o.planService.UpdateStatus(ctx, p.ID, status.StatusFailed, &errMsg); err != nil {
				return nil, err
			}
		}

		return &ExecutionResult{
			Status: status.StatusFailed,
			Error:  execErr,
		}, nil
	}

	// Step completed successfully
	if result.Status == status.StatusCompleted {
		if err := o.planService.CompleteStep(ctx, step.ID, result.Output); err != nil {
			return nil, err
		}

		// Update plan progress
		progress, err := o.planService.GetProgress(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		if err := o.planService.UpdateProgress(ctx, p.ID, progress.CompletedSteps); err != nil {
			return nil, err
		}
	}

	// Handle user input required
	if result.RequiresUser {
		if err := o.planService.UpdateStatus(ctx, p.ID, status.StatusWaitForUser, nil); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// completePlan marks the plan as completed.
func (o *DefaultOrchestrator) completePlan(ctx context.Context, p *plan.Plan) (*ExecutionResult, error) {
	if err := o.planService.UpdateStatus(ctx, p.ID, status.StatusCompleted, nil); err != nil {
		return nil, err
	}

	return &ExecutionResult{
		Status:     status.StatusCompleted,
		ArtifactID: p.FinalArtifactID,
	}, nil
}

// ResumePlan resumes a paused or waiting plan.
func (o *DefaultOrchestrator) ResumePlan(ctx context.Context, planID string, userInput *UserInput) error {
	p, err := o.planService.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	if p.Status != status.StatusWaitForUser {
		return status.ErrInvalidTransition
	}

	// Store user selection if provided
	if userInput != nil && userInput.Selection != "" {
		if err := o.planService.SetUserSelection(ctx, planID, userInput.Selection); err != nil {
			return err
		}
	}

	// Resume execution
	return o.planService.UpdateStatus(ctx, planID, status.StatusInProgress, nil)
}

// CancelPlan cancels an in-progress plan.
func (o *DefaultOrchestrator) CancelPlan(ctx context.Context, planID string, reason string) error {
	return o.planService.Cancel(ctx, planID, reason)
}

// GetStatus returns the current execution status.
func (o *DefaultOrchestrator) GetStatus(ctx context.Context, planID string) (*OrchestratorStatus, error) {
	p, err := o.planService.GetPlanWithDetails(ctx, planID)
	if err != nil {
		return nil, err
	}

	orchestratorStatus := &OrchestratorStatus{
		PlanID:    p.ID,
		Status:    p.Status,
		Progress:  p.Progress,
		LastError: p.ErrorMessage,
		StartedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	// Find current task and step
	for _, task := range p.Tasks {
		if task.Status == status.StatusInProgress {
			orchestratorStatus.CurrentTask = &TaskStatus{
				TaskID:   task.ID,
				Title:    task.Title,
				Status:   task.Status,
				Sequence: task.Sequence,
			}

			for _, step := range task.Steps {
				if step.Status == status.StatusInProgress {
					orchestratorStatus.CurrentStep = &StepStatus{
						StepID:     step.ID,
						Action:     step.Action,
						Status:     step.Status,
						RetryCount: step.RetryCount,
					}
					break
				}
			}
			break
		}
	}

	return orchestratorStatus, nil
}
