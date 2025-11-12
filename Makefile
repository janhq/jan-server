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
#   make setup          - Initial project setup (dependencies, networks, .env)
#   make build-all      - Build all Docker images
#   make up-full        - Start all services (infrastructure + API + MCP)
#   make dev-full       - Start all services with host.docker.internal support (for testing)
#   make health-check   - Check if all services are healthy
#   make test-all       - Run all integration tests
#   make stop           - Stop all services (keeps containers & volumes)
#   make down           - Stop and remove containers (keeps volumes)
#   make down-clean     - Stop, remove containers and volumes (full cleanup)
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
#   6. TESTING                   - Integration tests with Newman
#   7. DEVELOPER UTILITIES       - Development helpers (dev-full mode)
#   8. HEALTH CHECKS             - Service health validation
#
# Documentation:
#   📖 docs/guides/development.md - Complete development guide
#   📖 README.md                  - Project overview and quick reference
#
# ============================================================================================================
# VARIABLES
# ============================================================================================================

COMPOSE = docker compose
COMPOSE_DEV_FULL = docker compose -f docker-compose.yml -f docker-compose.dev-full.yml
MONITOR_COMPOSE = docker compose -f docker/observability.yml
NEWMAN = newman
NEWMAN_AUTH_COLLECTION = tests/automation/auth-postman-scripts.json
NEWMAN_CONVERSATION_COLLECTION = tests/automation/conversations-postman-scripts.json
NEWMAN_RESPONSES_COLLECTION = tests/automation/responses-postman-scripts.json
NEWMAN_MEDIA_COLLECTION = tests/automation/media-postman-scripts.json
NEWMAN_MCP_COLLECTION = tests/automation/mcp-postman-scripts.json
NEWMAN_E2E_COLLECTION = tests/automation/test-all.postman.json

MEDIA_SERVICE_KEY ?= changeme-media-key
MEDIA_API_KEY ?= changeme-media-key

# ============================================================================================================
# SECTION 1: SETUP & ENVIRONMENT
# ============================================================================================================

.PHONY: setup check-deps install-deps

setup:
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/setup.ps1
else
	@bash scripts/setup.sh
endif

check-deps:
	@echo "Checking dependencies..."
ifeq ($(OS),Windows_NT)
	@docker --version >nul 2>&1 || echo "Docker not found"
	@docker compose version >nul 2>&1 || echo "Docker Compose V2 not found"
	@go version >nul 2>&1 || echo "Go not found (optional)"
	@newman --version >nul 2>&1 || echo "Newman not found (optional)"
else
	@docker --version >/dev/null 2>&1 || echo "Docker not found"
	@docker compose version >/dev/null 2>&1 || echo "Docker Compose V2 not found"
	@go version >/dev/null 2>&1 || echo "Go not found (optional)"
	@newman --version >/dev/null 2>&1 || echo "Newman not found (optional)"
endif
	@echo "Dependency check complete"

install-deps:
	@echo "Installing development dependencies..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo " Development dependencies installed"


# ============================================================================================================
# SECTION 3: BUILD TARGETS
# ============================================================================================================

.PHONY: build build-api build-mcp build-all clean-build build-llm-api build-media-api

build: build-api build-mcp

build-api: build-llm-api build-media-api

build-llm-api:
	@echo "Building LLM API..."
	@cd services/llm-api && go build -o bin/llm-api ./cmd/server
	@echo " LLM API built: services/llm-api/bin/llm-api"

build-media-api:
	@echo "Building Media API..."
	@cd services/media-api && go build -o bin/media-api ./cmd/server
	@echo " Media API built: services/media-api/bin/media-api"

build-mcp:
	@echo "Building MCP Tools..."
	@cd services/mcp-tools && go build -o bin/mcp-tools .
	@echo " MCP Tools built: services/mcp-tools/bin/mcp-tools"

build-all:
	@echo "Building all Docker images..."
	$(COMPOSE) --profile full build
	@echo " All services built"

clean-build:
	@echo "Cleaning build artifacts..."
	@rm -rf services/llm-api/bin
	@rm -rf services/media-api/bin
	@rm -rf services/mcp-tools/bin
	@echo " Build artifacts cleaned"

# --- Swagger Documentation ---

.PHONY: swagger swagger-llm-api swagger-media-api swagger-mcp-tools swagger-response-api swagger-combine swagger-install

swagger:
	@echo "Generating Swagger documentation for all services..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/generate-swagger.ps1
else
	@bash scripts/generate-swagger.sh
endif
	@echo ""
	@echo "Combining swagger specs..."
	@$(MAKE) swagger-combine

