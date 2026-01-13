package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
)

var errStorageDisabled = errors.New("media storage backend is not configured; set MEDIA_S3_* to enable uploads")

// S3Storage handles uploads and downloads to S3-compatible storage.
type S3Storage struct {
	bucket   string
	client   *s3.Client
	log      zerolog.Logger
	disabled bool
}

func NewS3Storage(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*S3Storage, error) {
	logger := log.With().Str("component", "s3-storage").Logger()
	storage := &S3Storage{
		bucket: strings.TrimSpace(cfg.S3Bucket),
		log:    logger,
	}

	accessKey := strings.TrimSpace(cfg.S3AccessKeyID)
	secretKey := strings.TrimSpace(cfg.S3SecretKey)
	if storage.bucket == "" || accessKey == "" || secretKey == "" {
		logger.Warn().Msg("MEDIA_S3_BUCKET or credentials are not set; media uploads will be disabled until configured")
		storage.disabled = true
		return storage, nil
	}

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3Endpoint != "" {
			return aws.Endpoint{
				URL:           cfg.S3Endpoint,
				PartitionID:   "aws",
				SigningRegion: cfg.S3Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretKey, "")),
		awsconfig.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	storage.client = client
	return storage, nil
}

func (s *S3Storage) ensureEnabled() error {
	if s.disabled {
		return errStorageDisabled
	}
	return nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return err
	}
	return nil
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, "", err
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	mime := ""
	if out.ContentType != nil {
		mime = *out.ContentType
	}
	return out.Body, mime, nil
}

// Health performs a simple HeadObject request.
func (s *S3Storage) Health(ctx context.Context) error {
	if s.disabled {
		return nil
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}
