package chathandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/logger"
	"jan-server/services/llm-api/internal/infrastructure/mediaresolver"
	"jan-server/services/llm-api/internal/infrastructure/observability"
	conversationHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	modelHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	chatrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/chat"
	"jan-server/services/llm-api/internal/utils/httpclients/chat"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

const ConversationReferrerContextKey = "conversation_referrer"

// ChatCompletionResult wraps the response with conversation context
type ChatCompletionResult struct {
	Response          *openai.ChatCompletionResponse
	ConversationID    string
	ConversationTitle *string
}

// ChatHandler handles chat completion requests
type ChatHandler struct {
	inferenceProvider   *inference.InferenceProvider
	providerHandler     *modelHandler.ProviderHandler
	conversationHandler *conversationHandler.ConversationHandler
	conversationService *conversation.ConversationService
	mediaResolver       mediaresolver.Resolver
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	inferenceProvider *inference.InferenceProvider,
	providerHandler *modelHandler.ProviderHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	conversationService *conversation.ConversationService,
	mediaResolver mediaresolver.Resolver,
) *ChatHandler {
	return &ChatHandler{
		inferenceProvider:   inferenceProvider,
		providerHandler:     providerHandler,
		conversationHandler: conversationHandler,
		conversationService: conversationService,
		mediaResolver:       mediaResolver,
	}
}

