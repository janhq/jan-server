# Configuration Improvement Plan

**Generated:** November 15, 2025  
**Status:** Planning Phase  
**Target:** Simplify, standardize, and improve customizability of configuration management

---

## Executive Summary

After comprehensive review of the Jan Server configuration system across:
- Environment files (`.env.template`, `config/*.env.example`)
- Docker Compose files (`docker-compose.yml`, `docker/*.yml`)
- Code-level config (`services/*/internal/config/*.go`)
- Runtime configurations (`kong.yml`, `providers.yml`, `mcp-providers.yml`)
- Deployment configs (`k8s/`, `Makefile`)

**Key Finding:** The configuration system is functional but suffers from:
1. **Fragmentation** - 8+ places to configure the same service
2. **Duplication** - Same env vars defined in multiple files
3. **Complexity** - 200+ environment variables across the stack
4. **Inconsistency** - Different patterns per service (caarlos0/env vs manual parsing)
5. **Poor Discoverability** - Hard to find what config affects what

---

## Current Configuration Landscape

### 1. Environment Variables (200+ variables)
**Locations:**
- `.env.template` (205 lines) - Master template
- `config/production.env.example` (145 lines) - Production overrides
- `config/secrets.env.example` (161 lines) - Secret documentation
- `config/README.md` (361 lines) - Configuration guide

**Issues:**
- ❌ Duplicate definitions across files
- ❌ No validation until runtime
- ❌ Unclear which are required vs optional
- ❌ Poor organization (mixed concerns)
- ❌ Hard to track what changed between environments

### 2. Docker Compose Configuration (6 files)
**Files:**
- `docker-compose.yml` - Main orchestration (includes)
- `docker/infrastructure.yml` - Postgres, Keycloak, Kong
- `docker/services-api.yml` - LLM API, Media API, Response API
- `docker/services-mcp.yml` - MCP Tools, Vector Store
- `docker/inference.yml` - vLLM inference
- `docker/dev-full.yml` - Development overrides
- `docker/observability.yml` - Monitoring stack

**Issues:**
- ❌ Environment variables embedded in YAML (`${VAR:-default}`)
- ❌ 150+ env var references scattered across compose files
- ❌ Hard-coded defaults differ from `.env.template`
- ❌ No way to validate compose env before `up`
- ❌ `env_file:` used inconsistently

### 3. Service-Level Configuration (Go structs)
**Services:**
- `llm-api/internal/config/config.go` (243 lines, 40+ env vars)
- `mcp-tools/infrastructure/config/config.go` (50 lines, 20+ env vars)
- `media-api/internal/config/config.go` (70+ env vars)
- `response-api/internal/config/config.go` (15+ env vars)
- `template-api/internal/config/config.go` (15+ env vars)

**Issues:**
- ❌ Each service uses `caarlos0/env` library independently
- ❌ No central validation or schema
- ❌ Config structs don't match .env organization
- ❌ Runtime errors only (no startup validation)
- ❌ Provider configs use separate YAML files

### 4. Runtime Configuration Files
**Files:**
- `kong/kong.yml` (378 lines) - Kong declarative config
- `kong/kong-dev-full.yml` - Kong dev overrides
- `services/llm-api/config/providers.yml` - Model provider setup
- `services/llm-api/config/providers_metadata_default.yml` - Provider defaults
- `services/mcp-tools/configs/mcp-providers.yml` - MCP provider config
- `keycloak/import/realm-jan.json` - Keycloak realm config

**Issues:**
- ❌ Mix of YAML and JSON configs
- ❌ Some use env var interpolation, some don't
- ❌ No schema validation
- ❌ Hard to override for different environments
- ❌ Provider configs require manual editing

### 5. Kubernetes Configuration
**Files:**
- `k8s/jan-server/values.yaml` (552 lines)
- `k8s/jan-server/values-development.yaml`
- `k8s/jan-server/values-production.yaml`

**Issues:**
- ❌ Duplicate config definitions from Docker
- ❌ No shared schema with docker-compose
- ❌ Manual sync required with env changes

### 6. Build & Task Configuration
**Files:**
- `Makefile` (728 lines)
- `.vscode/tasks.json` (via workspace)

**Issues:**
- ❌ Hard-coded env vars in tasks
- ❌ No dynamic env switching in tasks
- ❌ Makefile has embedded config logic

---

## Problems by Category

### A. Duplication & Consistency
**Problem:** Same configuration defined in 3-5 places
- Database URL: `.env`, `docker-compose`, Go config, K8s values
- Keycloak settings: 4 different files
- Port numbers: Spread across 7 files

**Impact:**
- Change one place, must update 4+ others
- Sync errors cause runtime failures
- Hard to know "source of truth"

### B. Validation & Type Safety
**Problem:** No pre-flight validation
- Invalid URLs only fail at runtime
- Missing required vars cause service crashes
- Type mismatches (string vs int) not caught

**Impact:**
- Long debug cycles
- Production deploy failures
- Hard to catch config errors in CI

### C. Environment Switching
**Problem:** Manual, error-prone process
- Must edit multiple files to switch dev → staging → prod
- `make env-switch` only copies one file
- Docker vs K8s configs diverge

**Impact:**
- Deployment mistakes
- Developer friction
- Testing doesn't match production

