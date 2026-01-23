package messageshandler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/observability"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	messagesrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/messages"
	messagesresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/messages"
	"jan-server/services/llm-api/internal/utils/httpclients/chat"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

const (
	dataPrefix = "data: "
	doneMarker = "[DONE]"
)

// MessagesHandler handles Anthropic Messages API requests
type MessagesHandler struct {
	inferenceProvider   *inference.InferenceProvider
	providerHandler     *modelhandler.ProviderHandler
	conversationService *conversation.ConversationService
}

// NewMessagesHandler creates a new messages handler
func NewMessagesHandler(
	inferenceProvider *inference.InferenceProvider,
	providerHandler *modelhandler.ProviderHandler,
	conversationService *conversation.ConversationService,
) *MessagesHandler {
	return &MessagesHandler{
		inferenceProvider:   inferenceProvider,
		providerHandler:     providerHandler,
		conversationService: conversationService,
	}
}

// CreateMessage handles POST /v1/messages
func (h *MessagesHandler) CreateMessage(ctx context.Context, reqCtx *gin.Context, userID uint, request messagesrequests.AnthropicMessagesRequest) error {
	// Start OpenTelemetry span
	ctx, span := observability.StartSpan(ctx, "llm-api", "MessagesHandler.CreateMessage")
	defer span.End()

	startTime := time.Now()

	// Add span attributes
	observability.AddSpanAttributes(ctx,
		attribute.String("anthropic.model", request.Model),
		attribute.Bool("anthropic.stream", request.Stream),
		attribute.Int("anthropic.message_count", len(request.Messages)),
		attribute.Int("anthropic.max_tokens", request.MaxTokens),
		attribute.Int("user.id", int(userID)),
	)

	// Get provider for the requested model
	observability.AddSpanEvent(ctx, "selecting_provider")
	selectedProviderModel, selectedProvider, err := h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
	if err != nil {
		observability.RecordError(ctx, err)
		return h.writeErrorResponse(reqCtx, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Model not found: %s", request.Model))
	}

	if selectedProviderModel == nil || selectedProvider == nil {
		return h.writeErrorResponse(reqCtx, http.StatusNotFound, "not_found_error", fmt.Sprintf("Model not found: %s", request.Model))
	}

	// Add provider info to span
	observability.AddSpanAttributes(ctx,
		attribute.String("provider.display_name", selectedProvider.DisplayName),
		attribute.String("provider.id", selectedProvider.PublicID),
		attribute.String("model.original_id", selectedProviderModel.ProviderOriginalModelID),
	)

	// Get chat completion client
	chatClient, err := h.inferenceProvider.GetChatCompletionClient(ctx, selectedProvider)
	if err != nil {
		observability.RecordError(ctx, err)
		return h.writeErrorResponse(reqCtx, http.StatusInternalServerError, "api_error", "Failed to initialize model client")
	}

	// Convert Anthropic request to OpenAI format
	openaiRequest := ConvertAnthropicToOpenAI(&request)
	openaiRequest.Model = selectedProviderModel.ProviderOriginalModelID

	// Handle conversation context if provided
	var conv *conversation.Conversation
	var conversationID string
	if request.Conversation != nil && request.Conversation.GetID() != "" {
		conversationID = request.Conversation.GetID()
		conv, err = h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
		if err != nil {
			// Log but don't fail - conversation is optional
			observability.AddSpanEvent(ctx, "conversation_not_found", attribute.String("conversation_id", conversationID))
		}
	}

	// Build the completion request
	llmRequest := chat.CompletionRequest{
		ChatCompletionRequest: openaiRequest,
	}
	if request.TopK != nil {
		llmRequest.TopK = request.TopK
	}

	observability.AddSpanEvent(ctx, "calling_llm")
	llmStartTime := time.Now()

	if request.Stream {
		err = h.streamCompletion(ctx, reqCtx, chatClient, llmRequest, request.Model, conv, conversationID)
	} else {
		err = h.callCompletion(ctx, reqCtx, chatClient, llmRequest, request.Model, conv, conversationID, userID, request)
	}

	llmDuration := time.Since(llmStartTime)
	totalDuration := time.Since(startTime)

	observability.AddSpanAttributes(ctx,
		attribute.Float64("completion.llm_duration_ms", float64(llmDuration.Milliseconds())),
		attribute.Float64("completion.total_duration_ms", float64(totalDuration.Milliseconds())),
	)

	if err != nil {
		observability.RecordError(ctx, err)
		return err
	}

	observability.SetSpanStatus(ctx, codes.Ok, "anthropic message created successfully")
	return nil
}

// callCompletion handles non-streaming completion
func (h *MessagesHandler) callCompletion(
	ctx context.Context,
	reqCtx *gin.Context,
	chatClient *chat.ChatCompletionClient,
	request chat.CompletionRequest,
	originalModel string,
	conv *conversation.Conversation,
	conversationID string,
	userID uint,
	anthropicReq messagesrequests.AnthropicMessagesRequest,
) error {
	// Make the completion request
	response, err := chatClient.CreateChatCompletion(ctx, "", request)
	if err != nil {
		return h.writeErrorResponse(reqCtx, http.StatusInternalServerError, "api_error", "Failed to generate response")
	}

	// Convert to Anthropic format
	anthropicResponse := ConvertOpenAIToAnthropic(response, originalModel)

	// Generate message ID
	if msgID, err := idgen.GenerateSecureID("msg", 24); err == nil {
		anthropicResponse.ID = msgID
	}

	// Add conversation context if available
	if conversationID != "" {
		anthropicResponse.Conversation = &messagesresponses.ConversationContext{
			ID: conversationID,
		}
		if conv != nil && conv.Title != nil {
			anthropicResponse.Conversation.Title = conv.Title
		}
	}

	// Store to conversation if requested
	if anthropicReq.Store != nil && *anthropicReq.Store && conv != nil {
		h.storeToConversation(ctx, conv, anthropicReq, response)
	}

	reqCtx.JSON(http.StatusOK, anthropicResponse)
	return nil
}

// streamCompletion handles streaming completion with Anthropic SSE format
func (h *MessagesHandler) streamCompletion(
	ctx context.Context,
	reqCtx *gin.Context,
	chatClient *chat.ChatCompletionClient,
	request chat.CompletionRequest,
	originalModel string,
	conv *conversation.Conversation,
	conversationID string,
) error {
	// Ensure streaming is enabled
	request.Stream = true
	request.StreamOptions = &openai.StreamOptions{
		IncludeUsage: true,
	}

	// Create streaming request
	reader, err := chatClient.CreateChatCompletionStream(ctx, "", request)
	if err != nil {
		return h.writeErrorResponse(reqCtx, http.StatusInternalServerError, "api_error", "Failed to start streaming")
	}
	defer reader.Close()

	// Setup SSE headers for Anthropic format
	h.setupAnthropicSSEHeaders(reqCtx)

	// Generate message ID
	messageID, _ := idgen.GenerateSecureID("msg", 24)

	// Initialize streaming state
	state := NewStreamingState(messageID, originalModel)

	// Create scanner for reading SSE events
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 12*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse data lines
		if strings.HasPrefix(line, dataPrefix) {
			data := strings.TrimPrefix(line, dataPrefix)

			// Check for done marker
			if data == doneMarker {
				// Finalize the stream
				finalEvents := FinalizeAnthropicStream(state)
				for _, event := range finalEvents {
					if err := h.writeAnthropicSSEEvent(reqCtx, event); err != nil {
						return err
					}
				}
				break
			}

			// Convert OpenAI chunk to Anthropic events
			events := ConvertOpenAIStreamChunkToAnthropic(data, state)
			for _, event := range events {
				if err := h.writeAnthropicSSEEvent(reqCtx, event); err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "streaming error")
	}

	return nil
}

// setupAnthropicSSEHeaders sets up headers for Anthropic SSE streaming
func (h *MessagesHandler) setupAnthropicSSEHeaders(reqCtx *gin.Context) {
	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Transfer-Encoding", "chunked")
	reqCtx.Writer.WriteHeaderNow()
}

// writeAnthropicSSEEvent writes a single Anthropic SSE event
func (h *MessagesHandler) writeAnthropicSSEEvent(reqCtx *gin.Context, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Get event type
	eventType := "message"
	if m, ok := event.(map[string]interface{}); ok {
		if t, ok := m["type"].(string); ok {
			eventType = t
		}
	} else {
		// Try to extract type from struct
		if bytes, err := json.Marshal(event); err == nil {
			var m map[string]interface{}
			if json.Unmarshal(bytes, &m) == nil {
				if t, ok := m["type"].(string); ok {
					eventType = t
				}
			}
		}
	}

	// Write event in Anthropic SSE format
	// event: <type>
	// data: <json>
	//
	_, err = fmt.Fprintf(reqCtx.Writer, "event: %s\ndata: %s\n\n", eventType, string(data))
	if err != nil {
		return err
	}

	reqCtx.Writer.Flush()
	return nil
}

// writeErrorResponse writes an Anthropic-formatted error response
func (h *MessagesHandler) writeErrorResponse(reqCtx *gin.Context, statusCode int, errorType, message string) error {
	reqCtx.JSON(statusCode, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errorType,
			"message": message,
		},
	})
	return fmt.Errorf("%s: %s", errorType, message)
}