// CreateChatCompletion handles chat completion requests (both streaming and non-streaming)
func (h *ChatHandler) CreateChatCompletion(
	ctx context.Context,
	reqCtx *gin.Context,
	userID uint,
	request chatrequests.ChatCompletionRequest,
) (*ChatCompletionResult, error) {
	// Start OpenTelemetry span for chat completion
	ctx, span := observability.StartSpan(ctx, "llm-api", "ChatHandler.CreateChatCompletion")
	defer span.End()

	// Track request start time for duration metrics
	startTime := time.Now()

	// Add basic attributes
	observability.AddSpanAttributes(ctx,
		attribute.String("chat.model", request.Model),
		attribute.Bool("chat.stream", request.Stream),
		attribute.Int("chat.message_count", len(request.Messages)),
		attribute.Int("user.id", int(userID)),
	)

	var conv *conversation.Conversation
	var conversationID string
	var err error
	newMessages := append([]openai.ChatCompletionMessage(nil), request.Messages...)

	// Extract referrer from context or query parameters
	referrer := strings.TrimSpace(reqCtx.GetString(ConversationReferrerContextKey))
	if referrer == "" {
		referrer = strings.TrimSpace(reqCtx.Param("referrer"))
	}
	if referrer == "" {
		referrer = strings.TrimSpace(reqCtx.Query("referrer"))
	}

	// Check if conversation.id exists in request
	if referrer != "" || (request.Conversation != nil && !request.Conversation.IsEmpty()) {
		observability.AddSpanEvent(ctx, "conversation_context_detected")

		// Get or create conversation with referrer (referrer can be empty)
		conv, err = h.getOrCreateConversation(ctx, userID, request.Conversation, referrer)
		if err != nil {
			observability.RecordError(ctx, err)
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get or create conversation")
		}

		// Auto-generate title from first message if conversation was just created
		conv = h.updateConversationTitleFromMessages(ctx, userID, conv, request.Messages)

		// Prepend conversation items to messages
		conversationID = conv.PublicID
		observability.AddSpanAttributes(ctx,
			attribute.String("conversation.id", conversationID),
		)
		request.Messages = h.prependConversationItems(conv, request.Messages)
	}
	// If no conversation.id exists, bypass as non-conversation completion

	// Validate messages (after prepending conversation items)
	if len(request.Messages) == 0 {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "messages cannot be empty", nil, "")
		observability.RecordError(ctx, err)
		return nil, err
	}

	// Get provider based on the requested model
	observability.AddSpanEvent(ctx, "selecting_provider")
	selectedProviderModel, selectedProvider, err := h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to select provider model")
	}

	if selectedProviderModel == nil {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, fmt.Sprintf("model not found: %s", request.Model), nil, "")
		observability.RecordError(ctx, err)
		return nil, err
	}

	if selectedProvider == nil {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider not found", nil, "")
		observability.RecordError(ctx, err)
		return nil, err
	}

	// Add provider information to span
	observability.AddSpanAttributes(ctx,
		attribute.String("provider.display_name", selectedProvider.DisplayName),
		attribute.String("provider.id", selectedProvider.PublicID),
		attribute.String("provider.kind", string(selectedProvider.Kind)),
		attribute.String("model.original_id", selectedProviderModel.ProviderOriginalModelID),
	)

	// Override the request model with the provider's original model ID
	request.Model = selectedProviderModel.ProviderOriginalModelID

	// Resolve jan_* media placeholders (best-effort)
	request.Messages = h.resolveMediaPlaceholders(ctx, reqCtx, request.Messages)

	// Get chat completion client
	chatClient, err := h.inferenceProvider.GetChatCompletionClient(ctx, selectedProvider)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create chat client")
	}

	var response *openai.ChatCompletionResponse

	// Handle streaming vs non-streaming
	observability.AddSpanEvent(ctx, "calling_llm")
	llmStartTime := time.Now()
	if request.Stream {
		response, err = h.streamCompletion(ctx, reqCtx, chatClient, conv, request.ChatCompletionRequest)
	} else {
		response, err = h.callCompletion(ctx, chatClient, request.ChatCompletionRequest)
	}
	llmDuration := time.Since(llmStartTime)

	if err != nil {
		observability.RecordError(ctx, err)
		observability.AddSpanAttributes(ctx,
			attribute.String("completion.status", "failed"),
		)
		return nil, err
	}

	// Add LLM response metrics
	if response != nil && response.Usage.TotalTokens > 0 {
		observability.AddSpanAttributes(ctx,
			attribute.Int("completion.prompt_tokens", response.Usage.PromptTokens),
			attribute.Int("completion.completion_tokens", response.Usage.CompletionTokens),
			attribute.Int("completion.total_tokens", response.Usage.TotalTokens),
			attribute.Float64("completion.llm_duration_ms", float64(llmDuration.Milliseconds())),
			attribute.String("completion.status", "success"),
		)
		if len(response.Choices) > 0 {
			observability.AddSpanAttributes(ctx,
				attribute.String("completion.finish_reason", string(response.Choices[0].FinishReason)),
			)
		}
	}

	// Add request and response to conversation if conversation context was provided
	storeConversation := true
	if request.Store != nil {
		storeConversation = *request.Store
	}

	if conv != nil && response != nil && storeConversation {
		observability.AddSpanEvent(ctx, "storing_conversation")
		var askItemID, completionItemID string
		if id, genErr := idgen.GenerateSecureID("msg", 16); genErr == nil {
			askItemID = id
		} else {
			log := logger.GetLogger()
			log.Warn().
				Err(genErr).
				Str("conversation_id", conv.PublicID).
				Msg("failed to generate ask item id")
		}
		if id, genErr := idgen.GenerateSecureID("msg", 16); genErr == nil {
			completionItemID = id
		} else {
			log := logger.GetLogger()
			log.Warn().
				Err(genErr).
				Str("conversation_id", conv.PublicID).
				Msg("failed to generate completion item id")
		}
		storeReasoning := false
		if request.StoreReasoning != nil {
			storeReasoning = *request.StoreReasoning
		}

		if err := h.addCompletionToConversation(ctx, conv, newMessages, response, askItemID, completionItemID, storeReasoning); err != nil {
			// Log error but don't fail the request
			log := logger.GetLogger()
			log.Warn().
				Err(err).
				Str("conversation_id", conv.PublicID).
				Msg("failed to store completion in conversation")
			observability.AddSpanEvent(ctx, "conversation_storage_failed",
				attribute.String("error", err.Error()),
			)
		} else {
			observability.AddSpanAttributes(ctx,
				attribute.Bool("completion.stored", true),
			)
		}
	}

	// Calculate total duration
	totalDuration := time.Since(startTime)
	observability.AddSpanAttributes(ctx,
		attribute.Float64("completion.total_duration_ms", float64(totalDuration.Milliseconds())),
	)

	// Set span status to OK
	observability.SetSpanStatus(ctx, codes.Ok, "chat completion successful")

	// Prepare conversation title for response
	var conversationTitle *string
	if conv != nil && conv.Title != nil {
		conversationTitle = conv.Title
	}

	return &ChatCompletionResult{
		Response:          response,
		ConversationID:    conversationID,
		ConversationTitle: conversationTitle,
	}, nil
}

