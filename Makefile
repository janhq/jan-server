# ============================================================================================================
# JAN SERVER MAKEFILE
# ============================================================================================================
#
# A comprehensive build system for Jan Server - a microservices-based LLM API platform
# with MCP (Model Context Protocol) tool integration.
#
# ============================================================================================================
# QUICK START
# ============================================================================================================
#
#   make quickstart              - Interactive setup and run (core: infra + API + MCP web search + web app)
#   make setup                   - Initial project setup (dependencies, networks, .env)
#   make cli-install             - Install jan-cli tool globally
#   make build-all               - Build all Docker images (including platform & web)
#   make up-full                 - Start services based on COMPOSE_PROFILES in .env
#   make up-platform             - Start platform web app (http://localhost:3000)
#   make up-web                  - Start web chat app (http://localhost:3001)
#   make dev-full                - Start all services with host.docker.internal support (for testing)
#   make swagger                 - Generate swagger docs and sync to platform
#   make sync-docs               - Sync /docs to apps/platform/content/docs
#   make sync-swagger            - Sync swagger files to apps/platform/api
#   make health-check            - Check if all services are healthy
#   make test-all                - Run all integration tests
#   make stop                    - Stop all services (keeps containers & volumes)
#   make down                    - Stop and remove containers (keeps volumes)
#   make down-clean              - Stop, remove containers and volumes (full cleanup)
#
# ============================================================================================================
# MAKEFILE STRUCTURE
# ============================================================================================================
#
# This Makefile is organized into the following sections:
#
#   1. SETUP & ENVIRONMENT       - Initial setup and dependency checks
#   2. BUILD TARGETS             - Building services, code quality, Swagger documentation
#   3. SERVICE MANAGEMENT        - Starting/stopping services (infra, API, MCP, vLLM, full stack)
#   4. DATABASE MANAGEMENT       - DB operations, migrations, backups, restore
#   5. MONITORING                - Observability stack (Prometheus, Grafana, Jaeger)
#   6. TESTING                   - Integration tests with API Test
#   7. DEVELOPER UTILITIES       - Development helpers (dev-full mode)
#   8. HEALTH CHECKS             - Service health validation
#
# Documentation:
#   docs/guides/development.md - Complete development guide
#   README.md                  - Project overview and quick reference
#
# ============================================================================================================
# VARIABLES
# ============================================================================================================

# Docker Compose
COMPOSE = docker compose
COMPOSE_DEV_FULL = docker compose -f docker-compose.yml -f docker-compose.dev-full.yml
MONITOR_COMPOSE = docker compose -f infra/docker/observability.yml

MEDIA_SERVICE_KEY ?= changeme-media-key
MEDIA_API_KEY ?= changeme-media-key

