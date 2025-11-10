package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
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
	ID           string `json:"id"`
	Mime         string `json:"mime"`
	Bytes        int64  `json:"bytes"`
	Deduped      bool   `json:"deduped"`
	PresignedURL string `json:"presigned_url,omitempty"`
}

type resolveRequest struct {
	Payload JSONPayload `json:"payload" binding:"required"`
}

type resolveResponse struct {
	Payload JSONPayload `json:"payload"`
}

// JSONPayload is used to document arbitrary JSON blobs in swagger.
type JSONPayload = json.RawMessage

// Ingest godoc
// @Summary      Upload media
// @Description  Accepts data URLs or remote URLs and stores content privately.
// @Tags         media
// @Accept       json
// @Produce      json
// @Param        request  body      domain.IngestRequest  true  "Media request"
// @Success      200      {object}  ingestResponse
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media [post]
func (h *MediaHandler) Ingest(c *gin.Context) {
	var req domain.IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	obj, dedup, err := h.service.Ingest(c.Request.Context(), req)
	if err != nil {
		h.log.Error().Err(err).Msg("ingest failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate presigned URL for immediate access
	presignedURL, err := h.service.Presign(c.Request.Context(), obj.ID)
	if err != nil {
		h.log.Warn().Err(err).Msg("failed to generate presigned URL, continuing without it")
		presignedURL = ""
	}

	c.JSON(http.StatusOK, ingestResponse{
		ID:           obj.ID,
		Mime:         obj.MimeType,
		Bytes:        obj.Bytes,
		Deduped:      dedup,
		PresignedURL: presignedURL,
	})
}

// Resolve godoc
// @Summary      Resolve jan_* placeholders
// @Description  Replaces pseudo data URLs with short-lived signed URLs.
// @Tags         media
// @Accept       json
// @Produce      json
// @Param        request  body      resolveRequest  true  "Payload to resolve"
// @Success      200      {object}  resolveResponse
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media/resolve [post]
func (h *MediaHandler) Resolve(c *gin.Context) {
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.service.ResolvePayload(c.Request.Context(), json.RawMessage(req.Payload))
	if err != nil {
		h.log.Error().Err(err).Msg("resolve failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resolveResponse{Payload: JSONPayload(out)})
}

// PrepareUpload godoc
// @Summary      Request presigned upload URL
// @Description  Generates a presigned upload URL and reserves a jan_id. Client uploads directly to S3 using the URL.
// @Tags         media
// @Accept       json
// @Produce      json
// @Param        request  body      domain.PrepareUploadRequest  true  "Upload preparation request"
// @Success      200      {object}  domain.UploadPreparation
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media/prepare-upload [post]
func (h *MediaHandler) PrepareUpload(c *gin.Context) {
	var req domain.PrepareUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prep, err := h.service.PrepareUpload(c.Request.Context(), req.MimeType, req.UserID)
	if err != nil {
		h.log.Error().Err(err).Msg("prepare upload failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prep)
}

// Proxy godoc
// @Summary      Stream media bytes
// @Description  Streams the object through the media API without exposing storage URLs.
// @Tags         media
// @Produce      octet-stream
// @Param        id   path      string  true  "Media ID"
// @Success      200  "binary data"
// @Failure      404  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /v1/media/{id} [get]
func (h *MediaHandler) Proxy(c *gin.Context) {
	id := c.Param("id")

	if !h.cfg.ProxyDownload {
		url, err := h.service.Presign(c.Request.Context(), id)
		if err != nil {
			h.log.Error().Err(err).Msg("presign failed")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
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