// callCompletion handles non-streaming chat completion
func (h *ChatHandler) callCompletion(
	ctx context.Context,
	chatClient *chat.ChatCompletionClient,
	request openai.ChatCompletionRequest,
) (*openai.ChatCompletionResponse, error) {
	chatCompletion, err := chatClient.CreateChatCompletion(ctx, "", request)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "chat completion failed")
	}

	return chatCompletion, nil
}

// streamCompletion handles streaming chat completion
func (h *ChatHandler) streamCompletion(
	ctx context.Context,
	reqCtx *gin.Context,
	chatClient *chat.ChatCompletionClient,
	conv *conversation.Conversation,
	request openai.ChatCompletionRequest,
) (*openai.ChatCompletionResponse, error) {
	// Create callback to send conversation data before [DONE]
	var beforeDoneCallback chat.BeforeDoneCallback
	if conv != nil && conv.PublicID != "" {
		beforeDoneCallback = func(reqCtx *gin.Context) error {
			// Build conversation data with ID and title
			conversationData := map[string]interface{}{
				"id": conv.PublicID,
			}

			// Include title if available
			if conv.Title != nil && *conv.Title != "" {
				conversationData["title"] = *conv.Title
			}

			conversationChunk := map[string]interface{}{
				"conversation": conversationData,
				"created":      time.Now().Unix(),
				"id":           "", // Empty for conversation-only chunk
				"model":        request.Model,
				"object":       "chat.completion.chunk",
			}

			chunkJSON, err := json.Marshal(conversationChunk)
			if err != nil {
				return err
			}

			// Write conversation context as an SSE event BEFORE [DONE]
			return h.writeSSEData(reqCtx, string(chunkJSON))
		}
	}

	// Stream completion response to context with callback
	resp, err := chatClient.StreamChatCompletionToContextWithCallback(reqCtx, "", request, beforeDoneCallback)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "streaming completion failed")
	}

	return resp, nil
}

func (h *ChatHandler) resolveMediaPlaceholders(ctx context.Context, reqCtx *gin.Context, messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	if h.mediaResolver == nil || len(messages) == 0 {
		return messages
	}

	if reqCtx != nil {
		if authHeader := strings.TrimSpace(reqCtx.GetHeader("Authorization")); authHeader != "" {
			ctx = mediaresolver.ContextWithAuthorization(ctx, authHeader)
		}
		if principal, ok := middlewares.PrincipalFromContext(reqCtx); ok {
			ctx = mediaresolver.ContextWithPrincipal(ctx, principal)
		}
	}

	resolved, changed, err := h.mediaResolver.ResolveMessages(ctx, messages)
	if err != nil {
		log := logger.GetLogger()
		log.Warn().Err(err).Msg("media placeholder resolution failed")
		return messages
	}
	if changed {
		observability.AddSpanEvent(ctx, "media_placeholders_resolved")
		return resolved
	}
	return messages
}

func (h *ChatHandler) createConversationWithReferrer(ctx context.Context, userID uint, referrer string) (*conversation.Conversation, error) {
	cleaned := strings.TrimSpace(referrer)
	if cleaned == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "referrer cannot be empty", nil, "")
	}

	referrerCopy := cleaned
	input := conversation.CreateConversationInput{
		UserID:   userID,
		Referrer: &referrerCopy,
	}

	conv, err := h.conversationService.CreateConversationWithInput(ctx, input)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create conversation")
	}
	return conv, nil
}

// generateTitleFromMessage generates a conversation title from the first user message
func (h *ChatHandler) generateTitleFromMessage(messages []openai.ChatCompletionMessage) string {
	// Find the first user message
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			// Extract first 60 characters for title
			content := strings.TrimSpace(msg.Content)
			if len(content) > 60 {
				// Find a good breaking point (end of word)
				truncated := content[:60]
				if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 30 {
					content = content[:lastSpace] + "..."
				} else {
					content = truncated + "..."
				}
			}
			return content
		}
	}
	return "New Conversation"
}