EMBED_TEST_URL = $(if $(strip $(EMBEDDING_SERVICE_URL)),$(strip $(EMBEDDING_SERVICE_URL)),http://localhost:8091)
EMBED_TEST_PROFILES = --profile infra --profile memory
EMBED_TEST_SERVICES = api-db memory-tools
ifeq ($(strip $(EMBEDDING_SERVICE_URL)),)
EMBED_TEST_PROFILES += --profile memory-mock
EMBED_TEST_SERVICES += bge-m3
endif

# ============================================================================================================
# SECTION 1: SETUP & ENVIRONMENT
# ============================================================================================================

.PHONY: setup check-deps install-deps setup-and-run quickstart

setup-and-run quickstart:
	@echo "Starting interactive setup and run..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 setup-and-run --skip-realtime --skip-memory
else
	@bash tools/jan-cli.sh setup-and-run --skip-realtime --skip-memory
endif

setup:
	@echo "Running setup via jan-cli..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 dev setup
else
	@bash tools/jan-cli.sh dev setup
endif

check-deps:
	@echo "Checking dependencies..."
	@docker --version >/dev/null 2>&1 || echo "Docker not found"
	@docker compose version >/dev/null 2>&1 || echo "Docker Compose V2 not found"
	@go version >/dev/null 2>&1 || echo "Go not found (optional)"
	@echo "Dependency check complete"

install-deps:
	@echo "Installing development dependencies..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo " Development dependencies installed"


# ============================================================================================================
# SECTION 3: BUILD TARGETS
# ============================================================================================================

.PHONY: build build-api build-mcp build-memory build-realtime build-all clean-build build-llm-api build-media-api build-response-api build-realtime-api build-memory-tools build-platform-docker build-web-docker

build: build-api build-mcp build-memory

build-api: build-llm-api build-media-api build-response-api

build-realtime: build-realtime-api

build-memory: build-memory-tools

build-llm-api:
	@echo "Building LLM API..."
ifeq ($(OS),Windows_NT)
	@cd services/llm-api && go build -o bin/llm-api.exe ./cmd/server
else
	@cd services/llm-api && go build -o bin/llm-api ./cmd/server
endif
	@echo " LLM API built: services/llm-api/bin/llm-api"

build-media-api:
	@echo "Building Media API..."
ifeq ($(OS),Windows_NT)
	@cd services/media-api && go build -o bin/media-api.exe ./cmd/server
else
	@cd services/media-api && go build -o bin/media-api ./cmd/server
endif
	@echo " Media API built: services/media-api/bin/media-api"

build-response-api:
	@echo "Building Response API..."
ifeq ($(OS),Windows_NT)
	@cd services/response-api && go build -o bin/response-api.exe ./cmd/server
else
	@cd services/response-api && go build -o bin/response-api ./cmd/server
endif
	@echo " Response API built: services/response-api/bin/response-api"

build-realtime-api:
	@echo "Building Realtime API..."
ifeq ($(OS),Windows_NT)
	@cd services/realtime-api && go build -o bin/realtime-api.exe ./cmd/server
else
	@cd services/realtime-api && go build -o bin/realtime-api ./cmd/server
endif
	@echo " Realtime API built: services/realtime-api/bin/realtime-api"

build-mcp:
	@echo "Building MCP Tools..."
ifeq ($(OS),Windows_NT)
	@cd services/mcp-tools && go build -o bin/mcp-tools.exe .
else
	@cd services/mcp-tools && go build -o bin/mcp-tools .
endif
	@echo " MCP Tools built: services/mcp-tools/bin/mcp-tools"

build-memory-tools:
	@echo "Building Memory Tools..."
ifeq ($(OS),Windows_NT)
	@cd services/memory-tools && go build -o bin/memory-tools.exe ./cmd/server
else
	@cd services/memory-tools && go build -o bin/memory-tools ./cmd/server
endif
	@echo " Memory Tools built: services/memory-tools/bin/memory-tools"

build-all:
	@echo "Building all Docker images..."
	$(COMPOSE) --profile full --profile platform --profile web build
	@echo " All services built"

build-platform-docker:
	@echo "Building Platform Docker image..."
	$(COMPOSE) --profile platform build platform
	@echo " Platform image built"

build-web-docker:
	@echo "Building Web Docker image..."
	$(COMPOSE) --profile web build web
	@echo " Web image built"

.PHONY: config-generate config-test config-drift-check config-help

config-generate:
	@echo "Generating configuration files from Go structs..."
	@cd tools/jan-cli && go run . config generate
	@echo " Configuration files generated:"
	@echo "  - config/defaults.yaml (auto-generated)"
	@echo "  - config/schema/*.schema.json (auto-generated)"

config-drift-check:
	@echo "Checking for configuration drift..."
	@cd tools/jan-cli && go run . config generate
ifeq ($(OS),Windows_NT)
	@git diff --exit-code config/ && echo " No configuration drift detected" || (echo " Configuration drift detected! Run 'make config-generate' to update." && exit 1)
else
	@git diff --exit-code config/ && echo " No configuration drift detected" || (echo " Configuration drift detected! Run 'make config-generate' to update." && exit 1)
endif

config-help:
	@echo "Configuration Management Targets:"
	@echo "  config-generate      Generate config files from Go structs (YAML, JSON schema)"
	@echo "  config-drift-check  Verify generated files are in sync with code"
	@echo ""
	@echo "Files auto-generated by config-generate:"
	@echo "  - config/defaults.yaml                Default configuration values"
	@echo "  - config/schema/*.schema.json         JSON Schemas for validation"
	@echo ""
	@echo "Usage:"
	@echo "  1. Update packages/go-common/config/types.go with your configuration changes"
	@echo "  2. Run 'make config-generate' to regenerate all files"
	@echo "  4. Use 'make config-drift-check' in CI to prevent drift  "

# --- CLI Tool ---

.PHONY: cli-install cli-build cli-clean

cli-build:
	@echo "Building jan-cli..."
	@cd tools/jan-cli && go build -o jan-cli$(if $(filter Windows_NT,$(OS)),.exe,) .
	@echo " jan-cli built successfully"

cli-install: cli-build
	@echo "Installing jan-cli to local bin directory..."
ifeq ($(OS),Windows_NT)
	@tools/jan-cli/jan-cli.exe install
else
	@tools/jan-cli/jan-cli install
endif

cli-clean:
	@echo "Cleaning jan-cli binary..."
	@rm -f tools/jan-cli/jan-cli tools/jan-cli/jan-cli.exe
	@echo " jan-cli binary removed"

# --- Swagger Documentation ---

.PHONY: swagger swagger-llm-api swagger-media-api swagger-mcp-tools swagger-response-api swagger-realtime-api swagger-combine swagger-install sync-docs sync-swagger

swagger: cli-build sync-swagger
	@echo "Generating Swagger documentation for all services..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate --combine
	@echo "Syncing swagger to platform..."
	@copy /Y "services\llm-api\docs\swagger\swagger-combined.json" "apps\platform\api\server.json" >nul 2>&1 || echo "swagger-combined.json not found"
	@copy /Y "services\llm-api\docs\swagger\swagger.yaml" "apps\platform\api\server.yaml" >nul 2>&1 || echo "swagger.yaml not found"
else
	@bash tools/jan-cli.sh swagger generate --combine
	@echo "Syncing swagger to platform..."
	@cp -f services/llm-api/docs/swagger/swagger-combined.json apps/platform/api/server.json 2>/dev/null || echo "swagger-combined.json not found"
	@cp -f services/llm-api/docs/swagger/swagger.yaml apps/platform/api/server.yaml 2>/dev/null || echo "swagger.yaml not found"
endif
	@echo " Swagger synced to apps/platform/api/"

sync-swagger:
	@echo "Syncing swagger files to platform..."
ifeq ($(OS),Windows_NT)
	@if exist "services\llm-api\docs\swagger\swagger-combined.json" copy /Y "services\llm-api\docs\swagger\swagger-combined.json" "apps\platform\api\server.json" >nul
	@if exist "services\llm-api\docs\swagger\swagger.yaml" copy /Y "services\llm-api\docs\swagger\swagger.yaml" "apps\platform\api\server.yaml" >nul
else
	@cp -f services/llm-api/docs/swagger/swagger-combined.json apps/platform/api/server.json 2>/dev/null || true
	@cp -f services/llm-api/docs/swagger/swagger.yaml apps/platform/api/server.yaml 2>/dev/null || true
endif
	@echo " Swagger synced to apps/platform/api/"

sync-docs: cli-build
	@echo "Syncing docs to platform content..."
ifeq ($(OS),Windows_NT)
	@cd tools/jan-cli && jan-cli.exe docs sync
else
	@cd tools/jan-cli && ./jan-cli docs sync
endif

swagger-llm-api:
	@echo "Generating Swagger for llm-api service..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate -s llm-api
else
	@bash tools/jan-cli.sh swagger generate -s llm-api
endif
	@echo " llm-api swagger generated at services/llm-api/docs/swagger"

swagger-media-api: cli-build
	@echo "Generating Swagger for media-api service..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate -s media-api
else
	@bash tools/jan-cli.sh swagger generate -s media-api
endif
	@echo " media-api swagger generated at services/media-api/docs/swagger"

swagger-mcp-tools: cli-build
	@echo "Generating Swagger for mcp-tools service..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate -s mcp-tools
else
	@bash tools/jan-cli.sh swagger generate -s mcp-tools
endif
	@echo " mcp-tools swagger generated at services/mcp-tools/docs/swagger"

swagger-response-api: cli-build
	@echo "Generating Swagger for response-api service..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate -s response-api
else
	@bash tools/jan-cli.sh swagger generate -s response-api
endif
	@echo " response-api swagger generated at services/response-api/docs/swagger"

swagger-realtime-api: cli-build
	@echo "Generating Swagger for realtime-api service..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger generate -s realtime-api
else
	@bash tools/jan-cli.sh swagger generate -s realtime-api
endif
	@echo " realtime-api swagger generated at services/realtime-api/docs/swagger"

swagger-combine: cli-build
	@echo \"Merging LLM API, MCP Tools, and Realtime API swagger specs...\"
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 swagger combine
else
	@bash tools/jan-cli.sh swagger combine
endif
	@echo \" Combined swagger created for unified API documentation\"

swagger-install:
	@echo "Installing swagger tools..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo " swag installed successfully"

# --- Code Quality ---

.PHONY: fmt lint vet

fmt:
	@echo "Formatting Go code..."
	@gofmt -w $$(go list -f '{{.Dir}}' ./...)
	@echo " Code formatted"

lint:
	@echo "Running linter..."
	@go vet ./...
	@echo " Linting complete"

vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo " Vet complete"

# ============================================================================================================
# SECTION 4: SERVICE MANAGEMENT
# ============================================================================================================

# --- Infrastructure Services ---

.PHONY: up-infra down-infra restart-infra logs-infra

up-infra:
	@echo "Starting infrastructure services..."
	$(COMPOSE) --profile infra up -d
	@echo " Infrastructure services started"
	@echo ""
	@echo "Services:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Keycloak:   http://localhost:8085"
	@echo "  - Kong:       http://localhost:8000"

down-infra:
	$(COMPOSE) --profile infra down

restart-infra:
	$(COMPOSE) --profile infra restart

logs-infra:
	$(COMPOSE) --profile infra logs -f

# --- LLM API Service ---

.PHONY: up-api down-api restart-api logs-api logs-media-api

up-api:
	@echo "Starting LLM API..."
	$(COMPOSE) --profile api up -d
	@echo " API services started:"
	@echo "   - LLM API:   http://localhost:8080"
	@echo "   - Media API: http://localhost:8285"

down-api:
	$(COMPOSE) --profile api down

restart-api:
	$(COMPOSE) --profile api restart

logs-api:
	$(COMPOSE) --profile api logs -f llm-api

logs-media-api:
	$(COMPOSE) --profile api logs -f media-api

# --- MCP Services ---

.PHONY: up-mcp down-mcp restart-mcp logs-mcp

up-mcp:
	@echo "Starting MCP services..."
	$(COMPOSE) --profile mcp up -d
	@echo " MCP services started"
	@echo ""
	@echo "Services:"
	@echo "  - MCP Tools:      http://localhost:8091"
	@echo "  - SearXNG:        http://localhost:8086"
	@echo "  - Vector Store:   http://localhost:3015"
	@echo "  - SandboxFusion:  http://localhost:3010"
	@echo ""
	@echo "Test MCP tools:"
	@echo "  curl -X POST http://localhost:8091/v1/mcp -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}'"

down-mcp:
	$(COMPOSE) --profile mcp down

restart-mcp:
	$(COMPOSE) --profile mcp restart

logs-mcp:
	$(COMPOSE) --profile mcp logs -f

# --- AIO Sandbox Services ---

.PHONY: up-aio down-aio restart-aio logs-aio aio-verify

up-aio:
	@echo "Starting AIO Sandbox..."
	$(COMPOSE) --profile aio up -d aio-sandbox
	@echo " AIO Sandbox started"
	@echo ""
	@echo "Services:"
	@echo "  - AIO Sandbox:    http://localhost:$${AIO_HOST_PORT:-8180}"
	@echo "  - API Docs:       http://localhost:$${AIO_HOST_PORT:-8180}/v1/docs"
	@echo ""
	@echo "Test AIO health:"
	@echo "  curl http://localhost:$${AIO_HOST_PORT:-8180}/v1/docs"

down-aio:
	$(COMPOSE) --profile aio down

restart-aio:
	$(COMPOSE) --profile aio restart aio-sandbox

logs-aio:
	$(COMPOSE) --profile aio logs -f aio-sandbox

aio-verify:
	@echo "Verifying AIO Sandbox..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "try { $$response = Invoke-WebRequest -Uri 'http://localhost:8180/v1/docs' -UseBasicParsing -TimeoutSec 5; if ($$response.StatusCode -eq 200) { Write-Host ' AIO Sandbox is healthy' } } catch { Write-Host ' AIO Sandbox is not responding' }"
	@echo ""
	@echo "Testing shell execution..."
	@powershell -Command "try { $$result = Invoke-RestMethod -Uri 'http://localhost:8180/v1/shell/exec' -Method POST -ContentType 'application/json' -Body '{\"command\": \"echo Hello from AIO Sandbox\"}' -TimeoutSec 10; Write-Host $$result.data.output } catch { Write-Host 'Shell exec failed:' $$_.Exception.Message }"
else
	@curl -s http://localhost:$${AIO_HOST_PORT:-8180}/v1/docs > /dev/null 2>&1 && echo " AIO Sandbox is healthy" || echo " AIO Sandbox is not responding"
	@echo ""
	@echo "Testing shell execution..."
	@curl -s -X POST http://localhost:$${AIO_HOST_PORT:-8180}/v1/shell/exec \
		-H "Content-Type: application/json" \
		-d '{"command": "echo Hello from AIO Sandbox"}' | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('output',d.get('message','No output')))" 2>/dev/null || echo "Shell exec failed"
