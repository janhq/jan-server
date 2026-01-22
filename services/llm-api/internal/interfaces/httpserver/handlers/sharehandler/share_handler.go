package sharehandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/janhq/jan-server/packages/go-common/analytics"
	analyticsMiddleware "github.com/janhq/jan-server/packages/go-common/analytics/middleware"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/share"
	"jan-server/services/llm-api/internal/infrastructure/metrics"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	sharerequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/share"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	shareresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/share"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// ShareHandler handles share-related HTTP requests
type ShareHandler struct {
	shareService        *share.ShareService
	conversationHandler *conversationhandler.ConversationHandler
	cfg                 *config.Config
}

// NewShareHandler creates a new share handler
func NewShareHandler(
	shareService *share.ShareService,
	conversationHandler *conversationhandler.ConversationHandler,
	cfg *config.Config,
) *ShareHandler {
	return &ShareHandler{
		shareService:        shareService,
		conversationHandler: conversationHandler,
		cfg:                 cfg,
	}
}

// CreateShare handles POST /v1/conversations/:conv_public_id/share
func (h *ShareHandler) CreateShare(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Check feature flag
	if !h.cfg.ConversationSharingEnabled {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeForbidden,
			"conversation sharing is not enabled", "share-disabled-001")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized,
			"authentication required", "share-auth-001")
		return
	}

	// Get conversation from middleware context
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound,
			"conversation not found", "share-conv-001")
		return
	}

	var req sharerequests.CreateShareRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation,
			"invalid request body", "share-body-001")
		return
	}

	// Check for branch query parameter (takes precedence over body)
	if branchParam := reqCtx.Query("branch"); branchParam != "" {
		req.Branch = &branchParam
	}

	// Validate item_id is provided for item scope
	if req.Scope == "item" && (req.ItemID == nil || *req.ItemID == "") {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation,
			"item_id is required when scope is 'item'", "share-item-001")
		return
	}

	input := share.CreateShareInput{
		ConversationID:         conv.ID,
		ItemPublicID:           req.ItemID,
		OwnerUserID:            user.ID,
		Title:                  req.Title,
		Scope:                  req.ToShareScope(),
		IncludeImages:          req.IncludeImages,
		IncludeContextMessages: req.IncludeContextMessages,
		Branch:                 req.Branch,
	}

	output, err := h.shareService.CreateShare(ctx, input)
	if err != nil {
		metrics.RecordShare(req.Scope, "error")
		responses.HandleError(reqCtx, err, "failed to create share")
		return
	}

	metrics.RecordShare(req.Scope, "success")

	// Track share_created event
	values := analytics.ValuesFromContext(ctx)
	if values.DistinctID != "" {
		props := map[string]interface{}{
			analytics.PropConversationID: conv.PublicID,
			analytics.PropShareType:      req.Scope,
			analytics.PropUserStatus:     values.UserStatus,
			analytics.PropPlatform:       values.Platform,
		}
		_ = analyticsMiddleware.TrackEvent(reqCtx, analytics.EventShareCreated, props)
	}

	resp := shareresponses.NewShareResponse(output.Share, h.getBaseURL(reqCtx))
	reqCtx.JSON(http.StatusCreated, resp)
}

// ListShares handles GET /v1/conversations/:conv_public_id/shares
func (h *ShareHandler) ListShares(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Check feature flag
	if !h.cfg.ConversationSharingEnabled {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeForbidden,
			"conversation sharing is not enabled", "share-disabled-002")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized,
			"authentication required", "share-auth-002")
		return
	}

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound,
			"conversation not found", "share-conv-002")
		return
	}

	shares, err := h.shareService.ListSharesByConversation(ctx, conv.ID, user.ID, true)
	if err != nil {
		responses.HandleError(reqCtx, err, "failed to list shares")
		return
	}

	resp := shareresponses.NewShareListResponse(shares, h.getBaseURL(reqCtx))
	reqCtx.JSON(http.StatusOK, resp)
}

// ListUserShares handles GET /v1/shares
func (h *ShareHandler) ListUserShares(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Check feature flag
	if !h.cfg.ConversationSharingEnabled {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeForbidden,
			"conversation sharing is not enabled", "share-disabled-004")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized,
			"authentication required", "share-auth-004")
		return
	}

	// Parse include_revoked query param (default true)
	includeRevoked := true
	if incRevokedParam := reqCtx.Query("include_revoked"); incRevokedParam == "false" {
		includeRevoked = false
	}

	shares, err := h.shareService.ListUserShares(ctx, user.ID, includeRevoked)
	if err != nil {
		responses.HandleError(reqCtx, err, "failed to list user shares")
		return
	}

	resp := shareresponses.NewShareListResponse(shares, h.getBaseURL(reqCtx))
	reqCtx.JSON(http.StatusOK, resp)
}

