package plan

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	domain "jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
	"jan-server/services/response-api/internal/utils/platformerrors"
)

// PostgresRepository provides persistence for plans.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository constructs the repository.
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new plan record.
func (r *PostgresRepository) Create(ctx context.Context, plan *domain.Plan) error {
	responseID, err := r.resolveResponseID(ctx, plan.ResponseID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"response not found for plan creation",
			err,
			"plan-create-response-001",
		)
	}

	currentTaskID, err := r.resolveTaskID(ctx, plan.CurrentTaskID)
	if err != nil {
		return err
	}

	finalArtifactID, err := r.resolveArtifactID(ctx, plan.FinalArtifactID)
	if err != nil {
		return err
	}

	entity, err := mapPlanToEntity(plan, responseID, currentTaskID, finalArtifactID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeInternal,
			"failed to map plan to entity",
			err,
			"plan-create-map-001",
		)
	}

	if entity.PublicID == "" {
		entity.PublicID = uuid.New().String()
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create plan",
			err,
			"plan-create-db-001",
		)
	}

	plan.ID = entity.PublicID
	return nil
}

// Update persists changes to a plan.
func (r *PostgresRepository) Update(ctx context.Context, plan *domain.Plan) error {
	responseID, err := r.resolveResponseID(ctx, plan.ResponseID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"response not found for plan update",
			err,
			"plan-update-response-001",
		)
	}

	currentTaskID, err := r.resolveTaskID(ctx, plan.CurrentTaskID)
	if err != nil {
		return err
	}

	finalArtifactID, err := r.resolveArtifactID(ctx, plan.FinalArtifactID)
	if err != nil {
		return err
	}

	entity, err := mapPlanToEntity(plan, responseID, currentTaskID, finalArtifactID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeInternal,
			"failed to map plan to entity for update",
			err,
			"plan-update-map-001",
		)
	}

	updates := map[string]interface{}{
		"response_id":      entity.ResponseID,
		"status":           entity.Status,
		"progress":         entity.Progress,
		"agent_type":       entity.AgentType,
		"planning_config":  entity.PlanningConfig,
		"estimated_steps":  entity.EstimatedSteps,
		"completed_steps":  entity.CompletedSteps,
		"current_task_id":  entity.CurrentTaskID,
		"final_artifact_id": entity.FinalArtifactID,
		"user_selection":   entity.UserSelection,
		"error_message":    entity.ErrorMessage,
		"updated_at":       entity.UpdatedAt,
		"completed_at":     entity.CompletedAt,
	}

	if err := r.db.WithContext(ctx).
		Model(&entities.Plan{}).
		Where("public_id = ?", plan.ID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update plan",
			err,
			"plan-update-db-001",
		)
	}
	return nil
}

// FindByID fetches a plan by public ID.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*domain.Plan, error) {
	var entity entities.Plan
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Where("public_id = ?", id).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"plan not found",
				err,
				"plan-find-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find plan",
			err,
			"plan-find-db-001",
		)
	}

	plan, err := mapPlanFromEntity(&entity)
	if err != nil {
		return nil, err
	}

	if err := r.hydratePlanRefs(ctx, plan, &entity); err != nil {
		return nil, err
	}

	return plan, nil
}

// FindByResponseID fetches a plan by response ID.
func (r *PostgresRepository) FindByResponseID(ctx context.Context, responseID string) (*domain.Plan, error) {
	var entity entities.Plan
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Joins("JOIN responses ON responses.id = plans.response_id").
		Where("responses.public_id = ?", responseID).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"plan not found for response",
				err,
				"plan-find-response-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find plan by response",
			err,
			"plan-find-response-db-001",
		)
	}

	plan, err := mapPlanFromEntity(&entity)
	if err != nil {
		return nil, err
	}

	if err := r.hydratePlanRefs(ctx, plan, &entity); err != nil {
		return nil, err
	}

	return plan, nil
}