endif

# --- vLLM Inference Services ---

.PHONY: up-vllm-gpu up-vllm-cpu down-vllm logs-vllm

up-vllm-gpu:
	@echo "Starting vLLM GPU inference..."
	$(COMPOSE) --profile gpu up -d
	@echo " vLLM GPU started at http://localhost:8101"
	@echo ""
	@echo "Test inference:"
	@echo "  curl http://localhost:8101/v1/models"

up-vllm-cpu:
	@echo "Starting vLLM CPU inference..."
	$(COMPOSE) --profile cpu up -d
	@echo " vLLM CPU started at http://localhost:8101"
	@echo ""
	@echo "Test inference:"
	@echo "  curl http://localhost:8101/v1/models"

down-vllm:
	@echo "Stopping vLLM services..."
	$(COMPOSE) --profile gpu --profile cpu down

logs-vllm:
	$(COMPOSE) --profile gpu --profile cpu logs -f

# --- Platform Web Application ---

.PHONY: up-platform down-platform restart-platform logs-platform build-platform

up-platform:
	@echo "Starting Platform web application..."
	@echo "Note: Platform requires infra services (Kong, Keycloak) to be running."
	@echo "Starting infra + platform..."
	$(COMPOSE) --profile infra --profile platform up -d
	@echo " Platform started"
	@echo ""
	@echo "Services:"
	@echo "  - Platform:  http://localhost:3000"
	@echo "  - Kong:      http://localhost:8000"
	@echo "  - Keycloak:  http://localhost:8085"

