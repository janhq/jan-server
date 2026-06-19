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
	group.GET("/media/:id/metadata", r.handlers.Media.GetMetadata)
	group.POST("/files", r.handlers.Media.Ingest)
	group.POST("/files/upload", r.handlers.Media.DirectUpload)
	group.GET("/files/:id", r.handlers.Media.Proxy)
	group.GET("/files/:id/metadata", r.handlers.Media.GetMetadata)

	// NOTE: Local files are served by ID through the Proxy handler above
	// (GET /v1/files/:id -> service.Download). A path-based gin Static mount at
	// "/files" registers "/v1/files/*filepath", which conflicts with the
	// "/v1/files/:id" route and panics at startup, so it is intentionally omitted.
}