### D. Secrets Management
**Problem:** Secrets mixed with config
- `.env` contains both config and secrets
- No external secret injection (Vault, etc.)
- Secret rotation requires code changes

**Impact:**
- Security risks
- Hard to rotate credentials
- Can't use external secret managers easily

### E. Discoverability
**Problem:** Hard to understand configuration
- 200+ vars with no categorization
- No interactive config tool
- Documentation out of sync

**Impact:**
- Onboarding takes days
- Frequent misconfiguration
- Support burden

---

## Proposed Solution Architecture

### Phase 1: Configuration Schema & Validation ⭐ HIGH PRIORITY

**Goal:** Single source of truth with type safety

**Implementation:**
```
# Root-level configuration (infrastructure & cross-cutting)
config/
├── schema/
│   ├── config.schema.json          # JSON Schema for infrastructure
│   ├── services/
│   │   ├── llm-api.schema.json     # Schema for service env vars
│   │   ├── mcp-tools.schema.json
│   │   ├── media-api.schema.json
│   │   └── response-api.schema.json
│   └── environments/
│       ├── base.schema.json
│       ├── development.schema.json
│       ├── production.schema.json
│       └── kubernetes.schema.json
├── defaults.yaml                    # Infrastructure defaults only
├── environments/
│   ├── development.yaml             # Dev overrides only
│   ├── staging.yaml
│   ├── production.yaml
│   └── local-hybrid.yaml
├── secrets/
│   └── secrets.template.yaml        # Secret placeholders
└── validation/
    ├── validate.go                  # Go validator
    └── validate.sh                  # Pre-flight script

# Service-level pluggable configs (keep existing structure)
services/
├── llm-api/
│   └── config/
│       ├── providers.yml            # ⚠️ Managed by CI/CD
│       └── providers_metadata_default.yml
├── mcp-tools/
│   └── configs/
│       └── mcp-providers.yml        # ⚠️ Managed by CI/CD
├── media-api/
│   └── config/
│       └── storage-providers.yml    # ⚠️ Managed by CI/CD (future)
└── response-api/
    └── config/
        └── tool-config.yml          # ⚠️ Managed by CI/CD (future)
```

**Benefits:**
- ✅ Single schema validates all configs
- ✅ Type-safe (ports are ints, URLs validated)
- ✅ Auto-generate docs from schema
- ✅ IDE autocomplete support
- ✅ Catch errors before docker-compose up
- ✅ Service-specific configs stay in service dirs (CI/CD friendly)

**Design Principle:**
- **Root `/config`**: Infrastructure & environment settings (database, auth, ports, etc.)
- **Service `/config` or `/configs`**: Pluggable service configs (providers, plugins, tools) - overwritten by CI/CD

**Canonical Source of Truth: Go Structs**
```go
// pkg/config/types.go - Single source of truth
type Config struct {
    Database   DatabaseConfig   `yaml:"database" json:"database" jsonschema:"required"`
    Auth       AuthConfig       `yaml:"auth" json:"auth" jsonschema:"required"`
    Services   ServicesConfig   `yaml:"services" json:"services"`
    Monitoring MonitoringConfig `yaml:"monitoring" json:"monitoring"`
}

// Generate from this:
// 1. JSON Schema (via go-jsonschema or invopop/jsonschema)
// 2. Default YAML (via struct tags + defaults)
// 3. Documentation (via struct tags + comments)
// 4. .env template (for migration period)
```

**Generation Pipeline:**
```bash
# Single command generates all artifacts from Go structs
make config-generate
  ├── config/schema/*.schema.json     (from Go structs)
  ├── config/defaults.yaml            (from Go structs + defaults)
  ├── docs/configuration/*.md         (from Go structs + comments)
  └── .env.template                   (for reference only)
```

**Tasks:**
- [ ] Define canonical Go structs in `pkg/config/types.go` with full annotations
- [ ] Implement JSON Schema generator from Go structs (using invopop/jsonschema)
- [ ] Implement YAML defaults generator from Go structs
- [ ] Implement documentation generator from Go struct comments
- [ ] Add pre-commit hook to regenerate on struct changes
- [ ] Add CI test: verify generated files match source (prevent drift)
- [ ] Document CI/CD override pattern for service configs

---

### Phase 2: Unified Configuration Loader ⭐ HIGH PRIORITY

**Goal:** Load config with explicit precedence and conflict resolution

**Configuration Precedence (Highest to Lowest):**
```
1. CLI flags/commands        (jan-config set ...)
2. Environment variables      (export DATABASE_HOST=...)
3. Secret providers          (Vault, K8s secrets)
4. Environment file          (config/environments/{env}.yaml)
5. Defaults file             (config/defaults.yaml)
6. Struct default tags       (Go struct `envDefault`)
```

**Conflict Resolution Rules:**
- Higher precedence always wins (no merging for simple values)
- Complex objects (maps, arrays) merge by key unless override flag set
- Missing required values after all layers = validation error
- Secrets loaded lazily (not during initial parse)