// updateConversationTitleFromMessages updates conversation title if it's still default and returns the updated conversation
func (h *ChatHandler) updateConversationTitleFromMessages(ctx context.Context, userID uint, conv *conversation.Conversation, messages []openai.ChatCompletionMessage) *conversation.Conversation {
	if conv == nil {
		return nil
	}

	// Only update if title is not set or is empty
	if conv.Title == nil || *conv.Title == "" {
		newTitle := h.generateTitleFromMessage(messages)
		if newTitle != "" {
			// Update the conversation title
			titleCopy := newTitle
			updateInput := conversation.UpdateConversationInput{
				Title: &titleCopy,
			}
			updatedConv, err := h.conversationService.UpdateConversationWithInput(ctx, userID, conv.PublicID, updateInput)
			if err != nil {
				// Log but don't fail the request
				log := logger.GetLogger()
				log.Warn().
					Err(err).
					Str("conversation_id", conv.PublicID).
					Msg("failed to update conversation title")
				return conv
			}
			return updatedConv
		}
	}
	return conv
}

// getOrCreateConversation retrieves an existing conversation or creates a new one with optional referrer
func (h *ChatHandler) getOrCreateConversation(
	ctx context.Context,
	userID uint,
	convRef *chatrequests.ConversationReference,
	referrer string,
) (*conversation.Conversation, error) {
	// If a conversation ID was provided (either directly or from object), fetch it from the service
	if convRef != nil && convRef.GetID() != "" {
		conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, convRef.GetID(), userID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
		}

		// Return existing conversation with its original referrer
		// Note: Referrer is immutable after creation - it represents the conversation's origin
		return conv, nil
	}

	// If no ID was provided, create a new conversation
	if referrer != "" {
		return h.createConversationWithReferrer(ctx, userID, referrer)
	}

	// Create conversation without referrer
	input := conversation.CreateConversationInput{
		UserID: userID,
	}
	conv, err := h.conversationService.CreateConversationWithInput(ctx, input)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create conversation")
	}
	return conv, nil
}

// prependConversationItems prepends conversation items to the request messages
func (h *ChatHandler) prependConversationItems(
	conv *conversation.Conversation,
	messages []openai.ChatCompletionMessage,
) []openai.ChatCompletionMessage {
	if conv == nil {
		return messages
	}

	// Get items from the active branch or main branch
	var items []conversation.Item
	if conv.Branches != nil && conv.ActiveBranch != "" {
		items = conv.Branches[conv.ActiveBranch]
	} else {
		items = conv.Items
	}

	if len(items) == 0 {
		return messages
	}

	// Convert conversation items to chat messages
	conversationMessages := make([]openai.ChatCompletionMessage, 0, len(items))
	for _, item := range items {
		msg := h.itemToMessage(item)
		if msg != nil {
			conversationMessages = append(conversationMessages, *msg)
		}
	}

	// Prepend conversation messages to request messages
	return append(conversationMessages, messages...)
}

// itemToMessage converts a conversation item to a chat completion message
func (h *ChatHandler) itemToMessage(item conversation.Item) *openai.ChatCompletionMessage {
	// Skip items that aren't in completed status
	if item.Status != nil && *item.Status != conversation.ItemStatusCompleted {
		return nil
	}

	role := conversation.ItemRoleUser
	if item.Role != nil {
		role = *item.Role
	}

	msg := &openai.ChatCompletionMessage{
		Role: h.itemRoleToOpenAI(role),
	}

	// Extract content from item - handle both text and multimodal content
	if len(item.Content) > 0 {
		hasMultiModal := false
		var textParts []string
		var multiContent []openai.ChatMessagePart

		for _, content := range item.Content {
			// Handle text content
			if content.Text != nil && content.Text.Text != "" {
				textParts = append(textParts, content.Text.Text)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: content.Text.Text,
				})
			} else if content.InputText != nil {
				textParts = append(textParts, *content.InputText)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: *content.InputText,
				})
			} else if content.OutputText != nil {
				textParts = append(textParts, content.OutputText.Text)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: content.OutputText.Text,
				})
			}

			// Handle image content
			if content.Image != nil && content.Image.URL != "" {
				hasMultiModal = true
				imageURL := &openai.ChatMessageImageURL{
					URL: content.Image.URL,
				}
				if content.Image.Detail != "" {
					imageURL.Detail = openai.ImageURLDetail(content.Image.Detail)
				}
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type:     openai.ChatMessagePartTypeImageURL,
					ImageURL: imageURL,
				})
			}
		}

		// Use multimodal format if there are images, otherwise use simple string content
		if hasMultiModal && len(multiContent) > 0 {
			msg.MultiContent = multiContent
		} else if len(textParts) > 0 {
			msg.Content = textParts[0] // OpenAI typically uses single string content for text-only
		}
	}

	return msg
}

