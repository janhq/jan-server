package chathandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/domain"
	"jan-server/services/llm-api/infrastructure/idempotency"
	"jan-server/services/llm-api/infrastructure/provider"
	"jan-server/services/llm-api/infrastructure/repo"
	chatrequests "jan-server/services/llm-api/interfaces/httpserver/requests/chat"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
	chatresponses "jan-server/services/llm-api/interfaces/httpserver/responses/chat"
	"jan-server/services/llm-api/utils/idgen"
)

// middleware functions we'll need
func PrepareSSE(c *gin.Context) (http.Flusher, bool) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	flusher, ok := c.Writer.(http.Flusher)
	return flusher, ok
}

func PrincipalFromContext(c *gin.Context) (domain.Principal, bool) {
	val, ok := c.Get("principal")
	if !ok {
		return domain.Principal{}, false
	}
	principal, ok := val.(domain.Principal)
	return principal, ok
}

func RequestIDFromContext(c *gin.Context) string {
	if val, ok := c.Get("X-Request-Id"); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ChatHandler handles chat completion requests with conversation support
type ChatHandler struct {
	registry         *provider.Registry
	store            *idempotency.Store
	conversationRepo *repo.ConversationRepository
	logger           zerolog.Logger
}

// NewChatHandler constructs an enhanced chat handler with conversation support
func NewChatHandler(
	registry *provider.Registry,
	store *idempotency.Store,
	conversationRepo *repo.ConversationRepository,
	logger zerolog.Logger,
) *ChatHandler {
	return &ChatHandler{
		registry:         registry,
		store:            store,
		conversationRepo: conversationRepo,
		logger:           logger,
	}
}

// ChatCompletionResult wraps the response with conversation context
type ChatCompletionResult struct {
	Response          map[string]interface{}
	RawBody           []byte
	StatusCode        int
	Headers           map[string]string
	ConversationID    string
	ConversationTitle *string
}

// ChatCompletions handles POST /v1/chat/completions with streaming and conversation support
// @Summary Create chat completion
// @Description Creates a model response for the given chat conversation. Supports streaming, non-streaming, and conversation persistence.
// @Tags chat
// @Security BearerJWT
// @Security ApiKey
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body chatrequests.ChatCompletionRequest true "Chat completion request"
// @Success 200 {object} chatresponses.ChatCompletionResponse "Non-streaming response"
// @Success 200 {string} string "Streaming response (SSE format)"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Model not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (h *ChatHandler) ChatCompletions(c *gin.Context) {
	ctx := c.Request.Context()
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_payload",
			Message:   "failed to read request body",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	defer c.Request.Body.Close()

	var req chatrequests.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_json",
			Message:   err.Error(),
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	// Validate required fields
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "model_required",
			Message:   "model is required",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "messages_required",
			Message:   "messages must be provided",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	// Process conversation context if provided
	result, err := h.createChatCompletion(ctx, c, principal, req)
	if err != nil {
		h.logger.Error().Err(err).Msg("chat completion failed")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "completion_failed",
			Message:   err.Error(),
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	// For non-streaming, return JSON response with conversation context
	if !req.Stream {
		// Parse response body and add conversation context
		var responseData map[string]interface{}
		if err := json.Unmarshal(result.RawBody, &responseData); err == nil {
			if result.ConversationID != "" {
				convCtx := chatresponses.ChatConversationContext{
					ID:    result.ConversationID,
					Title: result.ConversationTitle,
				}
				responseData["conversation"] = convCtx
			}

			// Set headers
			for k, v := range result.Headers {
				c.Header(k, v)
			}

			c.JSON(result.StatusCode, responseData)
		} else {
			// Fallback to raw response
			for k, v := range result.Headers {
				c.Header(k, v)
			}
			c.Data(result.StatusCode, "application/json", result.RawBody)
		}
	}
	// For streaming, response is already sent in streamChatCompletion
}

// createChatCompletion handles the core chat completion logic
func (h *ChatHandler) createChatCompletion(
	ctx context.Context,
	c *gin.Context,
	principal domain.Principal,
	req chatrequests.ChatCompletionRequest,
) (*ChatCompletionResult, error) {
	var conv *domain.Conversation
	var conversationID string
	newMessagesCount := len(req.Messages)

	// Get or create conversation if conversation context is provided
	if req.Conversation != nil && !req.Conversation.IsEmpty() {
		convID := req.Conversation.GetID()
		if convID != "" {
			// Fetch existing conversation
			existingConv, err := h.conversationRepo.Get(ctx, convID)
			if err != nil {
				return nil, fmt.Errorf("failed to get conversation: %w", err)
			}
			// Verify ownership
			if existingConv != nil && existingConv.OwnerPrincipalID != principal.ID {
				return nil, fmt.Errorf("conversation not found or access denied")
			}
			conv = existingConv

			// Prepend conversation messages to request messages
			req.Messages = h.prependConversationMessages(conv, req.Messages)
		} else {
			// Create new conversation
			createdConv, err := h.createConversation(ctx, principal.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create conversation: %w", err)
			}
			conv = createdConv

			// Auto-generate title from first message
			if title := h.generateTitleFromMessages(req.Messages); title != "" {
				conv.Title = title
				// TODO: Add Update method to ConversationRepository
				// For now, title will be set on next update when conversation service is fully implemented
				h.logger.Debug().Str("conversation_id", conv.ID).Str("title", title).Msg("generated conversation title (update pending)")
			}
		}
		conversationID = conv.ID
	}

	// Resolve model and provider
	route, err := h.registry.Resolve(req.Model)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", req.Model)
	}

	// Build provider request
	providerReq := h.buildProviderRequest(req, route)

	// Handle idempotency
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey != "" {
		record, err := h.store.Get(ctx, principal.ID, c.Request.Method, c.FullPath(), idempotencyKey)
		if err != nil {
			h.logger.Warn().Err(err).Msg("idempotency lookup failed")
		} else if record != nil {
			if req.Stream {
				replayStream(c, record)
			} else {
				return &ChatCompletionResult{
					RawBody:           record.Response,
					StatusCode:        record.Status,
					Headers:           make(map[string]string),
					ConversationID:    conversationID,
					ConversationTitle: h.getConversationTitle(conv),
				}, nil
			}
			return &ChatCompletionResult{}, nil
		}
	}

	forwardHeaders := map[string]string{
		"X-Principal-Id": principal.ID,
		"X-Auth-Method":  string(principal.AuthMethod),
	}
	if len(principal.Scopes) > 0 {
		forwardHeaders["X-Scopes"] = strings.Join(principal.Scopes, " ")
	}

	// Execute completion
	var result *ChatCompletionResult
	if req.Stream {
		result, err = h.streamChatCompletion(ctx, c, route, providerReq, forwardHeaders, conv, principal, idempotencyKey)
	} else {
		result, err = h.callChatCompletion(ctx, route, providerReq, forwardHeaders, conv)
	}

	if err != nil {
		return nil, err
	}

	// Store conversation if requested
	storeConversation := true
	if req.Store != nil {
		storeConversation = *req.Store
	}

	if conv != nil && storeConversation && !req.Stream && result.RawBody != nil {
		storeReasoning := false
		if req.StoreReasoning != nil {
			storeReasoning = *req.StoreReasoning
		}

		// Extract the new messages that were just added
		newMessages := make([]chatrequests.ChatMessage, newMessagesCount)
		if newMessagesCount > 0 {
			copy(newMessages, req.Messages[len(req.Messages)-newMessagesCount:])
		}

		if err := h.addCompletionToConversation(ctx, conv, newMessages, result.RawBody, storeReasoning); err != nil {
			h.logger.Warn().Err(err).Msg("failed to store completion in conversation")
		}
	}

	result.ConversationID = conversationID
	result.ConversationTitle = h.getConversationTitle(conv)

	return result, nil
}

