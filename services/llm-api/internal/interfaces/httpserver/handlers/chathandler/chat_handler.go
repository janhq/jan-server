package chathandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/conversation"
	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/prompt"
	"jan-server/services/llm-api/internal/domain/usersettings"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	memclient "jan-server/services/llm-api/internal/infrastructure/memory"
	"jan-server/services/llm-api/internal/infrastructure/metrics"
	"jan-server/services/llm-api/internal/infrastructure/observability"
	conversationHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	modelHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	chatrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/chat"
	"jan-server/services/llm-api/internal/utils/httpclients/chat"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
	"jan-server/services/llm-api/internal/utils/stringutils"

	"github.com/shopspring/decimal"
)

const ConversationReferrerContextKey = "conversation_referrer"

// ChatCompletionResult wraps the response with conversation context
type ChatCompletionResult struct {
	Response          *openai.ChatCompletionResponse
	ConversationID    string
	ConversationTitle *string
	Trimmed           bool // True if messages were trimmed to fit context
}

// ChatHandler handles chat completion requests
type ChatHandler struct {
	inferenceProvider   *inference.InferenceProvider
	providerHandler     *modelHandler.ProviderHandler
	conversationHandler *conversationHandler.ConversationHandler
	conversationService *conversation.ConversationService
	projectService      *project.ProjectService
	promptProcessor     *prompt.ProcessorImpl
	memoryHandler       *MemoryHandler
	userSettingsService *usersettings.Service
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	inferenceProvider *inference.InferenceProvider,
	providerHandler *modelHandler.ProviderHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	conversationService *conversation.ConversationService,
	projectService *project.ProjectService,
	promptProcessor *prompt.ProcessorImpl,
	memoryHandler *MemoryHandler,
	userSettingsService *usersettings.Service,
) *ChatHandler {
	return &ChatHandler{
		inferenceProvider:   inferenceProvider,
		providerHandler:     providerHandler,
		conversationHandler: conversationHandler,
		conversationService: conversationService,
		projectService:      projectService,
		promptProcessor:     promptProcessor,
		memoryHandler:       memoryHandler,
		userSettingsService: userSettingsService,
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
	var projectInstruction string
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

		// Prepend conversation items to messages
		conversationID = conv.PublicID
		observability.AddSpanAttributes(ctx,
			attribute.String("conversation.id", conversationID),
		)
		request.Messages = h.prependConversationItems(conv, request.Messages)

		// Load project instruction for this conversation (if any)
		projectInstruction = h.getProjectInstruction(ctx, userID, conv)
	}
	// If no conversation.id exists, bypass as non-conversation completion

	// Validate messages (after prepending conversation items)
	if len(request.Messages) == 0 {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "messages cannot be empty", nil, "c9d0e1f2-a3b4-4c5d-6e7f-8a9b0c1d2e3f")
		observability.RecordError(ctx, err)
		return nil, err
	}

	// Load memory context (best-effort) when a conversation is present
	loadedMemory := h.collectPromptMemory(conv, reqCtx)

	// Load user settings once for prompt orchestration and m	emory (best-effort)
	var userSettings *usersettings.UserSettings
	if h.userSettingsService != nil {
		userSettings, err = h.userSettingsService.GetOrCreateSettings(ctx, userID)
		if err != nil {
			userSettings = nil
		}
	}

	// Load memory using memory_handler (respects MEMORY_ENABLED and user settings)
	// Memory injection is controlled by PROMPT_ORCHESTRATION_MEMORY in the prompt processor
	if h.memoryHandler != nil && conversationID != "" {
		memoryContext, memErr := h.memoryHandler.LoadMemoryContext(ctx, userID, conversationID, conv, newMessages, userSettings)
		if memErr == nil && len(memoryContext) > 0 {
			loadedMemory = append(loadedMemory, memoryContext...)
		}
	}

	// Get provider based on the requested model
	observability.AddSpanEvent(ctx, "selecting_provider")
	selectedProviderModel, selectedProvider, err := h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to select provider model")
	}

	if selectedProviderModel == nil {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, fmt.Sprintf("model not found: %s", request.Model), nil, "d0e1f2a3-b4c5-4d6e-7f8a-9b0c1d2e3f4a")
		observability.RecordError(ctx, err)
		return nil, err
	}

	if selectedProvider == nil {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider not found", nil, "e1f2a3b4-c5d6-4e7f-8a9b-0c1d2e3f4a5b")
		observability.RecordError(ctx, err)
		return nil, err
	}

	imageRequested := request.Image != nil && *request.Image

	if imageRequested && request.EnableThinking != nil && !*request.EnableThinking {
		observability.AddSpanEvent(ctx, "ignoring_enable_thinking_for_image_request")
		request.EnableThinking = nil
	}

	// Check if we should use the instruct model instead
	// This happens when enable_thinking is explicitly false and the model has an instruct model configured
	if request.EnableThinking != nil && !*request.EnableThinking && selectedProviderModel.InstructModelID != nil && !imageRequested {
		instructModel, instructProvider, err := h.providerHandler.GetProviderModelByID(ctx, *selectedProviderModel.InstructModelID)
		if err == nil && instructModel != nil && instructProvider != nil {
			observability.AddSpanEvent(ctx, "switching_to_instruct_model",
				attribute.String("original_model", selectedProviderModel.ModelPublicID),
				attribute.String("instruct_model", instructModel.ModelPublicID),
			)
			selectedProviderModel = instructModel
			selectedProvider = instructProvider
		}
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

	// Optionally load model catalog (used later to apply default parameters)
	var modelCatalog *domainmodel.ModelCatalog
	if selectedProviderModel.ModelCatalogID != nil {
		modelCatalog, err = h.providerHandler.GetModelCatalogByID(ctx, *selectedProviderModel.ModelCatalogID)
		if err != nil {
			// Ignore error, model catalog is optional
		}
	}

	// Ensure project instruction is the first system message when available
	if projectInstruction != "" {
		request.Messages = prompt.PrependProjectInstruction(request.Messages, projectInstruction)
	}

	// Apply prompt orchestration (if enabled)
	if h.promptProcessor != nil {
		observability.AddSpanEvent(ctx, "processing_prompts")

		preferences := make(map[string]interface{})
		if len(request.Tools) > 0 || request.ToolChoice != nil {
			preferences["use_tools"] = true
		}
		if persona := strings.TrimSpace(reqCtx.GetHeader("X-Prompt-Persona")); persona != "" {
			preferences["persona"] = persona
		}
		if persona := strings.TrimSpace(reqCtx.Query("persona")); persona != "" {
			preferences["persona"] = persona
		}

		// Pass deep_research flag to prompt orchestration
		if request.DeepResearch != nil && *request.DeepResearch {
			preferences["deep_research"] = true
			observability.AddSpanAttributes(ctx, attribute.Bool("chat.deep_research", true))
		}

		// Pass image flag to prompt orchestration for image generation tools
		if request.Image != nil && *request.Image {
			preferences["image"] = true
			observability.AddSpanAttributes(ctx, attribute.Bool("chat.image", true))
		}

		var profileSettings *usersettings.ProfileSettings
		if userSettings != nil {
			profileSettings = &userSettings.ProfileSettings
		}

		// Get model catalog ID for model-specific template resolution
		var modelCatalogID *string
		if modelCatalog != nil && modelCatalog.PublicID != "" {
			modelCatalogID = &modelCatalog.PublicID
		}

		promptCtx := &prompt.Context{
			UserID:             userID,
			ConversationID:     conversationID,
			Language:           strings.TrimSpace(reqCtx.GetHeader("Accept-Language")),
			Preferences:        preferences,
			Memory:             loadedMemory,
			ProjectInstruction: projectInstruction,
			Profile:            profileSettings,
			ModelCatalogID:     modelCatalogID,
			Tools:              request.Tools,
		}

		processedMessages, processErr := h.promptProcessor.Process(ctx, promptCtx, request.Messages)
		if processErr != nil {
			// Continue with original messages
		} else {
			request.Messages = processedMessages
			if len(promptCtx.AppliedModules) > 0 {
				reqCtx.Header("X-Applied-Prompt-Modules", strings.Join(promptCtx.AppliedModules, ","))
			}
			observability.AddSpanEvent(ctx, "prompts_processed")
		}
	}

	// Get chat completion client
	chatClient, err := h.inferenceProvider.GetChatCompletionClient(ctx, selectedProvider)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create chat client")
	}

	// Build token budget for context management
	contextLength := DefaultContextLength
	if modelCatalog != nil && modelCatalog.ContextLength != nil && *modelCatalog.ContextLength > 0 {
		contextLength = *modelCatalog.ContextLength
	}

	// Validate user input size BEFORE any processing
	// This returns an error if the current user input exceeds MaxUserContentTokens
	if err := ValidateUserInputSize(request.Messages); err != nil {
		observability.RecordError(ctx, err)
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, err.Error(), nil, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	// Get max_tokens from request (0 if not set)
	maxCompletionTokens := 0
	if request.MaxTokens > 0 {
		maxCompletionTokens = request.MaxTokens
	}

	// Track whether any trimming occurred
	wasTrimmed := false

	// Build and validate token budget
	budget := BuildTokenBudget(contextLength, request.Tools, maxCompletionTokens)
	if err := budget.Validate(); err != nil {
		// Fall back to legacy trimming if budget validation fails
		trimResult := TrimMessagesToFitContext(request.Messages, contextLength)
		if trimResult.TrimmedCount > 0 {
			wasTrimmed = true
			observability.AddSpanEvent(ctx, "messages_trimmed",
				attribute.Int("trimmed_count", trimResult.TrimmedCount),
				attribute.Int("estimated_tokens", trimResult.EstimatedTokens),
				attribute.Int("context_length", contextLength),
			)
			request.Messages = trimResult.Messages
		}
	} else {
		// First, truncate oversized user content in HISTORICAL messages (not current input)
		userTruncatedMessages, userTruncEvents := TruncateLargeUserContent(request.Messages)
		if len(userTruncEvents) > 0 {
			wasTrimmed = true
			observability.AddSpanEvent(ctx, "user_content_truncated",
				attribute.Int("truncation_count", len(userTruncEvents)),
			)
			request.Messages = userTruncatedMessages
		}

		// Second, truncate oversized tool content (with JSON-aware parsing)
		truncatedMessages, truncEvents := TruncateLargeToolContent(request.Messages)
		if len(truncEvents) > 0 {
			wasTrimmed = true
			observability.AddSpanEvent(ctx, "tool_content_truncated",
				attribute.Int("truncation_count", len(truncEvents)),
			)
			request.Messages = truncatedMessages
		}

		// Third, limit images to prevent context overflow from multimodal tokens
		// Tool messages: max 10 images, User messages: max 15 images
		request.Messages = LimitImagesInMessages(request.Messages)

		// Then trim messages using the validated budget (oldest items first)
		trimResult := TrimMessagesToFitBudget(request.Messages, budget)
		if trimResult.TrimmedCount > 0 {
			wasTrimmed = true
			observability.AddSpanEvent(ctx, "messages_trimmed",
				attribute.Int("trimmed_count", trimResult.TrimmedCount),
				attribute.Int("estimated_tokens", trimResult.EstimatedTokens),
				attribute.Int("context_length", contextLength),
				attribute.Int("tools_tokens", budget.ToolsTokens),
			)
			request.Messages = trimResult.Messages
		}
	}

	var response *openai.ChatCompletionResponse

	// Handle streaming vs non-streaming
	llmRequest := chat.CompletionRequest{
		ChatCompletionRequest: request.ChatCompletionRequest,
		TopK:                  request.TopK,
		RepetitionPenalty:     request.RepetitionPenalty,
	}
	if modelCatalog != nil {
		h.applyModelDefaultsFromCatalog(&llmRequest, modelCatalog)
	}

	observability.AddSpanEvent(ctx, "calling_llm")

	llmStartTime := time.Now()
	if request.Stream {
		response, err = h.streamCompletion(ctx, reqCtx, chatClient, conv, llmRequest)
	} else {
		response, err = h.callCompletion(ctx, chatClient, llmRequest)
	}
	llmDuration := time.Since(llmStartTime)

	if err != nil {
		observability.AddSpanEvent(ctx, "completion_fallback",
			attribute.String("error", err.Error()),
		)
		response = h.BuildFallbackResponse(request.Model)
		err = nil
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

		// Record Prometheus metrics for token usage and LLM duration
		metrics.RecordTokens(request.Model, selectedProvider.DisplayName, response.Usage.PromptTokens, response.Usage.CompletionTokens)
		metrics.RecordLLMDuration(request.Model, selectedProvider.DisplayName, request.Stream, llmDuration.Seconds())
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
		}
		if id, genErr := idgen.GenerateSecureID("msg", 16); genErr == nil {
			completionItemID = id
		}
		storeReasoning := false
		if request.StoreReasoning != nil {
			storeReasoning = *request.StoreReasoning
		}

		if err := h.addCompletionToConversation(ctx, conv, newMessages, response, askItemID, completionItemID, storeReasoning); err != nil {
			// Don't fail the request
			observability.AddSpanEvent(ctx, "conversation_storage_failed",
				attribute.String("error", err.Error()),
			)
		} else {
			observability.AddSpanAttributes(ctx,
				attribute.Bool("completion.stored", true),
			)

			// Observe conversation for memory extraction using memory_handler
			if h.memoryHandler != nil && response != nil && len(response.Choices) > 0 {
				finishReason := response.Choices[0].FinishReason
				observability.AddSpanEvent(ctx, "observing_for_memory",
					attribute.String("finish_reason", string(finishReason)),
				)
				go h.memoryHandler.ObserveConversation(conv, userID, newMessages, response, finishReason)
			}
		}
	}

	if conv != nil && response != nil {
		conv = h.updateConversationTitleFromCompletion(ctx, userID, conv, newMessages, response)
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
		Trimmed:           wasTrimmed,
	}, nil
}

