package memoryrepo

import (
	"context"
	"fmt"
	"strings"

	"jan-server/mono/apps/backend/internal/domain/memory"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// helper converts embeddings to pgvector literal.
func embeddingToString(embedding []float32) string {
	if len(embedding) == 0 {
		return "[]"
	}

	parts := make([]string, len(embedding))
	for i, val := range embedding {
		parts[i] = fmt.Sprintf("%f", val)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// ensure interfaces are implemented
var _ interface {
	GetUserMemoryItems(ctx context.Context, userID string) ([]memory.UserMemoryItem, error)
	UpsertUserMemoryItem(ctx context.Context, item *memory.UserMemoryItem) (string, error)
	DeleteUserMemoryItem(ctx context.Context, id string) error
	SearchUserMemory(ctx context.Context, userID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]memory.UserMemoryItem, error)

	GetProjectFacts(ctx context.Context, projectID string) ([]memory.ProjectFact, error)
	UpsertProjectFact(ctx context.Context, fact *memory.ProjectFact) (string, error)
	DeleteProjectFact(ctx context.Context, id string) error
	SearchProjectFacts(ctx context.Context, projectID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]memory.ProjectFact, error)

	GetEpisodicEvents(ctx context.Context, userID string, limit int) ([]memory.EpisodicEvent, error)
	CreateEpisodicEvent(ctx context.Context, event *memory.EpisodicEvent) error
	DeleteEpisodicEvent(ctx context.Context, id string) error
	SearchEpisodicEvents(ctx context.Context, userID string, queryEmbedding []float32, limit int, minSimilarity float32) ([]memory.EpisodicEvent, error)

	CreateConversationItem(ctx context.Context, item *memory.ConversationItem) error
	GetConversationItems(ctx context.Context, conversationID string) ([]memory.ConversationItem, error)
} = (*Repository)(nil)
