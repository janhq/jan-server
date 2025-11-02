package dbschema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/services/llm-api/domain/conversation"
)

type Conversation struct {
	BaseModel
	PublicID     string                          `gorm:"type:varchar(50);uniqueIndex;not null"`
	Object       string                          `gorm:"type:varchar(50);not null;default:'conversation'"`
	Title        *string                         `gorm:"type:varchar(256)"`
	UserID       uint                            `gorm:"index:idx_conversation_user_referrer;index:idx_conversation_user_status;not null"`
	Status       conversation.ConversationStatus `gorm:"type:varchar(20);index:idx_conversation_user_status;not null;default:'active'"`
	ActiveBranch string                          `gorm:"type:varchar(50);not null;default:'MAIN'"`
	Referrer     *string                         `gorm:"type:varchar(100);index:idx_conversation_user_referrer"`
	Metadata     JSONMap                         `gorm:"type:jsonb"`
	IsPrivate    *bool                           `gorm:"default:false"`
	Items        []ConversationItem              `gorm:"foreignKey:ConversationID"`
	Branches     []ConversationBranch            `gorm:"foreignKey:ConversationID"`
}

func (Conversation) TableName() string {
	return "conversations"
}

type ConversationBranch struct {
	BaseModel
	ConversationID   uint         `gorm:"uniqueIndex:idx_conversation_branch_name;not null"`
	Conversation     Conversation `gorm:"foreignKey:ConversationID"`
	Name             string       `gorm:"type:varchar(50);uniqueIndex:idx_conversation_branch_name;not null"`
	Description      *string      `gorm:"type:text"`
	ParentBranch     *string      `gorm:"type:varchar(50)"`
	ForkedAt         *time.Time   `gorm:"type:timestamp"`
	ForkedFromItemID *string      `gorm:"type:varchar(50)"`
	ItemCount        int          `gorm:"default:0"`
}

func (ConversationBranch) TableName() string {
	return "conversation_branches"
}

type ConversationItem struct {
	BaseModel
	ConversationID    uint                  `gorm:"index:idx_item_conversation_branch;index:idx_item_conversation_sequence;not null"`
	Conversation      Conversation          `gorm:"foreignKey:ConversationID"`
	PublicID          string                `gorm:"type:varchar(50);uniqueIndex;not null"`
	Object            string                `gorm:"type:varchar(50);not null;default:'conversation.item'"`
	Branch            string                `gorm:"type:varchar(50);index:idx_item_conversation_branch;not null;default:'MAIN'"`
	SequenceNumber    int                   `gorm:"index:idx_item_conversation_sequence;not null"`
	Type              conversation.ItemType `gorm:"type:varchar(50);not null"`
	Role              *string               `gorm:"type:varchar(20)"`
	Content           JSONContent           `gorm:"type:jsonb"`
	Status            *string               `gorm:"type:varchar(20)"`
	IncompleteAt      *time.Time            `gorm:"type:timestamp"`
	IncompleteDetails JSONIncompleteDetails `gorm:"type:jsonb"`
	CompletedAt       *time.Time            `gorm:"type:timestamp"`
	ResponseID        *uint                 `gorm:"index"`
	Rating            *string               `gorm:"type:varchar(10)"`
	RatedAt           *time.Time            `gorm:"type:timestamp"`
	RatingComment     *string               `gorm:"type:text"`
}

func (ConversationItem) TableName() string {
	return "conversation_items"
}

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

// JSONContent is a custom type for []Content stored as JSON
type JSONContent []conversation.Content

func (j JSONContent) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONContent) Scan(value any) error {
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

// JSONIncompleteDetails is a custom type for IncompleteDetails stored as JSON
type JSONIncompleteDetails conversation.IncompleteDetails

func (j JSONIncompleteDetails) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONIncompleteDetails) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// NewSchemaConversation creates a database schema from domain conversation
func NewSchemaConversation(c *conversation.Conversation) *Conversation {
	isPrivate := c.IsPrivate
	return &Conversation{
		BaseModel: BaseModel{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		},
		PublicID:     c.PublicID,
		Object:       c.Object,
		Title:        c.Title,
		UserID:       c.UserID,
		Status:       c.Status,
		ActiveBranch: c.ActiveBranch,
		Referrer:     c.Referrer,
		Metadata:     JSONMap(c.Metadata),
		IsPrivate:    &isPrivate,
	}
}

// NewSchemaConversationBranch creates a database schema from domain branch metadata
func NewSchemaConversationBranch(conversationID uint, meta conversation.BranchMetadata) *ConversationBranch {
	return &ConversationBranch{
		BaseModel: BaseModel{
			CreatedAt: meta.CreatedAt,
			UpdatedAt: meta.UpdatedAt,
		},
		ConversationID:   conversationID,
		Name:             meta.Name,
		Description:      meta.Description,
		ParentBranch:     meta.ParentBranch,
		ForkedAt:         meta.ForkedAt,
		ForkedFromItemID: meta.ForkedFromItemID,
		ItemCount:        meta.ItemCount,
	}
}