// callCompletion handles non-streaming chat completion
func (h *ChatHandler) callCompletion(
	ctx context.Context,
	chatClient *chat.ChatCompletionClient,
	request chat.CompletionRequest,
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
	request chat.CompletionRequest,
) (*openai.ChatCompletionResponse, error) {
	// Stream completion response to context with callback
	resp, err := chatClient.StreamChatCompletionToContextWithCallback(reqCtx, "", request, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "streaming completion failed")
	}

	return resp, nil
}

// BuildFallbackResponse constructs a minimal assistant reply when upstream completion fails.
func (h *ChatHandler) BuildFallbackResponse(model string) *openai.ChatCompletionResponse {
	now := time.Now().Unix()
	return &openai.ChatCompletionResponse{
		ID:      fmt.Sprintf("fallback_%d", now),
		Object:  "chat.completion",
		Created: now,
		Model:   model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "I'm having trouble reaching the model right now, but here's a fallback response.",
				},
				FinishReason: openai.FinishReasonStop,
			},
		},
	}
}

// applyModelDefaultsFromCatalog fills in missing request parameters using defaults from the model catalog.
func (h *ChatHandler) applyModelDefaultsFromCatalog(req *chat.CompletionRequest, catalog *domainmodel.ModelCatalog) {
	if req == nil || catalog == nil {
		return
	}

	defaults := catalog.SupportedParameters.Default
	if len(defaults) == 0 {
		return
	}

	if req.Temperature == 0 {
		if val, ok := decimalToFloat32(defaults["temperature"]); ok {
			req.Temperature = val
		}
	}
	if req.TopP == 0 {
		if val, ok := decimalToFloat32(defaults["top_p"]); ok {
			req.TopP = val
		}
	}
	if req.PresencePenalty == 0 {
		if val, ok := decimalToFloat32(defaults["presence_penalty"]); ok {
			req.PresencePenalty = val
		}
	}
	if req.FrequencyPenalty == 0 {
		if val, ok := decimalToFloat32(defaults["frequency_penalty"]); ok {
			req.FrequencyPenalty = val
		}
	}
	if req.MaxTokens == 0 {
		if val, ok := decimalToInt(defaults["max_tokens"]); ok {
			req.MaxTokens = val
		}
	}
	if req.TopK == nil || (req.TopK != nil && *req.TopK == 0) {
		if val, ok := decimalToInt(defaults["top_k"]); ok {
			req.TopK = &val
		}
	}
	if req.RepetitionPenalty == nil || (req.RepetitionPenalty != nil && *req.RepetitionPenalty == 0) {
		if val, ok := decimalToFloat32(defaults["repetition_penalty"]); ok {
			req.RepetitionPenalty = &val
		}
	}
}

