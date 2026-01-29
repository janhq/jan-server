package queue

import (
	"context"
	"time"
)

// Task represents a background task to be processed.
type Task struct {
	ResponseID string
	PublicID   string
	UserID     string
	Model      string
	QueuedAt   time.Time
}

// TaskQueue defines the interface for task queue operations.
type TaskQueue interface {
	// Enqueue adds a task to the queue
	Enqueue(ctx context.Context, task *Task) error

	// Dequeue fetches the next available task using SELECT FOR UPDATE SKIP LOCKED
	Dequeue(ctx context.Context) (*Task, error)

	// MarkProcessing updates task status to in_progress
	MarkProcessing(ctx context.Context, taskID string) error

	// MarkCompleted updates task status to completed
	MarkCompleted(ctx context.Context, taskID string) error

	// MarkFailed updates task status to failed
	MarkFailed(ctx context.Context, taskID string, err error) error

	// GetQueueDepth returns the number of queued tasks
	GetQueueDepth(ctx context.Context) (int64, error)
}