// List retrieves plans matching the filter.
func (r *PostgresRepository) List(ctx context.Context, filter *domain.Filter) ([]*domain.Plan, int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.Plan{}).Preload("Response")

	if filter.ResponseID != nil {
		query = query.Joins("JOIN responses ON responses.id = plans.response_id").
			Where("responses.public_id = ?", *filter.ResponseID)
	}
	if filter.Status != nil {
		query = query.Where("plans.status = ?", string(*filter.Status))
	}
	if filter.AgentType != nil {
		query = query.Where("plans.agent_type = ?", string(*filter.AgentType))
	}
	if filter.CreatedAfter != nil {
		query = query.Where("plans.created_at > ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("plans.created_at < ?", *filter.CreatedBefore)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count plans",
			err,
			"plan-list-count-001",
		)
	}

	var entities []entities.Plan
	if err := query.
		Order("plans.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&entities).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list plans",
			err,
			"plan-list-db-001",
		)
	}

	plans := make([]*domain.Plan, 0, len(entities))
	for _, e := range entities {
		p, err := mapPlanFromEntity(&e)
		if err != nil {
			return nil, 0, err
		}
		if err := r.hydratePlanRefs(ctx, p, &e); err != nil {
			return nil, 0, err
		}
		plans = append(plans, p)
	}

	return plans, total, nil
}

// Delete removes a plan.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Where("public_id = ?", id).
		Delete(&entities.Plan{}).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete plan",
			err,
			"plan-delete-db-001",
		)
	}
	return nil
}

// CreateTask inserts a new task record.
func (r *PostgresRepository) CreateTask(ctx context.Context, task *domain.Task) error {
	// Get plan internal ID
	var plan entities.Plan
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", task.PlanID).
		First(&plan).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"plan not found for task creation",
			err,
			"task-create-plan-001",
		)
	}

	entity := mapTaskToEntity(task, plan.ID)
	if entity.PublicID == "" {
		entity.PublicID = uuid.New().String()
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create task",
			err,
			"task-create-db-001",
		)
	}

	task.ID = entity.PublicID
	return nil
}

// UpdateTask persists changes to a task.
func (r *PostgresRepository) UpdateTask(ctx context.Context, task *domain.Task) error {
	updates := map[string]interface{}{
		"status":        string(task.Status),
		"title":         task.Title,
		"description":   task.Description,
		"error_message": task.ErrorMessage,
		"updated_at":    task.UpdatedAt,
		"completed_at":  task.CompletedAt,
	}

	if err := r.db.WithContext(ctx).
		Model(&entities.PlanTask{}).
		Where("public_id = ?", task.ID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update task",
			err,
			"task-update-db-001",
		)
	}
	return nil
}

// FindTaskByID fetches a task by public ID.
func (r *PostgresRepository) FindTaskByID(ctx context.Context, id string) (*domain.Task, error) {
	var entity entities.PlanTask
	if err := r.db.WithContext(ctx).
		Where("public_id = ?", id).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"task not found",
				err,
				"task-find-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find task",
			err,
			"task-find-db-001",
		)
	}

	// Get plan public ID
	var plan entities.Plan
	r.db.WithContext(ctx).Select("public_id").Where("id = ?", entity.PlanID).First(&plan)

	return mapTaskFromEntity(&entity, plan.PublicID), nil
}

// ListTasksByPlanID retrieves all tasks for a plan.
func (r *PostgresRepository) ListTasksByPlanID(ctx context.Context, planID string) ([]*domain.Task, error) {
	var plan entities.Plan
	if err := r.db.WithContext(ctx).
		Select("id, public_id").
		Where("public_id = ?", planID).
		First(&plan).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"plan not found",
			err,
			"task-list-plan-001",
		)
	}

	var entities []entities.PlanTask
	if err := r.db.WithContext(ctx).
		Where("plan_id = ?", plan.ID).
		Order("sequence ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list tasks",
			err,
			"task-list-db-001",
		)
	}

	tasks := make([]*domain.Task, 0, len(entities))
	for _, e := range entities {
		tasks = append(tasks, mapTaskFromEntity(&e, planID))
	}

	return tasks, nil
}

