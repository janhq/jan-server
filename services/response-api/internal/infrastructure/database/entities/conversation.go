package entities

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"jan-server/services/response-api/internal/domain/conversation"
)

// Conversation represents the database schema for conversations
type Conversation struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	PublicID        string                          `gorm:"type:varchar(50);uniqueIndex;not null"`
	Object          string                          `gorm:"type:varchar(50);not null;default:'conversation'"`
	Title           *string                         `gorm:"type:varchar(256)"`
	UserID          string                          `gorm:"type:varchar(64);index:idx_conversation_user_referrer;index:idx_conversation_user_status;not null"`
	ProjectID       *uint                           `gorm:"index:idx_conversations_project_updated_at"`
	ProjectPublicID *string                         `gorm:"type:varchar(64);index:idx_conversations_project_public_id"`
	Status          conversation.ConversationStatus `gorm:"type:varchar(20);index:idx_conversation_user_status;not null;default:'active'"`
	ActiveBranch    string                          `gorm:"type:varchar(50);not null;default:'MAIN'"`
	Referrer        *string                         `gorm:"type:varchar(100);index:idx_conversation_user_referrer"`
	Metadata        JSONMap                         `gorm:"type:jsonb"`
	IsPrivate       *bool                           `gorm:"default:false"`

	// Project instruction inheritance
	InstructionVersion           int     `gorm:"not null;default:1"`
	EffectiveInstructionSnapshot *string `gorm:"type:text"`

	Items    []ConversationItem   `gorm:"foreignKey:ConversationID"`
	Branches []ConversationBranch `gorm:"foreignKey:ConversationID"`
}

// TableName specifies the table name for Conversation.
func (Conversation) TableName() string {
	return "conversations"
}

// ConversationBranch represents metadata about a conversation branch
type ConversationBranch struct {
	ID               uint         `gorm:"primaryKey"`
	CreatedAt        time.Time    `gorm:"autoCreateTime"`
	UpdatedAt        time.Time    `gorm:"autoUpdateTime"`
	ConversationID   uint         `gorm:"uniqueIndex:idx_conversation_branch_name;not null"`
	Conversation     Conversation `gorm:"foreignKey:ConversationID"`
	Name             string       `gorm:"type:varchar(50);uniqueIndex:idx_conversation_branch_name;not null"`
	Description      *string      `gorm:"type:text"`
	ParentBranch     *string      `gorm:"type:varchar(50)"`
	ForkedAt         *time.Time   `gorm:"type:timestamp"`
	ForkedFromItemID *string      `gorm:"type:varchar(50)"`
	ItemCount        int          `gorm:"default:0"`
}

// TableName specifies the table name for ConversationBranch.
func (ConversationBranch) TableName() string {
	return "conversation_branches"
}

// ===============================================
// JSON Types for GORM
// ===============================================

// JSONMap is a custom type for map[string]string stored as JSON
type JSONMap map[string]string

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// ===============================================
// Conversion Functions
// ===============================================

// EtoD converts database entity to domain model
func (c *Conversation) EtoD() *conversation.Conversation {
	isPrivate := false
	if c.IsPrivate != nil {
		isPrivate = *c.IsPrivate
	}

	metadata := make(map[string]string)
	if c.Metadata != nil {
		metadata = c.Metadata
	}

	// Convert items
	items := make([]conversation.Item, len(c.Items))
	for i, item := range c.Items {
		items[i] = *item.EtoD()
	}

	// Convert branch metadata
	branchMetadata := make(map[string]conversation.BranchMetadata)
	for _, branch := range c.Branches {
		branchMetadata[branch.Name] = conversation.BranchMetadata{
			Name:             branch.Name,
			Description:      branch.Description,
			ParentBranch:     branch.ParentBranch,
			ForkedAt:         branch.ForkedAt,
			ForkedFromItemID: branch.ForkedFromItemID,
			ItemCount:        branch.ItemCount,
			CreatedAt:        branch.CreatedAt,
			UpdatedAt:        branch.UpdatedAt,
		}
	}

	return &conversation.Conversation{
		ID:                           c.ID,
		PublicID:                     c.PublicID,
		Object:                       c.Object,
		Title:                        c.Title,
		UserID:                       c.UserID,
		ProjectID:                    c.ProjectID,
		ProjectPublicID:              c.ProjectPublicID,
		Status:                       c.Status,
		ActiveBranch:                 c.ActiveBranch,
		Items:                        items,
		BranchMetadata:               branchMetadata,
		Metadata:                     metadata,
		Referrer:                     c.Referrer,
		IsPrivate:                    isPrivate,
		InstructionVersion:           c.InstructionVersion,
		EffectiveInstructionSnapshot: c.EffectiveInstructionSnapshot,
		CreatedAt:                    c.CreatedAt,
		UpdatedAt:                    c.UpdatedAt,
	}
}

// NewSchemaConversation creates a database entity from domain model
func NewSchemaConversation(c *conversation.Conversation) *Conversation {
	isPrivate := c.IsPrivate
	return &Conversation{
		ID:                           c.ID,
		PublicID:                     c.PublicID,
		Object:                       c.Object,
		Title:                        c.Title,
		UserID:                       c.UserID,
		ProjectID:                    c.ProjectID,
		ProjectPublicID:              c.ProjectPublicID,
		Status:                       c.Status,
		ActiveBranch:                 c.ActiveBranch,
		Referrer:                     c.Referrer,
		Metadata:                     c.Metadata,
		IsPrivate:                    &isPrivate,
		InstructionVersion:           c.InstructionVersion,
		EffectiveInstructionSnapshot: c.EffectiveInstructionSnapshot,
		CreatedAt:                    c.CreatedAt,
		UpdatedAt:                    c.UpdatedAt,
	}
}
