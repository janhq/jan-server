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
#   make up-full        - Start all services (infrastructure + API + MCP)
#   make health-check   - Check if all services are healthy
#   make test-all       - Run all integration tests
#   make stop           - Stop all services (keeps containers & volumes)
#   make down           - Stop and remove containers (keeps volumes)
#   make down-clean     - Stop, remove containers and volumes (full cleanup)
#
# ============================================================================================================
# COMMON COMMANDS
# ============================================================================================================
#
# Environment Management:
#   make env-create                  - Create .env from template
#   make env-switch ENV=development  - Switch environment (development/testing/hybrid)
#   make env-validate                - Validate .env file
#   make env-info                    - Show current environment info
#   make env-secrets                 - Check required secrets
#
# Service Management:
#   make up-infra                    - Start infrastructure (postgres, keycloak, kong)
#   make up-api                      - Start LLM API service
#   make up-mcp                      - Start MCP services
#   make up-full                     - Start all services (infra + api + mcp + vllm-gpu)
#   make up-vllm-gpu                 - Start vLLM GPU inference only
#   make up-vllm-cpu                 - Start vLLM CPU inference only
#   make stop                        - Stop all services (keeps containers & volumes)
#   make down                        - Stop and remove containers (keeps volumes)
#   make down-clean                  - Stop, remove containers and volumes (full cleanup)
#
# Development (Hybrid Mode):
#   make hybrid-dev-api              - Setup for API development (native + Docker)
#   make hybrid-run-api              - Run API natively with hot reload
#   make hybrid-dev-mcp              - Setup for MCP development
#   make hybrid-run-mcp              - Run MCP natively with hot reload
#
# Testing:
#   make test                        - Run unit tests
#   make test-all                    - Run all integration tests
#   make test-auth                   - Run authentication tests
#   make test-conversations          - Run conversation API tests
#   make test-mcp-integration        - Run MCP integration tests
#
# Build & Code Quality:
#   make build-all                   - Build all services
#   make swagger                     - Generate API documentation
#   make fmt                         - Format Go code
#   make lint                        - Run linters
#
# Utilities:
#   make dev-status                  - Show status of all services
#   make logs-api                    - View API logs
#   make logs-mcp                    - View MCP logs
#   make db-console                  - Open database console
#   make curl-health                 - Test health endpoints
#
# ============================================================================================================
# FILE ORGANIZATION (Single Makefile Structure)
# ============================================================================================================
#
# This Makefile is organized into the following sections:
#
#   1. SETUP & ENVIRONMENT       - Initial setup, dependency checks, environment management
#   2. DOCKER INFRASTRUCTURE     - Network and volume management
#   3. BUILD TARGETS             - Building services, code quality, Swagger
#   4. SERVICE MANAGEMENT        - Starting/stopping services (infra, API, MCP)
#   5. DATABASE MANAGEMENT       - DB operations, migrations, backups
#   6. MONITORING                - Observability stack (Prometheus, Grafana, Jaeger)
#   7. TESTING                   - Unit tests, integration tests, CI/CD
#   8. HYBRID DEVELOPMENT        - Native development mode with Docker infrastructure
#   9. DEVELOPER UTILITIES       - Dev tools, debugging, performance testing
#   10. HEALTH CHECKS            - Service health validation
#
# ============================================================================================================
# DOCUMENTATION
# ============================================================================================================
#
#   📖 docs/DEVELOPMENT.md      - Complete development guide
#   📖 docs/TESTING.md          - Testing procedures and best practices
#   📖 docs/HYBRID_MODE.md      - Hybrid development workflow
#   📖 docs/MIGRATION.md        - Migration from old structure
#   📖 README.md                - Project overview and quick reference
#
# ============================================================================================================
# VARIABLES
# ============================================================================================================

COMPOSE = docker compose
MONITOR_COMPOSE = docker compose -f docker/observability.yml
NEWMAN = newman
NEWMAN_AUTH_COLLECTION = tests/automation/auth-postman-scripts.json
NEWMAN_CONVERSATION_COLLECTION = tests/automation/conversations-postman-scripts.json
NEWMAN_MCP_COLLECTION = tests/automation/mcp-postman-scripts.json

# ============================================================================================================
# BACKWARD COMPATIBILITY ALIASES
# ============================================================================================================

.PHONY: up up-llm-api up-mcp-tools

up: up-infra
up-llm-api: up-api
up-mcp-tools: up-mcp

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
	@docker --version >nul 2>&1 || echo "Docker not found"
	@docker compose version >nul 2>&1 || echo "Docker Compose V2 not found"
	@go version >nul 2>&1 || echo "Go not found (optional)"
	@newman --version >nul 2>&1 || echo "Newman not found (optional)"
	@echo "Dependency check complete"

