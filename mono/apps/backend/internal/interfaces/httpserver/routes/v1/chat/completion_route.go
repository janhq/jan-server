package chat

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/chathandler"
	chatrequests "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/chat"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	chatresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/chat"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ChatCompletionRoute handles chat completion requests with streaming support by delegating to the chat handler.
type ChatCompletionRoute struct {
	chatHandler *chathandler.ChatHandler
	authHandler *authhandler.AuthHandler
}

func NewChatCompletionRoute(
	chatHandler *chathandler.ChatHandler,
	authHandler *authhandler.AuthHandler,
) *ChatCompletionRoute {
	return &ChatCompletionRoute{
		chatHandler: chatHandler,
		authHandler: authHandler,
	}
}

func (chatCompletionRoute *ChatCompletionRoute) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions",
		chatCompletionRoute.authHandler.WithAppUserAuthChain(
			chatCompletionRoute.PostCompletion,
		)...,
	)
}

// PostCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation. This is a standard chat completion API that supports both streaming and non-streaming modes without conversation persistence.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with real-time streaming
// @Description - Streams completion chunks directly from the inference model
// @Description - Final event contains "[DONE]" marker
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete completion
// @Description - Standard OpenAI ChatCompletionResponse format
// @Description
// @Description **Storage Options:**
// @Description - `store=true`: Persist the latest input message and assistant response to the active conversation
// @Description - `store_reasoning=true`: Additionally persist reasoning content provided by the model
// @Description - When `store` is omitted or false, the conversation remains read-only
// @Description
// @Description **Features:**
// @Description - Supports all OpenAI ChatCompletionRequest parameters
// @Description - Optional conversation context for conversation persistence
// @Description - User authentication required
// @Description - Direct inference model integration
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body chatrequests.ChatCompletionRequest true "Chat completion request with streaming options and optional conversation"
// @Success 200 {object} chatresponses.ChatCompletionResponse "Successful non-streaming response (when stream=false)"
// @Success 200 {string} string "Successful streaming response (when stream=true) - SSE format with data: {json} events"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload, empty messages, or inference failure"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (chatCompletionRoute *ChatCompletionRoute) PostCompletion(reqCtx *gin.Context) {
	// Get authenticated user ID
	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "81b47b8b-ddaa-4819-a7b4-a29042c60100")
		return
	}

	var request chatrequests.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	// Delegate to chat handler
	result, err := chatCompletionRoute.chatHandler.CreateChatCompletion(reqCtx.Request.Context(), reqCtx, user.ID, request)
	if err != nil {
		// Check if it's a validation error (user input too large)
		if platformerrors.IsValidationError(err) {
			responses.HandleError(reqCtx, err, err.Error())
			return
		}

		// Check if it's a NOT_FOUND error (conversation not found, etc.) - return proper error
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound, err.Error(), "conv-not-found-001")
			return
		}

		// Check if it's an internal/server error that's NOT from LLM - return proper error
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeForbidden) ||
			platformerrors.IsErrorType(err, platformerrors.ErrorTypeUnauthorized) ||
			platformerrors.IsErrorType(err, platformerrors.ErrorTypeConflict) {
			responses.HandleError(reqCtx, err, err.Error())
			return
		}

		// Only for LLM/model communication errors, return fallback response
		fallback := chatCompletionRoute.chatHandler.BuildFallbackResponse(request.Model)
		chatResponse := chatresponses.NewChatCompletionResponse(fallback, "", nil, false)
		reqCtx.JSON(http.StatusOK, chatResponse)
		return
	}

	// For non-streaming requests, return the response with conversation context
	if !request.Stream {
		// Wrap the OpenAI response with conversation context (including title)
		chatResponse := chatresponses.NewChatCompletionResponse(result.Response, result.ConversationID, result.ConversationTitle, result.Trimmed)
		reqCtx.JSON(http.StatusOK, chatResponse)
	}

}
