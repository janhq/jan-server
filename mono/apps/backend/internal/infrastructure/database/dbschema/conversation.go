package dbschema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(Conversation{})
	registry.RegisterSchemaForAutoMigrate(ConversationItem{})
	registry.RegisterSchemaForAutoMigrate(ConversationBranch{})
}

// Conversation represents the database schema for conversations
type Conversation struct {
	BaseModel
	PublicID        string                          `gorm:"type:varchar(50);uniqueIndex;not null"`
	Object          string                          `gorm:"type:varchar(50);not null;default:'conversation'"`
	Title           *string                         `gorm:"type:varchar(256)"`
	UserID          string                          `gorm:"type:varchar(255);index:idx_conversation_user_referrer;index:idx_conversation_user_status;not null"` // JWT subject or user identifier
	ProjectID       *uint                           `gorm:"index:idx_conversations_project_updated_at"`                 // Optional project grouping
	ProjectPublicID *string                         `gorm:"type:varchar(64);index:idx_conversations_project_public_id"` // Public ID of the project
	Status          conversation.ConversationStatus `gorm:"type:varchar(20);index:idx_conversation_user_status;not null;default:'active'"`
	ActiveBranch    string                          `gorm:"type:varchar(50);not null;default:'MAIN'"` // Currently active branch
	Referrer        *string                         `gorm:"type:varchar(100);index:idx_conversation_user_referrer"`
	Metadata        JSONMap                         `gorm:"type:jsonb"`
	IsPrivate       *bool                           `gorm:"default:false"`

	// Project instruction inheritance
	InstructionVersion           int     `gorm:"not null;default:1"` // Version of project instruction when conversation was created
	EffectiveInstructionSnapshot *string `gorm:"type:text"`          // Snapshot of merged instruction for reproducibility

	Items    []ConversationItem   `gorm:"foreignKey:ConversationID"`
	Branches []ConversationBranch `gorm:"foreignKey:ConversationID"`
}

// ConversationBranch represents metadata about a conversation branch
type ConversationBranch struct {
	BaseModel
	ConversationID   uint         `gorm:"uniqueIndex:idx_conversation_branch_name;not null"`
	Conversation     Conversation `gorm:"foreignKey:ConversationID"`
	Name             string       `gorm:"type:varchar(50);uniqueIndex:idx_conversation_branch_name;not null"` // Branch identifier (MAIN, EDIT_1, etc.)
	Description      *string      `gorm:"type:text"`
	ParentBranch     *string      `gorm:"type:varchar(50)"` // Branch this was forked from
	ForkedAt         *time.Time   `gorm:"type:timestamp"`
	ForkedFromItemID *string      `gorm:"type:varchar(50)"` // Item ID where fork occurred
	ItemCount        int          `gorm:"default:0"`        // Cached count of items in this branch
}