func decimalToFloat32(val *decimal.Decimal) (float32, bool) {
	if val == nil {
		return 0, false
	}
	f, _ := val.Float64()
	return float32(f), true
}

func decimalToInt(val *decimal.Decimal) (int, bool) {
	if val == nil {
		return 0, false
	}
	return int(val.IntPart()), true
}

// getProjectInstruction loads the project instruction for the conversation, falling back to the stored snapshot.
func (h *ChatHandler) getProjectInstruction(ctx context.Context, userID uint, conv *conversation.Conversation) string {
	if conv == nil || h.projectService == nil {
		return ""
	}
	if ctx != nil && ctx.Err() != nil {
		return ""
	}

	if conv.EffectiveInstructionSnapshot != nil {
		if snapshot := strings.TrimSpace(*conv.EffectiveInstructionSnapshot); snapshot != "" {
			return snapshot
		}
	}

	if conv.ProjectPublicID == nil {
		return ""
	}

	projectID := strings.TrimSpace(*conv.ProjectPublicID)
	if projectID == "" {
		return ""
	}

	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectID, userID)
	if err != nil {
		return ""
	}

	if proj.Instruction == nil {
		return ""
	}

	return strings.TrimSpace(*proj.Instruction)
}

// collectPromptMemory gathers memory hints from request headers, conversation metadata, or recent turns.
func (h *ChatHandler) collectPromptMemory(conv *conversation.Conversation, reqCtx *gin.Context) []string {
	memory := make([]string, 0)

	if reqCtx != nil {
		if headerMemory := strings.TrimSpace(reqCtx.GetHeader("X-Prompt-Memory")); headerMemory != "" {
			for _, part := range strings.Split(headerMemory, ";") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					memory = append(memory, trimmed)
				}
			}
		}
	}

	if conv != nil {
		if conv.Metadata != nil {
			for key, val := range conv.Metadata {
				if strings.HasPrefix(strings.ToLower(key), "memory") && strings.TrimSpace(val) != "" {
					memory = append(memory, strings.TrimSpace(val))
				}
			}
		}

		if len(memory) == 0 {
			memory = append(memory, h.recentConversationMemory(conv)...)
		}
	}

	return memory
}

