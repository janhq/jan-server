package plan

import (
	"context"
	"time"

	"jan-server/services/response-api/internal/domain/status"
)

// Service defines the interface for plan business logic.
type Service interface {
	// Plan lifecycle
	Create(ctx context.Context, params CreateParams) (*Plan, error)
	GetByID(ctx context.Context, id string) (*Plan, error)
	GetByResponseID(ctx context.Context, responseID string) (*Plan, error)
	UpdateStatus(ctx context.Context, id string, newStatus status.Status, errorMsg *string) error
	UpdateProgress(ctx context.Context, id string, completedSteps int) error
	SetUserSelection(ctx context.Context, id string, selection string) error
	SetFinalArtifact(ctx context.Context, id string, artifactID string) error
	Cancel(ctx context.Context, id string, reason string) error
	Delete(ctx context.Context, id string) error

	// Task operations
	CreateTask(ctx context.Context, planID string, params CreateTaskParams) (*Task, error)
	StartNextTask(ctx context.Context, planID string) (*Task, error)
	CompleteTask(ctx context.Context, taskID string) error
	FailTask(ctx context.Context, taskID string, errorMsg string) error

	// Step operations
	CreateStep(ctx context.Context, taskID string, params CreateStepParams) (*Step, error)
	StartStep(ctx context.Context, stepID string) error
	CompleteStep(ctx context.Context, stepID string, output []byte) error
	FailStep(ctx context.Context, stepID string, errorMsg string, severity status.ErrorSeverity) error
	RetryStep(ctx context.Context, stepID string) (*Step, error)
	SkipStep(ctx context.Context, stepID string, reason string) error

	// Detail operations
	AddStepDetail(ctx context.Context, stepID string, detail *StepDetail) error

	// Query operations
	GetProgress(ctx context.Context, planID string) (*PlanProgress, error)
	GetPlanWithDetails(ctx context.Context, planID string) (*Plan, error)
	List(ctx context.Context, filter *Filter) ([]*Plan, int64, error)
}

// CreateParams contains parameters for creating a new plan.
type CreateParams struct {
	ResponseID     string
	AgentType      AgentType
	Config         *PlanConfig
	EstimatedSteps int
}

// CreateTaskParams contains parameters for creating a new task.
type CreateTaskParams struct {
	Sequence    int
	TaskType    TaskType
	Title       string
	Description *string
}

// CreateStepParams contains parameters for creating a new step.
type CreateStepParams struct {
	Sequence    int
	Action      ActionType
	InputParams []byte
	MaxRetries  int
}

// DefaultService implements the Service interface.
type DefaultService struct {
	repo Repository
}

// NewService creates a new plan service.
func NewService(repo Repository) Service {
	return &DefaultService{repo: repo}
}

// Create creates a new plan.
func (s *DefaultService) Create(ctx context.Context, params CreateParams) (*Plan, error) {
	config := DefaultPlanConfig()
	if params.Config != nil {
		config = *params.Config
	}

	plan := &Plan{
		ResponseID:     params.ResponseID,
		Status:         status.StatusPending,
		Progress:       0,
		AgentType:      params.AgentType,
		PlanningConfig: config,
		EstimatedSteps: params.EstimatedSteps,
		CompletedSteps: 0,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

// GetByID retrieves a plan by ID.
func (s *DefaultService) GetByID(ctx context.Context, id string) (*Plan, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByResponseID retrieves a plan by response ID.
func (s *DefaultService) GetByResponseID(ctx context.Context, responseID string) (*Plan, error) {
	return s.repo.FindByResponseID(ctx, responseID)
}

// UpdateStatus updates the plan status with validation.
func (s *DefaultService) UpdateStatus(ctx context.Context, id string, newStatus status.Status, errorMsg *string) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !plan.Status.CanTransitionTo(newStatus) {
		return status.ErrInvalidTransition
	}

	plan.Status = newStatus
	plan.UpdatedAt = time.Now().UTC()
	plan.ErrorMessage = errorMsg

	if newStatus.IsTerminal() {
		now := time.Now().UTC()
		plan.CompletedAt = &now
	}

	return s.repo.Update(ctx, plan)
}

// UpdateProgress updates the plan's progress.
func (s *DefaultService) UpdateProgress(ctx context.Context, id string, completedSteps int) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	plan.CompletedSteps = completedSteps
	plan.UpdateProgress()
	plan.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, plan)
}

// SetUserSelection stores the user's selection for the plan.
func (s *DefaultService) SetUserSelection(ctx context.Context, id string, selection string) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	plan.UserSelection = &selection
	plan.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, plan)
}

// SetFinalArtifact sets the final artifact ID for the plan.
func (s *DefaultService) SetFinalArtifact(ctx context.Context, id string, artifactID string) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	plan.FinalArtifactID = &artifactID
	plan.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, plan)
}

// Cancel cancels the plan execution.
func (s *DefaultService) Cancel(ctx context.Context, id string, reason string) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if plan.Status.IsTerminal() {
		return status.ErrInvalidTransition
	}

	plan.Status = status.StatusCancelled
	plan.ErrorMessage = &reason
	plan.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	plan.CompletedAt = &now

	return s.repo.Update(ctx, plan)
}