// CreateStep inserts a new step record.
func (r *PostgresRepository) CreateStep(ctx context.Context, step *domain.Step) error {
	// Get task internal ID
	var task entities.PlanTask
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", step.TaskID).
		First(&task).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"task not found for step creation",
			err,
			"step-create-task-001",
		)
	}

	entity := mapStepToEntity(step, task.ID)
	if entity.PublicID == "" {
		entity.PublicID = uuid.New().String()
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create step",
			err,
			"step-create-db-001",
		)
	}

	step.ID = entity.PublicID
	return nil
}

// UpdateStep persists changes to a step.
func (r *PostgresRepository) UpdateStep(ctx context.Context, step *domain.Step) error {
	outputData, _ := marshalJSON(step.OutputData)
	var errorSeverity *string
	if step.ErrorSeverity != "" {
		s := string(step.ErrorSeverity)
		errorSeverity = &s
	}

	updates := map[string]interface{}{
		"status":         string(step.Status),
		"output_data":    outputData,
		"retry_count":    step.RetryCount,
		"error_message":  step.ErrorMessage,
		"error_severity": errorSeverity,
		"duration_ms":    step.DurationMs,
		"started_at":     step.StartedAt,
		"completed_at":   step.CompletedAt,
	}

	if err := r.db.WithContext(ctx).
		Model(&entities.PlanStep{}).
		Where("public_id = ?", step.ID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update step",
			err,
			"step-update-db-001",
		)
	}
	return nil
}

// FindStepByID fetches a step by public ID.
func (r *PostgresRepository) FindStepByID(ctx context.Context, id string) (*domain.Step, error) {
	var entity entities.PlanStep
	if err := r.db.WithContext(ctx).
		Where("public_id = ?", id).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"step not found",
				err,
				"step-find-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find step",
			err,
			"step-find-db-001",
		)
	}

	// Get task public ID
	var task entities.PlanTask
	r.db.WithContext(ctx).Select("public_id").Where("id = ?", entity.TaskID).First(&task)

	return mapStepFromEntity(&entity, task.PublicID), nil
}

// ListStepsByTaskID retrieves all steps for a task.
func (r *PostgresRepository) ListStepsByTaskID(ctx context.Context, taskID string) ([]*domain.Step, error) {
	var task entities.PlanTask
	if err := r.db.WithContext(ctx).
		Select("id, public_id").
		Where("public_id = ?", taskID).
		First(&task).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"task not found",
			err,
			"step-list-task-001",
		)
	}

	var entities []entities.PlanStep
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", task.ID).
		Order("sequence ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list steps",
			err,
			"step-list-db-001",
		)
	}

	steps := make([]*domain.Step, 0, len(entities))
	for _, e := range entities {
		steps = append(steps, mapStepFromEntity(&e, taskID))
	}

	return steps, nil
}

// CreateStepDetail inserts a new step detail record.
func (r *PostgresRepository) CreateStepDetail(ctx context.Context, detail *domain.StepDetail) error {
	// Get step internal ID
	var step entities.PlanStep
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", detail.StepID).
		First(&step).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"step not found for detail creation",
			err,
			"detail-create-step-001",
		)
	}

	metadata, _ := marshalJSON(detail.Metadata)
	entity := &entities.PlanStepDetail{
		PublicID:   uuid.New().String(),
		StepID:     step.ID,
		DetailType: string(detail.DetailType),
		ToolCallID: detail.ToolCallID,
		Metadata:   metadata,
		CreatedAt:  detail.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create step detail",
			err,
			"detail-create-db-001",
		)
	}

	detail.ID = entity.PublicID
	return nil
}