// recentConversationMemory builds lightweight context lines from the latest conversation turns.
func (h *ChatHandler) recentConversationMemory(conv *conversation.Conversation) []string {
	items := conv.GetActiveBranchItems()
	if len(items) == 0 {
		return nil
	}

	memories := make([]string, 0, 3)
	collected := 0
	for i := len(items) - 1; i >= 0 && collected < 3; i-- {
		text := firstTextFromItem(items[i])
		if text == "" {
			continue
		}
		role := "user"
		if items[i].Role != nil {
			role = string(*items[i].Role)
		}
		memories = append(memories, fmt.Sprintf("Recent %s message: %s", role, text))
		collected++
	}

	// Reverse to keep chronological order
	for i, j := 0, len(memories)-1; i < j; i, j = i+1, j-1 {
		memories[i], memories[j] = memories[j], memories[i]
	}

	return memories
}

func formatMemoryForPromptCtx(resp *memclient.LoadResponse) []string {
	if resp == nil {
		return nil
	}
	memory := make([]string, 0, len(resp.CoreMemory)+len(resp.SemanticMemory)+len(resp.EpisodicMemory))
	for _, item := range resp.CoreMemory {
		if strings.TrimSpace(item.Text) != "" {
			memory = append(memory, fmt.Sprintf("User memory: %s", item.Text))
		}
	}
	for _, fact := range resp.SemanticMemory {
		if strings.TrimSpace(fact.Text) != "" {
			if strings.TrimSpace(fact.Title) != "" {
				memory = append(memory, fmt.Sprintf("Project fact - %s: %s", fact.Title, fact.Text))
			} else {
				memory = append(memory, fmt.Sprintf("Project fact: %s", fact.Text))
			}
		}
	}
	for _, event := range resp.EpisodicMemory {
		if strings.TrimSpace(event.Text) != "" {
			memory = append(memory, fmt.Sprintf("Recent event: %s", event.Text))
		}
	}
	return memory
}

