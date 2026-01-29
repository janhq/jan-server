# AGENTS.md - Detailed Agent Guidelines for Jan Server Mono

> Comprehensive guidelines for AI agents working on the Jan Server Mono codebase. This document provides detailed context, patterns, and procedures for making changes.

## System Overview

### What is Jan Server Mono?

Jan Server Mono is a **unified backend API** for LLM-powered applications. It provides:

| Feature | Description |
|---------|-------------|
| **Chat API** | OpenAI-compatible `/v1/chat/completions` with streaming |
| **Auth System** | JWT-based local auth + Keycloak OIDC support |
| **Conversations** | Persistent chat history with branching/sharing |
| **Models** | Multi-provider model management (OpenAI, Anthropic, etc.) |
| **Artifacts** | Code/content storage with versioning |
| **Media** | S3-compatible file uploads |
| **Connectors** | OAuth integrations (GitHub, Google) |
| **MCP** | Model Context Protocol for tool integration |
| **Response API** | Multi-step tool orchestration |

### Tech Stack

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                            │
│                    React + Vite (apps/web)                 │
├─────────────────────────────────────────────────────────────┤
│                       Go Backend                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │    Gin      │  │    GORM     │  │   JWT/Auth  │        │
│  │  (HTTP)     │  │   (ORM)     │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
├─────────────────────────────────────────────────────────────┤
│                     Infrastructure                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ PostgreSQL  │  │   MinIO     │  │   Redis     │        │
│  │  (Database) │  │    (S3)     │  │  (Cache)    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

---

## Directory Structure Deep Dive

```
mono/
├── apps/
│   ├── backend/                          # Go API Server
│   │   ├── cmd/
│   │   │   └── server/
│   │   │       └── main.go               # Entry point - starts HTTP server
│   │   │
│   │   ├── internal/                     # Private application code
│   │   │   ├── domain/                   # BUSINESS LOGIC LAYER
│   │   │   │   ├── user/
│   │   │   │   │   ├── entity.go         # User, APIKey, RefreshToken types
│   │   │   │   │   └── service.go        # Auth logic: register, login, JWT
│   │   │   │   ├── conversation/
│   │   │   │   │   ├── entity.go         # Conversation, Message types
│   │   │   │   │   └── service.go        # CRUD, branching, sharing
│   │   │   │   ├── model/
│   │   │   │   │   ├── entity.go         # Provider, Model types
│   │   │   │   │   └── service.go        # Provider/model management
│   │   │   │   ├── artifact/
│   │   │   │   │   ├── entity.go         # Artifact, Version types
│   │   │   │   │   └── service.go        # CRUD with versioning
│   │   │   │   ├── media/
│   │   │   │   │   ├── entity.go         # Media file types
│   │   │   │   │   └── service.go        # Upload, presigned URLs
│   │   │   │   └── connector/
│   │   │   │       ├── entity.go         # OAuth connector types
│   │   │   │       └── service.go        # OAuth flow handling
│   │   │   │
│   │   │   ├── infrastructure/           # EXTERNAL INTEGRATIONS
│   │   │   │   ├── config/
│   │   │   │   │   └── config.go         # All env vars, loaded at startup
│   │   │   │   └── database/
│   │   │   │       ├── connection.go     # DB connection, auto-migration
│   │   │   │       ├── dbschema/         # GORM models (with tags)
│   │   │   │       │   ├── user.go
│   │   │   │       │   ├── llm.go        # Provider, Model, Conversation, Message
│   │   │   │       │   ├── response.go   # Artifact, Response
│   │   │   │       │   ├── media.go
│   │   │   │       │   └── connector.go
│   │   │   │       └── repository/       # Data access implementations
│   │   │   │           ├── user_repo.go
│   │   │   │           ├── conversation_repo.go
│   │   │   │           ├── model_repo.go
│   │   │   │           ├── artifact_repo.go
│   │   │   │           ├── media_repo.go
│   │   │   │           └── connector_repo.go
│   │   │   │
│   │   │   └── interfaces/               # HTTP LAYER
│   │   │       └── httpserver/
│   │   │           ├── server.go         # Server setup, route registration
│   │   │           ├── middlewares/
│   │   │           │   ├── auth.go       # JWT validation, principal extraction
│   │   │           │   ├── cors.go       # CORS configuration
│   │   │           │   ├── logger.go     # Request logging
│   │   │           │   └── request_id.go # Request ID generation
│   │   │           └── routes/
│   │   │               ├── routes.go     # Route registration functions
│   │   │               ├── handlers.go   # Misc handlers (agents, memory, etc.)
│   │   │               ├── auth_handlers.go
│   │   │               ├── chat_handlers.go
│   │   │               ├── conversation_handlers.go
│   │   │               ├── model_handlers.go
│   │   │               ├── artifact_handlers.go
│   │   │               ├── media_handlers.go
│   │   │               └── connector_handlers.go
│   │   │
│   │   ├── pkg/common/                   # Shared utilities
│   │   │   ├── errors/                   # Platform error types
│   │   │   ├── logger/                   # Zerolog wrapper
│   │   │   ├── config/                   # Env var helpers
│   │   │   ├── responses/                # HTTP response helpers
│   │   │   ├── middleware/               # Reusable middleware
│   │   │   ├── observability/            # Tracing utilities
│   │   │   └── utils/                    # Pagination, pointers, strings
│   │   │
│   │   ├── migrations/                   # SQL migrations
│   │   │   ├── 000001_init.up.sql
│   │   │   └── 000001_init.down.sql
│   │   │
│   │   ├── tests/                        # Integration tests
│   │   │   └── integration/
│   │   │
│   │   ├── Dockerfile                    # Multi-stage build
│   │   ├── Makefile                      # Build commands
│   │   ├── go.mod                        # Go module definition
│   │   └── .air.toml                     # Hot reload config
│   │
│   └── web/                              # React frontend
│       └── Dockerfile
│
├── scripts/
│   ├── setup.sh                          # Environment setup
│   └── dev.sh                            # Development helpers
│
├── docker-compose.yml                    # Container orchestration
├── Makefile                              # Root build commands
├── .env                                  # Environment variables
├── .env.example                         # Environment template
├── .gitignore
├── README.md
├── CLAUDE.md                             # AI assistant quick reference
└── AGENTS.md                             # This file
```