**Implementation:**
```go
// pkg/config/loader.go
type ConfigLoader struct {
    schema      *Schema
    defaults    Config
    environment string
    precedence  []ConfigSource  // Ordered list of sources
}

type ConfigSource interface {
    Load(cfg *Config) error
    Priority() int
    Name() string
}

func NewLoader(env string, opts ...LoaderOption) (*ConfigLoader, error) {
    loader := &ConfigLoader{environment: env}
    
    // Build precedence stack
    loader.precedence = []ConfigSource{
        &StructDefaultSource{},      // Priority 100 (lowest)
        &YAMLDefaultSource{},        // Priority 200
        &YAMLEnvSource{env: env},    // Priority 300
        &SecretSource{},             // Priority 400
        &EnvVarSource{},             // Priority 500
        &CLISource{},                // Priority 600 (highest)
    }
    
    // Load in precedence order
    for _, source := range loader.precedence {
        if err := source.Load(&loader.config); err != nil {
            return nil, fmt.Errorf("load %s: %w", source.Name(), err)
        }
    }
    
    // Validate final merged config
    if err := loader.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    return loader, nil
}

// Debug method to show where each value came from
func (l *ConfigLoader) Provenance(path string) (ConfigSource, interface{}, error) {
    // Returns which source provided the final value
}
```

**Precedence Table Documentation:**
```yaml
# config/precedence.md - Auto-generated

| Config Path | Default | Environment | Secret | EnvVar | CLI | Final Value | Source |
|-------------|---------|-------------|--------|--------|-----|-------------|--------|
| database.host | api-db | localhost | - | prod-db | - | prod-db | EnvVar |
| database.port | 5432 | 5432 | - | - | 5433 | 5433 | CLI |
| auth.keycloak.password | admin | - | ******** | - | - | ******** | Secret |
```

**Benefits:**
- ✅ Explicit, documented precedence
- ✅ No hidden merge behavior
- ✅ Debug tool shows value provenance
- ✅ Predictable across all services
- ✅ Testable override behavior

**Two-Tier Loading Strategy:**
1. **Infrastructure Config**: Loaded from root `/config` (shared across services)
2. **Service Config**: Loaded from `services/{name}/config` (service-specific, CI/CD managed)

**Tasks:**
- [ ] Create shared config package: `pkg/config`
- [ ] Implement ConfigSource interface and precedence stack
- [ ] Implement each source: Struct defaults, YAML, Secrets, EnvVars, CLI
- [ ] Add conflict resolution with merge strategies
- [ ] Add Provenance() debug method
- [ ] Write precedence regression tests (100+ test cases)
- [ ] Generate precedence documentation table
- [ ] Keep existing service config loaders (providers.yml, mcp-providers.yml)
- [ ] Migrate llm-api to use unified loader for env vars
- [ ] Migrate mcp-tools to use unified loader for env vars
- [ ] Document service config file conventions

---

### Phase 3: Configuration Categories & Organization

**Goal:** Organize 200+ vars into logical groups

**Proposed Structure:**
```yaml
# config/defaults.yaml (Infrastructure & Environment Settings ONLY)
meta:
  version: "1.0"
  environment: development
  
# Infrastructure
infrastructure:
  database:
    postgres:
      user: jan_user
      database: jan_llm_api
      port: 5432
      max_connections: 100
      ssl_mode: disable
  
  auth:
    keycloak:
      realm: jan
      http_port: 8085
      admin_user: admin
      features: [token-exchange, preview]
  
  gateway:
    kong:
      http_port: 8000
      admin_port: 8001
      log_level: info

# Services (Environment Variables & Ports Only)
services:
  llm_api:
    http_port: 8080
    log_level: info
    auto_migrate: true
    # Provider config referenced, NOT defined here
    provider_config_file: services/llm-api/config/providers.yml  # CI/CD managed
    provider_config_set: default  # Which set to use from providers.yml
    api_keys:
      prefix: sk_live
      default_ttl: 2160h
      max_per_user: 5
  
  mcp_tools:
    http_port: 8091
    log_level: info
    search_engine: serper  # Which engine to use
    # MCP provider config referenced, NOT defined here
    mcp_config_file: services/mcp-tools/configs/mcp-providers.yml  # CI/CD managed
    sandbox_require_approval: true
  
  media_api:
    http_port: 8285
    log_level: info
    max_upload_bytes: 20971520
    retention_days: 30
    # Future: storage_config_file: services/media-api/config/storage-providers.yml

# Inference (vLLM environment settings, NOT model definitions)
inference:
  vllm:
    enabled: true
    port: 8001
    gpu_utilization: 0.95
    # Model list comes from llm-api's providers.yml (CI/CD managed)

# Monitoring
monitoring:
  otel:
    enabled: false
    endpoint: http://otel-collector:4318
  prometheus:
    port: 9090
  grafana:
    port: 3001
```

**Separate Service-Specific Configs (Unchanged, CI/CD Managed):**

```yaml
# services/llm-api/config/providers.yml
# ⚠️ This file is REPLACED by CI/CD deployments
providers:
  default:
    - name: Local vLLM
      type: vllm
      url: http://10.200.108.153:9000/v1
      api_key: ${VLLM_INTERNAL_KEY}
      auto_enable_new_models: true
      metadata:
        environment: internal
        image_input: '{"supported":true}'
  
  production:
    - name: External OpenAI
      type: openai
      url: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
```

