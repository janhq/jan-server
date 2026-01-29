package plan

import (
	"context"
)

// Repository defines the interface for plan persistence.
type Repository interface {
	// Plan operations
	Create(ctx context.Context, plan *Plan) error
	Update(ctx context.Context, plan *Plan) error
	FindByID(ctx context.Context, id string) (*Plan, error)
	FindByResponseID(ctx context.Context, responseID string) (*Plan, error)
	List(ctx context.Context, filter *Filter) ([]*Plan, int64, error)
	Delete(ctx context.Context, id string) error

	// Task operations
	CreateTask(ctx context.Context, task *Task) error
	UpdateTask(ctx context.Context, task *Task) error
	FindTaskByID(ctx context.Context, id string) (*Task, error)
	ListTasksByPlanID(ctx context.Context, planID string) ([]*Task, error)

	// Step operations
	CreateStep(ctx context.Context, step *Step) error
	UpdateStep(ctx context.Context, step *Step) error
	FindStepByID(ctx context.Context, id string) (*Step, error)
	ListStepsByTaskID(ctx context.Context, taskID string) ([]*Step, error)

	// Step detail operations
	CreateStepDetail(ctx context.Context, detail *StepDetail) error
	ListDetailsByStepID(ctx context.Context, stepID string) ([]*StepDetail, error)

	// Aggregated queries
	GetProgress(ctx context.Context, planID string) (*PlanProgress, error)
	FindPlanWithDetails(ctx context.Context, id string) (*Plan, error)
}
