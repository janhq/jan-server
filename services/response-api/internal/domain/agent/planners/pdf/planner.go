// Package pdf contains the PDF generator planner.
package pdf

import (
	"context"
	"encoding/json"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// Planner creates execution plans for PDF generation.
type Planner struct {
	planService     plan.Service
	artifactService artifact.Service
}

// Config holds configuration for PDF generation.
type Config struct {
	PageSize    string `json:"page_size"`
	Orientation string `json:"orientation"`
}

// DefaultConfig returns defaults.
func DefaultConfig() Config {
	return Config{
		PageSize:    "A4",
		Orientation: "portrait",
	}
}

// NewPlanner creates a new PDF generator planner.
func NewPlanner(planService plan.Service, artifactService artifact.Service) *Planner {
	return &Planner{
		planService:     planService,
		artifactService: artifactService,
	}
}

// Name returns the planner's unique identifier.
func (p *Planner) Name() string {
	return string(plan.AgentTypePDFGenerator)
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
	return agentTypeStr == string(plan.AgentTypePDFGenerator)
}

// CreatePlan creates a plan for PDF generation.
func (p *Planner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[pdf_generator] CreatePlan started")
	config := p.parseConfig(request)
	log.Debug().Interface("config", config).Msg("[pdf_generator] parsed config")
	estimatedSteps := 3

	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypePDFGenerator,
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
		Title:       "Generate Content",
		Description: strPtr("Generate PDF sections as structured JSON"),
	})
	if err != nil {
		return nil, err
	}

	contentParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_pdf_json",
		"description": "Generate structured PDF JSON for reportlab",
		"config": map[string]interface{}{
			"page_size":   config.PageSize,
			"orientation": config.Orientation,
			"prompt":      request.UserMessage,
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
		Title:       "Generate PDF",
		Description: strPtr("Generate PDF file using skill execution"),
	})
	if err != nil {
		return nil, err
	}

	skillParams, _ := json.Marshal(map[string]interface{}{
		"skill_type": "pdfs",
		"options": map[string]interface{}{
			"page_size":   config.PageSize,
			"orientation": config.Orientation,
			"title":       request.UserMessage,
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
		Description: strPtr("Store PDF as artifact"),
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_artifact",
		"description":   "Store PDF as downloadable artifact",
		"artifact_type": "document",
		"config": map[string]interface{}{
			"format":           "pdf",
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
		if pageSize, ok := options["page_size"].(string); ok {
			config.PageSize = pageSize
		}
		if orientation, ok := options["orientation"].(string); ok {
			config.Orientation = orientation
		}
	}
	return config
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

var _ agent.Planner = (*Planner)(nil)