```yaml
# services/mcp-tools/configs/mcp-providers.yml
# ⚠️ This file is REPLACED by CI/CD deployments
providers:
  - name: searxng
    description: Meta search engine
    enabled: true
    endpoint: http://searxng:8080
    type: http
  
  - name: vector-store
    enabled: true
    endpoint: ${VECTOR_STORE_URL}
    type: http

settings:
  max_timeout: 120s
  debug_logging: false
```

**Benefits:**
- ✅ Clear hierarchy and organization
- ✅ Easy to find related settings
- ✅ Comments and documentation inline
- ✅ Diff-friendly (YAML better than .env)
- ✅ **Service configs stay in service dirs** (CI/CD can replace them)
- ✅ Infrastructure vs service config separation

**Separation of Concerns:**
- **Root `/config`**: What infrastructure needs (ports, URLs, credentials, feature flags)
- **Service `/config`**: What plugins/providers to load (provider lists, tool configs, integrations)

**CI/CD Workflow:**
```bash
# CI/CD can replace service configs without touching root config
# Example: Deploy with different model providers
$ kubectl create configmap llm-api-providers \
    --from-file=services/llm-api/config/providers.yml \
    --dry-run=client -o yaml | kubectl apply -f -

# Example: Deploy with different MCP tools
$ docker cp custom-mcp-providers.yml mcp-tools:/app/configs/mcp-providers.yml
```

**Tasks:**
- [ ] Design category hierarchy (infrastructure only)
- [ ] Map all 200+ vars to categories (separate infra vs service)
- [ ] Create defaults.yaml with infrastructure structure
- [ ] Document each category purpose
- [ ] Document service config file conventions
- [ ] Create CI/CD examples for service config replacement
- [ ] Create migration guide from .env

---

### Phase 4: Environment-Specific Overrides

**Goal:** DRY principle - only define what changes

**Example:**
```yaml
# config/environments/development.yaml
infrastructure:
  database:
    postgres:
      host: api-db           # Docker internal DNS
      password: jan_password # Weak password OK for dev
  
  auth:
    keycloak:
      base_url: http://keycloak:8085
      admin_password: admin  # Weak password OK for dev

services:
  llm_api:
    log_level: debug
    log_format: console

# config/environments/production.yaml
infrastructure:
  database:
    postgres:
      host: prod-db.example.com
      ssl_mode: require
      # Password from secret manager
  
  auth:
    keycloak:
      base_url: https://auth.yourdomain.com
      # Admin password from secret manager

services:
  llm_api:
    log_level: info
    log_format: json
    auto_migrate: false

monitoring:
  otel:
    enabled: true
```

**Benefits:**
- ✅ See exactly what differs per environment
- ✅ Reduce config size by 80%
- ✅ Easier code review of config changes
- ✅ Merge conflicts reduced

**Tasks:**
- [ ] Create environment override files
- [ ] Implement merge logic (defaults + env)
- [ ] Add environment validation (prod requires secrets)
- [ ] Create environment diff tool
- [ ] Document override precedence

---

### Phase 5: Secret Management Integration

**STATUS: ❌ SKIPPED - Too complex for current needs. DevOps team handles secrets via K8s Secrets, Vault, and env files directly.**

**Goal:** Externalize secrets with clear developer workflow

**Implementation:**
```go
// pkg/config/secrets/provider.go
type SecretProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
    ListSecrets(ctx context.Context, prefix string) (map[string]string, error)
    Refresh(ctx context.Context) error  // For rotation support
}

// Implementations:
// - EnvVarProvider (development)
// - VaultProvider (production)
// - K8sSecretProvider (kubernetes)
// - AWSSecretsProvider (AWS)
// - AzureKeyVaultProvider (Azure)
// - CachedProvider (wraps any provider with local cache)
```

**Developer Workflow:**

**Local Development (Online):**
```bash
# One-time setup: authenticate to Vault
export VAULT_ADDR=https://vault.dev.example.com
vault login -method=oidc

# Pull secrets to local cache
jan-config secrets pull --env=development
# Creates: ~/.jan/secrets/development.enc (encrypted with local key)

# Run services (reads from cache)
make up-full
# Fallback: If cache missing, prompts for Vault auth
```

**Local Development (Offline):**
```bash
# Use cached secrets (valid for 24h by default)
jan-config secrets status
# Output: ✓ development: 47 secrets cached, expires in 18h

# Or use .env file (less secure, for quick testing)
cp .env.example .env
# Edit .env with dummy secrets
jan-config validate --secrets-source=env
```

**CI/CD Workflow:**
```bash
# CI jobs authenticate via service account
export VAULT_ROLE_ID=$CI_VAULT_ROLE_ID
export VAULT_SECRET_ID=$CI_VAULT_SECRET_ID

# Pull secrets at runtime (no caching in CI)
jan-config secrets pull --env=staging --no-cache
# Secrets injected as env vars for docker-compose
```

**Secret Rotation:**
```bash
# Rotate a secret in Vault
vault kv put secret/jan-server/database postgres_password=new_password

# Update local cache
jan-config secrets refresh --key=database.postgres_password

# Or refresh all
jan-config secrets pull --force

# Services auto-reload on secret change (via file watch or SIGHUP)
```

