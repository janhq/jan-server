package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/tool"
)

// ServiceImpl provides the domain implementation.
type ServiceImpl struct {
	responses         Repository
	conversations     conversation.Repository
	conversationItems conversation.ItemRepository
	toolExecutions    ToolExecutionRepository
	orchestrator      *tool.Orchestrator
	mcpClient         tool.MCPClient
	log               zerolog.Logger
}

// NewService wires dependencies.
func NewService(
	responses Repository,
	conversations conversation.Repository,
	conversationItems conversation.ItemRepository,
	toolExecutions ToolExecutionRepository,
	orchestrator *tool.Orchestrator,
	mcpClient tool.MCPClient,
	log zerolog.Logger,
) *ServiceImpl {
	return &ServiceImpl{
		responses:         responses,
		conversations:     conversations,
		conversationItems: conversationItems,
		toolExecutions:    toolExecutions,
		orchestrator:      orchestrator,
		mcpClient:         mcpClient,
		log:               log.With().Str("component", "response-service").Logger(),
	}
}

// Create orchestrates a complete response lifecycle.
func (s *ServiceImpl) Create(ctx context.Context, params CreateParams) (*Response, error) {
	var conv *conversation.Conversation
	var err error

	if params.ConversationID != nil && strings.TrimSpace(*params.ConversationID) != "" {
		conv, err = s.conversations.FindByPublicID(ctx, *params.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("fetch conversation: %w", err)
		}
	} else {
		conv = &conversation.Conversation{
			PublicID: newPublicID("conv"),
			UserID:   params.UserID,
		}
		if err := s.conversations.Create(ctx, conv); err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
	}

	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("list conversation items: %w", err)
	}

	responseModel := &Response{
		PublicID:             newPublicID("resp"),
		Object:               "response",
		UserID:               params.UserID,
		Model:                params.Model,
		SystemPrompt:         params.SystemPrompt,
		Input:                params.Input,
		Status:               StatusInProgress,
		Stream:               params.Stream,
		Metadata:             params.Metadata,
		ConversationID:       &conv.ID,
		ConversationPublicID: &conv.PublicID,
		PreviousResponseID:   params.PreviousResponseID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if params.StreamObserver != nil {
		params.StreamObserver.OnResponseCreated(responseModel)
	}

	if err := s.responses.Create(ctx, responseModel); err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	baseMessages, err := s.buildBaseMessages(params.SystemPrompt, existingItems)
	if err != nil {
		return s.failResponse(ctx, responseModel, fmt.Errorf("build base messages: %w", err))
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), params.Input)
	if err != nil {
		return s.failResponse(ctx, responseModel, err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	toolDefs := params.Tools
	if len(toolDefs) == 0 {
		if toolDefs, err = s.fetchAvailableTools(ctx); err != nil {
			return s.failResponse(ctx, responseModel, err)
		}
	}

	execParams := func(defs []llm.ToolDefinition, toolChoice *llm.ToolChoice) tool.ExecuteParams {
		return tool.ExecuteParams{
			Ctx:             ctx,
			Model:           params.Model,
			Messages:        messages,
			Temperature:     params.Temperature,
			MaxTokens:       params.MaxTokens,
			ToolChoice:      toolChoice,
			ToolDefinitions: defs,
			StreamObserver:  params.StreamObserver,
		}
	}

	orchestratorResult, err := s.orchestrator.Execute(execParams(toolDefs, params.ToolChoice))
	if err != nil && shouldRetryWithoutTools(err) && len(toolDefs) > 0 {
		s.log.Warn().Err(err).Str("response_id", responseModel.PublicID).Msg("llm provider rejected tool definitions, retrying without tools")
		orchestratorResult, err = s.orchestrator.Execute(execParams(nil, nil))
	}
	if err != nil {
		return s.failResponse(ctx, responseModel, err)
	}

	responseModel.Status = StatusCompleted
	responseModel.Output = orchestratorResult.FinalMessage.Content
	responseModel.Usage = orchestratorResult.Usage
	now := time.Now()
	responseModel.CompletedAt = &now
	responseModel.UpdatedAt = now

	if err := s.responses.Update(ctx, responseModel); err != nil {
		return nil, err
	}

	if err := s.toolExecutions.RecordExecutions(ctx, responseModel.ID, orchestratorResult.Executions); err != nil {
		s.log.Error().Err(err).Str("response_id", responseModel.PublicID).Msg("store tool executions failed")
	}

	newMessages := orchestratorResult.Messages[initialLength:]
	newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
	if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
		s.log.Error().Err(err).Str("response_id", responseModel.PublicID).Msg("store conversation items failed")
	}

	return responseModel, nil
}

// GetByPublicID returns the response by id.
func (s *ServiceImpl) GetByPublicID(ctx context.Context, publicID string) (*Response, error) {
	return s.responses.FindByPublicID(ctx, publicID)
}

