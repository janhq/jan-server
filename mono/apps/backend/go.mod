module jan-server/mono/apps/backend

go 1.24.0

toolchain go1.24.7

require (
	// Core framework
	github.com/gin-gonic/gin v1.10.0
	github.com/google/wire v0.7.0

	// Configuration
	github.com/caarlos0/env/v10 v10.0.0
	github.com/joho/godotenv v1.5.1

	// Authentication
	github.com/MicahParks/keyfunc/v2 v2.1.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	golang.org/x/crypto v0.45.0

	// Database
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.26.0
	gorm.io/datatypes v1.2.7
	gorm.io/gen v0.3.27
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/lib/pq v1.10.9

	// HTTP client
	github.com/go-resty/resty/v2 v2.11.0
	github.com/imroc/req/v3 v3.45.0
	resty.dev/v3 v3.0.0-beta.3

	// AWS S3
	github.com/aws/aws-sdk-go-v2 v1.33.0
	github.com/aws/aws-sdk-go-v2/config v1.27.13
	github.com/aws/aws-sdk-go-v2/credentials v1.17.13
	github.com/aws/aws-sdk-go-v2/service/s3 v1.54.2

	// MCP Protocol
	github.com/modelcontextprotocol/go-sdk v1.1.0
	github.com/agent-infra/sandbox-sdk-go v0.0.2

	// OpenAI client
	github.com/sashabaranov/go-openai v1.41.2

	// Utilities
	github.com/google/uuid v1.6.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/shopspring/decimal v1.4.0
	github.com/gabriel-vasile/mimetype v1.4.3
	github.com/go-playground/validator/v10 v10.20.0
	github.com/mileusna/crontab v1.2.0

	// Logging
	github.com/rs/zerolog v1.33.0

	// Observability
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/sdk/metric v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0

	// Metrics
	github.com/prometheus/client_golang v1.23.2

	// Swagger
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.16.6

	// YAML
	gopkg.in/yaml.v3 v3.0.1

	// Sync
	golang.org/x/sync v0.18.0
	golang.org/x/net v0.47.0
	golang.org/x/oauth2 v0.30.0
)

replace go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.24.0

replace go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.24.0

replace go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.24.0

replace go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.24.0

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace => go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0

replace go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