// Delete removes a plan.
func (s *DefaultService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// CreateTask creates a new task within a plan.
func (s *DefaultService) CreateTask(ctx context.Context, planID string, params CreateTaskParams) (*Task, error) {
	task := &Task{
		PlanID:      planID,
		Sequence:    params.Sequence,
		TaskType:    params.TaskType,
		Status:      status.StatusPending,
		Title:       params.Title,
		Description: params.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// StartNextTask finds and starts the next pending task.
func (s *DefaultService) StartNextTask(ctx context.Context, planID string) (*Task, error) {
	tasks, err := s.repo.ListTasksByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}

	var nextTask *Task
	for _, t := range tasks {
		if t.Status == status.StatusPending {
			nextTask = t
			break
		}
	}

	if nextTask == nil {
		return nil, nil // No more pending tasks
	}

	plan, err := s.repo.FindByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	nextTask.Status = status.StatusInProgress
	nextTask.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateTask(ctx, nextTask); err != nil {
		return nil, err
	}

	plan.CurrentTaskID = &nextTask.ID
	plan.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, err
	}

	return nextTask, nil
}

// CompleteTask marks a task as completed.
func (s *DefaultService) CompleteTask(ctx context.Context, taskID string) error {
	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	task.Status = status.StatusCompleted
	task.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	task.CompletedAt = &now

	return s.repo.UpdateTask(ctx, task)
}

// FailTask marks a task as failed.
func (s *DefaultService) FailTask(ctx context.Context, taskID string, errorMsg string) error {
	task, err := s.repo.FindTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	task.Status = status.StatusFailed
	task.ErrorMessage = &errorMsg
	task.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateTask(ctx, task)
}

// CreateStep creates a new step within a task.
func (s *DefaultService) CreateStep(ctx context.Context, taskID string, params CreateStepParams) (*Step, error) {
	maxRetries := params.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	step := &Step{
		TaskID:      taskID,
		Sequence:    params.Sequence,
		Action:      params.Action,
		Status:      status.StatusPending,
		InputParams: params.InputParams,
		RetryCount:  0,
		MaxRetries:  maxRetries,
	}

	if err := s.repo.CreateStep(ctx, step); err != nil {
		return nil, err
	}
	return step, nil
}

// StartStep marks a step as in-progress.
func (s *DefaultService) StartStep(ctx context.Context, stepID string) error {
	step, err := s.repo.FindStepByID(ctx, stepID)
	if err != nil {
		return err
	}

	step.Status = status.StatusInProgress
	now := time.Now().UTC()
	step.StartedAt = &now

	return s.repo.UpdateStep(ctx, step)
}

// CompleteStep marks a step as completed with output.
func (s *DefaultService) CompleteStep(ctx context.Context, stepID string, output []byte) error {
	step, err := s.repo.FindStepByID(ctx, stepID)
	if err != nil {
		return err
	}

	step.Status = status.StatusCompleted
	step.OutputData = output
	now := time.Now().UTC()
	step.CompletedAt = &now

	if step.StartedAt != nil {
		durationMs := now.Sub(*step.StartedAt).Milliseconds()
		step.DurationMs = &durationMs
	}

	return s.repo.UpdateStep(ctx, step)
}

// FailStep marks a step as failed with error details.
func (s *DefaultService) FailStep(ctx context.Context, stepID string, errorMsg string, severity status.ErrorSeverity) error {
	step, err := s.repo.FindStepByID(ctx, stepID)
	if err != nil {
		return err
	}

	step.Status = status.StatusFailed
	step.ErrorMessage = &errorMsg
	step.ErrorSeverity = severity
	now := time.Now().UTC()
	step.CompletedAt = &now

	if step.StartedAt != nil {
		durationMs := now.Sub(*step.StartedAt).Milliseconds()
		step.DurationMs = &durationMs
	}

	return s.repo.UpdateStep(ctx, step)
}

// RetryStep resets a failed step for retry.
func (s *DefaultService) RetryStep(ctx context.Context, stepID string) (*Step, error) {
	step, err := s.repo.FindStepByID(ctx, stepID)
	if err != nil {
		return nil, err
	}

	if !step.CanRetry() {
		return nil, status.ErrInvalidTransition
	}

	step.IncrementRetry()
	step.Status = status.StatusPending
	step.ErrorMessage = nil
	step.StartedAt = nil
	step.CompletedAt = nil
	step.DurationMs = nil

	if err := s.repo.UpdateStep(ctx, step); err != nil {
		return nil, err
	}
	return step, nil
}

// SkipStep marks a step as skipped.
func (s *DefaultService) SkipStep(ctx context.Context, stepID string, reason string) error {
	step, err := s.repo.FindStepByID(ctx, stepID)
	if err != nil {
		return err
	}

	step.Status = status.StatusSkipped
	step.ErrorMessage = &reason

	return s.repo.UpdateStep(ctx, step)
}

// AddStepDetail adds a detail record to a step.
func (s *DefaultService) AddStepDetail(ctx context.Context, stepID string, detail *StepDetail) error {
	detail.StepID = stepID
	detail.CreatedAt = time.Now().UTC()
	return s.repo.CreateStepDetail(ctx, detail)
}

// GetProgress retrieves the current progress of a plan.
func (s *DefaultService) GetProgress(ctx context.Context, planID string) (*PlanProgress, error) {
	return s.repo.GetProgress(ctx, planID)
}

// GetPlanWithDetails retrieves a plan with all its tasks and steps.
func (s *DefaultService) GetPlanWithDetails(ctx context.Context, planID string) (*Plan, error) {
	return s.repo.FindPlanWithDetails(ctx, planID)
}

// List retrieves plans matching the filter.
func (s *DefaultService) List(ctx context.Context, filter *Filter) ([]*Plan, int64, error) {
	return s.repo.List(ctx, filter)
}
