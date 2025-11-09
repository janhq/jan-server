# Hybrid Development Mode Guide

Hybrid mode allows you to run services natively (outside Docker) while keeping infrastructure services in Docker. This provides the best of both worlds: fast iteration with native builds and full infrastructure support.

## Table of Contents

1. [Why Hybrid Mode?](#why-hybrid-mode)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Workflows](#workflows)
5. [Configuration](#configuration)
6. [Debugging](#debugging)
7. [Troubleshooting](#troubleshooting)

## Why Hybrid Mode?

### Benefits

✅ **Faster Development**
- Instant code changes (no Docker rebuild)
- Native debugging tools
- Hot reloading

✅ **Better IDE Integration**
- Native breakpoints
- Real-time code analysis
- Better autocomplete

✅ **Easier Debugging**
- Direct console output
- Native debugger (delve)
- Full stack traces

✅ **Full Infrastructure**
- PostgreSQL in Docker
- Keycloak in Docker
- Kong in Docker
- MCP services in Docker

### When to Use

- **Active feature development**
- **Debugging complex issues**
- **Learning the codebase**
- **Hot-reloading workflows**

### When NOT to Use

- **Integration testing** (use full Docker)
- **Production-like testing** (use full Docker)
- **CI/CD pipelines** (use full Docker)

## Prerequisites

```bash
# Required
- Docker & Docker Compose V2
- Go 1.25+
- Make

# Optional but recommended
- Delve debugger: go install github.com/go-delve/delve/cmd/dlv@latest
- Air (hot reload): go install github.com/air-verse/air@latest
```

## Quick Start

### Hybrid API Development

```bash
# Terminal 1: Start infrastructure
make hybrid-dev-api

# Terminal 2: Run APIs natively
make hybrid-run-api        # LLM API
make hybrid-run-media      # Media API

# Or manually
cd services/llm-api
source ../../config/hybrid.env  # Linux/Mac
# For Windows: load env vars manually
go run .
```

### Hybrid MCP Development

```bash
# Terminal 1: Start MCP infrastructure
make hybrid-dev-mcp

# Terminal 2: Run MCP Tools natively
make hybrid-run-mcp

# Or manually
cd services/mcp-tools
source ../../config/hybrid.env
go run .
```

## Workflows

### 1. API Development Workflow

```bash
# 1. Setup hybrid environment
make hybrid-dev-api

# What this does:
# - Starts PostgreSQL on localhost:5432
# - Starts Keycloak on localhost:8085
# - Creates networks
# - Sets up environment

# 2. In another terminal, run API
cd services/llm-api
source ../../config/hybrid.env

# Option A: Direct run
go run .

# Option B: With hot reload (requires air)
air

# Option C: With debugger
dlv debug --headless --listen=:2345 --api-version=2

# 3. Make changes, save, and see results immediately!

# 4. Stop when done
# Ctrl+C in API terminal
make hybrid-stop  # Stop infrastructure
```

### 2. MCP Tools Development Workflow

```bash
# 1. Setup
make hybrid-dev-mcp

# This starts:
# - SearXNG on localhost:8086
# - Vector Store on localhost:3015
# - SandboxFusion on localhost:3010

# 2. Run MCP Tools
cd services/mcp-tools
source ../../config/hybrid.env
go run .

# 3. Test
curl -X POST http://localhost:8091/v1/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# 4. Cleanup
make hybrid-stop
```

### 3. Full Hybrid Development

Run both API and MCP natively:

```bash
# Terminal 1: Infrastructure
make hybrid-dev-full

# Terminal 2: LLM API
cd services/llm-api
source ../../config/hybrid.env
go run .

# Terminal 3: Media API
cd services/media-api
source ../../config/hybrid.env
go run .

# Terminal 4: MCP
cd services/mcp-tools
source ../../config/hybrid.env
go run .
```

## Configuration

### Environment Variables

Hybrid mode uses `config/hybrid.env`:

```bash
# Database (from Docker, exposed on localhost)
DATABASE_URL=postgres://jan_user:jan_password@localhost:5432/jan_llm_api

# Keycloak (from Docker)
KEYCLOAK_BASE_URL=http://localhost:8085
JWKS_URL=http://localhost:8085/realms/jan/protocol/openid-connect/certs

# MCP Services (from Docker)
SEARXNG_URL=http://localhost:8086
VECTOR_STORE_URL=http://localhost:3015
SANDBOXFUSION_URL=http://localhost:3010

# Service ports
HTTP_PORT=8080              # API port
MCP_TOOLS_HTTP_PORT=8091    # MCP port
MEDIA_API_PORT=8285         # Media API port
MEDIA_SERVICE_KEY=changeme-media-key

# Logging (console for native)
LOG_LEVEL=debug
LOG_FORMAT=console

# Auto-migrate
AUTO_MIGRATE=true

# Media storage defaults
MEDIA_S3_ENDPOINT=https://s3.menlo.ai
MEDIA_S3_BUCKET=platform-dev
MEDIA_S3_ACCESS_KEY=XXXXX
MEDIA_S3_SECRET_KEY=YYYY
```

### Loading Environment

**Linux/macOS**:
```bash
source config/hybrid.env
```

**Windows PowerShell**:
```powershell
Get-Content config\hybrid.env | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), "Process")
    }
}
```

**Or use helper scripts**:
```bash
./scripts/hybrid-run-api.sh         # Linux/Mac (LLM API)
./scripts/hybrid-run-media-api.sh   # Linux/Mac (Media API)
.\scripts\hybrid-run-api.ps1        # Windows (LLM API)
.\scripts\hybrid-run-media-api.ps1  # Windows (Media API)
```

## Debugging

### Using Delve

**Install Delve**:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Debug API**:
```bash
# Terminal 1: Infrastructure
make hybrid-infra-up

# Terminal 2: API with debugger
make hybrid-debug-api

# Debugger listens on localhost:2345
```

**Debug MCP**:
```bash
# Terminal 1: MCP infrastructure
make hybrid-mcp-up

# Terminal 2: MCP with debugger
make hybrid-debug-mcp

# Debugger listens on localhost:2346
```

**Connect from VS Code**:

Add to `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to API",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}/services/llm-api",
            "port": 2345,
            "host": "localhost"
        },
        {
            "name": "Connect to MCP",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}/services/mcp-tools",
            "port": 2346,
            "host": "localhost"
        }
    ]
}
```

**Connect from GoLand/IntelliJ**:

1. Run > Edit Configurations
2. Add new "Go Remote" configuration
3. Host: `localhost`, Port: `2345` (or `2346` for MCP)
4. Click Debug

### Hot Reload with Air

**Install Air**:
```bash
go install github.com/air-verse/air@latest
```

**Use Air for hot reloading**:

```bash
# In services/llm-api or services/mcp-tools
source ../../config/hybrid.env
air

# Air watches for changes and rebuilds automatically
```

**Configure Air** (`.air.toml`):
```toml
[build]
  cmd = "go build -o ./tmp/main ."
  bin = "./tmp/main"
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_dir = ["tmp", "vendor", "docs"]
  delay = 1000
```

## Troubleshooting

### Infrastructure Not Available

```bash
# Check infrastructure is running
make dev-status

# Restart infrastructure
make hybrid-infra-down
make hybrid-infra-up

# Check connectivity
curl http://localhost:8085    # Keycloak
curl http://localhost:5432    # PostgreSQL (will fail but should connect)
```

### Database Connection Fails

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection
docker exec -it jan-server-api-db-1 psql -U jan_user -d jan_llm_api

# Reset database
make db-reset
make hybrid-infra-up
```

### MCP Services Not Accessible

```bash
# Check MCP services
make health-mcp

# Restart MCP infrastructure
make hybrid-mcp-down
make hybrid-mcp-up

# Test individual services
curl http://localhost:8086    # SearXNG
curl http://localhost:3015/health  # Vector Store
curl http://localhost:3010    # SandboxFusion
```

### Port Already in Use

If port 8080 or 8091 is in use:

```bash
# Find what's using the port
# Windows
netstat -ano | findstr ":8080"

# Linux/Mac
lsof -i :8080

# Change port in config/hybrid.env
HTTP_PORT=8081
MCP_TOOLS_HTTP_PORT=8092
```

### Environment Variables Not Loaded

```bash
# Show current environment
make hybrid-env-api      # API environment
make hybrid-env-mcp      # MCP environment

# Manually load
source config/hybrid.env  # Linux/Mac

# Verify variables are set
echo $DATABASE_URL
echo $HTTP_PORT
```

## Best Practices

### 1. Always Start Infrastructure First

```bash
# Wrong
go run .  # Fails - no database

# Right
make hybrid-dev-api  # Start infrastructure
go run .             # Now it works
```

### 2. Use Helper Scripts

```bash
# Instead of manual setup
./scripts/hybrid-run-api.sh        # LLM API
./scripts/hybrid-run-media-api.sh  # Media API

# Or make targets
make hybrid-run-api
make hybrid-run-media
make hybrid-run-mcp
```

### 3. Check Health Before Starting

```bash
make hybrid-dev-api
make health-check     # Ensure everything is ready
go run .
```

### 4. Stop Infrastructure When Done

```bash
# Stop specific
make hybrid-infra-down
make hybrid-mcp-down

# Stop all
make hybrid-stop
```

### 5. Switch to Testing for Integration Tests

```bash
# Hybrid mode for development
make hybrid-dev-api

# Full Docker for testing
make hybrid-stop
make env-switch ENV=testing
make up-full
make test-all
```

## Comparison: Docker vs Hybrid

| Aspect | Full Docker | Hybrid Mode |
|--------|-------------|-------------|
| **Startup Time** | 30-60s | 5-10s |
| **Code Changes** | Rebuild image | Instant |
| **Debugging** | Docker logs | Native debugger |
| **Hot Reload** | Not available | With Air |
| **IDE Integration** | Limited | Full |
| **Network** | Docker network | localhost |
| **Production Parity** | High | Medium |
| **Best For** | Testing | Development |

## Advanced Tips

### Connect to Services from Host

All Docker services are accessible on `localhost`:

```bash
# PostgreSQL
psql -h localhost -p 5432 -U jan_user -d jan_llm_api

# Keycloak Admin Console
open http://localhost:8085

# SearXNG
open http://localhost:8086
```

### Use host.docker.internal in Kong

If you need Kong to route to your native API:

1. Update `kong/kong-hybrid.yml`:
```yaml
services:
  - name: llm-api
    url: http://host.docker.internal:8080
```

2. Start Kong in hybrid mode:
```bash
make hybrid-kong-up
```

### Database GUI Tools

Connect with your favorite GUI:

- **TablePlus**: postgres://jan_user:jan_password@localhost:5432/jan_llm_api
- **DBeaver**: Same connection string
- **pgAdmin**: Host: localhost, Port: 5432, User: jan_user

---

**Happy Hybrid Development! 🚀**
