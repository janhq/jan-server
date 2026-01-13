package share

import (
	"context"
	"encoding/json"
	"fmt"

	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

const (
	// MaxSnapshotSize is the maximum allowed size for a serialized snapshot (10MB)
	MaxSnapshotSize = 10 * 1024 * 1024

	// DefaultTitleMaxLength is the max length for auto-generated titles
	DefaultTitleMaxLength = 50

	// DefaultUntitledTitle is the fallback title
	DefaultUntitledTitle = "Untitled Conversation"

	// MinItemsForShare is the minimum number of items required for sharing
	MinItemsForShare = 2

	// TemporaryChatID cannot be shared
	TemporaryChatID = "TEMPORARY_CHAT_ID"
)

// ShareService handles business logic for conversation sharing
type ShareService struct {
	repo          ShareRepository
	convRepo      conversation.ConversationRepository
	itemRepo      conversation.ItemRepository
	slugGenerator *SlugGenerator
}

// NewShareService creates a new share service
func NewShareService(repo ShareRepository, convRepo conversation.ConversationRepository, itemRepo conversation.ItemRepository) *ShareService {
	return &ShareService{
		repo:          repo,
		convRepo:      convRepo,
		itemRepo:      itemRepo,
		slugGenerator: NewSlugGenerator(repo),
	}
}

// CreateShareInput contains the input for creating a share
type CreateShareInput struct {
	ConversationID         uint
	ItemPublicID           *string // For single-message share
	OwnerUserID            uint
	Title                  *string
	Scope                  ShareScope
	IncludeImages          bool
	IncludeContextMessages bool    // For single-message share: include preceding user turn
	Branch                 *string // Branch to share from (defaults to active branch if not specified)
}

// CreateShareOutput contains the result of creating a share
type CreateShareOutput struct {
	Share    *Share
	ShareURL string
}

// CreateShare creates a new share for a conversation or item
func (s *ShareService) CreateShare(ctx context.Context, input CreateShareInput) (*CreateShareOutput, error) {
	// Fetch conversation
	conv, err := s.convRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "conversation not found", "7b8c9d0e-1f2a-4b3c-4d5e-6f7a8b9c0d1e")
	}

	// Validate ownership
	if conv.UserID != input.OwnerUserID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeForbidden,
			"you do not have permission to share this conversation", nil, "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d")
	}

	// Check if conversation is temporary
	if conv.PublicID == TemporaryChatID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			"temporary conversations cannot be shared", nil, "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e")
	}

	// Check if conversation is private
	if conv.IsPrivate {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			"private conversations cannot be shared", nil, "3c4d5e6f-7a8b-4c9d-0e1f-2a3b4c5d6e7f")
	}

	// Determine which branch to share from
	branchName := conv.ActiveBranch
	if input.Branch != nil && *input.Branch != "" {
		branchName = *input.Branch
	}

	// Fetch items from the specified branch
	itemPtrs, err := s.convRepo.GetBranchItems(ctx, input.ConversationID, branchName, nil)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to fetch conversation items", "8c9d0e1f-2a3b-4c4d-5e6f-7a8b9c0d1e2f")
	}

	// Convert []*Item to []Item for processing
	items := make([]conversation.Item, len(itemPtrs))
	for i, ptr := range itemPtrs {
		items[i] = *ptr
	}

	if len(items) < MinItemsForShare {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			"conversation must have at least 2 items to share", nil, "4d5e6f7a-8b9c-4d0e-1f2a-3b4c5d6e7f8a")
	}

	// For single-message share, validate item
	if input.Scope == ShareScopeItem && input.ItemPublicID != nil {
		if err := s.validateItemForShare(ctx, items, *input.ItemPublicID); err != nil {
			return nil, err
		}
	}

	// Revoke existing shares for this conversation (re-share creates new slug)
	if err := s.repo.RevokeAllByConversationID(ctx, input.ConversationID); err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to revoke existing shares", "9d0e1f2a-3b4c-4d5e-6f7a-8b9c0d1e2f3a")
	}

	// Generate unique slug
	slug, err := s.slugGenerator.GenerateUniqueSlug(ctx)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to generate share slug", "0e1f2a3b-4c5d-4e6f-7a8b-9c0d1e2f3a4b")
	}

	// Generate public ID
	publicID, err := GenerateSharePublicID()
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to generate share public ID", "1f2a3b4c-5d6e-4f7a-8b9c-0d1e2f3a4b5c")
	}

	// Build snapshot
	snapshot, err := s.buildSnapshot(ctx, conv, items, input)
	if err != nil {
		return nil, err
	}

	// Validate snapshot size
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to serialize snapshot", "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d")
	}
	if len(snapshotJSON) > MaxSnapshotSize {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			fmt.Sprintf("snapshot size (%d bytes) exceeds maximum allowed size (%d bytes)", len(snapshotJSON), MaxSnapshotSize),
			nil, "3a4b5c6d-7e8f-4a9b-0c1d-2e3f4a5b6c7d")
	}

	// Determine title
	title := s.determineTitle(conv, items, input.Title)

	// Create share
	share := &Share{
		PublicID:        publicID,
		Slug:            slug,
		ConversationID:  input.ConversationID,
		ItemPublicID:    input.ItemPublicID,
		OwnerUserID:     input.OwnerUserID,
		Title:           &title,
		Visibility:      VisibilityUnlisted,
		SnapshotVersion: 1,
		Snapshot:        snapshot,
		ShareOptions: &Options{
			IncludeImages:          input.IncludeImages,
			IncludeContextMessages: input.IncludeContextMessages,
		},
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to create share", "3b4c5d6e-7f8a-4b9c-0d1e-2f3a4b5c6d7e")
	}

	return &CreateShareOutput{
		Share: share,
	}, nil
}

