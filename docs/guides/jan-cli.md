# Jan CLI - Complete Guide

Complete documentation for the Jan CLI tool - installation, usage, commands, and technical details.

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Installation](#installation)
4. [Commands Reference](#commands-reference)
5. [Configuration Management](#configuration-management)
6. [Service Operations](#service-operations)
7. [Development Tools](#development-tools)
8. [Troubleshooting](#troubleshooting)
9. [Shell Completion](#shell-completion)
10. [Technical Details](#technical-details)

---

## Overview

Jan CLI is the official command-line interface for Jan Server, providing unified access to:

- **Configuration Management** - Validate, export, and inspect configuration
- **Service Operations** - List services, view logs, check status
- **Development Tools** - Setup environment, scaffold services
- **Shell Completion** - Auto-completion for all major shells

Built with [Cobra framework](https://github.com/spf13/cobra), the industry standard used by kubectl, docker, and github CLI.

### Key Features

- **Unified Interface** - Single command for all Jan Server operations
- **Professional Structure** - Industry-standard Cobra framework
- **Extensible** - Easy to add new commands
- **Well-Documented** - Comprehensive help and examples
- **Cross-Platform** - Works on Windows, Linux, macOS
- **Shell Completion** - Bash, Zsh, Fish, PowerShell support

---

## Quick Start

### Install Globally (Recommended)

```bash
# From project root
make cli-install
```

This will:

1. Build the `jan-cli` binary
2. Install to your user's local bin directory
3. Display PATH setup instructions

**Installation Locations:**

- **Linux/macOS:** `~/bin/jan-cli`
- **Windows:** `%USERPROFILE%\bin\jan-cli.exe`

### Add to PATH

**Windows (PowerShell):**

```powershell
# Temporary (current session)
$env:PATH += ";$env:USERPROFILE\bin"

# Permanent (add to PowerShell profile)
notepad $PROFILE
# Add this line:
$env:PATH += ";$env:USERPROFILE\bin"
```

**Linux/macOS (Bash/Zsh):**

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$PATH:$HOME/bin"

# Reload your shell
source ~/.bashrc # or source ~/.zshrc
```

### Verify Installation

```bash
jan-cli --version
# Output: jan-cli version <version>

jan-cli --help
# Output: Full help text with all commands
```

### First Commands

```bash
# List all services
jan-cli service list

# Validate configuration
jan-cli config validate

# Show help for any command
jan-cli config --help
```

---

## Installation

### Method 1: Global Installation (Recommended)

Use the Makefile target to build and install `jan-cli`:

```bash
# From project root
make cli-install
```

**What it does:**

1. Builds the binary with `go build`
2. Creates `~/bin` or `%USERPROFILE%\bin` if needed
3. Copies binary to bin directory
4. Sets execute permissions (Unix)
5. Checks if bin is in PATH
6. Shows PATH setup instructions if needed

**After installation:**

```bash
# Add to PATH (see instructions from install output)
# Then use from anywhere
jan-cli --version
jan-cli config validate
jan-cli service list
```

### Method 2: Wrapper Scripts (No Installation)

Run from the project root using the wrapper scripts in `tools/`:

```bash
# Linux/macOS
./tools/jan-cli.sh config validate
./tools/jan-cli.sh service list

# Windows PowerShell
.\tools\jan-cli.ps1 config validate
.\tools\jan-cli.ps1 service list
```

**Advantages:**

- No installation needed
- Auto-builds if binary missing or outdated
- Always uses latest code
- Good for development

**Disadvantages:**

- Run from the project root
- Requires the file extension (`.sh` or `.ps1`)

### Method 3: Manual Build

```bash
# Navigate to CLI directory
cd tools/jan-cli

# Build
go build

# Run
./jan-cli --help # Linux/macOS
.\jan-cli.exe --help # Windows

# Optional: Copy to a location in your PATH
cp jan-cli ~/bin/ # Linux/macOS
copy jan-cli.exe %USERPROFILE%\bin\ # Windows
```

### Makefile Targets

```bash
make cli-build # Build the binary
make cli-install # Build and install to local bin
make cli-clean # Remove the binary
```

**cli-build** - Builds binary in `tools/jan-cli/`:

- Linux/macOS: `tools/jan-cli/jan-cli`
- Windows: `tools/jan-cli/jan-cli.exe`

**cli-install** - Builds and installs:

1. Calls `cli-build`
2. Creates bin directory if needed
3. Copies binary
4. Shows PATH instructions

**cli-clean** - Removes binary:

- Useful for clean rebuilds
- Frees disk space

---

## Commands Reference

### Command Hierarchy

```
jan-cli (root)
+-- config (configuration management)
| +-- validate - Validate configuration files
| +-- export - Export configuration
| +-- show - Display configuration values
| +-- generate - Generate schemas and defaults
| +-- k8s-values - Generate Kubernetes Helm values from config
+-- service (service operations)
| +-- list - List all services
| +-- logs - Show service logs
| +-- status - Check service status
+-- dev (development tools)
| +-- setup - Initialize development environment
| +-- scaffold - Generate new service from template
| +-- run - Run a service natively (dev-full / hybrid mode)
+-- swagger (API documentation)
| +-- generate - Generate OpenAPI documentation
+-- api-test (integration testing)
| +-- run - Run one or more Postman collections
+-- request (lightweight API requests)
| +-- run - Run a request defined in a JSON file
+-- monitor (observability stack)
| +-- up / dev / down - Start/stop the monitoring stack
| +-- test / status / query - Validate, inspect, and query
+-- install - Install the jan-cli binary to your PATH
+-- completion (shell completions)
 +-- bash
 +-- zsh
 +-- fish
 +-- powershell
```

### Global Flags

Available on all commands:

- `-v, --verbose` - Enable verbose output
- `--config-dir <path>` - Configuration directory (default: "config")
- `-h, --help` - Show help
- `--version` - Show version

---

## Configuration Management

The `config` subcommand manages Jan Server configuration files.

### config validate

Validate configuration files against schema:

```bash
# Validate with default environment
jan-cli config validate

# Validate specific environment
jan-cli config validate --env production
jan-cli config validate --env development

# Verbose validation
jan-cli config validate --verbose
```

**Output:**

- Configuration valid, or
- Validation errors with details

### config export

Export configuration in various formats:

```bash
# Export as environment variables
jan-cli config export --format env

# Export as Docker env file
jan-cli config export --format docker-env --output.env

# Export as JSON
jan-cli config export --format json --output config.json

# Export as YAML
jan-cli config export --format yaml --output config.yaml

# Export for specific environment
jan-cli config export --env production --format env
```

**Formats:**

- `env` - Shell environment variables (`KEY=value`)
- `docker-env` - Docker Compose env file
- `json` - JSON format
- `yaml` - YAML format

**Flags:**

- `--format <format>` - Output format (required)
- `--output <file>` - Output file (default: stdout)
- `--env <environment>` - Environment to export

### config show

Display configuration values with path navigation:

```bash
# Show all configuration
jan-cli config show

# Show specific service
jan-cli config show llm-api
jan-cli config show media-api

# Show as JSON
jan-cli config show llm-api --format json

# Show with specific environment
jan-cli config show llm-api --env production
```

**Flags:**

- `<service>` - Service name (optional)
- `--format <format>` - Output format (yaml, json)
- `--env <environment>` - Environment

### config generate

Generate JSON schemas and defaults.yaml:

```bash
# Generate all schemas
jan-cli config generate

# Generates:
# - config/schema/config.schema.json
# - config/schema/inference.schema.json
# - config/schema/infrastructure.schema.json
# - config/schema/monitoring.schema.json
# - config/schema/services.schema.json
# - config/defaults.yaml
```

---

## Service Operations

The `service` subcommand manages Jan Server services.

### service list

List all available services:

```bash
jan-cli service list
```

**Output:**

```
Available services:
 llm-api:8080 LLM API - OpenAI-compatible chat completions
 media-api:8285 Media API - File upload and management
 response-api:8082 Response API - Multi-step orchestration
 mcp-tools:8091 MCP Tools - Model Context Protocol tools
```

### service logs

Show Docker Compose logs for a specific service:

```bash
# View logs for a service
jan-cli service logs llm-api

# Follow logs
jan-cli service logs llm-api --follow

# Show last N lines
jan-cli service logs llm-api --tail 50
```

`jan-cli service logs` wraps `docker compose logs`, so it works on every platform where Docker Desktop is installed.

### service status

Check container status (and optionally health endpoints):

```bash
# Check all services via Makefile health check
jan-cli service status

# Check specific service
jan-cli service status llm-api
```

- `jan-cli service status` without arguments runs `make health-check`
- With a service argument it shows `docker compose ps <service>` and invokes the service-specific `/healthz` endpoint (PowerShell `Invoke-WebRequest` on Windows or `curl` on macOS/Linux)

---

## Development Tools

The `dev` subcommand provides development utilities.

### dev setup

Initialize development environment:

```bash
jan-cli dev setup
```

**What it does:**

1. Creates required directories (logs/, tmp/, uploads/)
2. Creates Docker networks (jan-network, jan-dev)
3. Generates.env file from templates
4. Optional: Sets up Docker environment

**Features:**

- Cross-platform (Windows, Linux, macOS)
- Docker optional (warns if not available)
- Idempotent (safe to run multiple times)

### dev scaffold

Generate a new service from `services/template-api`:

```bash
# Create new API service
jan-cli dev scaffold my-service

# Specify template/port (future templates can be added later)
jan-cli dev scaffold worker-service --template api --port 8999
```

What it does today:

- Copies `services/template-api` to `services/<name>`
- Replaces placeholders (module import paths, README text, comments)
- Prints next steps (run `go mod tidy`, update docker-compose, add Kong routes)

If the destination already exists the command aborts without touching files.

---

## Swagger Documentation

The `swagger` subcommand generates OpenAPI documentation.

### swagger generate

Generate OpenAPI/Swagger documentation for services:

```bash
# Generate for specific service
jan-cli swagger generate --service llm-api
jan-cli swagger generate --service media-api

# Generates:
# - services/llm-api/docs/swagger.yaml
# - services/llm-api/docs/swagger.json
```

**Requirements:**

- Service must have Swagger annotations in code
- `swag` CLI tool must be installed (`go install github.com/swaggo/swag/cmd/swag@latest`)

---

## API Testing

The `api-test` subcommand runs Postman-style collections (this is what the `make test-*` targets call under the hood).

### api-test run

```bash
# Run a single collection
jan-cli api-test run tests/e2e/automation/collections/auth.postman.json

# Run multiple collections with environment variables
jan-cli api-test run \
  tests/e2e/automation/collections/mcp-runtime.postman.json \
  tests/e2e/automation/collections/mcp-admin.postman.json \
  --env-var "kong_url=http://localhost:8000" \
  --verbose
```

**Common flags:**

- `--env-var "key=value"` - Set a collection variable (repeatable)
- `--env-file <file>` - Load variables from a file
- `--auto-auth <role>` - Acquire a token automatically (e.g. `admin`)
- `--timeout-request <ms>` - Per-request timeout
- `--reporters cli` - Reporter selection
- `--verbose`, `--debug`, `--bail` - Output and failure controls

## Lightweight Requests

The `request` subcommand runs a single API call described in a small JSON file - a simpler alternative to full collections for quick checks.

### request run

```bash
jan-cli request run tests/requests/response-api-test.json
jan-cli request run tests/requests/response-api-test.json --base-url http://localhost:8000
jan-cli request run tests/requests/response-api-test.json --debug
```

## Monitoring Stack

The `monitor` subcommand manages the observability stack (Prometheus, Grafana, Jaeger, OTEL Collector).

```bash
jan-cli monitor up        # Start the monitoring stack
jan-cli monitor dev       # Start with full trace sampling for development
jan-cli monitor test      # Validate that all monitoring services are healthy
jan-cli monitor status    # Show status and resource usage
jan-cli monitor query     # Interactive Prometheus/Jaeger queries
jan-cli monitor down      # Stop the monitoring stack
```

## Install

The `install` subcommand copies the built `jan-cli` binary to a directory on your PATH.

```bash
jan-cli install                 # Install to your user bin directory
jan-cli install --global        # System-wide install (requires admin)
jan-cli install --path /custom/bin
```

## Kubernetes Helm Values

`config k8s-values` renders a Helm values file from the canonical configuration.

```bash
jan-cli config k8s-values
```

---

## Troubleshooting

### "jan-cli: command not found" (Linux/macOS)

**Problem:** The bin directory is not in your PATH.

**Solution:**

1. Check if `~/bin` exists:

```bash
ls ~/bin/jan-cli
```

2. Add to PATH:

```bash
export PATH="$PATH:$HOME/bin"
```

3. Make permanent by adding to `~/.bashrc` or `~/.zshrc`:

```bash
echo 'export PATH="$PATH:$HOME/bin"' >> ~/.bashrc
source ~/.bashrc
```

### "jan-cli is not recognized" (Windows)

**Problem:** The bin directory is not in your PATH.

**Solution:**

1. Check if file exists:

```powershell
Test-Path $env:USERPROFILE\bin\jan-cli.exe
```

2. Add to PATH (temporary):

```powershell
$env:PATH += ";$env:USERPROFILE\bin"
```

3. Make permanent:

```powershell
notepad $PROFILE
# Add this line:
$env:PATH += ";$env:USERPROFILE\bin"
```

4. Restart PowerShell

### "Permission denied" (Linux/macOS)

**Problem:** The binary is not executable.

**Solution:**

```bash
chmod +x ~/bin/jan-cli
```

The `make cli-install` target handles this automatically, but if you installed manually, you may need to set execute permissions.

### Binary Not Updated After Code Changes

**Problem:** Installed binary is outdated after modifying source code.

**Solution:**

```bash
# Rebuild and reinstall
make cli-install

# Or clean and rebuild
make cli-clean
make cli-install
```

### Wrapper Scripts Don't Work

**Problem:** Wrapper script shows errors or doesn't build.

**Solution:**

1. Ensure Go is installed:

```bash
go version
```

2. Ensure in project root:

```bash
pwd # Should show jan-server directory
```

3. Check script is executable (Linux/macOS):

```bash
chmod +x tools/jan-cli.sh
```

4. Try manual build:

```bash
cd tools/jan-cli && go build
```

---

## Shell Completion

Jan CLI supports shell completion for bash, zsh, fish, and PowerShell.

### Generate Completion Script

```bash
# Bash
jan-cli completion bash > /etc/bash_completion.d/jan-cli

# Zsh
jan-cli completion zsh > "${fpath[1]}/_jan-cli"

# Fish
jan-cli completion fish > ~/.config/fish/completions/jan-cli.fish

# PowerShell
jan-cli completion powershell > jan-cli.ps1
# Then source it in your profile
```

### Enable Completion

**Bash:**

```bash
# Add to ~/.bashrc
source /etc/bash_completion.d/jan-cli
```

**Zsh:**

```zsh
# Add to ~/.zshrc
autoload -U compinit
compinit
```

**Fish:**

```fish
# Completion is auto-loaded from ~/.config/fish/completions/
```

**PowerShell:**

```powershell
# Add to $PROFILE
. /path/to/jan-cli.ps1
```

---

## Technical Details

### Framework: Cobra

Jan CLI uses [spf13/cobra](https://github.com/spf13/cobra) v1.8.1, the industry-standard CLI framework.

**Why Cobra:**

- Used by kubectl, docker, gh, helm
- Auto-generated help text
- Built-in completion generation
- Nested subcommand support
- Flag parsing and validation
- POSIX-compliant

**Dependencies:**

```go
require (
 github.com/spf13/cobra v1.8.1
 gopkg.in/yaml.v3 v3.0.1
)
```

### Project Structure

```
tools/jan-cli/
+-- main.go # Root command and initialization
+-- cmd_config.go # Configuration management
+-- cmd_service.go # Service operations
+-- cmd_dev.go # Development tools
+-- cmd_setup.go # Interactive setup wizard
+-- cmd_swagger.go # Swagger generation
+-- utils.go # Utility functions
+-- go.mod # Go module dependencies
+-- README.md # CLI documentation
```

### Build Details

**Build Command:**

```bash
cd tools/jan-cli
go build -o jan-cli
```

**Cross-Platform Builds:**

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o jan-cli-linux

# macOS
GOOS=darwin GOARCH=amd64 go build -o jan-cli-darwin

# Windows
GOOS=windows GOARCH=amd64 go build -o jan-cli.exe
```

**Binary Size:** ~10MB (includes dependencies)

### Wrapper Scripts

**PowerShell (jan-cli.ps1):**

- Auto-builds if binary missing or outdated
- Checks all `*.go` files for changes
- Supports all jan-cli commands
- Works on Windows PowerShell 5.1+

**Bash (jan-cli.sh):**

- Auto-builds if binary missing or outdated
- Checks all `*.go` files for changes
- Supports all jan-cli commands
- Works on Linux/macOS with Bash 3.2+

---

## Examples

### Configuration Workflow

```bash
# Generate schemas and defaults
jan-cli config generate

# Validate configuration
jan-cli config validate --env production

# Export as environment variables
jan-cli config export --format env --env production >.env.production

# Show specific service config
jan-cli config show llm-api --format json
```

### Service Management

```bash
# List all services
jan-cli service list

# View logs
jan-cli service logs llm-api --follow

# Check health
jan-cli service status
```

### Development Setup

```bash
# Setup environment
jan-cli dev setup

# Create new service from template
jan-cli dev scaffold worker-service --template api

# Generate API documentation
jan-cli swagger generate --service llm-api
```

---

## Best Practices

### For Daily Use

1. Install globally with `make cli-install`
2. Add to PATH once
3. Use `jan-cli` from anywhere
4. Run `make cli-install` after pulling updates

### For Development

1. Use wrapper scripts (`./tools/jan-cli.sh` or `.\tools\jan-cli.ps1`)
2. Always uses latest code
3. Auto-builds if needed
4. Good for testing changes

### For CI/CD

1. Use wrapper scripts (no installation needed)
2. Or install and add to PATH
3. Verify with `jan-cli --version`
4. Run commands directly

---

## Summary

**Quick Reference:**

- **Build:** `make cli-build`
- **Install:** `make cli-install`
- **Clean:** `make cli-clean`
- **Use:** `jan-cli <command>`

**Recommended Workflow:**

1. Run `make cli-install` once
2. Add to PATH as instructed
3. Use `jan-cli` from anywhere
4. Run `make cli-install` again after updates

**Key Commands:**

- `jan-cli config validate` - Validate configuration
- `jan-cli config generate` - Generate schemas
- `jan-cli service list` - List services
- `jan-cli dev setup` - Setup environment
- `jan-cli swagger generate --service <name>` - Generate API docs

---

## Related Documentation

- [Testing Guide](testing.md) - Cross-platform testing procedures
- [Configuration System](../configuration/README.md) - Configuration management
- [Development Guide](development.md) - Local development setup
- [Architecture Overview](../architecture/README.md) - System design

---

**Cross-Platform:** Windows, Linux, macOS
