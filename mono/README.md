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

## Quick Start

```bash
# Setup and start all services
make quickstart

# Or step by step:
make setup           # Create .env from template
make docker-up       # Start all services
make health-check    # Verify services are running
```

See [docs/quickstart.md](docs/quickstart.md) for detailed setup instructions.

## Architecture

```
mono/
├── apps/
│   ├── backend/                    # Go API Server
│   │   ├── cmd/server/             # Application entrypoint (wire DI)
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
├── tools/
│   └── jan-cli/                    # CLI tool for development & testing
├── tests/
│   └── e2e/                        # End-to-end API tests
│       └── automation/collections/ # Postman test collections
├── docs/                           # Documentation
├── scripts/                        # Development scripts
├── docker-compose.yml              # Container orchestration
└── Makefile                        # Build automation
```

### Clean Architecture

The backend follows Clean Architecture principles with Wire-based dependency injection:

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
- Wire generates dependency injection code at compile time

## Development

### Prerequisites

- Docker and Docker Compose
- Go 1.24+ (optional, for local development without Docker)

### Common Commands

```bash
# Quick Start
make quickstart        # Interactive setup and run
make setup             # Create .env from template
make docker-up         # Start all services
make docker-down       # Stop all services

# Development
make dev-backend       # Run backend with hot reload
make dev-web           # Run web frontend with hot reload

# Testing
make test              # Run unit tests
make test-all          # Run all API integration tests
make test-auth         # Run authentication tests only
make test-quick        # Quick smoke test

# CLI Tool
make cli-build         # Build jan-cli tool
make cli-deps          # Install CLI dependencies

# Code Quality
make fmt               # Format code
make lint              # Lint code
make swagger           # Generate API docs

# Database
make db-console        # Open PostgreSQL shell
make db-migrate        # Run migrations
make db-reset          # Reset database

# Status
make health-check      # Check service health
make status            # Show running containers

# Cleanup
make clean             # Clean build artifacts
make clean-all         # Clean everything including Docker
```

### jan-cli Tool

The `jan-cli` tool provides utilities for development and testing:

```bash
# Build the CLI
make cli-build

# Run API tests
./tools/jan-cli/jan-cli api-test run tests/e2e/automation/collections/auth.postman.json \
  --env-var "gateway_url=http://localhost:8080" \
  --auto-auth guest \
  --verbose

# Development setup
./tools/jan-cli/jan-cli dev setup
```

### Running Tests

```bash
# Unit tests
make test
make test-coverage

# API integration tests (requires running server)
make test-all          # All collections
make test-auth         # Authentication tests
make test-conversation # Conversation tests
make test-model        # Model tests
make test-media        # Media upload tests
make test-messages     # Messages tests
make test-image        # Image generation tests
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
| `LOCAL_JWT_EXPIRATION` | Token expiration | `15m` |
| `LOCAL_REFRESH_TOKEN_TTL` | Refresh token TTL | `168h` |
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

## Docker Services

| Service | Port | Description |
|---------|------|-------------|
| `backend` | 8080 | Go API server |
| `postgres` | 5432 | PostgreSQL database |
| `minio` | 9000/9001 | S3-compatible storage |
| `redis` | 6379 | Cache (optional) |
| `keycloak` | 8085 | OIDC provider (optional) |
| `web` | 3001 | React frontend |

## Project Structure

```
apps/backend/
├── cmd/server/
│   ├── main.go              # Application entrypoint
│   ├── wire.go              # Wire dependency injection
│   └── wire_gen.go          # Generated wire code
├── internal/
│   ├── domain/              # Business logic (NO external deps)
│   │   ├── provider.go      # Domain service providers (wire)
│   │   └── {entity}/
│   │       ├── entity.go    # Domain types
│   │       └── service.go   # Business logic
│   ├── infrastructure/
│   │   ├── config/          # Configuration
│   │   ├── provider.go      # Infrastructure providers (wire)
│   │   └── database/
│   │       ├── dbschema/    # GORM models
│   │       └── repository/  # Data access
│   └── interfaces/httpserver/
│       ├── routes/          # HTTP handlers
│       │   └── provider.go  # Route providers (wire)
│       ├── middlewares/     # Auth, logging, etc.
│       └── http_server.go   # Server setup
├── pkg/common/              # Shared utilities
├── migrations/              # SQL migrations
└── tests/                   # Integration tests
```

### Adding a New Domain

1. Create `internal/domain/{name}/entity.go` with types
2. Create `internal/domain/{name}/service.go` with business logic
3. Add service constructor to `internal/domain/provider.go`
4. Add GORM schema in `internal/infrastructure/database/dbschema/`
5. Add repository in `internal/infrastructure/database/repository/`
6. Add to `internal/infrastructure/database/repository/repository_provider.go`
7. Add HTTP handlers in `internal/interfaces/httpserver/routes/`
8. Register routes in `internal/interfaces/httpserver/routes/provider.go`
9. Regenerate wire: `go generate ./cmd/server`
10. Add migration in `migrations/`

## License

MIT License

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feat/amazing-feature`)
3. Commit changes (`git commit -m 'feat: add amazing feature'`)
4. Push branch (`git push origin feat/amazing-feature`)
5. Open Pull Request
