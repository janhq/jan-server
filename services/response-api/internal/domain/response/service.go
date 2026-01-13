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

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/webhook"
)

// ServiceImpl provides the domain implementation.
type ServiceImpl struct {
	responses         Repository
	conversations     conversation.Repository
	conversationItems conversation.ItemRepository
	toolExecutions    ToolExecutionRepository
	orchestrator      *tool.Orchestrator
	mcpClient         tool.MCPClient
	modelInfoProvider llm.ModelInfoProvider
	webhookService    webhook.Service
	agentRegistry     agent.Registry
	planService       plan.Service
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
	modelInfoProvider llm.ModelInfoProvider,
	webhookService webhook.Service,
	agentRegistry agent.Registry,
	planService plan.Service,
	log zerolog.Logger,
) *ServiceImpl {
	return &ServiceImpl{
		responses:         responses,
		conversations:     conversations,
		conversationItems: conversationItems,
		toolExecutions:    toolExecutions,
		orchestrator:      orchestrator,
		mcpClient:         mcpClient,
		modelInfoProvider: modelInfoProvider,
		webhookService:    webhookService,
		agentRegistry:     agentRegistry,
		planService:       planService,
		log:               log.With().Str("component", "response-service").Logger(),
	}
}

// Create orchestrates a complete response lifecycle.
// If Background=true, it enqueues the task and returns immediately.
// Otherwise, it executes synchronously.
func (s *ServiceImpl) Create(ctx context.Context, params CreateParams) (*Response, error) {
	// Validate background constraints
	if params.Background && !params.Store {
		return nil, errors.New("background mode requires store=true")
	}

	if params.Background {
		return s.createAsync(ctx, params)
	}
	return s.createSync(ctx, params)
}

// createAsync creates a response record with status=queued and returns immediately.
func (s *ServiceImpl) createAsync(ctx context.Context, params CreateParams) (*Response, error) {
	var conv *conversation.Conversation
	var err error

	// Handle conversation context
	if params.PreviousResponseID != nil && strings.TrimSpace(*params.PreviousResponseID) != "" {
		prevResp, err := s.responses.FindByPublicID(ctx, *params.PreviousResponseID)
		if err != nil {
			s.log.Warn().Err(err).Str("previous_response_id", *params.PreviousResponseID).Msg("failed to load previous response")
		} else if prevResp.ConversationPublicID != nil {
			conv, err = s.conversations.FindByPublicID(ctx, *prevResp.ConversationPublicID)
			if err != nil {
				s.log.Warn().Err(err).Str("conversation_id", *prevResp.ConversationPublicID).Msg("failed to load conversation")
			}
		}
	}

	if conv == nil {
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
	}

	now := time.Now()
	responseModel := &Response{
		PublicID:             newPublicID("resp"),
		Object:               "response",
		UserID:               params.UserID,
		Model:                params.Model,
		SystemPrompt:         params.SystemPrompt,
		Input:                params.Input,
		Status:               StatusQueued,
		Stream:               params.Stream,
		Background:           params.Background,
		Store:                params.Store,
		APIKey:               params.APIKey, // Store API key for background execution
		Metadata:             params.Metadata,
		ConversationID:       &conv.ID,
		ConversationPublicID: &conv.PublicID,
		PreviousResponseID:   params.PreviousResponseID,
		CreatedAt:            now,
		UpdatedAt:            now,
		QueuedAt:             &now,
	}

	if err := s.responses.Create(ctx, responseModel); err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	s.log.Info().
		Str("response_id", responseModel.PublicID).
		Str("user_id", params.UserID).
		Str("model", params.Model).
		Msg("background response queued")

	return responseModel, nil
}

