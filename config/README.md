# Environment Configuration Guide

## Overview

Jan Server uses environment variables for configuration. This directory contains environment-specific configuration files that override the base template.

## Quick Start

```bash
# 1. Create .env from template
make env-create

# 2. Edit .env and set required secrets:
#    - HF_TOKEN (HuggingFace)
#    - SERPER_API_KEY (Serper)
#    - Update passwords/secrets

# 3. Choose your environment
make env-switch ENV=development   # Docker development (default)
make env-switch ENV=hybrid        # Hybrid development
make env-switch ENV=testing       # Integration testing
```

## Environment Files

### `.env.template` (Root)
- **Purpose**: Comprehensive template with documentation
- **Usage**: Copy to `.env` to get started
- **Includes**: All variables with explanations and defaults
- **This is the ONLY template** - use this instead of old .env.example

### `config/defaults.env`
- **Purpose**: Base defaults inherited by all environments
- **DO NOT** modify unless changing defaults globally
- **Includes**: Non-sensitive defaults, ports, flags

### `config/development.env`
- **Purpose**: Full Docker development (all services containerized)
- **Use when**: Running everything in Docker
- **URLs**: Use Docker internal DNS (e.g., `keycloak:8085`)
- **Command**: `make env-switch ENV=development` or `make up-full`

### `config/hybrid.env`
- **Purpose**: Hybrid development (services run natively, infra in Docker)
- **Use when**: Debugging llm-api or mcp-tools with native debugger
- **URLs**: Use localhost (e.g., `localhost:8085`)
- **Command**: `make env-switch ENV=hybrid` then run services natively

### `config/testing.env`
- **Purpose**: Integration testing with Newman
- **Use when**: Running `make test-all`
- **URLs**: Use localhost for test access
- **Command**: Used automatically by test targets

### `config/production.env.example`
- **Purpose**: Production deployment template
- **Use when**: Deploying to production
- **Security**: Copy to `config/production.env` and customize
- **WARNING**: NEVER commit actual production.env

### `config/secrets.env.example`
- **Purpose**: List of all required secrets
- **Use when**: Setting up new environment
- **Security**: Reference for secret management setup

## Environment Patterns

### URL Patterns by Environment

| Environment | Database | Keycloak | MCP Tools |
|-------------|----------|----------|-----------|
| **Development** | `api-db:5432` | `keycloak:8085` | `searxng:8080` |
| **Hybrid** | `localhost:5432` | `localhost:8085` | `localhost:8086` |
| **Testing** | `localhost:5432` | `localhost:8085` | `localhost:8086` |
| **Production** | External DB URL | External Keycloak | External URLs |

### Required Secrets

All environments require these secrets (set in `.env`):

```bash
# API Keys
HF_TOKEN=hf_xxxxx                    # HuggingFace token
SERPER_API_KEY=xxxxx                 # Serper API key

# Security
POSTGRES_PASSWORD=xxxxx              # Database password
KEYCLOAK_ADMIN_PASSWORD=xxxxx        # Keycloak admin password
BACKEND_CLIENT_SECRET=xxxxx          # OAuth client secret
VLLM_INTERNAL_KEY=xxxxx              # vLLM API key
MODEL_PROVIDER_SECRET=xxxxx          # Model provider secret
```

## Switching Environments

### Method 1: Makefile (Recommended)

```bash
# Switch to hybrid development
make env-switch ENV=hybrid

# Switch back to Docker development
make env-switch ENV=development

# Switch to testing
make env-switch ENV=testing
```

### Method 2: Manual Copy

```bash
# Linux/macOS
cp config/hybrid.env .env

# Windows PowerShell
Copy-Item config\hybrid.env .env
```

### Method 3: ENV_FILE Variable

```bash
# Use specific env file with docker-compose
ENV_FILE=config/hybrid.env docker-compose up
```

## Validation

### Check Current Environment

```bash
make env-validate
```

### Verify Required Variables

```bash
make check-deps
```

## Best Practices

### 1. Never Commit Secrets
- `.env` is gitignored
- Only commit `.env.template` and `config/*.env.example`
- Use secret management in production

### 2. Use Environment Switcher
```bash
# Good
make env-switch ENV=hybrid

# Avoid
vi .env  # Manual editing error-prone
```

### 3. Document Custom Variables
```bash
# Add to .env.template with comments
# MY_CUSTOM_VAR=default_value  # Description of what this does
```

