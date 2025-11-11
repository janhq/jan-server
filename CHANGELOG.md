# Changelog

All notable changes to Jan Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2025-11-10

### 🎯 Major Architectural Changes

This release represents a **complete architectural overhaul** from a Kubernetes-native monolithic platform to a microservices-first architecture with Docker Compose and enhanced developer experience.

### Added

#### 🏗️ New Microservices Architecture
- **Response API Service** (Port 8082) - Multi-step tool orchestration with configurable execution depth and timeout
- **Media API Service** (Port 8285) - S3-integrated media ingestion and resolution with `jan_*` ID system
- **MCP Tools Service** (Port 8091) - Model Context Protocol integration for external tools
- **Service Template System** - Reusable Go microservice skeleton with standardized structure
  - `scripts/new-service-from-template.ps1` - Automated service generation script
  - Complete template with config, logging, tracing, HTTP server, Makefile, and Dockerfile

#### 🛠️ Developer Experience
- **100+ Makefile commands** organized into 10 sections:
  - Environment management (setup, clean, health checks)
  - Infrastructure management (Docker Compose profiles)
  - Service management (build, run, logs)
  - Database operations (migrations, reset)
  - Testing suite (auth, conversations, media, responses, MCP, E2E)
  - Hybrid development mode (native execution with hot reload)
  - Monitoring stack (Prometheus, Grafana, Jaeger)
  - Build automation and utilities
- **Hybrid Development Mode** - Run services natively for faster iteration with `make hybrid-dev`
- **Quick Start** - One-command setup: `make setup && make up-full`
- **Health Check Utilities** - `make health-check` for service monitoring

#### 🧪 Comprehensive Testing Infrastructure
- **6 Newman/Postman test collections** in `tests/automation/`:
  - `auth-postman-scripts.json` - Authentication tests
  - `conversations-postman-scripts.json` - Conversation API tests
  - `responses-postman-scripts.json` - Response API tests
  - `media-postman-scripts.json` - Media API tests
  - `mcp-postman-scripts.json` - MCP tools tests
  - `test-all.postman.json` - Complete E2E test suite
- **Test commands**:
  - `make test-all` - Run all test suites
  - `make test-auth` - Authentication tests
  - `make test-conversations` - Conversation tests
  - `make test-response` - Response API tests
  - `make test-media` - Media API tests
  - `make test-mcp` - MCP tools tests
  - `make test-e2e` - Gateway E2E tests

