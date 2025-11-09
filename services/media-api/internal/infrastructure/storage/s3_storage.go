package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
)

// S3Storage handles uploads and downloads to S3-compatible storage.
type S3Storage struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
	log       zerolog.Logger
}

func NewS3Storage(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*S3Storage, error) {
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
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
		awsconfig.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	presigner := s3.NewPresignClient(client)

	return &S3Storage{
		bucket:    cfg.S3Bucket,
		client:    client,
		presigner: presigner,
		log:       log.With().Str("component", "s3-storage").Logger(),
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
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

func (s *S3Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	resp, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return resp.URL, nil
}

func (s *S3Storage) PresignPut(ctx context.Context, key string, contentType string, ttl time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}
	resp, err := s.presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return resp.URL, nil
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, string, error) {
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
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	return err
}
