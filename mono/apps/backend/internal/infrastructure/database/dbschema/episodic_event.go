package dbschema

import (
	"time"

	"jan-server/mono/apps/backend/internal/domain/memory"
)

type EpisodicEvent struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	ProjectID      string    `db:"project_id"`
	ConversationID string    `db:"conversation_id"`
	Time           time.Time `db:"time"`
	Text           string    `db:"text"`
	Kind           string    `db:"kind"`
	Embedding      []float32 `db:"embedding"`
	IsDeleted      bool      `db:"is_deleted"`
	CreatedAt      time.Time `db:"created_at"`
}

func NewSchemaEpisodicEvent(d *memory.EpisodicEvent) *EpisodicEvent {
	if d == nil {
		return nil
	}

	return &EpisodicEvent{
		ID:             d.ID,
		UserID:         d.UserID,
		ProjectID:      d.ProjectID,
		ConversationID: d.ConversationID,
		Time:           d.Time,
		Text:           d.Text,
		Kind:           d.Kind,
		Embedding:      d.Embedding,
		IsDeleted:      d.IsDeleted,
		CreatedAt:      d.CreatedAt,
	}
}

func (s *EpisodicEvent) EtoD() *memory.EpisodicEvent {
	if s == nil {
		return nil
	}

	return &memory.EpisodicEvent{
		ID:             s.ID,
		UserID:         s.UserID,
		ProjectID:      s.ProjectID,
		ConversationID: s.ConversationID,
		Time:           s.Time,
		Text:           s.Text,
		Kind:           s.Kind,
		Embedding:      s.Embedding,
		IsDeleted:      s.IsDeleted,
		CreatedAt:      s.CreatedAt,
	}
}