// formatAndFilterMemory formats memory response and filters based on user settings
func (h *ChatHandler) formatAndFilterMemory(resp *memclient.LoadResponse, settings *usersettings.UserSettings) []string {
	if resp == nil {
		return nil
	}

	memory := make([]string, 0)

	// Add core memory (user preferences) if enabled
	if settings.MemoryConfig.InjectUserCore {
		for _, item := range resp.CoreMemory {
			if strings.TrimSpace(item.Text) != "" {
				memory = append(memory, fmt.Sprintf("User memory: %s", item.Text))
			}
		}
	}

	// Add semantic memory (project facts) if enabled
	if settings.MemoryConfig.InjectSemantic {
		for _, fact := range resp.SemanticMemory {
			if strings.TrimSpace(fact.Text) != "" {
				if strings.TrimSpace(fact.Title) != "" {
					memory = append(memory, fmt.Sprintf("Project fact - %s: %s", fact.Title, fact.Text))
				} else {
					memory = append(memory, fmt.Sprintf("Project fact: %s", fact.Text))
				}
			}
		}
	}

	// Add episodic memory (conversation history) if enabled
	if settings.MemoryConfig.InjectEpisodic {
		for _, event := range resp.EpisodicMemory {
			if strings.TrimSpace(event.Text) != "" {
				memory = append(memory, fmt.Sprintf("Recent event: %s", event.Text))
			}
		}
	}

	return memory
}

func firstTextFromItem(item conversation.Item) string {
	for _, content := range item.Content {
		if content.TextString != nil {
			if trimmed := strings.TrimSpace(*content.TextString); trimmed != "" {
				return trimmed
			}
		}
		if content.Text != nil {
			if trimmed := strings.TrimSpace(content.Text.Text); trimmed != "" {
				return trimmed
			}
		}
		if content.OutputText != nil {
			if trimmed := strings.TrimSpace(content.OutputText.Text); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func (h *ChatHandler) createConversationWithReferrer(ctx context.Context, userID uint, referrer string) (*conversation.Conversation, error) {
	cleaned := strings.TrimSpace(referrer)
	if cleaned == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "referrer cannot be empty", nil, "f2a3b4c5-d6e7-4f8a-9b0c-1d2e3f4a5b6c")
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
	maxLen := conversationTitleMaxLength()
	// Find the first user message
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			title := stringutils.GenerateTitle(msg.Content, maxLen)
			if title != "" {
				return title
			}
		}
	}
	return "New Conversation"
}

func (h *ChatHandler) generateTitleFromMessages(ctx context.Context, messages []openai.ChatCompletionMessage) string {
	cfg := config.GetGlobal()
	if cfg == nil {
		return h.generateTitleFromMessage(messages)
	}

	if cfg != nil && cfg.ConversationTitleGenerationEnabled {
		maxLen := conversationTitleMaxLength()
		if title, err := h.generateTitleWithModel(ctx, cfg.ConversationTitleGenerationModelID, messages, maxLen); err == nil && title != "" {
			return title
		} else if err != nil {
		}
	}

	return h.generateTitleFromMessage(messages)
}