---

## Domain Entities Reference

### User Domain (`internal/domain/user/`)

```go
// Entities
type User struct {
    ID, Email, Username, PasswordHash, Name string
    IsActive, IsAdmin bool
    LastLoginAt *time.Time
    CreatedAt, UpdatedAt time.Time
}

type APIKey struct {
    ID, UserID, Name, KeyHash, KeyPrefix string
    Scopes []string
    ExpiresAt, RevokedAt, LastUsedAt *time.Time
    CreatedAt time.Time
}

type RefreshToken struct {
    ID, UserID, TokenHash string
    ExpiresAt time.Time
    RevokedAt *time.Time
}

// Key Methods
func (s *Service) Register(ctx, RegisterRequest) (*AuthResponse, error)
func (s *Service) Login(ctx, LoginRequest) (*AuthResponse, error)
func (s *Service) ValidateToken(ctx, token) (*Claims, error)
func (s *Service) CreateAPIKey(ctx, userID, name, expiry) (*APIKey, rawKey, error)
func (s *Service) ValidateAPIKey(ctx, rawKey) (*APIKey, error)
```

### Conversation Domain (`internal/domain/conversation/`)

```go
// Entities
type Conversation struct {
    ID, UserID, Title, ModelID, SystemPrompt string
    IsArchived, IsPinned bool
    ShareToken *string
    Metadata map[string]any
}

type Message struct {
    ID, ConversationID, Role, Content string
    ModelID, ParentID *string
    ToolCalls []ToolCall
    TokenCount *int
}

// Key Methods
func (s *Service) Create(ctx, userID, CreateRequest) (*Conversation, error)
func (s *Service) GetByID(ctx, userID, id) (*Conversation, error)
func (s *Service) AddMessage(ctx, userID, convID, *Message) error
func (s *Service) GetMessages(ctx, userID, convID, limit, offset) ([]*Message, total, error)
func (s *Service) Share(ctx, userID, convID) (shareToken, error)
```

