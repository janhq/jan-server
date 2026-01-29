package public

import (
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/sharehandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	_ "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	_ "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/share"

	"github.com/gin-gonic/gin"
)

// PublicShareRoute handles routing for public share endpoints (no auth required)
type PublicShareRoute struct {
	handler *sharehandler.ShareHandler
}

// NewPublicShareRoute creates a new public share route handler
func NewPublicShareRoute(handler *sharehandler.ShareHandler) *PublicShareRoute {
	return &PublicShareRoute{
		handler: handler,
	}
}

// RegisterRouter registers public share routes
// These routes do NOT require authentication but have rate limiting
func (route *PublicShareRoute) RegisterRouter(router gin.IRouter) {
	publicShares := router.Group("/public/shares")

	// Apply rate limiting middleware (100 req/min per IP)
	publicShares.Use(middlewares.RateLimitMiddleware(100))

	publicShares.GET("/:slug", route.getPublicShare)
	publicShares.HEAD("/:slug", route.headPublicShare)
}

// getPublicShare godoc
// @Summary Get a public share by slug
// @Description Retrieves a publicly shared conversation or message by its slug
// @Tags Public Shares API
// @Produce json
// @Param slug path string true "Share slug"
// @Success 200 {object} shareresponses.PublicShareResponse "Public share content"
// @Failure 404 {object} responses.ErrorResponse "Share not found or revoked"
// @Failure 410 {object} responses.ErrorResponse "Share has been revoked"
// @Router /v1/public/shares/{slug} [get]
func (route *PublicShareRoute) getPublicShare(reqCtx *gin.Context) {
	route.handler.GetPublicShare(reqCtx)
}

// headPublicShare godoc
// @Summary Check if a public share exists
// @Description Checks if a share exists and is accessible (for preloading)
// @Tags Public Shares API
// @Param slug path string true "Share slug"
// @Success 200 "Share exists and is accessible"
// @Failure 404 "Share not found"
// @Failure 410 "Share has been revoked"
// @Router /v1/public/shares/{slug} [head]
func (route *PublicShareRoute) headPublicShare(reqCtx *gin.Context) {
	route.handler.HeadPublicShare(reqCtx)
}
