// Package doc contains the document generator planner.
package doc

import (
	"context"
	"encoding/json"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"

	"github.com/rs/zerolog/log"
)

// Planner creates execution plans for document generation.
type Planner struct {
	planService     plan.Service
	artifactService artifact.Service
}

// Config holds configuration for document generation.
type Config struct {
	Template      string `json:"template"`
	Format        string `json:"format"`         // docx, pdf
	ResearchDepth string `json:"research_depth"` // minimal, standard, deep
}

// DefaultConfig returns defaults.
func DefaultConfig() Config {
	return Config{
		Template:      "professional",
		Format:        "docx",
		ResearchDepth: "standard",
	}
}

// NewPlanner creates a new doc generator planner.
func NewPlanner(planService plan.Service, artifactService artifact.Service) *Planner {
	return &Planner{
		planService:     planService,
		artifactService: artifactService,
	}
}

// Name returns the planner's unique identifier.
func (p *Planner) Name() string {
	return string(plan.AgentTypeDocGenerator)
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
	return agentTypeStr == string(plan.AgentTypeDocGenerator)
}

// CreatePlan creates a plan for document generation.
func (p *Planner) CreatePlan(ctx context.Context, request *agent.PlanRequest) (*agent.PlanResult, error) {
	log.Debug().Interface("request", request).Msg("[doc_generator] CreatePlan started")
	config := p.parseConfig(request)
	log.Debug().Interface("config", config).Msg("[doc_generator] parsed config")
	estimatedSteps := 3

	createdPlan, err := p.planService.Create(ctx, plan.CreateParams{
		ResponseID:     request.ResponseID,
		Model:          request.Model,
		AgentType:      plan.AgentTypeDocGenerator,
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
	log.Debug().Str("plan_id", createdPlan.ID).Msg("[doc_generator] plan created")

	// Task 1: Content Generation
	log.Debug().Msg("[doc_generator] creating content generation task")
	contentTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    1,
		TaskType:    plan.TaskTypeGeneration,
		Title:       "Generate Content",
		Description: strPtr("Generate document sections as structured JSON"),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("task_id", contentTask.ID).Msg("[doc_generator] content task created")

	contentParams, _ := json.Marshal(map[string]interface{}{
		"action":      "generate_doc_json",
		"description": "Generate structured document JSON for python-docx",
		"config": map[string]interface{}{
			"template": config.Template,
			"format":   config.Format,
			"prompt":   request.UserMessage,
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

	log.Debug().Msg("[doc_generator] creating skill execution task")
	// Task 2: Skill Execution
	execTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    2,
		TaskType:    plan.TaskTypeExecution,
		Title:       "Generate Document",
		Description: strPtr("Generate DOCX file using skill execution"),
	})
	if err != nil {
		return nil, err
	}

	skillParams, _ := json.Marshal(map[string]interface{}{
		"skill_type": "docs",
		"options": map[string]interface{}{
			"template": config.Template,
			"format":   config.Format,
			"title":    request.UserMessage,
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

	// Task 3: Artifact Creation
	artifactTask, err := p.planService.CreateTask(ctx, createdPlan.ID, plan.CreateTaskParams{
		Sequence:    3,
		TaskType:    plan.TaskTypeFinalization,
		Title:       "Finalize",
		Description: strPtr("Store document as artifact"),
	})
	if err != nil {
		return nil, err
	}

	artifactParams, _ := json.Marshal(map[string]interface{}{
		"action":        "store_artifact",
		"description":   "Store document as downloadable artifact",
		"artifact_type": "document",
		"config": map[string]interface{}{
			"format":           config.Format,
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
		if template, ok := options["template"].(string); ok {
			config.Template = template
		}
		if format, ok := options["format"].(string); ok {
			config.Format = format
		}
		if depth, ok := options["research_depth"].(string); ok {
			config.ResearchDepth = depth
		}
	}
	return config
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

var _ agent.Planner = (*Planner)(nil)
