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

func (r *Repository) GetProjectFacts(ctx context.Context, projectID string) ([]memory.ProjectFact, error) {
	query := `
		id, project_id, kind, title, text, confidence, 
		source_conversation_id, created_at, updated_at
	`

	var rows []dbschema.ProjectFact
	if err := r.db.WithContext(ctx).
		Table("project_facts").
		Select(query).
		Where("project_id = ? AND is_deleted = false", projectID).
		Order("confidence DESC, updated_at DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query project facts: %w", err)
	}

	facts := make([]memory.ProjectFact, 0, len(rows))
	for _, row := range rows {
		facts = append(facts, *row.EtoD())
	}

	return facts, nil
}

func (r *Repository) UpsertProjectFact(ctx context.Context, fact *memory.ProjectFact) (string, error) {
	if fact.ID == "" {
		fact.ID = uuid.New().String()
	}

	now := time.Now()
	if fact.CreatedAt.IsZero() {
		fact.CreatedAt = now
	}
	fact.UpdatedAt = now

	schema := dbschema.NewSchemaProjectFact(fact)

	if err := r.db.WithContext(ctx).
		Table("project_facts").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"kind", "title", "text", "confidence", "embedding", "is_deleted", "updated_at"}),
		}).
		Create(map[string]any{
			"id":                     schema.ID,
			"project_id":             schema.ProjectID,
			"kind":                   schema.Kind,
			"title":                  schema.Title,
			"text":                   schema.Text,
			"confidence":             schema.Confidence,
			"embedding":              embeddingToString(schema.Embedding),
			"source_conversation_id": schema.SourceConversationID,
			"is_deleted":             schema.IsDeleted,
			"created_at":             schema.CreatedAt,
			"updated_at":             schema.UpdatedAt,
		}).Error; err != nil {
		return "", fmt.Errorf("upsert project fact: %w", err)
	}

	return schema.ID, nil
}

func (r *Repository) DeleteProjectFact(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Table("project_facts").
		Where("id = ?", id).
		Updates(map[string]any{
			"is_deleted": true,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("project fact not found")
	}
	return nil
}

func (r *Repository) SearchProjectFacts(
	ctx context.Context,
	projectID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.ProjectFact, error) {
	var rows []struct {
		dbschema.ProjectFact
		Similarity float32 `db:"similarity"`
	}

	if err := r.db.WithContext(ctx).
		Table("project_facts").
		Select("id, project_id, kind, title, text, confidence, source_conversation_id, created_at, updated_at, 1 - (embedding <=> ?::vector) AS similarity", embeddingToString(queryEmbedding)).
		Where("project_id = ? AND is_deleted = false AND confidence >= 0.7 AND 1 - (embedding <=> ?::vector) >= ?", projectID, embeddingToString(queryEmbedding), minSimilarity).
		Order(clause.Expr{SQL: "embedding <=> ?::vector", Vars: []any{embeddingToString(queryEmbedding)}}).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search project facts: %w", err)
	}

	facts := make([]memory.ProjectFact, 0, len(rows))
	for _, row := range rows {
		fact := row.ProjectFact.EtoD()
		fact.Similarity = row.Similarity
		facts = append(facts, *fact)
	}

	return facts, nil
}
