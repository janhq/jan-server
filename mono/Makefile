.PHONY: all setup build run dev test clean docker-up docker-down help cli-build cli-install test-all test-auth test-conversation test-model

# Variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# CLI Tool
ifeq ($(OS),Windows_NT)
API_TEST := tools/jan-cli/jan-cli.exe api-test run
else
API_TEST := tools/jan-cli/jan-cli api-test run
endif

# Test Configuration
GATEWAY_URL ?= http://localhost:8080
TIMEOUT_MS ?= 30000
COLLECTIONS_DIR := tests/e2e/automation/collections
AUTH_MODE ?= guest

# API test base flags
API_TEST_FLAGS := --env-file tests/e2e/.env \
  --env-var gateway_url=$(GATEWAY_URL) \
  --auto-models \
  --auto-auth $(AUTH_MODE) \
  --timeout-request $(TIMEOUT_MS) \
  --debug

# ============================================
# Quick Start
# ============================================

# Interactive setup and run (like original jan-server)
quickstart: cli-build
	@echo "Starting interactive setup and run..."
ifeq ($(OS),Windows_NT)
	@tools/jan-cli/jan-cli.exe setup-and-run
else
	@tools/jan-cli/jan-cli setup-and-run
endif

# Non-interactive setup and run
quickstart-auto: setup docker-up
	@echo "Jan Server is starting..."
	@echo "Backend API: http://localhost:8080"
	@echo "Web UI: http://localhost:3001"
	@echo "MinIO Console: http://localhost:9001"

# Setup environment file
setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env from template..."; \
		cp .env.template .env; \
		echo "Please review and update .env with your settings"; \
	else \
		echo ".env already exists"; \
	fi

# ============================================
# Development
# ============================================

# Run backend in development mode
dev-backend:
	cd apps/backend && make dev

# Run web in development mode
dev-web:
	cd apps/web && npm run dev

# Run all in development mode (requires multiple terminals or tmux)
dev:
	@echo "Starting development servers..."
	@echo "Run 'make dev-backend' in one terminal"
	@echo "Run 'make dev-web' in another terminal"

# ============================================
# Build
# ============================================

# Build backend
build-backend:
	cd apps/backend && make build

# Build web
build-web:
	cd apps/web && npm run build

# Build all
build: build-backend build-web

# ============================================
# Docker
# ============================================

# Start all services with Docker Compose
docker-up:
	docker compose up -d

# Start with Keycloak
docker-up-keycloak:
	docker compose --profile keycloak up -d

# Start with Redis
docker-up-redis:
	docker compose --profile redis up -d

# Start all optional services
docker-up-full:
	docker compose --profile keycloak --profile redis up -d

# Stop all services
docker-down:
	docker compose down

# Stop and remove volumes
docker-down-clean:
	docker compose down -v

# View logs
docker-logs:
	docker compose logs -f

# View backend logs
docker-logs-backend:
	docker compose logs -f backend

# Rebuild and restart
docker-rebuild:
	docker compose build --no-cache
	docker compose up -d

# ============================================
# Testing
# ============================================

# Run backend tests
test-backend:
	cd apps/backend && make test

# Run all tests
test: test-backend

# Run tests with coverage
test-coverage:
	cd apps/backend && make test-coverage

# ============================================
# CLI Tool
# ============================================

# Build jan-cli
cli-build:
	@echo "Building jan-cli..."
	@cd tools/jan-cli && go build -o jan-cli$(if $(filter Windows_NT,$(OS)),.exe,) .
	@echo "jan-cli built successfully"

# Install jan-cli dependencies
cli-deps:
	@echo "Installing jan-cli dependencies..."
	@cd tools/jan-cli && go mod tidy
	@echo "Dependencies installed"

# ============================================
# Integration Tests (API Tests)
# ============================================

