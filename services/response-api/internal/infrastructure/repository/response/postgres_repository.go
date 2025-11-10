package response

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"jan-server/services/response-api/internal/domain/llm"
	domain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
)

// PostgresRepository provides persistence for responses.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository constructs the repository.
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new response record.
func (r *PostgresRepository) Create(ctx context.Context, resp *domain.Response) error {
	entity, err := mapToEntity(resp)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return err
	}

	return mapFromEntity(entity, resp)
}

// Update persists changes to a response (status/output/etc).
func (r *PostgresRepository) Update(ctx context.Context, resp *domain.Response) error {
	entity, err := mapToEntity(resp)
	if err != nil {
		return err
	}
	entity.ID = resp.ID

	if err := r.db.WithContext(ctx).Model(&entities.Response{ID: resp.ID}).Updates(entity).Error; err != nil {
		return err
	}
	return nil
}

// FindByPublicID fetches a response and hydrates the domain model.
func (r *PostgresRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Response, error) {
	var entity entities.Response
	if err := r.db.WithContext(ctx).
		Preload("Conversation").
		Where("public_id = ?", publicID).
		First(&entity).Error; err != nil {
		return nil, err
	}

	resp := &domain.Response{}
	if err := mapFromEntity(&entity, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// MarkCancelled sets the status and timestamps for a cancelled response.
func (r *PostgresRepository) MarkCancelled(ctx context.Context, resp *domain.Response) error {
	now := time.Now()
	resp.Status = domain.StatusCancelled
	resp.CancelledAt = &now
	return r.Update(ctx, resp)
}

// RecordExecutions persists tool execution snapshot rows.
func (r *PostgresRepository) RecordExecutions(ctx context.Context, responseID uint, executions []tool.Execution) error {
	if len(executions) == 0 {
		return nil
	}

	rows := make([]entities.ToolExecution, 0, len(executions))
	for _, exec := range executions {
		args, err := json.Marshal(exec.Arguments)
		if err != nil {
			return fmt.Errorf("marshal tool arguments: %w", err)
		}
		var result datatypes.JSON
		if exec.Result != nil {
			if result, err = json.Marshal(exec.Result); err != nil {
				return fmt.Errorf("marshal tool result: %w", err)
			}
		}
		rows = append(rows, entities.ToolExecution{
			ResponseID:     responseID,
			CallID:         exec.CallID,
			ToolName:       exec.ToolName,
			Arguments:      args,
			Result:         result,
			Status:         string(exec.Status),
			ErrorMessage:   exec.ErrorMessage,
			ExecutionOrder: exec.ExecutionOrder,
		})
	}

	return r.db.WithContext(ctx).Create(&rows).Error
}

func mapToEntity(resp *domain.Response) (*entities.Response, error) {
	input, err := marshalJSON(resp.Input)
	if err != nil {
		return nil, fmt.Errorf("marshal response input: %w", err)
	}
	output, err := marshalJSON(resp.Output)
	if err != nil {
		return nil, fmt.Errorf("marshal response output: %w", err)
	}
	metadata, err := marshalJSON(resp.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	usage, err := marshalJSON(resp.Usage)
	if err != nil {
		return nil, fmt.Errorf("marshal usage: %w", err)
	}
	errJSON, err := marshalJSON(resp.Error)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return &entities.Response{
		PublicID:           resp.PublicID,
		UserID:             resp.UserID,
		Model:              resp.Model,
		SystemPrompt:       resp.SystemPrompt,
		Input:              input,
		Output:             output,
		Status:             string(resp.Status),
		Stream:             resp.Stream,
		Metadata:           metadata,
		Usage:              usage,
		Error:              errJSON,
		ConversationID:     resp.ConversationID,
		PreviousResponseID: resp.PreviousResponseID,
		Object:             resp.Object,
		CompletedAt:        resp.CompletedAt,
		CancelledAt:        resp.CancelledAt,
		FailedAt:           resp.FailedAt,
	}, nil
}

func mapFromEntity(entity *entities.Response, resp *domain.Response) error {
	resp.ID = entity.ID
	resp.PublicID = entity.PublicID
	resp.UserID = entity.UserID
	resp.Model = entity.Model
	resp.SystemPrompt = entity.SystemPrompt
	resp.Status = domain.Status(entity.Status)
	resp.Stream = entity.Stream
	resp.ConversationID = entity.ConversationID
	resp.PreviousResponseID = entity.PreviousResponseID
	resp.CreatedAt = entity.CreatedAt
	resp.UpdatedAt = entity.UpdatedAt
	resp.CompletedAt = entity.CompletedAt
	resp.CancelledAt = entity.CancelledAt
	resp.FailedAt = entity.FailedAt
	resp.Object = entity.Object

	if err := json.Unmarshal(entity.Input, &resp.Input); err != nil {
		return fmt.Errorf("unmarshal input: %w", err)
	}
	if len(entity.Output) > 0 {
		if err := json.Unmarshal(entity.Output, &resp.Output); err != nil {
			return fmt.Errorf("unmarshal output: %w", err)
		}
	}
	if len(entity.Metadata) > 0 {
		if err := json.Unmarshal(entity.Metadata, &resp.Metadata); err != nil {
			return fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	if len(entity.Usage) > 0 {
		var usage llm.Usage
		if err := json.Unmarshal(entity.Usage, &usage); err == nil {
			resp.Usage = &usage
		}
	}
	if len(entity.Error) > 0 {
		var errDetails domain.ErrorDetails
		if err := json.Unmarshal(entity.Error, &errDetails); err == nil {
			resp.Error = &errDetails
		}
	}

	if resp.ConversationPublicID == nil && entity.Conversation != nil {
		resp.ConversationPublicID = &entity.Conversation.PublicID
	}

	return nil
}

func marshalJSON(value interface{}) (datatypes.JSON, error) {
	if value == nil {
		return datatypes.JSON([]byte("null")), nil
	}
	bytes, err := json.Marshal(value)
	return datatypes.JSON(bytes), err
}
