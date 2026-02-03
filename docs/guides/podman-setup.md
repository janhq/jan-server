# Podman Setup for Jan Server

This guide explains how to run Jan Server using Podman instead of Docker.

## Requirements

- **Podman** 4.0+ (tested with 5.7.1)
- **podman-compose** 1.0.6+ (tested with 1.5.0)

### Arch Linux Installation

```bash
sudo pacman -S podman podman-compose
```

### Verify Installation

```bash
podman --version      # Should be 4.0+
podman-compose --version  # Should be 1.0.6+
```

## Quick Start

The Makefile automatically detects Podman when Docker is not available:

```bash
# Auto-detect (uses Podman if Docker not found)
make up-full

# Explicitly use Podman
make CONTAINER_ENGINE=podman up-full

# Check which engine is being used
make engine-info
```

## Setup Steps

### 1. Initial Setup

```bash
# Setup Podman networks
make CONTAINER_ENGINE=podman podman-setup

# Generate .env file
make setup
```

### 2. Start Services

```bash
# Start all services
make CONTAINER_ENGINE=podman up-full

# Or start specific profiles
make CONTAINER_ENGINE=podman up-infra
make CONTAINER_ENGINE=podman up-api
make CONTAINER_ENGINE=podman up-mcp
```

### 3. Verify Services

```bash
make health-check
```

## Shell Configuration (Optional)

To always use Podman without specifying `CONTAINER_ENGINE`:

```bash
# Add to ~/.bashrc or ~/.zshrc
export CONTAINER_ENGINE=podman

# Or create an alias
alias docker=podman
```

## Podman-Specific Commands

```bash
# List Jan Server containers
make podman-ps

# Clean up all resources
make podman-clean

# Show engine info
make engine-info
```

## Rootless Mode

Podman runs in rootless mode by default on most Linux distributions. This is the recommended configuration for development.

To verify rootless mode:

```bash
podman info --format '{{.Host.Security.Rootless}}'
# Should output: true
```

## Networking

### Container-to-Container Communication

Containers communicate via Podman networks. The Makefile automatically creates:

- `jan-server_default` - Main network for services
- `jan-server_mcp-network` - Network for MCP tools

### Host Access from Containers

In Podman, use `host.containers.internal` instead of `host.docker.internal`:

```yaml
# In docker-compose.yml or environment variables
SOME_HOST_URL: http://host.containers.internal:8080
```

## Troubleshooting

### Permission Denied Errors

For rootless Podman, ensure your user has subuids/subgids configured:

```bash
# Check configuration
cat /etc/subuid | grep $USER
cat /etc/subgid | grep $USER

# If missing, add them
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER
```

### Network Issues

```bash
# Reset networks
podman network rm jan-server_default jan-server_mcp-network
make podman-network
```

### Storage Issues

```bash
# Reset Podman storage (WARNING: removes all containers/images)
podman system reset
```

### Slow Startup

Podman's first run may be slow as it sets up user namespaces. Subsequent runs should be faster.

### cgroups v2 Issues

Modern distributions use cgroups v2. Verify:

```bash
# Check cgroups version
cat /sys/fs/cgroup/cgroup.controllers

# If this file exists, you're using cgroups v2 (good)
```

## Known Limitations

1. **BuildKit**: Podman doesn't use Docker's BuildKit. Multi-stage builds still work but some advanced features may differ.

2. **Docker Socket**: Some tools expect `/var/run/docker.sock`. Enable Podman socket for compatibility:
   ```bash
   systemctl --user enable --now podman.socket
   export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
   ```

3. **Compose File Compatibility**: podman-compose 1.5.0 supports most Docker Compose v3 features including:
   - Profiles
   - Include directives
   - Named volumes
   - Networks
   - Health checks

## Performance Tips

1. **Use native overlay storage**:
   ```bash
   # Check storage driver
   podman info --format '{{.Store.GraphDriverName}}'
   # Should be 'overlay' for best performance
   ```

2. **Enable parallel pulls**:
   ```bash
   # In ~/.config/containers/registries.conf
   [engine]
   num_locks = 2048
   ```

## Comparison: Docker vs Podman

| Feature | Docker | Podman |
|---------|--------|--------|
| Daemon | Required | Daemonless |
| Root by default | Yes | No (rootless) |
| Compose command | `docker compose` | `podman-compose` |
| Socket | `/var/run/docker.sock` | `$XDG_RUNTIME_DIR/podman/podman.sock` |
| Host access | `host.docker.internal` | `host.containers.internal` |

## See Also

- [Development Guide](development.md)
- [Main README](../../README.md)
- [Podman Documentation](https://podman.io/docs)