### Model Domain (`internal/domain/model/`)

```go
// Entities
type Provider struct {
    ID, Name, Type, BaseURL, APIKey string
    IsEnabled bool
    Priority int
    Config map[string]any
}

type Model struct {
    ID, ProviderID, Name, DisplayName, Description string
    MaxTokens, ContextWindow int
    InputCost, OutputCost decimal.Decimal
    IsEnabled, SupportsStreaming, SupportsFunctions, SupportsVision bool
}

// Key Methods
func (s *Service) CreateProvider(ctx, CreateProviderRequest) (*Provider, error)
func (s *Service) CreateModel(ctx, CreateModelRequest) (*Model, error)
func (s *Service) GetModelByID(ctx, id) (*Model, error)  // Also searches by name
func (s *Service) ListModels(ctx, filter) ([]*Model, total, error)
```

---

## API Routes Reference

### Authentication Routes

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| POST | `/v1/auth/local/register` | No | `localRegisterHandler` |
| POST | `/v1/auth/local/login` | No | `localLoginHandler` |
| POST | `/v1/auth/local/refresh` | No | `localRefreshHandler` |
| POST | `/v1/auth/logout` | No | `logoutHandler` |
| POST | `/v1/auth/validate` | No | `validateTokenHandler` |
| GET | `/v1/auth/me` | Yes | `meHandler` |
| POST | `/v1/auth/api-keys` | Yes | `createAPIKeyHandler` |
| GET | `/v1/auth/api-keys` | Yes | `listAPIKeysHandler` |
| DELETE | `/v1/auth/api-keys/:id` | Yes | `deleteAPIKeyHandler` |

### Chat Routes

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| POST | `/v1/chat/completions` | Yes | `chatCompletionsHandler` |

### Conversation Routes

| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/v1/conversations` | Yes | `listConversationsHandler` |
| POST | `/v1/conversations` | Yes | `createConversationHandler` |
| GET | `/v1/conversations/:id` | Yes | `getConversationHandler` |
| PUT | `/v1/conversations/:id` | Yes | `updateConversationHandler` |
| DELETE | `/v1/conversations/:id` | Yes | `deleteConversationHandler` |
| GET | `/v1/messages` | Yes | `listMessagesHandler` |
| GET | `/share/:token` | No | `getSharedConversationHandler` |

### Admin Routes (Requires admin role)

| Method | Path | Handler |
|--------|------|---------|
| POST | `/v1/admin/providers` | `adminCreateProviderHandler` |
| PUT | `/v1/admin/providers/:id` | `adminUpdateProviderHandler` |
| DELETE | `/v1/admin/providers/:id` | `adminDeleteProviderHandler` |
| POST | `/v1/admin/models` | `adminCreateModelHandler` |
| PUT | `/v1/admin/models/:id` | `adminUpdateModelHandler` |
| DELETE | `/v1/admin/models/:id` | `adminDeleteModelHandler` |

---

## Common Patterns

### Creating a New Handler

```go
// routes/{domain}_handlers.go

func myNewHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Auth check (if protected route)
        principal := middlewares.GetPrincipal(c)
        if principal == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        // 2. Parse request body (for POST/PUT)
        var req MyRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        // 3. Parse path/query params
        id := c.Param("id")
        limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

        // 4. Create service and call
        repo := repository.NewMyRepository(db)
        svc := mydomain.NewService(repo)
        result, err := svc.DoSomething(c.Request.Context(), principal.ID, req)

        // 5. Handle errors
        if err != nil {
            switch {
            case errors.Is(err, mydomain.ErrNotFound):
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            case errors.Is(err, mydomain.ErrUnauthorized):
                c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
            default:
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            }
            return
        }

        // 6. Return response
        c.JSON(http.StatusOK, result.ToResponse())
    }
}
```

### Creating a Repository

```go
// repository/{domain}_repo.go

type MyRepository struct {
    db *gorm.DB
}

func NewMyRepository(db *gorm.DB) *MyRepository {
    return &MyRepository{db: db}
}