down-platform:
	$(COMPOSE) --profile platform down

restart-platform:
	$(COMPOSE) --profile platform restart

logs-platform:
	$(COMPOSE) --profile platform logs -f platform

build-platform:
	@echo "Building Platform Docker image..."
	$(COMPOSE) --profile platform build platform
	@echo " Platform image built"

# --- Web Application ---

.PHONY: up-web down-web restart-web logs-web build-web

up-web:
	@echo "Starting Web application..."
	@echo "Note: Web app requires infra services (Kong, Keycloak) to be running."
	@echo "Starting infra + web..."
	$(COMPOSE) --profile infra --profile web up -d
	@echo " Web app started"
	@echo ""
	@echo "Services:"
	@echo "  - Web App:   http://localhost:3001"
	@echo "  - Kong:      http://localhost:8000"
	@echo "  - Keycloak:  http://localhost:8085"

down-web:
	$(COMPOSE) --profile web down

restart-web:
	$(COMPOSE) --profile web restart

logs-web:
	$(COMPOSE) --profile web logs -f web

build-web:
	@echo "Building Web Docker image..."
	$(COMPOSE) --profile web build web
	@echo " Web image built"

# --- Full Stack ---

.PHONY: up-full down-full restart-full logs stop down down-clean dev-full dev-full-down dev-full-stop

