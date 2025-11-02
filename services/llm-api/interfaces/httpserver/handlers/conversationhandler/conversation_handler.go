package conversationhandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/datatypes"

	"jan-server/services/llm-api/domain"
	"jan-server/services/llm-api/infrastructure/repo"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
	"jan-server/services/llm-api/utils/idgen"
)

func PrincipalFromContext(c *gin.Context) (domain.Principal, bool) {
	val, ok := c.Get("principal")
	if !ok {
		return domain.Principal{}, false
	}
	principal, ok := val.(domain.Principal)
	return principal, ok
}

func RequestIDFromContext(c *gin.Context) string {
	if val, ok := c.Get("X-Request-Id"); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ConversationsHandler manages conversation resources.
type ConversationsHandler struct {
	conversations *repo.ConversationRepository
	messages      *repo.MessageRepository
	logger        zerolog.Logger
}

// NewConversationsHandler constructs the handler.
func NewConversationsHandler(conversations *repo.ConversationRepository, messages *repo.MessageRepository, logger zerolog.Logger) *ConversationsHandler {
	return &ConversationsHandler{conversations: conversations, messages: messages, logger: logger}
}

// Create handles POST /v1/conversations
// @Summary Create a conversation
// @Tags conversations
// @Security BearerJWT
// @Security ApiKey
// @Accept json
// @Produce json
// @Param request body map[string]string false "Conversation request"
// @Success 201 {object} gin.H
// @Router /v1/conversations [post]
func (h *ConversationsHandler) Create(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	var payload struct {
		Title    string         `json:"title"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_payload",
			Message:   err.Error(),
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	convID, err := idgen.GenerateSecureID("conv", 16)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate conversation ID")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "id_generation_failed",
			Message:   "failed to generate conversation ID",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	conv := &domain.Conversation{
		ID:               convID,
		OwnerPrincipalID: principal.ID,
		Title:            payload.Title,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if payload.Metadata != nil {
		raw, err := json.Marshal(payload.Metadata)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse{
				Type:      responses.ErrorTypeInvalidRequest,
				Code:      "invalid_metadata",
				Message:   "metadata must be valid JSON",
				RequestID: RequestIDFromContext(c),
			})
			return
		}
		conv.Metadata = datatypes.JSON(raw)
	}

	if err := h.conversations.Create(c.Request.Context(), conv); err != nil {
		h.logger.Error().Err(err).Msg("create conversation")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_create_failed",
			Message:   "unable to create conversation",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"conversation_id": conv.ID})
}

// List handles GET /v1/conversations
// @Summary List conversations
// @Tags conversations
// @Security BearerJWT
// @Security ApiKey
// @Produce json
// @Param limit query int false "Page size"
// @Param after query string false "Cursor"
// @Success 200 {object} gin.H
// @Router /v1/conversations [get]
func (h *ConversationsHandler) List(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}

	conversations, next, err := h.conversations.List(c.Request.Context(), principal.ID, limit, c.Query("after"))
	if err != nil {
		h.logger.Error().Err(err).Msg("list conversations")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_list_failed",
			Message:   "unable to list conversations",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	items := make([]gin.H, 0, len(conversations))
	for _, conv := range conversations {
		items = append(items, gin.H{
			"id":                 conv.ID,
			"owner_principal_id": conv.OwnerPrincipalID,
			"title":              conv.Title,
			"created_at":         conv.CreatedAt,
			"updated_at":         conv.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       items,
		"next_after": next,
	})
}

// Get handles GET /v1/conversations/:conversation_id
// @Summary Conversation details
// @Tags conversations
// @Security BearerJWT
// @Security ApiKey
// @Param conversation_id path string true "Conversation ID"
// @Produce json
// @Success 200 {object} gin.H
// @Router /v1/conversations/{conversation_id} [get]
func (h *ConversationsHandler) Get(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	conversationID := c.Param("conversation_id")
	conv, err := h.conversations.Get(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("get conversation")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_get_failed",
			Message:   "unable to fetch conversation",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "conversation_not_found",
			Message:   "conversation not found",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	if conv.OwnerPrincipalID != principal.ID {
		c.JSON(http.StatusNotFound, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "conversation_not_found",
			Message:   "conversation not found",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conv})
}

// AppendMessage handles POST /v1/conversations/{conversation_id}/messages
// @Summary Append message to conversation
// @Tags messages
// @Security BearerJWT
// @Security ApiKey
// @Param conversation_id path string true "Conversation ID"
// @Param request body map[string]any true "Message"
// @Produce json
// @Success 201 {object} gin.H
// @Router /v1/conversations/{conversation_id}/messages [post]
func (h *ConversationsHandler) AppendMessage(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	conversationID := c.Param("conversation_id")
	conv, err := h.conversations.Get(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("get conversation before append")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_get_failed",
			Message:   "unable to fetch conversation",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "conversation_not_found",
			Message:   "conversation not found",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	if conv.OwnerPrincipalID != principal.ID {
		c.JSON(http.StatusForbidden, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "forbidden",
			Message:   "not permitted",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	var payload struct {
		Role      string           `json:"role"`
		Content   map[string]any   `json:"content"`
		ToolCalls []map[string]any `json:"tool_calls"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_payload",
			Message:   err.Error(),
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	rawContent, err := json.Marshal(payload.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_content",
			Message:   "content must be JSON",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	rawTools, _ := json.Marshal(payload.ToolCalls)

	msgID, err := idgen.GenerateSecureID("msg", 16)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate message ID")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "id_generation_failed",
			Message:   "failed to generate message ID",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	message := &domain.Message{
		ID:             msgID,
		ConversationID: conversationID,
		Role:           payload.Role,
		Content:        datatypes.JSON(rawContent),
		ToolCalls:      datatypes.JSON(rawTools),
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.messages.Append(c.Request.Context(), message); err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("append message")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "message_append_failed",
			Message:   "unable to append message",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message_id": message.ID})
}

// ListMessages handles GET /v1/conversations/{conversation_id}/messages
// @Summary List conversation messages
// @Tags messages
// @Security BearerJWT
// @Security ApiKey
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Page size"
// @Param after query string false "Cursor"
// @Produce json
// @Success 200 {object} gin.H
// @Router /v1/conversations/{conversation_id}/messages [get]
func (h *ConversationsHandler) ListMessages(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	conversationID := c.Param("conversation_id")
	conv, err := h.conversations.Get(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("get conversation before list messages")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_get_failed",
			Message:   "unable to fetch conversation",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "conversation_not_found",
			Message:   "conversation not found",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if conv.OwnerPrincipalID != principal.ID {
		c.JSON(http.StatusForbidden, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "forbidden",
			Message:   "not permitted",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}

	messages, next, err := h.messages.List(c.Request.Context(), conversationID, limit, c.Query("after"))
	if err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("list messages")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "message_list_failed",
			Message:   "unable to list messages",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       messages,
		"next_after": next,
	})
}

// RunConversation handles POST /v1/conversations/{conversation_id}/runs
// @Summary Trigger conversation run
// @Tags runs
// @Security BearerJWT
// @Security ApiKey
// @Param conversation_id path string true "Conversation ID"
// @Produce json
// @Success 202 {object} gin.H
// @Router /v1/conversations/{conversation_id}/runs [post]
func (h *ConversationsHandler) RunConversation(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "principal missing",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	conversationID := c.Param("conversation_id")
	conv, err := h.conversations.Get(c.Request.Context(), conversationID)
	if err != nil {
		h.logger.Error().Err(err).Str("conversation_id", conversationID).Msg("get conversation before run")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "conversation_get_failed",
			Message:   "unable to fetch conversation",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if conv == nil || conv.OwnerPrincipalID != principal.ID {
		c.JSON(http.StatusForbidden, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "forbidden",
			Message:   "not permitted",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	runID, err := idgen.GenerateSecureID("run", 16)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate run ID")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "id_generation_failed",
			Message:   "failed to generate run ID",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"run_id":          runID,
		"status":          "queued",
		"conversation_id": conversationID,
	})
}