swagger-llm-api:
	@echo "Generating Swagger for llm-api service..."
	@cd services/llm-api && go run github.com/swaggo/swag/cmd/swag@v1.8.12 init \
		--dir ./cmd/server,./internal/interfaces/httpserver/routes \
		--generalInfo server.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo " llm-api swagger generated at services/llm-api/docs/swagger"

swagger-media-api:
	@echo "Generating Swagger for media-api service..."
	@cd services/media-api && go run github.com/swaggo/swag/cmd/swag@v1.8.12 init \
		--dir ./cmd/server,./internal/interfaces/httpserver/handlers,./internal/interfaces/httpserver/routes/v1 \
		--generalInfo server.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo " media-api swagger generated at services/media-api/docs/swagger"

swagger-mcp-tools:
	@echo "Generating Swagger for mcp-tools service..."
	@cd services/mcp-tools && go run github.com/swaggo/swag/cmd/swag@v1.8.12 init \
		--dir . \
		--generalInfo main.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo " mcp-tools swagger generated at services/mcp-tools/docs/swagger"

swagger-response-api:
	@echo "Generating Swagger for response-api service..."
	@cd services/response-api && go run github.com/swaggo/swag/cmd/swag@v1.8.12 init \
		--dir ./cmd/server,./internal/interfaces/httpserver/handlers,./internal/interfaces/httpserver/routes/v1 \
		--generalInfo server.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo " response-api swagger generated at services/response-api/docs/swagger"

swagger-combine:
	@echo "Merging LLM API and MCP Tools swagger specs..."
	@go run scripts/swagger-combine.go \
		-llm-api services/llm-api/docs/swagger/swagger.json \
		-mcp-tools services/mcp-tools/docs/swagger/swagger.json \
		-output services/llm-api/docs/swagger/swagger-combined.json
	@echo " Combined swagger created for unified API documentation"

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

# --- Full Stack ---

.PHONY: up-full down-full restart-full logs stop down down-clean dev-full dev-full-down dev-full-stop

up-full:
	@echo "Starting full stack..."
	$(COMPOSE) --profile full up -d
	@echo " Full stack started"
	@echo ""
	@echo "Infrastructure:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Keycloak:   http://localhost:8085 (admin/admin)"
	@echo "  - Kong:       http://localhost:8000"
	@echo ""
	@echo "Services:"
	@echo "  - LLM API:        http://localhost:8080"
	@echo "  - MCP Tools:      http://localhost:8091"
	@echo "  - SearXNG:        http://localhost:8086"
	@echo "  - Vector Store:   http://localhost:3015"
	@echo "  - SandboxFusion:  http://localhost:3010"
	@echo "  - vLLM GPU:       http://localhost:8101"

down-full:
	$(COMPOSE) --profile full down

restart-full:
	$(COMPOSE) --profile full restart

stop:
	@echo "Stopping all services (containers will be preserved)..."
	$(COMPOSE) --profile full stop
	@echo " All services stopped (containers preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To remove containers: make down"

down:
	@echo "Stopping and removing all containers (volumes will be preserved)..."
	$(COMPOSE) --profile full down
	@echo " All containers stopped and removed (volumes preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To clean volumes: make down-clean"

down-clean:
	@echo "Stopping and removing all containers and volumes..."
	$(COMPOSE) --profile full down -v
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
	@docker volume rm jan-server_api-db-data || true
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
	@echo "Starting observability stack..."
	$(MONITOR_COMPOSE) up -d
	@echo " Monitoring stack started"
	@echo ""
	@echo "Dashboards:"
	@echo "  - Grafana:    http://localhost:3001 (admin/admin)"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Jaeger:     http://localhost:16686"

monitor-down:
	@echo "Stopping monitoring stack..."
	$(MONITOR_COMPOSE) down
	@echo " Monitoring stack stopped"

monitor-logs:
	$(MONITOR_COMPOSE) logs -f

monitor-clean:
	@echo "Stopping monitoring stack and removing volumes..."
	$(MONITOR_COMPOSE) down -v
	@echo " Monitoring stack cleaned"

# --- Integration Tests (Newman) ---

.PHONY: test-all test-auth test-conversations test-response test-media test-mcp-integration test-e2e newman-debug

test-all: test-auth test-conversations test-response test-media test-mcp-integration test-e2e
	@echo ""
	@echo " All integration tests passed!"

