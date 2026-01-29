package webhook

import (
	"context"
	"time"
)

// Service handles webhook notifications for response events.
type Service interface {
	// NotifyCompleted sends a webhook notification when a response completes.
	NotifyCompleted(ctx context.Context, responseID string, output interface{}, metadata map[string]interface{}, completedAt *time.Time) error

	// NotifyFailed sends a webhook notification when a response fails.
	NotifyFailed(ctx context.Context, responseID string, errorCode string, errorMessage string, metadata map[string]interface{}) error
}

// ErrorDetails contains machine readable error info.
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WebhookPayload is the structure sent to webhook URLs.
type WebhookPayload struct {
	ID          string                 `json:"id"`
	Event       string                 `json:"event"` // "response.completed" or "response.failed"
	Status      string                 `json:"status"`
	Output      interface{}            `json:"output,omitempty"`
	Error       *ErrorDetails          `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CompletedAt *string                `json:"completed_at,omitempty"`
}
