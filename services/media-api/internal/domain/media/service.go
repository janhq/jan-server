package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
	"jan-server/services/media-api/utils/mediaid"
)

var allowedMIMEs = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
	"image/bmp":  "bmp",
	"image/tiff": "tiff",
}

var placeholderPattern = regexp.MustCompile(`data:(image/[a-z0-9.+-]+);(jan_[A-Za-z0-9]+)`)

// Repository defines persistence operations needed by the service.
type Repository interface {
	FindByHash(ctx context.Context, hash string) (*MediaObject, error)
	Create(ctx context.Context, obj *MediaObject) error
	GetByID(ctx context.Context, id string) (*MediaObject, error)
}

// Storage defines media storage operations.
type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignPut(ctx context.Context, key string, contentType string, ttl time.Duration) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// Service orchestrates media ingestion and retrieval.
type Service struct {
	cfg        *config.Config
	repo       Repository
	storage    Storage
	log        zerolog.Logger
	httpClient *http.Client
}

func NewService(cfg *config.Config, repo Repository, storage Storage, log zerolog.Logger) *Service {
	return &Service{
		cfg:     cfg,
		repo:    repo,
		storage: storage,
		log:     log.With().Str("component", "media-service").Logger(),
		httpClient: &http.Client{
			Timeout: cfg.RemoteFetchTimeout,
		},
	}
}

// Ingest stores media and returns metadata. bool indicates whether content was deduplicated.
func (s *Service) Ingest(ctx context.Context, req IngestRequest) (*MediaObject, bool, error) {
	data, err := s.loadBytes(ctx, req.Source)
	if err != nil {
		return nil, false, err
	}

	if int64(len(data)) == 0 {
		return nil, false, errors.New("file is empty")
	}
	if int64(len(data)) > s.cfg.MaxMediaBytes {
		return nil, false, fmt.Errorf("file exceeds max size of %d bytes", s.cfg.MaxMediaBytes)
	}

	mimeType := mimetype.Detect(data).String()
	ext, ok := allowedMIMEs[mimeType]
	if !ok {
		return nil, false, fmt.Errorf("unsupported mime type %s", mimeType)
	}

	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum[:])

	if existing, err := s.repo.FindByHash(ctx, hash); err != nil {
		return nil, false, err
	} else if existing != nil {
		return existing, true, nil
	}

	id := mediaid.New()
	key := fmt.Sprintf("images/%s.%s", id, ext)

	if err := s.storage.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		return nil, false, err
	}

	obj := &MediaObject{
		ID:              id,
		StorageProvider: "s3",
		StorageKey:      key,
		MimeType:        mimeType,
		Bytes:           int64(len(data)),
		Sha256:          hash,
		CreatedBy:       req.UserID,
		RetentionUntil:  time.Now().Add(time.Duration(s.cfg.RetentionDays) * 24 * time.Hour),
	}

	if err := s.repo.Create(ctx, obj); err != nil {
		return nil, false, err
	}

	return obj, false, nil
}

// ResolvePayload replaces jan_* placeholders with presigned URLs.
func (s *Service) ResolvePayload(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	text := string(payload)
	matches := placeholderPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return payload, nil
	}

	replacements := make(map[string]string)
	for _, match := range matches {
		token := match[2]
		if _, exists := replacements[token]; exists {
			continue
		}

		obj, err := s.repo.GetByID(ctx, token)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return nil, fmt.Errorf("unknown media id %s", token)
		}

		url, err := s.storage.PresignGet(ctx, obj.StorageKey, s.cfg.S3PresignTTL)
		if err != nil {
			return nil, err
		}

		replacements[match[0]] = url
	}

	builder := strings.Builder{}
	builder.Grow(len(text))
	lastIndex := 0
	indices := placeholderPattern.FindAllStringIndex(text, -1)
	for i, match := range matches {
		start, end := indices[i][0], indices[i][1]
		builder.WriteString(text[lastIndex:start])
		builder.WriteString(replacements[match[0]])
		lastIndex = end
	}
	builder.WriteString(text[lastIndex:])

	return json.RawMessage([]byte(builder.String())), nil
}

// Download fetches object contents for proxying.
func (s *Service) Download(ctx context.Context, id string) (io.ReadCloser, string, error) {
	obj, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if obj == nil {
		return nil, "", fmt.Errorf("media %s not found", id)
	}
	reader, mime, err := s.storage.Download(ctx, obj.StorageKey)
	if err != nil {
		return nil, "", err
	}
	if mime == "" {
		mime = obj.MimeType
	}
	return reader, mime, nil
}

// Presign returns a short-lived URL for the media object.
func (s *Service) Presign(ctx context.Context, id string) (string, error) {
	obj, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if obj == nil {
		return "", fmt.Errorf("media %s not found", id)
	}
	return s.storage.PresignGet(ctx, obj.StorageKey, s.cfg.S3PresignTTL)
}

// PrepareUpload generates a presigned upload URL and reserves a jan_id for client-side upload.
func (s *Service) PrepareUpload(ctx context.Context, mimeType string, userID string) (*UploadPreparation, error) {
	// Validate MIME type
	ext, ok := allowedMIMEs[mimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported mime type %s", mimeType)
	}

	// Generate jan_id and storage key
	id := mediaid.New()
	key := fmt.Sprintf("images/%s.%s", id, ext)

	// Generate presigned PUT URL
	uploadURL, err := s.storage.PresignPut(ctx, key, mimeType, s.cfg.S3PresignTTL)
	if err != nil {
		return nil, err
	}

	// Create placeholder record in database (with zero bytes initially)
	obj := &MediaObject{
		ID:              id,
		StorageProvider: "s3",
		StorageKey:      key,
		MimeType:        mimeType,
		Bytes:           0,  // Will be updated after upload
		Sha256:          "", // Will be updated after upload
		CreatedBy:       userID,
		RetentionUntil:  time.Now().Add(time.Duration(s.cfg.RetentionDays) * 24 * time.Hour),
	}

	if err := s.repo.Create(ctx, obj); err != nil {
		return nil, err
	}

	return &UploadPreparation{
		ID:        id,
		UploadURL: uploadURL,
		MimeType:  mimeType,
		ExpiresIn: int(s.cfg.S3PresignTTL.Seconds()),
	}, nil
}

func (s *Service) loadBytes(ctx context.Context, source Source) ([]byte, error) {
	switch strings.ToLower(source.Type) {
	case "data_url", "datauri", "dataurl":
		return decodeDataURL(source.DataURL)
	case "remote_url", "remoteuri", "remote":
		return s.fetchRemote(ctx, source.URL)
	default:
		return nil, fmt.Errorf("unknown source type %s", source.Type)
	}
}

func decodeDataURL(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("data_url is required")
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid data url")
	}
	if !strings.Contains(parts[0], ";base64") {
		return nil, errors.New("data url must be base64 encoded")
	}
	return base64.StdEncoding.DecodeString(parts[1])
}

func (s *Service) fetchRemote(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, errors.New("url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("remote fetch error: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, s.cfg.MaxMediaBytes+1))
	if err != nil {
		return nil, err
	}
	return data, nil
}
