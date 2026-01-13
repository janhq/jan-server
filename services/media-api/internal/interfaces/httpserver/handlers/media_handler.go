package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
	"jan-server/services/media-api/internal/interfaces/httpserver/responses"
	"jan-server/services/media-api/internal/utils/platformerrors"
)

// MediaHandler exposes media endpoints.
type MediaHandler struct {
	cfg     *config.Config
	service *domain.Service
	log     zerolog.Logger
}

func NewMediaHandler(cfg *config.Config, service *domain.Service, log zerolog.Logger) *MediaHandler {
	return &MediaHandler{
		cfg:     cfg,
		service: service,
		log:     log.With().Str("component", "media-handler").Logger(),
	}
}

type ingestResponse struct {
	ID      string `json:"id"`
	Mime    string `json:"mime"`
	Bytes   int64  `json:"bytes"`
	Deduped bool   `json:"deduped"`
	URL     string `json:"url"`
}

// Ingest godoc
// @Summary      Upload media
// @Description  Accepts data URLs or remote URLs and stores content privately.
// @Tags         media
// @Accept       json
// @Produce      json
// @Param        request  body      domain.IngestRequest  true  "Media request"
// @Success      200      {object}  ingestResponse
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media [post]
func (h *MediaHandler) Ingest(c *gin.Context) {
	var req domain.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(c, platformerrors.ErrorTypeValidation, "invalid request body", "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
		return
	}

	obj, dedup, err := h.service.Ingest(c.Request.Context(), req)
	if err != nil {
		h.log.Error().Err(err).Msg("ingest failed")
		responses.HandleError(c, err, "failed to ingest media")
		return
	}

	// Generate direct public URL for embedding in HTML
	directURL := h.buildMediaURL(obj)

	c.JSON(http.StatusOK, ingestResponse{
		ID:      obj.ID,
		Mime:    obj.MimeType,
		Bytes:   obj.Bytes,
		Deduped: dedup,
		URL:     directURL,
	})
}

// Proxy godoc
// @Summary      Stream media bytes
// @Description  Streams the object through the media API without exposing storage URLs. If proxying is disabled, returns a direct URL instead.
// @Tags         media
// @Produce      json
// @Produce      octet-stream
// @Param        id   path      string  true  "Media ID"
// @Success      200  {object}  map[string]interface{} "Direct URL response when proxying is disabled; otherwise binary stream"
// @Failure      404  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media/{id} [get]
func (h *MediaHandler) Proxy(c *gin.Context) {
	id := c.Param("id")

	if !h.cfg.ProxyDownload {
		obj, err := h.service.Get(c.Request.Context(), id)
		if err != nil {
			h.log.Error().Err(err).Msg("lookup failed")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": h.buildMediaURL(obj)})
		return
	}

	reader, mime, err := h.service.Download(c.Request.Context(), id)
	if err != nil {
		h.log.Error().Err(err).Msg("download failed")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	if mime == "" {
		mime = "application/octet-stream"
	}

	c.Header("Content-Type", mime)
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		h.log.Error().Err(err).Msg("stream error")
	}
}

// DirectUpload godoc
// @Summary      Direct file upload
// @Description  Accepts multipart file upload for local storage.
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData  file    true  "File to upload"
// @Param        user_id   formData  string  false "User ID"
// @Success      200       {object}  ingestResponse
// @Failure      400       {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media/upload [post]
func (h *MediaHandler) DirectUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	userID := c.Request.FormValue("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to read file")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}

	// Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Create an ingest request with data URL
	dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)

	req := domain.IngestRequest{
		Source: domain.Source{
			Type:    "data_url",
			DataURL: dataURL,
		},
		Filename: header.Filename,
		UserID:   userID,
	}

	obj, dedup, err := h.service.Ingest(c.Request.Context(), req)
	if err != nil {
		h.log.Error().Err(err).Msg("ingest failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate direct public URL for embedding in HTML
	directURL := h.buildMediaURL(obj)

	c.JSON(http.StatusOK, ingestResponse{
		ID:      obj.ID,
		Mime:    obj.MimeType,
		Bytes:   obj.Bytes,
		Deduped: dedup,
		URL:     directURL,
	})
}

// PublicServe godoc
// @Summary      Serve media publicly
// @Description  Streams the media file directly for use in HTML img src. This endpoint does not require authentication.
// @Tags         media
// @Produce      image/jpeg
// @Produce      image/png
// @Produce      image/webp
// @Produce      image/gif
// @Param        id   path      string  true  "Media ID"
// @Success      200  {file}    binary
// @Failure      404  {object}  map[string]string
// @Router       /api/media/{id} [get]
func (h *MediaHandler) PublicServe(c *gin.Context) {
	id := c.Param("id")

	reader, mime, err := h.service.Download(c.Request.Context(), id)
	if err != nil {
		h.log.Error().Err(err).Str("id", id).Msg("public serve failed")
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}
	defer reader.Close()

	if mime == "" {
		mime = "application/octet-stream"
	}

	// Set cache headers for browser caching
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Type", mime)
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		h.log.Error().Err(err).Msg("stream error")
	}
}

// buildDirectURL constructs the public URL for direct media access
func (h *MediaHandler) buildMediaURL(obj *domain.MediaObject) string {
	if h.cfg.S3URLEnabled && h.cfg.IsS3Storage() {
		publicEndpoint := strings.TrimSpace(h.cfg.S3PublicEndpoint)
		if publicEndpoint != "" && strings.TrimSpace(obj.StorageKey) != "" {
			base := strings.TrimSuffix(publicEndpoint, "/")
			key := strings.TrimPrefix(strings.TrimSpace(obj.StorageKey), "/")
			return fmt.Sprintf("%s/%s", base, key)
		}
	}
	return h.buildDirectURL(obj.ID)
}

func (h *MediaHandler) buildDirectURL(id string) string {
	publicURL := h.cfg.PublicURL
	if publicURL == "" {
		// Fallback to localhost if not configured
		publicURL = "http://localhost:8000"
	}
	return fmt.Sprintf("%s/api/media/%s", strings.TrimSuffix(publicURL, "/"), id)
}
