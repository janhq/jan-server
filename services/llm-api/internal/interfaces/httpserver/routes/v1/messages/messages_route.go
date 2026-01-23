package messages

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/messageshandler"
	messagesrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/messages"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// MessagesRoute handles Anthropic Messages API endpoints
type MessagesRoute struct {
	messagesHandler *messageshandler.MessagesHandler
	authHandler     *authhandler.AuthHandler
}

// NewMessagesRoute creates a new messages route
func NewMessagesRoute(
	messagesHandler *messageshandler.MessagesHandler,
	authHandler *authhandler.AuthHandler,
) *MessagesRoute {
	return &MessagesRoute{
		messagesHandler: messagesHandler,
		authHandler:     authHandler,
	}
}

// RegisterRouter registers the messages routes
func (r *MessagesRoute) RegisterRouter(router gin.IRouter) {
	messagesGroup := router.Group("/messages")
	{
		// POST /v1/messages - Create a message (Anthropic Messages API)
		messagesGroup.POST("",
			r.authHandler.WithAppUserAuthChain(
				r.CreateMessage,
			)...,
		)

		// POST /v1/messages/count_tokens - Count tokens in a message
		messagesGroup.POST("/count_tokens",
			r.authHandler.WithAppUserAuthChain(
				r.CountTokens,
			)...,
		)
	}
}

// CreateMessage godoc
// @Summary Create a message (Anthropic Messages API compatible)
// @Description Send a structured list of input messages and receive a model-generated response.
// @Description This endpoint is compatible with the Anthropic Messages API format.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with Anthropic event types
// @Description - Events: message_start, content_block_start, content_block_delta, content_block_stop, message_delta, message_stop
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete message
// @Description
// @Description **Features:**
// @Description - Multi-turn conversations with user/assistant messages
// @Description - System prompts for setting context
// @Description - Tool use (function calling) with tool_use/tool_result content blocks
// @Description - Vision support with image content blocks (base64 and URL)
// @Description - Extended thinking parameter support
// @Tags Anthropic Messages API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body messagesrequests.AnthropicMessagesRequest true "Message creation request"
// @Success 200 {object} messagesresponses.AnthropicMessagesResponse "Successful non-streaming response"
// @Success 200 {string} string "Successful streaming response - SSE format with Anthropic event types"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - missing required fields or invalid format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Model not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/messages [post]
func (r *MessagesRoute) CreateMessage(reqCtx *gin.Context) {
	// Get authenticated user
	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "msg-auth-001")
		return
	}

	// Parse request
	var request messagesrequests.AnthropicMessagesRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if request.Model == "" {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "model is required",
			},
		})
		return
	}

	if len(request.Messages) == 0 {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "messages is required and must not be empty",
			},
		})
		return
	}

	if request.MaxTokens <= 0 {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "max_tokens is required and must be greater than 0",
			},
		})
		return
	}

	// Delegate to handler
	if err := r.messagesHandler.CreateMessage(reqCtx.Request.Context(), reqCtx, user.ID, request); err != nil {
		// Error already written by handler
		return
	}
}

// CountTokens godoc
// @Summary Count tokens in a message (Anthropic Messages API compatible)
// @Description Count the number of tokens in a message before sending it.
// @Description This is useful for managing context window limits.
// @Tags Anthropic Messages API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body messagesrequests.AnthropicCountTokensRequest true "Token counting request"
// @Success 200 {object} messagesresponses.AnthropicCountTokensResponse "Token count response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Model not found"
// @Router /v1/messages/count_tokens [post]
func (r *MessagesRoute) CountTokens(reqCtx *gin.Context) {
	// Get authenticated user
	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "msg-auth-002")
		return
	}

	// Parse request
	var request messagesrequests.AnthropicCountTokensRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if request.Model == "" {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "model is required",
			},
		})
		return
	}

	if len(request.Messages) == 0 {
		reqCtx.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "messages is required and must not be empty",
			},
		})
		return
	}

	// Delegate to handler
	if err := r.messagesHandler.CountTokens(reqCtx.Request.Context(), reqCtx, user.ID, request); err != nil {
		// Error already written by handler
		return
	}
}
