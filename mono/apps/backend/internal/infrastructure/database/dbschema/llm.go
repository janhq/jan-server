package dbschema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Provider represents an LLM provider (e.g., OpenAI, Anthropic).
type Provider struct {
	ID          string         `gorm:"type:varchar(36);primaryKey"`
	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	DisplayName string         `gorm:"type:varchar(100)"`
	BaseURL     string         `gorm:"type:varchar(500)"`
	APIKey      string         `gorm:"type:varchar(500)"` // Encrypted
	IsEnabled   *bool          `gorm:"not null;default:true"`
	Config      datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
}

func (Provider) TableName() string {
	return "llm_providers"
}

func (p *Provider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// Model represents an LLM model.
type Model struct {
	ID           string         `gorm:"type:varchar(36);primaryKey"`
	ProviderID   string         `gorm:"type:varchar(36);index;not null"`
	Name         string         `gorm:"type:varchar(100);not null"`
	DisplayName  string         `gorm:"type:varchar(100)"`
	Description  string         `gorm:"type:text"`
	ContextWindow int           `gorm:"default:4096"`
	MaxTokens    int            `gorm:"default:4096"`
	IsEnabled    *bool          `gorm:"not null;default:true"`
	Capabilities datatypes.JSON `gorm:"type:jsonb"` // vision, function_calling, etc.
	Pricing      datatypes.JSON `gorm:"type:jsonb"` // input/output pricing
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`

	Provider Provider `gorm:"foreignKey:ProviderID;constraint:OnDelete:CASCADE"`
}

func (Model) TableName() string {
	return "llm_models"
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// Conversation represents a chat conversation.
type Conversation struct {
	ID          string         `gorm:"type:varchar(36);primaryKey"`
	UserID      string         `gorm:"type:varchar(36);index;not null"`
	Title       string         `gorm:"type:varchar(255)"`
	ModelID     string         `gorm:"type:varchar(36);index"`
	SystemPrompt string        `gorm:"type:text"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	IsArchived  *bool          `gorm:"not null;default:false"`
	IsPinned    *bool          `gorm:"not null;default:false"`
	SharedToken *string        `gorm:"type:varchar(100);uniqueIndex"`
	SharedAt    *time.Time
	ParentID    *string        `gorm:"type:varchar(36);index"` // For branching
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`

	Messages []Message `gorm:"foreignKey:ConversationID"`
}

func (Conversation) TableName() string {
	return "llm_conversations"
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// Message represents a message in a conversation.
type Message struct {
	ID             string         `gorm:"type:varchar(36);primaryKey"`
	ConversationID string         `gorm:"type:varchar(36);index;not null"`
	Role           string         `gorm:"type:varchar(20);not null"` // user, assistant, system, tool
	Content        string         `gorm:"type:text"`
	Name           *string        `gorm:"type:varchar(100)"` // For tool messages
	ToolCallID     *string        `gorm:"type:varchar(100)"` // For tool responses
	ToolCalls      datatypes.JSON `gorm:"type:jsonb"`        // Tool calls made by assistant
	Attachments    datatypes.JSON `gorm:"type:jsonb"`        // File attachments
	Metadata       datatypes.JSON `gorm:"type:jsonb"`
	TokensInput    int            `gorm:"default:0"`
	TokensOutput   int            `gorm:"default:0"`
	ModelID        *string        `gorm:"type:varchar(36)"`
	ParentID       *string        `gorm:"type:varchar(36);index"` // For branching
	CreatedAt      time.Time      `gorm:"autoCreateTime"`

	Conversation Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
}

func (Message) TableName() string {
	return "llm_messages"
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// PromptTemplate represents a reusable prompt template.
type PromptTemplate struct {
	ID          string         `gorm:"type:varchar(36);primaryKey"`
	Name        string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string         `gorm:"type:text"`
	Content     string         `gorm:"type:text;not null"`
	Variables   datatypes.JSON `gorm:"type:jsonb"` // Variable definitions
	Category    string         `gorm:"type:varchar(50);index"`
	IsPublic    *bool          `gorm:"not null;default:false"`
	CreatedBy   string         `gorm:"type:varchar(36);index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
}

func (PromptTemplate) TableName() string {
	return "llm_prompt_templates"
}

func (p *PromptTemplate) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
