# Testing Guide

How to test Jan Server on Windows, Linux, and macOS.

---

## Quick Start

### Run All Tests

```bash
make test-all
```

### Specific Test Suites

```bash
make test-auth           # Authentication tests
make test-conversations  # Conversation tests
make test-response       # Response API tests
make test-media          # Media API tests
make test-mcp            # MCP Tools tests
```

### Unit Tests

```bash
go test ./services/llm-api/...
go test ./services/response-api/...
```

---

## Integration Testing

### jan-cli api-test Collections

Test collections are located in `tests/e2e/automation/collections/` and can be run via:

```bash
make test-all            # Run all integration tests
make test-auth           # Auth tests (guest-login, API keys, token refresh)
make test-conversation  # Conversation CRUD, messages
make test-response       # Tool orchestration
make test-media          # File uploads
make test-mcp            # MCP tool execution
```

---

## Platform-Specific Notes

### Linux (Ubuntu/Debian)

Full Docker support. All tests pass.

### macOS

Requires Docker Desktop or Colima:

```bash
# Install Colima
brew install docker colima

# Start with resources
colima start --cpu 4 --memory 8 --disk 100
```

### Windows

Windows CI focuses on CLI and build verification. Full Docker tests run on Ubuntu.

---

## Troubleshooting

### Permission Denied on jan-cli.sh

```bash
chmod +x jan-cli.sh
```

### Docker Commands Fail

```bash
# Check Docker is running
docker ps

# Linux: Add user to docker group
sudo usermod -aG docker $USER
```

### Build Failures

```bash
make clean-build
make build-llm-api
go mod download
go mod verify
```

---

## Related Documentation

- [Jan CLI Guide](jan-cli.md) - CLI command reference
- [Configuration System](../configuration/README.md)
- [Development Guide](development.md)