**Bootstrap Flow:**
```
┌─────────────────┐
│  jan-config CLI │
└────────┬────────┘
         │
         ├─1─> Detect environment (dev/staging/prod)
         │
         ├─2─> Check secret cache (~/.jan/secrets/{env}.enc)
         │     └─> Hit: Use cached secrets (check TTL)
         │     └─> Miss: Go to step 3
         │
         ├─3─> Authenticate to secret provider
         │     ├─> Vault: OIDC or AppRole
         │     ├─> K8s: ServiceAccount token
         │     ├─> AWS: IAM role
         │     └─> Fallback: Prompt for credentials
         │
         ├─4─> Fetch secrets from provider
         │     └─> Store in encrypted cache (optional)
         │
         └─5─> Inject into config loader
               └─> Services start with secrets
```

**Configuration:**
```yaml
# config/secrets/secrets.yaml (template)
secrets:
  # Secret definitions
  database:
    postgres_password:
      source: vault
      path: secret/data/jan-server/database
      key: postgres_password
      required: true
  
  auth:
    keycloak_admin_password:
      source: vault
      path: secret/data/jan-server/keycloak
      key: admin_password
      required: true
  
  providers:
    hf_token:
      source: env  # Local dev: from .env
      key: HF_TOKEN
      required: true
      
    serper_api_key:
      source: k8s
      name: external-apis
      key: serper_api_key
      required: false  # Optional

# Cache settings
cache:
  enabled: true
  location: ~/.jan/secrets
  ttl: 24h
  encryption: aes-256-gcm
  
# Provider configs
providers:
  vault:
    address: https://vault.example.com
    auth_method: oidc  # or: approle, token
    mount_path: secret
  
  k8s:
    namespace: jan-server
    service_account: jan-server-sa
```

**Benefits:**
- ✅ No secrets in config files
- ✅ Easy secret rotation
- ✅ Audit secret access
- ✅ Works offline (cached)
- ✅ Different sources per environment
- ✅ Clear developer onboarding

**Tasks:**
- [x] ~~Design SecretProvider interface with context + refresh~~ SKIPPED
- [x] ~~Implement EnvVarProvider (for local dev)~~ SKIPPED
- [x] ~~Implement VaultProvider with OIDC + AppRole auth~~ SKIPPED
- [x] ~~Implement K8sSecretProvider with ServiceAccount~~ SKIPPED
- [x] ~~Implement CachedProvider with encryption~~ SKIPPED
- [x] ~~Build `jan-config secrets` CLI commands (pull, push, refresh, status)~~ SKIPPED
- [x] ~~Add secret bootstrap to service startup~~ SKIPPED
- [x] ~~Document authentication setup per provider~~ SKIPPED
- [x] ~~Add secret rotation guide~~ SKIPPED
- [x] ~~Create troubleshooting guide (auth failures, cache issues)~~ SKIPPED

**Note:** Secrets are handled directly via:
- Local dev: `.env` files
- CI/CD: Environment variables injected by CI system
- K8s: K8s Secrets mounted as env vars or volumes
- Production: HashiCorp Vault managed by DevOps team

---

### Phase 6: Docker Compose Integration

**STATUS: ✅ COMPLETE - Pragmatic documentation approach**

**Goal:** Generate docker-compose from unified config

**Approach:**
```bash
# Generate docker-compose.generated.yml from config
make compose-generate ENV=development

# Or use dynamic compose file that reads from config
docker-compose -f docker-compose.dynamic.yml up
```

**Implementation:**
```go
// cmd/compose-generate/main.go
// Reads config/defaults.yaml + config/environments/{env}.yaml
// Generates docker-compose.yml with correct env vars
```

**Benefits:**
- ✅ No duplicate definitions
- ✅ Guaranteed sync between config and compose
- ✅ Can regenerate compose anytime
- ✅ Version control generated file for safety

**Tasks:**
- [ ] Create compose generator tool
- [ ] Template docker-compose with Go templates
- [ ] Handle conditional services (profiles)
- [ ] Add validation of generated compose
- [ ] Update Makefile to auto-generate
- [ ] Keep legacy compose files for transition

---

### Phase 7: Interactive Configuration Tool

**Goal:** Minimal CLI for validation and export (defer full UI until loader stabilizes)

**Phase 7a: Core Commands (Sprint 6 - 1 week)**
```bash
# Validation only (no mutations)
jan-config validate --env=production
jan-config validate-service llm-api --file=config/providers.yml

# Export for shell/docker
eval $(jan-config export --env=development)
jan-config export --env=production --format=docker-env > .env.prod

# Show effective config (debugging)
jan-config show --env=development --path=database.postgres
jan-config show --env=production --all
```

**Phase 7b: Full CLI (Sprint 10+ - after Docker/K8s gen stabilizes)**
```bash
# Initialize new environment
jan-config init --env=staging

# Show service-specific config (from service dir)
jan-config show-service llm-api --file=providers.yml

# Diff environments
jan-config diff development production

# Set/update values (infrastructure config only)
jan-config set infrastructure.database.postgres.port 5433

# Check for missing secrets
jan-config check-secrets --env=production

# Interactive mode
jan-config interactive --env=development
```

**Benefits (Phase 7a):**
- ✅ Validate before deployment (CI/CD friendly)
- ✅ Export for existing workflows
- ✅ Debug config issues
- ✅ No risk of half-baked UI blocking migrations

**Benefits (Phase 7b):**
- ✅ User-friendly config management
- ✅ No manual YAML editing
- ✅ Guided setup for new users