// ListDetailsByStepID retrieves all details for a step.
func (r *PostgresRepository) ListDetailsByStepID(ctx context.Context, stepID string) ([]*domain.StepDetail, error) {
	var step entities.PlanStep
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", stepID).
		First(&step).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"step not found",
			err,
			"detail-list-step-001",
		)
	}

	var entities []entities.PlanStepDetail
	if err := r.db.WithContext(ctx).
		Where("step_id = ?", step.ID).
		Order("created_at ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list step details",
			err,
			"detail-list-db-001",
		)
	}

	details := make([]*domain.StepDetail, 0, len(entities))
	for _, e := range entities {
		details = append(details, mapDetailFromEntity(&e, stepID))
	}

	return details, nil
}

// GetProgress retrieves the current progress of a plan.
func (r *PostgresRepository) GetProgress(ctx context.Context, planID string) (*domain.PlanProgress, error) {
	plan, err := r.FindByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Count completed steps
	var completedCount int64
	if err := r.db.WithContext(ctx).
		Model(&entities.PlanStep{}).
		Joins("JOIN plan_tasks ON plan_tasks.id = plan_steps.task_id").
		Joins("JOIN plans ON plans.id = plan_tasks.plan_id").
		Where("plans.public_id = ?", planID).
		Where("plan_steps.status = ?", "completed").
		Count(&completedCount).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count completed steps",
			err,
			"plan-progress-completed-001",
		)
	}

	// Count failed steps
	var failedCount int64
	if err := r.db.WithContext(ctx).
		Model(&entities.PlanStep{}).
		Joins("JOIN plan_tasks ON plan_tasks.id = plan_steps.task_id").
		Joins("JOIN plans ON plans.id = plan_tasks.plan_id").
		Where("plans.public_id = ?", planID).
		Where("plan_steps.status = ?", "failed").
		Count(&failedCount).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count failed steps",
			err,
			"plan-progress-failed-001",
		)
	}

	progress := &domain.PlanProgress{
		PlanID:         plan.ID,
		Status:         plan.Status,
		Progress:       plan.Progress,
		EstimatedSteps: plan.EstimatedSteps,
		CompletedSteps: int(completedCount),
		FailedSteps:    int(failedCount),
	}

	// Get current task info if available
	if plan.CurrentTaskID != nil {
		task, err := r.FindTaskByID(ctx, *plan.CurrentTaskID)
		if err == nil {
			progress.CurrentTask = &domain.TaskProgress{
				TaskID: task.ID,
				Title:  task.Title,
				Status: task.Status,
			}
		}
	}

	return progress, nil
}

// FindPlanWithDetails retrieves a plan with all its tasks and steps.
func (r *PostgresRepository) FindPlanWithDetails(ctx context.Context, id string) (*domain.Plan, error) {
	var entity entities.Plan
	if err := r.db.WithContext(ctx).
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence ASC")
		}).
		Preload("Tasks.Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence ASC")
		}).
		Where("public_id = ?", id).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"plan not found",
				err,
				"plan-details-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find plan with details",
			err,
			"plan-details-db-001",
		)
	}

	plan, err := mapPlanFromEntityWithDetails(&entity)
	if err != nil {
		return nil, err
	}

	if err := r.hydratePlanRefs(ctx, plan, &entity); err != nil {
		return nil, err
	}

	return plan, nil
}

// Mapping functions

func mapPlanToEntity(plan *domain.Plan, responseID uint, currentTaskID, finalArtifactID *uint) (*entities.Plan, error) {
	config, err := json.Marshal(plan.PlanningConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal planning config: %w", err)
	}

	var userSelection datatypes.JSON
	if plan.UserSelection != nil {
		userSelection = datatypes.JSON([]byte(*plan.UserSelection))
	}

	return &entities.Plan{
		PublicID:       plan.ID,
		ResponseID:     responseID,
		Status:         string(plan.Status),
		Progress:       plan.Progress,
		AgentType:      string(plan.AgentType),
		PlanningConfig: datatypes.JSON(config),
		EstimatedSteps: plan.EstimatedSteps,
		CompletedSteps: plan.CompletedSteps,
		CurrentTaskID:  currentTaskID,
		FinalArtifactID: finalArtifactID,
		UserSelection:  userSelection,
		ErrorMessage:   plan.ErrorMessage,
		CreatedAt:      plan.CreatedAt,
		UpdatedAt:      plan.UpdatedAt,
		CompletedAt:    plan.CompletedAt,
	}, nil
}