// EtoD converts database branch to domain branch metadata
func (b *ConversationBranch) EtoD() conversation.BranchMetadata {
	return conversation.BranchMetadata{
		Name:             b.Name,
		Description:      b.Description,
		ParentBranch:     b.ParentBranch,
		ForkedAt:         b.ForkedAt,
		ForkedFromItemID: b.ForkedFromItemID,
		ItemCount:        b.ItemCount,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
	}
}

// EtoD converts database schema to domain conversation
func (c *Conversation) EtoD() *conversation.Conversation {
	isPrivate := false
	if c.IsPrivate != nil {
		isPrivate = *c.IsPrivate
	}
	conv := &conversation.Conversation{
		ID:             c.ID,
		PublicID:       c.PublicID,
		Object:         c.Object,
		Title:          c.Title,
		UserID:         c.UserID,
		Status:         c.Status,
		ActiveBranch:   c.ActiveBranch,
		Branches:       make(map[string][]conversation.Item),
		BranchMetadata: make(map[string]conversation.BranchMetadata),
		Metadata:       map[string]string(c.Metadata),
		IsPrivate:      isPrivate,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
	if c.Referrer != nil {
		conv.Referrer = c.Referrer
	}

	// Convert branch metadata
	if len(c.Branches) > 0 {
		for _, branch := range c.Branches {
			conv.BranchMetadata[branch.Name] = branch.EtoD()
		}
	}

	// Convert and organize items by branch
	if len(c.Items) > 0 {
		for _, item := range c.Items {
			domainItem := item.EtoD()
			branchName := domainItem.Branch
			if branchName == "" {
				branchName = "MAIN"
			}
			conv.Branches[branchName] = append(conv.Branches[branchName], *domainItem)
		}

		// Also populate legacy Items field with MAIN branch
		if mainItems, exists := conv.Branches["MAIN"]; exists {
			conv.Items = mainItems
		}
	}

	return conv
}

// NewSchemaConversationItem creates a database schema from domain item
func NewSchemaConversationItem(item *conversation.Item) *ConversationItem {
	branch := item.Branch
	if branch == "" {
		branch = "MAIN"
	}

	schemaItem := &ConversationItem{
		BaseModel: BaseModel{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
		},
		ConversationID: item.ConversationID,
		PublicID:       item.PublicID,
		Object:         item.Object,
		Branch:         branch,
		SequenceNumber: item.SequenceNumber,
		Type:           item.Type,
		Content:        JSONContent(item.Content),
		IncompleteAt:   item.IncompleteAt,
		CompletedAt:    item.CompletedAt,
		ResponseID:     item.ResponseID,
	}

	// Convert Role pointer to string pointer
	if item.Role != nil {
		roleStr := string(*item.Role)
		schemaItem.Role = &roleStr
	}

	// Convert Status pointer to string pointer
	if item.Status != nil {
		statusStr := string(*item.Status)
		schemaItem.Status = &statusStr
	}

	// Convert IncompleteDetails
	if item.IncompleteDetails != nil {
		details := JSONIncompleteDetails(*item.IncompleteDetails)
		schemaItem.IncompleteDetails = details
	}

	// Convert Rating
	if item.Rating != nil {
		ratingStr := string(*item.Rating)
		schemaItem.Rating = &ratingStr
	}
	schemaItem.RatedAt = item.RatedAt
	schemaItem.RatingComment = item.RatingComment

	return schemaItem
}

// EtoD converts database schema to domain item
func (i *ConversationItem) EtoD() *conversation.Item {
	item := &conversation.Item{
		ID:             i.ID,
		ConversationID: i.ConversationID,
		PublicID:       i.PublicID,
		Object:         i.Object,
		Branch:         i.Branch,
		SequenceNumber: i.SequenceNumber,
		Type:           i.Type,
		Content:        []conversation.Content(i.Content),
		IncompleteAt:   i.IncompleteAt,
		CompletedAt:    i.CompletedAt,
		ResponseID:     i.ResponseID,
		RatedAt:        i.RatedAt,
		RatingComment:  i.RatingComment,
		CreatedAt:      i.CreatedAt,
	}

	// Convert Role string pointer to ItemRole pointer
	if i.Role != nil {
		role := conversation.ItemRole(*i.Role)
		item.Role = &role
	}

	// Convert Status string pointer to ItemStatus pointer
	if i.Status != nil {
		status := conversation.ItemStatus(*i.Status)
		item.Status = &status
	}

	// Convert IncompleteDetails
	details := conversation.IncompleteDetails(i.IncompleteDetails)
	if details.Reason != nil {
		item.IncompleteDetails = &details
	}

	// Convert Rating
	if i.Rating != nil {
		rating := conversation.ItemRating(*i.Rating)
		item.Rating = &rating
	}

	return item
}