// storeToConversation stores the request and response to the conversation
func (h *MessagesHandler) storeToConversation(
	ctx context.Context,
	conv *conversation.Conversation,
	req messagesrequests.AnthropicMessagesRequest,
	resp *openai.ChatCompletionResponse,
) {
	if conv == nil || resp == nil || len(resp.Choices) == 0 {
		return
	}

	// Store user message
	userContent := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userContent = msg.Content.GetText()
		}
	}

	if userContent != "" {
		askItemID, _ := idgen.GenerateSecureID("msg", 16)
		userRole := conversation.ItemRoleUser
		completedStatus := conversation.ItemStatusCompleted
		askItem := conversation.Item{
			PublicID: askItemID,
			Type:     conversation.ItemTypeMessage,
			Role:     &userRole,
			Content:  []conversation.Content{conversation.NewTextContent(userContent)},
			Status:   &completedStatus,
		}
		conv.Items = append(conv.Items, askItem)
	}

	// Store assistant response
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		completionItemID, _ := idgen.GenerateSecureID("msg", 16)
		content := resp.Choices[0].Message.Content
		assistantRole := conversation.ItemRoleAssistant
		completedStatus := conversation.ItemStatusCompleted
		completionItem := conversation.Item{
			PublicID: completionItemID,
			Type:     conversation.ItemTypeMessage,
			Role:     &assistantRole,
			Content:  []conversation.Content{conversation.NewTextContent(content)},
			Status:   &completedStatus,
		}
		conv.Items = append(conv.Items, completionItem)
	}

	// Save conversation (best effort)
	if h.conversationService != nil {
		_, _ = h.conversationService.UpdateConversation(ctx, conv)
	}
}