// Cancel marks the response as cancelled.
func (s *ServiceImpl) Cancel(ctx context.Context, publicID string) (*Response, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	if resp.Status == StatusCompleted || resp.Status == StatusCancelled {
		return resp, nil
	}

	if err := s.responses.MarkCancelled(ctx, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListConversationItems returns the textual conversation history for the response.
func (s *ServiceImpl) ListConversationItems(ctx context.Context, publicID string) ([]ConversationItem, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	if resp.ConversationID == nil {
		return nil, errors.New("response has no conversation")
	}

	items, err := s.conversationItems.ListByConversationID(ctx, *resp.ConversationID)
	if err != nil {
		return nil, err
	}

	result := make([]ConversationItem, 0, len(items))
	for _, item := range items {
		result = append(result, ConversationItem{
			Role:    string(item.Role),
			Content: item.Content,
			Status:  string(item.Status),
		})
	}
	return result, nil
}

func (s *ServiceImpl) failResponse(ctx context.Context, resp *Response, failure error) (*Response, error) {
	now := time.Now()
	resp.Status = StatusFailed
	resp.FailedAt = &now
	resp.Error = &ErrorDetails{
		Code:    "response_failed",
		Message: failure.Error(),
	}
	if err := s.responses.Update(ctx, resp); err != nil {
		s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("update failed response")
	}
	return nil, failure
}

func (s *ServiceImpl) buildBaseMessages(systemPrompt *string, items []conversation.Item) ([]llm.ChatMessage, error) {
	messages := make([]llm.ChatMessage, 0, len(items)+1)
	if systemPrompt != nil && strings.TrimSpace(*systemPrompt) != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "system",
			Content: strings.TrimSpace(*systemPrompt),
		})
	}

	for _, item := range items {
		messages = append(messages, llm.ChatMessage{
			Role:    string(item.Role),
			Content: contentToLLM(item.Content),
		})
	}
	return messages, nil
}

func (s *ServiceImpl) convertInputToMessages(conversationID uint, startingSeq int, input interface{}) ([]llm.ChatMessage, []conversation.Item, error) {
	var messages []llm.ChatMessage
	var convoItems []conversation.Item

	switch v := input.(type) {
	case string:
		msg := llm.ChatMessage{Role: "user", Content: strings.TrimSpace(v)}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	case []interface{}:
		for _, raw := range v {
			msg, err := mapToChatMessage(raw)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, msg)
			convoItems = append(convoItems, newConversationItem(conversationID, startingSeq+len(convoItems), msg))
		}
	case map[string]interface{}:
		msg, err := mapToChatMessage(v)
		if err != nil {
			return nil, nil, err
		}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	default:
		bytes, _ := json.Marshal(input)
		msg := llm.ChatMessage{
			Role:    "user",
			Content: string(bytes),
		}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	}

	return messages, convoItems, nil
}

func (s *ServiceImpl) convertMessagesToItems(conversationID uint, startingSeq int, messages []llm.ChatMessage) []conversation.Item {
	items := make([]conversation.Item, 0, len(messages))
	for _, msg := range messages {
		items = append(items, newConversationItem(conversationID, startingSeq+len(items), msg))
	}
	return items
}

func (s *ServiceImpl) fetchAvailableTools(ctx context.Context) ([]llm.ToolDefinition, error) {
	mcpTools, err := s.mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}

	defs := make([]llm.ToolDefinition, 0, len(mcpTools))
	for _, tool := range mcpTools {
		defs = append(defs, tool.ToLLMTool())
	}
	return defs, nil
}

func newConversationItem(conversationID uint, sequence int, msg llm.ChatMessage) conversation.Item {
	content := normalizeContent(msg.Content)
	role := conversation.ItemRole(msg.Role)
	if role == "" {
		role = conversation.RoleUser
	}
	return conversation.Item{
		ConversationID: conversationID,
		Role:           role,
		Status:         conversation.ItemStatusCompleted,
		Content:        content,
		Sequence:       sequence + 1,
		CreatedAt:      time.Now(),
	}
}

func contentToLLM(content map[string]interface{}) interface{} {
	if content == nil {
		return nil
	}
	if text, ok := content["text"]; ok {
		return text
	}
	return content
}

func normalizeContent(content interface{}) map[string]interface{} {
	switch v := content.(type) {
	case string:
		return map[string]interface{}{"type": "text", "text": v}
	case map[string]interface{}:
		return v
	case []interface{}:
		return map[string]interface{}{"type": "list", "items": v}
	default:
		bytes, _ := json.Marshal(v)
		return map[string]interface{}{"type": "json", "text": string(bytes)}
	}
}

func mapToChatMessage(raw interface{}) (llm.ChatMessage, error) {
	payload, ok := raw.(map[string]interface{})
	if !ok {
		return llm.ChatMessage{}, errors.New("input items must be objects with role/content")
	}

	role, _ := payload["role"].(string)
	if role == "" {
		role = "user"
	}
	content := payload["content"]
	if content == nil {
		content = payload["text"]
	}
	if content == nil {
		return llm.ChatMessage{}, errors.New("input item missing content")
	}

	return llm.ChatMessage{
		Role:    role,
		Content: content,
	}, nil
}

func newPublicID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.NewString())
}

func shouldRetryWithoutTools(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "failed to complete chat request") {
		return true
	}
	if strings.Contains(message, "tools unsupported") {
		return true
	}
	return false
}