// GetShareBySlug retrieves a share by its public slug
func (s *ShareService) GetShareBySlug(ctx context.Context, slug string) (*Share, error) {
	if !ValidateSlug(slug) {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			"invalid share link", nil, "5e6f7a8b-9c0d-4e1f-2a3b-4c5d6e7f8a9b")
	}

	share, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "share not found", "4c5d6e7f-8a9b-4c0d-1e2f-3a4b5c6d7e8f")
	}

	if share.IsRevoked() {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound,
			"this share has been revoked", nil, "6f7a8b9c-0d1e-4f2a-3b4c-5d6e7f8a9b0c")
	}

	// Increment view count asynchronously (fire and forget)
	go func() {
		// Use background context for async operation
		bgCtx := context.Background()
		_ = s.repo.IncrementViewCount(bgCtx, share.ID)
	}()

	return share, nil
}

// ListSharesByConversation lists all shares for a conversation
func (s *ShareService) ListSharesByConversation(ctx context.Context, conversationID uint, ownerUserID uint, includeRevoked bool) ([]*Share, error) {
	// Verify ownership
	conv, err := s.convRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "conversation not found", "5d6e7f8a-9b0c-4d1e-2f3a-4b5c6d7e8f9a")
	}

	if conv.UserID != ownerUserID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeForbidden,
			"you do not have permission to view shares for this conversation", nil, "7a8b9c0d-1e2f-4a3b-4c5d-6e7f8a9b0c1d")
	}

	filter := ShareFilter{
		ConversationID: &conversationID,
		IncludeRevoked: includeRevoked,
	}

	shares, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to list shares", "6e7f8a9b-0c1d-4e2f-3a4b-5c6d7e8f9a0b")
	}

	return shares, nil
}

// ListUserShares lists all shares for a user across all conversations
func (s *ShareService) ListUserShares(ctx context.Context, ownerUserID uint, includeRevoked bool) ([]*Share, error) {
	filter := ShareFilter{
		OwnerUserID:    &ownerUserID,
		IncludeRevoked: includeRevoked,
	}

	shares, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "failed to list user shares", "7e8f9a0b-1c2d-4e3f-5a6b-7c8d9e0f1a2b")
	}

	return shares, nil
}