// RevokeUserShare handles DELETE /v1/shares/:share_id
func (h *ShareHandler) RevokeUserShare(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Check feature flag
	if !h.cfg.ConversationSharingEnabled {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeForbidden,
			"conversation sharing is not enabled", "share-disabled-005")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized,
			"authentication required", "share-auth-005")
		return
	}

	shareID := reqCtx.Param("share_id")
	if shareID == "" {
		metrics.RecordShare("unknown", "error")
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation,
			"share_id is required", "share-id-002")
		return
	}

	err := h.shareService.RevokeShare(ctx, shareID, user.ID)
	if err != nil {
		metrics.RecordShare("unknown", "error")
		responses.HandleError(reqCtx, err, "failed to revoke share")
		return
	}

	metrics.RecordShare("unknown", "success")
	resp := shareresponses.NewShareDeletedResponse(shareID)
	reqCtx.JSON(http.StatusOK, resp)
}

// RevokeShare handles DELETE /v1/conversations/:conv_public_id/shares/:share_id
func (h *ShareHandler) RevokeShare(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Check feature flag
	if !h.cfg.ConversationSharingEnabled {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeForbidden,
			"conversation sharing is not enabled", "share-disabled-003")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized,
			"authentication required", "share-auth-003")
		return
	}

	shareID := reqCtx.Param("share_id")
	if shareID == "" {
		metrics.RecordShare("unknown", "error")
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation,
			"share_id is required", "share-id-001")
		return
	}

	err := h.shareService.RevokeShare(ctx, shareID, user.ID)
	if err != nil {
		metrics.RecordShare("unknown", "error")
		responses.HandleError(reqCtx, err, "failed to revoke share")
		return
	}

	metrics.RecordShare("unknown", "success")
	resp := shareresponses.NewShareDeletedResponse(shareID)
	reqCtx.JSON(http.StatusOK, resp)
}

// GetPublicShare handles GET /v1/public/shares/:slug
func (h *ShareHandler) GetPublicShare(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Note: Feature flag check is NOT done here to allow viewing existing shares
	// even if new share creation is disabled

	slug := reqCtx.Param("slug")
	if slug == "" {
		metrics.RecordPublicShareRequest(reqCtx.Request.Method, "400")
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation,
			"slug is required", "public-share-slug-001")
		return
	}

	sh, err := h.shareService.GetShareBySlug(ctx, slug)
	if err != nil {
		// Check if this is a "revoked" error
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			metrics.RecordPublicShareRequest(reqCtx.Request.Method, "410")
			reqCtx.AbortWithStatusJSON(http.StatusGone, gin.H{
				"error":   "share_revoked",
				"message": "This share has been revoked",
			})
			return
		}
		metrics.RecordPublicShareRequest(reqCtx.Request.Method, "404")
		responses.HandleError(reqCtx, err, "share not found")
		return
	}

	metrics.RecordPublicShareRequest(reqCtx.Request.Method, "200")
	resp := shareresponses.NewPublicShareResponse(sh)

	// Set cache headers (5 minute TTL)
	reqCtx.Header("Cache-Control", "public, max-age=300")

	reqCtx.JSON(http.StatusOK, resp)
}

// HeadPublicShare handles HEAD /v1/public/shares/:slug
func (h *ShareHandler) HeadPublicShare(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	slug := reqCtx.Param("slug")
	if slug == "" {
		metrics.RecordPublicShareRequest(reqCtx.Request.Method, "400")
		reqCtx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	sh, err := h.shareService.GetShareBySlug(ctx, slug)
	if err != nil {
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			metrics.RecordPublicShareRequest(reqCtx.Request.Method, "410")
			reqCtx.AbortWithStatus(http.StatusGone)
			return
		}
		metrics.RecordPublicShareRequest(reqCtx.Request.Method, "404")
		reqCtx.AbortWithStatus(http.StatusNotFound)
		return
	}

	if sh.IsRevoked() {
		metrics.RecordPublicShareRequest(reqCtx.Request.Method, "410")
		reqCtx.AbortWithStatus(http.StatusGone)
		return
	}

	metrics.RecordPublicShareRequest(reqCtx.Request.Method, "200")
	reqCtx.Status(http.StatusOK)
}

// getBaseURL returns the base URL for constructing share URLs
func (h *ShareHandler) getBaseURL(reqCtx *gin.Context) string {
	scheme := "https"
	if reqCtx.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + reqCtx.Request.Host
}
