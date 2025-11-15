# jan-cli - Jan Server Command-Line Interface

The official CLI tool for Jan Server, providing unified access to configuration management, service operations, and development tools.

## Quick Start

### Using Wrapper Scripts (Recommended)

The easiest way to use jan-cli from the project root:

```bash
# Linux/macOS
./jan-cli.sh --help
./jan-cli.sh config validate
./jan-cli.sh service list

# Windows PowerShell
.\jan-cli.ps1 --help
.\jan-cli.ps1 config validate
.\jan-cli.ps1 service list
```

The wrapper scripts automatically build jan-cli if needed and run it with your arguments.

## Installation

### Using Wrapper Scripts

No installation needed! Just use the wrapper scripts from the project root:

- **`jan-cli.sh`** - For Linux/macOS/WSL
- **`jan-cli.ps1`** - For Windows PowerShell

The scripts will automatically:
1. Check if jan-cli binary exists
2. Build it if missing or outdated
3. Run your command

### From Source

```bash
cd cmd/jan-cli
go build -o jan-cli
# Move to PATH (optional)
sudo mv jan-cli /usr/local/bin/  # Linux/macOS
# or for Windows, add to PATH
```

### Using Go Install

```bash
go install github.com/janhq/jan-server/cmd/jan-cli@latest
```

### Installation

**Option 1: Install Globally (Recommended)**

Use the Makefile target to build and install `jan-cli` to your local bin directory:

```bash
# From project root
make cli-install
```

This will:
- Build the `jan-cli` binary
- Install it to `~/bin` (Linux/macOS) or `%USERPROFILE%\bin` (Windows)
- Display instructions for adding to PATH if needed

After installation and adding to PATH, you can run `jan-cli` from anywhere:

```bash
jan-cli --version
jan-cli config validate
jan-cli service list
```

**Option 2: Use Wrapper Scripts**

Run from the project root using wrapper scripts (no installation needed):

```bash
# Linux/macOS
./jan-cli.sh --help
./jan-cli.sh config validate

# Windows PowerShell
.\jan-cli.ps1 --help
.\jan-cli.ps1 config validate
```

The wrapper scripts automatically build the CLI if needed.

**Option 3: Build and Run Manually**

```bash
# Build
cd cmd/jan-cli
go build

# Run
./jan-cli --help  # Linux/macOS
.\jan-cli.exe --help  # Windows
```

### Adding to PATH (Optional)

For easier access, you can add the built binary to your PATH or create an alias:

```bash
# Linux/macOS - Add to ~/.bashrc or ~/.zshrc
alias jan-cli='/path/to/jan-server/cmd/jan-cli/jan-cli'

# Or use the wrapper from anywhere
alias jan-cli='/path/to/jan-server/jan-cli.sh'

# Windows PowerShell - Add to $PROFILE
function jan-cli { & 'C:\path\to\jan-server\jan-cli.ps1' $args }
```

## Commands Overview

### Configuration Management (`config`)

Manage Jan Server configuration files.

```bash
# Validate configuration
jan-cli config validate
jan-cli config validate --env production

# Export configuration
jan-cli config export --format env > .env
jan-cli config export --format json
jan-cli config export --format docker-env --output docker.env

# Show configuration
jan-cli config show llm-api
jan-cli config show --path services.llm-api.database
jan-cli config show --format json

# Generate Kubernetes values
jan-cli config k8s-values --env production > k8s/values-prod.yaml
jan-cli config k8s-values --env development --output k8s/values-dev.yaml
```

### Service Operations (`service`)

Manage and inspect Jan Server services.

```bash
# List all services
jan-cli service list

# Show service logs
jan-cli service logs llm-api
jan-cli service logs llm-api --tail 50 --follow

# Check service status
jan-cli service status
jan-cli service status llm-api
```

### Development Tools (`dev`)

Development utilities for Jan Server.

```bash
# Setup development environment
jan-cli dev setup

# Scaffold new service
jan-cli dev scaffold my-service
jan-cli dev scaffold worker-service --template worker --port 8999
```

## Configuration Commands

### `config validate`

Validate configuration files for syntax errors and required fields.

**Usage:**
```bash
jan-cli config validate [flags]
```

**Flags:**
- `-f, --file string` - Config file to validate (default: `config/defaults.yaml`)
- `--schema string` - Schema file to validate against
- `-e, --env string` - Environment to validate (development, production, etc.)

**Examples:**
```bash
# Validate default configuration
jan-cli config validate

# Validate production configuration
jan-cli config validate --env production

# Validate specific file with schema
jan-cli config validate --file custom-config.yaml --schema config-schema.json
```

### `config export`

Export configuration in various formats.

**Usage:**
```bash
jan-cli config export [flags]
```

**Flags:**
- `-f, --file string` - Config file to export (default: `config/defaults.yaml`)
- `--format string` - Output format: `env`, `docker-env`, `json`, `yaml` (default: `env`)
- `--prefix string` - Add prefix to exported variables
- `-o, --output string` - Output file (default: stdout)

**Examples:**
```bash
# Export as shell environment variables
eval $(jan-cli config export)

# Export as docker-compose .env file
jan-cli config export --format docker-env --output .env

# Export as JSON
jan-cli config export --format json > config.json

# Export with prefix
jan-cli config export --prefix MYAPP --format env
```

### `config show`

Display configuration values.

**Usage:**
```bash
jan-cli config show [service] [flags]
```

**Flags:**
- `-f, --file string` - Config file to read (default: `config/defaults.yaml`)
- `--path string` - Config path to show (e.g., `services.llm-api`)
- `--format string` - Output format: `yaml`, `json`, `value` (default: `yaml`)

