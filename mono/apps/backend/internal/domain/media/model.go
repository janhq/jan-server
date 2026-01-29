package media

import "time"

// MediaObject represents stored media metadata.
type MediaObject struct {
	ID              string    `json:"id"`
	StorageProvider string    `json:"storage_provider"`
	StorageKey      string    `json:"storage_key"`
	MimeType        string    `json:"mime"`
	Bytes           int64     `json:"bytes"`
	Sha256          string    `json:"sha256"`
	CreatedBy       string    `json:"created_by"`
	RetentionUntil  time.Time `json:"retention_until"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IngestRequest defines the payload for ingesting new media.
type IngestRequest struct {
	Source   Source `json:"source" binding:"required"`
	Filename string `json:"filename"`
	UserID   string `json:"user_id"`
}

// Source describes the media source.
type Source struct {
	Type    string `json:"type" binding:"required"`
	DataURL string `json:"data_url"`
	URL     string `json:"url"`
}
