# Getting Started with Jan Server

Welcome! This guide will help you get Jan Server up and running in minutes.

> **Note:** This guide covers Docker Compose setup for local development. For production deployment, see:
>
> - [Deployment Guide](guides/deployment.md) - All deployment options

## Prerequisites

Before you begin, ensure you have:

- **Docker Desktop** (Windows/macOS) or **Docker Engine + Docker Compose** (Linux)
- **Make** (usually pre-installed on macOS/Linux, [install on Windows](https://gnuwin32.sourceforge.net/packages/make.htm))
- **Git**
- At least 8GB RAM available
- For GPU inference: NVIDIA GPU with CUDA support

Optional (for development):

- Go 1.21+
- Go 1.23+ (for jan-cli api-test)

## Quick Setup

### 1. Clone the Repository

```bash
git clone https://github.com/janhq/jan-server.git
cd jan-server
```

### 2. Run the Setup Wizard (Recommended)

```bash
make quickstart
```

`make quickstart` launches the `jan-cli` wizard. It prompts for your LLM provider (local vLLM vs remote API), MCP search provider, and Media API preference, then writes `.env` plus `config/secrets.env`. When configuration finishes it automatically starts Docker Compose. Re-run the command anytime to update settings (answer **Y** when asked to overwrite `.env`).

#### Wizard options at a glance

- **LLM provider**: Local vLLM (GPU, downloads models) or remote OpenAI-compatible endpoint (no GPU required).
- **MCP search**: Serper (API key), SearXNG (local, no key), or None (MCP Tools still run without search).
- **Media API**: Enable for uploads and jan\_\* IDs, or disable if not needed.

#### Example configurations

- **Full local (GPU)**: Local vLLM + SearXNG + Media enabled → everything runs locally.
- **Cloud LLM + Serper**: Remote API endpoint + Serper key + Media enabled → light local footprint, best search.
- **Minimal**: Remote API endpoint + no search + Media disabled → smallest local runtime.

### Manual configuration (if you cannot run the wizard)

```bash
# Copy templates
cp .env.template .env
cp config/secrets.env.example config/secrets.env

# Edit with your values
nano .env
nano config/secrets.env

# Populate defaults and validate
make setup
```

`make setup` uses `jan-cli` in non-interactive mode to check dependencies, ensure directories exist, and pull base images.

**Configuration details:**

- Canonical defaults live in `config/defaults.yaml` (generated from Go structs)
- Secrets belong in `config/secrets.env` (copied from `config/secrets.env.example`)
- Environment templates (Docker/Kubernetes) are documented in [Configuration System](configuration/README.md)

### 3. Start Services (skip if quickstart already did this)

```bash
# Start full stack (CPU inference)
make up-full

# Optional: start monitoring stack
make monitor-up
```

Wait for all services to start (30-60 seconds). You can monitor progress with:

```bash
make logs
```

### What the wizard does

1. Prompts for LLM/search/media choices.
2. Writes `.env` and `config/secrets.env`.
3. Checks Docker availability and networks.
4. Starts the Compose stack and waits for health (about 30 seconds).

### 5. Verify Installation

```bash
make health-check
```

You should see all services reporting as healthy.

## Access Services

Once running, you can access:

| Service                | URL                                          | Credentials                           |
| ---------------------- | -------------------------------------------- | ------------------------------------- |
| **API Gateway**        | http://localhost:8000                        | -                                     |
| **API Documentation**  | http://localhost:8000/api/swagger/index.html | -                                     |
| **LLM API**            | http://localhost:8080                        | `Authorization: Bearer <token>`       |
| **Response API**       | http://localhost:8082                        | `Authorization: Bearer <token>`       |
| **Media API**          | http://localhost:8285                        | `Authorization: Bearer <token>`       |
| **MCP Tools**          | http://localhost:8091                        | `Authorization: Bearer <token>`       |
| **Memory Tools**       | http://localhost:8090                        | `Authorization: Bearer <token>`       |
| **Realtime API**       | http://localhost:8186                        | `Authorization: Bearer <token>`       |
| **Keycloak Console**   | http://localhost:8085                        | admin/admin                           |
| **Grafana Dashboards** | http://localhost:3331                        | admin/admin (after `make monitor-up`) |
| **Prometheus**         | http://localhost:9090                        | - (after `make monitor-up`)           |
| **Jaeger Tracing**     | http://localhost:16686                       | - (after `make monitor-up`)           |

**Service activation**:

- vLLM starts only if you choose the local provider (GPU). Use the remote provider option for a lighter stack.
- MCP Tools and Vector Store always run; the search choice only affects which search provider is used.
- Media API is optional based on the wizard choice.

## Your First API Call

### 1. Get a Guest Token via Kong

```bash
curl -X POST http://localhost:8000/llm/auth/guest-login
```

All traffic to `http://localhost:8000` flows through the Kong gateway, which validates Keycloak-issued JWTs or API keys (use `Authorization: Bearer <token>` or `X-API-Key: sk_*` headers).

Response:

```json
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci...",
  "expires_in": 300
}
```

### 2. Make a Chat Completion Request

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
 "messages": [
 {"role": "user", "content": "What is the capital of France?"}
 ],
 "stream": false
 }'
```

### 3. Try Streaming

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
 "messages": [
 {"role": "user", "content": "Tell me a short story"}
 ],
 "stream": true
 }'
```

### 4. Use MCP Tools

```bash
# List available tools
curl -X POST http://localhost:8000/v1/mcp \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 1,
 "method": "tools/list"
 }'

# Google search
curl -X POST http://localhost:8000/v1/mcp \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 2,
 "method": "tools/call",
 "params": {
 "name": "google_search",
 "arguments": {
 "q": "latest AI news"
 }
 }
 }'
```

## Enable Monitoring (Optional)

To enable full observability stack:

```bash
make monitor-up
```

Access:

- **Grafana**: http://localhost:3331 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

## Common Commands

```bash
# View logs
make logs        # All services
make logs-api    # API profile (LLM, Response, Media)
make logs-mcp    # MCP Tools profile

# Check status
make health-check # Hit health endpoints
docker compose ps # Container status

# Restart services
make restart-full   # Restart everything
make restart-api    # Restart API profile
make restart-llm-api # Restart only LLM API

# Stop services
make down       # Stop all containers (keeps volumes)
make down-clean # Stop containers and remove volumes
```

## Troubleshooting

### Services won't start

```bash
# Check Docker
docker --version
docker compose version

# Check status
make health-check
docker compose ps

# View errors
make logs

# Full reset
make down
make down-clean
make setup
make up-full
```

### Port conflicts

If you get port binding errors:

```bash
# Check what's using ports
# Windows PowerShell:
netstat -ano | findstr "8000 8080 8085"

# macOS/Linux:
lsof -i:8000
lsof -i:8080
lsof -i:8085

# Kill conflicting processes or change ports in .env
```

### vLLM GPU issues

```bash
# Verify GPU availability
docker run --rm --gpus all nvidia/cuda:11.8.0-base-ubuntu22.04 nvidia-smi
```

If no GPU is detected:

- Rerun `make quickstart` and choose the remote API option (skips local vLLM)
- Or run `make up-vllm-cpu` to start the CPU-only vLLM profile when testing locally

### Database connection errors

```bash
# Reset database
make db-reset

# Check database logs
docker compose logs api-db

# Verify connection
make db-console
```

### API returns 401 Unauthorized

- Check token hasn't expired (default: 5 minutes)
- Get new guest token: `curl -X POST http://localhost:8000/llm/auth/guest-login`
- Check `Authorization: Bearer <token>` header is set

## Update configuration later

```bash
make quickstart  # rerun the wizard and choose Y to overwrite .env
```

The wizard safely reuses existing values and updates your choices without manual edits.

## What's Next?

Now that you have Jan Server running:

1. **Explore the API**:

- [API Reference](api/README.md)
- [API Examples](api/examples/README.md)
- [Swagger UI](http://localhost:8000/api/swagger/index.html)

2. **Learn Development**:

- [Development Guide](guides/development.md)
- [Development Guide - Dev-Full Mode](guides/development.md#dev-full-mode-hybrid-debugging) (recommended for development)
- [Testing Guide](guides/testing.md)

3. **Understand Architecture**:

- [Architecture Overview](architecture/README.md)
- [System Design](architecture/system-design.md)
- [Security Model](architecture/security.md)

4. **Deploy to Production**:

- [Deployment Guide](guides/deployment.md)
- [Monitoring Guide](guides/monitoring.md)

## Need Help?

- [Full Documentation](README.md)
- [Report Issues](https://github.com/janhq/jan-server/issues)
- [Discussions](https://github.com/janhq/jan-server/discussions)
- [Troubleshooting Guide](guides/troubleshooting.md)

---

**Quick Reference**: `make help` | **All Commands**: `make help-all`