**Examples:**
```bash
# Show entire configuration
jan-cli config show

# Show specific service config
jan-cli config show llm-api

# Show specific path
jan-cli config show --path services.llm-api.database

# Show as JSON
jan-cli config show llm-api --format json

# Show single value
jan-cli config show --path services.llm-api.http.port --format value
```

### `config k8s-values`

Generate Kubernetes Helm values file from configuration.

**Usage:**
```bash
jan-cli config k8s-values [flags]
```

**Flags:**
- `-e, --env string` - Environment (development, production, etc.) (default: `development`)
- `-o, --output string` - Output file (default: stdout)
- `--set stringSlice` - Override values (key=value)

**Examples:**
```bash
# Generate development values
jan-cli config k8s-values --env development > k8s/values-dev.yaml

# Generate production values
jan-cli config k8s-values --env production > k8s/values-prod.yaml

# Generate with overrides
jan-cli config k8s-values --env production \
  --set services.llm-api.replicas=3 \
  --set services.llm-api.resources.limits.memory=2Gi \
  --output k8s/values-prod-scaled.yaml
```

## Service Commands

### `service list`

List all available Jan Server services.

**Usage:**
```bash
jan-cli service list
```

**Example Output:**
```
Available services:
  llm-api         :8080  LLM API - OpenAI-compatible chat completions
  media-api       :8285  Media API - File upload and management
  response-api    :8082  Response API - Multi-step orchestration
  mcp-tools       :8091  MCP Tools - Model Context Protocol tools
```

### `service logs`

Show logs for a specific service.

**Usage:**
```bash
jan-cli service logs [service] [flags]
```

**Flags:**
- `-n, --tail int` - Number of lines to show (default: 100)
- `-f, --follow` - Follow log output

**Examples:**
```bash
# Show last 100 lines
jan-cli service logs llm-api

# Show last 50 lines
jan-cli service logs llm-api --tail 50

# Follow logs in real-time
jan-cli service logs llm-api --follow
```

### `service status`

Show service status and health information.

**Usage:**
```bash
jan-cli service status [service]
```

**Examples:**
```bash
# Show status for all services
jan-cli service status

# Show status for specific service
jan-cli service status llm-api
```

## Development Commands

### `dev setup`

Initialize development environment.

**Usage:**
```bash
jan-cli dev setup
```

This command will:
- Check for required dependencies (Docker, Go)
- Create `.env` file from template
- Pull required Docker images
- Set up development directories

### `dev scaffold`

Generate a new service from template.

**Usage:**
```bash
jan-cli dev scaffold [service-name] [flags]
```

**Flags:**
- `-t, --template string` - Service template: `api`, `worker` (default: `api`)
- `-p, --port string` - Service port

**Examples:**
```bash
# Scaffold API service
jan-cli dev scaffold my-service

# Scaffold with specific port
jan-cli dev scaffold my-service --port 8999

# Scaffold worker service
jan-cli dev scaffold my-worker --template worker
```

## Global Flags

Available for all commands:

- `-v, --verbose` - Enable verbose output
- `--config-dir string` - Configuration directory (default: `config`)
- `-h, --help` - Show help for any command
- `--version` - Show version information

## Shell Completion

Generate shell completion scripts for better command-line experience.

### Bash

```bash
jan-cli completion bash > /etc/bash_completion.d/jan-cli
```

### Zsh

```bash
jan-cli completion zsh > "${fpath[1]}/_jan-cli"
```

### Fish

```bash
jan-cli completion fish > ~/.config/fish/completions/jan-cli.fish
```

### PowerShell

```powershell
jan-cli completion powershell | Out-String | Invoke-Expression
```

## Examples

### Typical Development Workflow

```bash
# 1. Setup development environment
jan-cli dev setup

# 2. Validate configuration
jan-cli config validate

# 3. Export configuration for Docker Compose
jan-cli config export --format docker-env --output .env

# 4. Start services (using make or docker compose)
make up-full

# 5. Check service status
jan-cli service status

# 6. View logs
jan-cli service logs llm-api --follow
```

### Configuration Management

```bash
# Validate all environments
jan-cli config validate
jan-cli config validate --env production
jan-cli config validate --env staging

# Export for different targets
jan-cli config export --format env > .env
jan-cli config export --format json > config.json
jan-cli config k8s-values --env production > k8s/values-prod.yaml

# Inspect configuration
jan-cli config show llm-api
jan-cli config show --path services.llm-api.database --format json
```

### Service Operations

```bash
# Quick service overview
jan-cli service list
jan-cli service status

# Debug specific service
jan-cli service logs llm-api --tail 100
jan-cli service logs llm-api --follow
jan-cli service status llm-api
```

## Integration with Make

You can integrate jan-cli with your Makefile:

```makefile
.PHONY: config-validate
config-validate:
	jan-cli config validate

.PHONY: config-export
config-export:
	jan-cli config export --format docker-env --output .env

.PHONY: k8s-values
k8s-values:
	jan-cli config k8s-values --env production > k8s/values-prod.yaml
```

## Troubleshooting

### Command Not Found

Ensure jan-cli is in your PATH:

```bash
# Check if jan-cli is installed
which jan-cli

# If not, add to PATH or use full path
export PATH=$PATH:/path/to/jan-cli
```

### Configuration Validation Errors

```bash
# Verbose output for debugging
jan-cli -v config validate

# Check specific file
jan-cli config validate --file config/defaults.yaml
```

### Permission Denied

```bash
# Make executable (Linux/macOS)
chmod +x jan-cli

# Or run with sudo if accessing protected files
sudo jan-cli config export --output /etc/jan/config.env
```

## Contributing

See [../../CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines on contributing to jan-cli.

## License

See [../../LICENSE](../../LICENSE) for license information.