// updateConversationTitleFromMessages updates conversation title if it's still default and returns the updated conversation
func (h *ChatHandler) updateConversationTitleFromMessages(ctx context.Context, userID uint, conv *conversation.Conversation, messages []openai.ChatCompletionMessage) *conversation.Conversation {
	if conv == nil {
		return nil
	}

	// Only update if title is not set or is empty
	if conv.Title == nil || *conv.Title == "" {
		newTitle := h.generateTitleFromMessages(ctx, messages)
		if newTitle != "" {
			// Update the conversation title
			titleCopy := newTitle
			updateInput := conversation.UpdateConversationInput{
				Title: &titleCopy,
			}
			updatedConv, err := h.conversationService.UpdateConversationWithInput(ctx, userID, conv.PublicID, updateInput)
			if err != nil {
				// Don't fail the request
				return conv
			}
			return updatedConv
		}
	}
	return conv
}

func (h *ChatHandler) updateConversationTitleFromCompletion(ctx context.Context, userID uint, conv *conversation.Conversation, messages []openai.ChatCompletionMessage, response *openai.ChatCompletionResponse) *conversation.Conversation {
	if conv == nil || response == nil || len(response.Choices) == 0 {
		return conv
	}
	userMessageCount := countUserMessages(messages)
	if !h.shouldUpdateTitleForUserMessageCount(messages) {
		return conv
	}
	if isTitleLocked(conv) {
		return conv
	}
	if conv.Title != nil && strings.TrimSpace(*conv.Title) != "" {
		currentTitle := strings.TrimSpace(*conv.Title)
		defaultTitle := strings.TrimSpace(h.generateTitleFromMessage(messages))
		if userMessageCount == 1 && (defaultTitle == "" || !strings.EqualFold(currentTitle, defaultTitle)) {
			return conv
		}
	}

	combined := append([]openai.ChatCompletionMessage{}, messages...)
	combined = append(combined, response.Choices[0].Message)
	newTitle := h.generateTitleFromMessages(ctx, combined)
	if newTitle == "" {
		return conv
	}

	titleCopy := newTitle
	updateInput := conversation.UpdateConversationInput{
		Title: &titleCopy,
	}
	updatedConv, err := h.conversationService.UpdateConversationWithInput(ctx, userID, conv.PublicID, updateInput)
	if err != nil {
		return conv
	}
	return updatedConv
}

func (h *ChatHandler) shouldUpdateTitleForUserMessageCount(messages []openai.ChatCompletionMessage) bool {
	count := countUserMessages(messages)
	if count == 1 || count%5 == 0 {
		return true
	}
	return false
}

func countUserMessages(messages []openai.ChatCompletionMessage) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == openai.ChatMessageRoleUser {
			count++
		}
	}
	return count
}

func isTitleLocked(conv *conversation.Conversation) bool {
	if conv == nil || conv.Metadata == nil {
		return false
	}
	value, ok := conv.Metadata["title_locked"]
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func (h *ChatHandler) generateTitleWithModel(ctx context.Context, modelPublicID string, messages []openai.ChatCompletionMessage, maxLen int) (string, error) {
	modelPublicID = strings.TrimSpace(modelPublicID)
	if modelPublicID == "" {
		return "", fmt.Errorf("title generation model id is empty")
	}

	selectedProviderModel, selectedProvider, err := h.providerHandler.SelectProviderModelForProviderOriginalModelIDIncludingInactive(ctx, modelPublicID)
	if err != nil {
		return "", err
	}

	if selectedProviderModel == nil || selectedProvider == nil {
		return "", fmt.Errorf("title generation model provider not found")
	}
	if !selectedProviderModel.Active {
	}

	chatClient, err := h.inferenceProvider.GetChatCompletionClient(ctx, selectedProvider)
	if err != nil {
		return "", err
	}

	var modelCatalog *domainmodel.ModelCatalog
	if selectedProviderModel.ModelCatalogID != nil {
		if catalog, catalogErr := h.providerHandler.GetModelCatalogByID(ctx, *selectedProviderModel.ModelCatalogID); catalogErr == nil {
			modelCatalog = catalog
		}
	}

	promptMessages := buildConversationTitlePromptMessages(messages, maxLen)
	llmRequest := chat.CompletionRequest{
		ChatCompletionRequest: openai.ChatCompletionRequest{
			Model:       selectedProviderModel.ProviderOriginalModelID,
			Messages:    promptMessages,
			MaxTokens:   64,
			Temperature: 0.2,
		},
	}
	if modelCatalog != nil {
		h.applyModelDefaultsFromCatalog(&llmRequest, modelCatalog)
	}

	response, err := chatClient.CreateChatCompletion(ctx, "", llmRequest)
	if err != nil {
		return "", err
	}
	if response == nil || len(response.Choices) == 0 {
		return "", fmt.Errorf("empty title generation response")
	}

	rawTitle := strings.TrimSpace(response.Choices[0].Message.Content)
	if rawTitle == "" {
		return "", fmt.Errorf("empty title generation content")
	}

	title := stringutils.GenerateTitle(rawTitle, maxLen)
	if title == "" {
		return "", fmt.Errorf("title generation result is empty after sanitization")
	}
	return title, nil
}

func buildConversationTitlePromptMessages(messages []openai.ChatCompletionMessage, maxLen int) []openai.ChatCompletionMessage {
	systemPrompt := "You generate short, descriptive conversation titles. Return only the title text with no quotes or extra words."
	userPrompt := fmt.Sprintf(
		"Create a concise title for this conversation. Max length: %d characters.\nConversation:\n%s",
		maxLen,
		formatConversationForTitlePrompt(messages),
	)

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}
}

