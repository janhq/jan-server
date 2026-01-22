package planners

import (
	"context"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
)

// RoutingExecutor delegates execution based on the plan's agent type.
type RoutingExecutor struct {
	defaultExecutor      agent.Executor
	slideCreatorExecutor agent.Executor
}

// NewRoutingExecutor creates a new routing executor.
func NewRoutingExecutor(defaultExecutor agent.Executor, slideCreatorExecutor agent.Executor) *RoutingExecutor {
	return &RoutingExecutor{
		defaultExecutor:      defaultExecutor,
		slideCreatorExecutor: slideCreatorExecutor,
	}
}

// Execute routes execution to the slide executor for slide plans, otherwise uses the default executor.
func (e *RoutingExecutor) Execute(ctx context.Context, step *plan.Step, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	if input.PlanContext != nil {
		switch input.PlanContext.AgentType {
		case plan.AgentTypeSlideCreator:
			if e.slideCreatorExecutor != nil {
				return e.slideCreatorExecutor.Execute(ctx, step, input)
			}
		}
	}
	if e.defaultExecutor == nil {
		return &agent.ExecutionResult{Status: status.StatusFailed, Error: &agent.ExecutionError{
			Code:     "executor_missing",
			Message:  "no default executor configured",
			Severity: status.ErrorSeverityFatal,
		}}, nil
	}
	return e.defaultExecutor.Execute(ctx, step, input)
}

// CanExecute returns true if either delegate can handle the action type.
func (e *RoutingExecutor) CanExecute(action plan.ActionType) bool {
	if e.slideCreatorExecutor != nil && e.slideCreatorExecutor.CanExecute(action) {
		return true
	}
	if e.defaultExecutor != nil && e.defaultExecutor.CanExecute(action) {
		return true
	}
	return false
}

// Rollback delegates rollback based on agent type if possible, otherwise no-op.
func (e *RoutingExecutor) Rollback(ctx context.Context, step *plan.Step) error {
	if e.slideCreatorExecutor != nil {
		return e.slideCreatorExecutor.Rollback(ctx, step)
	}
	if e.defaultExecutor != nil {
		return e.defaultExecutor.Rollback(ctx, step)
	}
	return nil
}

var _ agent.Executor = (*RoutingExecutor)(nil)