up-full: ## Start full stack (all services in Docker)
	@echo "Starting services (based on COMPOSE_PROFILES in .env)..."
	$(COMPOSE) up -d
	@echo " Services started"
	@echo ""
	@echo "Infrastructure:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Keycloak:   http://localhost:8085 (admin/admin)"
	@echo "  - Kong:       http://localhost:8000"
	@echo ""
	@echo "Core Services:"
	@echo "  - LLM API:    http://localhost:8080"
	@echo "  - Media API:  http://localhost:8285"
	@echo "  - MCP Tools:  http://localhost:8091 (web search)"
	@echo ""
	@echo "Optional Services (add to COMPOSE_PROFILES in .env):"
	@echo "  - Platform:       http://localhost:3000 (profile: platform)"
	@echo "  - Web App:        http://localhost:3001 (profile: web)"
	@echo "  - Code Sandbox:   (profile: sandbox - for code execution)"
	@echo "  - Vector Store:   http://localhost:3015 (profile: vector)"
	@echo "  - Memory Tools:   http://localhost:8090 (profile: memory)"
	@echo "  - Realtime API:   http://localhost:8186 (profile: realtime)"
	@echo "  - vLLM:           http://localhost:8101 (profile: full)"
	@echo ""
	@echo "To enable optional services, edit COMPOSE_PROFILES in .env"
	@echo "To start monitoring stack: make monitor-up"

down-full:
	$(COMPOSE) down

restart-full:
	$(COMPOSE) restart

stop:
	@echo "Stopping all services (containers will be preserved)..."
	$(COMPOSE) stop
	@echo " All services stopped (containers preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To remove containers: make down"

down:
	@echo "Stopping and removing all containers (volumes will be preserved)..."
	$(COMPOSE) down
	@echo " All containers stopped and removed (volumes preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To clean volumes: make down-clean"

down-clean:
	@echo "Stopping and removing all containers and volumes..."
	$(COMPOSE) down -v
	@echo " All containers and volumes removed (full cleanup)"
	@echo ""
	@echo "To restart: make up-full"

logs:
	$(COMPOSE) logs -f

# --- Individual Service Control ---

.PHONY: restart-kong restart-keycloak restart-postgres

restart-kong:
	@echo "Restarting Kong..."
	$(COMPOSE) restart kong
ifeq ($(OS),Windows_NT)
	@powershell -Command "Start-Sleep -Seconds 3"
else
	@sleep 3
endif
	@echo " Kong restarted"

restart-keycloak:
	$(COMPOSE) restart keycloak

restart-postgres:
	$(COMPOSE) restart api-db

# ============================================================================================================
# SECTION 5: DATABASE MANAGEMENT
# ============================================================================================================

.PHONY: db-reset db-migrate db-console db-backup db-restore db-dump

db-reset:
	@echo "  WARNING: This will delete all database data!"
	@echo "Stopping and removing API database..."
	$(COMPOSE) stop api-db
	$(COMPOSE) rm -f api-db
	@docker volume rm jan-server_api-db-data 2>nul || echo Volume removed or didn't exist
	@echo " Database reset complete. Run 'make up-api' to restart."

db-migrate:
	@echo "Running database migrations..."
	$(COMPOSE) exec llm-api /app/llm-api migrate
	@echo " Migrations complete"

db-console:
	@echo "Opening database console..."
	$(COMPOSE) exec api-db psql -U jan_user -d jan_llm_api

db-backup:
	@echo "Backing up database..."
	@mkdir -p backups
	@$(COMPOSE) exec -T api-db pg_dump -U jan_user jan_llm_api > backups/db_backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo " Database backed up to backups/"

db-restore:
	@if [ -z "$(FILE)" ]; then \
		echo " FILE variable required. Usage: make db-restore FILE=backups/db_backup.sql"; \
		exit 1; \
	fi
	@echo "Restoring database from $(FILE)..."
	@cat $(FILE) | $(COMPOSE) exec -T api-db psql -U jan_user -d jan_llm_api
	@echo " Database restored"

db-dump:
	@echo "Dumping database schema..."
	@$(COMPOSE) exec api-db pg_dump -U jan_user -d jan_llm_api --schema-only

# ============================================================================================================
# SECTION 6: MONITORING
# ============================================================================================================

.PHONY: monitor-up monitor-down monitor-logs monitor-clean

monitor-up:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 monitor up
else
	@bash tools/jan-cli.sh monitor up
endif

monitor-down:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 monitor down
else
	@bash tools/jan-cli.sh monitor down
endif

