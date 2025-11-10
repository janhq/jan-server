package routes

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
	v1 "jan-server/services/response-api/internal/interfaces/httpserver/routes/v1"
)

// Provider coordinates all route registrations.
type Provider struct {
	V1 *v1.Routes
}

// NewProvider constructs the route provider.
func NewProvider(handlerProvider *handlers.Provider) *Provider {
	return &Provider{
		V1: v1.NewRoutes(handlerProvider),
	}
}

// Register attaches all available routes to the gin engine.
func (p *Provider) Register(engine *gin.Engine) {
	p.V1.Register(engine)
}
