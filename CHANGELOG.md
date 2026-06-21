# Changelog - Jan Server

> **Note:** This changelog tracks all releases and changes across Jan Server's microservices architecture, organized chronologically by version.

---

## Unreleased

### Removed

- **Realtime API** (LiveKit-based real-time communication) — service and all references removed.
- **Memory Tools** service — removed.
- **Platform web app** (Next.js admin/docs site) — removed; the React + Vite web app (`apps/web`) is the only frontend.
- **slide_creator agent** and slide generation — removed.

---

## December 2025

### [v0.0.14] - Multi-vLLM Provider, MCP Tools & User Experience Enhancements

**Status:** ✅ Complete | **Date:** December 2025

Major feature release with realtime communication via LiveKit, enhanced MCP capabilities, improved automation testing, multiple vLLM instance support, and refined user experience features.

#### What's New

- **Multi-vLLM Instance Provider**:
  - Round-robin load balancing across multiple vLLM instances
  - Enhanced swagger documentation
  - Improved provider instance management
- **MCP Tools Enhancements**:
  - Admin API integration for MCP management and monitoring
  - Model prompt administration
  - URL filtering and validation
  - Performance improvements for search operations (reduced latency)
  - Improved error handling for invalid format handling
  - Tool call tracking and verification
- **User Settings & Personalization**:
  - User-specific settings storage and management
  - Default settings for existing users
  - Settings validation and improvement
  - Browser user support
- **Conversation & Content Management**:
  - Image uploading and processing
  - Image content parsing improvements
  - Conversation title generation improvements
  - Model context size validation
  - Message sharing features
  - Deep research capability integration
  - Project-level conversation management
  - Conversation deletion (including bulk operations)
  - Model tool support flag
  - Better conversation naming and titling strategies
- **File Search & Content Management**:
  - File search disabled by default for better performance
  - Content filtering and optimization
  - File upload and handling improvements
- **Instruct Model Setup**:
  - Dedicated configuration for instruct-based models
  - Enhanced model initialization
- **Monitoring & Metrics**:
  - Enhanced monitoring metrics tracking
  - Improved observability for system performance
  - Latest prompt orchestration metrics
- **Automation & Testing**:
  - Refactored automation test suite
  - Improved test execution and logging
  - Cleaner test output and reporting
- **Bug Fixes & Improvements**:
  - Context size reload handling improvements
  - Token size calculation fixes
  - Conversation recreation with same name fixes
  - Invalid format parsing fixes
  - Browser compatibility improvements
  - Settings configuration improvements

#### Developer Experience

- Improved CI/CD pipelines for all services
- Enhanced swagger API documentation
- Better code organization and standardization
- Cleaner logging and debugging output
- Improved test infrastructure and reporting

---

### [v0.0.13] - Admin API & Model Catalog Enhancements

**Status:** ✅ Complete | **Date:** December 2025

Major expansion of admin capabilities with comprehensive management APIs, enhanced model catalog features, user personalization system, and robust middleware stack.

#### What's New

- **Admin API Endpoints**: Full operations for user management, group management, and feature flag administration
- **Model Catalog Enhancements**:
  - Feature flag integration for controlled model access
  - Experimental model support with explicit flagging
  - Advanced capability filtering (images, embeddings, reasoning, audio, video)
  - Model ordering and categorization improvements
- **User Personalization & Project Instructions**:
  - Custom project instructions per workspace/conversation
  - User-specific preferences and settings storage
  - Context-aware instruction injection for enhanced AI responses
  - Project-level configuration management
- **Database Schema Updates**:
  - Feature flags table with user/group assignments
  - Audit logging for admin operations
  - Model display names and ordering fields
  - Enhanced provider model metadata
  - Project instructions and user preferences tables
- **Middleware Additions**:
  - Admin authorization middleware for protected endpoints
  - Token bucket rate limiting for API protection
  - Feature flag checking middleware for gated features
  - Request validation and sanitization

#### Developer Experience

- Platform web admin UI for model catalog management
- Comprehensive API documentation for admin endpoints
- Improved error handling and validation messages

---

## November 2025

### [v0.0.12] - Microservices Architecture Release

**Status:** ✅ Complete | **Date:** November 2025

Complete architectural overhaul from Kubernetes-native monolithic platform to microservices-first architecture with Docker Compose, Kong Gateway, and comprehensive observability.