monitor-logs:
	$(MONITOR_COMPOSE) logs -f

monitor-clean:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File tools/jan-cli.ps1 monitor reset
else
	@bash tools/jan-cli.sh monitor reset
endif

# --- Advanced Monitoring Targets (from monitoring improvement plan) ---




# =============================================================================
# API Test Targets
# =============================================================================

ifeq ($(OS),Windows_NT)
API_TEST := tools/jan-cli/jan-cli.exe api-test run
else
API_TEST := tools/jan-cli/jan-cli api-test run
endif

GATEWAY_URL ?= http://localhost:8000
TIMEOUT_MS ?= 30000
COLLECTIONS_DIR := tests/e2e/automation/collections
AUTH_MODE ?= guest
# Exclude memory.postman.json (no memory service), model-prompt-templates.postman.json (API not implemented),
# and user-management.postman.json (requires manual admin token setup)
COLLECTION_FILES := $(filter-out $(COLLECTIONS_DIR)/memory.postman.json $(COLLECTIONS_DIR)/model-prompt-templates.postman.json $(COLLECTIONS_DIR)/user-management.postman.json,$(wildcard $(COLLECTIONS_DIR)/*.postman.json))

# Base flags without auth (for targets that need custom auth)
API_TEST_BASE_FLAGS := --env-file tests/e2e/.env \
  --env-var gateway_url=$(GATEWAY_URL) \
  --env-var media_service_key=$(MEDIA_SERVICE_KEY) \
  --auto-models \
  --timeout-request $(TIMEOUT_MS)

# Full flags with default auth mode
API_TEST_FLAGS := $(API_TEST_BASE_FLAGS) --auto-auth $(AUTH_MODE) --debug

.PHONY: test-all test-auth test-conversation test-response test-response-aio test-response-artifact test-mcp-agent test-model test-media test-mcp test-user-management test-model-prompts test-image test-dev seed-response-artifact

test-all:
	$(API_TEST) $(COLLECTION_FILES) $(API_TEST_FLAGS) --timeout-request 120000

test-auth:
	$(API_TEST) $(COLLECTIONS_DIR)/auth.postman.json $(API_TEST_FLAGS)

test-conversation:
	$(API_TEST) $(COLLECTIONS_DIR)/conversation.postman.json $(API_TEST_FLAGS)

test-response:
	$(API_TEST) $(COLLECTIONS_DIR)/response.postman.json $(API_TEST_FLAGS) --timeout-request 120000

test-response-aio:
	$(API_TEST) $(COLLECTIONS_DIR)/response-aio.postman.json $(API_TEST_FLAGS) --timeout-request 180000

seed-response-artifact:
	@echo "Seeding response/artifact test data..."
	@$(COMPOSE) exec -T api-db psql -U jan_user -d jan_llm_api -v ON_ERROR_STOP=1 < tests/e2e/automation/seeds/response_artifact_seed.sql

test-response-artifact: seed-response-artifact
	$(API_TEST) $(COLLECTIONS_DIR)/response-artifact.postman.json $(API_TEST_FLAGS)

test-mcp-agent:
	$(API_TEST) $(COLLECTIONS_DIR)/mcp-agent.postman.json $(API_TEST_FLAGS) --timeout-request 120000

test-model:
	$(API_TEST) $(COLLECTIONS_DIR)/model.postman.json $(API_TEST_BASE_FLAGS) --auto-auth admin

test-media:
	$(API_TEST) $(COLLECTIONS_DIR)/media.postman.json $(API_TEST_FLAGS)

test-mcp:
	$(API_TEST) $(COLLECTIONS_DIR)/mcp-runtime.postman.json $(COLLECTIONS_DIR)/mcp-admin.postman.json $(API_TEST_FLAGS)

test-user-management:
	$(API_TEST) $(COLLECTIONS_DIR)/user-management.postman.json $(API_TEST_BASE_FLAGS) --auto-auth admin

test-model-prompts:
	$(API_TEST) $(COLLECTIONS_DIR)/model-prompt-templates.postman.json $(API_TEST_FLAGS)

test-image:
	$(API_TEST) $(COLLECTIONS_DIR)/image.postman.json $(API_TEST_FLAGS) --timeout-request 120000

test-memory:
	$(API_TEST) $(COLLECTIONS_DIR)/memory.postman.json $(API_TEST_FLAGS)

test-dev:
	$(API_TEST) $(COLLECTION_FILES) $(API_TEST_FLAGS) --bail


# ============================================================================================================
# SECTION 8: DEVELOPER UTILITIES
# ============================================================================================================

# --- Development Full Stack (with host.docker.internal support) ---

.PHONY: dev-full dev-full-stop dev-full-down

dev-full: ## Start development full stack with host.docker.internal support
	@echo "Starting development full stack with host.docker.internal support..."
	@echo ""
	@echo "This mode allows you to:"
	@echo "  1. Stop any Docker service: docker compose stop <service>"
	@echo "  2. Run it manually on host for debugging"
	@echo "  3. Kong will automatically route to host.docker.internal"
	@echo ""
	$(COMPOSE_DEV_FULL) --profile full up -d
	@echo ""
	@echo " Development full stack started!"
	@echo ""
	@echo "Infrastructure:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Keycloak:   http://localhost:8085 (admin/admin)"
	@echo "  - Kong:       http://localhost:8000 (with upstreams to host)"
	@echo ""
	@echo "Services (running in Docker):"
	@echo "  - LLM API:        http://localhost:8080"
	@echo "  - Media API:      http://localhost:8285"
	@echo "  - Response API:   http://localhost:8082"
	@echo "  - MCP Tools:      http://localhost:8091"
	@echo "  - SearXNG:        http://localhost:8086"
	@echo "  - Vector Store:   http://localhost:3015"
	@echo "  - SandboxFusion:  http://localhost:3010"
	@echo ""
	@echo "To run a service manually on host:"
	@echo "  1. Stop Docker service:"
	@echo "     docker compose stop llm-api"
	@echo ""
	@echo "  2. Run on host:"
ifeq ($(OS),Windows_NT)
	@echo "     jan-cli dev run llm-api"
else
	@echo "     jan-cli dev run llm-api"
endif
	@echo ""
	@echo "  3. Kong will automatically route requests to your host service"
	@echo ""
	@echo "Check service routing: curl http://localhost:8000/healthz"
	@echo ""
	@echo "Documentation: docs/guides/dev-full-mode.md"

dev-full-stop:
	@echo "Stopping dev-full services..."
	$(COMPOSE_DEV_FULL) --profile full stop
	@echo " Dev-full services stopped"

dev-full-down:
	@echo "Stopping and removing dev-full containers..."
	$(COMPOSE_DEV_FULL) --profile full down
	@echo " Dev-full containers removed"

# ============================================================================================================
# SECTION 9: HEALTH CHECKS
# ============================================================================================================

.PHONY: health-check health-api health-mcp health-infra

health-check:
ifeq ($(OS),Windows_NT)
	@echo ============================================
	@echo Checking All Services Health Status
	@echo ============================================
	@echo [Infrastructure Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8085 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Keycloak:   healthy' } catch { Write-Host '  Keycloak:   unhealthy' }"
	@powershell -Command "try { $$response = Invoke-WebRequest -Uri http://localhost:8000 -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue; if ($$response.StatusCode -ge 200 -and $$response.StatusCode -lt 500) { Write-Host '  Kong:       healthy' } else { Write-Host '  Kong:       unhealthy' } } catch { try { if ($$PSItem.Exception.Response.StatusCode.Value__ -eq 404) { Write-Host '  Kong:       healthy' } else { Write-Host '  Kong:       unhealthy' } } catch { Write-Host '  Kong:       unhealthy' } }"
	@powershell -Command "try { $$null = docker compose exec -T api-db pg_isready -U jan_user 2>&1 | Out-Null; if ($$LASTEXITCODE -eq 0) { Write-Host '  PostgreSQL: healthy' } else { Write-Host '  PostgreSQL: unhealthy' } } catch { Write-Host '  PostgreSQL: unhealthy' }"
	@echo.
	@echo [API Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8080/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  LLM API:      healthy' } catch { Write-Host '  LLM API:      unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8285/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Media API:    healthy' } catch { try { if ($$PSItem.Exception.Response.StatusCode.Value__ -eq 401) { Write-Host '  Media API:    healthy' } else { Write-Host '  Media API:    unhealthy' } } catch { Write-Host '  Media API:    unhealthy' } }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8186/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Realtime API: healthy' } catch { Write-Host '  Realtime API: not running' }"
	@echo.
	@echo [MCP Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8091/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  MCP Tools:      healthy' } catch { Write-Host '  MCP Tools:      unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3015/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Vector Store:   healthy' } catch { Write-Host '  Vector Store:   unhealthy' }"
	@echo.
	@echo [Optional Services - may show unhealthy if disabled]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8086 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  SearXNG:        healthy' } catch { Write-Host '  SearXNG:        not running' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3010 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  SandboxFusion:  healthy' } catch { Write-Host '  SandboxFusion:  not running' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8101/v1/models -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  vLLM:           healthy' } catch { Write-Host '  vLLM:           not running' }"
	@echo ============================================
else
	@echo "============================================"
	@echo "Checking All Services Health Status"
	@echo "============================================"
	@echo ""
	@echo "[Infrastructure Services]"
	@curl -sf http://localhost:8085 >/dev/null && echo "  Keycloak:   healthy" || echo "  Keycloak:   unhealthy"
	@curl -f --max-time 2 http://localhost:8000 >/dev/null 2>&1 || (curl --max-time 2 http://localhost:8000 2>&1 | grep -q "no Route matched" && echo "  Kong:       healthy" || echo "  Kong:       unhealthy")
	@$(COMPOSE) exec -T api-db pg_isready -U jan_user >/dev/null 2>&1 && echo "  PostgreSQL: healthy" || echo "  PostgreSQL: unhealthy"
	@echo ""
	@echo "[API Services]"
	@curl -sf http://localhost:8080/healthz >/dev/null && echo "  LLM API:      healthy" || echo "  LLM API:      unhealthy"
	@curl -s http://localhost:8285/healthz >/dev/null && echo "  Media API:    healthy" || (curl -s -w "%{http_code}" -o /dev/null http://localhost:8285/healthz | grep -q "401" && echo "  Media API:    healthy" || echo "  Media API:    unhealthy")
	@curl -sf http://localhost:8186/healthz >/dev/null && echo "  Realtime API: healthy" || echo "  Realtime API: not running"
	@echo ""
	@echo "[MCP Services]"
	@curl -sf http://localhost:8091/healthz >/dev/null && echo "  MCP Tools:      healthy" || echo "  MCP Tools:      unhealthy"
	@curl -sf http://localhost:3015/healthz >/dev/null && echo "  Vector Store:   healthy" || echo "  Vector Store:   unhealthy"
	@echo ""
	@echo "[Optional Services - may show 'not running' if disabled]"
	@curl -sf http://localhost:8086 >/dev/null && echo "  SearXNG:        healthy" || echo "  SearXNG:        not running"
	@curl -sf http://localhost:3010 >/dev/null && echo "  SandboxFusion:  healthy" || echo "  SandboxFusion:  not running"
	@curl -sf http://localhost:8101/v1/models >/dev/null && echo "  vLLM:           healthy" || echo "  vLLM:           not running"
	@echo ""
	@echo "============================================"
endif

health-infra:
ifeq ($(OS),Windows_NT)
	@echo Checking infrastructure services...
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8085 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK Keycloak: healthy' } catch { Write-Host 'ERROR Keycloak: unhealthy' }"
	@powershell -Command "try { $$response = Invoke-WebRequest -Uri http://localhost:8000 -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue; if ($$response.StatusCode -ge 200 -and $$response.StatusCode -lt 500) { Write-Host 'OK Kong: healthy' } else { Write-Host 'ERROR Kong: unhealthy' } } catch { try { if ($$PSItem.Exception.Response.StatusCode.Value__ -eq 404) { Write-Host 'OK Kong: healthy' } else { Write-Host 'ERROR Kong: unhealthy' } } catch { Write-Host 'ERROR Kong: unhealthy' } }"
	@powershell -Command "try { $$null = docker compose exec -T api-db pg_isready -U jan_user 2>&1 | Out-Null; if ($$LASTEXITCODE -eq 0) { Write-Host 'OK PostgreSQL: healthy' } else { Write-Host 'ERROR PostgreSQL: unhealthy' } } catch { Write-Host 'ERROR PostgreSQL: unhealthy' }"
else
	@curl -sf http://localhost:8085 >/dev/null && echo " Keycloak: healthy" || echo " Keycloak: unhealthy"
	@curl -f --max-time 2 http://localhost:8000 >/dev/null 2>&1 || (curl --max-time 2 http://localhost:8000 2>&1 | grep -q "no Route matched" && echo " Kong: healthy" || echo " Kong: unhealthy")
	@$(COMPOSE) exec -T api-db pg_isready -U jan_user >/dev/null 2>&1 && echo " PostgreSQL: healthy" || echo " PostgreSQL: unhealthy"
endif

health-api:
ifeq ($(OS),Windows_NT)
	@powershell -Command "try { Invoke-WebRequest -Uri http://localhost:8080/healthz -UseBasicParsing | Select-Object -ExpandProperty Content | ConvertFrom-Json | ConvertTo-Json } catch { Write-Host 'ERROR LLM API not responding' }"
	@powershell -Command "try { Invoke-WebRequest -Uri http://localhost:8285/healthz -UseBasicParsing | Select-Object -ExpandProperty Content | ConvertFrom-Json | ConvertTo-Json } catch { Write-Host 'ERROR Media API not responding' }"
else
	@curl -sf http://localhost:8080/healthz | jq || echo "? LLM API not responding"
	@curl -sf http://localhost:8285/healthz | jq || echo "? Media API not responding"
endif

health-mcp:
ifeq ($(OS),Windows_NT)
	@echo Checking MCP services...
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8091/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK MCP Tools: healthy' } catch { Write-Host 'ERROR MCP Tools: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8086 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK SearXNG: healthy' } catch { Write-Host 'ERROR SearXNG: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3015/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK Vector Store: healthy' } catch { Write-Host 'ERROR Vector Store: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3010 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK SandboxFusion: healthy' } catch { Write-Host 'ERROR SandboxFusion: unhealthy' }"
else
	@curl -sf http://localhost:8091/healthz >/dev/null && echo " MCP Tools: healthy" || echo " MCP Tools: unhealthy"
	@curl -sf http://localhost:8086 >/dev/null && echo " SearXNG: healthy" || echo " SearXNG: unhealthy"
	@curl -sf http://localhost:3015/healthz >/dev/null && echo " Vector Store: healthy" || echo " Vector Store: unhealthy"
	@curl -sf http://localhost:3010 >/dev/null && echo " SandboxFusion: healthy" || echo " SandboxFusion: unhealthy"
endif

# ============================================================================================================
# END OF MAKEFILE
# ============================================================================================================
