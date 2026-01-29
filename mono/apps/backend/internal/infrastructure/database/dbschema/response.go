package dbschema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Response represents a multi-step LLM response (Response API).
type Response struct {
	ID               string         `gorm:"type:varchar(36);primaryKey"`
	UserID           string         `gorm:"type:varchar(36);index;not null"`
	ConversationID   *string        `gorm:"type:varchar(36);index"`
	ModelID          string         `gorm:"type:varchar(36);index"`
	Status           string         `gorm:"type:varchar(20);index;not null"` // pending, in_progress, completed, failed, cancelled
	Input            datatypes.JSON `gorm:"type:jsonb"`                      // Input messages/items
	Output           datatypes.JSON `gorm:"type:jsonb"`                      // Output items
	Instructions     string         `gorm:"type:text"`
	Tools            datatypes.JSON `gorm:"type:jsonb"`
	ToolChoice       string         `gorm:"type:varchar(50)"`
	Temperature      *float64
	MaxTokens        *int
	Error            *string        `gorm:"type:text"`
	Usage            datatypes.JSON `gorm:"type:jsonb"` // Token usage
	Metadata         datatypes.JSON `gorm:"type:jsonb"`
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}

func (Response) TableName() string {
	return "resp_responses"
}

func (r *Response) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// ResponseItem represents an item in a response (message, tool_call, tool_result).
type ResponseItem struct {
	ID         string         `gorm:"type:varchar(36);primaryKey"`
	ResponseID string         `gorm:"type:varchar(36);index;not null"`
	Type       string         `gorm:"type:varchar(30);not null"` // message, tool_call, tool_result
	Role       string         `gorm:"type:varchar(20)"`          // user, assistant, system
	Content    datatypes.JSON `gorm:"type:jsonb"`
	Name       *string        `gorm:"type:varchar(100)"`
	CallID     *string        `gorm:"type:varchar(100)"`
	Arguments  *string        `gorm:"type:text"`
	Result     *string        `gorm:"type:text"`
	Status     string         `gorm:"type:varchar(20)"` // pending, in_progress, completed, failed
	Sequence   int            `gorm:"index"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`

	Response Response `gorm:"foreignKey:ResponseID;constraint:OnDelete:CASCADE"`
}

func (ResponseItem) TableName() string {
	return "resp_response_items"
}

func (r *ResponseItem) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// Artifact represents a code artifact (like Claude Artifacts).
type Artifact struct {
	ID             string         `gorm:"type:varchar(36);primaryKey"`
	UserID         string         `gorm:"type:varchar(36);index;not null"`
	ConversationID *string        `gorm:"type:varchar(36);index"`
	ResponseID     *string        `gorm:"type:varchar(36);index"`
	Title          string         `gorm:"type:varchar(255);not null"`
	Description    string         `gorm:"type:text"`
	Type           string         `gorm:"type:varchar(50);index;not null"` // code, html, svg, markdown, mermaid
	Language       string         `gorm:"type:varchar(50)"`                // For code artifacts
	Content        string         `gorm:"type:text;not null"`
	Version        int            `gorm:"not null;default:1"`
	Metadata       datatypes.JSON `gorm:"type:jsonb"`
	IsPublic       *bool          `gorm:"not null;default:false"`
	ShareToken     *string        `gorm:"type:varchar(100);uniqueIndex"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
}

func (Artifact) TableName() string {
	return "resp_artifacts"
}

func (a *Artifact) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// ArtifactVersion stores previous versions of an artifact.
type ArtifactVersion struct {
	ID         string    `gorm:"type:varchar(36);primaryKey"`
	ArtifactID string    `gorm:"type:varchar(36);index;not null"`
	Version    int       `gorm:"not null"`
	Content    string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`

	Artifact Artifact `gorm:"foreignKey:ArtifactID;constraint:OnDelete:CASCADE"`
}

func (ArtifactVersion) TableName() string {
	return "resp_artifact_versions"
}

func (v *ArtifactVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// Agent represents a configured AI agent.
type Agent struct {
	ID           string         `gorm:"type:varchar(36);primaryKey"`
	Name         string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	DisplayName  string         `gorm:"type:varchar(100)"`
	Description  string         `gorm:"type:text"`
	Instructions string         `gorm:"type:text"`
	ModelID      string         `gorm:"type:varchar(36)"`
	Tools        datatypes.JSON `gorm:"type:jsonb"`
	Capabilities datatypes.JSON `gorm:"type:jsonb"`
	Schema       datatypes.JSON `gorm:"type:jsonb"` // Input/output schema
	IsEnabled    *bool          `gorm:"not null;default:true"`
	IsPublic     *bool          `gorm:"not null;default:false"`
	CreatedBy    string         `gorm:"type:varchar(36);index"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
}

func (Agent) TableName() string {
	return "resp_agents"
}

func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
