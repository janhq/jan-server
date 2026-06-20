# Configuration Precedence

How Jan Server loads configuration values and decides which source wins.

## Quick Summary

| Priority | Source                       | Notes                                         |
| -------- | ---------------------------- | --------------------------------------------- |
| **600**  | CLI flags (future)           | Planned for jan-cli integrations              |
| **500**  | Shell / OS environment vars  | Highest effective source today; secrets, CI/CD |
| **450**  | Root `.env`                  | Loaded into the process environment by Docker Compose |
| **300**  | `config/environments/*.yaml` | Per-environment overrides (opt-in; dir may be absent) |
| **200**  | `config/defaults.yaml`       | Generated defaults (`make config-generate`)   |
| **100**  | Struct tags (`envDefault`)   | Absolute fallback values                      |

**Rule**: the highest priority number wins every time. Example: `POSTGRES_PORT=5433` (priority 500) beats `port: 5432` in `defaults.yaml` (priority 200).

> **About the `.env` layer:** Jan Server ships a single root `.env` (created from
> `.env.template` by `make setup` / `make quickstart`). Docker Compose injects it
> into each container via `env_file: ${ENV_FILE:-../../.env}`, so by the time a
> service starts, `.env` values appear as ordinary OS environment variables.
> Anything you export in your shell still wins over the `.env` file. There is no
> `docker/.env` and no per-environment `config/<env>.env` files.

## Configuration Sources

### 1. StructDefaultSource (Priority 100)

**Lowest priority**. These are the hardcoded defaults embedded in Go struct tags in `packages/go-common/config/types.go`.

```go
type PostgresConfig struct {
 Port int `yaml:"port" json:"port" env:"POSTGRES_PORT" envDefault:"5432"`
 // ^^^^^^^^^^^^^^^^
 // Priority 100
}

// NOTE: Never store secrets in envDefault tags
```

**When to use:** Never set directly - these are fallback defaults.

**Example:**

```go
// From types.go
Host string `envDefault:"api-db"` // Priority 100: "api-db"
```

### 2. YAMLDefaultSource (Priority 200)

**Second priority**. Auto-generated `config/defaults.yaml` created by `make config-generate`.

```yaml
infrastructure:
  database:
  postgres:
  host: "api-db" # Priority 200
  port: 5432 # Priority 200
  user: "jan_user"
```

**When to use:** Baseline configuration for every environment. Generated via `make config-generate`; never edit manually.

**Example:**

```bash
# Generate defaults.yaml
make config-generate

# This creates config/defaults.yaml with all default values
```

### 3. YAMLEnvSource (Priority 300)

**Third priority**. Environment-specific configuration files in `config/environments/*.yaml`. This directory is **opt-in and not created by default**---add it only when you need per-environment overrides.

```yaml
# config/environments/production.yaml
infrastructure:
  database:
  postgres:
  host: "prod-db.example.com" # Priority 300: overrides defaults.yaml
  port: 5432 # Priority 300: same as default, redundant
  max_connections: 500 # Priority 300: production tuning
```

**When to use:**

- Production/staging-specific settings
- Infrastructure endpoints that differ per environment
- Feature flags per environment

**File naming** (create only the ones you need):

- `config/environments/development.yaml`
- `config/environments/staging.yaml`
- `config/environments/production.yaml`

**Example:**

```bash
# Load development environment
loader := config.NewConfigLoader("development", "config/defaults.yaml")
cfg, err := loader.Load(context.Background())

# Loads in order:
# 1. Struct defaults (100)
# 2. config/defaults.yaml (200)
# 3. config/environments/development.yaml (300) <- Environment-specific (if present)
# 4. Root .env (450, injected as env vars by Docker Compose)
# 5. Shell/OS environment variables (500)
```

### 4. EnvVarSource (Priority 500)

**Fourth priority**. System environment variables with names matching `env` struct tags.

```bash
# Priority 500: Overrides everything except CLI flags
export POSTGRES_HOST=override-db
export POSTGRES_PORT=5433
export AUTO_MIGRATE=false
```

**When to use:**

- Docker/Kubernetes deployments
- CI/CD pipelines
- Local development overrides
- Secrets and sensitive values (managed by DevOps)

**Tag mapping:**

```go
// From types.go
Host string `env:"POSTGRES_HOST"` // Set with: export POSTGRES_HOST=value
Port int `env:"POSTGRES_PORT"` // Set with: export POSTGRES_PORT=5433
```

**Example:**