**Tasks (Phase 7a - Sprint 6):**
- [ ] Design minimal CLI interface (validate + export only)
- [ ] Implement validate command with clear error messages
- [ ] Implement export command (env, docker-env, json formats)
- [ ] Implement show command for debugging
- [ ] Add bash/zsh completions for core commands
- [ ] Write CLI documentation for core commands

**Tasks (Phase 7b - Sprint 10+):**
- [ ] Implement init command with templates
- [ ] Implement diff command with color output
- [ ] Implement set command with validation
- [ ] Add interactive mode with prompts
- [ ] Add service config commands
- [ ] Extend completions for all commands
- [ ] Update documentation

---

### Phase 8: Kubernetes Values Unification

**Goal:** Generate K8s values from same config source

**Approach:**
```bash
# Generate Helm values.yaml from unified config
jan-config k8s-generate --env=production --output=k8s/jan-server/values-production.yaml
```

**Benefits:**
- ✅ Docker and K8s use same config
- ✅ No manual sync needed
- ✅ Consistent across deployment methods

**Tasks:**
- [ ] Create K8s values generator
- [ ] Map config schema to Helm values structure
- [ ] Handle K8s-specific overrides
- [ ] Test generated values
- [ ] Update K8s deployment docs

---

### Phase 9: Configuration Documentation

**Goal:** Auto-generate docs from schema

**Outputs:**
1. **Reference Documentation**: All config options with types, defaults, descriptions
2. **Examples**: Pre-built configs for common scenarios
3. **Migration Guides**: How to move from old to new system
4. **Troubleshooting**: Common config issues and solutions

**Implementation:**
```bash
# Generate markdown docs from schema
jan-config docs generate --output=docs/configuration/

# Generate config examples
jan-config examples generate --output=examples/configs/
```

**Tasks:**
- [ ] Add descriptions to schema
- [ ] Create doc generator
- [ ] Generate reference docs
- [ ] Create example configs
- [ ] Write migration guide
- [ ] Add validation error messages

---

## Implementation Timeline

### Sprint 1 (Week 1-2): Foundation - Go Structs as Source of Truth
**Priority:** HIGH
- [ ] Define canonical Go structs in `pkg/config/types.go` with full annotations
- [ ] Set up generation pipeline (schema, YAML, docs from structs)
- [ ] Implement struct → JSON Schema generator
- [ ] Implement struct → YAML defaults generator
- [ ] Add CI test: verify generated files match structs (prevent drift)

**Deliverables:**
- `pkg/config/types.go` (canonical definitions)
- `pkg/config/codegen/` (generators)
- `config/schema/*.schema.json` (generated)
- `config/defaults.yaml` (generated)
- CI drift detection test

### Sprint 2 (Week 3-4): Core Loader with Precedence
**Priority:** HIGH
- [ ] Implement ConfigSource interface and precedence stack
- [ ] Implement each source: Struct defaults, YAML, EnvVars
- [ ] Add conflict resolution with documented merge strategies
- [ ] Add Provenance() debug method
- [ ] Write precedence regression tests (100+ test cases)
- [ ] Generate precedence documentation table

**Deliverables:**
- `pkg/config/loader.go` with precedence system
- Precedence regression test suite
- `docs/configuration/precedence.md` (auto-generated)

### Sprint 3 (Week 5-6): Service Migration
**Priority:** HIGH
- [ ] Migrate llm-api to use unified loader for env vars
- [ ] Keep llm-api providers.yml loading as-is (CI/CD managed)
- [ ] Migrate mcp-tools to use unified loader for env vars
- [ ] Keep mcp-tools mcp-providers.yml loading as-is (CI/CD managed)
- [ ] Integration tests with new config loader

**Deliverables:**
- Updated llm-api config loading
- Updated mcp-tools config loading
- Integration tests passing
- Service config documentation

### Sprint 4 (Week 7-8): Docker Integration
**Priority:** HIGH
- [ ] Create docker-compose generator from YAML config
- [ ] Generate compose files with correct env vars
- [ ] Update Makefile targets to use generated compose
- [ ] Test all deployment profiles (full, hybrid, etc.)

**Deliverables:**
- `cmd/compose-generate/main.go`
- Generated `docker-compose.generated.yml`
- Updated Makefile
- Profile tests

### Sprint 5 (Week 9-10): Secret Management
**Priority:** HIGH
- [ ] Implement SecretProvider interface with context + refresh
- [ ] Implement EnvVarProvider (for local dev)
- [ ] Implement VaultProvider with OIDC + AppRole auth
- [ ] Implement CachedProvider with encryption
- [ ] Add secret bootstrap to service startup
- [ ] Document authentication setup per provider

**Deliverables:**
- `pkg/config/secrets/` package
- Vault + EnvVar + Cache providers
- Secret bootstrap in loader
- Developer authentication guide

### Sprint 6 (Week 11): Minimal CLI Tool
**Priority:** MEDIUM (deferred full CLI)
- [ ] Build minimal `jan-config` CLI (validate + export only)
- [ ] Implement validate command with clear error messages
- [ ] Implement export command (env, docker-env, json formats)
- [ ] Implement show command for debugging
- [ ] Add bash/zsh completions for core commands

**Deliverables:**
- `cmd/jan-config/` minimal CLI
- Core commands (validate, export, show)
- Shell completions
- CLI documentation

