package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
// @Description Retrieves all artifacts associated with a response
// @Tags Artifacts
// @Produce json
// @Param response_id path string true "Response ID"
// @Param latest query bool false "Only return latest versions"
// @Param limit query int false "Maximum number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
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

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	artifacts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		responses.HandleError(c, err, "failed to list artifacts")
		return
	}

	c.JSON(http.StatusOK, responses.ArtifactListResponse{
		Data:   responses.MapArtifactsToResponse(artifacts),
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
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
		// For stored content, would need to stream from storage
		// This is a placeholder - actual implementation would use storage service
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":        "file download not yet implemented",
			"storage_path": *a.StoragePath,
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