// callChatCompletion handles non-streaming completions
func (h *ChatHandler) callChatCompletion(
	ctx context.Context,
	route provider.Route,
	req provider.ChatCompletionRequest,
	forwardHeaders map[string]string,
	conv *domain.Conversation,
) (*ChatCompletionResult, error) {
	response, err := route.Provider.ChatCompletions(ctx, req, forwardHeaders)
	if err != nil {
		return nil, fmt.Errorf("provider call failed: %w", err)
	}

	return &ChatCompletionResult{
		RawBody:    response.Body,
		StatusCode: response.StatusCode,
		Headers:    response.Headers,
	}, nil
}

// streamChatCompletion handles streaming completions
func (h *ChatHandler) streamChatCompletion(
	ctx context.Context,
	c *gin.Context,
	route provider.Route,
	req provider.ChatCompletionRequest,
	forwardHeaders map[string]string,
	conv *domain.Conversation,
	principal domain.Principal,
	idempotencyKey string,
) (*ChatCompletionResult, error) {
	flusher, ok := PrepareSSE(c)
	if !ok {
		return nil, fmt.Errorf("response writer cannot stream")
	}

	stream, err := route.Provider.ChatCompletionsStream(ctx, req, forwardHeaders)
	if err != nil {
		return nil, fmt.Errorf("provider stream failed: %w", err)
	}
	defer stream.Close()

	for k, v := range stream.Headers() {
		c.Header(k, v)
	}

	status := stream.StatusCode()
	if status == 0 {
		status = http.StatusOK
	}
	c.Status(status)

	var buffer bytes.Buffer
	conversationSent := false

	err = stream.Stream(ctx, func(chunk []byte) error {
		if len(chunk) == 0 {
			return nil
		}

		// Write chunk to client
		if _, err := c.Writer.Write(chunk); err != nil {
			return err
		}
		buffer.Write(chunk)
		flusher.Flush()

		// Send conversation context before [DONE] marker
		if !conversationSent && conv != nil {
			chunkStr := string(chunk)
			if strings.Contains(chunkStr, "[DONE]") {
				// Send conversation context
				convData := map[string]interface{}{
					"id": conv.ID,
				}
				if conv.Title != "" {
					convData["title"] = conv.Title
				}

				convChunk := map[string]interface{}{
					"conversation": convData,
					"created":      time.Now().Unix(),
					"id":           "",
					"model":        req.Model,
					"object":       "chat.completion.chunk",
				}

				chunkJSON, _ := json.Marshal(convChunk)
				c.Writer.Write([]byte("data: "))
				c.Writer.Write(chunkJSON)
				c.Writer.Write([]byte("\n\n"))
				flusher.Flush()
				conversationSent = true
			}
		}

		return nil
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("streaming to client failed")
		return nil, err
	}

	// Store idempotent stream
	if idempotencyKey != "" {
		if saveErr := h.store.Save(ctx, &idempotency.Record{
			Key:         idempotencyKey,
			PrincipalID: principal.ID,
			Method:      c.Request.Method,
			Path:        c.FullPath(),
			Status:      status,
			Response:    buffer.Bytes(),
			CreatedAt:   time.Now().UTC(),
		}); saveErr != nil {
			h.logger.Warn().Err(saveErr).Msg("store idempotent stream")
		}
	}

	return &ChatCompletionResult{
		StatusCode: status,
		Headers:    stream.Headers(),
	}, nil
}

// buildProviderRequest converts transport request to provider request
func (h *ChatHandler) buildProviderRequest(
	req chatrequests.ChatCompletionRequest,
	route provider.Route,
) provider.ChatCompletionRequest {
	providerReq := provider.ChatCompletionRequest{
		Model:    route.Model.ServedName,
		Messages: make([]map[string]any, len(req.Messages)),
		Stream:   req.Stream,
		Extras:   make(map[string]any),
	}

	// Convert messages
	for i, msg := range req.Messages {
		msgMap := map[string]any{
			"role": msg.Role,
		}
		if msg.Content != nil {
			msgMap["content"] = msg.Content
		}
		if msg.Name != "" {
			msgMap["name"] = msg.Name
		}
		if msg.FunctionCall != nil {
			msgMap["function_call"] = msg.FunctionCall
		}
		if msg.ToolCalls != nil {
			msgMap["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			msgMap["tool_call_id"] = msg.ToolCallID
		}
		if msg.ReasoningContent != "" {
			msgMap["reasoning_content"] = msg.ReasoningContent
		}
		providerReq.Messages[i] = msgMap
	}

	// Copy parameters
	providerReq.Temperature = req.Temperature
	providerReq.TopP = req.TopP
	providerReq.MaxTokens = req.MaxTokens
	providerReq.Metadata = req.Metadata

	// Copy extras
	if req.FrequencyPenalty != nil {
		providerReq.Extras["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		providerReq.Extras["presence_penalty"] = *req.PresencePenalty
	}
	if req.Stop != nil {
		providerReq.Extras["stop"] = req.Stop
	}
	if req.N != nil {
		providerReq.Extras["n"] = *req.N
	}

	return providerReq
}

// Helper functions for conversation management

// createConversation creates a new conversation with a secure generated ID
func (h *ChatHandler) createConversation(ctx context.Context, ownerID string) (*domain.Conversation, error) {
	convID, err := idgen.GenerateSecureID("conv", 16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate conversation ID: %w", err)
	}

	conv := &domain.Conversation{
		ID:               convID,
		OwnerPrincipalID: ownerID,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.conversationRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

func (h *ChatHandler) getConversationTitle(conv *domain.Conversation) *string {
	if conv == nil || conv.Title == "" {
		return nil
	}
	title := conv.Title
	return &title
}

func (h *ChatHandler) generateTitleFromMessages(messages []chatrequests.ChatMessage) string {
	for _, msg := range messages {
		if msg.Role == "user" {
			if content, ok := msg.Content.(string); ok && content != "" {
				content = strings.TrimSpace(content)
				if len(content) > 60 {
					if lastSpace := strings.LastIndex(content[:60], " "); lastSpace > 30 {
						content = content[:lastSpace] + "..."
					} else {
						content = content[:60] + "..."
					}
				}
				return content
			}
		}
	}
	return "New Conversation"
}

func (h *ChatHandler) prependConversationMessages(
	conv *domain.Conversation,
	messages []chatrequests.ChatMessage,
) []chatrequests.ChatMessage {
	// TODO: Implement conversation message prepending
	// For now, just return the original messages
	// This will be implemented when we have the full conversation item structure
	return messages
}

func (h *ChatHandler) addCompletionToConversation(
	ctx context.Context,
	conv *domain.Conversation,
	newMessages []chatrequests.ChatMessage,
	responseBody []byte,
	storeReasoning bool,
) error {
	// TODO: Implement conversation persistence
	// Parse response and create conversation items
	// This will be implemented with the full conversation service
	h.logger.Debug().
		Str("conversation_id", conv.ID).
		Int("new_messages", len(newMessages)).
		Msg("storing completion in conversation (not yet implemented)")
	return nil
}

func replayStream(c *gin.Context, record *idempotency.Record) {
	flusher, ok := PrepareSSE(c)
	if ok {
		c.Status(record.Status)
		if len(record.Response) > 0 {
			if _, err := c.Writer.Write(record.Response); err == nil {
				flusher.Flush()
			}
		}
		return
	}
	c.Data(record.Status, "text/event-stream", record.Response)
}
