package conversation

import (
	"context"
	"testing"
)

// MockConversationRepository is a mock implementation for testing
type MockConversationRepository struct {
	conversations map[string]*Conversation
	messages      map[string][]*Message
}

func NewMockConversationRepository() *MockConversationRepository {
	return &MockConversationRepository{
		conversations: make(map[string]*Conversation),
		messages:      make(map[string][]*Message),
	}
}

func (m *MockConversationRepository) Create(ctx context.Context, conv *Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id string) (*Conversation, error) {
	conv, ok := m.conversations[id]
	if !ok {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

func (m *MockConversationRepository) FindByUserID(ctx context.Context, userID string, filter ListFilter) ([]*Conversation, int64, error) {
	var result []*Conversation
	for _, conv := range m.conversations {
		if conv.UserID == userID {
			if filter.Search != "" && conv.Title != filter.Search {
				continue
			}
			result = append(result, conv)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockConversationRepository) Update(ctx context.Context, conv *Conversation) error {
	if _, ok := m.conversations[conv.ID]; !ok {
		return ErrConversationNotFound
	}
	m.conversations[conv.ID] = conv
	return nil
}

func (m *MockConversationRepository) Delete(ctx context.Context, id string) error {
	delete(m.conversations, id)
	delete(m.messages, id)
	return nil
}

func (m *MockConversationRepository) AddMessage(ctx context.Context, msg *Message) error {
	m.messages[msg.ConversationID] = append(m.messages[msg.ConversationID], msg)
	return nil
}

func (m *MockConversationRepository) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, int64, error) {
	msgs := m.messages[conversationID]
	total := len(msgs)

	if offset >= total {
		return []*Message{}, int64(total), nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return msgs[offset:end], int64(total), nil
}

func (m *MockConversationRepository) DeleteMessage(ctx context.Context, messageID string) error {
	for convID, msgs := range m.messages {
		for i, msg := range msgs {
			if msg.ID == messageID {
				m.messages[convID] = append(msgs[:i], msgs[i+1:]...)
				return nil
			}
		}
	}
	return ErrMessageNotFound
}

func (m *MockConversationRepository) FindSharedConversation(ctx context.Context, shareToken string) (*Conversation, error) {
	for _, conv := range m.conversations {
		if conv.ShareToken != nil && *conv.ShareToken == shareToken {
			return conv, nil
		}
	}
	return nil, ErrConversationNotFound
}

func TestService_Create(t *testing.T) {
	repo := NewMockConversationRepository()
	svc := NewService(repo)

	tests := []struct {
		name    string
		userID  string
		req     CreateConversationRequest
		wantErr bool
	}{
		{
			name:   "create conversation with title",
			userID: "user-123",
			req: CreateConversationRequest{
				Title:   "Test Conversation",
				ModelID: "gpt-4",
			},
			wantErr: false,
		},
		{
			name:   "create conversation without title",
			userID: "user-123",
			req: CreateConversationRequest{
				ModelID: "gpt-4",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := svc.Create(context.Background(), tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if conv.UserID != tt.userID {
					t.Errorf("Expected userID %s, got %s", tt.userID, conv.UserID)
				}
				if tt.req.Title != "" && conv.Title != tt.req.Title {
					t.Errorf("Expected title %s, got %s", tt.req.Title, conv.Title)
				}
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	repo := NewMockConversationRepository()
	svc := NewService(repo)

	// Create a conversation first
	conv, err := svc.Create(context.Background(), "user-123", CreateConversationRequest{
		Title: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	tests := []struct {
		name    string
		userID  string
		convID  string
		wantErr bool
	}{
		{
			name:    "get existing conversation",
			userID:  "user-123",
			convID:  conv.ID,
			wantErr: false,
		},
		{
			name:    "get non-existent conversation",
			userID:  "user-123",
			convID:  "non-existent",
			wantErr: true,
		},
		{
			name:    "get conversation from different user",
			userID:  "user-456",
			convID:  conv.ID,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetByID(context.Background(), tt.userID, tt.convID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.ID != tt.convID {
				t.Errorf("Expected conversation ID %s, got %s", tt.convID, result.ID)
			}
		})
	}
}

func TestService_AddMessage(t *testing.T) {
	repo := NewMockConversationRepository()
	svc := NewService(repo)

	// Create a conversation first
	conv, err := svc.Create(context.Background(), "user-123", CreateConversationRequest{
		Title: "Test",
	})
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	tests := []struct {
		name    string
		userID  string
		convID  string
		msg     *Message
		wantErr bool
	}{
		{
			name:   "add user message",
			userID: "user-123",
			convID: conv.ID,
			msg: &Message{
				Role:    "user",
				Content: "Hello!",
			},
			wantErr: false,
		},
		{
			name:   "add assistant message",
			userID: "user-123",
			convID: conv.ID,
			msg: &Message{
				Role:    "assistant",
				Content: "Hi there!",
			},
			wantErr: false,
		},
		{
			name:   "add message to non-existent conversation",
			userID: "user-123",
			convID: "non-existent",
			msg: &Message{
				Role:    "user",
				Content: "Hello!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AddMessage(context.Background(), tt.userID, tt.convID, tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.AddMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	repo := NewMockConversationRepository()
	svc := NewService(repo)

	// Create some conversations
	for i := 0; i < 5; i++ {
		_, err := svc.Create(context.Background(), "user-123", CreateConversationRequest{
			Title: "Test " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}
	}

	// Create conversations for another user
	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), "user-456", CreateConversationRequest{
			Title: "Other " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}
	}

	// Test listing
	convs, total, err := svc.List(context.Background(), "user-123", ListFilter{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Service.List() error = %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if len(convs) != 5 {
		t.Errorf("Expected 5 conversations, got %d", len(convs))
	}
}
