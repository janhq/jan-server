module jan-server/mono/apps/backend

go 1.24.0

toolchain go1.24.7

require (
	// Authentication
	github.com/MicahParks/keyfunc/v2 v2.1.0
	github.com/agent-infra/sandbox-sdk-go v0.0.2

	// AWS S3
	github.com/aws/aws-sdk-go-v2 v1.33.0
	github.com/aws/aws-sdk-go-v2/config v1.27.13
	github.com/aws/aws-sdk-go-v2/credentials v1.17.13
	github.com/aws/aws-sdk-go-v2/service/s3 v1.54.2

	// Configuration
	github.com/caarlos0/env/v10 v10.0.0
	github.com/caarlos0/env/v11 v11.0.0
	github.com/gabriel-vasile/mimetype v1.4.3
	// Core framework
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.20.0
	github.com/go-redsync/redsync/v4 v4.15.0

	// HTTP client
	github.com/go-resty/resty/v2 v2.11.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/golang-migrate/migrate/v4 v4.17.1

	// Utilities
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.7.0
	github.com/hashicorp/golang-lru v1.0.2
	github.com/imroc/req/v3 v3.45.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9

	// LiveKit (WebRTC)
	github.com/livekit/protocol v1.24.0
	github.com/livekit/server-sdk-go/v2 v2.3.0
	github.com/mileusna/crontab v1.2.0

	// MCP Protocol
	github.com/modelcontextprotocol/go-sdk v1.1.0
	github.com/oklog/ulid/v2 v2.1.0

	// Metrics
	github.com/prometheus/client_golang v1.23.2

	// Redis
	github.com/redis/go-redis/v9 v9.17.2

	// Logging
	github.com/rs/zerolog v1.33.0

	// OpenAI client
	github.com/sashabaranov/go-openai v1.41.2
	github.com/shopspring/decimal v1.4.0

	// Swagger
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0

	// Observability
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/sdk/metric v1.34.0
	go.opentelemetry.io/otel/trace v1.35.0
	golang.org/x/crypto v0.45.0
	golang.org/x/net v0.47.0

	// Sync
	golang.org/x/sync v0.18.0

	// YAML
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/datatypes v1.2.5

	// Database
	gorm.io/driver/postgres v1.5.7
	gorm.io/gen v0.3.27
	gorm.io/gorm v1.26.0
	gorm.io/plugin/dbresolver v1.6.2
	resty.dev/v3 v3.0.0-beta.3
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.33.0-20240401165935-b983156c5e99.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.24.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.7 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bep/debounce v1.2.1 // indirect
	github.com/bufbuild/protovalidate-go v0.6.1 // indirect
	github.com/bufbuild/protoyaml-go v0.1.9 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.4.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-sql-driver/mysql v1.9.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/google/pprof v0.0.0-20240903155634-a8630aee4ab9 // indirect
	github.com/gorilla/websocket v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jxskiss/base62 v1.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lithammer/shortuuid/v4 v4.0.0 // indirect
	github.com/livekit/mageutil v0.0.0-20230125210925-54e8a70427c1 // indirect
	github.com/livekit/mediatransportutil v0.0.0-20240730083616-559fa5ece598 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/microsoft/go-mssqldb v1.8.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/ginkgo/v2 v2.20.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pion/datachannel v1.5.8 // indirect
	github.com/pion/dtls/v2 v2.2.12 // indirect
	github.com/pion/ice/v2 v2.3.34 // indirect
	github.com/pion/interceptor v0.1.30 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.12 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.14 // indirect
	github.com/pion/rtp v1.8.9 // indirect
	github.com/pion/sctp v1.8.19 // indirect
	github.com/pion/sdp/v3 v3.0.9 // indirect
	github.com/pion/srtp/v2 v2.0.20 // indirect
	github.com/pion/stun v0.6.1 // indirect
	github.com/pion/transport/v2 v2.2.10 // indirect
	github.com/pion/turn/v2 v2.1.6 // indirect
	github.com/pion/webrtc/v3 v3.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.1.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.47.0 // indirect
	github.com/refraction-networking/utls v1.6.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect

	// Testing
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/swaggo/swag v1.16.6
	github.com/twitchtv/twirp v8.1.3+incompatible // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/wlynxg/anet v0.0.3 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.uber.org/zap/exp v0.2.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/mysql v1.5.7 // indirect
	gorm.io/hints v1.1.0 // indirect
)

require github.com/posthog/posthog-go v1.9.1

require github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
