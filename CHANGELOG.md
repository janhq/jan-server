# Changelog

All notable changes to Jan Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation restructure with clear sections
- API reference documentation structure
- Getting started guides
- Troubleshooting and deployment guides
- CONTRIBUTING.md for contribution guidelines

### Changed
- Reorganized documentation into logical categories (getting-started, guides, api, architecture)
- Moved monitoring configs to dedicated `monitoring/` directory
- Split large architecture.md into focused documents
- Streamlined README.md to < 100 lines

### Removed
- Redundant OpenAPI file copies (source of truth in services/*/docs/swagger/)
- Manual HTTP test files (replaced by Postman/Newman tests)
- Internal process documentation (RESTRUCTURE_*, MAKEFILE_CONSOLIDATION, etc.)

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
