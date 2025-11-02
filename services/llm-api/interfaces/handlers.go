package interfaces

import (
	"jan-server/services/llm-api/handlers"
	"jan-server/services/llm-api/infrastructure/idempotency"
	"jan-server/services/llm-api/infrastructure/provider"
	"jan-server/services/llm-api/infrastructure/repo"

	"github.com/rs/zerolog"
)

// Handlers aggregates all HTTP handlers
type Handlers struct {
	Models        *handlers.ModelsHandler
	Chat          *handlers.ChatHandler
	Conversations *handlers.ConversationsHandler
}

// NewHandlers creates all HTTP handlers with their dependencies
func NewHandlers(
	registry *provider.Registry,
	idempotencyStore *idempotency.Store,
	modelRepo *repo.ModelRepository,
	conversationRepo *repo.ConversationRepository,
	messageRepo *repo.MessageRepository,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		Models:        handlers.NewModelsHandler(registry, modelRepo, logger),
		Chat:          handlers.NewChatHandler(registry, idempotencyStore, conversationRepo, logger),
		Conversations: handlers.NewConversationsHandler(conversationRepo, messageRepo, logger),
	}
}