// RevokeShare revokes a share
func (s *ShareService) RevokeShare(ctx context.Context, sharePublicID string, ownerUserID uint) error {
	share, err := s.repo.FindByPublicID(ctx, sharePublicID)
	if err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerDomain, err, "share not found", "7f8a9b0c-1d2e-4f3a-4b5c-6d7e8f9a0b1c")
	}

	if share.OwnerUserID != ownerUserID {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeForbidden,
			"you do not have permission to revoke this share", nil, "8b9c0d1e-2f3a-4b4c-5d6e-7f8a9b0c1d2e")
	}

	if share.IsRevoked() {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
			"share is already revoked", nil, "9c0d1e2f-3a4b-4c5d-6e7f-8a9b0c1d2e3f")
	}

	return s.repo.Revoke(ctx, share.ID)
}

// Helper methods

func (s *ShareService) getActiveItems(conv *conversation.Conversation) []conversation.Item {
	// Get items from active branch
	if conv.Branches != nil {
		if items, exists := conv.Branches[conv.ActiveBranch]; exists {
			return items
		}
	}
	// Fallback to legacy Items field
	return conv.Items
}

func (s *ShareService) validateItemForShare(ctx context.Context, items []conversation.Item, itemPublicID string) error {
	for _, item := range items {
		if item.PublicID == itemPublicID {
			// Must be an assistant message
			if item.Role == nil || *item.Role != conversation.ItemRoleAssistant {
				return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
					"only assistant messages can be shared individually", nil, "0d1e2f3a-4b5c-4d6e-7f8a-9b0c1d2e3f4a")
			}
			// Must be completed
			if item.Status == nil || *item.Status != conversation.ItemStatusCompleted {
				return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation,
					"only completed messages can be shared", nil, "1e2f3a4b-5c6d-4e7f-8a9b-0c1d2e3f4a5b")
			}
			return nil
		}
	}
	return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound,
		"item not found in conversation", nil, "2f3a4b5c-6d7e-4f8a-9b0c-1d2e3f4a5b6c")
}

func (s *ShareService) determineTitle(conv *conversation.Conversation, items []conversation.Item, requestTitle *string) string {
	// Use provided title if available
	if requestTitle != nil && *requestTitle != "" {
		return *requestTitle
	}

	// Use conversation title
	if conv.Title != nil && *conv.Title != "" {
		return *conv.Title
	}

	// Truncate first user message
	for _, item := range items {
		if item.Role != nil && *item.Role == conversation.ItemRoleUser && len(item.Content) > 0 {
			for _, content := range item.Content {
				text := extractTextFromContent(content)
				if text != "" {
					if len(text) > DefaultTitleMaxLength {
						return text[:DefaultTitleMaxLength] + "..."
					}
					return text
				}
			}
		}
	}

	return DefaultUntitledTitle
}

func (s *ShareService) buildSnapshot(ctx context.Context, conv *conversation.Conversation, items []conversation.Item, input CreateShareInput) (*Snapshot, error) {
	snapshot := &Snapshot{
		Title:     s.determineTitle(conv, items, input.Title),
		CreatedAt: conv.CreatedAt,
		Items:     make([]SnapshotItem, 0),
	}

	// Extract model/assistant name from metadata if available
	if conv.Metadata != nil {
		if model, ok := conv.Metadata["model"]; ok {
			snapshot.ModelName = &model
		}
		if assistant, ok := conv.Metadata["assistant"]; ok {
			snapshot.AssistantName = &assistant
		}
	}

	// Filter and sanitize items
	var itemsToInclude []conversation.Item
	if input.Scope == ShareScopeItem && input.ItemPublicID != nil {
		// Single-message share: include the target item and optionally preceding context
		itemsToInclude = s.getItemsForSingleShare(items, *input.ItemPublicID, input.IncludeContextMessages)
	} else {
		// Full conversation share: include all user/assistant items on active branch
		itemsToInclude = s.filterItemsForShare(items, input.IncludeImages)
	}

	for _, item := range itemsToInclude {
		snapshotItem := s.sanitizeItem(item, input.IncludeImages)
		if snapshotItem != nil {
			snapshot.Items = append(snapshot.Items, *snapshotItem)
		}
	}

	return snapshot, nil
}

