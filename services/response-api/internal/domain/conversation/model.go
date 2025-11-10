package conversation

import "time"

// Conversation represents a logical chat thread for the Responses API.
type Conversation struct {
	ID        uint                   `json:"-"`
	PublicID  string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ItemRole indicates who authored the conversation item.
type ItemRole string

// ItemStatus tracks whether the item is finalised.
type ItemStatus string

const (
	RoleSystem    ItemRole = "system"
	RoleUser      ItemRole = "user"
	RoleAssistant ItemRole = "assistant"
	RoleTool      ItemRole = "tool"

	ItemStatusCompleted ItemStatus = "completed"
	ItemStatusPending   ItemStatus = "pending"
)

// Item contains individual conversation message state.
type Item struct {
	ID             uint                   `json:"-"`
	ConversationID uint                   `json:"conversation_id"`
	Role           ItemRole               `json:"role"`
	Status         ItemStatus             `json:"status"`
	Content        map[string]interface{} `json:"content"`
	Sequence       int                    `json:"sequence"`
	CreatedAt      time.Time              `json:"created_at"`
}
