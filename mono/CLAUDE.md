# CLAUDE.md - AI Assistant Guidelines for Jan Server Mono

> Instructions for AI coding assistants (Claude, GitHub Copilot, Cursor, etc.) working on this codebase.

## Project Overview

**Jan Server Mono** is a unified backend API platform for LLM applications. It consolidates multiple microservices into a single Go application with OpenAI-compatible APIs, authentication, conversation management, and tool integrations.

**Tech Stack:**
- **Language:** Go 1.24
- **Framework:** Gin (HTTP)
- **ORM:** GORM
- **Database:** PostgreSQL
- **Storage:** S3/MinIO
- **Auth:** JWT (local) + Keycloak (OIDC)

---

## Quick Reference

```bash
# Development
docker compose up -d postgres minio    # Start infrastructure
cd apps/backend && make dev            # Run with hot reload
make test                              # Run tests
make fmt                               # Format code

# Docker
docker compose up -d                   # Start all services
docker compose logs -f backend         # View logs
docker compose down                    # Stop services

# Database
make db-console                        # PostgreSQL shell
make db-migrate                        # Run migrations
```

---

## Repository Structure

```
mono/
├── apps/backend/                      # Go API Server
│   ├── cmd/server/main.go             # Application entrypoint
│   ├── internal/
│   │   ├── domain/                    # Business logic (NO external deps)
│   │   │   ├── user/                  # Authentication & users
│   │   │   ├── conversation/          # Chat conversations
│   │   │   ├── model/                 # LLM models & providers
│   │   │   ├── artifact/              # Code artifacts
│   │   │   ├── media/                 # File storage
│   │   │   └── connector/             # OAuth connectors
│   │   ├── infrastructure/
│   │   │   ├── config/                # Configuration loading
│   │   │   └── database/
│   │   │       ├── dbschema/          # GORM models
│   │   │       └── repository/        # Data access layer
│   │   └── interfaces/httpserver/
│   │       ├── routes/                # HTTP handlers
│   │       ├── middlewares/           # Auth, logging, CORS
│   │       └── server.go              # Server setup
│   ├── pkg/common/                    # Shared utilities
│   ├── migrations/                    # SQL migrations
│   └── tests/                         # Integration tests
├── apps/web/                          # React frontend
├── scripts/                           # Dev scripts
├── docker-compose.yml                 # Container orchestration
└── Makefile                           # Build automation
```

---

## Architecture Rules

### Clean Architecture Layers

```
Interfaces (HTTP handlers, routes)
        ↓ calls
Domain (entities, services, business logic)
        ↓ calls
Infrastructure (repositories, external clients)
```

**CRITICAL RULES:**

1. **Domain packages NEVER import HTTP or database packages**
   - Domain defines interfaces
   - Infrastructure implements them
   - This keeps business logic testable and portable

2. **HTTP handlers are thin**
   - Parse request → call domain service → return response
   - No business logic in handlers

3. **Repositories implement domain interfaces**
   - Domain defines `Repository` interface
   - `repository/` package implements it with GORM

### Example Domain Structure

```go
// internal/domain/user/entity.go
type User struct {
    ID       string
    Email    string
    // ... domain fields (not GORM tags)
}

type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
    // ... other methods
}

// internal/domain/user/service.go
type Service struct {
    repo Repository  // Interface, not concrete type
}

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    return s.repo.FindByID(ctx, id)
}
```

---

## Code Conventions

### Naming

| Context | Convention | Example |
|---------|------------|---------|
| Exported Go | PascalCase | `UserService`, `CreateUser` |
| Unexported Go | camelCase | `userRepo`, `buildQuery` |
| Database columns | snake_case | `created_at`, `user_id` |
| JSON fields | snake_case | `"user_id"`, `"created_at"` |
| Environment vars | SCREAMING_SNAKE | `JWT_SECRET`, `DATABASE_URL` |

### Import Order

```go
import (
    // Standard library
    "context"
    "fmt"

    // Third-party packages
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // Internal packages
    "jan-server/mono/apps/backend/internal/domain/user"
)
```

### Error Handling

Use domain-specific error types:

```go
// In domain/user/entity.go
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidPassword = errors.New("invalid password")
)

// In service
if user == nil {
    return nil, ErrUserNotFound
}

// In handler
if errors.Is(err, user.ErrUserNotFound) {
    c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
    return
}
```

---

## GORM Patterns

### Pointer Types for Zero Values

GORM skips zero values (`false`, `0`, `""`) on updates. Use pointers:

```go
// BAD: Cannot set Enabled to false
type Model struct {
    Enabled bool `gorm:"not null;default:true"`
}

// GOOD: Use pointer
type Model struct {
    Enabled *bool `gorm:"not null;default:true"`
}
```

### Schema ↔ Domain Conversion

```go
// dbschema/user.go
type User struct {
    ID        string `gorm:"primaryKey"`
    Email     string `gorm:"uniqueIndex"`
    IsActive  *bool  `gorm:"not null;default:true"`
    CreatedAt time.Time
}

// Convert to domain
func (u *User) ToDomain() *domain.User {
    isActive := true
    if u.IsActive != nil {
        isActive = *u.IsActive
    }
    return &domain.User{
        ID:       u.ID,
        Email:    u.Email,
        IsActive: isActive,
    }
}

// Convert from domain
func NewSchemaUser(d *domain.User) *User {
    isActive := d.IsActive
    return &User{
        ID:       d.ID,
        Email:    d.Email,
        IsActive: &isActive,
    }
}
```