// createSync executes the response synchronously (original behavior).
func (s *ServiceImpl) createSync(ctx context.Context, params CreateParams) (*Response, error) {
	var conv *conversation.Conversation
	var err error
	conversationID := ""

	// If PreviousResponseID is provided, load that response's conversation for context
	if params.PreviousResponseID != nil && strings.TrimSpace(*params.PreviousResponseID) != "" {
		prevResp, err := s.responses.FindByPublicID(ctx, *params.PreviousResponseID)
		if err != nil {
			s.log.Warn().Err(err).Str("previous_response_id", *params.PreviousResponseID).Msg("failed to load previous response, continuing without context")
		} else if prevResp.ConversationPublicID != nil {
			// Use the previous response's conversation to maintain context
			conv, err = s.conversations.FindByPublicID(ctx, *prevResp.ConversationPublicID)
			if err != nil {
				s.log.Warn().Err(err).Str("conversation_id", *prevResp.ConversationPublicID).Msg("failed to load conversation, creating new one")
			}
		}
	}

	if conv != nil {
		conversationID = conv.PublicID
	}

	// If we still don't have a conversation, check for explicit conversation_id or create new
	if conv == nil {
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
		Background:           params.Background,
		Store:                params.Store,
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

	// Fetch model context length for message trimming
	var contextLength *int
	if s.modelInfoProvider != nil {
		modelInfo, err := s.modelInfoProvider.GetModelInfo(ctx, params.Model)
		if err != nil {
			s.log.Warn().Err(err).Str("model", params.Model).Msg("failed to fetch model info, using default context length")
		} else if modelInfo != nil && modelInfo.ContextLength != nil {
			contextLength = modelInfo.ContextLength
		}
	}

	execParams := func(defs []llm.ToolDefinition, toolChoice *llm.ToolChoice) tool.ExecuteParams {
		return tool.ExecuteParams{
			Ctx:             ctx,
			Model:           params.Model,
			Messages:        messages,
			RequestID:       params.RequestID,
			ConversationID:  conversationID,
			UserID:          params.UserID,
			Temperature:     params.Temperature,
			MaxTokens:       params.MaxTokens,
			ContextLength:   contextLength,
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

// GetWithPlanDetails returns the response with its associated plan, tasks, and steps.
func (s *ServiceImpl) GetWithPlanDetails(ctx context.Context, publicID string) (*ResponseWithPlan, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	result := &ResponseWithPlan{
		Response: resp,
	}

	// Try to get plan details if plan service is available
	if s.planService != nil {
		planDetails, err := s.planService.GetByResponseID(ctx, publicID)
		if err == nil && planDetails != nil {
			// Get full plan with tasks and steps
			fullPlan, err := s.planService.GetPlanWithDetails(ctx, planDetails.ID)
			if err == nil {
				result.PlanDetails = fullPlan
			} else {
				// Fallback to basic plan info
				result.PlanDetails = planDetails
			}
		}
		// If no plan exists, PlanDetails will be nil (omitempty)
	}

	return result, nil
}

// Cancel marks the response as cancelled.
// For queued tasks, this prevents them from being picked up by workers.
// For in-progress tasks, workers should periodically check cancellation status.
func (s *ServiceImpl) Cancel(ctx context.Context, publicID string) (*Response, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	// Already in terminal state
	if resp.Status == StatusCompleted || resp.Status == StatusCancelled || resp.Status == StatusFailed {
		return resp, nil
	}

	// Cancel the response
	if err := s.responses.MarkCancelled(ctx, resp); err != nil {
		return nil, err
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("previous_status", string(resp.Status)).
		Msg("response cancelled")

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
			Role:    string(item.GetRole()),
			Content: item.GetContent(),
			Status:  string(item.GetStatus()),
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
			Role:    string(item.GetRole()),
			Content: contentToLLM(item.GetContent()),
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
		LegacyContent:  content,
		Sequence:       sequence + 1,
		SequenceNumber: sequence + 1,
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

func mapToChatMessage(messageData interface{}) (llm.ChatMessage, error) {
	payload, ok := messageData.(map[string]interface{})
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

// ExecuteBackground processes a queued background task.
// This method is called by workers from the worker pool.
func (s *ServiceImpl) ExecuteBackground(ctx context.Context, publicID string) error {
	// Load the response record
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return fmt.Errorf("failed to load response: %w", err)
	}
	requestID := publicID // reuse response identifier for traceability

	// Verify it's in a processable state
	if resp.Status != StatusInProgress {
		return fmt.Errorf("response %s is not in_progress (current: %s)", publicID, resp.Status)
	}

	// Inject API key into context for LLM API calls
	if resp.APIKey != nil && *resp.APIKey != "" {
		ctx = llm.ContextWithAuthToken(ctx, *resp.APIKey)
	}

	// Check if this is an agent-based request that requires a plan
	agentType := s.extractAgentType(resp.Metadata)
	if agentType != "" && s.agentRegistry != nil && s.planService != nil {
		s.log.Info().
			Str("response_id", publicID).
			Str("agent_type", agentType).
			Msg("detected agent request, creating plan")
		return s.executeWithPlan(ctx, resp, agentType)
	}

	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}
	conversationID := conv.PublicID

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Build messages from history and current input
	baseMessages, err := s.buildBaseMessages(resp.SystemPrompt, existingItems)
	if err != nil {
		return fmt.Errorf("build base messages: %w", err)
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), resp.Input)
	if err != nil {
		return fmt.Errorf("convert input: %w", err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	// Load tool definitions
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Fetch model context length for message trimming
	var contextLength *int
	if s.modelInfoProvider != nil {
		modelInfo, err := s.modelInfoProvider.GetModelInfo(ctx, resp.Model)
		if err != nil {
			s.log.Warn().Err(err).Str("model", resp.Model).Msg("failed to fetch model info for background task, using default context length")
		} else if modelInfo != nil && modelInfo.ContextLength != nil {
			contextLength = modelInfo.ContextLength
		}
	}

	// Execute orchestration (no streaming in background mode)
	execParams := tool.ExecuteParams{
		Ctx:             ctx,
		Model:           resp.Model,
		Messages:        messages,
		RequestID:       requestID,
		ConversationID:  conversationID,
		UserID:          resp.UserID,
		Temperature:     nil, // Use model defaults for background tasks
		MaxTokens:       nil,
		ContextLength:   contextLength,
		ToolDefinitions: toolDefs,
		StreamObserver:  nil, // Background mode never streams
	}

	orchestratorResult, execErr := s.orchestrator.Execute(execParams)
	if execErr != nil && shouldRetryWithoutTools(execErr) && len(toolDefs) > 0 {
		s.log.Warn().Err(execErr).Str("response_id", resp.PublicID).Msg("llm provider rejected tool definitions, retrying without tools")
		execParams.ToolDefinitions = nil
		orchestratorResult, execErr = s.orchestrator.Execute(execParams)
	}

	// Update response status
	now := time.Now()
	if execErr != nil {
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{Message: execErr.Error()}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
	} else {
		resp.Status = StatusCompleted
		resp.Output = orchestratorResult.FinalMessage.Content
		resp.Usage = orchestratorResult.Usage
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		// Record tool executions
		if err := s.toolExecutions.RecordExecutions(ctx, resp.ID, orchestratorResult.Executions); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store tool executions failed")
		}

		// Record conversation items (skip initial messages, only store new ones)
		newMessages := orchestratorResult.Messages[initialLength:]
		newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
		if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store conversation items failed")
		}
	}

	// Persist final state
	if err := s.responses.Update(ctx, resp); err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	// Send webhook notifications (async, don't block on webhook failures)
	go func() {
		webhookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if execErr != nil {
			errorCode := "execution_failed"
			errorMsg := execErr.Error()
			if resp.Error != nil {
				errorCode = resp.Error.Code
				errorMsg = resp.Error.Message
			}
			if err := s.webhookService.NotifyFailed(webhookCtx, resp.PublicID, errorCode, errorMsg, resp.Metadata); err != nil {
				s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("webhook notification failed")
			}
		} else {
			if err := s.webhookService.NotifyCompleted(webhookCtx, resp.PublicID, resp.Output, resp.Metadata, resp.CompletedAt); err != nil {
				s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("webhook notification failed")
			}
		}
	}()

	return execErr
}

// extractAgentType extracts the agent_type from metadata if present.
func (s *ServiceImpl) extractAgentType(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	agentType, ok := metadata["agent_type"]
	if !ok {
		return ""
	}
	agentTypeStr, ok := agentType.(string)
	if !ok {
		return ""
	}
	return agentTypeStr
}

// executeWithPlan handles agent-based execution with plan creation and tracking.
func (s *ServiceImpl) executeWithPlan(ctx context.Context, resp *Response, agentType string) error {
	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Extract user message from input
	userMessage := s.extractUserMessage(resp.Input)

	// Load tool definitions for the plan request
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools for plan, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Convert tool definitions to agent format
	agentTools := make([]agent.ToolDefinition, len(toolDefs))
	for i, td := range toolDefs {
		paramsBytes, _ := json.Marshal(td.Function.Parameters)
		agentTools[i] = agent.ToolDefinition{
			Name:        td.Function.Name,
			Description: td.Function.Description,
			Parameters:  paramsBytes,
		}
	}

	// Convert history to agent format
	agentHistory := make([]agent.ConversationItem, len(existingItems))
	for i, item := range existingItems {
		content := ""
		// Convert Content slice to string
		if len(item.Content) > 0 {
			// Try to extract text from the first content item
			if b, err := json.Marshal(item.Content); err == nil {
				content = string(b)
			}
		}
		agentHistory[i] = agent.ConversationItem{
			Role:    string(item.Role),
			Content: content,
		}
	}

	// Build plan request
	planRequest := &agent.PlanRequest{
		ResponseID:          resp.PublicID,
		ConversationID:      conv.PublicID,
		UserMessage:         userMessage,
		SystemPrompt:        resp.SystemPrompt,
		Model:               resp.Model,
		Tools:               agentTools,
		ConversationHistory: agentHistory,
		Metadata:            resp.Metadata,
	}

	// Find a planner that can handle this request
	planner, err := s.agentRegistry.FindPlanner(ctx, planRequest)
	if err != nil {
		s.log.Warn().
			Err(err).
			Str("agent_type", agentType).
			Msg("no planner found for agent type, falling back to standard execution")
		// Fall through to standard execution by returning a marker error
		return s.executeWithoutPlan(ctx, resp)
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("planner", planner.Name()).
		Msg("creating execution plan")

	// Create the plan
	planResult, err := planner.CreatePlan(ctx, planRequest)
	if err != nil {
		now := time.Now()
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{
			Code:    "plan_creation_failed",
			Message: fmt.Sprintf("failed to create plan: %v", err),
		}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
		if updateErr := s.responses.Update(ctx, resp); updateErr != nil {
			s.log.Error().Err(updateErr).Msg("failed to update response after plan creation failure")
		}
		return err
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("plan_id", planResult.Plan.ID).
		Int("task_count", len(planResult.Tasks)).
		Msg("plan created successfully")

	// Execute the plan using the standard orchestrator but track progress via plan
	return s.executePlanWithOrchestrator(ctx, resp, planResult)
}

// executeWithoutPlan falls back to standard tool orchestration without plan tracking.
func (s *ServiceImpl) executeWithoutPlan(ctx context.Context, resp *Response) error {
	// This is essentially the original ExecuteBackground logic
	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}
	conversationID := conv.PublicID
	requestID := resp.PublicID

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Build messages from history and current input
	baseMessages, err := s.buildBaseMessages(resp.SystemPrompt, existingItems)
	if err != nil {
		return fmt.Errorf("build base messages: %w", err)
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), resp.Input)
	if err != nil {
		return fmt.Errorf("convert input: %w", err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	// Load tool definitions
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Execute orchestration
	execParams := tool.ExecuteParams{
		Ctx:             ctx,
		Model:           resp.Model,
		Messages:        messages,
		RequestID:       requestID,
		ConversationID:  conversationID,
		UserID:          resp.UserID,
		ToolDefinitions: toolDefs,
		StreamObserver:  nil,
	}

	orchestratorResult, execErr := s.orchestrator.Execute(execParams)
	if execErr != nil && shouldRetryWithoutTools(execErr) && len(toolDefs) > 0 {
		execParams.ToolDefinitions = nil
		orchestratorResult, execErr = s.orchestrator.Execute(execParams)
	}

	// Update response status
	now := time.Now()
	if execErr != nil {
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{Message: execErr.Error()}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
	} else {
		resp.Status = StatusCompleted
		resp.Output = orchestratorResult.FinalMessage.Content
		resp.Usage = orchestratorResult.Usage
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		if err := s.toolExecutions.RecordExecutions(ctx, resp.ID, orchestratorResult.Executions); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store tool executions failed")
		}

		newMessages := orchestratorResult.Messages[initialLength:]
		newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
		if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store conversation items failed")
		}
	}

	if err := s.responses.Update(ctx, resp); err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	return execErr
}

// executePlanWithOrchestrator executes a plan using the tool orchestrator while updating plan progress.
func (s *ServiceImpl) executePlanWithOrchestrator(ctx context.Context, resp *Response, planResult *agent.PlanResult) error {
	// Load conversation for context
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}
	conversationID := conv.PublicID
	requestID := resp.PublicID

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Build messages from history and current input
	baseMessages, err := s.buildBaseMessages(resp.SystemPrompt, existingItems)
	if err != nil {
		return fmt.Errorf("build base messages: %w", err)
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), resp.Input)
	if err != nil {
		return fmt.Errorf("convert input: %w", err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	// Load tool definitions
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Update plan status to in_progress
	if err := s.planService.UpdateStatus(ctx, planResult.Plan.ID, status.StatusInProgress, nil); err != nil {
		s.log.Warn().Err(err).Msg("failed to update plan status to in_progress")
	}

	// Start the first task
	if _, err := s.planService.StartNextTask(ctx, planResult.Plan.ID); err != nil {
		s.log.Warn().Err(err).Msg("failed to start first task")
	}

	// Execute orchestration
	execParams := tool.ExecuteParams{
		Ctx:             ctx,
		Model:           resp.Model,
		Messages:        messages,
		RequestID:       requestID,
		ConversationID:  conversationID,
		UserID:          resp.UserID,
		ToolDefinitions: toolDefs,
		StreamObserver:  nil,
	}

	orchestratorResult, execErr := s.orchestrator.Execute(execParams)
	if execErr != nil && shouldRetryWithoutTools(execErr) && len(toolDefs) > 0 {
		execParams.ToolDefinitions = nil
		orchestratorResult, execErr = s.orchestrator.Execute(execParams)
	}

	// Extract partial results from depth exceeded error
	var partialExecutions []tool.Execution
	if depthErr, ok := execErr.(*tool.DepthExceededError); ok {
		partialExecutions = depthErr.Executions
	}

	// Update response and plan status
	now := time.Now()
	if execErr != nil {
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{Message: execErr.Error()}
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		// Mark plan as failed
		errMsg := execErr.Error()
		if err := s.planService.UpdateStatus(ctx, planResult.Plan.ID, status.StatusFailed, &errMsg); err != nil {
			s.log.Error().Err(err).Msg("failed to update plan status to failed")
		}
	} else {
		resp.Status = StatusCompleted
		resp.Output = orchestratorResult.FinalMessage.Content
		resp.Usage = orchestratorResult.Usage
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		// Mark plan as completed
		if err := s.planService.UpdateStatus(ctx, planResult.Plan.ID, status.StatusCompleted, nil); err != nil {
			s.log.Error().Err(err).Msg("failed to update plan status to completed")
		}

		// Update plan progress to 100%
		if err := s.planService.UpdateProgress(ctx, planResult.Plan.ID, planResult.Plan.EstimatedSteps); err != nil {
			s.log.Error().Err(err).Msg("failed to update plan progress")
		}

		if err := s.toolExecutions.RecordExecutions(ctx, resp.ID, orchestratorResult.Executions); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store tool executions failed")
		}

		newMessages := orchestratorResult.Messages[initialLength:]
		newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
		if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store conversation items failed")
		}
	}

	// Complete all tasks in the plan and their steps
	// Get executions from either successful result or partial error results
	var executions []tool.Execution
	if orchestratorResult != nil {
		executions = orchestratorResult.Executions
	} else if len(partialExecutions) > 0 {
		executions = partialExecutions
	}

	executionIndex := 0
	for _, task := range planResult.Tasks {
		// Complete each step in the task
		for _, step := range task.Steps {
			var outputData []byte

			// For tool_call steps, try to match with tool executions in order
			if step.Action == plan.ActionTypeToolCall && executionIndex < len(executions) {
				exec := executions[executionIndex]
				outputBytes, _ := json.Marshal(map[string]interface{}{
					"tool_name":  exec.ToolName,
					"call_id":    exec.CallID,
					"status":     exec.Status,
					"result":     exec.Result,
					"error":      exec.ErrorMessage,
					"created_at": exec.CreatedAt,
				})
				outputData = outputBytes
				executionIndex++
			} else if step.Action == plan.ActionTypeLLMCall && orchestratorResult != nil {
				// For LLM calls, include final output summary
				if orchestratorResult.FinalMessage.Content != "" {
					outputBytes, _ := json.Marshal(map[string]interface{}{
						"type":    "llm_response",
						"content": orchestratorResult.FinalMessage.Content,
					})
					outputData = outputBytes
				}
			}

			if err := s.planService.CompleteStep(ctx, step.ID, outputData); err != nil {
				s.log.Warn().Err(err).Str("step_id", step.ID).Msg("failed to complete step")
			}
		}

		if err := s.planService.CompleteTask(ctx, task.ID); err != nil {
			s.log.Warn().Err(err).Str("task_id", task.ID).Msg("failed to complete task")
		}
	}

	if err := s.responses.Update(ctx, resp); err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("plan_id", planResult.Plan.ID).
		Str("status", string(resp.Status)).
		Msg("plan execution completed")

	return execErr
}

// extractUserMessage extracts the user message from the input.
func (s *ServiceImpl) extractUserMessage(input interface{}) string {
	if input == nil {
		return ""
	}
	switch v := input.(type) {
	case string:
		return v
	case []interface{}:
		// Handle array of message objects
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if role, _ := m["role"].(string); role == "user" {
					if content, ok := m["content"].(string); ok {
						return content
					}
				}
			}
		}
	}
	// Fallback: serialize to JSON
	if b, err := json.Marshal(input); err == nil {
		return string(b)
	}
	return ""
}
