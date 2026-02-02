package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/interfaces/httpserver/responses"
)

// ArtifactHandler exposes HTTP entrypoints for the Artifacts API.
type ArtifactHandler struct {
	service artifact.Service
	log     zerolog.Logger
}

// NewArtifactHandler constructs the handler.
func NewArtifactHandler(service artifact.Service, log zerolog.Logger) *ArtifactHandler {
	return &ArtifactHandler{
		service: service,
		log:     log.With().Str("handler", "artifact").Logger(),
	}
}

// List handles GET /v1/artifacts
// @Summary List all artifacts for the authenticated user
// @Description Retrieves all artifacts belonging to the authenticated user with cursor-based pagination
// @Tags Artifacts
// @Produce json
// @Param content_type query string false "Filter by content type (slides, document, research, code, image, etc.)"
// @Param search query string false "Search by title (case-insensitive)"
// @Param latest query bool false "Only return latest versions" default(true)
// @Param limit query int false "Maximum number of results" default(20)
// @Param after query string false "Return artifacts after the given artifact ID (cursor pagination)"
// @Param order query string false "Sort order: asc or desc" default(desc)
// @Success 200 {object} responses.ArtifactListResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/artifacts [get]
func (h *ArtifactHandler) List(c *gin.Context) {
	// Get user ID from JWT token (set by auth middleware)
	userID := extractUserIDFromToken(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	filter := artifact.NewFilter().WithUserID(userID).WithLatestOnly()

	// Parse query params
	if contentType := c.Query("content_type"); contentType != "" {
		filter = filter.WithContentType(artifact.ContentType(contentType))
	}

	if search := c.Query("search"); search != "" {
		filter = filter.WithTitleSearch(search)
	}

	if latestStr := c.Query("latest"); latestStr == "false" {
		filter = filter.WithAllVersions()
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Parse cursor-based pagination params
	if afterStr := c.Query("after"); afterStr != "" {
		internalID, err := h.service.ResolveInternalID(c.Request.Context(), afterStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination cursor"})
			return
		}
		filter.After = internalID
	}

	if order := c.Query("order"); order == "asc" || order == "desc" {
		filter.Order = order
	}

	artifacts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		responses.HandleError(c, err, "failed to list artifacts")
		return
	}

	// Determine has_more and trim to limit
	hasMore := len(artifacts) > filter.Limit
	if hasMore {
		artifacts = artifacts[:filter.Limit]
	}

	// Build response with cursor info
	data := responses.MapArtifactsToResponse(artifacts)
	firstID := ""
	lastID := ""
	if len(data) > 0 {
		firstID = data[0].ID
		lastID = data[len(data)-1].ID
	}

	c.JSON(http.StatusOK, responses.ArtifactListResponse{
		Object:  "list",
		Data:    data,
		FirstID: firstID,
		LastID:  lastID,
		HasMore: hasMore,
		Total:   total,
	})
}

// Get handles GET /v1/artifacts/:artifact_id
// @Summary Get artifact by ID
// @Description Retrieves an artifact by its ID
// @Tags Artifacts
// @Produce json
// @Param artifact_id path string true "Artifact ID"
// @Success 200 {object} responses.ArtifactResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/artifacts/{artifact_id} [get]
func (h *ArtifactHandler) Get(c *gin.Context) {
	artifactID := c.Param("artifact_id")

	a, err := h.service.GetByID(c.Request.Context(), artifactID)
	if err != nil {
		responses.HandleError(c, err, "failed to get artifact")
		return
	}

	c.JSON(http.StatusOK, responses.MapArtifactToResponse(a))
}

// GetByResponse handles GET /v1/responses/:response_id/artifacts
// @Summary List artifacts for a response
// @Description Retrieves all artifacts associated with a response with cursor-based pagination
// @Tags Artifacts
// @Produce json
// @Param response_id path string true "Response ID"
// @Param latest query bool false "Only return latest versions"
// @Param limit query int false "Maximum number of results" default(20)
// @Param after query string false "Return artifacts after the given artifact ID (cursor pagination)"
// @Param order query string false "Sort order: asc or desc" default(desc)
// @Success 200 {object} responses.ArtifactListResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/artifacts [get]
func (h *ArtifactHandler) GetByResponse(c *gin.Context) {
	responseID := c.Param("response_id")

	filter := artifact.NewFilter().WithResponseID(responseID)

	// Parse query params
	if latestStr := c.Query("latest"); latestStr == "true" {
		filter = filter.WithLatestOnly()
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Parse cursor-based pagination params
	if afterStr := c.Query("after"); afterStr != "" {
		internalID, err := h.service.ResolveInternalID(c.Request.Context(), afterStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination cursor"})
			return
		}
		filter.After = internalID
	}

	if order := c.Query("order"); order == "asc" || order == "desc" {
		filter.Order = order
	}

	artifacts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		responses.HandleError(c, err, "failed to list artifacts")
		return
	}

	// Determine has_more and trim to limit
	hasMore := len(artifacts) > filter.Limit
	if hasMore {
		artifacts = artifacts[:filter.Limit]
	}

	// Build response with cursor info
	data := responses.MapArtifactsToResponse(artifacts)
	firstID := ""
	lastID := ""
	if len(data) > 0 {
		firstID = data[0].ID
		lastID = data[len(data)-1].ID
	}

	c.JSON(http.StatusOK, responses.ArtifactListResponse{
		Object:  "list",
		Data:    data,
		FirstID: firstID,
		LastID:  lastID,
		HasMore: hasMore,
		Total:   total,
	})
}

// GetLatestByResponse handles GET /v1/responses/:response_id/artifacts/latest
// @Summary Get latest artifact for a response
// @Description Retrieves the most recent artifact for a response
// @Tags Artifacts
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ArtifactResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/artifacts/latest [get]
func (h *ArtifactHandler) GetLatestByResponse(c *gin.Context) {
	responseID := c.Param("response_id")

	a, err := h.service.GetLatestByResponseID(c.Request.Context(), responseID)
	if err != nil {
		responses.HandleError(c, err, "failed to get latest artifact")
		return
	}

	c.JSON(http.StatusOK, responses.MapArtifactToResponse(a))
}

// GetVersions handles GET /v1/artifacts/:artifact_id/versions
// @Summary Get artifact versions
// @Description Retrieves all versions of an artifact
// @Tags Artifacts
// @Produce json
// @Param artifact_id path string true "Artifact ID"
// @Success 200 {array} responses.ArtifactResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/artifacts/{artifact_id}/versions [get]
func (h *ArtifactHandler) GetVersions(c *gin.Context) {
	artifactID := c.Param("artifact_id")

	versions, err := h.service.GetVersions(c.Request.Context(), artifactID)
	if err != nil {
		responses.HandleError(c, err, "failed to get artifact versions")
		return
	}

	c.JSON(http.StatusOK, responses.MapArtifactsToResponse(versions))
}

// Download handles GET /v1/artifacts/:artifact_id/download
// @Summary Download artifact content
// @Description Downloads the artifact content with appropriate content type
// @Tags Artifacts
// @Produce application/octet-stream
// @Param artifact_id path string true "Artifact ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/artifacts/{artifact_id}/download [get]
func (h *ArtifactHandler) Download(c *gin.Context) {
	artifactID := c.Param("artifact_id")

	a, err := h.service.GetByID(c.Request.Context(), artifactID)
	if err != nil {
		responses.HandleError(c, err, "failed to get artifact for download")
		return
	}

	// Set content headers
	c.Header("Content-Type", a.MimeType)
	c.Header("Content-Disposition", "attachment; filename=\""+a.Title+"\"")

	if a.HasInlineContent() && a.Content != nil {
		c.String(http.StatusOK, *a.Content)
		return
	}

	if a.HasStoredContent() && a.StoragePath != nil {
		// StoragePath contains the media-api download URL - redirect to it
		storagePath := *a.StoragePath
		if strings.HasPrefix(storagePath, "http://") || strings.HasPrefix(storagePath, "https://") {
			c.Redirect(http.StatusTemporaryRedirect, storagePath)
			return
		}
		// For non-URL paths, return the path info (legacy behavior)
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":        "file download not yet implemented",
			"storage_path": storagePath,
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "artifact has no content"})
}

// Delete handles DELETE /v1/artifacts/:artifact_id
// @Summary Delete artifact
// @Description Deletes an artifact by ID
// @Tags Artifacts
// @Param artifact_id path string true "Artifact ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/artifacts/{artifact_id} [delete]
func (h *ArtifactHandler) Delete(c *gin.Context) {
	artifactID := c.Param("artifact_id")

	if err := h.service.Delete(c.Request.Context(), artifactID); err != nil {
		responses.HandleError(c, err, "failed to delete artifact")
		return
	}

	c.Status(http.StatusNoContent)
}

// extractUserIDFromToken extracts the user ID (sub claim) from the JWT token
func extractUserIDFromToken(c *gin.Context) string {
	tokenValue, exists := c.Get("auth_token")
	if !exists {
		return ""
	}
	token, ok := tokenValue.(*jwt.Token)
	if !ok {
		return ""
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if sub, ok := claims["sub"].(string); ok {
			return sub
		}
	}
	return ""
}
