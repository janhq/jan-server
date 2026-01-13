package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/services/response-api/internal/domain/conversation"
)

// ConversationItem represents the database schema for conversation items
type ConversationItem struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	ConversationID    uint                  `gorm:"index:idx_item_conversation_branch;index:idx_item_conversation_sequence;not null"`
	PublicID          string                `gorm:"type:varchar(50);index:idx_conversation_items_public_id"`
	Object            string                `gorm:"type:varchar(50);not null;default:'conversation.item'"`
	Branch            string                `gorm:"type:varchar(50);index:idx_item_conversation_branch;not null;default:'MAIN'"`
	SequenceNumber    int                   `gorm:"column:sequence;index:idx_item_conversation_sequence;not null"`
	Type              conversation.ItemType `gorm:"type:varchar(50);not null"`
	Role              *string               `gorm:"type:varchar(20)"`
	Content           JSONContent           `gorm:"type:jsonb"`
	Status            *string               `gorm:"type:varchar(20)"`
	IncompleteAt      *time.Time            `gorm:"type:timestamp"`
	IncompleteDetails JSONIncompleteDetails `gorm:"type:jsonb"`
	CompletedAt       *time.Time            `gorm:"type:timestamp"`
	ResponseID        *uint                 `gorm:"index"`

	// User feedback/rating
	Rating        *string    `gorm:"type:varchar(10)"`
	RatedAt       *time.Time `gorm:"type:timestamp"`
	RatingComment *string    `gorm:"type:text"`

	// OpenAI-compatible fields
	CallID                   *string          `gorm:"type:varchar(50);index:idx_conversation_items_call_id"`
	Name                     *string          `gorm:"type:varchar(255)"`
	ServerLabel              *string          `gorm:"type:varchar(255);index:idx_conversation_items_server_label"`
	ApprovalRequestID        *string          `gorm:"type:varchar(50);index:idx_conversation_items_approval_request_id"`
	Arguments                *string          `gorm:"type:text"`
	Output                   *string          `gorm:"type:text"`
	Error                    *string          `gorm:"type:text"`
	Action                   JSONAction       `gorm:"type:jsonb"`
	Tools                    JSONMcpTools     `gorm:"type:jsonb"`
	PendingSafetyChecks      JSONSafetyChecks `gorm:"type:jsonb"`
	AcknowledgedSafetyChecks JSONSafetyChecks `gorm:"type:jsonb"`
	Approve                  *bool            `gorm:"type:boolean"`
	Reason                   *string          `gorm:"type:text"`
	Commands                 JSONCommands     `gorm:"type:jsonb"`
	MaxOutputLength          *int64           `gorm:"type:bigint"`
	ShellOutputs             JSONShellOutputs `gorm:"type:jsonb"`
	Operation                JSONOperation    `gorm:"type:jsonb"`
}

// TableName specifies the table name for ConversationItem.
func (ConversationItem) TableName() string {
	return "conversation_items"
}

// ===============================================
// JSON Types for GORM
// ===============================================

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

// ===============================================
// Conversion Functions
// ===============================================

// EtoD converts database entity to domain model
func (ci *ConversationItem) EtoD() *conversation.Item {
	var role *conversation.ItemRole
	var roleValue conversation.ItemRole
	if ci.Role != nil {
		r := conversation.ItemRole(*ci.Role)
		role = &r
		roleValue = r
	}

	var status *conversation.ItemStatus
	var statusValue conversation.ItemStatus
	if ci.Status != nil {
		s := conversation.ItemStatus(*ci.Status)
		status = &s
		statusValue = s
	}

	var rating *conversation.ItemRating
	if ci.Rating != nil {
		r := conversation.ItemRating(*ci.Rating)
		rating = &r
	}

	var incompleteDetails *conversation.IncompleteDetails
	if ci.IncompleteDetails.Reason != "" {
		incompleteDetails = &conversation.IncompleteDetails{
			Reason: ci.IncompleteDetails.Reason,
			Error:  ci.IncompleteDetails.Error,
		}
	}

	// Convert content
	content := make([]conversation.Content, len(ci.Content))
	for i, c := range ci.Content {
		content[i] = c
	}

	// Convert tools
	tools := make([]conversation.McpTool, len(ci.Tools))
	for i, t := range ci.Tools {
		tools[i] = t
	}

	// Convert safety checks
	pendingSafetyChecks := make([]conversation.SafetyCheck, len(ci.PendingSafetyChecks))
	for i, s := range ci.PendingSafetyChecks {
		pendingSafetyChecks[i] = s
	}

	acknowledgedSafetyChecks := make([]conversation.SafetyCheck, len(ci.AcknowledgedSafetyChecks))
	for i, s := range ci.AcknowledgedSafetyChecks {
		acknowledgedSafetyChecks[i] = s
	}

	// Convert shell outputs
	shellOutputs := make([]conversation.ShellOutput, len(ci.ShellOutputs))
	for i, s := range ci.ShellOutputs {
		shellOutputs[i] = s
	}

	return &conversation.Item{
		ID:                       ci.ID,
		ConversationID:           ci.ConversationID,
		PublicID:                 ci.PublicID,
		Object:                   ci.Object,
		Branch:                   ci.Branch,
		SequenceNumber:           ci.SequenceNumber,
		Sequence:                 ci.SequenceNumber, // Legacy compatibility
		Type:                     ci.Type,
		Role:                     roleValue, // Direct value for legacy compat
		RolePtr:                  role,      // Pointer for new code
		Content:                  content,
		Status:                   statusValue, // Direct value for legacy compat
		StatusPtr:                status,      // Pointer for new code
		IncompleteAt:             ci.IncompleteAt,
		IncompleteDetails:        incompleteDetails,
		CompletedAt:              ci.CompletedAt,
		ResponseID:               ci.ResponseID,
		Rating:                   rating,
		RatedAt:                  ci.RatedAt,
		RatingComment:            ci.RatingComment,
		CallID:                   ci.CallID,
		Name:                     ci.Name,
		ServerLabel:              ci.ServerLabel,
		ApprovalRequestID:        ci.ApprovalRequestID,
		Arguments:                ci.Arguments,
		Output:                   ci.Output,
		Error:                    ci.Error,
		Action:                   ci.Action,
		Tools:                    tools,
		PendingSafetyChecks:      pendingSafetyChecks,
		AcknowledgedSafetyChecks: acknowledgedSafetyChecks,
		Approve:                  ci.Approve,
		Reason:                   ci.Reason,
		Commands:                 ci.Commands,
		MaxOutputLength:          ci.MaxOutputLength,
		ShellOutputs:             shellOutputs,
		Operation:                ci.Operation,
		CreatedAt:                ci.CreatedAt,
	}
}