#### 📊 Enhanced Monitoring & Observability
- **Complete observability stack** with Docker Compose profiles:
  - **Grafana** dashboards (http://localhost:3001, admin/admin)
  - **Prometheus** metrics collection (http://localhost:9090)
  - **Jaeger** distributed tracing (http://localhost:16686)
  - **OpenTelemetry Collector** for telemetry aggregation
- **Service-specific log viewing** - `make logs-llm-api`, `make logs-mcp`, etc.
- **Profile-based monitoring** - `make monitor-up` to start monitoring stack

#### ⚙️ Configuration Management
- **Multiple environment configurations**:
  - `config/defaults.env` - Base configuration for all environments
  - `config/development.env` - Docker internal DNS configuration
  - `config/testing.env` - localhost URLs for Newman tests
  - `config/hybrid.env` - Native development configuration
  - `config/secrets.env.example` - Secrets template
- **Profile-based deployment**:
  - `make up-full` - Full stack with all services
  - `make up-gpu` - With GPU inference support
  - `make up-cpu` - CPU-only inference
  - `make monitor-up` - With monitoring stack

#### 🔐 Authentication Enhancements
- **Guest authentication** - Quick access via `/llm/auth/guest-login` endpoint
- **Keycloak OIDC integration** - Full OAuth/OIDC support
- **Simplified token management** - Streamlined authentication flow

#### 📚 Documentation Overhaul
- **Comprehensive documentation structure**:
  - `docs/getting-started/README.md` - Setup guides and first steps
  - `docs/guides/` - In-depth guides:
    - `development.md` - Complete development workflow (updated with all services)
    - `testing.md` - Testing procedures and test suites
    - `deployment.md` - Production deployment guide
    - `hybrid-mode.md` - Native development setup
    - `monitoring.md` - Observability configuration
    - `mcp-testing.md` - MCP tools testing guide
    - `services-template.md` - Service template usage
  - `docs/api/` - API reference:
    - `llm-api/` - LLM API documentation
    - `mcp-tools/` - MCP tools documentation
  - `docs/architecture/` - System design documents
  - `docs/conventions/` - Code standards and patterns
  - `docs/QUICK_REFERENCE.md` - Command reference (100+ commands)
  - `config/README.md` - Configuration guide
- **Kubernetes documentation**:
  - `k8s/README.md` - K8s deployment overview (updated for all services)
  - `k8s/SETUP.md` - Step-by-step setup guide (updated for response-api and media-api)
  - Complete Helm chart for all microservices

#### 🚢 Kubernetes/Helm Enhancements
- **Response API Kubernetes templates**:
  - `k8s/jan-server/templates/response-api-deployment.yaml`
  - `k8s/jan-server/templates/response-api-secret.yaml`
  - `k8s/jan-server/templates/response-api-ingress.yaml`
- **Updated Helm chart** (version 1.1.0):
  - Added response-api configuration in all values files
  - Fixed media-api configuration (added `apiKey` field)
  - Updated Kong gateway routing for all services
  - Enhanced values for development and production environments
- **Kong API Gateway routing**:
  - `/api/llm/*` → llm-api:8080
  - `/api/media/*` → media-api:8285
  - `/api/responses/*` → response-api:8082
  - `/api/mcp/*` → mcp-tools:8091

#### 🎨 MCP Tools Integration
- **Google Search** - `google_search` tool integration
- **Web Scraping** - Web content extraction tools
- **MCP Protocol Support** - Full Model Context Protocol implementation
- **Serper API Integration** - Web search capabilities
- **MCP endpoint** - `/v1/mcp` for tool interactions

### Changed

#### 🏛️ Architecture Transformation
- **Deployment strategy**: Kubernetes-only → **Docker Compose-first** with Kubernetes support
- **API Gateway**: Custom Jan API Gateway → **Kong 3.5**
- **Authentication**: Google OAuth2 only → **Keycloak (OIDC)** with guest auth
- **Service structure**: Monolithic (2 services) → **Microservices (4+ services)**
- **Database**: PostgreSQL with read/write replicas → **PostgreSQL 18** (simplified single instance)
- **Inference**: Jan Inference Model (Python) → **vLLM**
- **MCP Framework**: Not specified → **mark3labs/mcp-go**

#### 📦 Service Organization
- **Restructured** from `apps/` to `services/` directory:
  - `services/llm-api/` - Core LLM orchestration (Go)
  - `services/mcp-tools/` - MCP tools integration (Go)
  - `services/media-api/` - Media management (Go)
  - `services/response-api/` - Response orchestration (Go)
  - `services/template-api/` - Service template (Go)
- **Separated concerns** into specialized microservices
- **All services in Go** (removed Python inference service)

#### 🔧 Technology Stack Updates
- **Go version**: 1.24.6 → **Go 1.21+**
- **PostgreSQL**: With replicas → **PostgreSQL 18** (single instance)
- **API Gateway**: Custom → **Kong 3.5**
- **Web Framework**: Gin (remains)
- **Monitoring**: Grafana Pyroscope → **OpenTelemetry + Prometheus + Jaeger + Grafana**

#### 📝 API Endpoints
- **Gateway URL**: `http://localhost:8080` → **`http://localhost:8000`** (Kong)
- **Swagger UI**: `/api/swagger/index.html` → **`/v1/swagger/`**
- **Health endpoint**: `/healthcheck` → **`/healthz`** (on each service)
- **New endpoints**:
  - `/v1/chat/completions` - OpenAI-compatible chat endpoint
  - `/v1/mcp` - MCP tools endpoint
- `/llm/auth/guest-login` - Guest authentication
  - `/api/media/*` - Media API routes
  - `/api/responses/*` - Response API routes

#### 📖 README.md Optimization
- **Reduced from 345 lines to 235 lines** (-32%)
- **Focus on quick start** - `make setup && make up-full`
- **Better organization** with clear sections
- **Enhanced examples** for API usage
- **Improved documentation links**

#### 🎯 Developer Workflow
- **Build commands**: Docker build → **`make build-llm-api`**, etc.
- **Run commands**: `./scripts/run.sh` → **`make up-full`**
- **Test commands**: None → **`make test-all`** and specific test suites
- **Development mode**: Kubernetes only → **`make hybrid-dev`** for native execution
- **Log viewing**: `kubectl logs` → **`make logs-llm-api`**

### Removed

#### 🗑️ Deprecated Features
- **Multi-tenant organization management** - Removed organization/project-level access control
- **PostgreSQL read/write replicas** - Simplified to single instance
- **Google OAuth2 direct integration** - Now handled through Keycloak
- **Python inference service** - Replaced with vLLM
- **Database migration tools (Atlas)** - Changed migration approach
- **Complex API key scoping** - Simplified authentication model
- **pprof endpoints** (port 6060) - Replaced with comprehensive monitoring stack

#### 📁 Cleaned Up
- Legacy `apps/` directory structure
- Old Jan API Gateway monolithic service
- Custom authentication implementation
- Kubernetes-only deployment scripts

### Fixed

#### 🐛 Bug Fixes
- **Media API configuration** - Added missing `apiKey` field alongside `serviceKey`
- **Response API port** - Corrected port from 8280 to 8082 throughout documentation
- **Kong gateway routing** - Updated to properly route all four API services
- **Kubernetes templates** - Fixed media-api deployment to include MEDIA_API_KEY environment variable

### Security

#### 🔒 Security Enhancements
- **Keycloak OIDC** - Industry-standard authentication
- **Service-level authentication** - Each service can be independently secured
- **API key management** - Secure key handling for media and MCP services
- **Environment variable security** - Proper secrets management with `.env` files

### Migration Guide

#### 🔄 Breaking Changes
This is a **major version change** with breaking changes. Organizations using v2.0.0 need to:

1. **Update deployment infrastructure** - Migrate from Kubernetes-only to Docker Compose or new Helm charts
2. **Update authentication integration** - Migrate from Google OAuth2 to Keycloak
3. **Update API client code** - New gateway URL and routing paths
4. **Update service architecture** - Adapt to microservices structure
5. **Update database schema** - Apply new migrations for multiple services
6. **Update monitoring integration** - Configure new observability stack

#### 📊 Statistics
| Metric | v2.0.0 | v0.2.0 | Change |
|--------|--------|--------|--------|
| **Services** | 2 | 4+ | +100% |
| **Deployment Methods** | 1 (K8s) | 2 (Docker + K8s) | +100% |
| **Make Commands** | ~20 | 100+ | +400% |
| **Test Suites** | Basic | 6 collections | New |
| **Documentation Pages** | ~5 | 20+ | +300% |
| **Monitoring Tools** | 2 | 4 | +100% |
| **Auth Methods** | 1 | 2 | +100% |

### Performance

#### ⚡ Improvements
- **Faster iteration** - Hybrid mode allows native execution with hot reload
- **Better resource utilization** - Microservices can be scaled independently
- **Improved developer experience** - One-command setup reduces onboarding time
- **Enhanced observability** - Better troubleshooting with distributed tracing

### Dependencies

#### 📦 Updated Dependencies
- **Kong**: 3.5
- **Keycloak**: Latest with OIDC
- **PostgreSQL**: 18
- **Go**: 1.21+
- **mark3labs/mcp-go**: Latest
- **OpenTelemetry**: Latest
- **Prometheus**: Latest
- **Jaeger**: Latest
- **Grafana**: Latest

## [2.0.0] - 2025-01-07

### Added
- Consolidated Makefile structure (single file with 10 sections)
- Hybrid development mode for faster iteration
- MCP (Model Context Protocol) provider integration
- Full observability stack (Prometheus, Jaeger, Grafana)
- OpenTelemetry integration
- Guest authentication with Keycloak token exchange
- Comprehensive testing suite with Newman
- Documentation for all major features

### Changed
- Restructured project from monolithic to microservices architecture
- Updated to PostgreSQL 16
- Migrated to Kong 3.5 API Gateway
- Improved Docker Compose organization with profiles

### Removed
- Modular Makefile files (consolidated into single Makefile)
- Legacy authentication system

## [1.0.0] - Initial Release

### Added
- Initial LLM API service with OpenAI-compatible endpoints
- Basic authentication
- Conversation and message management
- Docker Compose deployment
- PostgreSQL database backend

---

[Unreleased]: https://github.com/janhq/jan-server/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/janhq/jan-server/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/janhq/jan-server/releases/tag/v1.0.0
