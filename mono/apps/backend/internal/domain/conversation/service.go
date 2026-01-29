package conversation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrUnauthorized         = errors.New("unauthorized")
)

// Repository defines the interface for conversation data operations.
type Repository interface {
	// Conversation operations
	Create(ctx context.Context, conv *Conversation) error
	GetByID(ctx context.Context, id string) (*Conversation, error)
	GetByIDWithMessages(ctx context.Context, id string, messageLimit int) (*Conversation, error)
	Update(ctx context.Context, conv *Conversation) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListConversationsFilter) ([]*Conversation, int64, error)
	GetBySharedToken(ctx context.Context, token string) (*Conversation, error)

	// Message operations
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessageByID(ctx context.Context, id string) (*Message, error)
	ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, int64, error)
	DeleteMessage(ctx context.Context, id string) error
}

// Service handles conversation-related business logic.
type Service struct {
	repo Repository
}

// NewService creates a new conversation service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new conversation.
func (s *Service) Create(ctx context.Context, userID string, req CreateConversationRequest) (*Conversation, error) {
	conv := &Conversation{
		UserID:       userID,
		Title:        req.Title,
		ModelID:      req.ModelID,
		SystemPrompt: req.SystemPrompt,
		Metadata:     req.Metadata,
		IsArchived:   false,
		IsPinned:     false,
	}

	if conv.Title == "" {
		conv.Title = "New Conversation"
	}

	if err := s.repo.Create(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// GetByID retrieves a conversation by ID.
func (s *Service) GetByID(ctx context.Context, userID, id string) (*Conversation, error) {
	conv, err := s.repo.GetByIDWithMessages(ctx, id, 50)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	return conv, nil
}

// Update updates a conversation.
func (s *Service) Update(ctx context.Context, userID, id string, req UpdateConversationRequest) (*Conversation, error) {
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	if req.Title != nil {
		conv.Title = *req.Title
	}
	if req.ModelID != nil {
		conv.ModelID = *req.ModelID
	}
	if req.SystemPrompt != nil {
		conv.SystemPrompt = *req.SystemPrompt
	}
	if req.IsArchived != nil {
		conv.IsArchived = *req.IsArchived
	}
	if req.IsPinned != nil {
		conv.IsPinned = *req.IsPinned
	}
	if req.Metadata != nil {
		conv.Metadata = req.Metadata
	}

	if err := s.repo.Update(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// Delete deletes a conversation.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrConversationNotFound
	}

	if conv.UserID != userID {
		return ErrUnauthorized
	}

	return s.repo.Delete(ctx, id)
}

// List lists conversations for a user.
func (s *Service) List(ctx context.Context, filter ListConversationsFilter) ([]*Conversation, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	return s.repo.List(ctx, filter)
}

// Branch creates a copy of a conversation for branching.
func (s *Service) Branch(ctx context.Context, userID, id string, fromMessageID *string) (*Conversation, error) {
	conv, err := s.repo.GetByIDWithMessages(ctx, id, 0)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Create new conversation
	newConv := &Conversation{
		UserID:       userID,
		Title:        conv.Title + " (Branch)",
		ModelID:      conv.ModelID,
		SystemPrompt: conv.SystemPrompt,
		Metadata:     conv.Metadata,
		ParentID:     &id,
		IsArchived:   false,
		IsPinned:     false,
	}

	if err := s.repo.Create(ctx, newConv); err != nil {
		return nil, err
	}

	// Copy messages up to the specified message
	for _, msg := range conv.Messages {
		newMsg := &Message{
			ConversationID: newConv.ID,
			Role:           msg.Role,
			Content:        msg.Content,
			Name:           msg.Name,
			ToolCallID:     msg.ToolCallID,
			ToolCalls:      msg.ToolCalls,
			Attachments:    msg.Attachments,
			Metadata:       msg.Metadata,
			ParentID:       &msg.ID,
		}
		if err := s.repo.CreateMessage(ctx, newMsg); err != nil {
			return nil, err
		}

		if fromMessageID != nil && msg.ID == *fromMessageID {
			break
		}
	}

	return newConv, nil
}

// Share generates a share token for a conversation.
func (s *Service) Share(ctx context.Context, userID, id string) (*Conversation, error) {
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Generate share token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	now := time.Now()

	conv.SharedToken = &token
	conv.SharedAt = &now

	if err := s.repo.Update(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// Unshare removes the share token from a conversation.
func (s *Service) Unshare(ctx context.Context, userID, id string) (*Conversation, error) {
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	conv.SharedToken = nil
	conv.SharedAt = nil

	if err := s.repo.Update(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// GetBySharedToken retrieves a conversation by its share token.
func (s *Service) GetBySharedToken(ctx context.Context, token string) (*Conversation, error) {
	return s.repo.GetBySharedToken(ctx, token)
}

// AddMessage adds a message to a conversation.
func (s *Service) AddMessage(ctx context.Context, userID, conversationID string, msg *Message) error {
	conv, err := s.repo.GetByID(ctx, conversationID)
	if err != nil {
		return ErrConversationNotFound
	}

	if conv.UserID != userID {
		return ErrUnauthorized
	}

	msg.ConversationID = conversationID
	return s.repo.CreateMessage(ctx, msg)
}

// ListMessages lists messages in a conversation.
func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, limit, offset int) ([]*Message, int64, error) {
	conv, err := s.repo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, 0, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, 0, ErrUnauthorized
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return s.repo.ListMessages(ctx, conversationID, limit, offset)
}

// GetMessage retrieves a message by ID.
func (s *Service) GetMessage(ctx context.Context, userID, id string) (*Message, error) {
	msg, err := s.repo.GetMessageByID(ctx, id)
	if err != nil {
		return nil, ErrMessageNotFound
	}

	// Verify ownership via conversation
	conv, err := s.repo.GetByID(ctx, msg.ConversationID)
	if err != nil {
		return nil, ErrConversationNotFound
	}

	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	return msg, nil
}
