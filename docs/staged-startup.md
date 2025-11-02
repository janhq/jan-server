# Staged Startup Guide

This guide explains how to start the Jan Server components in stages for better debugging and control.

## Overview

The system is now configured with Docker Compose profiles that allow you to:
1. Start infrastructure services first (databases, Keycloak, vLLM, GuestAuth)
2. Start llm-api when ready (after debugging/fixing issues)
3. Start Kong API Gateway last (after llm-api is healthy)

## Quick Start - Staged GPU Mode

### Step 1: Start Infrastructure
```powershell
make up-gpu-infra
```

This starts:
- PostgreSQL databases (api-db, keycloak-db)
- Keycloak
- GuestAuth
- vLLM GPU inference server
- OpenTelemetry Collector
- Database migrations

**What's NOT started:** llm-api, Kong

### Step 2: Start LLM API (when ready)
```powershell
make up-gpu-llm-api
```

This starts:
- llm-api service

Wait for llm-api to be healthy before proceeding.

### Step 3: Start Kong Gateway (last)
```powershell
make up-gpu-kong
```

This starts:
- Kong API Gateway (routes to llm-api)

---

## Alternative: Start Everything at Once

If you want the old behavior (start everything including llm-api and Kong):

```powershell
make up-gpu-full
```

Or the original commands still work:
```powershell
make up-gpu  # Same as up-gpu-full
make up-cpu  # CPU mode, everything
```

---

## Manual Docker Compose Commands

### GPU Mode - Infrastructure Only
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile gpu up -d --build
```

### GPU Mode - Add LLM API
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile gpu --profile llm-api up -d --build
```

### GPU Mode - Add Kong
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile gpu --profile kong up -d
```

### GPU Mode - Start All
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile gpu --profile full up -d --build
```

---

## CPU Mode - Staged Startup

### Step 1: Infrastructure
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile cpu up -d --build
```

### Step 2: LLM API
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile cpu --profile llm-api up -d --build
```

### Step 3: Kong
```powershell
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile cpu --profile kong up -d
```

---

## Debugging Workflow

### 1. Check Infrastructure Health
```powershell
docker compose ps
docker compose logs -f api-db
docker compose logs -f keycloak
docker compose logs -f vllm-jan-gpu-1
```

### 2. Fix LLM API Issues
```powershell
# View logs
docker logs jan-server-llm-api-1 --tail=100 -f

# Rebuild and restart just llm-api
make up-gpu-llm-api

# Or manually
docker compose -f docker-compose.yml -f docker-compose.vllm.yml --profile gpu --profile llm-api up -d --build
```

### 3. Start Kong Only When Ready
```powershell
# Check llm-api health first
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz

# If healthy, start Kong
make up-gpu-kong
```

---

## Service Dependencies

```
┌─────────────────┐
│   Databases     │ ← Always start first
│  (api-db,       │
│   keycloak-db)  │
└────────┬────────┘
         │
         ├──────────┬──────────┬──────────┐
         ▼          ▼          ▼          ▼
    ┌─────────┐ ┌──────┐ ┌────────┐ ┌──────────┐
    │Keycloak │ │ vLLM │ │ Migrate│ │   OTEL   │
    └────┬────┘ └──┬───┘ └────────┘ └──────────┘
         │         │
         └────┬────┘
              ▼
        ┌──────────┐
        │ GuestAuth│
        └──────────┘
              │
              ▼
        ┌──────────┐ ← Start when ready (profile: llm-api)
        │ LLM-API  │
        └────┬─────┘
             │
             ▼
        ┌────────┐  ← Start last (profile: kong)
        │  Kong  │
        └────────┘
```

---

## Available Make Targets

### GPU Mode
| Target | Description |
|--------|-------------|
| `make up-gpu-infra` | Start infrastructure only (DBs, Keycloak, vLLM, GuestAuth) |
| `make up-gpu-llm-api` | Start infrastructure + llm-api |
| `make up-gpu-kong` | Start infrastructure + llm-api + Kong |
| `make up-gpu-full` | Start everything (same as `make up-gpu`) |
| `make up-gpu-only` | Start only vLLM GPU server |

### CPU Mode
| Target | Description |
|--------|-------------|
| `make up-cpu` | Start everything with CPU vLLM |
| `make up-cpu-only` | Start only vLLM CPU server |

### Infrastructure Only
| Target | Description |
|--------|-------------|
| `make up-infra` | Start base infrastructure (no vLLM, no llm-api, no Kong) |
| `make up-llm-api` | Start infrastructure + llm-api |
| `make up-kong` | Start infrastructure + llm-api + Kong |
| `make up-full` | Start everything (no vLLM) |

### Utility
| Target | Description |
|--------|-------------|
| `make down` | Stop all services and remove volumes |
| `make logs` | Follow all service logs |

---

## Example: Debugging JSONB Issue

```powershell
# 1. Start infrastructure
make up-gpu-infra

# 2. Wait for everything to be healthy
docker compose ps

# 3. Check vLLM is ready
curl http://localhost:8000/v1/models

# 4. Manually test database connection
docker exec -it jan-server-api-db-1 psql -U llm_api -d llm_api -c "\d models"

# 5. Fix code in services/llm-api/

# 6. Rebuild and start llm-api
make up-gpu-llm-api

# 7. Watch logs
docker logs jan-server-llm-api-1 -f

# 8. If successful, start Kong
make up-gpu-kong

# 9. Test through Kong
curl http://localhost:8000/v1/models
```

---

## Port Reference

| Service | Internal Port | External Port | Notes |
|---------|--------------|---------------|-------|
| Kong | 8000 | 8000 | Public API Gateway |
| llm-api | 8080 | 8080 | LLM API (direct access) |
| GuestAuth | 8090 | 8090 | Guest authentication |
| Keycloak | 8080 | 8085 | Identity provider |
| vLLM GPU | 8000 | - | Internal only |
| vLLM CPU | 8001 | - | Internal only |
| api-db | 5432 | - | Internal only |
| keycloak-db | 5432 | - | Internal only |
| OTEL Collector | 4318 | 4318 | Telemetry |

---

## Troubleshooting

### llm-api Won't Start
- Check logs: `docker logs jan-server-llm-api-1 --tail=200`
- Verify database is ready: `docker compose ps api-db`
- Verify Keycloak is ready: `curl http://localhost:8085/health/ready`
- Check migrations ran: `docker compose logs llm-api`

### Kong Can't Connect to llm-api
- Verify llm-api is healthy: `curl http://localhost:8080/healthz`
- Check Kong logs: `docker compose logs kong`
- Restart Kong: `docker compose --profile kong restart kong`

### vLLM Model Not Loading
- Check HuggingFace token: verify `HF_TOKEN` in `.env`
- Check GPU availability: `docker run --rm --gpus all nvidia/cuda:11.8.0-base-ubuntu22.04 nvidia-smi`
- Check vLLM logs: `docker compose logs vllm-jan-gpu-1`

---

## Clean Slate Restart

```powershell
# Stop everything
make down

# Remove all volumes (WARNING: deletes data)
docker volume prune -f

# Start fresh
make up-gpu-infra
make up-gpu-llm-api
make up-gpu-kong
```
