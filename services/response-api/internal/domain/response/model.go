package response

import (
	"context"
	"time"

	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/tool"
)

// Status represents the lifecycle of a response.
type Status string

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Response is the main aggregate persisted to the database.
type Response struct {
	ID                   uint                   `json:"-"`
	PublicID             string                 `json:"id"`
	Object               string                 `json:"object"`
	UserID               string                 `json:"user_id"`
	Model                string                 `json:"model"`
	SystemPrompt         *string                `json:"system_prompt,omitempty"`
	Input                interface{}            `json:"input"`
	Output               interface{}            `json:"output,omitempty"`
	Status               Status                 `json:"status"`
	Stream               bool                   `json:"stream"`
	Background           bool                   `json:"background"`
	Store                bool                   `json:"store"`
	APIKey               *string                `json:"-"` // API key (X-API-Key or Bearer token) for background LLM calls
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	Usage                *llm.Usage             `json:"usage,omitempty"`
	Error                *ErrorDetails          `json:"error,omitempty"`
	ConversationID       *uint                  `json:"-"`
	ConversationPublicID *string                `json:"conversation_id,omitempty"`
	PreviousResponseID   *string                `json:"previous_response_id,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	QueuedAt             *time.Time             `json:"queued_at,omitempty"`
	StartedAt            *time.Time             `json:"started_at,omitempty"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty"`
	CancelledAt          *time.Time             `json:"cancelled_at,omitempty"`
	FailedAt             *time.Time             `json:"failed_at,omitempty"`
}

// ErrorDetails contains machine readable error info surfaced to clients.
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateParams contains inputs collected from the HTTP layer.
type CreateParams struct {
	UserID             string
	Model              string
	Input              interface{}
	RequestID          string
	SystemPrompt       *string
	Temperature        *float64
	MaxTokens          *int
	Stream             bool
	Background         bool
	Store              bool
	APIKey             *string // API key (X-API-Key or Bearer token) for background LLM calls
	ToolChoice         *llm.ToolChoice
	Tools              []llm.ToolDefinition
	PreviousResponseID *string
	ConversationID     *string
	Metadata           map[string]interface{}
	StreamObserver     StreamObserver
}

// Service exposes the Response domain operations.
type Service interface {
	Create(ctx context.Context, params CreateParams) (*Response, error)
	GetByPublicID(ctx context.Context, publicID string) (*Response, error)
	GetWithPlanDetails(ctx context.Context, publicID string) (*ResponseWithPlan, error)
	Cancel(ctx context.Context, publicID string) (*Response, error)
	ListConversationItems(ctx context.Context, publicID string) ([]ConversationItem, error)
}

// ConversationItem is returned when listing stored conversation history.
type ConversationItem struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Status  string      `json:"status"`
}

// ResponseWithPlan combines a response with its execution plan details.
type ResponseWithPlan struct {
	Response    *Response   `json:"response"`
	PlanDetails interface{} `json:"plan,omitempty"` // *plan.Plan with tasks/steps
}

// StreamObserver receives streaming lifecycle events.
type StreamObserver interface {
	tool.StreamObserver
	OnResponseCreated(resp *Response)
}