// NewSchemaConversationItem creates a database entity from domain model
func NewSchemaConversationItem(item *conversation.Item) *ConversationItem {
	// Handle role - check both RolePtr and Role (legacy)
	var role *string
	if item.RolePtr != nil {
		r := string(*item.RolePtr)
		role = &r
	} else if item.Role != "" {
		r := string(item.Role)
		role = &r
	}

	// Handle status - check both StatusPtr and Status (legacy)
	var status *string
	if item.StatusPtr != nil {
		s := string(*item.StatusPtr)
		status = &s
	} else if item.Status != "" {
		s := string(item.Status)
		status = &s
	}

	var rating *string
	if item.Rating != nil {
		r := string(*item.Rating)
		rating = &r
	}

	var incompleteDetails JSONIncompleteDetails
	if item.IncompleteDetails != nil {
		incompleteDetails = JSONIncompleteDetails{
			Reason: item.IncompleteDetails.Reason,
			Error:  item.IncompleteDetails.Error,
		}
	}

	// Convert content
	content := make(JSONContent, len(item.Content))
	for i, c := range item.Content {
		content[i] = c
	}

	// Convert tools
	tools := make(JSONMcpTools, len(item.Tools))
	for i, t := range item.Tools {
		tools[i] = t
	}

	// Convert safety checks
	pendingSafetyChecks := make(JSONSafetyChecks, len(item.PendingSafetyChecks))
	for i, s := range item.PendingSafetyChecks {
		pendingSafetyChecks[i] = s
	}

	acknowledgedSafetyChecks := make(JSONSafetyChecks, len(item.AcknowledgedSafetyChecks))
	for i, s := range item.AcknowledgedSafetyChecks {
		acknowledgedSafetyChecks[i] = s
	}

	// Convert shell outputs
	shellOutputs := make(JSONShellOutputs, len(item.ShellOutputs))
	for i, s := range item.ShellOutputs {
		shellOutputs[i] = s
	}

	return &ConversationItem{
		ID:                       item.ID,
		ConversationID:           item.ConversationID,
		PublicID:                 item.PublicID,
		Object:                   item.Object,
		Branch:                   item.Branch,
		SequenceNumber:           item.GetSequence(), // Use helper to handle both fields
		Type:                     item.Type,
		Role:                     role,
		Content:                  content,
		Status:                   status,
		IncompleteAt:             item.IncompleteAt,
		IncompleteDetails:        incompleteDetails,
		CompletedAt:              item.CompletedAt,
		ResponseID:               item.ResponseID,
		Rating:                   rating,
		RatedAt:                  item.RatedAt,
		RatingComment:            item.RatingComment,
		CallID:                   item.CallID,
		Name:                     item.Name,
		ServerLabel:              item.ServerLabel,
		ApprovalRequestID:        item.ApprovalRequestID,
		Arguments:                item.Arguments,
		Output:                   item.Output,
		Error:                    item.Error,
		Action:                   item.Action,
		Tools:                    tools,
		PendingSafetyChecks:      pendingSafetyChecks,
		AcknowledgedSafetyChecks: acknowledgedSafetyChecks,
		Approve:                  item.Approve,
		Reason:                   item.Reason,
		Commands:                 item.Commands,
		MaxOutputLength:          item.MaxOutputLength,
		ShellOutputs:             shellOutputs,
		Operation:                item.Operation,
		CreatedAt:                item.CreatedAt,
	}
}