// itemRoleToOpenAI converts conversation item role to OpenAI chat message role
func (h *ChatHandler) itemRoleToOpenAI(role conversation.ItemRole) string {
	switch role {
	case conversation.ItemRoleSystem, conversation.ItemRoleDeveloper:
		return openai.ChatMessageRoleSystem
	case conversation.ItemRoleUser:
		return openai.ChatMessageRoleUser
	case conversation.ItemRoleAssistant:
		return openai.ChatMessageRoleAssistant
	case conversation.ItemRoleTool:
		return openai.ChatMessageRoleTool
	default:
		return openai.ChatMessageRoleUser // Default to user role
	}
}

// addCompletionToConversation persists the latest input and assistant response to the conversation
func (h *ChatHandler) addCompletionToConversation(
	ctx context.Context,
	conv *conversation.Conversation,
	newMessages []openai.ChatCompletionMessage,
	response *openai.ChatCompletionResponse,
	askItemID string,
	completionItemID string,
	storeReasoning bool,
) error {
	if conv == nil || response == nil || len(response.Choices) == 0 {
		return nil
	}

	items := make([]conversation.Item, 0, 2)

	if item := h.buildInputConversationItem(newMessages, storeReasoning, askItemID); item != nil {
		items = append(items, *item)
	}

	if item := h.buildAssistantConversationItem(response, storeReasoning, completionItemID); item != nil {
		items = append(items, *item)
	}

	if len(items) == 0 {
		return nil
	}

	if _, err := h.conversationService.AddItemsToConversation(ctx, conv, conversation.BranchMain, items); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to add items to conversation")
	}

	return nil
}

func (h *ChatHandler) buildInputConversationItem(
	messages []openai.ChatCompletionMessage,
	storeReasoning bool,
	publicID string,
) *conversation.Item {
	if len(messages) == 0 {
		return nil
	}

	msg := messages[len(messages)-1]
	item := h.messageToItem(msg)

	if item.Role != nil && *item.Role == conversation.ItemRoleSystem {
		return nil
	}

	item.Content = h.filterReasoningContent(item.Content, storeReasoning)
	if len(item.Content) == 0 && msg.Content == "" && len(msg.MultiContent) == 0 && msg.FunctionCall == nil && len(msg.ToolCalls) == 0 {
		return nil
	}

	if publicID != "" {
		item.PublicID = publicID
	}
	item.CreatedAt = time.Now().UTC()
	return &item
}

func (h *ChatHandler) buildAssistantConversationItem(
	response *openai.ChatCompletionResponse,
	storeReasoning bool,
	publicID string,
) *conversation.Item {
	if response == nil || len(response.Choices) == 0 {
		return nil
	}

	choice := response.Choices[0]
	item := h.messageToItem(choice.Message)
	item.Content = h.filterReasoningContent(item.Content, storeReasoning)

	if finishReason := string(choice.FinishReason); finishReason != "" && len(item.Content) > 0 {
		item.Content[0].FinishReason = &finishReason
	}

	if len(item.Content) == 0 && choice.Message.Content == "" && len(choice.Message.MultiContent) == 0 && choice.Message.FunctionCall == nil && len(choice.Message.ToolCalls) == 0 {
		return nil
	}

	if publicID != "" {
		item.PublicID = publicID
	}
	item.CreatedAt = time.Now().UTC()
	return &item
}

func (h *ChatHandler) filterReasoningContent(contents []conversation.Content, storeReasoning bool) []conversation.Content {
	if storeReasoning || len(contents) == 0 {
		return contents
	}

	filtered := make([]conversation.Content, 0, len(contents))
	for _, content := range contents {
		if strings.EqualFold(content.Type, "reasoning_content") {
			continue
		}
		filtered = append(filtered, content)
	}
	return filtered
}

