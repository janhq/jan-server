package agent

import (
	"context"
	"fmt"
	"sync"

	"jan-server/services/response-api/internal/domain/plan"
)

// Registry manages registered planners and executors.
type Registry interface {
	// Planner operations
	RegisterPlanner(planner Planner) error
	GetPlanner(name string) (Planner, bool)
	FindPlanner(ctx context.Context, request *PlanRequest) (Planner, error)
	ListPlanners() []string

	// Executor operations
	RegisterExecutor(action plan.ActionType, executor Executor) error
	GetExecutor(action plan.ActionType) (Executor, bool)
	ListExecutors() []plan.ActionType

	// Agent discovery operations
	GetAllAgents() []AgentMetadata
	GetAgent(agentType string) *AgentDetail
	GetAgentSchema(agentType string) *AgentInputSchema
	GetAgentCapabilitiesForLLM() []AgentCapability
}

// DefaultRegistry is the default implementation of Registry.
type DefaultRegistry struct {
	mu        sync.RWMutex
	planners  map[string]Planner
	executors map[plan.ActionType]Executor
}

// NewRegistry creates a new agent registry.
func NewRegistry() Registry {
	return &DefaultRegistry{
		planners:  make(map[string]Planner),
		executors: make(map[plan.ActionType]Executor),
	}
}

// RegisterPlanner registers a planner with the registry.
func (r *DefaultRegistry) RegisterPlanner(planner Planner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := planner.Name()
	if _, exists := r.planners[name]; exists {
		return fmt.Errorf("planner already registered: %s", name)
	}

	r.planners[name] = planner
	return nil
}

// GetPlanner retrieves a planner by name.
func (r *DefaultRegistry) GetPlanner(name string) (Planner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	planner, ok := r.planners[name]
	return planner, ok
}

// FindPlanner finds a planner that can handle the given request.
func (r *DefaultRegistry) FindPlanner(ctx context.Context, request *PlanRequest) (Planner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, planner := range r.planners {
		if planner.CanHandle(ctx, request) {
			return planner, nil
		}
	}
	return nil, fmt.Errorf("no planner found for request")
}

// ListPlanners returns all registered planner names.
func (r *DefaultRegistry) ListPlanners() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.planners))
	for name := range r.planners {
		names = append(names, name)
	}
	return names
}

// RegisterExecutor registers an executor for a specific action type.
func (r *DefaultRegistry) RegisterExecutor(action plan.ActionType, executor Executor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.executors[action]; exists {
		return fmt.Errorf("executor already registered for action: %s", action)
	}

	r.executors[action] = executor
	return nil
}

// GetExecutor retrieves an executor for a specific action type.
func (r *DefaultRegistry) GetExecutor(action plan.ActionType) (Executor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, ok := r.executors[action]
	return executor, ok
}

// ListExecutors returns all registered action types.
func (r *DefaultRegistry) ListExecutors() []plan.ActionType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actions := make([]plan.ActionType, 0, len(r.executors))
	for action := range r.executors {
		actions = append(actions, action)
	}
	return actions
}

// GetAllAgents returns metadata for all registered agents.
func (r *DefaultRegistry) GetAllAgents() []AgentMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return metadata for all registered planners
	agents := make([]AgentMetadata, 0, len(r.planners))
	for name := range r.planners {
		if detail := GetAgentDetail(name); detail != nil {
			agents = append(agents, detail.AgentMetadata)
		}
	}

	// If no planners registered yet, return default metadata
	if len(agents) == 0 {
		return GetAllAgentMetadata()
	}

	return agents
}

// GetAgent returns full detail for a specific agent type.
func (r *DefaultRegistry) GetAgent(agentType string) *AgentDetail {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if the planner is registered
	if _, exists := r.planners[agentType]; !exists {
		// Still return metadata if it exists (for discovery before registration)
		return GetAgentDetail(agentType)
	}

	return GetAgentDetail(agentType)
}

// GetAgentSchema returns the input schema for a specific agent type.
func (r *DefaultRegistry) GetAgentSchema(agentType string) *AgentInputSchema {
	detail := r.GetAgent(agentType)
	if detail == nil {
		return nil
	}
	return detail.InputSchema
}

// GetAgentCapabilitiesForLLM returns condensed capabilities for LLM selection.
func (r *DefaultRegistry) GetAgentCapabilitiesForLLM() []AgentCapability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all agents and filter to enabled ones
	agents := r.GetAllAgents()
	capabilities := make([]AgentCapability, 0, len(agents))

	for _, m := range agents {
		if m.Enabled {
			capabilities = append(capabilities, AgentCapability{
				Type:     m.Type,
				UseWhen:  m.UseWhen,
				Keywords: m.Keywords,
			})
		}
	}

	return capabilities
}