#### What's New

- **4 Core Microservices**: LLM API (8080), Media API (8285), Response API (8082), MCP Tools (8091)
- **Kong 3.5 Gateway**: Centralized routing, auth, and rate limiting
- **PostgreSQL 18**: Upgraded from v16, simplified single-instance architecture
- **100+ Makefile Commands**: Setup, testing, monitoring, deployment (400% increase)
- **6 Test Suites**: Auth, conversations, responses, media, MCP, E2E with jan-cli
- **Observability Stack**: OpenTelemetry, Prometheus (9090), Grafana (3331), Jaeger (16686)
- **Service Template**: Automated microservice generator with `new-service-from-template.ps1`

#### Developer Experience

- One-command setup: `make quickstart`
- Profile-based deployment: `make up-full`, `make up-gpu`, `make monitor-up`
- Hybrid development mode: Native execution with hot reload
- 20+ documentation pages covering guides, API reference, architecture

#### Infrastructure

- Docker Compose profiles (infra, api, mcp, full, monitoring)
- Kubernetes/Helm charts updated for all services
- S3-compatible encrypted storage
- Keycloak OIDC authentication + guest login
- Environment configs: defaults.env, development.env, testing.env, secrets.env.example

#### Tools & Integrations

- MCP Protocol integration (mark3labs/mcp-go)
- Google Search (Serper API), web scraping (SearXNG)
- Code execution sandbox (SandboxFusion)
- Vector store integration for semantic search

#### Breaking Changes

- Gateway port changed: `8080` → `8000`
- Swagger UI moved: `/api/swagger/index.html` → `/v1/swagger/`
- Health endpoint: `/healthcheck` → `/healthz`
- Project structure: `apps/` → `services/`
- Removed PostgreSQL replicas (now single instance)
- Removed Python inference service (replaced with vLLM)

---

## Oct 2025

### [v0.0.11] - Initial Microservices Restructuring

**Status:** ✅ Complete | **Date:** Oct 2025

Foundation for microservices transition with Makefile consolidation, MCP provider integration, and initial observability stack.

#### What's New

- **Makefile Consolidation**: Merged modular Makefiles into single structured file (10 sections)
- **Hybrid Development Mode**: Native binary execution with hot reload for faster iteration
- **MCP Provider Integration**: Basic Model Context Protocol support with JSON-RPC endpoint
- **Observability Foundation**: Prometheus, Jaeger, Grafana, OpenTelemetry collector
- **Guest Authentication**: Keycloak token exchange for quick access without full registration
- **PostgreSQL 16**: Upgraded from earlier versions
- **Kong Gateway Migration**: Replaced custom gateway with Kong 3.5
- **Service Restructuring**: Moved from `apps/` to `services/` directory structure

#### API & Infrastructure

- OpenAI-compatible endpoint: `POST /v1/chat/completions`
- Initial MCP tools endpoint: `POST /v1/mcp`
- Docker Compose organization with profiles
- Basic test collections for core functionality

---

## Initial Release

### [v0.0.10] - Foundation

**Status:** ✅ Complete | **Date:** Initial Release

First release of Jan Server with LLM API, authentication, conversation management, and Docker deployment.

#### Core Features

- **LLM API Service**: OpenAI-compatible chat completions, conversation management, message history
- **Authentication System**: User registration, API key generation, token-based auth
- **Conversation Management**: Create, manage, and persist conversations with message history
- **Docker Deployment**: Basic Docker Compose configuration for local development
- **PostgreSQL Backend**: Database schema for conversations, messages, users, and authentication

---

## Version Comparison

| Metric              | v0.0.10 | v0.0.11  | v0.0.12          |
| ------------------- | ------- | -------- | ---------------- |
| Services            | 1       | 2        | 4+               |
| Deployment Methods  | Docker  | Docker   | Docker + K8s     |
| Make Commands       | ~10     | ~20      | 100+             |
| Test Suites         | None    | Basic    | 6 collections    |
| Documentation Pages | ~3      | ~5       | 20+              |
| Monitoring Tools    | None    | 2        | 4 (full stack)   |
| Auth Methods        | Basic   | Keycloak | Keycloak + Guest |
| Gateway             | None    | Kong     | Kong 3.5         |

---
