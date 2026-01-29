package memory

import (
	"context"
	"time"
)

// UserMemoryItem represents a user's personal memory item
type UserMemoryItem struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Scope     string    `json:"scope"` // "core", "preference", "context"
	Key       string    `json:"key"`
	Text      string    `json:"text"`
	Score     int       `json:"score"` // Importance: 1-5
	Embedding []float32 `json:"-"`
	IsDeleted bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Computed fields
	Similarity float32 `json:"similarity,omitempty" db:"-"`
}

// ProjectFact represents a project-level fact or decision
type ProjectFact struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	Kind                 string    `json:"kind"` // "decision", "requirement", "constraint", "context"
	Title                string    `json:"title"`
	Text                 string    `json:"text"`
	Confidence           float32   `json:"confidence"` // 0.0-1.0
	Embedding            []float32 `json:"-"`
	SourceConversationID string    `json:"source_conversation_id"`
	IsDeleted            bool      `json:"-"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// Computed fields
	Similarity float32 `json:"similarity,omitempty" db:"-"`
}

// EpisodicEvent represents a time-bound event or interaction
type EpisodicEvent struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	ConversationID string    `json:"conversation_id"`
	Time           time.Time `json:"time"`
	Text           string    `json:"text"`
	Kind           string    `json:"kind"` // "interaction", "decision", "milestone"
	Embedding      []float32 `json:"-"`
	IsDeleted      bool      `json:"-"`
	CreatedAt      time.Time `json:"created_at"`

	// Computed fields
	Similarity float32 `json:"similarity,omitempty" db:"-"`
}

// ConversationItem represents a single message in a conversation
type ConversationItem struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // "user", "assistant", "system"
	Content        string    `json:"content"`
	ToolCalls      string    `json:"tool_calls,omitempty"` // JSON array
	CreatedAt      time.Time `json:"created_at"`
}

// ConversationSummary represents a summary of a conversation
type ConversationSummary struct {
	ID              string        `json:"id" db:"id"`
	ConversationID  string        `json:"conversation_id" db:"conversation_id"`
	DialogueSummary string        `json:"dialogue_summary" db:"dialogue_summary"`
	OpenTasks       []interface{} `json:"open_tasks" db:"open_tasks"`
	Entities        []interface{} `json:"entities" db:"entities"`
	Decisions       []interface{} `json:"decisions" db:"decisions"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// MemoryLoadRequest represents a request to load relevant memories
type MemoryLoadRequest struct {
	UserID         string            `json:"user_id"`
	ProjectID      string            `json:"project_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	Query          string            `json:"query"`
	Options        MemoryLoadOptions `json:"options"`
}

// MemoryLoadOptions contains options for memory loading
type MemoryLoadOptions struct {
	AugmentWithMemory bool    `json:"augment_with_memory"`
	MaxUserItems      int     `json:"max_user_items"`
	MaxProjectItems   int     `json:"max_project_items"`
	MaxEpisodicItems  int     `json:"max_episodic_items"`
	MinSimilarity     float32 `json:"min_similarity"`
}

// MemoryLoadResponse contains the loaded memories
type MemoryLoadResponse struct {
	CoreMemory     []UserMemoryItem `json:"core_memory"`
	EpisodicMemory []EpisodicEvent  `json:"episodic_memory"`
	SemanticMemory []ProjectFact    `json:"semantic_memory"`
}

// MemoryObserveRequest represents a request to observe and store conversation
type MemoryObserveRequest struct {
	UserID         string             `json:"user_id"`
	ProjectID      string             `json:"project_id,omitempty"`
	ConversationID string             `json:"conversation_id"`
	Messages       []ConversationItem `json:"messages"`
	ToolCalls      []ToolCall         `json:"tool_calls,omitempty"`
}

// MemoryObserveResponse represents the response from observe endpoint
type MemoryObserveResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result,omitempty"`
}

// MemoryAction represents an action to take on memory
type MemoryAction struct {
	Add    MemoryAddActions `json:"add"`
	Delete []string         `json:"delete"` // Memory item IDs to delete
}

// MemoryAddActions contains items to add to different memory types
type MemoryAddActions struct {
	UserMemory    []UserMemoryItemInput `json:"user_memory"`
	ProjectMemory []ProjectFactInput    `json:"project_memory"`
	Episodic      []EpisodicEventInput  `json:"episodic"`
}

// UserMemoryItemInput represents input for creating a user memory item
type UserMemoryItemInput struct {
	Scope      string `json:"scope"`
	Key        string `json:"key"`
	Text       string `json:"text"`
	Importance string `json:"importance"` // "low", "medium", "high", "critical"
}

// ProjectFactInput represents input for creating a project fact
type ProjectFactInput struct {
	Kind       string  `json:"kind"`
	Title      string  `json:"title"`
	Text       string  `json:"text"`
	Confidence float32 `json:"confidence"`
}

// EpisodicEventInput represents input for creating an episodic event
type EpisodicEventInput struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
}

// UserMemoryUpsertRequest represents a request to upsert user memories
type UserMemoryUpsertRequest struct {
	UserID string                `json:"user_id"`
	Items  []UserMemoryItemInput `json:"items"`
}

// ProjectFactUpsertRequest represents a request to upsert project facts
type ProjectFactUpsertRequest struct {
	ProjectID string             `json:"project_id"`
	Facts     []ProjectFactInput `json:"facts"`
}

// DeleteRequest represents a request to delete memories
type DeleteRequest struct {
	IDs []string `json:"ids"`
}

// DeleteResponse represents the response from delete endpoint
type DeleteResponse struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	DeletedCount int    `json:"deleted_count"`
}

// LLMClient interface for calling LLM services
type LLMClient interface {
	Complete(ctx context.Context, prompt string, options LLMOptions) (string, error)
}

// LLMOptions for LLM completion
type LLMOptions struct {
	Model          string
	Temperature    float32
	MaxTokens      int
	ResponseFormat string // "json" or "text"
}