### Sprint 7 (Week 12-13): K8s & Documentation
**Priority:** MEDIUM
- [ ] Create K8s values generator from YAML config
- [ ] Generate documentation from Go struct comments
- [ ] Create example configs for common scenarios
- [ ] Add configuration troubleshooting guide

**Deliverables:**
- `cmd/k8s-generate/main.go`
- Generated K8s values
- Auto-generated docs
- Example configs + troubleshooting

### Sprint 8 (Week 14): Final Services & Validation
**Priority:** MEDIUM
- [ ] Migrate remaining services (media-api, response-api)
- [ ] End-to-end integration tests
- [ ] Performance testing (config load time)
- [ ] Update all documentation

**Deliverables:**
- All services using unified config
- E2E test suite
- Performance benchmarks
- Complete documentation

### Sprint 9+ (Week 15+): Full CLI & Polish (Optional)
**Priority:** LOW (only after core stabilizes)
- [ ] Implement full CLI commands (init, diff, set, interactive)
- [ ] Add advanced features based on user feedback
- [ ] Create training materials

**Deliverables:**
- Full-featured CLI
- Training materials

---

## Governance & Change Management

### Schema Change Approval Process (RACI)

| Activity | Responsible | Accountable | Consulted | Informed |
|----------|-------------|-------------|-----------|----------|
| Propose schema change | Any Engineer | Team Lead | Platform Team | All Engineers |
| Review schema change | Platform Team | Tech Lead | Security, DevOps | - |
| Approve schema change | Tech Lead | Engineering Manager | - | All Teams |
| Implement change | Change Author | Team Lead | Platform Team | - |
| Update documentation | Change Author | Tech Lead | - | All Engineers |
| Deploy to production | DevOps | Tech Lead | Security | All Teams |

### Change Request Process

**For adding new config fields:**
```bash
# 1. Create branch
git checkout -b config/add-feature-flag

# 2. Update canonical Go structs
# Edit: pkg/config/types.go

# 3. Regenerate artifacts
make config-generate

# 4. Test changes
make config-test

# 5. Create PR with template
# Template includes: rationale, backward compat, migration plan

# 6. Get approval from Platform Team + Tech Lead

# 7. Merge and deploy
```

**For breaking changes (rare):**
- Requires Engineering Manager approval
- Must include migration script
- Requires communication plan (email to all engineers)
- Must be deployed with feature flag for rollback

### Configuration Versioning

**Semantic Versioning for Config Schema:**
- **Major**: Breaking changes (required field, type change, removed field)
- **Minor**: New optional fields, new services
- **Patch**: Documentation updates, defaults changes

**Version Tracking:**
```yaml
# config/defaults.yaml
meta:
  schema_version: "2.1.0"  # Validated on load
  compatible_with: ["2.0.0", "2.1.0"]  # Loader accepts these versions
```

### Deprecation Policy

**Deprecation Timeline:**
1. **Sprint N**: Mark field as deprecated in schema + docs
2. **Sprint N+2**: Add deprecation warning in loader
3. **Sprint N+4**: Remove field (if no usage in production)

**Deprecation Annotation:**
```go
type Config struct {
    // Deprecated: Use NewFieldName instead. Will be removed in v3.0.0
    OldFieldName string `yaml:"old_field" jsonschema:"deprecated"`
    
    NewFieldName string `yaml:"new_field" jsonschema:"required"`
}
```

### Testing Requirements

**All config changes must include:**
- [ ] Unit tests for new fields
- [ ] Precedence tests (if affects override behavior)
- [ ] Integration test in at least one service
- [ ] Validation test (invalid values rejected)
- [ ] Documentation updated (auto-generated + manual)

**CI Validation:**
```bash
# Runs on every PR touching pkg/config/
make config-test           # Run all config tests
make config-generate       # Ensure artifacts up to date
make config-drift-check    # Verify no hand-edits to generated files
make config-validate-all   # Validate all example configs
```

---

## References

### Internal Documentation
- `docs/guides/development.md` - Current dev guide
- `config/README.md` - Current config documentation
- `.env.template` - Current configuration template

### Related Issues/PRs
- (To be created after team review)