# Run all integration tests
test-all: cli-build
	@echo "Running all API integration tests..."
	$(API_TEST) $(wildcard $(COLLECTIONS_DIR)/*.postman.json) $(API_TEST_FLAGS) --timeout-request 120000

# Run auth tests
test-auth: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/auth.postman.json $(API_TEST_FLAGS)

# Run conversation tests
test-conversation: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/conversation.postman.json $(API_TEST_FLAGS)

# Run model tests
test-model: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/model.postman.json $(API_TEST_FLAGS)

# Run media tests
test-media: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/media.postman.json $(API_TEST_FLAGS)

# Run messages tests
test-messages: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/messages.postman.json $(API_TEST_FLAGS)

# Run image tests
test-image: cli-build
	$(API_TEST) $(COLLECTIONS_DIR)/image.postman.json $(API_TEST_FLAGS) --timeout-request 120000

# Run a quick test to verify setup
test-quick: cli-build
	@echo "Running quick test to verify API is working..."
	$(API_TEST) $(COLLECTIONS_DIR)/auth.postman.json $(API_TEST_FLAGS)

# ============================================
# Database
# ============================================

# Open database console
db-console:
	docker compose exec postgres psql -U jan -d jan

# Run migrations
db-migrate:
	cd apps/backend && make migrate-up

# Rollback migration
db-rollback:
	cd apps/backend && make migrate-down

# Reset database
db-reset:
	docker compose down postgres
	docker volume rm mono_postgres_data || true
	docker compose up -d postgres
	@echo "Waiting for postgres..."
	@sleep 5
	cd apps/backend && make migrate-up

# ============================================
# Code Quality
# ============================================

# Format code
fmt:
	cd apps/backend && make fmt

# Lint code
lint:
	cd apps/backend && make lint

# Security scan
security:
	cd apps/backend && make security

# Generate swagger docs
swagger:
	cd apps/backend && make swagger

# ============================================
# Health & Status
# ============================================

# Check service health
health-check:
	@echo "Checking backend health..."
	@curl -sf http://localhost:8080/healthz && echo "Backend: OK" || echo "Backend: FAILED"
	@echo "Checking database readiness..."
	@curl -sf http://localhost:8080/readyz && echo "Database: OK" || echo "Database: FAILED"

# Show running containers
status:
	docker compose ps

# ============================================
# Cleanup
# ============================================

# Clean build artifacts
clean:
	cd apps/backend && make clean
	rm -rf apps/web/dist
	rm -rf apps/web/node_modules

# Clean everything including Docker
clean-all: clean docker-down-clean

# ============================================
# Help
# ============================================

help:
	@echo "Jan Server Mono - Available Commands"
	@echo ""
	@echo "Quick Start:"
	@echo "  make quickstart      - Interactive setup and run (recommended)"
	@echo "  make quickstart-auto - Non-interactive setup and start"
	@echo "  make setup           - Create .env from template"
	@echo ""
	@echo "Development:"
	@echo "  make dev-backend     - Run backend with hot reload"
	@echo "  make dev-web         - Run web frontend with hot reload"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up       - Start all services"
	@echo "  make docker-up-full  - Start with Keycloak and Redis"
	@echo "  make docker-down     - Stop all services"
	@echo "  make docker-logs     - View logs"
	@echo "  make docker-rebuild  - Rebuild and restart"
	@echo ""
	@echo "Database:"
	@echo "  make db-console      - Open PostgreSQL shell"
	@echo "  make db-migrate      - Run migrations"
	@echo "  make db-reset        - Reset database"
	@echo ""
	@echo "Testing:"
	@echo "  make test            - Run unit tests"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo "  make test-all        - Run all API integration tests"
	@echo "  make test-auth       - Run authentication tests"
	@echo "  make test-conversation - Run conversation tests"
	@echo "  make test-model      - Run model tests"
	@echo "  make test-quick      - Quick test to verify setup"
	@echo ""
	@echo "CLI Tool:"
	@echo "  make cli-build       - Build jan-cli tool"
	@echo "  make cli-deps        - Install CLI dependencies"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo "  make swagger         - Generate API docs"
	@echo ""
	@echo "Status:"
	@echo "  make health-check    - Check service health"
	@echo "  make status          - Show running containers"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make clean-all       - Clean everything including Docker"
