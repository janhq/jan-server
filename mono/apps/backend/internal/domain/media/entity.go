package media

import (
	"time"
)

// Media represents an uploaded file.
type Media struct {
	ID           string
	UserID       string
	Filename     string
	OriginalName string
	MimeType     string
	Size         int64
	StorageKey   string
	Bucket       string
	ContentHash  string
	Metadata     map[string]any
	Purpose      string // attachment, avatar, artifact
	ExpiresAt    *time.Time
	CreatedAt    time.Time
}

// UploadRequest contains data for uploading a file.
type UploadRequest struct {
	Filename    string
	MimeType    string
	Size        int64
	Purpose     string
	ContentHash string
	Metadata    map[string]any
}

// PresignedUploadRequest contains data for getting a presigned upload URL.
type PresignedUploadRequest struct {
	Filename string
	MimeType string
	Size     int64
	Purpose  string
}

// PresignedUploadResponse contains the presigned upload URL and media ID.
type PresignedUploadResponse struct {
	MediaID   string            `json:"media_id"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// MediaResponse is the API response for a media file.
type MediaResponse struct {
	ID           string         `json:"id"`
	Filename     string         `json:"filename"`
	OriginalName string         `json:"original_name,omitempty"`
	MimeType     string         `json:"mime_type"`
	Size         int64          `json:"size"`
	Purpose      string         `json:"purpose,omitempty"`
	URL          string         `json:"url,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// MediaMetadataResponse is the API response for media metadata.
type MediaMetadataResponse struct {
	ID           string         `json:"id"`
	Filename     string         `json:"filename"`
	OriginalName string         `json:"original_name,omitempty"`
	MimeType     string         `json:"mime_type"`
	Size         int64          `json:"size"`
	ContentHash  string         `json:"content_hash,omitempty"`
	Purpose      string         `json:"purpose,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ToResponse converts a Media to MediaResponse.
func (m *Media) ToResponse(url string) MediaResponse {
	return MediaResponse{
		ID:           m.ID,
		Filename:     m.Filename,
		OriginalName: m.OriginalName,
		MimeType:     m.MimeType,
		Size:         m.Size,
		Purpose:      m.Purpose,
		URL:          url,
		Metadata:     m.Metadata,
		CreatedAt:    m.CreatedAt,
	}
}

// ToMetadataResponse converts a Media to MediaMetadataResponse.
func (m *Media) ToMetadataResponse() MediaMetadataResponse {
	return MediaMetadataResponse{
		ID:           m.ID,
		Filename:     m.Filename,
		OriginalName: m.OriginalName,
		MimeType:     m.MimeType,
		Size:         m.Size,
		ContentHash:  m.ContentHash,
		Purpose:      m.Purpose,
		Metadata:     m.Metadata,
		CreatedAt:    m.CreatedAt,
	}
}
