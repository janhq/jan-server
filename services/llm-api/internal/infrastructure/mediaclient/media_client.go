package mediaclient

import (
	"context"
	"encoding/json"
	"fmt"
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
