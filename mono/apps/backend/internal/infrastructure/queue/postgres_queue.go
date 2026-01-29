package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
)

// PostgresQueue implements TaskQueue using the responses table.
type PostgresQueue struct {
	db  *gorm.DB
	log zerolog.Logger
}

// NewPostgresQueue creates a new PostgreSQL-backed task queue.
func NewPostgresQueue(db *gorm.DB, log zerolog.Logger) *PostgresQueue {
	return &PostgresQueue{
		db:  db,
		log: log.With().Str("component", "postgres-queue").Logger(),
	}
}

// Enqueue is not used directly - tasks are created via service.createAsync
func (q *PostgresQueue) Enqueue(ctx context.Context, task *Task) error {
	return fmt.Errorf("enqueue should not be called directly - use service.createAsync")
}

// Dequeue fetches the next queued task using FOR UPDATE SKIP LOCKED.
func (q *PostgresQueue) Dequeue(ctx context.Context) (*Task, error) {
	var entity dbschema.Response

	err := q.db.WithContext(ctx).
		Raw("SELECT * FROM responses WHERE status = ? AND background = ? ORDER BY queued_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED", "queued", true).
		Scan(&entity).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No tasks available
		}
		return nil, fmt.Errorf("dequeue task: %w", err)
	}

	// Check if no rows were returned (entity.ID will be 0)
	if entity.ID == 0 {
		return nil, nil // No tasks available
	}

	task := &Task{
		ResponseID: fmt.Sprintf("%d", entity.ID),
		PublicID:   entity.PublicID,
		UserID:     entity.UserID,
		Model:      entity.Model,
		QueuedAt:   *entity.QueuedAt,
	}

	return task, nil
}

// MarkProcessing updates the response status to in_progress.
func (q *PostgresQueue) MarkProcessing(ctx context.Context, publicID string) error {
	now := time.Now()
	result := q.db.WithContext(ctx).
		Model(&dbschema.Response{}).
		Where("public_id = ?", publicID).
		Updates(map[string]interface{}{
			"status":     "in_progress",
			"started_at": now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("mark processing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("response not found: %s", publicID)
	}

	return nil
}

// MarkCompleted updates the response status to completed.
func (q *PostgresQueue) MarkCompleted(ctx context.Context, publicID string) error {
	now := time.Now()
	result := q.db.WithContext(ctx).
		Model(&dbschema.Response{}).
		Where("public_id = ?", publicID).
		Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("mark completed: %w", result.Error)
	}

	return nil
}

// MarkFailed updates the response status to failed.
func (q *PostgresQueue) MarkFailed(ctx context.Context, publicID string, taskErr error) error {
	now := time.Now()
	errorJSON := map[string]interface{}{
		"code":    "task_execution_failed",
		"message": taskErr.Error(),
	}

	result := q.db.WithContext(ctx).
		Model(&dbschema.Response{}).
		Where("public_id = ?", publicID).
		Updates(map[string]interface{}{
			"status":     "failed",
			"error":      errorJSON,
			"failed_at":  now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("mark failed: %w", result.Error)
	}

	return nil
}

// GetQueueDepth returns the number of queued background tasks.
func (q *PostgresQueue) GetQueueDepth(ctx context.Context) (int64, error) {
	var count int64
	err := q.db.WithContext(ctx).
		Model(&dbschema.Response{}).
		Where("status = ?", "queued").
		Where("background = ?", true).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("get queue depth: %w", err)
	}

	return count, nil
}
