package chat

import (
	"jan-server/services/llm-api/interfaces/httpserver/handlers/chathandler"

	"github.com/gin-gonic/gin"
)

// ChatRoute handles chat completion endpoints
type ChatRoute struct {
	chatHandler *chathandler.ChatHandler
}

// NewChatRoute creates a new ChatRoute
func NewChatRoute(chatHandler *chathandler.ChatHandler) *ChatRoute {
	return &ChatRoute{
		chatHandler: chatHandler,
	}
}

// RegisterRouter registers chat routes
func (r *ChatRoute) RegisterRouter(router gin.IRouter) {
	// Chat completions endpoints
	router.POST("/chat/completions", r.ChatCompletions)
	router.POST("/completions", r.ChatCompletions) // Legacy endpoint
}

// ChatCompletions handles chat completion requests
// @Summary Create chat completion
// @Description Creates a model response for the given chat conversation
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object true "Chat completion request"
// @Success 200 {object} object "Chat completion response"
// @Failure 400 {object} object "Invalid request"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/chat/completions [post]
func (r *ChatRoute) ChatCompletions(c *gin.Context) {
	r.chatHandler.ChatCompletions(c)
}
