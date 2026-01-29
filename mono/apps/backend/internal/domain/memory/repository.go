package memory

import (
	"context"
)

// Repository defines the interface for memory storage operations
type Repository interface {
	// User Memory
	GetUserMemoryItems(ctx context.Context, userID string) ([]UserMemoryItem, error)
	UpsertUserMemoryItem(ctx context.Context, item *UserMemoryItem) (string, error)
	DeleteUserMemoryItem(ctx context.Context, id string) error
	SearchUserMemory(ctx context.Context, userID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]UserMemoryItem, error)

	// Project Facts
	GetProjectFacts(ctx context.Context, projectID string) ([]ProjectFact, error)
	UpsertProjectFact(ctx context.Context, fact *ProjectFact) (string, error)
	DeleteProjectFact(ctx context.Context, id string) error
	SearchProjectFacts(ctx context.Context, projectID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]ProjectFact, error)

	// Episodic Events
	GetEpisodicEvents(ctx context.Context, userID string, limit int) ([]EpisodicEvent, error)
	CreateEpisodicEvent(ctx context.Context, event *EpisodicEvent) error
	DeleteEpisodicEvent(ctx context.Context, id string) error
	SearchEpisodicEvents(ctx context.Context, userID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]EpisodicEvent, error)

	// Conversation Items
	CreateConversationItem(ctx context.Context, item *ConversationItem) error
	GetConversationItems(ctx context.Context, conversationID string) ([]ConversationItem, error)
}
