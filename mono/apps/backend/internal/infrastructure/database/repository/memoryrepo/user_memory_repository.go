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

func (r *Repository) GetUserMemoryItems(ctx context.Context, userID string) ([]memory.UserMemoryItem, error) {
	query := `
		id, user_id, scope, key, text, score, created_at, updated_at
	`

	var rows []dbschema.UserMemoryItem
	if err := r.db.WithContext(ctx).
		Table("user_memory_items").
		Select(query).
		Where("user_id = ? AND is_deleted = false", userID).
		Order("score DESC, updated_at DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query user memory: %w", err)
	}

	items := make([]memory.UserMemoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, *row.EtoD())
	}

	return items, nil
}

func (r *Repository) UpsertUserMemoryItem(ctx context.Context, item *memory.UserMemoryItem) (string, error) {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	schema := dbschema.NewSchemaUserMemoryItem(item)

	if err := r.db.WithContext(ctx).
		Table("user_memory_items").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"scope", "key", "text", "score", "embedding", "is_deleted", "updated_at"}),
		}).
		Create(map[string]any{
			"id":         schema.ID,
			"user_id":    schema.UserID,
			"scope":      schema.Scope,
			"key":        schema.Key,
			"text":       schema.Text,
			"score":      schema.Score,
			"embedding":  embeddingToString(schema.Embedding),
			"is_deleted": schema.IsDeleted,
			"created_at": schema.CreatedAt,
			"updated_at": schema.UpdatedAt,
		}).Error; err != nil {
		return "", fmt.Errorf("upsert user memory item: %w", err)
	}

	return schema.ID, nil
}

func (r *Repository) DeleteUserMemoryItem(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Table("user_memory_items").
		Where("id = ?", id).
		Updates(map[string]any{
			"is_deleted": true,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user memory item not found")
	}
	return nil
}

func (r *Repository) SearchUserMemory(
	ctx context.Context,
	userID string,
	queryEmbedding []float32,
	limit int,
	minSimilarity float32,
) ([]memory.UserMemoryItem, error) {
	var rows []struct {
		dbschema.UserMemoryItem
		Similarity float32 `db:"similarity"`
	}

	if err := r.db.WithContext(ctx).
		Table("user_memory_items").
		Select("id, user_id, scope, key, text, score, created_at, updated_at, 1 - (embedding <=> ?::vector) AS similarity", embeddingToString(queryEmbedding)).
		Where("user_id = ? AND is_deleted = false AND score >= 2 AND 1 - (embedding <=> ?::vector) >= ?", userID, embeddingToString(queryEmbedding), minSimilarity).
		Order(clause.Expr{SQL: "embedding <=> ?::vector", Vars: []any{embeddingToString(queryEmbedding)}}).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search user memory: %w", err)
	}

	items := make([]memory.UserMemoryItem, 0, len(rows))
	for _, row := range rows {
		item := row.UserMemoryItem.EtoD()
		item.Similarity = row.Similarity
		items = append(items, *item)
	}

	return items, nil
}
