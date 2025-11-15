# Sprint 3: Service Migration Strategy

## Overview

Sprint 3 involves migrating services (llm-api, mcp-tools, media-api, response-api) to use the centralized configuration system in `pkg/config`.

## Challenge: Module Dependencies

The jan-server project uses a **workspace** structure where each service is a separate Go module:
- `services/llm-api` has its own `go.mod`
- `services/mcp-tools` has its own `go.mod`
- `pkg/config` is at the root workspace level

**Problem:** Services cannot directly import `github.com/janhq/jan-server/pkg/config` without either:
1. Restructuring into a monorepo with shared pkg/
2. Publishing pkg/config as a separate module
3. Using Go workspace features to share the package

## Recommended Approach

### Phase 1: Environment Variable Alignment (Immediate)

**Goal:** Ensure all services use the same environment variable names as defined in `pkg/config/types.go`

**Tasks:**
1. Audit each service's env tags against pkg/config/types.go
2. Update service env tags to match centralized naming
3. Update Docker Compose files to use new env var names
4. Test each service independently

**Example:**
```go
// Before (llm-api):
HTTPPort int `env:"HTTP_PORT"`

// After (aligned with pkg/config/types.go):
HTTPPort int `env:"HTTP_PORT"`  // ✅ Already matches!

// Before (llm-api):
DatabaseURL string `env:"DATABASE_URL"`

// After (should use components):
// Build from POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, etc.
```

### Phase 2: Configuration Bridge Pattern (Sprint 3-4)

**Goal:** Create bridge functions that convert centralized config to service-specific config

**Implementation:**
```go
// In services/llm-api/internal/config/bridge.go

import centralconfig "github.com/janhq/jan-server/pkg/config"

// FromCentralConfig converts pkg/config.Config to llm-api Config
func FromCentralConfig(central *centralconfig.Config) *Config {
    return &Config{
        HTTPPort: central.Services.LLMAPI.HTTPPort,
        DatabaseURL: buildDatabaseURL(central.Infrastructure.Database.Postgres),
        // ... map all fields
    }
}
```

**Benefits:**
- Gradual migration (can still use env vars)
- Backward compatibility
- Clear mapping between old and new config

### Phase 3: Direct Integration (Sprint 5+)

**Goal:** Services directly use pkg/config types

**Requires:**
1. Go workspace configuration or monorepo restructuring
2. Update Wire providers to inject centralconfig.Config
3. Remove service-specific Config structs
4. Update all service code to use central types

## Current Status

### Sprint 2 Complete ✅
- pkg/config foundation built
- Precedence system (100-600) implemented
- All tests passing
- Documentation complete

### Sprint 3 Next Steps

**Option A: Environment Variable Alignment (Recommended for Sprint 3)**
- ✅ Low risk, immediate value
- ✅ No code restructuring needed
- ✅ Can be done per-service incrementally
- Tasks:
  1. Create env var mapping document
  2. Update docker-compose.yml environment sections
  3. Update service env tags
  4. Test each service

**Option B: Module Restructuring (Deferred to Sprint 4-5)**
- ⚠️ Requires Go workspace setup or monorepo migration
- ⚠️ Higher risk, more invasive
- ⚠️ Blocks other development during migration
- Tasks:
  1. Set up Go workspace in root
  2. Update all go.mod files
  3. Implement bridge pattern
  4. Migrate services one by one
  5. Comprehensive integration testing

## Decision: Sprint 3 Scope

**RECOMMENDATION:** Focus on Option A for Sprint 3

**Rationale:**
1. **Immediate Value:** Standardizing env vars provides immediate operational benefits
2. **Low Risk:** No code changes, only configuration alignment
3. **Foundation for Phase 2:** Makes bridge pattern easier in Sprint 4
4. **Testable:** Can validate each service independently

**Deliverables for Sprint 3:**
1. Environment variable mapping document
2. Updated docker-compose.yml with aligned env vars
3. Service-by-service env var audit
4. Integration tests validating env var precedence

## Implementation Plan

### Task 3.1: Environment Variable Audit

Create `docs/configuration/env-var-mapping.md` documenting:
- All env vars from pkg/config/types.go
- Current env vars in each service
- Mapping/migration needed
- Deprecation timeline

### Task 3.2: Docker Compose Updates

Update `docker-compose.yml` and `docker/` files to use:
- Standardized env var names
- config/environments/*.yaml for environment-specific overrides

### Task 3.3: Service Validation

For each service (llm-api, mcp-tools, media-api, response-api):
1. Update internal config env tags to match pkg/config
2. Run unit tests
3. Run integration tests
4. Verify in Docker environment

### Task 3.4: Documentation

1. Update service READMEs with new env var names
2. Create migration guide for operators
3. Document any breaking changes

## Sprint 4+ Preview

Once Sprint 3 (env var alignment) is complete, Sprint 4 can tackle:
- Go workspace setup
- Bridge pattern implementation
- Gradual service migration to use pkg/config directly

This two-phase approach minimizes risk while delivering incremental value.
