package chathandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/usersettings"
	"jan-server/mono/apps/backend/internal/infrastructure/logger"
	memclient "jan-server/mono/apps/backend/internal/infrastructure/memory"
	"jan-server/mono/apps/backend/internal/infrastructure/observability"
)

// MemoryHandler handles memory-related operations for chat conversations
type MemoryHandler struct {
	memoryClient        *memclient.Client
	memoryEnabled       bool // Application-level config
	userSettingsService *usersettings.Service
}

// NewMemoryHandler creates a new memory handler
func NewMemoryHandler(
	memoryClient *memclient.Client,
	memoryEnabled bool,
	userSettingsService *usersettings.Service,
) *MemoryHandler {
	return &MemoryHandler{
		memoryClient:        memoryClient,
		memoryEnabled:       memoryEnabled,
		userSettingsService: userSettingsService,
	}
}

// LoadMemoryContext loads memory for a conversation based on application config and user settings
// Returns memory array for prompt context, respecting both MEMORY_ENABLED and user settings.
// If settings are provided, they are reused; otherwise the handler fetches them.
func (m *MemoryHandler) LoadMemoryContext(
	ctx context.Context,
	userID string,
	conversationID string,
	conv *conversation.Conversation,
	messages []openai.ChatCompletionMessage,
	settings *usersettings.UserSettings,
) ([]string, error) {
	// Check application-level config first
	if !m.memoryEnabled || m.memoryClient == nil || conversationID == "" {
		return nil, nil
	}

	// Load user settings if not provided
	if settings == nil {
		var err error
		settings, err = m.userSettingsService.GetOrCreateSettings(ctx, userID)
		if err != nil {
			log := logger.GetLogger()
			log.Warn().Err(err).Str("user_id", userID).Msg("failed to load user settings, memory disabled")
			return nil, nil
		}
	}

	// Check user-level memory enabled flag
	if !settings.MemoryConfig.Enabled {
		return nil, nil
	}

	observability.AddSpanEvent(ctx, "loading_memories")
	observability.AddSpanAttributes(ctx,
		attribute.Bool("memory.app_enabled", m.memoryEnabled),
		attribute.Bool("memory.user_enabled", settings.MemoryConfig.Enabled),
		attribute.Bool("memory.inject_user_core", settings.MemoryConfig.InjectUserCore),
		attribute.Bool("memory.inject_semantic", settings.MemoryConfig.InjectSemantic),
		attribute.Bool("memory.inject_episodic", settings.MemoryConfig.InjectEpisodic),
		attribute.Int("memory.max_user_items", settings.MemoryConfig.MaxUserItems),
	)

	// Load memory from memory-tools service
	memoryResp, memErr := m.loadConversationMemory(ctx, userID, conversationID, conv, messages, settings)
	if memErr != nil {
		log := logger.GetLogger()
		log.Warn().Err(memErr).Str("conversation_id", conversationID).Msg("failed to load memories, continuing without memory")
		return nil, nil
	}

	if memoryResp == nil {
		return nil, nil
	}

	// Format and filter memory based on user settings
	loadedMemory := m.formatAndFilterMemory(memoryResp, settings)

	observability.AddSpanEvent(ctx, "memories_loaded",
		attribute.Int("core_memory_count", len(memoryResp.CoreMemory)),
		attribute.Int("episodic_memory_count", len(memoryResp.EpisodicMemory)),
		attribute.Int("semantic_memory_count", len(memoryResp.SemanticMemory)),
		attribute.Int("injected_memory_count", len(loadedMemory)),
	)

	return loadedMemory, nil
}