install-deps:
	@echo "Installing development dependencies..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ Development dependencies installed"

# --- Environment Management ---

.PHONY: env-create env-list env-switch env-validate

env-create:
	@echo "Creating .env file..."
ifeq ($(OS),Windows_NT)
	@if exist .env (echo .env already exists) else (copy .env.template .env && echo .env created)
else
	@if [ -f .env ]; then echo ".env already exists"; else cp .env.template .env && echo ".env created"; fi
endif

env-list:
	@echo "Available environment configurations:"
	@echo "  - development  (config/development.env) - All services in Docker"
	@echo "  - testing      (config/testing.env)     - Integration testing"
	@echo "  - hybrid       (config/hybrid.env)      - Native services + Docker infra"
	@echo "  - production   (config/production.env.example) - Production template"
	@echo ""
	@echo "Usage: make env-switch ENV=<environment>"
	@echo "Docs:  See config/README.md for detailed guide"

env-switch:
ifndef ENV
	@echo "ENV variable required. Usage: make env-switch ENV=development"
	@exit 1
endif
ifeq ($(OS),Windows_NT)
	@if exist config\$(ENV).env (copy config\$(ENV).env .env && echo Switched to $(ENV)) else (echo config\$(ENV).env not found)
else
	@if [ -f "config/$(ENV).env" ]; then cp config/$(ENV).env .env && echo "Switched to $(ENV)"; else echo "config/$(ENV).env not found"; fi
endif

env-validate:
ifeq ($(OS),Windows_NT)
	@if exist .env (echo .env file exists) else (echo .env file not found. Run make env-create)
else
	@if [ -f .env ]; then echo ".env file exists"; else echo ".env file not found. Run make env-create"; fi
endif

env-info:
	@echo "=== Current Environment Info ==="
ifeq ($(OS),Windows_NT)
	@if exist .env (echo Environment file: .env exists && echo --- && echo Recent variables: && type .env | findstr /V "^#" | findstr /V "^$$" | more +1) else (echo No .env file found)
else
	@if [ -f .env ]; then echo "Environment file: .env exists" && echo "---" && echo "Variables set: $$(grep -c '^[A-Z]' .env || echo 0)"; else echo "No .env file found"; fi
endif
	@echo ""
	@echo "Switch environment: make env-switch ENV=<development|hybrid|testing>"
	@echo "Documentation: config/README.md"

env-secrets:
	@echo "=== Required Secrets Checklist ==="
	@echo "See config/secrets.env.example for details"
	@echo ""