func formatConversationForTitlePrompt(messages []openai.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return "(no messages)"
	}

	var builder strings.Builder
	for i, msg := range messages {
		content := extractChatMessageText(msg)
		if strings.TrimSpace(content) == "" {
			content = "(no text content)"
		}
		builder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, msg.Role, content))
	}

	return strings.TrimSpace(builder.String())
}

func extractChatMessageText(msg openai.ChatCompletionMessage) string {
	if msg.Content != "" {
		return msg.Content
	}

	if len(msg.MultiContent) > 0 {
		var parts []string
		for _, part := range msg.MultiContent {
			if part.Type == openai.ChatMessagePartTypeText && strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	if msg.FunctionCall != nil && msg.FunctionCall.Name != "" {
		return fmt.Sprintf("Function call: %s", msg.FunctionCall.Name)
	}

	if len(msg.ToolCalls) > 0 {
		var names []string
		for _, call := range msg.ToolCalls {
			if call.Function.Name != "" {
				names = append(names, call.Function.Name)
			}
		}
		if len(names) > 0 {
			return fmt.Sprintf("Tool calls: %s", strings.Join(names, ", "))
		}
	}

	return ""
}

func conversationTitleMaxLength() int {
	return conversation.DefaultConversationValidationConfig().MaxTitleLength
}

func summarizeTitleMessages(messages []openai.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return "(no messages)"
	}

	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		text := extractChatMessageText(msg)
		preview := strings.TrimSpace(text)
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		if preview == "" {
			preview = "(no text)"
		}
		parts = append(parts, fmt.Sprintf("%s:%s", msg.Role, preview))
	}
	return strings.Join(parts, " | ")
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
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "conversation not found for this user")
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

	// Set tool_call_id for tool messages
	if role == conversation.ItemRoleTool && item.CallID != nil && *item.CallID != "" {
		msg.ToolCallID = *item.CallID
	}

	// Extract content from item - handle both text and multimodal content
	if len(item.Content) > 0 {
		hasMultiModal := false
		var textParts []string
		var multiContent []openai.ChatMessagePart

		for _, content := range item.Content {
			// Handle text content
			if content.TextString != nil && *content.TextString != "" {
				textParts = append(textParts, *content.TextString)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: *content.TextString,
				})
			} else if content.Text != nil && content.Text.Text != "" {
				textParts = append(textParts, content.Text.Text)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: content.Text.Text,
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

		// Use multimodal format if there are images or if it's a tool message with multiple parts
		// Tool messages should use MultiContent format when they have structured content
		if (hasMultiModal || (role == conversation.ItemRoleTool && len(multiContent) > 1)) && len(multiContent) > 0 {
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

	// Use conversation's active branch instead of hardcoded MAIN
	branchName := conv.ActiveBranch
	if branchName == "" {
		branchName = conversation.BranchMain
	}

	items := make([]conversation.Item, 0, 2)

	// Build the user input item
	userItem := h.buildInputConversationItem(newMessages, storeReasoning, askItemID)

	// Check if we should skip adding the user message (avoid duplicates after regenerate)
	// This happens when regenerate creates a branch with the user message, then frontend
	// triggers a new completion which would add the same user message again
	if userItem != nil {
		skipUserItem := false

		// Get the last item in the branch to check for duplicates
		existingItems, err := h.conversationService.GetConversationItems(ctx, conv, branchName, nil)
		if err == nil && len(existingItems) > 0 {
			lastItem := existingItems[len(existingItems)-1]
			// If the last item is a user message, check if it has the same content
			if lastItem.Role != nil && *lastItem.Role == conversation.ItemRoleUser {
				// Compare content - if it's the same, skip adding
				if h.isSameMessageContent(userItem, &lastItem) {
					skipUserItem = true
				}
			}
		}

		if !skipUserItem {
			items = append(items, *userItem)
		}
	}

	if item := h.buildAssistantConversationItem(response, storeReasoning, completionItemID); item != nil {
		items = append(items, *item)
	}

	// Create mcp_call items (with status in_progress) for each tool_call
	// These items will be updated by mcp-tools service via PATCH when execution completes
	if len(response.Choices) > 0 && len(response.Choices[0].Message.ToolCalls) > 0 {
		for _, toolCall := range response.Choices[0].Message.ToolCalls {
			mcpItems := h.buildMCPCallItems(toolCall)
			items = append(items, mcpItems...)
		}
	}

	if len(items) == 0 {
		return nil
	}

	if _, err := h.conversationService.AddItemsToConversation(ctx, conv, branchName, items); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to add items to conversation")
	}

	return nil
}

