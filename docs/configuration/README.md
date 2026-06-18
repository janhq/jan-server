# Configuration System

Jan Server uses a simple configuration system with default values that you can override.

## Why This Matters

- **Safe:** Catches configuration errors before startup
- **Clear:** Easy to see what settings are available
- **Flexible:** Works in development and production
- **Validated:** Tells you exactly what's wrong if config is invalid

## How It Works

Configuration is loaded in this order (later overrides earlier):

1. **YAML defaults** - Built-in sensible defaults (`config/defaults.yaml`)
2. **Environment file** - Your specific settings (`config/production.yaml`)
3. **Environment variables** - Highest priority (great for secrets)

## Quick Commands

```bash
# Check if your configuration is valid
jan-cli config validate

# See all current settings
jan-cli config export

# See settings for one service
jan-cli config show llm-api
```

For the full set of commands (for example `config generate`, `config k8s-values`, and `config drift`), see the [Jan CLI Guide](../guides/jan-cli.md).

## Documentation Structure

This directory contains configuration system implementation details:

| Document                                     | Description                                      |
| -------------------------------------------- | ------------------------------------------------ |
| [precedence.md](precedence.md)               | Configuration precedence rules and loading order |
| [env-var-mapping.md](env-var-mapping.md)     | Environment variable to config mapping           |
| [docker-compose.md](docker-compose.md)       | Docker Compose integration                       |
| [service-migration.md](service-migration.md) | Migrating services to new config system          |

**For user-facing documentation:**

- **[Jan CLI Guide](../guides/jan-cli.md)** - Command-line tool for configuration management
- **[Testing Guide](../guides/testing.md)** - Cross-platform testing procedures

## Configuration Files

### defaults.yaml

Base configuration with sensible defaults for all environments:

```yaml
environment: development

services:
  llm-api:
  http:
  port: 8080
  timeout: 30s
  database:
  dsn: postgres://jan_user:jan_password@localhost:5432/jan_llm_api
  max_idle_conns: 10
  max_open_conns: 30
  auth:
  enabled: true
  issuer: http://localhost:8085/realms/jan
```

### Environment-specific YAMLs

Optional overrides live in `config/environments/<environment>.yaml`. The directory is not created by default---add it as needed:

```yaml
# config/environments/production.yaml
environment: production

services:
  llm-api:
  http:
  timeout: 60s
  database:
  max_idle_conns: 20
  max_open_conns: 100
  observability:
  enabled: true
  endpoint: https://otel-collector.prod.example.com
```

When you run the loader with `environment=production`, the stack becomes:

1. Struct defaults (priority 100)
2. `config/defaults.yaml` (priority 200)
3. `config/environments/production.yaml` (priority 300) --- **create this file**
4. Environment variables (priority 500)

### Environment Files (`.env`)

The repo ships with ready-made `.env` templates for Docker workflows:

| File                            | Purpose                                                         |
| ------------------------------- | --------------------------------------------------------------- |
| `.env.template`                 | Base template used by `make quickstart`                         |
| `.env`                          | Generated interactive configuration (git-ignored)               |
| `config/production.env.example` | Example values for production CI/CD                             |
| `config/secrets.env.example`    | Placeholder for sensitive values (copy to `config/secrets.env`) |

Use `make quickstart` or `make setup` to populate `.env`; copy the example files when preparing staging/production pipelines.

### Environment Variables

Highest priority - override any YAML setting:

```bash
# Override HTTP port
export LLM_API_HTTP_PORT=9090

# Override database connection
export LLM_API_DATABASE_DSN=postgres://user:pass@prod-db:5432/db

# Override observability
export LLM_API_OBSERVABILITY_ENABLED=true
```

## Common Tasks

### Adding a New Configuration Field

1. **Update Go struct** in `pkg/config/types.go`:

```go
type HTTPConfig struct {
Port int `yaml:"port" env:"HTTP_PORT" default:"8080"`
Timeout time.Duration `yaml:"timeout" env:"HTTP_TIMEOUT" default:"30s"`
// Add new field
MaxBodySize int64 `yaml:"max_body_size" env:"HTTP_MAX_BODY_SIZE" default:"10485760"`
}
```

2. **Regenerate config files**:

```bash
make config-generate
```

