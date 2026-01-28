package mediaclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/config"
)

// Client handles media uploads to the media-api service.
type Client struct {
	cfg    *config.Config
	client *req.Client
	log    zerolog.Logger
}

// Source describes the media source.
type Source struct {
	Type    string `json:"type"`
	DataURL string `json:"data_url,omitempty"`
	URL     string `json:"url,omitempty"`
}

// IngestRequest is the request format for media ingestion.
type IngestRequest struct {
	Source   Source `json:"source"`
	Filename string `json:"filename,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

// IngestResponse is the response from media ingestion.
type IngestResponse struct {
	Mime    string `json:"mime"`  // MIME type
	Bytes   int64  `json:"bytes"` // Size in bytes
	Deduped bool   `json:"deduped"`
	URL     string `json:"url"` // Direct media URL
}

// NewClient creates a new media client.
func NewClient(cfg *config.Config, log zerolog.Logger) *Client {
	if cfg.MediaIngestURL == "" {
		log.Warn().Msg("[MediaClient] MediaIngestURL not configured, media uploads disabled")
		return nil
	}

	client := req.C().
		SetTimeout(30 * time.Second).
		SetCommonContentType("application/json")

	return &Client{
		cfg:    cfg,
		client: client,
		log:    log.With().Str("component", "media-client").Logger(),
	}
}

// UploadBase64Image uploads a base64-encoded image to media-api.
// Returns the direct media URL.
func (c *Client) UploadBase64Image(ctx context.Context, base64Data string, mimeType string, authHeader string) (*IngestResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("media client not configured")
	}

	// Build data URL
	if mimeType == "" {
		mimeType = "image/png"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	req := IngestRequest{
		Source: Source{
			Type:    "data_url",
			DataURL: dataURL,
		},
		Filename: fmt.Sprintf("generated_%d.png", time.Now().UnixNano()),
	}

	c.log.Debug().
		Str("mime_type", mimeType).
		Int("data_length", len(base64Data)).
		Msg("[MediaClient] Uploading image to media-api")

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", authHeader).
		SetBody(req).
		Post(c.cfg.MediaIngestURL)

	if err != nil {
		c.log.Error().Err(err).Msg("[MediaClient] Failed to upload image")
		return nil, fmt.Errorf("media upload failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.log.Error().
			Int("status", resp.StatusCode).
			Str("body", resp.String()).
			Msg("[MediaClient] Media API returned error")
		return nil, fmt.Errorf("media API returned status %d: %s", resp.StatusCode, resp.String())
	}

	var result IngestResponse
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		c.log.Error().Err(err).Str("body", resp.String()).Msg("[MediaClient] Failed to parse response")
		return nil, fmt.Errorf("failed to parse media response: %w", err)
	}

	c.log.Debug().
		Str("media_url", result.URL).
		Msg("[MediaClient] Image uploaded successfully")

	return &result, nil
}

// MediaInfo contains metadata about a media object
type MediaInfo struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
}

// Resolve retrieves metadata about a media object by its ID (jan_* ID)
func (c *Client) Resolve(ctx context.Context, mediaObjectID string, authHeader string) (*MediaInfo, error) {
	if c == nil {
		return nil, fmt.Errorf("media client not configured")
	}

	// The media API metadata endpoint: MEDIA_RESOLVE_URL/<media_object_id>/metadata
	metadataURL := buildMetadataURL(c.cfg.MediaResolveURL, mediaObjectID)

	c.log.Debug().
		Str("media_object_id", mediaObjectID).
		Str("metadata_url", metadataURL).
		Msg("[MediaClient] Getting media object metadata")

	req := c.client.R().
		SetContext(ctx)

	// Add auth header if provided
	if authHeader != "" {
		req.SetHeader("Authorization", authHeader)
	}

	resp, err := req.Get(metadataURL)

	if err != nil {
		c.log.Error().Err(err).Str("media_object_id", mediaObjectID).Msg("[MediaClient] Failed to resolve media object")
		return nil, fmt.Errorf("media resolve failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.log.Error().
			Int("status", resp.StatusCode).
			Str("body", resp.String()).
			Str("media_object_id", mediaObjectID).
			Msg("[MediaClient] Media API returned error on resolve")
		return nil, fmt.Errorf("media API returned status %d: %s", resp.StatusCode, resp.String())
	}

	var result MediaInfo
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		c.log.Error().Err(err).Str("body", resp.String()).Msg("[MediaClient] Failed to parse resolve response")
		return nil, fmt.Errorf("failed to parse media resolve response: %w", err)
	}

	c.log.Debug().
		Str("media_url", result.URL).
		Str("content_type", result.ContentType).
		Msg("[MediaClient] Media object resolved successfully")

	return &result, nil
}

func buildMetadataURL(baseURL string, mediaObjectID string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return fmt.Sprintf("%s/%s/metadata", trimmed, mediaObjectID)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Sprintf("%s/%s/metadata", strings.TrimSuffix(trimmed, "/"), mediaObjectID)
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	if strings.HasSuffix(path, "/resolve") {
		path = strings.TrimSuffix(path, "/resolve")
	}
	if path == "" {
		path = "/v1/media"
	}

	parsed.Path = fmt.Sprintf("%s/%s/metadata", strings.TrimSuffix(path, "/"), mediaObjectID)
	return parsed.String()
}
