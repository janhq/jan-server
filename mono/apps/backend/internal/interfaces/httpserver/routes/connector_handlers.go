package routes

import (
	"errors"
	"net/http"

	"jan-server/mono/apps/backend/internal/domain/connector"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Note: Full implementation would include token encryption and OAuth clients
func getConnectorService(cfg *config.Config, db *gorm.DB) *connector.Service {
	repo := repository.NewConnectorRepository(db)
	// TODO: Initialize token encryptor and OAuth clients
	return connector.NewService(repo, nil, connector.ServiceConfig{
		GitHubEnabled:   cfg.GitHubConnectorEnabled,
		GoogleEnabled:   cfg.GoogleConnectorEnabled,
		StateExpiration: cfg.OAuthStateExpiration,
		FrontendURL:     cfg.OAuthFrontendURL,
		EncryptionKeyID: cfg.ConnectorTokenEncryptionKeyID,
	})
}

// ============================================
// Handlers
// ============================================

func listConnectorsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		svc := getConnectorService(cfg, db)
		connectors, err := svc.List(c.Request.Context(), principal.ID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]connector.ConnectorResponse, len(connectors))
		for i, conn := range connectors {
			responses[i] = conn.ToResponse()
		}

		// Also return available providers with their enabled status
		availableProviders := []gin.H{
			{
				"provider":    "github",
				"enabled":     cfg.GitHubConnectorEnabled,
				"display_name": "GitHub",
				"description": "Connect your GitHub account to access repositories",
			},
			{
				"provider":    "gmail",
				"enabled":     cfg.GoogleConnectorEnabled,
				"display_name": "Gmail",
				"description": "Connect your Gmail account to read and send emails",
			},
			{
				"provider":    "google_drive",
				"enabled":     cfg.GoogleConnectorEnabled,
				"display_name": "Google Drive",
				"description": "Connect your Google Drive to access files",
			},
			{
				"provider":    "google_calendar",
				"enabled":     cfg.GoogleConnectorEnabled,
				"display_name": "Google Calendar",
				"description": "Connect your Google Calendar to manage events",
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"connectors":         responses,
			"available_providers": availableProviders,
		})
	}
}

func connectorAuthHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		provider := c.Param("provider")
		if !connector.IsValidProvider(provider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
			return
		}

		redirectURL := c.Query("redirect_url")

		svc := getConnectorService(cfg, db)
		authURL, err := svc.GetAuthURL(c.Request.Context(), principal.ID, provider, redirectURL)

		if err != nil {
			if errors.Is(err, connector.ErrProviderNotEnabled) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "provider_not_enabled",
					"message": "This provider is not enabled",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return auth URL for frontend to redirect to
		c.JSON(http.StatusOK, authURL)
	}
}

func connectorCallbackHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")
		code := c.Query("code")
		state := c.Query("state")
		errorParam := c.Query("error")

		svc := getConnectorService(cfg, db)

		// Handle OAuth error
		if errorParam != "" {
			redirectURL := svc.BuildFrontendRedirectURL("", provider, false, errorParam)
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
			return
		}

		if code == "" || state == "" {
			redirectURL := svc.BuildFrontendRedirectURL("", provider, false, "missing_params")
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
			return
		}

		conn, redirectURL, err := svc.HandleCallback(c.Request.Context(), provider, code, state)

		if err != nil {
			errMsg := "connection_failed"
			if errors.Is(err, connector.ErrInvalidState) {
				errMsg = "invalid_state"
			}
			finalRedirectURL := svc.BuildFrontendRedirectURL(redirectURL, provider, false, errMsg)
			c.Redirect(http.StatusTemporaryRedirect, finalRedirectURL)
			return
		}

		// Success - redirect to frontend
		finalRedirectURL := svc.BuildFrontendRedirectURL(redirectURL, provider, true, "")
		_ = conn // Could pass connector ID in redirect if needed
		c.Redirect(http.StatusTemporaryRedirect, finalRedirectURL)
	}
}

func disconnectConnectorHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		provider := c.Param("provider")
		if !connector.IsValidProvider(provider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
			return
		}

		svc := getConnectorService(cfg, db)
		err := svc.Disconnect(c.Request.Context(), principal.ID, provider)

		if err != nil {
			if errors.Is(err, connector.ErrConnectorNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "connector disconnected"})
	}
}