func (r *MyRepository) Create(ctx context.Context, entity *domain.MyEntity) error {
    schema := dbschema.NewSchemaFromDomain(entity)
    return r.db.WithContext(ctx).Create(schema).Error
}

func (r *MyRepository) FindByID(ctx context.Context, id string) (*domain.MyEntity, error) {
    var schema dbschema.MyEntity
    if err := r.db.WithContext(ctx).First(&schema, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, domain.ErrNotFound
        }
        return nil, err
    }
    return schema.ToDomain(), nil
}

func (r *MyRepository) List(ctx context.Context, filter domain.Filter) ([]*domain.MyEntity, int64, error) {
    var schemas []dbschema.MyEntity
    var total int64

    query := r.db.WithContext(ctx).Model(&dbschema.MyEntity{})

    // Apply filters
    if filter.Search != "" {
        query = query.Where("name ILIKE ?", "%"+filter.Search+"%")
    }

    // Count total
    query.Count(&total)

    // Apply pagination
    query = query.Offset(filter.Offset).Limit(filter.Limit)
    query = query.Order("created_at DESC")

    if err := query.Find(&schemas).Error; err != nil {
        return nil, 0, err
    }

    result := make([]*domain.MyEntity, len(schemas))
    for i, s := range schemas {
        result[i] = s.ToDomain()
    }
    return result, total, nil
}
```

---

## Testing Patterns

### Domain Service Test

```go
// domain/{name}/service_test.go

type MockRepository struct {
    data map[string]*Entity
}

func (m *MockRepository) Create(ctx context.Context, e *Entity) error {
    m.data[e.ID] = e
    return nil
}

func (m *MockRepository) FindByID(ctx context.Context, id string) (*Entity, error) {
    if e, ok := m.data[id]; ok {
        return e, nil
    }
    return nil, ErrNotFound
}

func TestService_Create(t *testing.T) {
    repo := &MockRepository{data: make(map[string]*Entity)}
    svc := NewService(repo)

    result, err := svc.Create(context.Background(), CreateRequest{Name: "test"})

    assert.NoError(t, err)
    assert.NotEmpty(t, result.ID)
    assert.Equal(t, "test", result.Name)
}
```

### Integration Test

```go
// tests/integration/{name}_test.go

func TestMyEndpoint(t *testing.T) {
    router := setupTestRouter()

    body := `{"name": "test"}`
    req, _ := http.NewRequest("POST", "/v1/things", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testToken)

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.NotEmpty(t, response["id"])
}
```

---

## Debugging Guide

### Check Registered Routes

```bash
docker compose logs backend | grep "GIN-debug"
```

### Test Authentication

```bash
# Register
curl -X POST http://localhost:8080/v1/auth/local/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123","username":"test"}'

# Login
curl -X POST http://localhost:8080/v1/auth/local/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Use token
curl http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer <token>"
```

### Database Access

```bash
# Via Docker
docker compose exec postgres psql -U jan -d jan

# Common queries
SELECT * FROM users;
SELECT * FROM conversations WHERE user_id = 'xxx';
SELECT * FROM providers;
SELECT * FROM models;
```

---

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| `go.sum` missing | Run `go mod tidy` in Dockerfile before build |
| Port already in use | Change ports in `.env` file |
| Auth middleware fails | Check `LOCAL_JWT_SECRET` is set |
| DB connection fails | Verify `DB_POSTGRESQL_WRITE_DSN` format |
| Route returns 404 | Check route registration in `routes.go` |
| CORS errors | Check `middlewares/cors.go` allowed origins |

---

## Checklist for Changes

Before submitting changes:

- [ ] Code compiles: `go build ./...`
- [ ] Tests pass: `go test ./...`
- [ ] Code formatted: `go fmt ./...`
- [ ] Domain logic in domain layer (no HTTP/DB imports)
- [ ] Errors defined in entity.go
- [ ] Repository implements domain interface
- [ ] Handler follows standard pattern
- [ ] Route registered in routes.go
- [ ] No hardcoded secrets
- [ ] Logging uses structured fields
