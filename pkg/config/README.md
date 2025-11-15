# Configuration Management System

This directory contains the unified configuration management system for Jan Server.

## Overview

All configuration is defined canonically in Go structs (`pkg/config/types.go`). From these structs, we automatically generate:

- **JSON Schema** (`config/schema/*.schema.json`) - For validation and IDE autocomplete
- **YAML Defaults** (`config/defaults.yaml`) - Default values for all settings
- **Documentation** (future) - Auto-generated configuration reference

## Structure

```
pkg/config/
├── types.go              # 🌟 Canonical source of truth (Go structs)
├── codegen/
│   ├── schema.go         # JSON Schema generator
│   └── yaml.go           # YAML defaults generator
└── loader.go             # Configuration loader (Sprint 2)

cmd/config-generate/
└── main.go               # Code generation CLI tool

config/
├── schema/               # Generated JSON schemas
│   ├── config.schema.json
│   ├── infrastructure.schema.json
│   ├── services.schema.json
│   ├── inference.schema.json
│   └── monitoring.schema.json
├── defaults.yaml         # Generated default configuration
└── environments/         # Environment-specific overrides
    ├── development.yaml
    ├── staging.yaml
    └── production.yaml
```

## Usage

### Generate Configuration Artifacts

```bash
# Generate all artifacts (JSON Schema + YAML defaults)
make config-generate

# Test configuration
make config-test

# Check for drift (CI check)
make config-drift-check
```

### Adding New Configuration

1. **Edit Go structs** in `pkg/config/types.go`:
```go
type MyNewConfig struct {
    // Port for the new service
    Port int `yaml:"port" json:"port" env:"MY_SERVICE_PORT" envDefault:"8080" 
             jsonschema:"required,minimum=1,maximum=65535" 
             description:"My service HTTP port"`
}
```

2. **Regenerate artifacts**:
```bash
make config-generate
```

3. **Commit both** `types.go` and generated files:
```bash
git add pkg/config/types.go config/schema/ config/defaults.yaml
git commit -m "config: add MyNewConfig"
```

## Struct Tags Reference

Each field should have these tags:

- `yaml:"field_name"` - YAML field name
- `json:"field_name"` - JSON field name
- `env:"ENV_VAR_NAME"` - Environment variable name
- `envDefault:"value"` - Default value
- `jsonschema:"..."` - JSON Schema constraints (required, minimum, maximum, enum, etc.)
- `description:"..."` - Human-readable description

### Example:
```go
// Database port
Port int `yaml:"port" json:"port" env:"POSTGRES_PORT" envDefault:"5432" 
         jsonschema:"required,minimum=1,maximum=65535" 
         description:"PostgreSQL port"`
```

## Configuration Hierarchy

### Root `/config` - Infrastructure & Environment
- Database connections, ports, auth settings
- Environment-specific overrides
- Managed through YAML + env vars

### Service `/config` or `/configs` - Pluggable Configs (CI/CD Managed)
- `services/llm-api/config/providers.yml` - Model providers
- `services/mcp-tools/configs/mcp-providers.yml` - MCP tools
- These files are **replaced by CI/CD**, not loaded from root config

## Design Principles

1. **Go structs are the source of truth** - Everything generates from them
2. **No manual editing of generated files** - CI enforces this
3. **Service configs stay in service dirs** - CI/CD can replace them independently
4. **Environment-specific overrides** - Only define what changes
5. **Secrets externalized** - Never in config files

## CI/CD Integration

### Pre-commit Hook
```bash
# Regenerate and check for drift
make config-drift-check
```

### CI Pipeline
```yaml
- name: Check config drift
  run: |
    make config-generate
    git diff --exit-code config/
```

## Roadmap

### Sprint 1 (Current)
- [x] Define canonical Go structs
- [x] JSON Schema generator
- [x] YAML defaults generator
- [ ] CI drift detection test

### Sprint 2 (Next)
- [ ] Configuration loader with precedence
- [ ] Environment override support
- [ ] Secret provider integration

### Future
- [ ] Documentation generator
- [ ] CLI tool (`jan-config`)
- [ ] Docker Compose generator
- [ ] Kubernetes values generator

## Questions?

See `config-improve-todo.md` for the complete implementation plan.