---

## API Patterns

### Handler Structure

```go
func createUserHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Get authenticated user (if needed)
        principal := middlewares.GetPrincipal(c)
        if principal == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        // 2. Parse request
        var req CreateUserRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        // 3. Call domain service
        repo := repository.NewUserRepository(db)
        svc := user.NewService(repo, cfg)
        result, err := svc.CreateUser(c.Request.Context(), req)

        // 4. Handle errors
        if err != nil {
            if errors.Is(err, user.ErrEmailExists) {
                c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
                return
            }
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // 5. Return response
        c.JSON(http.StatusCreated, result.ToResponse())
    }
}
```

### Authentication

```go
// Get authenticated user in handler
principal := middlewares.GetPrincipal(c)
if principal == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
    return
}
userID := principal.ID

// Use in API key auth
// Header: Authorization: Bearer jan_xxxx
```

---

## Common Tasks

### Adding a New Entity

1. **Create domain types** (`internal/domain/{name}/entity.go`):
```go
type Thing struct {
    ID        string
    Name      string
    CreatedAt time.Time
}

var ErrThingNotFound = errors.New("thing not found")

type Repository interface {
    Create(ctx context.Context, t *Thing) error
    FindByID(ctx context.Context, id string) (*Thing, error)
}
```

2. **Create domain service** (`internal/domain/{name}/service.go`):
```go
type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, name string) (*Thing, error) {
    t := &Thing{
        ID:        uuid.New().String(),
        Name:      name,
        CreatedAt: time.Now(),
    }
    return t, s.repo.Create(ctx, t)
}
```

3. **Create GORM schema** (`internal/infrastructure/database/dbschema/{name}.go`)

4. **Create repository** (`internal/infrastructure/database/repository/{name}_repo.go`)

5. **Create HTTP handlers** (`internal/interfaces/httpserver/routes/{name}_handlers.go`)

6. **Register routes** in `routes.go`

7. **Add migration** in `migrations/`

### Adding an API Endpoint

1. Add handler function in appropriate `*_handlers.go` file
2. Register route in `Register*Routes` function in `routes.go`
3. Test with curl or integration test

---

## Testing

### Unit Tests

```go
// internal/domain/user/service_test.go
func TestService_Create(t *testing.T) {
    repo := NewMockRepository()
    svc := NewService(repo, config)

    user, err := svc.Create(ctx, CreateRequest{
        Email: "test@example.com",
    })

    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

### Integration Tests

```go
// tests/integration/auth_test.go
func TestRegister(t *testing.T) {
    router := setupTestRouter()

    body := `{"email":"test@example.com","password":"pass123"}`
    req, _ := http.NewRequest("POST", "/v1/auth/local/register", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)
}
```

---

## Git Guidelines

### Branch Naming
- `feat/` - New features
- `fix/` - Bug fixes
- `chore/` - Maintenance

### Commit Messages
Use conventional commits:
```
feat: add user registration endpoint
fix: correct JWT expiration calculation
chore: update dependencies
```

### Before Committing
1. `make fmt` - Format code
2. `make test` - Run tests
3. `make lint` - Run linter (if available)

---

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_POSTGRESQL_WRITE_DSN` | PostgreSQL connection string | Yes |
| `LOCAL_JWT_SECRET` | JWT signing key (32+ chars) | Yes |
| `S3_ENDPOINT` | MinIO/S3 endpoint | For media |
| `S3_ACCESS_KEY_ID` | S3 access key | For media |
| `S3_SECRET_ACCESS_KEY` | S3 secret key | For media |

---

## Troubleshooting

### Build Errors
- Missing `go.sum`: Run `go mod tidy` in Docker build
- Import errors: Check Clean Architecture violations

### Runtime Errors
- DB connection: Check `DB_POSTGRESQL_WRITE_DSN`
- Auth failures: Check `LOCAL_JWT_SECRET` is set
- 404 errors: Check route registration in `routes.go`

### Docker Issues
- Port conflicts: Change ports in `.env`
- Health check fails: Check logs with `docker compose logs backend`

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | Application entrypoint |
| `internal/infrastructure/config/config.go` | Configuration struct |
| `internal/infrastructure/database/connection.go` | DB setup & migrations |
| `internal/interfaces/httpserver/server.go` | HTTP server setup |
| `internal/interfaces/httpserver/routes/routes.go` | Route registration |
| `internal/interfaces/httpserver/middlewares/auth.go` | JWT middleware |

---

## AI Assistant Tips

When working on this codebase:

1. **Read before editing** - Always read files before suggesting changes
2. **Respect architecture** - Keep domain logic in domain, HTTP in interfaces
3. **Use domain errors** - Define errors in entity.go, check with `errors.Is`
4. **Test your changes** - Mention relevant test commands
5. **Check existing patterns** - Look at similar code before adding new
6. **Don't over-engineer** - Keep changes minimal and focused
