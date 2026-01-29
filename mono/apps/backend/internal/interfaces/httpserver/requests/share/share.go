package sharerequests

import "jan-server/mono/apps/backend/internal/domain/share"

// CreateShareRequest represents the request to create a share
type CreateShareRequest struct {
	Scope                  string  `json:"scope" binding:"required,oneof=conversation item"` // "conversation" or "item"
	ItemID                 *string `json:"item_id,omitempty"`                                // Required if scope is "item"
	Title                  *string `json:"title,omitempty"`
	IncludeImages          bool    `json:"include_images,omitempty"`
	IncludeContextMessages bool    `json:"include_context_messages,omitempty"` // For single-message share
	Branch                 *string `json:"branch,omitempty"`                   // Branch to share from (defaults to active branch)
}

// ToShareScope converts the scope string to a ShareScope
func (r *CreateShareRequest) ToShareScope() share.ShareScope {
	if r.Scope == "item" {
		return share.ShareScopeItem
	}
	return share.ShareScopeConversation
}