func (s *ShareService) getItemsForSingleShare(items []conversation.Item, targetItemID string, includeContext bool) []conversation.Item {
	var result []conversation.Item
	var targetIndex int = -1

	// Find the target item
	for i, item := range items {
		if item.PublicID == targetItemID {
			targetIndex = i
			break
		}
	}

	if targetIndex < 0 {
		return result
	}

	// If including context, find the last user message before the target
	if includeContext && targetIndex > 0 {
		for i := targetIndex - 1; i >= 0; i-- {
			if items[i].Role != nil && *items[i].Role == conversation.ItemRoleUser {
				result = append(result, items[i])
				break
			}
		}
	}

	// Add the target item
	result = append(result, items[targetIndex])

	return result
}

func (s *ShareService) filterItemsForShare(items []conversation.Item, includeImages bool) []conversation.Item {
	var result []conversation.Item

	for _, item := range items {
		// Skip mcp_call items - they're redundant since the tool call info
		// is already in the assistant message's tool_calls content
		if item.Type == conversation.ItemTypeMcpCall {
			continue
		}

		// Skip system, developer, critic roles
		if item.Role == nil {
			continue
		}
		role := *item.Role
		if role == conversation.ItemRoleSystem ||
			role == conversation.ItemRoleDeveloper ||
			role == conversation.ItemRoleCritic ||
			role == conversation.ItemRoleDiscriminator {
			continue
		}

		// Include user, assistant, and tool messages
		if role != conversation.ItemRoleUser && role != conversation.ItemRoleAssistant && role != conversation.ItemRoleTool {
			continue
		}

		result = append(result, item)
	}

	return result
}

func (s *ShareService) sanitizeItem(item conversation.Item, includeImages bool) *SnapshotItem {
	role := ""
	if item.Role != nil {
		role = string(*item.Role)
	}

	snapshotItem := &SnapshotItem{
		ID:        item.PublicID,
		Type:      string(item.Type),
		Role:      role,
		Content:   make([]SnapshotContent, 0),
		CallID:    item.CallID, // For tool role messages to match with tool_calls
		CreatedAt: item.CreatedAt,
	}

	for _, content := range item.Content {
		sanitized := s.sanitizeContent(content, includeImages)
		if sanitized != nil {
			snapshotItem.Content = append(snapshotItem.Content, *sanitized)
		}
	}

	// Skip items with no content
	if len(snapshotItem.Content) == 0 {
		return nil
	}

	return snapshotItem
}

