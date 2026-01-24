// Package spreadsheet contains the spreadsheet generator planner.
package spreadsheet

import (
	"context"
	"encoding/json"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// Planner creates execution plans for spreadsheet generation.
type Planner struct {
	planService     plan.Service
	artifactService artifact.Service
}

// Config holds configuration for spreadsheet generation.
type Config struct {
	IncludeCharts bool `json:"include_charts"`
}

// DefaultConfig returns defaults.
func DefaultConfig() Config {
	return Config{
		IncludeCharts: false,
	}
}

// NewPlanner creates a new spreadsheet generator planner.
func NewPlanner(planService plan.Service, artifactService artifact.Service) *Planner {
	return &Planner{
		planService:     planService,
		artifactService: artifactService,
	}
}

// Name returns the planner's unique identifier.
func (p *Planner) Name() string {
	return string(plan.AgentTypeSpreadsheetGenerator)
}

// CanHandle determines if this planner can handle the given request.
func (p *Planner) CanHandle(ctx context.Context, request *agent.PlanRequest) bool {
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
	return agentTypeStr == string(plan.AgentTypeSpreadsheetGenerator)
}

// CreatePlan creates a plan for spreadsheet generation.
func (p *Planner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[spreadsheet_generator] CreatePlan started")
	config := p.parseConfig(request)
	log.Debug().Interface("config", config).Msg("[spreadsheet_generator] parsed config")
	estimatedSteps := 3

	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeSpreadsheetGenerator,
		EstimatedSteps: estimatedSteps,
		Config: &plan.PlanConfig{
			MaxRetries:        3,
			TimeoutPerStep:    300000000000,
			EnableFallback:    true,
			UserApproval:      false,
			StreamProgress:    true,
			ArtifactRetention: "session",
		},
	})
	if err != nil {
		return nil, err
	}

	contentTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    1,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Generate Data",
		Description: strPtr("Generate spreadsheet JSON with sheets, rows, and formulas"),
	})
	if err != nil {
		return nil, err
	}

	contentParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_spreadsheet_json",
		"description": "Generate structured spreadsheet JSON for openpyxl",
		"config": map[string]interface{}{
			"include_charts": config.IncludeCharts,
			"prompt":         request.UserMessage,
		},
	})
	_, err = p.planService.CreateStep(ctx, contentTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeLLMCall,
		InputParams: contentParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	execTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    2,
		TaskType:    plan.TaskTypeExecution,
		Title:       "Generate Spreadsheet",
		Description: strPtr("Generate XLSX file using skill execution"),
	})
	if err != nil {
		return nil, err
	}

	skillParams, _ := json.Marshal(map[string]interface{}{
		"skill_type": "spreadsheets",
		"options": map[string]interface{}{
			"include_charts": config.IncludeCharts,
			"title":          request.UserMessage,
		},
	})
	_, err = p.planService.CreateStep(ctx, execTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeSkillExecute,
		InputParams: skillParams,
		MaxRetries:  2,
	})
	if err != nil {
		return nil, err
	}

	artifactTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    3,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Finalize",
		Description: strPtr("Store spreadsheet as artifact"),
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_artifact",
		"description":   "Store spreadsheet as downloadable artifact",
		"artifact_type": "spreadsheet",
		"config": map[string]interface{}{
			"format":           "xlsx",
			"retention_policy": "session",
		},
	})
	_, err = p.planService.CreateStep(ctx, artifactTask.ID, plan.CreateStepParams{
		Sequence:    1,
		Action:      plan.ActionTypeArtifactCreate,
		InputParams: artifactParams,
		MaxRetries:  1,
	})
	if err != nil {
		return nil, err
	}

	planWithDetails, err := p.planService.GetPlanWithDetails(ctx, createdPlan.ID)
	if err != nil {
		return nil, err
	}

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

func (p *Planner) parseConfig(request *agent.PlanRequest) Config {
	config := DefaultConfig()
	if request.Metadata == nil {
		return config
	}
	if options, ok := request.Metadata["options"].(map[string]interface{}); ok {
		if includeCharts, ok := options["include_charts"].(bool); ok {
			config.IncludeCharts = includeCharts
		}
	}
	return config
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

var _ agent.Planner = (*Planner)(nil)
