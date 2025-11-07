# Contributing to Jan Server

Thank you for your interest in contributing to Jan Server! This document provides guidelines and information for contributors.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Code Conventions](#code-conventions)
- [Pull Request Process](#pull-request-process)
- [Testing Requirements](#testing-requirements)

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker Desktop (Windows/macOS) or Docker Engine + Docker Compose (Linux)
- Git
- Make

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/jan-server.git
   cd jan-server
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/janhq/jan-server.git
   ```

## Development Setup

### Quick Setup

```bash
# Setup development environment
make setup

# Start services in hybrid mode (recommended)
make hybrid-dev

# Or start full Docker stack
make up-full
```

See [Development Guide](docs/guides/development.md) for detailed setup.

### Hybrid Development Mode

For the best development experience, use hybrid mode:

```bash
# Setup hybrid environment
make hybrid-dev

# Run service natively with hot reload
# (in separate terminal)
cd services/llm-api
air  # or go run .
```

Benefits:
- Faster iteration
- Better debugging
- Native IDE integration
- Hot reload

See [Hybrid Mode Guide](docs/guides/hybrid-mode.md) for details.

## Code Conventions

We follow strict code conventions documented in [docs/conventions/](docs/conventions/):

### Go Code

- **Package Structure**: Follow Clean Architecture principles
- **Naming**: Use Go conventions (camelCase for private, PascalCase for public)
- **Error Handling**: Always handle errors explicitly
- **Comments**: Document all exported functions and types
- **Formatting**: Use `gofmt` and `goimports`

Example:
```go
// ProcessRequest handles incoming chat completion requests
func (s *Service) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Process...
    return response, nil
}
```

### Architecture

- **Clean Architecture**: Domain → Use Cases → Interfaces → Infrastructure
- **Dependency Injection**: Use interfaces and constructor injection
- **Repository Pattern**: Separate data access logic
- **Service Pattern**: Business logic in services

See [Architecture Conventions](docs/conventions/architecture.md)

### API Design

- **RESTful**: Follow REST principles
- **OpenAPI**: Document all endpoints in Swagger
- **Versioning**: Use `/v1/` prefix
- **Error Responses**: Use consistent error format

See [API Conventions](docs/architecture/README.md#api-conventions)

## Pull Request Process

### Before Submitting

1. **Create a branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**:
   - Write clean, well-documented code
   - Follow code conventions
   - Add tests for new functionality

3. **Run tests**:
   ```bash
   make test-all
   ```

4. **Update documentation**:
   - Update relevant docs in `docs/`
   - Update API documentation (Swagger comments)
   - Add examples if needed

5. **Commit with clear messages**:
   ```bash
   git commit -m "feat: add new MCP tool for X"
   git commit -m "fix: resolve issue with Y"
   git commit -m "docs: update API examples"
   ```

### Commit Message Format

Use conventional commits:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Example:
```
feat(mcp): add calculator tool

- Implement basic arithmetic operations
- Add input validation
- Include unit tests
- Update MCP tools documentation
```

### Submitting PR

1. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create Pull Request on GitHub

3. Fill out PR template:
   - Description of changes
   - Related issues
   - Testing performed
   - Screenshots (if UI changes)

4. Wait for review:
   - Address review comments
   - Update PR as needed
   - Maintain clean commit history

### Review Process

- At least one maintainer approval required
- All tests must pass
- Code coverage should not decrease
- Documentation must be updated

## Testing Requirements

### Unit Tests

All new code must include unit tests:

```go
func TestProcessRequest(t *testing.T) {
    // Arrange
    service := NewService()
    req := &Request{...}
    
    // Act
    resp, err := service.ProcessRequest(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

### Integration Tests

For API endpoints, add Postman/Newman tests:

```json
{
  "name": "Test New Endpoint",
  "request": {
    "method": "POST",
    "url": "{{base_url}}/v1/new-endpoint",
    ...
  },
  "event": [{
    "listen": "test",
    "script": {
      "exec": [
        "pm.test('Status is 200', function() {",
        "    pm.response.to.have.status(200);",
        "});"
      ]
    }
  }]
}
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests
make test-all

# Coverage report
make test-coverage

# Specific test suite
make test-auth
make test-conversations
make test-mcp
```

### Test Coverage

- Aim for >80% coverage
- Critical paths must have 100% coverage
- Include edge cases and error scenarios

## Code Review Guidelines

When reviewing PRs:

- **Functionality**: Does it work as intended?
- **Tests**: Are there adequate tests?
- **Code Quality**: Follows conventions?
- **Documentation**: Is it documented?
- **Performance**: Any performance implications?
- **Security**: Any security concerns?

## Documentation

### When to Update Docs

- New features → Add to guides and API reference
- API changes → Update Swagger comments
- Architecture changes → Update architecture docs
- Breaking changes → Update migration guide

### Documentation Structure

```
docs/
├── getting-started/    # User onboarding
├── guides/            # How-to guides
├── api/               # API reference
├── architecture/      # System design
└── conventions/       # Code standards
```

## Questions?

- Check [Documentation](docs/README.md)
- Ask in GitHub Discussions
- Contact maintainers

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to Jan Server! 🎉