3. **Update defaults.yaml** (if auto-generated values aren't sufficient):

```yaml
services:
llm-api:
http:
max_body_size: 10485760 # 10 MB
```

4. **Test your changes**:

```bash
make config-test
jan-cli config validate
```

### Validating Configuration

```bash
# Validate current configuration
jan-cli config validate

# Validate with specific environment
ENVIRONMENT=production jan-cli config validate

# Check for configuration drift (CI/CD)
make config-drift-check
```

### Generating Kubernetes Values

```bash
# Generate values for all environments
jan-cli config k8s-values --env development > k8s/jan-server/values-development.yaml
jan-cli config k8s-values --env production > k8s/jan-server/values-production.yaml

# Generate with overrides
jan-cli config k8s-values --env production \
 --set services.llm-api.replicas=3 \
 --set services.llm-api.resources.limits.memory=2Gi \
 > k8s/values-prod-scaled.yaml
```

## Architecture

### Package Structure

```
pkg/config/
+-- types.go # Configuration structs (source of truth)
+-- loader.go # YAML + env loading logic
+-- validation.go # Validation rules
+-- provenance.go # Track config source
+-- env.go # Environment variable helpers
+-- k8s/
| +-- values_generator.go # Helm values generator
+-- testdata/ # Test fixtures

config/
+-- defaults.yaml # Auto-generated base defaults
+-- development.yaml # Dev overrides (optional)
+-- staging.yaml # Staging overrides (optional)
+-- production.yaml # Production overrides (optional)

tools/jan-cli/
+-- main.go # CLI tool
```

### Design Principles

1. **Single Source of Truth:** Go structs define all configuration
2. **Auto-Generation:** YAML, JSON Schema, and docs generated from code
3. **Fail Fast:** Validation at startup prevents runtime errors
4. **Environment Parity:** Same config structure across all environments
5. **Override by Exception:** Defaults work everywhere, override only what's different
6. **Explicit Over Implicit:** No magic values or hidden defaults

## Migration Guide

Migrating from old environment-variable-only approach:

### Before (Old Way)

```go
type Config struct {
 HTTPPort int `env:"HTTP_PORT" envDefault:"8080"`
 DBPostgresqlWriteDSN string `env:"DB_POSTGRESQL_WRITE_DSN,notEmpty"`
 LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
 AuthEnabled bool `env:"AUTH_ENABLED" envDefault:"false"`
 //... 50+ more variables
}
```

**Problems:**

- 50+ environment variables per service
- No validation until runtime
- Hard to see effective configuration
- Difficult to manage across environments
- No documentation of what's actually used

### After (New Way)

```go
import "jan-server/pkg/config"

cfg, _:= config.Load()
serviceCfg, _:= cfg.GetServiceConfig("llm-api")

// Type-safe access
port:= serviceCfg.HTTP.Port
dbDSN:= serviceCfg.Database.DSN
```

**Benefits:**

- ~10 environment variables per service (only overrides)
- Validated at load time
- `jan-cli config export` shows effective config
- YAML files for environment differences
- Auto-generated documentation

See [service-migration.md](service-migration.md) for detailed migration steps.

## Best Practices

### 1. Use Defaults for Common Values

Put shared defaults in `config/defaults.yaml`:

```yaml
# Good: Shared defaults
services:
  llm-api:
  http:
  timeout: 30s
  port: 8080
```

### 2. Override Only What's Different

Environment-specific files should be minimal:

```yaml
# config/production.yaml - Only overrides
services:
  llm-api:
  database:
  max_open_conns: 100 # Higher for production
  observability:
  enabled: true
```

### 3. Use Environment Variables for Secrets

Never put secrets in YAML files:

```bash
#.env or CI/CD secrets
export LLM_API_DATABASE_DSN=postgres://user:${DB_PASSWORD}@prod-db/db
export LLM_API_AUTH_CLIENT_SECRET=${KEYCLOAK_CLIENT_SECRET}
```

### 4. Validate Early

Add validation to your config loading:

```go
cfg, err:= config.Load()
if err != nil {
 log.Fatal("invalid configuration: %w", err)
}
```

### 5. Use CLI Tools in CI/CD

Prevent configuration drift:

```yaml
#.github/workflows/ci.yml
- name: Validate configuration
 run: |
 make config-drift-check
 jan-cli config validate
```

## Troubleshooting

### Configuration Not Loading

```bash
# Check what's being loaded
jan-cli config show llm-api

# Validate configuration
jan-cli config validate

# Export effective config
jan-cli config export
```

### Environment Variable Not Working

Check the naming convention - use service prefix:

```bash
# Wrong
export HTTP_PORT=9090

# Correct
export LLM_API_HTTP_PORT=9090
```

### Kubernetes Values Not Applying

Regenerate values after config changes:

```bash
make config-generate
jan-cli config k8s-values --env production > k8s/values-prod.yaml
helm upgrade jan-server k8s/jan-server -f k8s/values-prod.yaml
```

## Reference

### Documentation

- **CLI Guide:** [docs/guides/jan-cli.md](../guides/jan-cli.md) - Installation, usage, and examples
- **CLI Command Reference:** [tools/jan-cli/README.md](../../tools/jan-cli/README.md)
- **Configuration Types:** [pkg/config/README.md](../../pkg/config/README.md)

### Code References

- **Go Package:** `pkg/config/` in workspace root
- **Default Config:** [config/defaults.yaml](../../config/defaults.yaml)
- **JSON Schema:** [config-schema.json](config-schema.json) (auto-generated)

## Examples

See working examples in:

- **LLM API:** `services/llm-api/internal/config/`
- **Template API:** `services/template-api/internal/config/` (shows both approaches)
- **MCP Tools:** `services/mcp-tools/configs/`

---

**Need help?** See [service-migration.md](service-migration.md) or check existing service implementations for patterns.
