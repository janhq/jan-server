package repository

import (
	"context"
	"encoding/json"
	"errors"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// ConversationRepository implements conversation.Repository using GORM.
type ConversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository creates a new conversation repository.
func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

var _ conversation.Repository = (*ConversationRepository)(nil)

// ============================================
// Conversation operations
// ============================================

func (r *ConversationRepository) Create(ctx context.Context, conv *conversation.Conversation) error {
	schema := toConversationSchema(conv)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	conv.ID = schema.ID
	conv.CreatedAt = schema.CreatedAt
	conv.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*conversation.Conversation, error) {
	var schema dbschema.Conversation
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, conversation.ErrConversationNotFound
		}
		return nil, err
	}
	return toConversationDomain(&schema), nil
}

func (r *ConversationRepository) GetByIDWithMessages(ctx context.Context, id string, messageLimit int) (*conversation.Conversation, error) {
	var schema dbschema.Conversation
	query := r.db.WithContext(ctx).Where("id = ?", id)

	if messageLimit > 0 {
		query = query.Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(messageLimit)
		})
	} else {
		query = query.Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		})
	}

	if err := query.First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, conversation.ErrConversationNotFound
		}
		return nil, err
	}

	conv := toConversationDomain(&schema)
	for _, m := range schema.Messages {
		msg := toMessageDomain(&m)
		conv.Messages = append(conv.Messages, *msg)
	}

	return conv, nil
}

func (r *ConversationRepository) Update(ctx context.Context, conv *conversation.Conversation) error {
	schema := toConversationSchema(conv)
	return r.db.WithContext(ctx).Model(&dbschema.Conversation{}).Where("id = ?", conv.ID).Updates(schema).Error
}

func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Conversation{}, "id = ?", id).Error
}

func (r *ConversationRepository) List(ctx context.Context, filter conversation.ListConversationsFilter) ([]*conversation.Conversation, int64, error) {
	var schemas []dbschema.Conversation
	var total int64

	query := r.db.WithContext(ctx).Model(&dbschema.Conversation{}).Where("user_id = ?", filter.UserID)

	if filter.IsArchived != nil {
		query = query.Where("is_archived = ?", *filter.IsArchived)
	}
	if filter.IsPinned != nil {
		query = query.Where("is_pinned = ?", *filter.IsPinned)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("updated_at DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	conversations := make([]*conversation.Conversation, len(schemas))
	for i, s := range schemas {
		conversations[i] = toConversationDomain(&s)
	}

	return conversations, total, nil
}

func (r *ConversationRepository) GetBySharedToken(ctx context.Context, token string) (*conversation.Conversation, error) {
	var schema dbschema.Conversation
	if err := r.db.WithContext(ctx).
		Where("shared_token = ?", token).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, conversation.ErrConversationNotFound
		}
		return nil, err
	}

	conv := toConversationDomain(&schema)
	for _, m := range schema.Messages {
		msg := toMessageDomain(&m)
		conv.Messages = append(conv.Messages, *msg)
	}

	return conv, nil
}

// ============================================
// Message operations
// ============================================

func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *conversation.Message) error {
	schema := toMessageSchema(msg)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	msg.ID = schema.ID
	msg.CreatedAt = schema.CreatedAt
	return nil
}

func (r *ConversationRepository) GetMessageByID(ctx context.Context, id string) (*conversation.Message, error) {
	var schema dbschema.Message
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, conversation.ErrMessageNotFound
		}
		return nil, err
	}
	return toMessageDomain(&schema), nil
}

func (r *ConversationRepository) ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*conversation.Message, int64, error) {
	var schemas []dbschema.Message
	var total int64

	query := r.db.WithContext(ctx).Model(&dbschema.Message{}).Where("conversation_id = ?", conversationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at ASC").Offset(offset).Limit(limit).Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	messages := make([]*conversation.Message, len(schemas))
	for i, s := range schemas {
		messages[i] = toMessageDomain(&s)
	}

	return messages, total, nil
}

func (r *ConversationRepository) DeleteMessage(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Message{}, "id = ?", id).Error
}

// ============================================
// Conversion helpers
// ============================================

func toConversationSchema(c *conversation.Conversation) *dbschema.Conversation {
	metadataJSON, _ := json.Marshal(c.Metadata)
	isArchived := c.IsArchived
	isPinned := c.IsPinned
	return &dbschema.Conversation{
		ID:           c.ID,
		UserID:       c.UserID,
		Title:        c.Title,
		ModelID:      c.ModelID,
		SystemPrompt: c.SystemPrompt,
		Metadata:     metadataJSON,
		IsArchived:   &isArchived,
		IsPinned:     &isPinned,
		SharedToken:  c.SharedToken,
		SharedAt:     c.SharedAt,
		ParentID:     c.ParentID,
	}
}

func toConversationDomain(s *dbschema.Conversation) *conversation.Conversation {
	var metadata map[string]any
	_ = json.Unmarshal(s.Metadata, &metadata)

	isArchived := false
	if s.IsArchived != nil {
		isArchived = *s.IsArchived
	}
	isPinned := false
	if s.IsPinned != nil {
		isPinned = *s.IsPinned
	}

	return &conversation.Conversation{
		ID:           s.ID,
		UserID:       s.UserID,
		Title:        s.Title,
		ModelID:      s.ModelID,
		SystemPrompt: s.SystemPrompt,
		Metadata:     metadata,
		IsArchived:   isArchived,
		IsPinned:     isPinned,
		SharedToken:  s.SharedToken,
		SharedAt:     s.SharedAt,
		ParentID:     s.ParentID,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func toMessageSchema(m *conversation.Message) *dbschema.Message {
	toolCallsJSON, _ := json.Marshal(m.ToolCalls)
	attachmentsJSON, _ := json.Marshal(m.Attachments)
	metadataJSON, _ := json.Marshal(m.Metadata)

	return &dbschema.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           m.Role,
		Content:        m.Content,
		Name:           m.Name,
		ToolCallID:     m.ToolCallID,
		ToolCalls:      toolCallsJSON,
		Attachments:    attachmentsJSON,
		Metadata:       metadataJSON,
		TokensInput:    m.TokensInput,
		TokensOutput:   m.TokensOutput,
		ModelID:        m.ModelID,
		ParentID:       m.ParentID,
	}
}

func toMessageDomain(s *dbschema.Message) *conversation.Message {
	var toolCalls []conversation.ToolCall
	var attachments []conversation.Attachment
	var metadata map[string]any

	_ = json.Unmarshal(s.ToolCalls, &toolCalls)
	_ = json.Unmarshal(s.Attachments, &attachments)
	_ = json.Unmarshal(s.Metadata, &metadata)

	return &conversation.Message{
		ID:             s.ID,
		ConversationID: s.ConversationID,
		Role:           s.Role,
		Content:        s.Content,
		Name:           s.Name,
		ToolCallID:     s.ToolCallID,
		ToolCalls:      toolCalls,
		Attachments:    attachments,
		Metadata:       metadata,
		TokensInput:    s.TokensInput,
		TokensOutput:   s.TokensOutput,
		ModelID:        s.ModelID,
		ParentID:       s.ParentID,
		CreatedAt:      s.CreatedAt,
	}
}