test-auth:
	@echo "Running authentication tests..."
	@$(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=jan-client" \
		--verbose \
		--reporters cli
	@echo " Authentication tests passed"

test-conversations:
	@echo "Running conversation API tests..."
	@$(NEWMAN) run $(NEWMAN_CONVERSATION_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=jan-client" \
		--verbose \
		--reporters cli
	@echo " Conversation API tests passed"

test-response:
	@echo "Running response API tests..."
	@$(NEWMAN) run $(NEWMAN_RESPONSES_COLLECTION) \
		--env-var "response_api_url=http://localhost:8000/responses" \
		--env-var "mcp_tools_url=http://localhost:8000/mcp" \
		--verbose \
		--reporters cli
	@echo " Response API tests passed"

test-media:
	@echo "Running media API tests..."
	@$(NEWMAN) run $(NEWMAN_MEDIA_COLLECTION) \
		--env-var "media_api_url=http://localhost:8000/media" \
		--env-var "media_service_key=$(MEDIA_SERVICE_KEY)" \
		--verbose \
		--reporters cli
	@echo " Media API tests passed"

test-mcp-integration:
	@echo "Running MCP integration tests..."
	@$(NEWMAN) run $(NEWMAN_MCP_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "mcp_tools_url=http://localhost:8000/mcp" \
		--env-var "searxng_url=http://localhost:8086" \
		--verbose \
		--reporters cli
	@echo " MCP integration tests passed"

test-e2e:
	@echo "Running gateway end-to-end tests..."
	@$(NEWMAN) run $(NEWMAN_E2E_COLLECTION) \
		--env-var "gateway_url=http://localhost:8000" \
		--env-var "media_api_url=http://localhost:8000/media" \
		--env-var "response_api_url=http://localhost:8000/responses" \
		--env-var "mcp_tools_url=http://localhost:8000/mcp" \
		--env-var "media_service_key=$(MEDIA_SERVICE_KEY)" \
		--verbose \
		--reporters cli
	@echo "�o. Gateway end-to-end tests passed"

newman-debug:
	@echo "Running authentication tests with debug output..."
ifeq ($(OS),Windows_NT)
	@set NODE_DEBUG=request && $(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=jan-client" \
		--verbose \
		--reporter-cli-no-banner \
		--reporter-cli-no-summary \
		--reporter-cli-show-timestamps
else
	@NODE_DEBUG=request $(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=jan-client" \
		--verbose \
		--reporter-cli-no-banner \
		--reporter-cli-no-summary \
		--reporter-cli-show-timestamps
endif


# ============================================================================================================
# SECTION 8: DEVELOPER UTILITIES
# ============================================================================================================

# --- Development Full Stack (with host.docker.internal support) ---

.PHONY: dev-full dev-full-stop dev-full-down

dev-full:
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
	@echo "     .\\scripts\\dev-full-run.ps1 llm-api"
else
	@echo "     ./scripts/dev-full-run.sh llm-api"
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
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8080/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  LLM API:    healthy' } catch { Write-Host '  LLM API:    unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8285/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Media API:  healthy' } catch { try { if ($$PSItem.Exception.Response.StatusCode.Value__ -eq 401) { Write-Host '  Media API:  healthy' } else { Write-Host '  Media API:  unhealthy' } } catch { Write-Host '  Media API:  unhealthy' } }"
	@echo.
	@echo [MCP Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8091/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  MCP Tools:      healthy' } catch { Write-Host '  MCP Tools:      unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8086 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  SearXNG:        healthy' } catch { Write-Host '  SearXNG:        unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3015/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  Vector Store:   healthy' } catch { Write-Host '  Vector Store:   unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3010 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  SandboxFusion:  healthy' } catch { Write-Host '  SandboxFusion:  unhealthy' }"
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
	@curl -sf http://localhost:8080/healthz >/dev/null && echo "  LLM API:    healthy" || echo "  LLM API:    unhealthy"
	@curl -s http://localhost:8285/healthz >/dev/null && echo "  Media API:  healthy" || (curl -s -w "%{http_code}" -o /dev/null http://localhost:8285/healthz | grep -q "401" && echo "  Media API:  healthy" || echo "  Media API:  unhealthy")
	@echo ""
	@echo "[MCP Services]"
	@curl -sf http://localhost:8091/healthz >/dev/null && echo "  MCP Tools:      healthy" || echo "  MCP Tools:      unhealthy"
	@curl -sf http://localhost:8086 >/dev/null && echo "  SearXNG:        healthy" || echo "  SearXNG:        unhealthy"
	@curl -sf http://localhost:3015/healthz >/dev/null && echo "  Vector Store:   healthy" || echo "  V ector Store:   unhealthy"
	@curl -sf http://localhost:3010 >/dev/null && echo "  SandboxFusion:  healthy" || echo "  SandboxFusion:  unhealthy"
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