func (s *ShareService) sanitizeContent(content conversation.Content, includeImages bool) *SnapshotContent {
	switch content.Type {
	case "text":
		text := extractTextFromContent(content)
		if text == "" {
			return nil
		}
		return &SnapshotContent{
			Type: "text",
			Text: text,
		}

	case "input_text":
		text := extractTextFromContent(content)
		if text == "" {
			return nil
		}
		return &SnapshotContent{
			Type:      "input_text",
			InputText: text,
		}

	case "output_text":
		if content.OutputText == nil {
			return nil
		}
		sc := &SnapshotContent{
			Type:       "output_text",
			OutputText: content.OutputText.Text,
		}
		// Include annotations if present
		if len(content.OutputText.Annotations) > 0 {
			sc.Annotations = sanitizeAnnotations(content.OutputText.Annotations)
		}
		return sc

	case "image", "image_url":
		if !includeImages {
			return nil
		}
		if content.Image == nil {
			return nil
		}
		// Match the exact format from conversation items
		imageRef := &ImageRef{
			URL:    content.Image.URL,
			FileID: content.Image.FileID,
			Detail: content.Image.Detail,
		}
		return &SnapshotContent{
			Type:  "image",
			Image: imageRef,
		}

	case "file":
		if content.File == nil {
			return nil
		}
		// Only include file_id reference
		mimeType := content.File.MimeType
		name := content.File.Name
		return &SnapshotContent{
			Type: "file",
			FileRef: &FileRef{
				FileID:   content.File.FileID,
				MimeType: &mimeType,
				Name:     &name,
			},
		}

	case "reasoning_text":
		// Include reasoning/thinking content
		text := ""
		if content.Reasoning != nil {
			text = *content.Reasoning
		} else {
			text = extractTextFromContent(content)
		}
		if text == "" {
			return nil
		}
		return &SnapshotContent{
			Type: content.Type,
			Text: text,
		}

	case "tool_call_id":
		// Include tool call ID for tool role messages
		if content.ToolCallID == nil {
			return nil
		}
		return &SnapshotContent{
			Type:       "tool_call_id",
			ToolCallID: content.ToolCallID,
		}

	case "tool_calls":
		// Include tool calls from assistant messages
		if len(content.ToolCalls) == 0 {
			return nil
		}
		toolCalls := make([]ToolCall, 0, len(content.ToolCalls))
		for _, tc := range content.ToolCalls {
			toolCall := ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
			}
			if tc.Function.Name != "" {
				toolCall.Function = &ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
			toolCalls = append(toolCalls, toolCall)
		}
		return &SnapshotContent{
			Type:      "tool_calls",
			ToolCalls: toolCalls,
		}

	case "mcp_call":
		// Include MCP call data - include tool_calls if present
		if len(content.ToolCalls) > 0 {
			toolCalls := make([]ToolCall, 0, len(content.ToolCalls))
			for _, tc := range content.ToolCalls {
				toolCall := ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
				}
				if tc.Function.Name != "" {
					toolCall.Function = &ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					}
				}
				toolCalls = append(toolCalls, toolCall)
			}
			return &SnapshotContent{
				Type:       "mcp_call",
				ToolCalls:  toolCalls,
				ToolCallID: content.ToolCallID,
			}
		}
		// Fallback to text extraction for legacy format
		text := extractTextFromContent(content)
		return &SnapshotContent{
			Type:        "mcp_call",
			MCPCallData: text,
			ToolCallID:  content.ToolCallID,
		}

	case "tool_result":
		// Include tool result content with tool_call_id for frontend matching
		text := extractTextFromContent(content)
		if text == "" {
			return nil
		}
		return &SnapshotContent{
			Type:       "tool_result",
			ToolResult: text,
			ToolCallID: content.ToolCallID,
		}

	// Skip sensitive/internal content types
	case "audio", "input_audio":
		// Skip audio data entirely (contains audio.data, input_audio.data)
		return nil
	case "code":
		// Skip code outputs that might leak paths/env
		return nil
	case "computer_screenshot", "computer_action":
		// Skip computer use content
		return nil
	case "refusal":
		// Skip refusals
		return nil
	default:
		// Skip unknown types for safety
		return nil
	}
}

// extractTextFromContent extracts text from various content structures
func extractTextFromContent(content conversation.Content) string {
	if content.Text != nil {
		return content.Text.Text
	}
	if content.TextString != nil {
		return *content.TextString
	}
	if content.OutputText != nil {
		return content.OutputText.Text
	}
	if content.SummaryText != nil {
		return *content.SummaryText
	}
	return ""
}

// sanitizeAnnotations converts domain annotations to snapshot annotations
func sanitizeAnnotations(annotations []conversation.Annotation) []Annotation {
	result := make([]Annotation, 0, len(annotations))
	for _, a := range annotations {
		// Only include URL and file citations
		if a.Type != "url_citation" && a.Type != "file_citation" {
			continue
		}
		sa := Annotation{
			Type:     a.Type,
			Text:     a.Text,
			StartIdx: &a.StartIndex,
			EndIdx:   &a.EndIndex,
		}
		if a.URL != "" {
			sa.URL = a.URL
		}
		if a.FileID != "" {
			sa.FileID = a.FileID
		}
		result = append(result, sa)
	}
	return result
}