### External Resources
- [12-Factor App Configuration](https://12factor.net/config)
- [JSON Schema](https://json-schema.org/)
- [HashiCorp Vault](https://www.vaultproject.io/)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)

---

## Next Steps

### Immediate Actions (This Week)
1. **Review this document** with engineering team
2. **Prioritize sprints** based on team capacity
3. **Create GitHub issues** for each sprint
4. **Assign sprint 1 tasks** to begin implementation

### Team Discussion Topics
- [ ] Review and approve RACI matrix
- [ ] Agree on change approval process
- [ ] Set sprint schedule and team capacity
- [ ] Assign ownership for each phase
- [ ] Agree on Go structs as canonical source
- [ ] Review precedence rules and conflict resolution
- [ ] Review secret management workflow

### Approval Checklist
- [ ] Engineering team review (governance process)
- [ ] DevOps team review (CI/CD integration)
- [ ] Security review (secret management + audit)
- [ ] Platform team review (schema design)
- [ ] Tech Lead approval to proceed

---

## Appendix

### A. Current Configuration Files Inventory

**Environment Files (Root Level):**
- `.env.template` (205 lines, 60+ vars)
- `config/production.env.example` (145 lines)
- `config/secrets.env.example` (161 lines)
- `config/README.md` (361 lines docs)

**Service-Specific Configs (Pluggable, CI/CD Managed):**
- `services/llm-api/config/providers.yml` (60 lines) ⚠️ CI/CD managed
- `services/llm-api/config/providers_metadata_default.yml` ⚠️ CI/CD managed
- `services/mcp-tools/configs/mcp-providers.yml` (50 lines) ⚠️ CI/CD managed

**Docker Compose:**
- `docker-compose.yml` (8 lines, includes only)
- `docker/infrastructure.yml` (140 lines)
- `docker/services-api.yml` (120 lines)
- `docker/services-mcp.yml` (110 lines)
- `docker/inference.yml` (80 lines)
- `docker/dev-full.yml` (90 lines)
- `docker/observability.yml` (not reviewed)

**Service Configs:**
- `services/llm-api/internal/config/config.go` (243 lines, 40+ env vars)
- `services/llm-api/config/providers.yml` (60 lines)
- `services/llm-api/config/providers_metadata_default.yml` (not reviewed)
- `services/mcp-tools/infrastructure/config/config.go` (50 lines, 20+ vars)
- `services/mcp-tools/configs/mcp-providers.yml` (50 lines)
- `services/media-api/internal/config/config.go` (70+ env vars)
- `services/response-api/internal/config/config.go` (35+ env vars)

**Runtime Configs:**
- `kong/kong.yml` (378 lines)
- `kong/kong-dev-full.yml` (not reviewed)
- `keycloak/import/realm-jan.json` (not reviewed)

**K8s Configs:**
- `k8s/jan-server/values.yaml` (552 lines)
- `k8s/jan-server/values-development.yaml` (not reviewed)
- `k8s/jan-server/values-production.yaml` (not reviewed)

**Build Configs:**
- `Makefile` (728 lines)
- `.vscode/tasks.json` (via workspace, 20+ tasks)

**Total:** ~3,500+ lines of configuration across 30+ files

### B. Environment Variable Breakdown by Service

**Infrastructure (40 vars)**
- Postgres: 8 vars (user, password, db, host, port, ssl, connection pool)
- Keycloak: 15 vars (admin, realm, URLs, OAuth, JWKS)
- Kong: 5 vars (ports, admin URL, log level)
- Redis: 3 vars (host, port, password)
- Observability: 9 vars (OTEL, Prometheus, Grafana, Jaeger)

**LLM API Service (45 vars)**
- HTTP & Metrics: 3 vars
- Database: 4 vars
- Auth: 15 vars (Keycloak integration)
- API Keys: 6 vars
- Model Providers: 8 vars
- Features: 4 vars
- Logging: 5 vars

**MCP Tools Service (20 vars)**
- HTTP: 2 vars
- Search: 6 vars (Serper, SearXNG)
- Providers: 8 vars (Vector Store, SandboxFusion, etc.)
- Auth: 4 vars

**Media API Service (25 vars)**
- HTTP: 2 vars
- Database: 4 vars
- S3: 9 vars
- Features: 6 vars
- Auth: 4 vars

**Response API Service (15 vars)**
- HTTP: 2 vars
- Database: 4 vars
- Integrations: 4 vars
- Features: 3 vars
- Auth: 2 vars

**Inference (10 vars)**
- vLLM: 6 vars (model, port, GPU util, API key)
- HuggingFace: 1 var (token)
- Provider: 3 vars

**MCP Infrastructure (15 vars)**
- SearXNG: 4 vars
- Vector Store: 3 vars
- SandboxFusion: 3 vars
- Browser/Playwright: 5 vars

**Secrets (10 vars)**
- HF_TOKEN
- SERPER_API_KEY
- POSTGRES_PASSWORD
- KEYCLOAK_ADMIN_PASSWORD
- BACKEND_CLIENT_SECRET
- MODEL_PROVIDER_SECRET
- VLLM_INTERNAL_KEY
- GRAFANA_ADMIN_PASSWORD
- MEDIA_S3_ACCESS_KEY
- MEDIA_S3_SECRET_KEY

**Total: ~200 environment variables**

### C. Configuration Complexity Matrix

| Aspect | Current State | Target State | Improvement |
|--------|---------------|--------------|-------------|
| Files to Edit (Infra) | 8-12 files | 1-2 files | 80% reduction |
| Files to Edit (Service) | 1-2 files | 1-2 files | ✅ No change (CI/CD) |
| Lines of Config | 3,500+ lines | ~800 lines | 75% reduction |
| Duplication | High (4-5x) | None | 100% DRY |
| Validation | Runtime only | Pre-flight | ✅ Early catch |
| Type Safety | Weak | Strong | ✅ Schema-based |
| Documentation | Manual, stale | Auto-generated | ✅ Always current |
| Secret Management | In-file | External | ✅ Secure |
| Environment Switch | Manual | 1 command | ✅ Easy |
| Service Config Isolation | Mixed with infra | Separate dirs | ✅ CI/CD friendly |
| Discoverability | Poor | Excellent | ✅ Organized |
| Learning Curve | 2-3 days | 2-3 hours | 90% faster |

---

**End of Configuration Improvement Plan**

*For questions or feedback, please contact the platform team.*
