// Package planners contains agent planner implementations.
package planners

import (
	deepresearch "jan-server/services/response-api/internal/domain/agent/planners/deep_research"
	"jan-server/services/response-api/internal/domain/plan"
)

// DeepResearchPlanner is an alias for backward compatibility.
type DeepResearchPlanner = deepresearch.Planner

// DeepResearchExecutor is an alias for backward compatibility.
type DeepResearchExecutor = deepresearch.Executor

// NewDeepResearchPlanner creates a new deep research planner.
// This is a wrapper for backward compatibility.
func NewDeepResearchPlanner(planService plan.Service) *deepresearch.Planner {
	return deepresearch.NewPlanner(planService)
}

// NewDeepResearchExecutor creates a new deep research executor.
// This is a wrapper for backward compatibility.
// The MCPClient and LLMProvider interfaces are compatible with deepresearch package.
func NewDeepResearchExecutor(mcpClient MCPClient, llmProvider LLMProvider) *deepresearch.Executor {
	return deepresearch.NewExecutor(mcpClient, llmProvider)
}
