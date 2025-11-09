package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/media-api/internal/interfaces/httpserver/handlers"
)

// Routes encapsulates versioned route registration.
type Routes struct {
	handlers *handlers.Provider
}

func NewRoutes(provider *handlers.Provider) *Routes {
	return &Routes{handlers: provider}
}

// Register attaches all v1 routes under /v1 prefix.
func (r *Routes) Register(router gin.IRouter) {
	group := router.Group("/v1")
	group.POST("/media", r.handlers.Media.Ingest)
	group.POST("/media/prepare-upload", r.handlers.Media.PrepareUpload)
	group.POST("/media/resolve", r.handlers.Media.Resolve)
	group.GET("/media/:id", r.handlers.Media.Proxy)
}
