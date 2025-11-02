package conversation

import (
	"jan-server/services/llm-api/interfaces/httpserver/handlers/conversationhandler"

	"github.com/gin-gonic/gin"
)

// ConversationRoute handles conversation endpoints
type ConversationRoute struct {
	conversationsHandler *conversationhandler.ConversationsHandler
}

// NewConversationRoute creates a new ConversationRoute
func NewConversationRoute(conversationsHandler *conversationhandler.ConversationsHandler) *ConversationRoute {
	return &ConversationRoute{
		conversationsHandler: conversationsHandler,
	}
}

// RegisterRouter registers conversation routes
func (r *ConversationRoute) RegisterRouter(router gin.IRouter) {
	conversationsGroup := router.Group("/conversations")
	{
		conversationsGroup.GET("", r.ListConversations)
		conversationsGroup.POST("", r.CreateConversation)
		conversationsGroup.GET("/:conversation_id", r.GetConversation)
		conversationsGroup.POST("/:conversation_id/messages", r.AppendMessage)
		conversationsGroup.GET("/:conversation_id/messages", r.ListMessages)
		conversationsGroup.POST("/:conversation_id/runs", r.RunConversation)
		conversationsGroup.DELETE("/:conversation_id", r.DeleteConversation)
	}
}

// ListConversations lists all conversations
// @Summary List conversations
// @Description Retrieves a list of conversations for the authenticated user
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "List of conversations"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations [get]
func (r *ConversationRoute) ListConversations(c *gin.Context) {
	r.conversationsHandler.List(c)
}

// CreateConversation creates a new conversation
// @Summary Create conversation
// @Description Creates a new conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object true "Conversation creation request"
// @Success 201 {object} object "Created conversation"
// @Failure 400 {object} object "Invalid request"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations [post]
func (r *ConversationRoute) CreateConversation(c *gin.Context) {
	r.conversationsHandler.Create(c)
}

// GetConversation retrieves a specific conversation
// @Summary Get conversation
// @Description Retrieves a conversation by ID
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} object "Conversation details"
// @Failure 404 {object} object "Conversation not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (r *ConversationRoute) GetConversation(c *gin.Context) {
	r.conversationsHandler.Get(c)
}

// AppendMessage appends a message to a conversation
// @Summary Append message
// @Description Appends a new message to an existing conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body object true "Message content"
// @Success 200 {object} object "Updated conversation"
// @Failure 400 {object} object "Invalid request"
// @Failure 404 {object} object "Conversation not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations/{conversation_id}/messages [post]
func (r *ConversationRoute) AppendMessage(c *gin.Context) {
	r.conversationsHandler.AppendMessage(c)
}

// ListMessages lists messages in a conversation
// @Summary List messages
// @Description Retrieves all messages in a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} object "List of messages"
// @Failure 404 {object} object "Conversation not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations/{conversation_id}/messages [get]
func (r *ConversationRoute) ListMessages(c *gin.Context) {
	r.conversationsHandler.ListMessages(c)
}

// RunConversation runs a conversation
// @Summary Run conversation
// @Description Executes a conversation run
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body object true "Run configuration"
// @Success 200 {object} object "Run result"
// @Failure 400 {object} object "Invalid request"
// @Failure 404 {object} object "Conversation not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations/{conversation_id}/runs [post]
func (r *ConversationRoute) RunConversation(c *gin.Context) {
	r.conversationsHandler.RunConversation(c)
}

// DeleteConversation deletes a conversation
// @Summary Delete conversation
// @Description Deletes a conversation by ID
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 204 "Conversation deleted"
// @Failure 404 {object} object "Conversation not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/conversations/{conversation_id} [delete]
func (r *ConversationRoute) DeleteConversation(c *gin.Context) {
	// For now, return 204 No Content as a placeholder
	c.Status(204)
}
