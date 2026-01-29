package dbschema

import (
	"time"

	"jan-server/mono/apps/backend/internal/domain/memory"
)

type UserMemoryItem struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Scope     string    `db:"scope"`
	Key       string    `db:"key"`
	Text      string    `db:"text"`
	Score     int       `db:"score"`
	Embedding []float32 `db:"embedding"`
	IsDeleted bool      `db:"is_deleted"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewSchemaUserMemoryItem(d *memory.UserMemoryItem) *UserMemoryItem {
	if d == nil {
		return nil
	}

	return &UserMemoryItem{
		ID:        d.ID,
		UserID:    d.UserID,
		Scope:     d.Scope,
		Key:       d.Key,
		Text:      d.Text,
		Score:     d.Score,
		Embedding: d.Embedding,
		IsDeleted: d.IsDeleted,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func (s *UserMemoryItem) EtoD() *memory.UserMemoryItem {
	if s == nil {
		return nil
	}

	return &memory.UserMemoryItem{
		ID:        s.ID,
		UserID:    s.UserID,
		Scope:     s.Scope,
		Key:       s.Key,
		Text:      s.Text,
		Score:     s.Score,
		Embedding: s.Embedding,
		IsDeleted: s.IsDeleted,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