```bash
# Override database host for local testing
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5433

# Run service - will use localhost:5433 instead of config values
./llm-api
```

### 5. CLI Flags (Priority 600) - Planned

**Highest priority**. Command-line flags (not yet implemented; planned).

```bash
# Planned for future
./llm-api --db-host=emergency-db --db-port=5434
```

**When to use:**

- Emergency overrides
- One-time testing
- Troubleshooting in production

## Conflict Resolution

### Merge Strategy

The configuration loader uses a **non-zero override** strategy:

1. Start with an empty `Config` struct
2. Load sources in ascending priority order (100 -> 200 -> 300 -> 450 -> 500)
3. For each source:

- Load configuration values
- **Only non-zero values** from the source override existing values
- Zero values are skipped (don't override with empty strings, 0, false)

### Example: Port Precedence

```go
// Initial state (empty Config)
Port: 0

// Load StructDefaultSource (Priority 100)
Port: 5432 // From envDefault tag

// Load YAMLDefaultSource (Priority 200)
Port: 5432 // defaults.yaml matches struct default, no change

// Load YAMLEnvSource (Priority 300)
Port: 5433 // production.yaml overrides to 5433

// Load EnvVarSource (Priority 500)
Port: 5434 // POSTGRES_PORT env var wins
 // Final value: 5434
```

### Provenance Tracking

The loader tracks which source provided each final value:

```go
loader := config.NewConfigLoader("production", "config/defaults.yaml")
cfg, _ := loader.Load(ctx)

// Print configuration sources
fmt.Println(loader.Provenance())

// Output:
// Configuration Sources (priority order):
// [100] struct-defaults
// [200] yaml-defaults
// [300] yaml-env-production
// [500] env-vars
//
// Loaded 127 configuration values
```

### Debug: Finding Value Origin

To debug where a specific value came from:

```go
info, err := loader.Provenance("infrastructure.database.postgres.port")
if err == nil {
 fmt.Printf("Port came from: %s (priority %d)\n", info.Source, info.Priority)
 // Output: Port came from: env-vars (priority 500)
}
```

## Common Scenarios

### Scenario 1: Development Override

**Goal:** Use localhost database for local development

```bash
# Set environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432

# Run service
go run./cmd/server
```

**Result:**

- `POSTGRES_HOST`: localhost (priority 500, env var)
- `POSTGRES_PORT`: 5432 (priority 500, env var)
- All other settings: from defaults.yaml (priority 200)

### Scenario 2: Production Deployment

**Goal:** Use production database with secrets

```yaml
# config/environments/production.yaml (priority 300)
infrastructure:
  database:
  postgres:
  host: "prod-db.company.com"
  ssl_mode: "require"
```

```bash
# Kubernetes secret (priority 500)
export POSTGRES_PASSWORD=<secret-from-vault>
export POSTGRES_USER=<secret-from-vault>

# Run with production environment
./llm-api --environment=production
```

**Result:**

- `host`: prod-db.company.com (priority 300, environment YAML)
- `ssl_mode`: require (priority 300, environment YAML)
- `user`, `password`: from env vars (priority 500, secrets)
- All other settings: from defaults.yaml (priority 200)

### Scenario 3: Temporary Override

**Goal:** Test with different port without changing config

```bash
# One-time override for testing
export POSTGRES_PORT=9999

# Run test
go test./...

# Unset when done
unset POSTGRES_PORT
```

**Result:**

- `POSTGRES_PORT`: 9999 (priority 500) during test
- All other services still use 5432 from defaults

## Data Type Handling

### Strings

```bash
export POSTGRES_HOST=localhost # Simple string
export KEYCLOAK_REALM=jan # No quotes needed
```

### Integers

```bash
export POSTGRES_PORT=5433 # Parsed as int
export POSTGRES_MAX_CONNECTIONS=200
```

### Booleans

```bash
export AUTO_MIGRATE=true # Accepts: true, false, 1, 0
export OTEL_ENABLED=false
```

### Durations

```bash
export DB_CONN_MAX_LIFETIME=45m # Go duration format
export MEDIA_S3_PRESIGN_TTL=10m # Supports: ns, us, ms, s, m, h
```

### Slices (comma-separated)

```bash
export KEYCLOAK_FEATURES=token-exchange,preview,admin-api
# Parsed as: []string{"token-exchange", "preview", "admin-api"}
```

## Best Practices

### OK DO

1. **Use environment variables for secrets**

```bash
export POSTGRES_PASSWORD=$VAULT_SECRET
export AWS_ACCESS_KEY_ID=$AWS_KEY
```

2. **Use environment YAML for per-environment infrastructure**

```yaml
# config/environments/staging.yaml
infrastructure:
database:
postgres:
host: "staging-db.internal"
```

3. **Keep defaults.yaml comprehensive**

```bash
# Regenerate after adding new config fields
make config-generate
```

4. **Document precedence in comments**

```yaml
# config/environments/production.yaml
infrastructure:
database:
postgres:
# Override for production (priority 300)
# Can still be overridden by POSTGRES_HOST env var (priority 500)
host: "prod-db.company.com"
```

### [X] DON'T

1. **Don't manually edit defaults.yaml**

```bash
# [X] WRONG: Manual edits will be overwritten
vim config/defaults.yaml

# OK CORRECT: Edit types.go and regenerate
vim packages/go-common/config/types.go
make config-generate
```

2. **Don't put secrets in YAML files**

```yaml
# [X] WRONG: Secret in version control
infrastructure:
database:
postgres:
password: "super-secret-123" # DON'T DO THIS


# OK CORRECT: Use environment variable (managed by DevOps)
# export POSTGRES_PASSWORD=super-secret-123
# Or use K8s Secrets, Vault, etc.
```

3. **Don't override everything in environment YAML**

```yaml
# [X] WRONG: Duplicating all defaults
infrastructure:
database:
postgres:
host: "prod-db.com"
port: 5432 # Redundant with default
user: "jan_user" # Redundant with default
database: "jan_llm_api" # Redundant with default

# OK CORRECT: Only override what's different
infrastructure:
database:
postgres:
host: "prod-db.com" # Only this is different
```

4. **Don't rely on zero-value overrides**

```bash
# [X] WRONG: Trying to "unset" a value
export AUTO_MIGRATE= # Empty string won't override true

# OK CORRECT: Explicitly set to false
export AUTO_MIGRATE=false
```

## Testing Precedence

### Unit Test Example

```go
func TestConfigPrecedence(t *testing.T) {
 // Set environment variable (priority 500)
 os.Setenv("POSTGRES_PORT", "9999")
 defer os.Clearenv()

 // Create YAML file (priority 200)
 yaml := `
infrastructure:
 database:
 postgres:
 port: 5433
`
 os.WriteFile("config/defaults.yaml", []byte(yaml), 0644)

 // Load configuration
 loader := config.NewConfigLoader("development", "config/defaults.yaml")
 cfg, err := loader.Load(context.Background())

 // Environment variable should win
 assert.Equal(t, 9999, cfg.Infrastructure.Database.Postgres.Port)

 // Check provenance
 prov := loader.Provenance()
 assert.Contains(t, prov, "env-vars") // Confirm env vars were loaded
}
```

````

## Troubleshooting

### Problem: Configuration not being applied

**Symptom:** Set `POSTGRES_PORT=9999` but service still uses 5432

**Solution:**
1. Check environment variable name matches `env` tag:
 ```go
 Port int `env:"POSTGRES_PORT"` // Must be exact match
````

2. Verify environment variable is set in correct shell:

```bash
printenv | grep POSTGRES_PORT
```

3. Check provenance to see what overrode it:

```go
info, _ := loader.Provenance("infrastructure.database.postgres.port")
fmt.Printf("Source: %s (priority %d)\n", info.Source, info.Priority)
```

### Problem: Can't override YAML value

**Symptom:** Changed `production.yaml` but using old value

**Solution:**

1. Check file path matches environment:

```bash
ls config/environments/production.yaml # Must exist
```

2. Verify you're loading correct environment:

```go
loader := config.NewConfigLoader("production", "config/defaults.yaml")
// ^^^^^^^^^^^ Must match filename
```

3. Check for environment variable override (priority 500 > 300):

```bash
unset POSTGRES_HOST # Remove higher-priority override
```

### Problem: Defaults not updating

**Symptom:** Changed `envDefault` tag but `defaults.yaml` unchanged

**Solution:**

```bash
# Regenerate defaults.yaml
make config-generate

# Verify CI drift detection passes
make config-drift-check
```

## See Also

- [Configuration Package README](../../packages/go-common/config/README.md) - Implementation details
- [Config Types Reference](../../packages/go-common/config/types.go) - All configuration fields
- [Code Generation](../../packages/go-common/config/codegen/) - Schema and YAML generators
- [Configuration README](README.md) - Overview and common tasks
