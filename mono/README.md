# Jan Server Mono

A unified, enterprise-grade backend API platform for LLM applications. This monorepo consolidates all backend services into a single Go application with OpenAI-compatible APIs, multi-provider support, and comprehensive tool integrations.

## Overview

Jan Server Mono provides:

- **OpenAI-Compatible Chat API** - Drop-in replacement for `/v1/chat/completions`
- **Multi-Provider Support** - OpenAI, Anthropic, Google, Azure, Ollama, and more
- **Streaming Support** - Server-sent events for real-time responses
- **Authentication** - JWT-based auth with local password and API key support
- **Conversation Management** - Persistent chat history with branching and sharing
- **Artifact System** - Code/content artifacts with versioning
- **Media Storage** - S3-compatible file uploads with presigned URLs
- **OAuth Connectors** - GitHub, Gmail, Google Drive, Google Calendar
- **MCP Protocol** - Model Context Protocol for tool integration
- **Response API** - Multi-step tool orchestration (OpenAI Responses API compatible)

## Architecture

```
mono/
├── apps/
│   ├── backend/                    # Go API Server
│   │   ├── cmd/server/             # Application entrypoint
│   │   ├── internal/
│   │   │   ├── domain/             # Business logic layer
│   │   │   │   ├── user/           # User & authentication
│   │   │   │   ├── conversation/   # Chat conversations
│   │   │   │   ├── model/          # LLM models & providers
│   │   │   │   ├── artifact/       # Code artifacts
│   │   │   │   ├── media/          # File storage
│   │   │   │   └── connector/      # OAuth connectors
│   │   │   ├── infrastructure/
│   │   │   │   ├── config/         # Configuration management
│   │   │   │   └── database/       # GORM schemas & repositories
│   │   │   └── interfaces/
│   │   │       └── httpserver/     # HTTP handlers & middleware
│   │   ├── pkg/common/             # Shared utilities
│   │   ├── migrations/             # SQL migrations
│   │   └── tests/                  # Integration tests
│   └── web/                        # React frontend (Vite)
├── scripts/                        # Development scripts
├── docker-compose.yml              # Container orchestration
└── Makefile                        # Build automation
```

### Clean Architecture

The backend follows Clean Architecture principles:

```
┌─────────────────────────────────────────────────┐
│              Interfaces Layer                    │
│    (HTTP handlers, routes, middleware)          │
├─────────────────────────────────────────────────┤
│               Domain Layer                       │
│    (entities, services, business logic)         │
├─────────────────────────────────────────────────┤
│           Infrastructure Layer                   │
│    (database, external APIs, config)            │
└─────────────────────────────────────────────────┘
```

**Key Rules:**
- Domain layer has NO external dependencies
- Infrastructure implements domain interfaces
- HTTP handlers are thin - just DTO conversion and service calls

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for local development)

### Setup

```bash
# Clone repository
cd mono

# Create environment file
cp .env.template .env
# Edit .env with your configuration

# Start all services
docker compose up -d

# Check health
curl http://localhost:8080/healthz
```

### Development

```bash
# Start infrastructure only
docker compose up -d postgres minio

# Run backend locally with hot reload
cd apps/backend
make dev

# Run tests
make test

# Format code
make fmt
```

## API Reference

### Health Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |

### Authentication

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/local/register` | Register new user |
| POST | `/v1/auth/local/login` | Login with email/password |
| POST | `/v1/auth/local/refresh` | Refresh access token |
| POST | `/v1/auth/logout` | Logout (invalidate tokens) |
| GET | `/v1/auth/me` | Get current user |
| POST | `/v1/auth/api-keys` | Create API key |
| GET | `/v1/auth/api-keys` | List API keys |
| DELETE | `/v1/auth/api-keys/:id` | Delete API key |

### Chat Completions (OpenAI-Compatible)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/chat/completions` | Create chat completion |

**Request:**
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

### Conversations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/conversations` | List conversations |
| POST | `/v1/conversations` | Create conversation |
| GET | `/v1/conversations/:id` | Get conversation |
| PUT | `/v1/conversations/:id` | Update conversation |
| DELETE | `/v1/conversations/:id` | Delete conversation |
| POST | `/v1/conversations/:id/branch` | Branch conversation |

### Messages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/messages` | List messages (with conversation_id query) |
| GET | `/v1/messages/:id` | Get message |

### Models & Providers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/models` | List available models |
| GET | `/v1/models/:id` | Get model details |
| GET | `/v1/providers` | List providers |
| GET | `/v1/providers/:id` | Get provider details |

### Artifacts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/artifacts` | List artifacts |
| POST | `/v1/artifacts` | Create artifact |
| GET | `/v1/artifacts/:id` | Get artifact |
| PUT | `/v1/artifacts/:id` | Update artifact |
| DELETE | `/v1/artifacts/:id` | Delete artifact |
| GET | `/v1/artifacts/:id/versions` | List versions |
| GET | `/v1/artifacts/:id/download` | Download artifact |

