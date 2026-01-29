package share

import (
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/conversationhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/sharehandler"
	_ "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/share"
	_ "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	_ "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/share"

	"github.com/gin-gonic/gin"
)

// ShareRoute handles routing for conversation share endpoints
type ShareRoute struct {
	handler             *sharehandler.ShareHandler
	authHandler         *authhandler.AuthHandler
	conversationHandler *conversationhandler.ConversationHandler
}

// NewShareRoute creates a new share route handler
func NewShareRoute(
	handler *sharehandler.ShareHandler,
	authHandler *authhandler.AuthHandler,
	conversationHandler *conversationhandler.ConversationHandler,
) *ShareRoute {
	return &ShareRoute{
		handler:             handler,
		authHandler:         authHandler,
		conversationHandler: conversationHandler,
	}
}

// RegisterUserShareRoutes registers share routes under /shares
// These routes require authentication and operate on all user shares
func (route *ShareRoute) RegisterUserShareRoutes(router gin.IRouter) {
	router.GET("", route.authHandler.WithAppUserAuthChain(route.listUserShares)...)
	router.DELETE("/:share_id", route.authHandler.WithAppUserAuthChain(route.revokeUserShare)...)
}

// listUserShares godoc
// @Summary List all shares for the authenticated user
// @Description Lists all shares (active and revoked) for the authenticated user across all conversations
// @Tags Shares API
// @Security BearerAuth
// @Produce json
// @Param include_revoked query bool false "Include revoked shares" default(true)
// @Success 200 {object} shareresponses.ShareListResponse "List of shares"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Forbidden"
// @Router /v1/shares [get]
func (route *ShareRoute) listUserShares(reqCtx *gin.Context) {
	route.handler.ListUserShares(reqCtx)
}

// revokeUserShare godoc
// @Summary Revoke a share by share ID
// @Description Revokes an active share by its ID, making it inaccessible
// @Tags Shares API
// @Security BearerAuth
// @Produce json
// @Param share_id path string true "Share public ID"
// @Success 200 {object} shareresponses.ShareDeletedResponse "Share revoked successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Forbidden"
// @Failure 404 {object} responses.ErrorResponse "Share not found"
// @Router /v1/shares/{share_id} [delete]
func (route *ShareRoute) revokeUserShare(reqCtx *gin.Context) {
	route.handler.RevokeUserShare(reqCtx)
}

// RegisterConversationShareRoutes registers share routes under /conversations/:conv_public_id
// These routes require authentication
func (route *ShareRoute) RegisterConversationShareRoutes(router gin.IRouter) {
	router.POST("/:conv_public_id/share",
		route.authHandler.WithAppUserAuthChain(
			route.conversationHandler.ConversationMiddleware(),
			route.createShare,
		)...)
	router.GET("/:conv_public_id/shares",
		route.authHandler.WithAppUserAuthChain(
			route.conversationHandler.ConversationMiddleware(),
			route.listShares,
		)...)
	router.DELETE("/:conv_public_id/shares/:share_id",
		route.authHandler.WithAppUserAuthChain(
			route.conversationHandler.ConversationMiddleware(),
			route.revokeShare,
		)...)
}

// createShare godoc
// @Summary Create a share for a conversation
// @Description Creates a public share link for a conversation or a single message
// @Tags Shares API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation public ID"
// @Param request body sharerequests.CreateShareRequest true "Share creation request"
// @Success 201 {object} shareresponses.ShareResponse "Share created successfully"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Forbidden"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 413 {object} responses.ErrorResponse "Snapshot too large"
// @Router /v1/conversations/{conv_public_id}/share [post]
func (route *ShareRoute) createShare(reqCtx *gin.Context) {
	route.handler.CreateShare(reqCtx)
}

// listShares godoc
// @Summary List shares for a conversation
// @Description Lists all shares (active and revoked) for a conversation
// @Tags Shares API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation public ID"
// @Success 200 {object} shareresponses.ShareListResponse "List of shares"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Forbidden"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Router /v1/conversations/{conv_public_id}/shares [get]
func (route *ShareRoute) listShares(reqCtx *gin.Context) {
	route.handler.ListShares(reqCtx)
}

// revokeShare godoc
// @Summary Revoke a share
// @Description Revokes an active share, making it inaccessible
// @Tags Shares API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation public ID"
// @Param share_id path string true "Share public ID"
// @Success 200 {object} shareresponses.ShareDeletedResponse "Share revoked successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Forbidden"
// @Failure 404 {object} responses.ErrorResponse "Share not found"
// @Router /v1/conversations/{conv_public_id}/shares/{share_id} [delete]
func (route *ShareRoute) revokeShare(reqCtx *gin.Context) {
	route.handler.RevokeShare(reqCtx)
}