// messageToItem converts a chat completion message to a conversation item
func (h *ChatHandler) messageToItem(msg openai.ChatCompletionMessage) conversation.Item {
	status := conversation.ItemStatusCompleted
	role := h.openAIRoleToItem(msg.Role)

	item := conversation.Item{
		Type:   conversation.ItemTypeMessage,
		Role:   &role,
		Status: &status,
	}

	contents := make([]conversation.Content, 0, 4)

	// Handle simple string content
	if msg.Content != "" {
		switch role {
		case conversation.ItemRoleUser:
			contents = append(contents, conversation.NewInputTextContent(msg.Content))
		case conversation.ItemRoleTool:
			toolContent := conversation.Content{
				Type: "tool_result",
				Text: &conversation.Text{
					Text: msg.Content,
				},
			}
			contents = append(contents, toolContent)
		default:
			contents = append(contents, conversation.NewTextContent(msg.Content))
		}
	}

	// Handle multimodal content (text + images)
	if len(msg.MultiContent) > 0 {
		for _, part := range msg.MultiContent {
			switch part.Type {
			case openai.ChatMessagePartTypeText:
				if part.Text != "" {
					switch role {
					case conversation.ItemRoleUser:
						contents = append(contents, conversation.NewInputTextContent(part.Text))
					case conversation.ItemRoleTool:
						toolContent := conversation.Content{
							Type: "tool_result",
							Text: &conversation.Text{
								Text: part.Text,
							},
						}
						contents = append(contents, toolContent)
					default:
						contents = append(contents, conversation.NewTextContent(part.Text))
					}
				}
			case openai.ChatMessagePartTypeImageURL:
				if part.ImageURL != nil && part.ImageURL.URL != "" {
					imageContent := conversation.NewImageContent(
						part.ImageURL.URL,
						"", // fileID - could be extracted from jan_* URLs if needed
						string(part.ImageURL.Detail),
					)
					contents = append(contents, imageContent)
				}
			}
		}
	}

	if msg.ReasoningContent != "" {
		reasoning := msg.ReasoningContent
		contents = append(contents, conversation.Content{
			Type:             "reasoning_content",
			ReasoningContent: &reasoning,
		})
	}

	if msg.FunctionCall != nil {
		functionCall := conversation.FunctionCall{
			Name:      msg.FunctionCall.Name,
			Arguments: msg.FunctionCall.Arguments,
		}

		contents = append(contents, conversation.Content{
			Type:         "function_call",
			FunctionCall: &functionCall,
		})
	}

	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]conversation.ToolCall, 0, len(msg.ToolCalls))
		for _, call := range msg.ToolCalls {
			toolCall := conversation.ToolCall{
				ID:   call.ID,
				Type: string(call.Type),
				Function: conversation.FunctionCall{
					Name:      call.Function.Name,
					Arguments: call.Function.Arguments,
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}

		contents = append(contents, conversation.Content{
			Type:      "tool_calls",
			ToolCalls: toolCalls,
		})
	}

	if msg.ToolCallID != "" {
		toolCallID := msg.ToolCallID
		attached := false
		for i := range contents {
			if contents[i].ToolCallID == nil {
				content := contents[i]
				content.ToolCallID = &toolCallID
				contents[i] = content
				attached = true
				break
			}
		}
		if !attached {
			contents = append(contents, conversation.Content{
				Type:       "tool_reference",
				ToolCallID: &toolCallID,
			})
		}
	}

	if len(contents) > 0 {
		item.Content = contents
	}

	return item
}

// openAIRoleToItem converts OpenAI chat message role to conversation item role
func (h *ChatHandler) openAIRoleToItem(role string) conversation.ItemRole {
	switch role {
	case openai.ChatMessageRoleSystem:
		return conversation.ItemRoleSystem
	case openai.ChatMessageRoleUser:
		return conversation.ItemRoleUser
	case openai.ChatMessageRoleAssistant:
		return conversation.ItemRoleAssistant
	case openai.ChatMessageRoleTool:
		return conversation.ItemRoleTool
	default:
		return conversation.ItemRoleUnknown
	}
}

// writeSSEData writes an SSE data event to the response
func (h *ChatHandler) writeSSEData(reqCtx *gin.Context, data string) error {
	_, err := reqCtx.Writer.Write([]byte("data: "))
	if err != nil {
		return err
	}
	_, err = reqCtx.Writer.Write([]byte(data))
	if err != nil {
		return err
	}
	_, err = reqCtx.Writer.Write([]byte("\n\n"))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}