func mapPlanFromEntity(entity *entities.Plan) (*domain.Plan, error) {
	plan := &domain.Plan{
		ID:             entity.PublicID,
		Status:         status.Status(entity.Status),
		Progress:       entity.Progress,
		AgentType:      domain.AgentType(entity.AgentType),
		EstimatedSteps: entity.EstimatedSteps,
		CompletedSteps: entity.CompletedSteps,
		ErrorMessage:   entity.ErrorMessage,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
		CompletedAt:    entity.CompletedAt,
	}

	if entity.Response != nil {
		plan.ResponseID = entity.Response.PublicID
	}

	if len(entity.PlanningConfig) > 0 {
		if err := json.Unmarshal(entity.PlanningConfig, &plan.PlanningConfig); err != nil {
			return nil, fmt.Errorf("unmarshal planning config: %w", err)
		}
	}

	if len(entity.UserSelection) > 0 {
		s := string(entity.UserSelection)
		plan.UserSelection = &s
	}

	return plan, nil
}

func mapPlanFromEntityWithDetails(entity *entities.Plan) (*domain.Plan, error) {
	plan, err := mapPlanFromEntity(entity)
	if err != nil {
		return nil, err
	}

	// Map tasks
	plan.Tasks = make([]domain.Task, 0, len(entity.Tasks))
	for _, taskEntity := range entity.Tasks {
		task := *mapTaskFromEntity(&taskEntity, plan.ID)

		// Map steps
		task.Steps = make([]domain.Step, 0, len(taskEntity.Steps))
		for _, stepEntity := range taskEntity.Steps {
			step := *mapStepFromEntity(&stepEntity, task.ID)
			task.Steps = append(task.Steps, step)
		}

		plan.Tasks = append(plan.Tasks, task)
	}

	return plan, nil
}

func (r *PostgresRepository) resolveResponseID(ctx context.Context, publicID string) (uint, error) {
	if publicID == "" {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeValidation,
			"response_id is required",
			nil,
			"plan-response-missing-001",
		)
	}
	var response entities.Response
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", publicID).First(&response).Error; err != nil {
		return 0, err
	}
	return response.ID, nil
}

func (r *PostgresRepository) resolveTaskID(ctx context.Context, publicID *string) (*uint, error) {
	if publicID == nil || *publicID == "" {
		return nil, nil
	}
	var task entities.PlanTask
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", *publicID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"task not found",
				err,
				"plan-current-task-001",
			)
		}
		return nil, err
	}
	return &task.ID, nil
}

func (r *PostgresRepository) resolveArtifactID(ctx context.Context, publicID *string) (*uint, error) {
	if publicID == nil || *publicID == "" {
		return nil, nil
	}
	var artifact entities.Artifact
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", *publicID).First(&artifact).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"artifact not found",
				err,
				"plan-final-artifact-001",
			)
		}
		return nil, err
	}
	return &artifact.ID, nil
}

func (r *PostgresRepository) hydratePlanRefs(ctx context.Context, plan *domain.Plan, entity *entities.Plan) error {
	if plan.ResponseID == "" && entity.ResponseID != 0 {
		var response entities.Response
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", entity.ResponseID).First(&response).Error; err != nil {
			return err
		}
		plan.ResponseID = response.PublicID
	}

	if entity.CurrentTaskID != nil {
		var task entities.PlanTask
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", *entity.CurrentTaskID).First(&task).Error; err != nil {
			return err
		}
		plan.CurrentTaskID = &task.PublicID
	}

	if entity.FinalArtifactID != nil {
		var artifact entities.Artifact
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", *entity.FinalArtifactID).First(&artifact).Error; err != nil {
			return err
		}
		plan.FinalArtifactID = &artifact.PublicID
	}

	return nil
}

