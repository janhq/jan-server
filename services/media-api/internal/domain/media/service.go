package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
	"jan-server/services/media-api/internal/utils/platformerrors"
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

// Repository defines persistence operations needed by the service.
type Repository interface {
	FindByHash(ctx context.Context, hash string) (*MediaObject, error)
	Create(ctx context.Context, obj *MediaObject) error
	GetByID(ctx context.Context, id string) (*MediaObject, error)
}

// Storage defines media storage operations.
type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
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

// Get returns media metadata.
func (s *Service) Get(ctx context.Context, id string) (*MediaObject, error) {
	obj, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("media %s not found", id)
	}
	return obj, nil
}

func (s *Service) loadBytes(ctx context.Context, source Source) ([]byte, error) {
	switch strings.ToLower(source.Type) {
	case "data_url", "datauri", "dataurl":
		return decodeDataURL(source.DataURL)
	case "remote_url", "remoteuri", "remote":
		return s.fetchRemote(ctx, source.URL)
	default:
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, fmt.Sprintf("unknown source type %s", source.Type), nil, "")
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
