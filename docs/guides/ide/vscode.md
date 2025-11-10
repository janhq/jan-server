# VS Code Development Guide

Complete guide for developing Jan Server with Visual Studio Code.

## 📑 Table of Contents

- [Quick Start](#-quick-start)
- [Debug Configurations](#-debug-configurations)
- [VS Code Tasks](#-vs-code-tasks)
- [Environment Configuration](#-environment-configuration)
- [Provider Configuration](#-provider-configuration)
- [Common Workflows](#-common-workflows)
- [Troubleshooting](#-troubleshooting)
- [Configuration Reference](#-configuration-reference)

##  Quick Start

### Prerequisites

1. Install [VS Code](https://code.visualstudio.com/)
2. Install [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go)
3. Run initial setup:
   ```bash
   make setup
   ```

### Start Debugging

**Option 1: Debug LLM API**
1. Press `F5` or open Run and Debug (`Ctrl+Shift+D`)
2. Select "Debug LLM API"
3. Click Start Debugging
4. Set breakpoints and debug!

**Option 2: Debug MCP Tools**
1. Press `F5` or open Run and Debug (`Ctrl+Shift+D`)
2. Select "Debug MCP Tools"
3. Click Start Debugging
4. Set breakpoints and debug!

**What happens:**
-  All infrastructure starts automatically in Docker
-  Only the service you're debugging runs locally
-  Hot reload works automatically
-  Breakpoints work as expected

## 🐛 Debug Configurations

### Available Configurations

#### 1. Debug LLM API
**Best for:** Developing the main API service

**What it does:**
- Starts in Docker: PostgreSQL, Keycloak, Kong, SearXNG, Vector Store, SandboxFusion, MCP Tools
- Runs locally: LLM API only (with debugger attached)

**Use cases:**
- Debugging API endpoints
- Testing authentication flows
- Working on model provider integration
- Debugging database operations

**Keyboard shortcut:** `F5` (with "Debug LLM API" selected)

#### 2. Debug MCP Tools
**Best for:** Developing MCP (Model Context Protocol) tools

**What it does:**
- Starts in Docker: PostgreSQL, Keycloak, Kong, LLM API
- Runs locally: MCP Tools only (with debugger attached)

**Use cases:**
- Debugging MCP tool implementations
- Testing search integrations
- Working on code execution sandbox
- Debugging vector store operations

**Keyboard shortcut:** `F5` (with "Debug MCP Tools" selected)

#### 3. Debug LLM API (No PreLaunch)
**Best for:** Quick restarts when infrastructure is already running

**What it does:**
- Assumes all services are already running
- Just starts LLM API with debugger attached
- Faster startup (no Docker orchestration)

**Use when:**
- Services are already running
- You just want to restart the debugger
- Testing quick code changes

#### 4. Debug MCP Tools (No PreLaunch)
Same as above but for MCP Tools.

#### 5. Debug GormGen (LLM API)
**Best for:** Debugging database code generation

**Use when:**
- Working on GORM model generation
- Updating database schemas
- Debugging ORM code

### Debug Configuration Details

Each debug configuration includes:

```json
{
    "name": "Debug LLM API",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/services/llm-api/cmd/server",
    "cwd": "${workspaceFolder}/services/llm-api",
    "envFile": "${workspaceFolder}/.env",
    "env": {
        // Environment variable overrides
        "DATABASE_URL": "postgres://jan_user:jan_password@localhost:5432/...",
        "KEYCLOAK_BASE_URL": "http://localhost:8085",
        // ... more overrides
    },
    "preLaunchTask": "Start All Services For LLM API Debug",
    "console": "integratedTerminal"
}
```

**Key features:**
- `envFile`: Loads base config from `.env`
- `env`: Overrides specific variables for local debugging
- `preLaunchTask`: Auto-starts required Docker services
- `console: integratedTerminal`: Shows output in VS Code terminal

## 📋 VS Code Tasks

Access tasks via `Ctrl+Shift+P` → `Tasks: Run Task`

### Infrastructure Tasks

| Task | Description | Use Case |
|------|-------------|----------|
| **Start Hybrid Infrastructure** | Start PostgreSQL, Keycloak, Kong | Native API development |
| **Start Hybrid MCP Infrastructure** | Start SearXNG, Vector Store, SandboxFusion | Native MCP development |
| **Start All Services For LLM API Debug** | Start everything except LLM API | Automatic (via debug config) |
| **Start All Services For MCP Debug** | Start everything except MCP Tools | Automatic (via debug config) |
| **Start Full Stack** | Start all services in Docker | Full Docker development |
| **Stop All Services** | Stop all Docker services | Cleanup |
| **Stop Hybrid Infrastructure** | Stop hybrid mode services | Cleanup |

### Service Tasks

| Task | Description | Make Equivalent |
|------|-------------|-----------------|
| **Start LLM API** | Run LLM API natively | `make start-llm-api` |
| **Start MCP Tools** | Run MCP Tools natively | `make start-mcp-tools` |

### Build Tasks

| Task | Description | Make Equivalent |
|------|-------------|-----------------|
| **Build LLM API** | Build LLM API binary | `make build-api` |
| **Build MCP Tools** | Build MCP Tools binary | `make build-mcp` |
| **Build All** | Build all services | `make build-all` |

### Test Tasks

| Task | Description | Make Equivalent |
|------|-------------|-----------------|
| **Run All Tests** | Run all integration tests | `make run-all-tests` |
| **Run Unit Tests - LLM API** | Run API unit tests | `make test-api` |
| **Run Unit Tests - MCP Tools** | Run MCP unit tests | `make test-mcp` |
| **Run Integration Tests - Auth** | Test authentication | `make test-auth` |
| **Run Integration Tests - Conversations** | Test conversation API | `make test-conversations` |
| **Run Integration Tests - MCP** | Test MCP integration | `make test-mcp-integration` |

### Utility Tasks

| Task | Description | Make Equivalent |
|------|-------------|-----------------|
| **Health Check All Services** | Check service health | `make health-check` |
| **View Service Logs** | View Docker logs | `make logs` |
| **Generate Swagger Docs** | Generate API docs | `make swagger` |
| **Database Console** | Open PostgreSQL console | `make db-console` |

## 🔧 Environment Configuration

### How It Works

VS Code debug configurations use a two-layer environment setup:

1. **Base Layer (.env file)**
   - Loaded via `envFile: "${workspaceFolder}/.env"`
   - Contains Docker-compatible values (e.g., `http://keycloak:8085`)
   - Used when services run in Docker

2. **Override Layer (env object)**
   - Specified in `launch.json` `env` property
   - Overrides specific variables for local debugging
   - Uses `localhost` instead of Docker hostnames

### Environment Variable Overrides

#### LLM API Debug Configuration

```json
"env": {
    // Database (Docker → localhost)
    "DATABASE_URL": "postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable",
    "DB_DSN": "postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable",
    
    // Authentication (Docker → localhost)
    "KEYCLOAK_BASE_URL": "http://localhost:8085",
    "JWKS_URL": "http://localhost:8085/realms/jan/protocol/openid-connect/certs",
    "ISSUER": "http://localhost:8085/realms/jan",
    
    // Service Configuration
    "HTTP_PORT": "8080",
    "LOG_LEVEL": "debug",
    "LOG_FORMAT": "console",
    "AUTO_MIGRATE": "true",
    
    // Telemetry (disabled for debugging)
    "OTEL_EXPORTER_OTLP_ENDPOINT": "",
    
    // Provider Configuration (YAML mode)
    "VLLM_PROVIDER_URL": "http://localhost:8101/v1",
    "JAN_PROVIDER_CONFIGS": "true",
    "JAN_PROVIDER_CONFIGS_FILE": "config/providers.yml",
    "JAN_PROVIDER_CONFIG_SET": "default",
    "JAN_DEFAULT_NODE_SETUP": "false",
    
    // MCP Services (Docker → localhost)
    "SEARXNG_URL": "http://localhost:8086",
    "VECTOR_STORE_URL": "http://localhost:3015",
    "SANDBOXFUSION_URL": "http://localhost:3010",
    "MCP_TOOLS_URL": "http://localhost:8091",
    "CODE_SANDBOX_URL": "http://localhost:3000/mcp",
    "PLAYWRIGHT_URL": "http://localhost:3000"
}
```

#### MCP Tools Debug Configuration

```json
"env": {
    "HTTP_PORT": "8091",
    "VECTOR_STORE_URL": "http://localhost:3015",
    "SEARXNG_URL": "http://localhost:8086",
    "SANDBOXFUSION_URL": "http://localhost:3010",
    "LOG_LEVEL": "debug",
    "LOG_FORMAT": "console",
    "OTEL_EXPORTER_OTLP_ENDPOINT": ""
}
```

### Why Override?

**Problem:** Services in Docker communicate via internal hostnames (e.g., `http://keycloak:8085`)

**Solution:** Override with `localhost` when debugging locally (e.g., `http://localhost:8085`)

**Result:** Local service connects to Docker containers via published ports

## 🎯 Provider Configuration

### Two Configuration Modes

#### 1. YAML Configuration (Recommended)

**Enable in .env:**
```properties
JAN_PROVIDER_CONFIGS=true
JAN_PROVIDER_CONFIGS_FILE=config/providers.yml
JAN_PROVIDER_CONFIG_SET=default
JAN_DEFAULT_NODE_SETUP=false
```

**Benefits:**
-  Multiple providers (vLLM + Gemini + OpenAI, etc.)
-  Per-provider configuration in YAML
-  Environment-specific sets (default, production, testing)
-  Proper provider names in API responses

**providers.yml structure:**
```yaml
providers:
  default:
    - name: Local vLLM Provider      # Shows as "owned_by"
      type: jan
      url: ${VLLM_PROVIDER_URL}      # Environment variable
      api_key: ${VLLM_INTERNAL_KEY}
      auto_enable_new_models: true
      sync_models: true
      
    - name: External Gemini
      type: gemini
      url: https://generativelanguage.googleapis.com/v1beta/openai
      api_key: ${GEMINI_API_KEY}
      auto_enable_new_models: true
      sync_models: true
      
  production:
    - name: Production Provider
      type: openai
      url: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
```

**API Response:**
```json
{
    "object": "list",
    "data": [
        {
            "id": "qwen/qwen2.5-0.5b-instruct",
            "object": "model",
            "created": 1762597603,
            "owned_by": "Local vLLM Provider"  //  Correct name from YAML
        }
    ]
}
```

#### 2. Legacy Mode (Single Provider)

**Enable in .env:**
```properties
JAN_DEFAULT_NODE_SETUP=true
JAN_DEFAULT_NODE_URL=http://vllm-jan-gpu:8101/v1
JAN_DEFAULT_NODE_API_KEY=changeme
```

**Limitations:**
-  Only one provider
-  Hardcoded name "vLLM Provider"
-  No per-environment configuration
-  **Deprecated** - use YAML config instead

### Switching Providers

**For different environments:**
```properties
# Development
JAN_PROVIDER_CONFIG_SET=default

# Production
JAN_PROVIDER_CONFIG_SET=production

# Testing
JAN_PROVIDER_CONFIG_SET=testing
```

### Provider URL Configuration

**For Docker (in .env):**
```properties
VLLM_PROVIDER_URL=http://vllm-jan-gpu:8101/v1
```

**For local debugging (in launch.json):**
```json
"VLLM_PROVIDER_URL": "http://localhost:8101/v1"
```

**In providers.yml:**
```yaml
url: ${VLLM_PROVIDER_URL}  # Expands to appropriate value
```

## 💼 Common Workflows

### Workflow 1: Debug API Endpoint

1. **Open file** containing the endpoint (e.g., `services/llm-api/internal/interfaces/httpserver/handlers/`)
2. **Set breakpoint** by clicking left of line number
3. **Press F5** with "Debug LLM API" selected
4. **Wait for services** to start (~30 seconds)
5. **Make API request** (via Postman, curl, or tests)
6. **Debugger stops** at your breakpoint
7. **Inspect variables**, step through code, evaluate expressions

### Workflow 2: Test Provider Integration

1. **Ensure vLLM is running:**
   ```bash
   make up-vllm-gpu
   # or
   make up-vllm-cpu
   ```

2. **Update .env** to enable provider configs:
   ```properties
   JAN_PROVIDER_CONFIGS=true
   JAN_PROVIDER_CONFIG_SET=default
   ```

3. **Start debugging:**
   - Press F5 with "Debug LLM API"

4. **Test provider:**
   ```bash
   curl http://localhost:8080/v1/models
   ```

5. **Check logs** for provider loading:
   ```
   INF Loading providers from config/providers.yml set=default
   INF Provider bootstrapped name="Local vLLM Provider"
   ```

### Workflow 3: Debug MCP Tool

1. **Open MCP tool** file (e.g., `services/mcp-tools/tools/`)
2. **Set breakpoint**
3. **Press F5** with "Debug MCP Tools" selected
4. **Make MCP request** through LLM API:
   ```bash
   curl -X POST http://localhost:8080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "qwen2.5-0.5b-instruct",
       "messages": [{"role": "user", "content": "Search for latest news"}],
       "tools": [{"type": "function", "function": {"name": "google_search"}}]
     }'
   ```
5. **Debugger stops** when tool is invoked

### Workflow 4: Run Integration Tests

**Option 1: VS Code Task**
1. `Ctrl+Shift+P`
2. "Tasks: Run Test Task"
3. Select "Run All Tests"

**Option 2: Terminal**
```bash
make run-all-tests
```

**Option 3: Specific Test Suite**
```bash
# Auth tests only
make test-auth

# Conversation tests only
make test-conversations

# MCP tests only
make test-mcp-integration
```

### Workflow 5: Generate Code

**Generate Swagger docs:**
1. `Ctrl+Shift+P`
2. "Tasks: Run Task"
3. "Generate Swagger Docs"

**Or:**
```bash
make swagger
```

**Generate GORM models:**
1. Select "Debug GormGen (LLM API)" configuration
2. Press F5
3. Check `internal/infrastructure/database/repository/` for generated code

### Workflow 6: Database Operations

**Open database console:**
1. `Ctrl+Shift+P`
2. "Tasks: Run Task"
3. "Database Console"

**Or:**
```bash
make db-console
```

**Run migrations:**
```bash
make db-migrate
```

**Backup database:**
```bash
make db-backup
```

## 🆘 Troubleshooting

### Issue: Services Not Starting

**Symptoms:**
- Debug session hangs
- "Connection refused" errors
- Services don't appear in `docker compose ps`

**Solutions:**

1. **Check Docker is running:**
   ```bash
   docker info
   ```

2. **Check port conflicts:**
   ```bash
   # Windows PowerShell
   netstat -an | findstr "8080"
   netstat -an | findstr "5432"
   ```

3. **View Docker logs:**
   ```bash
   make logs-infra
   make logs-api
   make logs-mcp
   ```

4. **Clean restart:**
   ```bash
   make down
   docker system prune -f
   make setup
   ```

### Issue: Provider Shows Wrong Name

**Symptom:**
```json
{
    "owned_by": "vLLM Provider"  //  Not from providers.yml
}
```

**Solution:**

Check `.env` file:
```properties
#  Wrong (legacy mode)
JAN_DEFAULT_NODE_SETUP=true

#  Correct (YAML mode)
JAN_PROVIDER_CONFIGS=true
JAN_PROVIDER_CONFIGS_FILE=config/providers.yml
JAN_PROVIDER_CONFIG_SET=default
JAN_DEFAULT_NODE_SETUP=false
```

**Restart debug session** after fixing.

### Issue: Docker Hostname Errors

**Symptom:**
```
dial tcp: lookup vllm-jan-gpu: no such host
dial tcp: lookup keycloak: no such host
```

**Solution:**

Verify `launch.json` has localhost overrides:
```json
"env": {
    "VLLM_PROVIDER_URL": "http://localhost:8101/v1",
    "KEYCLOAK_BASE_URL": "http://localhost:8085",
    // ... etc
}
```

**Restart debug session** completely (Shift+F5, then F5).

### Issue: OpenTelemetry Still Uploading Metrics

**Symptom:**
- Seeing OTLP export logs
- Metrics being sent when debugging

**Solution:**

Check `launch.json`:
```json
//  Wrong (variable doesn't exist)
"OTEL_ENABLED": "false"

//  Correct (set endpoint to empty)
"OTEL_EXPORTER_OTLP_ENDPOINT": ""
```

**Why:** Code checks `if cfg.OTLPEndpoint != ""`, not `OTEL_ENABLED`.

### Issue: Database Connection Failed

**Symptoms:**
```
connection refused: localhost:5432
FATAL: password authentication failed
```

**Solutions:**

1. **Check PostgreSQL is running:**
   ```bash
   docker compose ps api-db
   ```

2. **Verify credentials in .env:**
   ```properties
   POSTGRES_USER=jan_user
   POSTGRES_PASSWORD=jan_password
   POSTGRES_DB=jan_llm_api
   ```

3. **Test connection:**
   ```bash
   make db-console
   ```

4. **Check DATABASE_URL in launch.json:**
   ```json
   "DATABASE_URL": "postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable"
   ```

### Issue: Keycloak Not Ready

**Symptoms:**
```
connection refused: localhost:8085
JWKS fetch failed
```

**Solution:**

Keycloak takes 20-30 seconds to start. Wait and check:

```bash
# Check status
curl http://localhost:8085

# View logs
docker compose logs keycloak

# Wait for "Started Keycloak" message
```

### Issue: Hot Reload Not Working

**Symptoms:**
- Code changes don't trigger recompilation
- Must manually restart debugger

**Solutions:**

1. **Check if using Air:**
   ```bash
   # Should be installed
   which air
   ```

2. **For tasks (not debug):**
   - Tasks use `air` for hot reload
   - Debug configs don't (debugger handles it)

3. **Restart debug session** after major changes

### Issue: Tests Failing

**Common causes:**

1. **Services not running:**
   ```bash
   make up-full
   sleep 10  # Wait for startup
   make run-all-tests
   ```

2. **Database not migrated:**
   ```bash
   make db-migrate
   ```

3. **Wrong environment:**
   ```bash
   # Check .env
   cat .env | grep JAN_PROVIDER
   ```

4. **Port conflicts:**
   ```bash
   # Check what's using ports
   netstat -an | findstr "8080"
   ```

### Issue: Can't Set Breakpoints

**Symptoms:**
- Breakpoints show as hollow circles
- Code doesn't stop at breakpoints

**Solutions:**

1. **Build in debug mode:**
   - Debug configurations automatically use `-gcflags="all=-N -l"`
   - Disables optimizations

2. **Check Go extension is installed:**
   - Extensions → Search "Go" → Install

3. **Reload VS Code:**
   - `Ctrl+Shift+P` → "Developer: Reload Window"

4. **Verify dlv (Delve debugger):**
   ```bash
   dlv version
   ```

## 📚 Configuration Reference

### Sample Configuration Files

Complete, ready-to-use configuration files are available in this directory:

- **[launch.json](launch.json)** - All debug configurations
- **[tasks.json](tasks.json)** - All VS Code tasks

**To use:**
1. Copy to `.vscode/` directory in project root
2. Restart VS Code
3. Press F5 to start debugging

**Or manually create:**
```bash
# Create .vscode directory if it doesn't exist
mkdir -p .vscode

# Copy sample files
cp docs/guides/ide/launch.json .vscode/
cp docs/guides/ide/tasks.json .vscode/
```

### Complete launch.json

See `.vscode/launch.json` for full configuration.

**Key configurations:**

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug LLM API",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/services/llm-api/cmd/server",
            "cwd": "${workspaceFolder}/services/llm-api",
            "envFile": "${workspaceFolder}/.env",
            "env": { /* overrides */ },
            "preLaunchTask": "Start All Services For LLM API Debug"
        }
        // ... more configurations
    ]
}
```

### Complete tasks.json

See `.vscode/tasks.json` for full configuration.

**Key task patterns:**

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Start LLM API",
            "type": "shell",
            "command": "make",
            "args": ["start-llm-api"],
            "dependsOn": ["Start Hybrid Infrastructure"],
            "isBackground": true
        }
        // ... more tasks
    ]
}
```

### Environment Variables Reference

**Database:**
- `DATABASE_URL` - PostgreSQL connection string
- `DB_DSN` - Alternative database connection string
- `POSTGRES_USER` - Database username
- `POSTGRES_PASSWORD` - Database password
- `POSTGRES_DB` - Database name
- `POSTGRES_PORT` - Database port (5432)

**Authentication:**
- `KEYCLOAK_BASE_URL` - Keycloak server URL
- `KEYCLOAK_REALM` - Keycloak realm name (jan)
- `KEYCLOAK_ADMIN` - Keycloak admin username
- `KEYCLOAK_ADMIN_PASSWORD` - Keycloak admin password
- `JWKS_URL` - JWKS endpoint for JWT validation
- `ISSUER` - JWT issuer URL
- `AUDIENCE` - JWT audience
- `BACKEND_CLIENT_ID` - Backend client ID
- `BACKEND_CLIENT_SECRET` - Backend client secret

**Provider Configuration:**
- `JAN_PROVIDER_CONFIGS` - Enable YAML provider config (true/false)
- `JAN_PROVIDER_CONFIGS_FILE` - Path to providers.yml
- `JAN_PROVIDER_CONFIG_SET` - Which provider set to use
- `JAN_DEFAULT_NODE_SETUP` - Enable legacy single provider
- `VLLM_PROVIDER_URL` - vLLM provider URL

**MCP Services:**
- `SEARXNG_URL` - SearXNG search engine URL
- `VECTOR_STORE_URL` - Vector store service URL
- `SANDBOXFUSION_URL` - Code execution sandbox URL
- `MCP_TOOLS_URL` - MCP tools service URL

**Telemetry:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry endpoint (empty to disable)
- `OTEL_SERVICE_NAME` - Service name for telemetry

**Logging:**
- `LOG_LEVEL` - Log level (debug, info, warn, error)
- `LOG_FORMAT` - Log format (console, json)

### Make Commands Reference

See [Quick Reference](../../QUICK_REFERENCE.md) for complete make command list.

**Most used:**
```bash
# Development
make start-llm-api          # Run API natively
make start-mcp-tools        # Run MCP natively
make hybrid-infra-up        # Start infrastructure
make hybrid-stop            # Stop hybrid mode

# Build & Test
make build-all              # Build everything
make run-all-tests          # Run all tests
make test                   # Unit tests only

# Docker
make up-full                # Start all in Docker
make down                   # Stop everything
make logs                   # View all logs

# Database
make db-console             # PostgreSQL CLI
make db-migrate             # Run migrations
```

## 🔗 Related Documentation

- [Quick Reference](../../QUICK_REFERENCE.md) - Make commands cheat sheet
- [Development Guide](../development.md) - General development workflow
- [Hybrid Mode Guide](../hybrid-mode.md) - Native development setup
- [Testing Guide](../testing.md) - Testing procedures
- [Monitoring Guide](../monitoring.md) - Observability setup

---

**Need help?** Open an issue or check the troubleshooting section above.