// isSameMessageContent checks if two items have the same text content
// Used to detect duplicate user messages after regenerate
func (h *ChatHandler) isSameMessageContent(newItem *conversation.Item, existingItem *conversation.Item) bool {
	if newItem == nil || existingItem == nil {
		return false
	}

	// Extract text content from both items
	newText := extractTextFromContent(newItem.Content)
	existingText := extractTextFromContent(existingItem.Content)

	// Compare normalized text (trim whitespace)
	return strings.TrimSpace(newText) == strings.TrimSpace(existingText)
}

// extractTextFromContent extracts the text content from a slice of Content
func extractTextFromContent(contents []conversation.Content) string {
	for _, c := range contents {
		if c.TextString != nil && *c.TextString != "" {
			return *c.TextString
		}
		if c.Text != nil && c.Text.Text != "" {
			return c.Text.Text
		}
		if c.OutputText != nil && c.OutputText.Text != "" {
			return c.OutputText.Text
		}
	}
	return ""
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

// buildMCPCallItems creates a single mcp_call item with status in_progress
// The item will be updated by mcp-tools service via PATCH when execution completes
func (h *ChatHandler) buildMCPCallItems(toolCall openai.ToolCall) []conversation.Item {
	if toolCall.ID == "" {
		return nil
	}

	callID := toolCall.ID
	args := toolCall.Function.Arguments
	toolName := toolCall.Function.Name
	serverLabel := "Jan MCP Server"
	now := time.Now().UTC()

	// Single mcp_call item with status in_progress (waiting for tool execution)
	inProgressStatus := conversation.ItemStatusInProgress
	toolRole := conversation.ItemRoleTool
	mcpCallItem := conversation.Item{
		Object:      "conversation.item",
		Type:        conversation.ItemTypeMcpCall,
		Role:        &toolRole,
		Status:      &inProgressStatus,
		CallID:      &callID,
		Name:        &toolName,
		Arguments:   &args,
		ServerLabel: &serverLabel,
		Content: []conversation.Content{
			{
				Type: "mcp_call",
				ToolCalls: []conversation.ToolCall{
					{
						ID:   toolCall.ID,
						Type: string(toolCall.Type),
						Function: conversation.FunctionCall{
							Name:      toolName,
							Arguments: args,
						},
					},
				},
			},
		},
		CreatedAt: now,
	}

	// Return only ONE item (not two)
	return []conversation.Item{mcpCallItem}
}

func (h *ChatHandler) filterReasoningContent(contents []conversation.Content, storeReasoning bool) []conversation.Content {
	if storeReasoning || len(contents) == 0 {
		return contents
	}

	filtered := make([]conversation.Content, 0, len(contents))
	for _, content := range contents {
		if strings.EqualFold(content.Type, "reasoning_text") {
			continue
		}
		filtered = append(filtered, content)
	}
	// If everything was reasoning, keep one entry so the assistant turn still gets persisted.
	if len(filtered) == 0 && len(contents) > 0 {
		filtered = append(filtered, contents[0])
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
			// For tool messages, use tool_result type
			contents = append(contents, conversation.Content{
				Type:       "tool_result",
				TextString: &msg.Content,
			})
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
						// For tool messages, use tool_result type
						contents = append(contents, conversation.Content{
							Type:       "tool_result",
							TextString: &part.Text,
						})
					default:
						contents = append(contents, conversation.NewTextContent(part.Text))
					}
				}
			case openai.ChatMessagePartTypeImageURL:
				if part.ImageURL != nil && part.ImageURL.URL != "" {
					imageContent := conversation.NewImageContent(
						part.ImageURL.URL,
						"",
						string(part.ImageURL.Detail),
					)
					contents = append(contents, imageContent)
				}
			}
		}
	}

	if msg.ReasoningContent != "" {
		contents = append(contents, conversation.Content{
			Type:       "reasoning_text",
			TextString: &msg.ReasoningContent,
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

	if len(contents) > 0 {
		item.Content = contents
	}

	// Store tool_call_id for tool messages
	if role == conversation.ItemRoleTool && msg.ToolCallID != "" {
		item.CallID = &msg.ToolCallID
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
