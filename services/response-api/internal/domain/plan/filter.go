package plan

import (
	"time"

	"jan-server/services/response-api/internal/domain/status"
)

// Filter contains criteria for filtering plans.
type Filter struct {
	ID         *string
	ResponseID *string
	Status     *status.Status
	AgentType  *AgentType
	UserID     *string

	// Time filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Pagination
	Limit  int
	Offset int
}

// NewFilter creates a new filter with default pagination.
func NewFilter() *Filter {
	return &Filter{
		Limit:  20,
		Offset: 0,
	}
}

// WithResponseID sets the response ID filter.
func (f *Filter) WithResponseID(responseID string) *Filter {
	f.ResponseID = &responseID
	return f
}

// WithStatus sets the status filter.
func (f *Filter) WithStatus(s status.Status) *Filter {
	f.Status = &s
	return f
}

// WithAgentType sets the agent type filter.
func (f *Filter) WithAgentType(t AgentType) *Filter {
	f.AgentType = &t
	return f
}

// WithPagination sets the pagination parameters.
func (f *Filter) WithPagination(limit, offset int) *Filter {
	f.Limit = limit
	f.Offset = offset
	return f
}

// TaskFilter contains criteria for filtering tasks.
type TaskFilter struct {
	ID       *string
	PlanID   *string
	Status   *status.Status
	TaskType *TaskType

	// Pagination
	Limit  int
	Offset int
}

// NewTaskFilter creates a new task filter with default pagination.
func NewTaskFilter() *TaskFilter {
	return &TaskFilter{
		Limit:  50,
		Offset: 0,
	}
}

// WithPlanID sets the plan ID filter.
func (f *TaskFilter) WithPlanID(planID string) *TaskFilter {
	f.PlanID = &planID
	return f
}

// StepFilter contains criteria for filtering steps.
type StepFilter struct {
	ID     *string
	TaskID *string
	Status *status.Status
	Action *ActionType

	// Include related data
	IncludeDetails bool

	// Pagination
	Limit  int
	Offset int
}

// NewStepFilter creates a new step filter with default pagination.
func NewStepFilter() *StepFilter {
	return &StepFilter{
		Limit:  100,
		Offset: 0,
	}
}

// WithTaskID sets the task ID filter.
func (f *StepFilter) WithTaskID(taskID string) *StepFilter {
	f.TaskID = &taskID
	return f
}

// WithIncludeDetails enables loading of step details.
func (f *StepFilter) WithIncludeDetails() *StepFilter {
	f.IncludeDetails = true
	return f
}
