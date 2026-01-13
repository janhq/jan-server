package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/media-api/internal/config"
	"jan-server/services/media-api/internal/interfaces/httpserver/handlers"
)

// Routes encapsulates versioned route registration.
type Routes struct {
	handlers *handlers.Provider
	cfg      *config.Config
}

func NewRoutes(provider *handlers.Provider, cfg *config.Config) *Routes {
	return &Routes{
		handlers: provider,
		cfg:      cfg,
	}
}

// Register attaches all v1 routes under /v1 prefix.
func (r *Routes) Register(router gin.IRouter) {
	group := router.Group("/v1")
	group.POST("/media", r.handlers.Media.Ingest)
	group.POST("/media/upload", r.handlers.Media.DirectUpload)
	group.GET("/media/:id", r.handlers.Media.Proxy)

	// Serve static files from local storage if configured
	if r.cfg.IsLocalStorage() && r.cfg.LocalStoragePath != "" {
		// Strip /v1 prefix and serve files from /v1/files/*
		group.Static("/files", r.cfg.LocalStoragePath)
	}
}
