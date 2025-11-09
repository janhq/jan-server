# Getting Started with Jan Server

Welcome! This guide will help you get Jan Server up and running in minutes.

## Prerequisites

Before you begin, ensure you have:

- **Docker Desktop** (Windows/macOS) or **Docker Engine + Docker Compose** (Linux)
- **Make** (usually pre-installed on macOS/Linux, [install on Windows](https://gnuwin32.sourceforge.net/packages/make.htm))
- **Git**
- At least 8GB RAM available
- For GPU inference: NVIDIA GPU with CUDA support

Optional (for development):
- Go 1.21+ 
- Node.js 18+ (for Newman tests)

## Quick Setup

### 1. Clone the Repository

```bash
git clone https://github.com/janhq/jan-server.git
cd jan-server
```

### 2. Configure Environment

```bash
# Create .env file from template
cp .env.example .env

# Edit .env and set required values:
# - SERPER_API_KEY (get from https://serper.dev)
# - HF_TOKEN (get from https://huggingface.co/settings/tokens)
# - Other passwords/secrets as needed
```

### 3. Run Setup

```bash
make setup
```

This command will:
- Check dependencies (Docker, Make)
- Create necessary directories
- Validate configuration
- Pull required Docker images

### 4. Start Services

```bash
# Start full stack (CPU inference)
make up-full

# OR with GPU inference
make up-gpu
```

Wait for all services to start (30-60 seconds). You can monitor progress with:
```bash
make logs-llm-api
```

### 5. Verify Installation

```bash
make health-check
```

You should see all services reporting as healthy.

## Access Services

Once running, you can access:

| Service | URL | Credentials |
|---------|-----|-------------|
| **API Gateway** | http://localhost:8000 | - |
| **API Documentation** | http://localhost:8000/v1/swagger/ | - |
| **LLM API** | http://localhost:8080 | - |
| **Media API** | http://localhost:8285 | `X-Media-Service-Key: <key>` |
| **Keycloak Console** | http://localhost:8085 | admin/admin |
| **Grafana Dashboards** | http://localhost:3001 | admin/admin (after `make monitor-up`) |
| **Prometheus** | http://localhost:9090 | - (after `make monitor-up`) |
| **Jaeger Tracing** | http://localhost:16686 | - (after `make monitor-up`) |

## Your First API Call

### 1. Get a Guest Token

```bash
curl -X POST http://localhost:8000/auth/guest
```

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
- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

## Common Commands

```bash
# View logs
make logs-llm-api         # API logs
make logs-error           # Error logs only

# Check status
make health-check         # All services health
make dev-status           # Detailed status

# Restart services
make restart              # Restart all
make restart-llm-api      # Restart API only

# Stop services
make down                 # Stop all (keeps data)
make clean                # Stop and remove all data
```

## Troubleshooting

### Services won't start

```bash
# Check Docker
docker --version
docker compose version

# Check status
make dev-status

# View errors
make logs-error

# Full reset
make down
make clean
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
lsof -i :8000
lsof -i :8080
lsof -i :8085

# Kill conflicting processes or change ports in .env
```

### vLLM GPU issues

```bash
# Verify GPU available
docker run --rm --gpus all nvidia/cuda:11.8.0-base-ubuntu22.04 nvidia-smi

# If no GPU, use CPU mode
make down
make up-cpu
```

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
- Get new guest token: `curl -X POST http://localhost:8000/auth/guest`
- Check `Authorization: Bearer <token>` header is set

## What's Next?

Now that you have Jan Server running:

1. **Explore the API**:
   - [API Reference](../api/README.md)
   - [API Examples](../api/llm-api/examples.md)
   - [Swagger UI](http://localhost:8000/v1/swagger/)

2. **Learn Development**:
   - [Development Guide](../guides/development.md)
   - [Hybrid Mode](../guides/hybrid-mode.md) (recommended for development)
   - [Testing Guide](../guides/testing.md)

3. **Understand Architecture**:
   - [Architecture Overview](../architecture/README.md)
   - [System Design](../architecture/system-design.md)
   - [Security Model](../architecture/security.md)

4. **Deploy to Production**:
   - [Deployment Guide](../guides/deployment.md)
   - [Monitoring Guide](../guides/monitoring.md)

## Need Help?

- 📚 [Full Documentation](../README.md)
- 🐛 [Report Issues](https://github.com/janhq/jan-server/issues)
- 💬 [Discussions](https://github.com/janhq/jan-server/discussions)
- 🔍 [Troubleshooting Guide](../guides/troubleshooting.md)

---

**Quick Reference**: `make help` | **All Commands**: `make help-all`