func mapTaskToEntity(task *domain.Task, planID uint) *entities.PlanTask {
	return &entities.PlanTask{
		PublicID:     task.ID,
		PlanID:       planID,
		Sequence:     task.Sequence,
		TaskType:     string(task.TaskType),
		Status:       string(task.Status),
		Title:        task.Title,
		Description:  task.Description,
		ErrorMessage: task.ErrorMessage,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
		CompletedAt:  task.CompletedAt,
	}
}

func mapTaskFromEntity(entity *entities.PlanTask, planID string) *domain.Task {
	return &domain.Task{
		ID:           entity.PublicID,
		PlanID:       planID,
		Sequence:     entity.Sequence,
		TaskType:     domain.TaskType(entity.TaskType),
		Status:       status.Status(entity.Status),
		Title:        entity.Title,
		Description:  entity.Description,
		ErrorMessage: entity.ErrorMessage,
		CreatedAt:    entity.CreatedAt,
		UpdatedAt:    entity.UpdatedAt,
		CompletedAt:  entity.CompletedAt,
	}
}

func mapStepToEntity(step *domain.Step, taskID uint) *entities.PlanStep {
	inputParams, _ := marshalJSON(step.InputParams)
	outputData, _ := marshalJSON(step.OutputData)

	var errorSeverity *string
	if step.ErrorSeverity != "" {
		s := string(step.ErrorSeverity)
		errorSeverity = &s
	}

	return &entities.PlanStep{
		PublicID:      step.ID,
		TaskID:        taskID,
		Sequence:      step.Sequence,
		Action:        string(step.Action),
		Status:        string(step.Status),
		InputParams:   inputParams,
		OutputData:    outputData,
		RetryCount:    step.RetryCount,
		MaxRetries:    step.MaxRetries,
		ErrorMessage:  step.ErrorMessage,
		ErrorSeverity: errorSeverity,
		DurationMs:    step.DurationMs,
		StartedAt:     step.StartedAt,
		CompletedAt:   step.CompletedAt,
	}
}

func mapStepFromEntity(entity *entities.PlanStep, taskID string) *domain.Step {
	step := &domain.Step{
		ID:           entity.PublicID,
		TaskID:       taskID,
		Sequence:     entity.Sequence,
		Action:       domain.ActionType(entity.Action),
		Status:       status.Status(entity.Status),
		InputParams:  json.RawMessage(entity.InputParams),
		OutputData:   json.RawMessage(entity.OutputData),
		RetryCount:   entity.RetryCount,
		MaxRetries:   entity.MaxRetries,
		ErrorMessage: entity.ErrorMessage,
		DurationMs:   entity.DurationMs,
		StartedAt:    entity.StartedAt,
		CompletedAt:  entity.CompletedAt,
	}

	if entity.ErrorSeverity != nil {
		step.ErrorSeverity = status.ErrorSeverity(*entity.ErrorSeverity)
	}

	return step
}

func mapDetailFromEntity(entity *entities.PlanStepDetail, stepID string) *domain.StepDetail {
	detail := &domain.StepDetail{
		ID:         entity.PublicID,
		StepID:     stepID,
		DetailType: domain.DetailType(entity.DetailType),
		ToolCallID: entity.ToolCallID,
		Metadata:   json.RawMessage(entity.Metadata),
		CreatedAt:  entity.CreatedAt,
	}

	if entity.ConversationItemID != nil {
		// Would need to look up public ID if needed
	}

	return detail
}

func marshalJSON(value interface{}) (datatypes.JSON, error) {
	if value == nil {
		return datatypes.JSON([]byte("null")), nil
	}
	bytes, err := json.Marshal(value)
	return datatypes.JSON(bytes), err
}
