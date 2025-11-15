# Jan CLI - Complete Guide

**Date:** November 15, 2025  
**Status:** Production Ready ✅  
**Version:** 1.0.0

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
11. [Migration from jan-config](#migration-from-jan-config)

---

## Overview

Jan CLI is the official command-line interface for Jan Server, providing unified access to:

- **Configuration Management** - Validate, export, and inspect configuration
- **Service Operations** - List services, view logs, check status
- **Development Tools** - Setup environment, scaffold services
- **Shell Completion** - Auto-completion for all major shells

Built with [Cobra framework](https://github.com/spf13/cobra), the industry standard used by kubectl, docker, and github CLI.

### Key Features

- ✅ **Unified Interface** - Single command for all Jan Server operations
- ✅ **Professional Structure** - Industry-standard Cobra framework
- ✅ **Extensible** - Easy to add new commands
- ✅ **Well-Documented** - Comprehensive help and examples
- ✅ **Cross-Platform** - Works on Windows, Linux, macOS
- ✅ **Shell Completion** - Bash, Zsh, Fish, PowerShell support

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
source ~/.bashrc  # or source ~/.zshrc
```

### Verify Installation

```bash
jan-cli --version
# Output: jan-cli version 1.0.0

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

Run directly from project root using wrapper scripts:

```bash
# Linux/macOS
./jan-cli.sh config validate
./jan-cli.sh service list

# Windows PowerShell
.\jan-cli.ps1 config validate
.\jan-cli.ps1 service list
```

**Advantages:**
- No installation needed
- Auto-builds if binary missing or outdated
- Always uses latest code
- Good for development

**Disadvantages:**
- Must be run from project root
- Requires file extension (.sh or .ps1)

### Method 3: Manual Build

```bash
# Navigate to CLI directory
cd cmd/jan-cli

# Build
go build

# Run
./jan-cli --help  # Linux/macOS
.\jan-cli.exe --help  # Windows

# Optional: Copy to a location in your PATH
cp jan-cli ~/bin/  # Linux/macOS
copy jan-cli.exe %USERPROFILE%\bin\  # Windows
```

### Makefile Targets

```bash
make cli-build      # Build the binary
make cli-install    # Build and install to local bin
make cli-clean      # Remove the binary
```

**cli-build** - Builds binary in `cmd/jan-cli/`:
- Linux/macOS: `cmd/jan-cli/jan-cli`
- Windows: `cmd/jan-cli/jan-cli.exe`

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
├── config (configuration management)
│   ├── validate    - Validate configuration files
│   ├── export      - Export configuration
│   ├── show        - Display configuration values
│   └── k8s-values  - Generate Kubernetes Helm values
├── service (service operations)
│   ├── list        - List all services
│   ├── logs        - Show service logs
│   └── status      - Check service status
├── dev (development tools)
│   ├── setup       - Initialize development environment
│   └── scaffold    - Generate new service from template
└── completion (shell completions)
    ├── bash
    ├── zsh
    ├── fish
    └── powershell
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
- ✅ Configuration valid
- ❌ Validation errors with details

### config export

Export configuration in various formats:

```bash
# Export as environment variables
jan-cli config export --format env

# Export as Docker env file
jan-cli config export --format docker-env --output .env

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

### config k8s-values

Generate Kubernetes Helm values from configuration:

```bash
# Generate for development
jan-cli config k8s-values --env development

# Generate for production
jan-cli config k8s-values --env production

# Save to file
jan-cli config k8s-values --env production --output k8s/values-prod.yaml

# Override values
jan-cli config k8s-values --env production \
  --set services.llm-api.replicas=3 \
  --set services.llm-api.resources.limits.memory=4Gi
```

**Flags:**
- `--env <environment>` - Environment (required)
- `--output <file>` - Output file (default: stdout)
- `--set <key=value>` - Override values

**Use Case:**
```bash
# Generate and deploy
jan-cli config k8s-values --env production > k8s/values-prod.yaml
helm upgrade jan-server k8s/jan-server -f k8s/values-prod.yaml
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
  llm-api         :8080  LLM API - OpenAI-compatible chat completions
  media-api       :8285  Media API - File upload and management
  response-api    :8082  Response API - Multi-step orchestration
  mcp-tools       :8091  MCP Tools - Model Context Protocol tools
```

### service logs

Show service logs (placeholder for future implementation):

```bash
# View logs for a service
jan-cli service logs llm-api

# Follow logs
jan-cli service logs llm-api --follow

# Show last N lines
jan-cli service logs llm-api --tail 50
```

**Note:** Currently a placeholder. Will integrate with Docker/Kubernetes logs.

### service status

Check service health status (placeholder for future implementation):

```bash
# Check all services
jan-cli service status

# Check specific service
jan-cli service status llm-api
```

**Note:** Currently a placeholder. Will integrate with health check endpoints.

---

## Development Tools

The `dev` subcommand provides development utilities.

### dev setup

Initialize development environment (placeholder for future implementation):

```bash
jan-cli dev setup
```

**Planned functionality:**
- Install dependencies
- Setup local databases
- Configure environment variables
- Initialize development tools

### dev scaffold

Generate new service from template (placeholder for future implementation):

```bash
# Create new service
jan-cli dev scaffold my-service

# With options
jan-cli dev scaffold worker-service --template worker --port 8999
```

**Planned functionality:**
- Generate service structure
- Create boilerplate code
- Setup configuration
- Add to docker-compose

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
   pwd  # Should show jan-server directory
   ```

3. Check script is executable (Linux/macOS):
   ```bash
   chmod +x jan-cli.sh
   ```

4. Try manual build:
   ```bash
   cd cmd/jan-cli && go build
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
cmd/jan-cli/
├── main.go              # Root command and initialization
├── cmd_config.go        # Configuration management (344 lines)
├── cmd_service.go       # Service operations (80 lines)
├── cmd_dev.go           # Development tools (70 lines)
├── go.mod               # Go module dependencies
└── README.md            # CLI documentation
```

### Build Details

**Build Command:**
```bash
cd cmd/jan-cli
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

### Installation Scripts

**PowerShell Script:** `scripts/install-cli.ps1`
- Builds jan-cli binary
- Creates `%USERPROFILE%\bin` if needed
- Copies binary with proper path handling
- Checks if bin directory is in PATH
- Shows detailed PATH setup instructions

**Bash Script:** `scripts/install-cli.sh`
- Builds jan-cli binary
- Creates `~/bin` if needed
- Copies and sets execute permissions
- Checks if bin directory is in PATH
- Shows detailed PATH setup instructions

---

## Migration from jan-config

### Background

Jan CLI was refactored from `jan-config` to provide a more professional, extensible command-line interface.

### Changes Made

**Before (jan-config):**
```bash
jan-config validate
jan-config export
jan-config show llm-api
```

**After (jan-cli):**
```bash
jan-cli config validate
jan-cli config export
jan-cli config show llm-api
jan-cli service list
jan-cli dev setup
```

### Breaking Changes

**Direct CLI usage:**
- All `jan-config` commands moved under `jan-cli config` subcommand
- New subcommands added: `service`, `dev`
- Global flags syntax remains the same

**No breaking changes for:**
- Make targets (`make config-generate`, `make config-test`, etc.)
- CI/CD pipelines using Make
- Documented workflows

### Migration Steps

**Update scripts:**
```bash
# Replace in scripts/automation
- jan-config validate
+ jan-cli config validate

# Replace exports
- eval $(jan-config export)
+ eval $(jan-cli config export)
```

**Update CI/CD:**
```yaml
# Before
- name: Validate config
  run: jan-config validate

# After
- name: Validate config
  run: jan-cli config validate
```

**Update documentation:**
- Update command examples
- Update integration guides
- Update troubleshooting docs

### Features Preserved

All original `jan-config` functionality is preserved:

✅ Configuration file validation  
✅ Multiple export formats (env, docker-env, json, yaml)  
✅ Configuration display with path navigation  
✅ Kubernetes Helm values generation  

### New Features Added

🎉 Service operations (list, logs, status)  
🎉 Development tools (setup, scaffold)  
🎉 Auto-completion for all shells  
🎉 Professional help text  
🎉 Extensible architecture  

---

## FAQ

### Q: Do I need to install globally?

**A:** No, you can use wrapper scripts (`./jan-cli.sh` or `.\jan-cli.ps1`) from the project root without installation.

### Q: Can I install to a different location?

**A:** Yes, but you'll need to modify `scripts/install-cli.ps1` or `scripts/install-cli.sh` to change the installation path. The default is `~/bin` for Unix and `%USERPROFILE%\bin` for Windows.

### Q: Will this conflict with other tools?

**A:** The command is named `jan-cli` which is unlikely to conflict with existing tools. If you have a naming conflict, you can create an alias with a different name.

### Q: How do I update the CLI after code changes?

**A:** Run `make cli-install` again. It rebuilds and reinstalls the binary.

### Q: Can I use this in CI/CD?

**A:** Yes! Use wrapper scripts in CI/CD:
```yaml
- name: Validate config
  run: ./jan-cli.sh config validate
```

Or install globally:
```yaml
- name: Install CLI
  run: make cli-install && export PATH="$PATH:$HOME/bin"

- name: Validate
  run: jan-cli config validate
```

### Q: Why does `make cli-install` use PowerShell on Windows?

**A:** Make on Windows typically uses `sh` shell, which has limited Windows path handling. Using PowerShell ensures proper path handling with `%USERPROFILE%` and backslashes.

### Q: How do I uninstall?

**A:** Remove the binary:
```bash
# Linux/macOS
rm ~/bin/jan-cli

# Windows
Remove-Item $env:USERPROFILE\bin\jan-cli.exe
```

Then remove the PATH export from your shell profile.

---

## Upgrading

### Update to Latest Version

After pulling latest code:

```bash
# Reinstall
make cli-install
```

This rebuilds the binary with latest changes and reinstalls it.

### Check for Updates

```bash
# Pull latest code
git pull

# Check if rebuild needed
cd cmd/jan-cli
go mod tidy

# Reinstall
cd ../..
make cli-install
```

---

## Best Practices

### For Daily Use

1. Install globally with `make cli-install`
2. Add to PATH once
3. Use `jan-cli` from anywhere
4. Run `make cli-install` after pulling updates

### For Development

1. Use wrapper scripts (`./jan-cli.sh` or `.\jan-cli.ps1`)
2. Always uses latest code
3. Auto-builds if needed
4. Good for testing changes

### For CI/CD

1. Use wrapper scripts (no installation needed)
2. Or install and add to PATH
3. Verify with `jan-cli --version`
4. Run commands directly

---

## Examples

### Configuration Workflow

```bash
# Validate before deploying
jan-cli config validate --env production

# Export as environment variables
jan-cli config export --format env --env production > .env.production

# Show specific service config
jan-cli config show llm-api --format json

# Generate Kubernetes values
jan-cli config k8s-values --env production --output k8s/values-prod.yaml
```

### Service Management

```bash
# List all services
jan-cli service list

# View logs (future)
jan-cli service logs llm-api --follow

# Check health (future)
jan-cli service status
```

### Development Setup

```bash
# Setup environment (future)
jan-cli dev setup

# Create new service (future)
jan-cli dev scaffold worker-service --template worker
```

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

**Documentation:**
- **Full CLI Docs:** [cmd/jan-cli/README.md](../../cmd/jan-cli/README.md)
- **Configuration Guide:** [README.md](./README.md)
- **Development Guide:** [../guides/development.md](../guides/development.md)

---

**Status:** Production Ready  
**Version:** 1.0.0  
**Last Updated:** November 15, 2025

For issues or questions, see the [troubleshooting section](#troubleshooting) or check the main configuration documentation.
