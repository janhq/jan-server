package dbschema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"jan-server/mono/apps/backend/internal/domain/share"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(ConversationShare{})
}

// ConversationShare represents the database schema for conversation shares
type ConversationShare struct {
	BaseModel
	PublicID        string           `gorm:"type:varchar(64);uniqueIndex;not null"`
	Slug            string           `gorm:"type:varchar(30);uniqueIndex;not null"`
	ConversationID  uint             `gorm:"index:idx_conversation_shares_conversation_id;not null"`
	Conversation    Conversation     `gorm:"foreignKey:ConversationID"`
	OwnerUserID     string           `gorm:"type:varchar(255);index:idx_conversation_shares_owner_user_id;not null"`
	ItemPublicID    *string          `gorm:"type:varchar(64)"`
	Title           *string          `gorm:"type:varchar(256)"`
	Visibility      string           `gorm:"type:varchar(20);not null;default:'unlisted'"`
	RevokedAt       *time.Time       `gorm:"type:timestamp"`
	ViewCount       int              `gorm:"not null;default:0"`
	LastViewedAt    *time.Time       `gorm:"type:timestamp"`
	SnapshotVersion int              `gorm:"not null;default:1"`
	Snapshot        JSONSnapshot     `gorm:"type:jsonb;not null"`
	ShareOptions    JSONShareOptions `gorm:"type:jsonb"`
}

// TableName returns the custom table name for conversation shares
func (ConversationShare) TableName() string {
	return "llm_api.conversation_shares"
}

// JSONSnapshot is a custom type for share.Snapshot stored as JSON
type JSONSnapshot share.Snapshot

func (j JSONSnapshot) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONSnapshot) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// JSONShareOptions is a custom type for share.Options stored as JSON
type JSONShareOptions share.Options

func (j JSONShareOptions) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONShareOptions) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// NewSchemaConversationShare creates a database schema from domain share
func NewSchemaConversationShare(s *share.Share) *ConversationShare {
	schema := &ConversationShare{
		BaseModel: BaseModel{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		PublicID:        s.PublicID,
		Slug:            s.Slug,
		ConversationID:  s.ConversationID,
		OwnerUserID:     s.OwnerUserID,
		ItemPublicID:    s.ItemPublicID,
		Title:           s.Title,
		Visibility:      string(s.Visibility),
		RevokedAt:       s.RevokedAt,
		ViewCount:       s.ViewCount,
		LastViewedAt:    s.LastViewedAt,
		SnapshotVersion: s.SnapshotVersion,
	}

	if s.Snapshot != nil {
		schema.Snapshot = JSONSnapshot(*s.Snapshot)
	}

	if s.ShareOptions != nil {
		schema.ShareOptions = JSONShareOptions(*s.ShareOptions)
	}

	return schema
}

// EtoD converts database schema to domain share (Entity to Domain)
func (s *ConversationShare) EtoD() *share.Share {
	domainShare := &share.Share{
		ID:              s.ID,
		PublicID:        s.PublicID,
		Slug:            s.Slug,
		ConversationID:  s.ConversationID,
		OwnerUserID:     s.OwnerUserID,
		ItemPublicID:    s.ItemPublicID,
		Title:           s.Title,
		Visibility:      share.Visibility(s.Visibility),
		RevokedAt:       s.RevokedAt,
		ViewCount:       s.ViewCount,
		LastViewedAt:    s.LastViewedAt,
		SnapshotVersion: s.SnapshotVersion,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}

	// Convert snapshot
	if s.Snapshot.Title != "" || len(s.Snapshot.Items) > 0 {
		snapshot := share.Snapshot(s.Snapshot)
		domainShare.Snapshot = &snapshot
	}

	// Convert options
	opts := share.Options(s.ShareOptions)
	if opts.IncludeImages || opts.IncludeContextMessages {
		domainShare.ShareOptions = &opts
	}

	return domainShare
}