### Media/Files

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/media` | Upload file |
| POST | `/v1/media/upload` | Get presigned upload URL |
| GET | `/v1/media/:id` | Get file |
| GET | `/v1/media/:id/metadata` | Get file metadata |

### Response API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/responses` | Create response |
| GET | `/v1/responses/:id` | Get response |
| DELETE | `/v1/responses/:id` | Delete response |
| POST | `/v1/responses/:id/cancel` | Cancel response |
| POST | `/v1/responses/:id/retry` | Retry response |

### Connectors (OAuth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/connectors` | List connected accounts |
| GET | `/v1/connectors/:provider/auth` | Start OAuth flow |
| GET | `/v1/connectors/:provider/callback` | OAuth callback |
| DELETE | `/v1/connectors/:provider` | Disconnect account |

### Admin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/admin/providers` | Create provider |
| PUT | `/v1/admin/providers/:id` | Update provider |
| DELETE | `/v1/admin/providers/:id` | Delete provider |
| POST | `/v1/admin/models` | Create model |
| PUT | `/v1/admin/models/:id` | Update model |
| DELETE | `/v1/admin/models/:id` | Delete model |
| GET | `/v1/admin/users` | List users |
| GET | `/v1/admin/users/:id` | Get user |
| PUT | `/v1/admin/users/:id` | Update user |
| DELETE | `/v1/admin/users/:id` | Delete user |

### MCP (Model Context Protocol)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mcp` | MCP JSON-RPC endpoint |
| GET | `/mcp` | MCP endpoint (WebSocket upgrade) |

## Configuration

### Environment Variables

#### Server
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | `8080` |
| `METRICS_PORT` | Metrics server port | `9091` |
| `ENVIRONMENT` | Environment name | `development` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |

#### Database
| Variable | Description | Default |
|----------|-------------|---------|
| `DB_POSTGRESQL_WRITE_DSN` | PostgreSQL connection string | Required |

#### Authentication
| Variable | Description | Default |
|----------|-------------|---------|
| `LOCAL_AUTH_ENABLED` | Enable local auth | `true` |
| `LOCAL_JWT_SECRET` | JWT signing secret | Required |
| `LOCAL_JWT_EXPIRATION` | Token expiration | `24h` |
| `KEYCLOAK_ENABLED` | Enable Keycloak OIDC | `false` |

#### Storage
| Variable | Description | Default |
|----------|-------------|---------|
| `S3_ENDPOINT` | S3/MinIO endpoint | Required |
| `S3_ACCESS_KEY_ID` | S3 access key | Required |
| `S3_SECRET_ACCESS_KEY` | S3 secret key | Required |
| `S3_BUCKET` | S3 bucket name | `jan-media` |

#### Optional Features
| Variable | Description | Default |
|----------|-------------|---------|
| `MEMORY_ENABLED` | Enable memory service | `false` |
| `REALTIME_ENABLED` | Enable realtime (LiveKit) | `false` |
| `GITHUB_CONNECTOR_ENABLED` | Enable GitHub OAuth | `false` |
| `GOOGLE_CONNECTOR_ENABLED` | Enable Google OAuth | `false` |

## Database Schema

The application uses PostgreSQL with GORM. Tables include:

- `users` - User accounts
- `api_keys` - API key credentials
- `refresh_tokens` - Session tokens
- `providers` - LLM provider configurations
- `models` - Model definitions
- `conversations` - Chat conversations
- `messages` - Chat messages
- `artifacts` - Code/content artifacts
- `artifact_versions` - Artifact history
- `media` - File metadata
- `connectors` - OAuth connections
- `connector_oauth_states` - OAuth state tokens

## Docker Services

| Service | Port | Description |
|---------|------|-------------|
| `backend` | 8080 | Go API server |
| `postgres` | 5432 | PostgreSQL database |
| `minio` | 9000/9001 | S3-compatible storage |
| `redis` | 6379 | Cache (optional) |
| `keycloak` | 8085 | OIDC provider (optional) |
| `web` | 3001 | React frontend |

## Development

### Project Structure

```
apps/backend/
├── cmd/server/main.go           # Entrypoint
├── internal/
│   ├── domain/                  # Business logic
│   │   └── {entity}/
│   │       ├── entity.go        # Domain types
│   │       └── service.go       # Business logic
│   ├── infrastructure/
│   │   ├── config/              # Configuration
│   │   └── database/
│   │       ├── dbschema/        # GORM models
│   │       └── repository/      # Data access
│   └── interfaces/httpserver/
│       ├── routes/              # HTTP handlers
│       ├── middlewares/         # Auth, logging, etc.
│       └── server.go            # Server setup
├── pkg/common/                  # Shared utilities
├── migrations/                  # SQL migrations
└── tests/                       # Integration tests
```

### Adding a New Domain

1. Create `internal/domain/{name}/entity.go` with types
2. Create `internal/domain/{name}/service.go` with business logic
3. Add GORM schema in `internal/infrastructure/database/dbschema/`
4. Add repository in `internal/infrastructure/database/repository/`
5. Add HTTP handlers in `internal/interfaces/httpserver/routes/`
6. Register routes in `routes.go`
7. Add migration in `migrations/`

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/domain/user/...
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feat/amazing-feature`)
3. Commit changes (`git commit -m 'feat: add amazing feature'`)
4. Push branch (`git push origin feat/amazing-feature`)
5. Open Pull Request
