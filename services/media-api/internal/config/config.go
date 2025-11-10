package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds the environment driven configuration for the media service.
type Config struct {
	ServiceName        string        `env:"SERVICE_NAME" envDefault:"media-api"`
	Environment        string        `env:"ENVIRONMENT" envDefault:"development"`
	HTTPPort           int           `env:"MEDIA_API_PORT" envDefault:"8285"`
	LogLevel           string        `env:"LOG_LEVEL" envDefault:"info"`
	EnableTracing      bool          `env:"ENABLE_TRACING" envDefault:"false"`
	OTLPEndpoint       string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	ShutdownTimeout    time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	DatabaseURL        string        `env:"MEDIA_DATABASE_URL" envDefault:"postgres://media:media@localhost:5432/media_api?sslmode=disable"`
	DBMaxIdleConns     int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBMaxOpenConns     int           `env:"DB_MAX_OPEN_CONNS" envDefault:"15"`
	DBConnLifetime     time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`
	ServiceKey         string        `env:"MEDIA_SERVICE_KEY"`
	APIKey             string        `env:"MEDIA_API_KEY"`
	APIURL             string        `env:"MEDIA_API_URL"`
	S3Endpoint         string        `env:"MEDIA_S3_ENDPOINT" envDefault:"https://s3.menlo.ai"`
	S3PublicEndpoint   string        `env:"MEDIA_S3_PUBLIC_ENDPOINT"`
	S3Region           string        `env:"MEDIA_S3_REGION" envDefault:"us-west-2"`
	S3Bucket           string        `env:"MEDIA_S3_BUCKET"`
	S3AccessKey        string        `env:"MEDIA_S3_ACCESS_KEY"`
	S3SecretKey        string        `env:"MEDIA_S3_SECRET_KEY"`
	S3UsePathStyle     bool          `env:"MEDIA_S3_USE_PATH_STYLE" envDefault:"true"`
	S3PresignTTL       time.Duration `env:"MEDIA_S3_PRESIGN_TTL" envDefault:"5m"`
	MaxMediaBytes      int64         `env:"MEDIA_MAX_BYTES" envDefault:"20971520"`
	ProxyDownload      bool          `env:"MEDIA_PROXY_DOWNLOAD" envDefault:"true"`
	RetentionDays      int           `env:"MEDIA_RETENTION_DAYS" envDefault:"30"`
	RemoteFetchTimeout time.Duration `env:"MEDIA_REMOTE_FETCH_TIMEOUT" envDefault:"15s"`
	GCSBucket          string        `env:"MEDIA_GCS_BUCKET"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	cfg.S3Bucket = strings.TrimSpace(cfg.S3Bucket)
	cfg.S3AccessKey = strings.TrimSpace(cfg.S3AccessKey)
	cfg.S3SecretKey = strings.TrimSpace(cfg.S3SecretKey)
	cfg.S3Endpoint = strings.TrimSpace(cfg.S3Endpoint)
	cfg.S3PublicEndpoint = strings.TrimSpace(cfg.S3PublicEndpoint)
	if cfg.MaxMediaBytes <= 0 {
		cfg.MaxMediaBytes = 20 * 1024 * 1024
	}
	if cfg.ServiceKey == "" {
		cfg.ServiceKey = cfg.APIKey
	}
	return cfg, nil
}

// Addr returns the HTTP listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
