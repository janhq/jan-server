# Jan Server Mono - Quickstart Guide

This guide will help you get Jan Server Mono running quickly on your local machine.

## Prerequisites

- **Docker and Docker Compose** - Required for running all services
- **Go 1.24+** (optional) - Only needed for local development without Docker

## Quick Start (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/janhq/server.git
cd server/mono

# Interactive setup and start all services
make quickstart
```

This will:
1. Build the `jan-cli` tool
2. Run an interactive setup wizard to configure:
   - JWT secret (auto-generated if not provided)
   - LLM provider URL and API key (optional)
   - Memory service (optional)
3. Start all Docker services (backend, postgres, minio, web)
4. Display service URLs and getting started commands

### Non-Interactive Mode

For CI/CD or automated deployments:

```bash
make quickstart-auto
```

This creates `.env` from template and starts services without prompts.

### Service URLs

After `make quickstart` completes:

| Service | URL | Description |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | Go API server |
| Web UI | http://localhost:3001 | React frontend |
| MinIO Console | http://localhost:9001 | S3-compatible storage admin |
| PostgreSQL | localhost:5432 | Database (internal) |

## Step-by-Step Setup

If you prefer more control, follow these steps:

### 1. Create Environment File

```bash
make setup
```

This creates `.env` from `.env.example`. Review and customize as needed:

```bash
# Required settings (defaults work for local development)
DB_POSTGRESQL_WRITE_DSN=postgres://jan:janpassword@postgres:5432/jan?sslmode=disable
LOCAL_JWT_SECRET=your-secure-secret-at-least-32-characters

# S3/MinIO settings
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
S3_BUCKET=jan-media
```

### 2. Start Services

```bash
# Start all services
make docker-up

# Or start with optional services
make docker-up-keycloak    # Include Keycloak OIDC
make docker-up-redis       # Include Redis cache
make docker-up-full        # Include all optional services
```

### 3. Verify Services

```bash
make health-check
```

Expected output:
```
Checking backend health...
Backend: OK
Checking database readiness...
Database: OK
```

## Configuration

### Essential Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | API server port | `8080` |
| `LOCAL_JWT_SECRET` | JWT signing secret | Required |
| `DB_POSTGRESQL_WRITE_DSN` | Database connection | Required |
| `S3_ENDPOINT` | MinIO/S3 endpoint | `http://minio:9000` |

### Optional Features

| Variable | Description | Default |
|----------|-------------|---------|
| `MEMORY_ENABLED` | Enable memory service | `false` |
| `KEYCLOAK_ENABLED` | Enable Keycloak OIDC | `false` |
| `GITHUB_CONNECTOR_ENABLED` | GitHub OAuth | `false` |
| `GOOGLE_CONNECTOR_ENABLED` | Google OAuth | `false` |

## Development Workflow

### Running with Hot Reload

For active development, run services with hot reload:

```bash
# Terminal 1: Start infrastructure
docker compose up -d postgres minio

# Terminal 2: Run backend with hot reload
make dev-backend

# Terminal 3: Run frontend with hot reload
make dev-web
```

### Running Tests

```bash
# Run unit tests
make test

# Run all API integration tests (requires running server)
make test-all

# Run specific test suites
make test-auth          # Authentication tests
make test-conversation  # Conversation tests
make test-model         # Model tests
make test-media         # Media upload tests
make test-messages      # Messages tests
make test-image         # Image generation tests

# Quick smoke test
make test-quick
```

### Using jan-cli

The `jan-cli` tool provides development utilities:

```bash
# Build the CLI
make cli-build

# Run API tests with custom options
./tools/jan-cli/jan-cli api-test run tests/e2e/automation/collections/auth.postman.json \
  --env-var "gateway_url=http://localhost:8080" \
  --auto-auth guest \
  --verbose

# Development setup
./tools/jan-cli/jan-cli dev setup
```

## API Quick Reference

### Health Checks

```bash
# Liveness probe
curl http://localhost:8080/healthz

# Readiness probe
curl http://localhost:8080/readyz
```

### Authentication

```bash
# Register a new user
curl -X POST http://localhost:8080/v1/auth/local/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123", "name": "Test User"}'

# Login
curl -X POST http://localhost:8080/v1/auth/local/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Use the returned access_token for authenticated requests
export TOKEN="your-access-token"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/auth/me
```

### Chat Completions (OpenAI-Compatible)

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

### List Models

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/models
```

## Common Tasks

### Reset Database

```bash
make db-reset
```

### View Logs

```bash
# All services
make docker-logs

# Backend only
make docker-logs-backend
```

### Stop Services

```bash
# Stop containers (keep data)
make docker-down

# Stop and remove all data
make docker-down-clean
```

### Rebuild Containers

```bash
make docker-rebuild
```

## Troubleshooting

### Backend Won't Start

1. Check if PostgreSQL is running:
   ```bash
   docker compose ps postgres
   ```

2. Check backend logs:
   ```bash
   make docker-logs-backend
   ```

3. Verify database connection:
   ```bash
   make db-console
   ```

### Authentication Fails

1. Ensure `LOCAL_JWT_SECRET` is set in `.env`
2. Check it's at least 32 characters
3. Restart backend after changing secrets

### Port Conflicts

If ports 8080, 3001, or 9000 are in use:

1. Stop conflicting services
2. Or modify ports in `.env`:
   ```bash
   HTTP_PORT=8081
   ```

### Database Migration Issues

```bash
# Run migrations manually
make db-migrate

# Or reset completely
make db-reset
```

## Next Steps

- Explore the [API Reference](../README.md#api-reference) for all endpoints
- Set up LLM providers by configuring API keys
- Enable optional features like memory and OAuth connectors
- Review the architecture documentation for deeper understanding

## Getting Help

- Check the [README](../README.md) for comprehensive documentation
- Run `make help` to see all available commands
- Open an issue at https://github.com/janhq/server/issues
