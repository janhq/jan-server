package chat

import (
	"github.com/gin-gonic/gin"
)

type ChatRoute struct {
	completionAPI *ChatCompletionRoute
}

func NewChatRoute(
	completionAPI *ChatCompletionRoute,
) *ChatRoute {
	return &ChatRoute{
		completionAPI: completionAPI,
	}
}

func (chatRoute *ChatRoute) RegisterRouter(router gin.IRouter) {
	chatRouter := router.Group("/chat")
	chatRoute.completionAPI.RegisterRouter(chatRouter)
}