// ConversationItem represents the database schema for conversation items
type ConversationItem struct {
	BaseModel
	ConversationID    uint                  `gorm:"index:idx_item_conversation_branch;index:idx_item_conversation_sequence;not null"`
	Conversation      Conversation          `gorm:"foreignKey:ConversationID"`
	PublicID          string                `gorm:"type:varchar(50);uniqueIndex;not null"`
	Object            string                `gorm:"type:varchar(50);not null;default:'conversation.item'"`
	Branch            string                `gorm:"type:varchar(50);index:idx_item_conversation_branch;not null;default:'MAIN'"` // Branch identifier
	SequenceNumber    int                   `gorm:"index:idx_item_conversation_sequence;not null"`                               // Order within branch
	Type              conversation.ItemType `gorm:"type:varchar(50);not null"`
	Role              *string               `gorm:"type:varchar(20)"` // Stored as string, converted to/from ItemRole
	Content           JSONContent           `gorm:"type:jsonb"`       // Stores []Content as JSON
	Status            *string               `gorm:"type:varchar(20)"` // Stored as string, converted to/from ItemStatus
	IncompleteAt      *time.Time            `gorm:"type:timestamp"`
	IncompleteDetails JSONIncompleteDetails `gorm:"type:jsonb"`
	CompletedAt       *time.Time            `gorm:"type:timestamp"`
	ResponseID        *uint                 `gorm:"index"`

	// User feedback/rating
	Rating        *string    `gorm:"type:varchar(10)"` // 'like' or 'unlike'
	RatedAt       *time.Time `gorm:"type:timestamp"`
	RatingComment *string    `gorm:"type:text"`

	// OpenAI-compatible fields (added in migration 000009)
	CallID                   *string      `gorm:"type:varchar(50);index:idx_conversation_items_call_id"`
	ServerLabel              *string      `gorm:"type:varchar(255);index:idx_conversation_items_server_label"`
	ApprovalRequestID        *string      `gorm:"type:varchar(50);index:idx_conversation_items_approval_request_id"`
	Arguments                *string      `gorm:"type:text"`
	Output                   *string      `gorm:"type:text"`
	Error                    *string      `gorm:"type:text"`
	Action                   JSONAction   `gorm:"type:jsonb"`
	Tools                    JSONMcpTools `gorm:"type:jsonb"`
	PendingSafetyChecks      JSONSafetyChecks `gorm:"type:jsonb"`
	AcknowledgedSafetyChecks JSONSafetyChecks `gorm:"type:jsonb"`
	Approve                  *bool        `gorm:"type:boolean"`
	Reason                   *string      `gorm:"type:text"`
	Commands                 JSONCommands `gorm:"type:jsonb"`
	MaxOutputLength          *int64       `gorm:"type:bigint"`
	ShellOutputs             JSONShellOutputs `gorm:"type:jsonb"`
	Operation                JSONOperation `gorm:"type:jsonb"`
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

// JSONAction is a custom type for action map stored as JSON
type JSONAction map[string]interface{}

func (j JSONAction) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONAction) Scan(value any) error {
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

// JSONMcpTools is a custom type for MCP tools array stored as JSON
type JSONMcpTools []conversation.McpTool

func (j JSONMcpTools) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONMcpTools) Scan(value any) error {
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

// JSONSafetyChecks is a custom type for safety checks array stored as JSON
type JSONSafetyChecks []conversation.SafetyCheck

func (j JSONSafetyChecks) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONSafetyChecks) Scan(value any) error {
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

// JSONCommands is a custom type for commands array stored as JSON
type JSONCommands []string

func (j JSONCommands) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONCommands) Scan(value any) error {
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

// JSONShellOutputs is a custom type for shell outputs array stored as JSON
type JSONShellOutputs []conversation.ShellOutput

func (j JSONShellOutputs) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONShellOutputs) Scan(value any) error {
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

// JSONOperation is a custom type for operation map stored as JSON
type JSONOperation map[string]interface{}

func (j JSONOperation) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONOperation) Scan(value any) error {
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

// NewSchemaConversation creates a database schema from domain conversation
func NewSchemaConversation(c *conversation.Conversation) *Conversation {
	isPrivate := c.IsPrivate
	return &Conversation{
		BaseModel: BaseModel{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		},
		PublicID:                     c.PublicID,
		Object:                       c.Object,
		Title:                        c.Title,
		UserID:                       c.UserID,
		ProjectID:                    c.ProjectID,
		ProjectPublicID:              c.ProjectPublicID,
		Status:                       c.Status,
		ActiveBranch:                 c.ActiveBranch,
		Referrer:                     c.Referrer,
		Metadata:                     JSONMap(c.Metadata),
		IsPrivate:                    &isPrivate,
		InstructionVersion:           c.InstructionVersion,
		EffectiveInstructionSnapshot: c.EffectiveInstructionSnapshot,
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

// EtoD converts database schema to domain conversation (Entity to Domain)
func (c *Conversation) EtoD() *conversation.Conversation {
	isPrivate := false
	if c.IsPrivate != nil {
		isPrivate = *c.IsPrivate
	}
	conv := &conversation.Conversation{
		ID:                           c.ID,
		PublicID:                     c.PublicID,
		Object:                       c.Object,
		Title:                        c.Title,
		UserID:                       c.UserID,
		ProjectID:                    c.ProjectID,
		ProjectPublicID:              c.ProjectPublicID,
		Status:                       c.Status,
		ActiveBranch:                 c.ActiveBranch,
		Branches:                     make(map[string][]conversation.Item),
		BranchMetadata:               make(map[string]conversation.BranchMetadata),
		Metadata:                     map[string]string(c.Metadata),
		IsPrivate:                    isPrivate,
		InstructionVersion:           c.InstructionVersion,
		EffectiveInstructionSnapshot: c.EffectiveInstructionSnapshot,
		CreatedAt:                    c.CreatedAt,
		UpdatedAt:                    c.UpdatedAt,
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
				branchName = "MAIN" // Default to MAIN if not set
			}
			conv.Branches[branchName] = append(conv.Branches[branchName], *domainItem)
		}

		// Also populate legacy Items field with MAIN branch for backward compatibility
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
		branch = "MAIN" // Default to MAIN if not set
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

	// Convert OpenAI-compatible fields
	schemaItem.CallID = item.CallID
	schemaItem.ServerLabel = item.ServerLabel
	schemaItem.ApprovalRequestID = item.ApprovalRequestID
	schemaItem.Arguments = item.Arguments
	schemaItem.Output = item.Output
	schemaItem.Error = item.Error
	schemaItem.Approve = item.Approve
	schemaItem.Reason = item.Reason
	schemaItem.MaxOutputLength = item.MaxOutputLength

	// Convert complex OpenAI fields
	if item.Action != nil {
		schemaItem.Action = JSONAction(item.Action)
	}
	if item.Tools != nil {
		schemaItem.Tools = JSONMcpTools(item.Tools)
	}
	if item.PendingSafetyChecks != nil {
		schemaItem.PendingSafetyChecks = JSONSafetyChecks(item.PendingSafetyChecks)
	}
	if item.AcknowledgedSafetyChecks != nil {
		schemaItem.AcknowledgedSafetyChecks = JSONSafetyChecks(item.AcknowledgedSafetyChecks)
	}
	if item.Commands != nil {
		schemaItem.Commands = JSONCommands(item.Commands)
	}
	if item.ShellOutputs != nil {
		schemaItem.ShellOutputs = JSONShellOutputs(item.ShellOutputs)
	}
	if item.Operation != nil {
		schemaItem.Operation = JSONOperation(item.Operation)
	}

	return schemaItem
}

// EtoD converts database schema to domain item (Entity to Domain)
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
	if i.IncompleteDetails != (JSONIncompleteDetails{}) {
		details := conversation.IncompleteDetails(i.IncompleteDetails)
		item.IncompleteDetails = &details
	}

	// Convert Rating
	if i.Rating != nil {
		rating := conversation.ItemRating(*i.Rating)
		item.Rating = &rating
	}
	item.RatedAt = i.RatedAt
	item.RatingComment = i.RatingComment

	// Convert OpenAI-compatible fields
	item.CallID = i.CallID
	item.ServerLabel = i.ServerLabel
	item.ApprovalRequestID = i.ApprovalRequestID
	item.Arguments = i.Arguments
	item.Output = i.Output
	item.Error = i.Error
	item.Approve = i.Approve
	item.Reason = i.Reason
	item.MaxOutputLength = i.MaxOutputLength

	// Convert complex OpenAI fields
	if i.Action != nil {
		item.Action = map[string]interface{}(i.Action)
	}
	if i.Tools != nil {
		item.Tools = []conversation.McpTool(i.Tools)
	}
	if i.PendingSafetyChecks != nil {
		item.PendingSafetyChecks = []conversation.SafetyCheck(i.PendingSafetyChecks)
	}
	if i.AcknowledgedSafetyChecks != nil {
		item.AcknowledgedSafetyChecks = []conversation.SafetyCheck(i.AcknowledgedSafetyChecks)
	}
	if i.Commands != nil {
		item.Commands = []string(i.Commands)
	}
	if i.ShellOutputs != nil {
		item.ShellOutputs = []conversation.ShellOutput(i.ShellOutputs)
	}
	if i.Operation != nil {
		item.Operation = map[string]interface{}(i.Operation)
	}

	return item
}