// ObserveConversation observes a conversation for memory extraction
// Respects both MEMORY_ENABLED and user settings for observation
func (m *MemoryHandler) ObserveConversation(
	conv *conversation.Conversation,
	userID string,
	messages []openai.ChatCompletionMessage,
	response *openai.ChatCompletionResponse,
	finishReason openai.FinishReason,
) {
	// Check application-level config first
	if !m.memoryEnabled || m.memoryClient == nil {
		return
	}

	ctx := context.Background()

	// Load user settings
	settings, err := m.userSettingsService.GetOrCreateSettings(ctx, userID)
	if err != nil {
		log := logger.GetLogger()
		log.Warn().Err(err).Str("user_id", userID).Msg("failed to load user settings for memory observation")
		return
	}

	// Check user-level memory enabled and observe enabled flags
	if !settings.MemoryConfig.Enabled || !settings.MemoryConfig.ObserveEnabled {
		return
	}

	// Only observe if completion finished with "stop" reason
	if finishReason != openai.FinishReasonStop {
		return
	}

	// Use a background context with timeout for async observation
	observeCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	// Build conversation items for observation
	conversationItems := buildMemoryConversationItems(messages, response)
	if len(conversationItems) == 0 {
		return
	}

	req := memclient.ObserveRequest{
		UserID:         userID,
		ConversationID: conv.PublicID,
		Messages:       conversationItems,
	}
	if conv.ProjectPublicID != nil {
		req.ProjectID = *conv.ProjectPublicID
	}

	if err := m.memoryClient.Observe(observeCtx, req); err != nil {
		log := logger.GetLogger()
		log.Warn().
			Err(err).
			Str("conversation_id", conv.PublicID).
			Str("user_id", userID).
			Msg("failed to observe conversation for memory extraction")
	}
}

// loadConversationMemory loads memory using the memory-tools service
func (m *MemoryHandler) loadConversationMemory(
	ctx context.Context,
	userID string,
	conversationID string,
	conv *conversation.Conversation,
	messages []openai.ChatCompletionMessage,
	settings *usersettings.UserSettings,
) (*memclient.LoadResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Use settings to configure memory load request
	maxUserItems := settings.MemoryConfig.MaxUserItems
	if maxUserItems <= 0 {
		maxUserItems = 3
	}
	maxProjectItems := settings.MemoryConfig.MaxProjectItems
	if maxProjectItems <= 0 {
		maxProjectItems = 5
	}
	maxEpisodicItems := settings.MemoryConfig.MaxEpisodicItems
	if maxEpisodicItems <= 0 {
		maxEpisodicItems = 3
	}
	minSimilarity := settings.MemoryConfig.MinSimilarity
	if minSimilarity <= 0 {
		minSimilarity = 0.5
	}

	req := memclient.LoadRequest{
		UserID:         userID,
		ConversationID: conversationID,
		Query:          extractQueryFromMessages(messages),
		Options: memclient.LoadOptions{
			MaxUserItems:     maxUserItems,
			MaxProjectItems:  maxProjectItems,
			MaxEpisodicItems: maxEpisodicItems,
			MinSimilarity:    minSimilarity,
		},
	}

	if conv != nil && conv.ProjectPublicID != nil {
		req.ProjectID = *conv.ProjectPublicID
	}

	return m.memoryClient.Load(ctx, req)
}

// formatAndFilterMemory formats memory items into strings based on user settings
func (m *MemoryHandler) formatAndFilterMemory(resp *memclient.LoadResponse, settings *usersettings.UserSettings) []string {
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

// extractQueryFromMessages extracts the last user message as the query
func extractQueryFromMessages(messages []openai.ChatCompletionMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openai.ChatMessageRoleUser && strings.TrimSpace(messages[i].Content) != "" {
			return messages[i].Content
		}
	}
	return ""
}

// buildMemoryConversationItems converts OpenAI messages to memory client format
func buildMemoryConversationItems(messages []openai.ChatCompletionMessage, response *openai.ChatCompletionResponse) []memclient.ConversationItem {
	items := make([]memclient.ConversationItem, 0, len(messages)+1)

	for _, msg := range messages {
		items = append(items, memclient.ConversationItem{
			Role:      string(msg.Role),
			Content:   msg.Content,
			CreatedAt: time.Now(),
		})
	}

	if response != nil && len(response.Choices) > 0 {
		items = append(items, memclient.ConversationItem{
			Role:      "assistant",
			Content:   response.Choices[0].Message.Content,
			CreatedAt: time.Now(),
		})
	}

	return items
}