// CountTokens handles POST /v1/messages/count_tokens
func (h *MessagesHandler) CountTokens(ctx context.Context, reqCtx *gin.Context, userID uint, request messagesrequests.AnthropicCountTokensRequest) error {
	// Start OpenTelemetry span
	ctx, span := observability.StartSpan(ctx, "llm-api", "MessagesHandler.CountTokens")
	defer span.End()

	observability.AddSpanAttributes(ctx,
		attribute.String("anthropic.model", request.Model),
		attribute.Int("anthropic.message_count", len(request.Messages)),
		attribute.Int("user.id", int(userID)),
	)

	// Get provider for the requested model to validate it exists
	selectedProviderModel, _, err := h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
	if err != nil || selectedProviderModel == nil {
		return h.writeErrorResponse(reqCtx, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Model not found: %s", request.Model))
	}

	// Convert to OpenAI format to estimate tokens
	messagesRequest := messagesrequests.AnthropicMessagesRequest{
		Model:    request.Model,
		Messages: request.Messages,
		System:   request.System,
		Tools:    request.Tools,
	}
	openaiRequest := ConvertAnthropicToOpenAI(&messagesRequest)

	// Estimate token count
	// This is a simplified estimation - in production, you'd want to use
	// the actual tokenizer for the model
	tokenCount := h.estimateTokenCount(openaiRequest)

	response := messagesresponses.AnthropicCountTokensResponse{
		InputTokens: tokenCount,
	}

	observability.AddSpanAttributes(ctx,
		attribute.Int("anthropic.input_tokens", tokenCount),
	)
	observability.SetSpanStatus(ctx, codes.Ok, "token count successful")

	reqCtx.JSON(http.StatusOK, response)
	return nil
}

// estimateTokenCount provides a rough token count estimation
// This uses a simple heuristic - ~4 characters per token for English text
func (h *MessagesHandler) estimateTokenCount(request openai.ChatCompletionRequest) int {
	totalChars := 0

	for _, msg := range request.Messages {
		// Count role (approximately)
		totalChars += len(msg.Role) * 2 // role tokens are usually counted with overhead

		// Count content
		if msg.Content != "" {
			totalChars += len(msg.Content)
		}

		// Count multi-content
		for _, part := range msg.MultiContent {
			if part.Type == openai.ChatMessagePartTypeText {
				totalChars += len(part.Text)
			}
			// Images are typically counted as fixed token amounts
			if part.Type == openai.ChatMessagePartTypeImageURL {
				totalChars += 1000 // Rough estimate for image tokens
			}
		}
	}

	// Count tools if any
	for _, tool := range request.Tools {
		if tool.Function != nil {
			totalChars += len(tool.Function.Name)
			totalChars += len(tool.Function.Description)
			if params, ok := tool.Function.Parameters.(map[string]interface{}); ok {
				if paramJSON, err := json.Marshal(params); err == nil {
					totalChars += len(paramJSON)
				}
			}
		}
	}

	// Rough estimation: ~4 characters per token, with some overhead
	estimatedTokens := (totalChars / 4) + 10 // +10 for message formatting overhead

	return estimatedTokens
}
