package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"jan-server/mono/apps/backend/internal/domain/memory"
)

// VectorSearcher handles vector similarity search operations
type VectorSearcher struct {
	db *pgxpool.Pool
}

// NewVectorSearcher creates a new vector searcher
func NewVectorSearcher(db *pgxpool.Pool) *VectorSearcher {
	return &VectorSearcher{db: db}
}

// SearchUserMemory performs vector similarity search on user memory
func (s *VectorSearcher) SearchUserMemory(
	ctx context.Context,
	userID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.UserMemoryItem, error) {
	query := `
		SELECT 
			id, user_id, scope, key, text, score, created_at, updated_at,
			1 - (embedding <=> $1::vector) AS similarity
		FROM user_memory_items
		WHERE user_id = $2 
		  AND is_deleted = false
		  AND score >= 2
		  AND 1 - (embedding <=> $1::vector) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query,
		embeddingToString(queryEmbedding),
		userID,
		minSimilarity,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search user memory: %w", err)
	}
	defer rows.Close()

	var items []memory.UserMemoryItem
	for rows.Next() {
		var item memory.UserMemoryItem
		err := rows.Scan(
			&item.ID, &item.UserID, &item.Scope, &item.Key,
			&item.Text, &item.Score, &item.CreatedAt, &item.UpdatedAt,
			&item.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// SearchProjectFacts performs vector similarity search on project facts
func (s *VectorSearcher) SearchProjectFacts(
	ctx context.Context,
	projectID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.ProjectFact, error) {
	query := `
		SELECT 
			id, project_id, kind, title, text, confidence,
			source_conversation_id, created_at, updated_at,
			1 - (embedding <=> $1::vector) AS similarity
		FROM project_facts
		WHERE project_id = $2 
		  AND is_deleted = false
		  AND confidence >= 0.7
		  AND 1 - (embedding <=> $1::vector) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query,
		embeddingToString(queryEmbedding),
		projectID,
		minSimilarity,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search project facts: %w", err)
	}
	defer rows.Close()

	var facts []memory.ProjectFact
	for rows.Next() {
		var fact memory.ProjectFact
		err := rows.Scan(
			&fact.ID, &fact.ProjectID, &fact.Kind, &fact.Title,
			&fact.Text, &fact.Confidence, &fact.SourceConversationID,
			&fact.CreatedAt, &fact.UpdatedAt, &fact.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// SearchEpisodicEvents performs vector similarity search on episodic events
func (s *VectorSearcher) SearchEpisodicEvents(
	ctx context.Context,
	userID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.EpisodicEvent, error) {
	query := `
		SELECT 
			id, user_id, project_id, conversation_id, time, text, kind, created_at,
			1 - (embedding <=> $1::vector) AS similarity
		FROM episodic_events
		WHERE user_id = $2 
		  AND is_deleted = false
		  AND time > NOW() - INTERVAL '2 weeks'
		  AND 1 - (embedding <=> $1::vector) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query,
		embeddingToString(queryEmbedding),
		userID,
		minSimilarity,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search episodic events: %w", err)
	}
	defer rows.Close()

	var events []memory.EpisodicEvent
	for rows.Next() {
		var event memory.EpisodicEvent
		err := rows.Scan(
			&event.ID, &event.UserID, &event.ProjectID, &event.ConversationID,
			&event.Time, &event.Text, &event.Kind, &event.CreatedAt,
			&event.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// Helper function to convert embedding to pgvector string format
func embeddingToString(embedding []float32) string {
	if len(embedding) == 0 {
		return "[]"
	}

	result := "["
	for i, val := range embedding {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", val)
	}
	result += "]"
	return result
}
