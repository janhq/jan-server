package dbschema

import (
	"time"

	"jan-server/mono/apps/backend/internal/domain/memory"
)

type ProjectFact struct {
	ID                   string    `db:"id"`
	ProjectID            string    `db:"project_id"`
	Kind                 string    `db:"kind"`
	Title                string    `db:"title"`
	Text                 string    `db:"text"`
	Confidence           float32   `db:"confidence"`
	Embedding            []float32 `db:"embedding"`
	SourceConversationID string    `db:"source_conversation_id"`
	IsDeleted            bool      `db:"is_deleted"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

func NewSchemaProjectFact(d *memory.ProjectFact) *ProjectFact {
	if d == nil {
		return nil
	}

	return &ProjectFact{
		ID:                   d.ID,
		ProjectID:            d.ProjectID,
		Kind:                 d.Kind,
		Title:                d.Title,
		Text:                 d.Text,
		Confidence:           d.Confidence,
		Embedding:            d.Embedding,
		SourceConversationID: d.SourceConversationID,
		IsDeleted:            d.IsDeleted,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
}

func (s *ProjectFact) EtoD() *memory.ProjectFact {
	if s == nil {
		return nil
	}

	return &memory.ProjectFact{
		ID:                   s.ID,
		ProjectID:            s.ProjectID,
		Kind:                 s.Kind,
		Title:                s.Title,
		Text:                 s.Text,
		Confidence:           s.Confidence,
		Embedding:            s.Embedding,
		SourceConversationID: s.SourceConversationID,
		IsDeleted:            s.IsDeleted,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
