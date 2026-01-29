package routes

import (
	"errors"
	"net/http"
	"strconv"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getConversationService(db *gorm.DB) *conversation.Service {
	repo := repository.NewConversationRepository(db)
	return conversation.NewService(repo)
}

// ============================================
// Request types
// ============================================

type createConversationRequest struct {
	Title        string         `json:"title"`
	ModelID      string         `json:"model_id"`
	SystemPrompt string         `json:"system_prompt"`
	Metadata     map[string]any `json:"metadata"`
}

type updateConversationRequest struct {
	Title        *string        `json:"title"`
	ModelID      *string        `json:"model_id"`
	SystemPrompt *string        `json:"system_prompt"`
	IsArchived   *bool          `json:"is_archived"`
	IsPinned     *bool          `json:"is_pinned"`
	Metadata     map[string]any `json:"metadata"`
}

type branchConversationRequest struct {
	FromMessageID *string `json:"from_message_id"`
}

// ============================================
// Handlers
// ============================================

func listConversationsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		var isArchived *bool
		if v := c.Query("is_archived"); v != "" {
			b := v == "true"
			isArchived = &b
		}

		var isPinned *bool
		if v := c.Query("is_pinned"); v != "" {
			b := v == "true"
			isPinned = &b
		}

		svc := getConversationService(db)
		conversations, total, err := svc.List(c.Request.Context(), conversation.ListConversationsFilter{
			UserID:     principal.ID,
			IsArchived: isArchived,
			IsPinned:   isPinned,
			Search:     search,
			Limit:      limit,
			Offset:     offset,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]conversation.ConversationResponse, len(conversations))
		for i, conv := range conversations {
			responses[i] = conv.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"conversations": responses,
			"total":         total,
			"limit":         limit,
			"offset":        offset,
		})
	}
}

func createConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req createConversationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getConversationService(db)
		conv, err := svc.Create(c.Request.Context(), principal.ID, conversation.CreateConversationRequest{
			Title:        req.Title,
			ModelID:      req.ModelID,
			SystemPrompt: req.SystemPrompt,
			Metadata:     req.Metadata,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, conv.ToResponse())
	}
}

func getConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getConversationService(db)
		conv, err := svc.GetByID(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
				return
			}
			if errors.Is(err, conversation.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := conv.ToResponse()
		// Include messages
		messages := make([]conversation.MessageResponse, len(conv.Messages))
		for i, msg := range conv.Messages {
			messages[i] = msg.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"conversation": resp,
			"messages":     messages,
		})
	}
}

func updateConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		var req updateConversationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getConversationService(db)
		conv, err := svc.Update(c.Request.Context(), principal.ID, id, conversation.UpdateConversationRequest{
			Title:        req.Title,
			ModelID:      req.ModelID,
			SystemPrompt: req.SystemPrompt,
			IsArchived:   req.IsArchived,
			IsPinned:     req.IsPinned,
			Metadata:     req.Metadata,
		})

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
				return
			}
			if errors.Is(err, conversation.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, conv.ToResponse())
	}
}

func deleteConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getConversationService(db)
		err := svc.Delete(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
				return
			}
			if errors.Is(err, conversation.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "conversation deleted"})
	}
}

func branchConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		var req branchConversationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getConversationService(db)
		conv, err := svc.Branch(c.Request.Context(), principal.ID, id, req.FromMessageID)

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
				return
			}
			if errors.Is(err, conversation.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, conv.ToResponse())
	}
}

// ============================================
// Message Handlers
// ============================================

func listMessagesHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		conversationID := c.Query("conversation_id")
		if conversationID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		svc := getConversationService(db)
		messages, total, err := svc.ListMessages(c.Request.Context(), principal.ID, conversationID, limit, offset)

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]conversation.MessageResponse, len(messages))
		for i, msg := range messages {
			responses[i] = msg.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": responses,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func getMessageHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getConversationService(db)
		msg, err := svc.GetMessage(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, conversation.ErrMessageNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, msg.ToResponse())
	}
}

// ============================================
// Share Handlers
// ============================================

func getSharedConversationHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		svc := getConversationService(db)
		conv, err := svc.GetBySharedToken(c.Request.Context(), token)

		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "shared conversation not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		resp := conv.ToResponse()
		messages := make([]conversation.MessageResponse, len(conv.Messages))
		for i, msg := range conv.Messages {
			messages[i] = msg.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"conversation": resp,
			"messages":     messages,
		})
	}
}
