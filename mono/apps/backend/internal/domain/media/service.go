package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMediaNotFound     = errors.New("media not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrFileTooLarge      = errors.New("file too large")
	ErrInvalidMimeType   = errors.New("invalid mime type")
	ErrUploadFailed      = errors.New("upload failed")
)

// Repository defines the interface for media data operations.
type Repository interface {
	Create(ctx context.Context, media *Media) error
	GetByID(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string, limit, offset int) ([]*Media, int64, error)
}

// StorageClient defines the interface for object storage operations.
type StorageClient interface {
	Upload(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetPresignedUploadURL(ctx context.Context, key string, contentType string, expires time.Duration) (string, error)
	GetPresignedDownloadURL(ctx context.Context, key string, expires time.Duration) (string, error)
}

// ServiceConfig holds configuration for the media service.
type ServiceConfig struct {
	Bucket        string
	MaxUploadSize int64
	PresignTTL    time.Duration
}

// Service handles media-related business logic.
type Service struct {
	repo    Repository
	storage StorageClient
	config  ServiceConfig
}

// NewService creates a new media service.
func NewService(repo Repository, storage StorageClient, config ServiceConfig) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		config:  config,
	}
}

// Upload uploads a file and creates a media record.
func (s *Service) Upload(ctx context.Context, userID string, req UploadRequest, data io.Reader) (*Media, error) {
	if req.Size > s.config.MaxUploadSize {
		return nil, ErrFileTooLarge
	}

	// Generate storage key
	id := uuid.New().String()
	ext := path.Ext(req.Filename)
	storageKey := fmt.Sprintf("%s/%s/%s%s", userID, req.Purpose, id, ext)

	// Calculate content hash while uploading
	hasher := sha256.New()
	teeReader := io.TeeReader(data, hasher)

	// Upload to storage
	if err := s.storage.Upload(ctx, storageKey, teeReader, req.Size, req.MimeType); err != nil {
		return nil, ErrUploadFailed
	}

	contentHash := hex.EncodeToString(hasher.Sum(nil))

	media := &Media{
		ID:           id,
		UserID:       userID,
		Filename:     sanitizeFilename(req.Filename),
		OriginalName: req.Filename,
		MimeType:     req.MimeType,
		Size:         req.Size,
		StorageKey:   storageKey,
		Bucket:       s.config.Bucket,
		ContentHash:  contentHash,
		Purpose:      req.Purpose,
		Metadata:     req.Metadata,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		// Try to clean up the uploaded file
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return media, nil
}

// GetPresignedUploadURL generates a presigned URL for direct upload.
func (s *Service) GetPresignedUploadURL(ctx context.Context, userID string, req PresignedUploadRequest) (*PresignedUploadResponse, error) {
	if req.Size > s.config.MaxUploadSize {
		return nil, ErrFileTooLarge
	}

	// Generate storage key
	id := uuid.New().String()
	ext := path.Ext(req.Filename)
	storageKey := fmt.Sprintf("%s/%s/%s%s", userID, req.Purpose, id, ext)

	// Get presigned URL
	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, storageKey, req.MimeType, s.config.PresignTTL)
	if err != nil {
		return nil, err
	}

	// Create media record (pending)
	media := &Media{
		ID:           id,
		UserID:       userID,
		Filename:     sanitizeFilename(req.Filename),
		OriginalName: req.Filename,
		MimeType:     req.MimeType,
		Size:         req.Size,
		StorageKey:   storageKey,
		Bucket:       s.config.Bucket,
		Purpose:      req.Purpose,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return nil, err
	}

	return &PresignedUploadResponse{
		MediaID:   id,
		UploadURL: uploadURL,
		Method:    "PUT",
		Headers: map[string]string{
			"Content-Type": req.MimeType,
		},
		ExpiresAt: time.Now().Add(s.config.PresignTTL),
	}, nil
}

// GetByID retrieves a media file with download URL.
func (s *Service) GetByID(ctx context.Context, userID, id string) (*Media, string, error) {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", ErrMediaNotFound
	}

	// Check ownership (unless it's a public file)
	if media.UserID != userID {
		return nil, "", ErrUnauthorized
	}

	// Generate presigned download URL
	url, err := s.storage.GetPresignedDownloadURL(ctx, media.StorageKey, s.config.PresignTTL)
	if err != nil {
		return nil, "", err
	}

	return media, url, nil
}

// GetMetadata retrieves media metadata without generating a download URL.
func (s *Service) GetMetadata(ctx context.Context, userID, id string) (*Media, error) {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrMediaNotFound
	}

	if media.UserID != userID {
		return nil, ErrUnauthorized
	}

	return media, nil
}

// Delete deletes a media file.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrMediaNotFound
	}

	if media.UserID != userID {
		return ErrUnauthorized
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, media.StorageKey); err != nil {
		// Log but don't fail - storage might be cleaned up separately
	}

	return s.repo.Delete(ctx, id)
}

// sanitizeFilename removes potentially dangerous characters from filenames.
func sanitizeFilename(filename string) string {
	// Remove path separators and null bytes
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Limit length
	if len(filename) > 255 {
		ext := path.Ext(filename)
		name := filename[:255-len(ext)]
		filename = name + ext
	}

	return filename
}
