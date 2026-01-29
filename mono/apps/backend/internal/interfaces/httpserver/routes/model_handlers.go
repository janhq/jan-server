package routes

import (
	"errors"
	"net/http"
	"strconv"

	"jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getModelService(db *gorm.DB) *model.Service {
	repo := repository.NewModelRepository(db)
	return model.NewService(repo)
}

// ============================================
// Request types
// ============================================

type createProviderRequest struct {
	Name        string         `json:"name" binding:"required"`
	DisplayName string         `json:"display_name"`
	BaseURL     string         `json:"base_url"`
	APIKey      string         `json:"api_key"`
	Config      map[string]any `json:"config"`
}

type createModelRequest struct {
	ProviderID    string                  `json:"provider_id" binding:"required"`
	Name          string                  `json:"name" binding:"required"`
	DisplayName   string                  `json:"display_name"`
	Description   string                  `json:"description"`
	ContextWindow int                     `json:"context_window"`
	MaxTokens     int                     `json:"max_tokens"`
	Capabilities  model.ModelCapabilities `json:"capabilities"`
	Pricing       model.ModelPricing      `json:"pricing"`
}

// ============================================
// Model Handlers
// ============================================

func listModelsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		providerID := c.Query("provider_id")
		search := c.Query("search")

		// Default to enabled only for regular users
		enabledOnly := true
		if v := c.Query("enabled_only"); v == "false" {
			enabledOnly = false
		}

		var isEnabled *bool
		if enabledOnly {
			isEnabled = &enabledOnly
		}

		svc := getModelService(db)
		models, total, err := svc.ListModels(c.Request.Context(), model.ListModelsFilter{
			ProviderID: providerID,
			IsEnabled:  isEnabled,
			Search:     search,
			Limit:      limit,
			Offset:     offset,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]model.ModelResponse, len(models))
		for i, m := range models {
			responses[i] = m.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"models": responses,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func getModelHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := getModelService(db)
		m, err := svc.GetModelByID(c.Request.Context(), id)

		if err != nil {
			if errors.Is(err, model.ErrModelNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, m.ToResponse())
	}
}

// ============================================
// Provider Handlers
// ============================================

func listProvidersHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabledOnly := c.Query("enabled_only") != "false"

		svc := getModelService(db)
		providers, err := svc.ListProviders(c.Request.Context(), enabledOnly)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]model.ProviderResponse, len(providers))
		for i, p := range providers {
			responses[i] = p.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"providers": responses,
		})
	}
}

func getProviderHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := getModelService(db)
		p, err := svc.GetProviderByID(c.Request.Context(), id)

		if err != nil {
			if errors.Is(err, model.ErrProviderNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, p.ToResponse())
	}
}

// ============================================
// Admin Model Handlers
// ============================================

func adminCreateModelHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req createModelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getModelService(db)
		m, err := svc.CreateModel(c.Request.Context(), model.CreateModelRequest{
			ProviderID:    req.ProviderID,
			Name:          req.Name,
			DisplayName:   req.DisplayName,
			Description:   req.Description,
			ContextWindow: req.ContextWindow,
			MaxTokens:     req.MaxTokens,
			Capabilities:  req.Capabilities,
			Pricing:       req.Pricing,
		})

		if err != nil {
			if errors.Is(err, model.ErrProviderNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "provider not found"})
				return
			}
			if errors.Is(err, model.ErrModelExists) {
				c.JSON(http.StatusConflict, gin.H{"error": "model already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, m.ToResponse())
	}
}

func adminUpdateModelHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getModelService(db)
		m, err := svc.UpdateModel(c.Request.Context(), id, updates)

		if err != nil {
			if errors.Is(err, model.ErrModelNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, m.ToResponse())
	}
}

func adminDeleteModelHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := getModelService(db)
		err := svc.DeleteModel(c.Request.Context(), id)

		if err != nil {
			if errors.Is(err, model.ErrModelNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
	}
}

// ============================================
// Admin Provider Handlers
// ============================================

func adminCreateProviderHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createProviderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getModelService(db)
		p, err := svc.CreateProvider(c.Request.Context(), model.CreateProviderRequest{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			BaseURL:     req.BaseURL,
			APIKey:      req.APIKey,
			Config:      req.Config,
		})

		if err != nil {
			if errors.Is(err, model.ErrProviderExists) {
				c.JSON(http.StatusConflict, gin.H{"error": "provider already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, p.ToResponse())
	}
}

func adminUpdateProviderHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]any
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getModelService(db)
		p, err := svc.UpdateProvider(c.Request.Context(), id, updates)

		if err != nil {
			if errors.Is(err, model.ErrProviderNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, p.ToResponse())
	}
}

func adminDeleteProviderHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		svc := getModelService(db)
		err := svc.DeleteProvider(c.Request.Context(), id)

		if err != nil {
			if errors.Is(err, model.ErrProviderNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "provider deleted"})
	}
}