### 4. Separate Secrets from Config
- Configuration: In version control (config/*.env)
- Secrets: In .env (not in version control)
- Production: Use secret management (Vault, AWS Secrets Manager, etc.)

## Troubleshooting

### Service Can't Connect to Database

**Symptom**: `connection refused` errors

**Solution**: Check DB_DSN matches your environment
- Development: `api-db:5432`
- Hybrid: `localhost:5432`

```bash
# Fix for hybrid mode
make env-switch ENV=hybrid
make restart
```

### Keycloak JWT Validation Fails

**Symptom**: `401 Unauthorized`

**Solution**: Check JWKS_URL matches your environment
- Development: `http://keycloak:8085/...`
- Hybrid: `http://localhost:8085/...`

```bash
# Verify URLs
grep JWKS_URL .env
grep KEYCLOAK_BASE_URL .env
```

### MCP Tools Not Found

**Symptom**: MCP tools timeout or not found

**Solution**: Check MCP provider URLs match your environment
```bash
# Development
SEARXNG_URL=http://searxng:8080

# Hybrid
SEARXNG_URL=http://localhost:8086
```

### Environment Switch Not Working

**Symptom**: Changes not reflected after `make env-switch`

**Solution**:
```bash
# Restart services to pick up new .env
make restart

# Or full restart
make down && make up-full
```

### Missing Required Secrets

**Symptom**: Services fail to start, missing API keys

**Solution**:
```bash
# Check what secrets are needed
cat config/secrets.env.example

# Set in .env
vi .env  # Add HF_TOKEN, SERPER_API_KEY, etc.
```

## Example Workflows

### Docker Development
```bash
make setup
make env-switch ENV=development
make up-full
# All services running in Docker
```

### Hybrid Development
```bash
make setup
make env-switch ENV=hybrid
make up-infra             # Start only infrastructure
cd services/llm-api
air                       # Run API natively with hot reload
```

### Integration Testing
```bash
make setup
make env-switch ENV=testing
make test-setup
make test-all
```

### Production Deployment
```bash
# 1. Copy production template
cp config/production.env.example config/production.env

# 2. Edit with production values
vi config/production.env

# 3. Set secrets (use secret manager in real production)
# Set HF_TOKEN, SERPER_API_KEY, passwords, etc.

# 4. Use in deployment
cp config/production.env .env
# Deploy with your orchestration tool (Docker Swarm, Kubernetes, etc.)
```

## Migration from Old Structure

### Removed Files

The following files were removed in the restructure:

| Old File | Replacement |
|----------|-------------|
| `.env.example` | Use `.env.template` |
| `.env.docker` | Use `config/development.env` |
| `.env.local` | Use `config/hybrid.env` |
| `.env.mcp.example` | Merged into `.env.template` |

### Migration Steps

If you were using old files:

```bash
# 1. Backup current .env
cp .env .env.backup

# 2. Create new .env from template
make env-create

# 3. Restore your secrets from backup
# Copy: HF_TOKEN, SERPER_API_KEY, passwords, etc.
# Use: vi .env or your preferred editor

# 4. Choose environment
make env-switch ENV=development  # or hybrid, testing
```

## Advanced Topics

### Adding a New Environment

1. Create `config/myenv.env`:
```bash
# My Custom Environment
DB_DSN=postgres://user:pass@custom-db:5432/dbname
KEYCLOAK_BASE_URL=http://custom-keycloak:8085
# ... other overrides
```

2. Switch to it:
```bash
make env-switch ENV=myenv
```

### Using Multiple Environments Simultaneously

```bash
# Terminal 1: Development environment
ENV_FILE=config/development.env docker-compose -p dev up

# Terminal 2: Testing environment
ENV_FILE=config/testing.env docker-compose -p test up
```

### Environment Variable Precedence

1. Shell environment variables (highest priority)
2. `.env` file
3. `config/<environment>.env` (when explicitly loaded)
4. `config/defaults.env` (lowest priority)
5. Defaults in docker-compose.yml

## Security Checklist

- [ ] `.env` is in `.gitignore`
- [ ] Never commit `.env` with real secrets
- [ ] Change default passwords in production
- [ ] Use strong passwords (20+ characters)
- [ ] Use secret management in production (Vault, AWS Secrets Manager)
- [ ] Rotate secrets regularly
- [ ] Limit access to production `.env` files
- [ ] Use read-only secrets in containers when possible

## Reference

### All Environment Variables

For complete list of variables, see:
- `.env.template` - Full template with documentation
- `config/secrets.env.example` - Required secrets list
- `config/defaults.env` - Default values

### Commands

```bash
make env-create          # Create .env from template
make env-switch ENV=X    # Switch to environment X
make env-validate        # Validate current .env
make env-list            # List available environments
make check-deps          # Check required tools
```

---

**Quick Reference**: `make help-env` | **Validate**: `make env-validate` | **Switch**: `make env-switch ENV=<environment>`