ifeq ($(OS),Windows_NT)
	@if exist .env (echo Checking .env for required secrets... && \
		(findstr /C:"HF_TOKEN=" .env >nul && echo [OK] HF_TOKEN is set || echo [MISSING] HF_TOKEN - Get from https://huggingface.co/settings/tokens) && \
		(findstr /C:"SERPER_API_KEY=" .env >nul && echo [OK] SERPER_API_KEY is set || echo [MISSING] SERPER_API_KEY - Get from https://serper.dev)) else (echo .env not found. Run make env-create)
else
	@if [ -f .env ]; then \
		echo "Checking .env for required secrets..." && \
		(grep -q "^HF_TOKEN=" .env && echo "[OK] HF_TOKEN is set" || echo "[MISSING] HF_TOKEN - Get from https://huggingface.co/settings/tokens") && \
		(grep -q "^SERPER_API_KEY=" .env && echo "[OK] SERPER_API_KEY is set" || echo "[MISSING] SERPER_API_KEY - Get from https://serper.dev"); \
	else echo ".env not found. Run make env-create"; fi
endif
	@echo ""
	@echo "Full secrets list: cat config/secrets.env.example"

# ============================================================================================================
# SECTION 2: DOCKER INFRASTRUCTURE
# ============================================================================================================

# --- Network Management ---

.PHONY: network-create network-list network-clean

network-create:
	@docker network inspect jan-server_default >/dev/null 2>&1 || \
		docker network create jan-server_default
	@docker network inspect jan-server_mcp-network >/dev/null 2>&1 || \
		docker network create jan-server_mcp-network
	@echo "✅ Docker networks created"

network-list:
	@docker network ls | grep jan-server

network-clean:
	@docker network rm jan-server_default jan-server_mcp-network 2>/dev/null || true
	@echo "✅ Docker networks removed"

# --- Volume Management ---

.PHONY: volumes-list volumes-clean

volumes-list:
	@docker volume ls | grep jan-server

volumes-clean:
	@echo "⚠️  WARNING: This will delete all data!"
	@echo -n "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]
	@docker volume ls -q | grep jan-server | xargs -r docker volume rm
	@echo "✅ Volumes removed"

# ============================================================================================================
# SECTION 3: BUILD TARGETS
# ============================================================================================================

.PHONY: build build-api build-mcp build-all clean-build

build: build-api build-mcp

build-api:
	@echo "Building LLM API..."
	@cd services/llm-api && go build -o bin/llm-api .
	@echo "✅ LLM API built: services/llm-api/bin/llm-api"

build-mcp:
	@echo "Building MCP Tools..."
	@cd services/mcp-tools && go build -o bin/mcp-tools .
	@echo "✅ MCP Tools built: services/mcp-tools/bin/mcp-tools"

build-all:
	@echo "Building all Docker images..."
	$(COMPOSE) --profile full build
	@echo "✅ All services built"

clean-build:
	@echo "Cleaning build artifacts..."
	@rm -rf services/llm-api/bin
	@rm -rf services/mcp-tools/bin
	@echo "✅ Build artifacts cleaned"

# --- Swagger Documentation ---

.PHONY: swagger swagger-llm-api swagger-mcp-tools swagger-combine swagger-install

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
	@cd services/llm-api && swag init \
		--dir ./cmd/server,./internal/interfaces/httpserver/routes \
		--generalInfo server.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo "✅ llm-api swagger generated at services/llm-api/docs/swagger"

swagger-mcp-tools:
	@echo "Generating Swagger for mcp-tools service..."
	@cd services/mcp-tools && swag init \
		--dir . \
		--generalInfo main.go \
		--output ./docs/swagger \
		--parseDependency \
		--parseInternal
	@echo "✅ mcp-tools swagger generated at services/mcp-tools/docs/swagger"

swagger-combine:
	@echo "Merging LLM API and MCP Tools swagger specs..."
	@go run scripts/swagger-combine.go \
		-llm-api services/llm-api/docs/swagger/swagger.json \
		-mcp-tools services/mcp-tools/docs/swagger/swagger.json \
		-output services/llm-api/docs/swagger/swagger-combined.json
	@echo "✅ Combined swagger created for unified API documentation"

swagger-install:
	@echo "Installing swagger tools..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ swag installed successfully"

# --- Code Quality ---

.PHONY: fmt lint vet

fmt:
	@echo "Formatting Go code..."
	@gofmt -w $$(go list -f '{{.Dir}}' ./...)
	@echo "✅ Code formatted"

lint:
	@echo "Running linter..."
	@go vet ./...
	@echo "✅ Linting complete"

vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✅ Vet complete"

# ============================================================================================================
# SECTION 4: SERVICE MANAGEMENT
# ============================================================================================================

# --- Infrastructure Services ---

.PHONY: up-infra down-infra restart-infra logs-infra

up-infra:
	@echo "Starting infrastructure services..."
	$(COMPOSE) --profile infra up -d
	@echo "✅ Infrastructure services started"
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

.PHONY: up-api down-api restart-api logs-api

up-api:
	@echo "Starting LLM API..."
	$(COMPOSE) --profile api up -d
	@echo "✅ LLM API started at http://localhost:8080"

down-api:
	$(COMPOSE) --profile api down

restart-api:
	$(COMPOSE) --profile api restart

logs-api:
	$(COMPOSE) --profile api logs -f llm-api

# --- MCP Services ---

.PHONY: up-mcp down-mcp restart-mcp logs-mcp

up-mcp:
	@echo "Starting MCP services..."
	$(COMPOSE) --profile mcp up -d
	@echo "✅ MCP services started"
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
	@echo "✅ vLLM GPU started at http://localhost:8001"
	@echo ""
	@echo "Test inference:"
	@echo "  curl http://localhost:8001/v1/models"

up-vllm-cpu:
	@echo "Starting vLLM CPU inference..."
	$(COMPOSE) --profile cpu up -d
	@echo "✅ vLLM CPU started at http://localhost:8001"
	@echo ""
	@echo "Test inference:"
	@echo "  curl http://localhost:8001/v1/models"

down-vllm:
	@echo "Stopping vLLM services..."
	$(COMPOSE) --profile gpu --profile cpu down

logs-vllm:
	$(COMPOSE) --profile gpu --profile cpu logs -f

# --- Full Stack ---

.PHONY: up-full down-full restart-full logs logs-follow stop down down-clean stop-info

stop-info:
	@echo ==========================================
	@echo Stop/Down Commands Comparison
	@echo ==========================================
	@echo.
	@echo make stop
	@echo   - Stops all services
	@echo   - Keeps containers (fast restart)
	@echo   - Keeps volumes
	@echo   - Keeps networks
	@echo   - Use: Quick pause during development
	@echo.
	@echo make down
	@echo   - Stops all services
	@echo   - Removes containers
	@echo   - Keeps volumes (data preserved)
	@echo   - Removes networks
	@echo   - Use: Clean shutdown, data preserved
	@echo.
	@echo make down-clean
	@echo   - Stops all services
	@echo   - Removes containers
	@echo   - Removes volumes (data deleted)
	@echo   - Removes networks
	@echo   - Use: Complete cleanup, fresh start
	@echo.
	@echo ==========================================

up-full:
	@echo "Starting full stack..."
	$(COMPOSE) --profile full up -d
	@echo "✅ Full stack started"
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
	@echo "  - vLLM GPU:       http://localhost:8001"

down-full:
	$(COMPOSE) --profile full down

restart-full:
	$(COMPOSE) --profile full restart

stop:
	@echo "Stopping all services (containers will be preserved)..."
	$(COMPOSE) --profile full stop
	@echo "✅ All services stopped (containers preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To remove containers: make down"

down:
	@echo "Stopping and removing all containers (volumes will be preserved)..."
	$(COMPOSE) --profile full down
	@echo "✅ All containers stopped and removed (volumes preserved)"
	@echo ""
	@echo "To restart: make up-full"
	@echo "To clean volumes: make down-clean"

down-clean:
	@echo "Stopping and removing all containers and volumes..."
	$(COMPOSE) --profile full down -v
	@echo "✅ All containers and volumes removed (full cleanup)"
	@echo ""
	@echo "To restart: make up-full"

logs:
	$(COMPOSE) logs -f

logs-follow:
	$(COMPOSE) logs -f $(SERVICE)

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
	@echo "✅ Kong restarted"

restart-keycloak:
	$(COMPOSE) restart keycloak

restart-postgres:
	$(COMPOSE) restart api-db

# ============================================================================================================
# SECTION 5: DATABASE MANAGEMENT
# ============================================================================================================

.PHONY: db-reset db-migrate db-console db-backup db-restore db-dump

db-reset:
	@echo "⚠️  WARNING: This will delete all database data!"
	@echo "Stopping and removing API database..."
	$(COMPOSE) stop api-db
	$(COMPOSE) rm -f api-db
	@docker volume rm jan-server_api-db-data || true
	@echo "✅ Database reset complete. Run 'make up-api' to restart."

db-migrate:
	@echo "Running database migrations..."
	$(COMPOSE) exec llm-api /app/llm-api migrate
	@echo "✅ Migrations complete"

db-console:
	@echo "Opening database console..."
	$(COMPOSE) exec api-db psql -U jan_user -d jan_llm_api

db-backup:
	@echo "Backing up database..."
	@mkdir -p backups
	@docker compose exec -T api-db pg_dump -U jan_user jan_llm_api > backups/db_backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✅ Database backed up to backups/"

db-restore:
	@if [ -z "$(FILE)" ]; then \
		echo "❌ FILE variable required. Usage: make db-restore FILE=backups/db_backup.sql"; \
		exit 1; \
	fi
	@echo "Restoring database from $(FILE)..."
	@cat $(FILE) | docker compose exec -T api-db psql -U jan_user -d jan_llm_api
	@echo "✅ Database restored"

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
	@echo "✅ Monitoring stack started"
	@echo ""
	@echo "Dashboards:"
	@echo "  - Grafana:    http://localhost:3001 (admin/admin)"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Jaeger:     http://localhost:16686"

monitor-down:
	@echo "Stopping monitoring stack..."
	$(MONITOR_COMPOSE) down
	@echo "✅ Monitoring stack stopped"

monitor-logs:
	$(MONITOR_COMPOSE) logs -f

monitor-clean:
	@echo "Stopping monitoring stack and removing volumes..."
	$(MONITOR_COMPOSE) down -v
	@echo "✅ Monitoring stack cleaned"

# ============================================================================================================
# SECTION 7: TESTING
# ============================================================================================================

# --- Unit Tests ---

.PHONY: test test-api test-mcp test-coverage

test:
	@echo "Running unit tests..."
	@go test ./...
	@echo "✅ Unit tests passed"

test-api:
	@echo "Running LLM API tests..."
	@cd services/llm-api && go test ./...
	@echo "✅ LLM API tests passed"

test-mcp:
	@echo "Running MCP Tools tests..."
	@cd services/mcp-tools && go test ./...
	@echo "✅ MCP Tools tests passed"

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# --- Integration Tests (Newman) ---

.PHONY: test-all test-auth test-conversations test-mcp-integration newman-debug

test-all: test-auth test-conversations test-mcp-integration
	@echo ""
	@echo "✅ All integration tests passed!"

test-auth:
	@echo "Running authentication tests..."
	@$(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8080" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--reporters cli
	@echo "✅ Authentication tests passed"

test-conversations:
	@echo "Running conversation API tests..."
	@$(NEWMAN) run $(NEWMAN_CONVERSATION_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8080" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--reporters cli
	@echo "✅ Conversation API tests passed"

test-mcp-integration:
	@echo "Running MCP integration tests..."
	@$(NEWMAN) run $(NEWMAN_MCP_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8080" \
		--env-var "mcp_tools_url=http://localhost:8091" \
		--env-var "searxng_url=http://localhost:8086" \
		--verbose \
		--reporters cli
	@echo "✅ MCP integration tests passed"

newman-debug:
	@echo "Running authentication tests with debug output..."
	@NODE_DEBUG=request $(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8080" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--verbose \
		--reporter-cli-no-banner \
		--reporter-cli-no-summary \
		--reporter-cli-show-timestamps

# --- Test Environment Management ---

.PHONY: test-setup test-teardown test-clean

test-setup:
	@echo "Setting up test environment..."
	@$(MAKE) env-switch ENV=testing
	@$(MAKE) up-full
	@echo "Waiting for services to be ready..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "Start-Sleep -Seconds 10"
else
	@sleep 10
endif
	@$(MAKE) health-check
	@echo "✅ Test environment ready"

test-teardown:
	@echo "Tearing down test environment..."
	@$(MAKE) down
	@echo "✅ Test environment stopped"

test-clean: test-teardown
	@rm -f newman.json coverage.out coverage.html
	@echo "✅ Test artifacts cleaned"

# --- CI/CD Helpers ---

.PHONY: ci-test ci-lint ci-build

ci-test: test test-all
	@echo "✅ All CI tests passed"

ci-lint: lint vet
	@echo "✅ CI linting passed"

ci-build: build-all
	@echo "✅ CI build complete"

# ============================================================================================================
# SECTION 8: HYBRID DEVELOPMENT
# ============================================================================================================

# --- Hybrid Infrastructure ---

.PHONY: hybrid-infra-up hybrid-infra-down hybrid-mcp-up hybrid-mcp-down

hybrid-infra-up:
	@echo "Starting infrastructure for hybrid mode..."
	docker compose -f docker-compose.yml -f docker/dev-hybrid.yml --profile hybrid up -d
	@echo "✅ Infrastructure ready for hybrid development"
	@echo ""
	@echo "Infrastructure services running in Docker:"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - Keycloak:   http://localhost:8085"
	@echo ""
	@echo "You can now run services natively:"
	@echo "  - API:  ./scripts/hybrid-run-api.sh (or .ps1 on Windows)"
	@echo "  - MCP:  ./scripts/hybrid-run-mcp.sh (or .ps1 on Windows)"

hybrid-infra-down:
	docker compose -f docker-compose.yml -f docker/dev-hybrid.yml --profile hybrid down

hybrid-mcp-up:
	@echo "Starting MCP infrastructure for hybrid mode..."
	docker compose -f docker-compose.yml -f docker/dev-hybrid.yml --profile hybrid-mcp up -d
	@echo "✅ MCP infrastructure ready"
	@echo ""
	@echo "MCP services running in Docker:"
	@echo "  - SearXNG:        http://localhost:8086"
	@echo "  - Vector Store:   http://localhost:3015"
	@echo "  - SandboxFusion:  http://localhost:3010"
	@echo ""
	@echo "Run MCP Tools natively: ./scripts/hybrid-run-mcp.sh"

hybrid-mcp-down:
	docker compose -f docker-compose.yml -f docker/dev-hybrid.yml --profile hybrid-mcp down

# --- Run Services Natively ---

.PHONY: hybrid-run-api hybrid-run-mcp hybrid-env-api hybrid-env-mcp

hybrid-run-api:
	@echo "Running LLM API natively..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/hybrid-run-api.ps1
else
	@bash scripts/hybrid-run-api.sh
endif

hybrid-run-mcp:
	@echo "Running MCP Tools natively..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/hybrid-run-mcp.ps1
else
	@bash scripts/hybrid-run-mcp.sh
endif

hybrid-env-api:
	@echo "Environment variables for native LLM API development:"
	@echo ""
	@echo "export DATABASE_URL='postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable'"
	@echo "export KEYCLOAK_BASE_URL='http://localhost:8085'"
	@echo "export JWKS_URL='http://localhost:8085/realms/jan/protocol/openid-connect/certs'"
	@echo "export ISSUER='http://localhost:8090/realms/jan'"
	@echo "export HTTP_PORT='8080'"
	@echo "export LOG_LEVEL='debug'"
	@echo "export LOG_FORMAT='console'"
	@echo "export AUTO_MIGRATE='true'"
	@echo ""
	@echo "Or load from config: source config/hybrid.env"

hybrid-env-mcp:
	@echo "Environment variables for native MCP Tools development:"
	@echo ""
	@echo "export HTTP_PORT='8091'"
	@echo "export VECTOR_STORE_URL='http://localhost:3015'"
	@echo "export SEARXNG_URL='http://localhost:8086'"
	@echo "export SANDBOXFUSION_URL='http://localhost:3010'"
	@echo "export LOG_LEVEL='debug'"
	@echo "export LOG_FORMAT='console'"
	@echo ""
	@echo "Or load from config: source config/hybrid.env"

# --- Complete Workflows ---

.PHONY: hybrid-dev-api hybrid-dev-mcp hybrid-dev-full hybrid-stop

hybrid-dev-api:
	@echo "Setting up hybrid API development environment..."
	@$(MAKE) hybrid-infra-up
	@echo ""
	@echo "✅ Ready for API development!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Start API: make hybrid-run-api"
	@echo "  2. Or manually: cd services/llm-api && source ../../config/hybrid.env && go run ."

hybrid-dev-mcp:
	@echo "Setting up hybrid MCP development environment..."
	@$(MAKE) hybrid-mcp-up
	@echo ""
	@echo "✅ Ready for MCP development!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Start MCP: make hybrid-run-mcp"
	@echo "  2. Or manually: cd services/mcp-tools && source ../../config/hybrid.env && go run ."

hybrid-dev-full:
	@echo "Setting up full hybrid development environment..."
	@$(MAKE) hybrid-infra-up
	@$(MAKE) hybrid-mcp-up
	@echo ""
	@echo "✅ Ready for full hybrid development!"
	@echo ""
	@echo "Run services:"
	@echo "  - API:  make hybrid-run-api"
	@echo "  - MCP:  make hybrid-run-mcp"

hybrid-stop:
	@echo "Stopping hybrid infrastructure..."
	@$(MAKE) hybrid-infra-down
	@$(MAKE) hybrid-mcp-down
	@echo "✅ Hybrid infrastructure stopped"

# --- Debugging ---

.PHONY: hybrid-debug-api hybrid-debug-mcp

hybrid-debug-api:
	@echo "Starting API with delve debugger..."
	@echo "Connect your IDE debugger to localhost:2345"
	@cd services/llm-api && \
		source ../../config/hybrid.env && \
		dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

hybrid-debug-mcp:
	@echo "Starting MCP Tools with delve debugger..."
	@echo "Connect your IDE debugger to localhost:2346"
	@cd services/mcp-tools && \
		source ../../config/hybrid.env && \
		dlv debug --headless --listen=:2346 --api-version=2 --accept-multiclient

# ============================================================================================================
# SECTION 9: DEVELOPER UTILITIES
# ============================================================================================================

# --- Quick Development Commands ---

.PHONY: dev-reset dev-clean dev-status

dev-reset:
	@echo "⚠️  Resetting development environment..."
	@$(MAKE) down
	@$(MAKE) volumes-clean
	@$(MAKE) network-clean
	@$(MAKE) clean-build
	@echo ""
	@$(MAKE) setup
	@$(MAKE) network-create
	@echo "✅ Development environment reset complete"

dev-clean:
	@echo "Cleaning development artifacts..."
	@$(MAKE) clean-build
	@rm -f newman.json coverage.out coverage.html
	@rm -rf services/llm-api/docs/swagger
	@rm -rf services/mcp-tools/docs/swagger
	@find . -name "*.log" -type f -delete
	@echo "✅ Development artifacts cleaned"

dev-status:
	@echo "=== Docker Services ==="
	@$(COMPOSE) ps
	@echo ""
	@echo "=== Docker Networks ==="
	@$(MAKE) network-list
	@echo ""
	@echo "=== Docker Volumes ==="
	@$(MAKE) volumes-list
	@echo ""
	@echo "=== Service Health ==="
	@$(MAKE) health-check

# --- API Testing Utilities ---

.PHONY: curl-health curl-chat curl-mcp

curl-health:
	@echo "Testing LLM API health..."
	@curl -s http://localhost:8080/healthz | jq
	@echo ""
	@echo "Testing MCP Tools health..."
	@curl -s http://localhost:8091/healthz | jq

curl-chat:
	@if [ -z "$$TOKEN" ]; then \
		echo "❌ TOKEN environment variable required"; \
		echo "Usage: TOKEN=your_token make curl-chat"; \
		exit 1; \
	fi
	@curl -s -H "Authorization: Bearer $$TOKEN" \
		-H "Content-Type: application/json" \
		-d '{"model":"jan-v1-4b","messages":[{"role":"user","content":"Hello"}]}' \
		http://localhost:8080/v1/chat/completions | jq

curl-mcp:
	@echo "Listing available MCP tools..."
	@curl -s -X POST http://localhost:8091/v1/mcp \
		-H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | jq

# --- Docker Utilities ---

.PHONY: docker-ps docker-images docker-prune docker-stats

docker-ps:
	@docker ps --filter "name=jan-server" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

docker-images:
	@docker images | grep -E "(jan|mcp|kong|keycloak|postgres)" || echo "No images found"

docker-prune:
	@echo "Cleaning up Docker system..."
	@docker system prune -f
	@echo "✅ Docker system cleaned"

docker-stats:
	@docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# --- Log Utilities ---

.PHONY: logs-api-tail logs-mcp-tail logs-error

logs-api-tail:
	@$(COMPOSE) logs --tail=100 llm-api

logs-mcp-tail:
	@$(COMPOSE) logs --tail=100 mcp-tools

logs-error:
	@$(COMPOSE) logs | grep -i error || echo "No errors found"

# --- Performance Testing ---

.PHONY: perf-test perf-load

perf-test:
	@echo "Running performance test..."
	@echo "Requires 'ab' (Apache Bench). Install: apt-get install apache2-utils"
	@ab -n 100 -c 10 http://localhost:8080/healthz

perf-load:
	@echo "Running load test with hey..."
	@hey -n 1000 -c 50 -m GET http://localhost:8080/healthz

# --- Code Generation ---

.PHONY: generate generate-mocks

generate:
	@echo "Running go generate..."
	@go generate ./...
	@echo "✅ Code generation complete"

generate-mocks:
	@echo "Generating mocks..."
	@echo "Install mockgen: go install go.uber.org/mock/mockgen@latest"
	@cd services/llm-api && go generate ./...
	@cd services/mcp-tools && go generate ./...
	@echo "✅ Mocks generated"

# --- Documentation ---

.PHONY: docs-serve docs-build

docs-serve:
	@echo "Serving documentation at http://localhost:6060"
	@echo "Install godoc: go install golang.org/x/tools/cmd/godoc@latest"
	@godoc -http=:6060

docs-build:
	@echo "Building documentation..."
	@$(MAKE) swagger
	@echo "✅ Documentation built"

# --- Git Utilities ---

.PHONY: git-clean git-status

git-clean:
	@echo "⚠️  This will delete all git-ignored files"
	@git clean -fdX

git-status:
	@git status
	@echo ""
	@echo "=== Uncommitted changes ==="
	@git diff --stat

# ============================================================================================================
# SECTION 10: HEALTH CHECKS
# ============================================================================================================

.PHONY: health-check health-api health-mcp health-infra

health-check:
ifeq ($(OS),Windows_NT)
	@echo ============================================
	@echo Checking All Services Health Status
	@echo ============================================
	@echo.
	@echo [Infrastructure Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8085 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ Keycloak:   healthy' } catch { Write-Host '  ✗ Keycloak:   unhealthy' }"
	@powershell -Command "try { $$response = Invoke-WebRequest -Uri http://localhost:8000 -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue; if ($$response.StatusCode -ge 200 -and $$response.StatusCode -lt 500) { Write-Host '  ✓ Kong:       healthy' } else { Write-Host '  ✗ Kong:       unhealthy' } } catch { if ($$_.Exception.Response.StatusCode.Value__ -eq 404) { Write-Host '  ✓ Kong:       healthy' } else { Write-Host '  ✗ Kong:       unhealthy' } }"
	@powershell -Command "try { $$null = docker compose exec -T api-db pg_isready -U jan_user 2>&1 | Out-Null; if ($$LASTEXITCODE -eq 0) { Write-Host '  ✓ PostgreSQL: healthy' } else { Write-Host '  ✗ PostgreSQL: unhealthy' } } catch { Write-Host '  ✗ PostgreSQL: unhealthy' }"
	@echo.
	@echo [API Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8080/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ LLM API:    healthy' } catch { Write-Host '  ✗ LLM API:    unhealthy' }"
	@echo.
	@echo [MCP Services]
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8091/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ MCP Tools:      healthy' } catch { Write-Host '  ✗ MCP Tools:      unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8086 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ SearXNG:        healthy' } catch { Write-Host '  ✗ SearXNG:        unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3015/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ Vector Store:   healthy' } catch { Write-Host '  ✗ Vector Store:   unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3010 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host '  ✓ SandboxFusion:  healthy' } catch { Write-Host '  ✗ SandboxFusion:  unhealthy' }"
	@echo.
	@echo ============================================
else
	@echo "============================================"
	@echo "Checking All Services Health Status"
	@echo "============================================"
	@echo ""
	@echo "[Infrastructure Services]"
	@curl -sf http://localhost:8085 >/dev/null && echo "  ✓ Keycloak:   healthy" || echo "  ✗ Keycloak:   unhealthy"
	@curl -f --max-time 2 http://localhost:8000 >/dev/null 2>&1 || (curl --max-time 2 http://localhost:8000 2>&1 | grep -q "no Route matched" && echo "  ✓ Kong:       healthy" || echo "  ✗ Kong:       unhealthy")
	@docker compose exec -T api-db pg_isready -U jan_user >/dev/null 2>&1 && echo "  ✓ PostgreSQL: healthy" || echo "  ✗ PostgreSQL: unhealthy"
	@echo ""
	@echo "[API Services]"
	@curl -sf http://localhost:8080/healthz >/dev/null && echo "  ✓ LLM API:    healthy" || echo "  ✗ LLM API:    unhealthy"
	@echo ""
	@echo "[MCP Services]"
	@curl -sf http://localhost:8091/healthz >/dev/null && echo "  ✓ MCP Tools:      healthy" || echo "  ✗ MCP Tools:      unhealthy"
	@curl -sf http://localhost:8086 >/dev/null && echo "  ✓ SearXNG:        healthy" || echo "  ✗ SearXNG:        unhealthy"
	@curl -sf http://localhost:3015/healthz >/dev/null && echo "  ✓ Vector Store:   healthy" || echo "  ✗ Vector Store:   unhealthy"
	@curl -sf http://localhost:3010 >/dev/null && echo "  ✓ SandboxFusion:  healthy" || echo "  ✗ SandboxFusion:  unhealthy"
	@echo ""
	@echo "============================================"
endif

health-infra:
ifeq ($(OS),Windows_NT)
	@echo Checking infrastructure services...
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8085 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK Keycloak: healthy' } catch { Write-Host 'ERROR Keycloak: unhealthy' }"
	@powershell -Command "try { $$response = Invoke-WebRequest -Uri http://localhost:8000 -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue; if ($$response.StatusCode -ge 200 -and $$response.StatusCode -lt 500) { Write-Host 'OK Kong: healthy' } else { Write-Host 'ERROR Kong: unhealthy' } } catch { if ($$_.Exception.Response.StatusCode.Value__ -eq 404) { Write-Host 'OK Kong: healthy' } else { Write-Host 'ERROR Kong: unhealthy' } }"
	@powershell -Command "try { $$null = docker compose exec -T api-db pg_isready -U jan_user 2>&1 | Out-Null; if ($$LASTEXITCODE -eq 0) { Write-Host 'OK PostgreSQL: healthy' } else { Write-Host 'ERROR PostgreSQL: unhealthy' } } catch { Write-Host 'ERROR PostgreSQL: unhealthy' }"
else
	@curl -sf http://localhost:8085 >/dev/null && echo "✅ Keycloak: healthy" || echo "❌ Keycloak: unhealthy"
	@curl -f --max-time 2 http://localhost:8000 >/dev/null 2>&1 || (curl --max-time 2 http://localhost:8000 2>&1 | grep -q "no Route matched" && echo "✅ Kong: healthy" || echo "❌ Kong: unhealthy")
	@docker compose exec -T api-db pg_isready -U jan_user >/dev/null 2>&1 && echo "✅ PostgreSQL: healthy" || echo "❌ PostgreSQL: unhealthy"
endif

health-api:
ifeq ($(OS),Windows_NT)
	@powershell -Command "try { Invoke-WebRequest -Uri http://localhost:8080/healthz -UseBasicParsing | Select-Object -ExpandProperty Content | ConvertFrom-Json | ConvertTo-Json } catch { Write-Host 'ERROR API not responding' }"
else
	@curl -sf http://localhost:8080/healthz | jq || echo "❌ API not responding"
endif

health-mcp:
ifeq ($(OS),Windows_NT)
	@echo Checking MCP services...
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8091/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK MCP Tools: healthy' } catch { Write-Host 'ERROR MCP Tools: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:8086 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK SearXNG: healthy' } catch { Write-Host 'ERROR SearXNG: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3015/healthz -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK Vector Store: healthy' } catch { Write-Host 'ERROR Vector Store: unhealthy' }"
	@powershell -Command "try { $$null = Invoke-WebRequest -Uri http://localhost:3010 -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'OK SandboxFusion: healthy' } catch { Write-Host 'ERROR SandboxFusion: unhealthy' }"
else
	@curl -sf http://localhost:8091/healthz >/dev/null && echo "✅ MCP Tools: healthy" || echo "❌ MCP Tools: unhealthy"
	@curl -sf http://localhost:8086 >/dev/null && echo "✅ SearXNG: healthy" || echo "❌ SearXNG: unhealthy"
	@curl -sf http://localhost:3015/healthz >/dev/null && echo "✅ Vector Store: healthy" || echo "❌ Vector Store: unhealthy"
	@curl -sf http://localhost:3010 >/dev/null && echo "✅ SandboxFusion: healthy" || echo "❌ SandboxFusion: unhealthy"
endif

# ============================================================================================================
# END OF MAKEFILE
# ============================================================================================================
