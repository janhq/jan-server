package memoryrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"jan-server/mono/apps/backend/internal/domain/memory"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"gorm.io/gorm/clause"
)

func (r *Repository) GetEpisodicEvents(ctx context.Context, userID string, limit int) ([]memory.EpisodicEvent, error) {
	var rows []dbschema.EpisodicEvent
	if err := r.db.WithContext(ctx).
		Table("episodic_events").
		Select("id, user_id, project_id, conversation_id, time, text, kind, created_at").
		Where("user_id = ? AND is_deleted = false", userID).
		Order("time DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query episodic events: %w", err)
	}

	events := make([]memory.EpisodicEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, *row.EtoD())
	}

	return events, nil
}

func (r *Repository) CreateEpisodicEvent(ctx context.Context, event *memory.EpisodicEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	schema := dbschema.NewSchemaEpisodicEvent(event)

	if err := r.db.WithContext(ctx).
		Table("episodic_events").
		Create(map[string]any{
			"id":              schema.ID,
			"user_id":         schema.UserID,
			"project_id":      schema.ProjectID,
			"conversation_id": schema.ConversationID,
			"time":            schema.Time,
			"text":            schema.Text,
			"kind":            schema.Kind,
			"embedding":       embeddingToString(schema.Embedding),
			"is_deleted":      schema.IsDeleted,
			"created_at":      schema.CreatedAt,
		}).Error; err != nil {
		return fmt.Errorf("create episodic event: %w", err)
	}

	return nil
}

func (r *Repository) DeleteEpisodicEvent(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Table("episodic_events").
		Where("id = ?", id).
		Update("is_deleted", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("episodic event not found")
	}
	return nil
}

func (r *Repository) SearchEpisodicEvents(
	ctx context.Context,
	userID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.EpisodicEvent, error) {
	var rows []struct {
		dbschema.EpisodicEvent
		Similarity float32 `db:"similarity"`
	}

	if err := r.db.WithContext(ctx).
		Table("episodic_events").
		Select("id, user_id, project_id, conversation_id, time, text, kind, created_at, 1 - (embedding <=> ?::vector) AS similarity", embeddingToString(queryEmbedding)).
		Where("user_id = ? AND is_deleted = false AND time > NOW() - INTERVAL '2 weeks' AND 1 - (embedding <=> ?::vector) >= ?", userID, embeddingToString(queryEmbedding), minSimilarity).
		Order(clause.Expr{SQL: "embedding <=> ?::vector", Vars: []any{embeddingToString(queryEmbedding)}}).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search episodic events: %w", err)
	}

	events := make([]memory.EpisodicEvent, 0, len(rows))
	for _, row := range rows {
		event := row.EpisodicEvent.EtoD()
		event.Similarity = row.Similarity
		events = append(events, *event)
	}

	return events, nil
}
