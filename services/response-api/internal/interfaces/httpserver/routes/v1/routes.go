package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

// Routes encapsulates versioned route registration.
type Routes struct {
	handlers *handlers.Provider
}

// NewRoutes builds the v1 route registrar.
func NewRoutes(handlerProvider *handlers.Provider) *Routes {
	return &Routes{
		handlers: handlerProvider,
	}
}

// Register attaches all v1 routes under /v1 prefix.
func (r *Routes) Register(engine *gin.Engine) {
	group := engine.Group("/v1")
	registerResponseRoutes(group, r.handlers.Response)

	// Plan routes (optional - only if handler is provided)
	if r.handlers.Plan != nil {
		registerPlanRoutes(group, r.handlers.Plan)
	}

	// Artifact routes (optional - only if handler is provided)
	if r.handlers.Artifact != nil {
		registerArtifactRoutes(group, r.handlers.Artifact)
	}
}
