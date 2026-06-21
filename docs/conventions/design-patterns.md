# Code Patterns & Best Practices

> **When to read this:** Daily reference for AI agents and developers writing code.
>
> **For structure:** See [architecture-patterns.md](architecture-patterns.md)
> **For workflow:** See [workflow.md](workflow.md)
> **Quick reference:** See [conventions.md](conventions.md)

---

## Table of Contents

1. [Database Patterns](#database-patterns)
2. [Error Handling](#error-handling)
3. [Domain Entity Creation](#domain-entity-creation)
4. [Performance Patterns](#performance-patterns)

> **Path convention:** The examples below assume you are working inside a service directory such as `services/llm-api/`. Adjust paths accordingly for other services.

---

## Database Patterns

### GORM Zero-Value Handling CRITICAL

**Problem:** GORM's `.Save()` silently skips fields with zero values (`false`, `0`, `0.0`) to avoid overwriting database data with uninitialized struct fields.

**Solution:** Use pointer types for fields that may legitimately be set to zero values.

```go
// Bad: Cannot set Enabled to false or Amount to 0.0
type User struct {
 BaseModel
 Enabled bool `gorm:"not null;default:true"`
 Amount float64 `gorm:"not null"`
}

// Good: Use pointers for zero-affected fields
type User struct {
 BaseModel
 Enabled *bool `gorm:"not null;default:true"`
 Amount *float64 `gorm:"not null"`
}

// Conversion pattern in NewSchemaUser() - create pointer from value
func NewSchemaUser(u *user.User) *User {
 enabled := u.Enabled
 amount := u.Amount
 return &User{
 Enabled: &enabled, // Always non-nil pointer
 Amount: &amount,
 }
}

// Conversion pattern in EtoD() - dereference with nil-check
func (u *User) EtoD() *user.User {
 enabled := false // Default value
 if u.Enabled != nil {
 enabled = *u.Enabled
 }
 amount := 0.0 // Default value
 if u.Amount != nil {
 amount = *u.Amount
 }
 return &user.User{
 Enabled: enabled, // Plain type in domain
 Amount: amount,
 }
}
```

**When to use pointers:**

- Boolean fields that need to be `false` (e.g., `Enabled`, `Active`, `IsPrivate`)
- Numeric fields that can be `0` or `0.0` (e.g., `Amount`, `Credits`)
- Fields that are always non-zero (e.g., IDs, timestamps)
- Counters that only increment (e.g., `ViewCount`)

**Why this works:** `*bool` zero value is `nil`, so `&false` is NOT a zero value -> GORM updates it

**Common scenarios fixed:**

- Disabling API keys (`Enabled = false`)
- Deactivating users/providers (`Active = false`)
- Recording $0.00 transactions (`Amount = 0.0`)
- Zero-credit operations (`Credits = 0`)

---

### Schema Definition Pattern

```go
// internal/infrastructure/database/dbschema/organization.go
type Organization struct {
 BaseModel // Must include BaseModel (ID, CreatedAt, UpdatedAt, DeletedAt)
 PublicID string `gorm:"size:64;not null;uniqueIndex"`
 Name string `gorm:"size:255;not null"`
 Active *bool `gorm:"not null;default:true;index"` // Pointer for zero-value
}

func init() {
 database.RegisterSchemaForAutoMigrate(Organization{})
}

// EtoD: Entity to Domain (method on schema struct)
func (e *Organization) EtoD() *domain.Organization {
 if e == nil {
 return nil
 }
 active := true // Default
 if e.Active != nil {
 active = *e.Active
 }
 return &domain.Organization{
 ID: e.ID,
 PublicID: e.PublicID,
 Name: e.Name,
 Active: active,
 CreatedAt: e.CreatedAt,
 UpdatedAt: e.UpdatedAt,
 }
}

// NewSchemaOrganization: Domain to Entity (package-level function)
func NewSchemaOrganization(d *domain.Organization) *Organization {
 if d == nil {
 return nil
 }
 active := d.Active
 return &Organization{
 BaseModel: BaseModel{
 ID: d.ID,
 CreatedAt: d.CreatedAt,
 UpdatedAt: d.UpdatedAt,
 },
 PublicID: d.PublicID,
 Name: d.Name,
 Active: &active,
 }
}
```

**Key points:**

- `EtoD()` is a method (has receiver)
- `NewSchema*()` is a function (no receiver)
- Always nil-check to prevent panics
- For pointers: create variable, then reference it

---

### Repository Pattern

```go
type OrganizationRepository struct {
 db *transaction.Database
}

// Create returns the created entity with generated fields
func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
 dbOrg := dbschema.NewSchemaOrganization(org)

 if err := r.db.GetQuery(ctx).Organization.WithContext(ctx).Create(dbOrg); err != nil {
 return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "failed to create organization", err,
 "c1d2e3f4-a5b6-4c7d-8e9f-0a1b2c3d4e5f") // Unique UUID
 }

 return dbOrg.EtoD(), nil
}

// FindByID uses GORM gen for type-safe queries
func (r *OrganizationRepository) FindByID(ctx context.Context, id string) (*domain.Organization, error) {
 o := r.db.GetQuery(ctx).Organization
 dbOrg, err := o.WithContext(ctx).Where(o.PublicID.Eq(id)).First()

 if err != nil {
 if errors.Is(err, gorm.ErrRecordNotFound) {
 return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeNotFound, "organization not found", err,
 "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a")
 }
 return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "failed to find organization", err,
 "e7f8a9b0-c1d2-4e3f-4a5b-6c7d8e9f0a1b")
 }

 return dbOrg.EtoD(), nil
}

// Update uses.Save() - works correctly with pointer types
func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
 dbOrg := dbschema.NewSchemaOrganization(org)

 if err := r.db.GetQuery(ctx).Organization.WithContext(ctx).
 Where(r.db.GetQuery(ctx).Organization.ID.Eq(org.ID)).
 Save(dbOrg); err != nil {
 return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "failed to update organization", err,
 "f0a1b2c3-d4e5-4f6a-7b8c-9d0e1f2a3b4c")
 }

 return dbOrg.EtoD(), nil
}

// List with filters and pagination
func (r *OrganizationRepository) List(ctx context.Context, filter *OrganizationFilter) ([]*domain.Organization, int64, error) {
 o := r.db.GetQuery(ctx).Organization
 query := o.WithContext(ctx)

 // Apply filters
 if filter.Active != nil {
 query = query.Where(o.Active.Is(*filter.Active))
 }
 if filter.Name != nil {
 query = query.Where(o.Name.Like("%" + *filter.Name + "%"))
 }

 // Count total
 total, err := query.Count()
 if err != nil {
 return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "failed to count", err, "uuid-here")
 }

 // Apply pagination (cursor-based preferred)
 if filter.Pagination != nil && filter.Pagination.LastID != "" {
 query = query.Where(o.ID.Gt(filter.Pagination.LastID))
 }
 if filter.Pagination != nil && filter.Pagination.Limit > 0 {
 query = query.Limit(filter.Pagination.Limit)
 }

 dbOrgs, err := query.Find()
 if err != nil {
 return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "failed to list", err, "uuid-here")
 }

 // Convert to domain
 orgs := make([]*domain.Organization, len(dbOrgs))
 for i, dbOrg := range dbOrgs {
 orgs[i] = dbOrg.EtoD()
 }

 return orgs, total, nil
}
```

**Filter signature:**

```go
type OrganizationFilter struct {
 ID *uint
 PublicID *string
 Name *string
 Active *bool
 Pagination *PaginationFilter
}

type PaginationFilter struct {
 Limit int
 LastID string // For cursor-based pagination
}
```

---

### GORM Gen Usage

```bash
# After schema changes, regenerate queries
cd services/llm-api
make gormgen
```

**Generated queries** live in `services/llm-api/internal/infrastructure/database/gormgen/`

**Type-safe queries:**

```go
// Good: Compile-time safe
o := query.Use(db).Organization
orgs, err := o.WithContext(ctx).
 Where(o.Active.Is(true)).
 Order(o.CreatedAt.Desc()).
 Limit(100).
 Find()

// Bad: String-based (not type-safe)
db.Where("active = ?", true).
 Order("created_at DESC").
 Find(&orgs)
```

---

### Transactions

```go
// Use transaction wrapper
err := r.db.Transaction(func(tx *gorm.DB) error {
 // All operations in this function are transactional
 if err := tx.Create(&org).Error; err != nil {
 return err // Rolls back
 }

 if err := tx.Create(&user).Error; err != nil {
 return err // Rolls back
 }

 return nil // Commits
})
```

---

## Error Handling

### Error Handling Philosophy

**3-Layer Pattern:**

1. **Repository (trigger point):** Use `NewError()` with unique UUID
2. **Domain (business layer):** Use `AsError()` to add context OR pass through
3. **Route (HTTP layer):** Use `HandleError()` or `HandleNewError()`

### Trigger Point Pattern (Repository)

```go
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
 u := query.Use(r.db.GetDB()).User
 dbUser, err := u.WithContext(ctx).Where(u.PublicID.Eq(id)).First()

 if err != nil {
 if errors.Is(err, gorm.ErrRecordNotFound) {
 // NewError at trigger point with unique UUID
 return nil, platformerrors.NewError(
 ctx,
 platformerrors.LayerRepository,
 platformerrors.ErrorTypeNotFound,
 "user not found",
 err,
 "3e47b618-b750-4064-9b22-ece9e244019d", // Generate unique UUID
 )
 }
 return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository,
 platformerrors.ErrorTypeDatabaseError, "database query failed", err,
 "7f29ac41-8d5e-4a73-b3c1-9e8f2d6a5c4b") // Different UUID per error
 }
 return dbUser.EtoD(), nil
}
```

**UUID Generation:**

```bash
# VS Code: Install "UUID Generator" by netcorext
# Command: Ctrl+Shift+P -> "Insert UUID"
# CLI: uuidgen
# Web: https://www.uuidgenerator.net/
```

### Domain Layer Pattern

```go
// Option 1: Add context with AsError()
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
 user, err := s.repo.FindByID(ctx, id)
 if err != nil {
 return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err,
 "failed to retrieve user")
 }
 return user, nil
}

// Option 2: Pass through (no additional context needed)
func (s *UserService) GetUserSimple(ctx context.Context, id string) (*User, error) {
 return s.repo.FindByID(ctx, id) // Just pass through
}
```

### Route Layer Pattern

```go
// Use HandleError for errors from services
func (r *UserRoute) GetUser(c *gin.Context) {
 id := c.Param("id")

 user, err := r.service.GetUser(c.Request.Context(), id)
 if err != nil {
 responses.HandleError(c, err) // Converts PlatformError to HTTP response
 return
 }
 responses.Success(c, BuildUserResponse(user))
}

// Use HandleNewError for errors at route level
func (r *UserRoute) CreateUser(c *gin.Context) {
 var req CreateUserRequest
 if err := c.ShouldBindJSON(&req); err != nil {
 responses.HandleNewError(c, platformerrors.LayerRoute,
 platformerrors.ErrorTypeValidation, "invalid request", err)
 return
 }

 user, err := r.service.CreateUser(c.Request.Context(), req.ToDomain())
 if err != nil {
 responses.HandleError(c, err)
 return
 }
 responses.Success(c, BuildUserResponse(user))
}
```

---

## Domain Entity Creation

### Step-by-Step Pattern

When creating a new entity (e.g., `organization`):

#### 1. Domain Entity

```go
// internal/domain/organization/organization.go
type Organization struct {
 ID uint
 PublicID string
 Name string
 Active bool
 CreatedAt time.Time
 UpdatedAt time.Time
}

// Entity validates itself
func (o *Organization) Normalize() error {
 o.Name = strings.TrimSpace(o.Name)
 if o.Name == "" {
 return errors.New("name required")
 }
 return nil
}
```

#### 2. Domain Service

```go
// internal/domain/organization/organizationservice.go
type OrganizationService struct {
 repo *organizationrepo.OrganizationRepository
 cache *organizationcache.OrganizationCache
}

// Work directly with domain entities
func (s *OrganizationService) Create(ctx context.Context, org *Organization) (*Organization, error) {
 if err := org.Normalize(); err != nil {
 return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
 platformerrors.ErrorTypeValidation, "invalid organization", err, "uuid-here")
 }

 created, err := s.repo.Create(ctx, org)
 if err != nil {
 return nil, err // Already wrapped at repository
 }

 return created, nil
}
```

#### 3. Database Schema

See [Schema Definition Pattern](#schema-definition-pattern) above

#### 4. Repository

See [Repository Pattern](#repository-pattern) above

#### 5. HTTP Route

```go
// internal/interfaces/httpserver/routes/v1/management/organizations/route.go
type OrganizationRoute struct {
 service *organization.OrganizationService
}

func (r *OrganizationRoute) Create(c *gin.Context) {
 var req CreateOrganizationRequest
 if err := c.ShouldBindJSON(&req); err != nil {
 responses.HandleNewError(c, platformerrors.LayerRoute,
 platformerrors.ErrorTypeValidation, "invalid request", err)
 return
 }

 org, err := r.service.Create(c.Request.Context(), req.ToDomain())
 if err != nil {
 responses.HandleError(c, err)
 return
 }

 responses.Success(c, BuildOrganizationResponse(org))
}
```

---

## Performance Patterns

### Avoid N+1 Queries

```go
// Bad: N+1 query problem
users, _ := db.Find(&users)
for _, user := range users {
 db.First(&profile, "user_id = ?", user.ID) // Query per user!
}

// Good: Preload relationships
db.Preload("Profile").Find(&users)

// Good: Use joins for filtering
db.Joins("JOIN profiles ON profiles.user_id = users.id").
 Where("profiles.verified = ?", true).
 Find(&users)
```

### Cursor-Based Pagination

```go
// Good: Cursor-based (scales well)
u := query.Use(db).User
users, err := u.WithContext(ctx).
 Where(u.ID.Gt(lastID)). // lastID from previous page
 Limit(pageSize).
 Find()

// Acceptable for small datasets: Offset pagination
users, err := u.WithContext(ctx).
 Offset(page * pageSize).
 Limit(pageSize).
 Find()
```

### Caching Pattern

```go
func (s *OrganizationService) GetByID(ctx context.Context, id string) (*Organization, error) {
 // Try cache first
 cacheKey := fmt.Sprintf("org:%s", id)
 if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
 return cached, nil
 }

 // Cache miss: fetch from DB
 org, err := s.repo.FindByID(ctx, id)
 if err != nil {
 return nil, err
 }

 // Store in cache (fire and forget, don't block on cache errors)
 go s.cache.Set(context.Background(), cacheKey, org, 5*time.Minute)

 return org, nil
}

// Invalidate cache on updates
func (s *OrganizationService) Update(ctx context.Context, org *Organization) (*Organization, error) {
 updated, err := s.repo.Update(ctx, org)
 if err != nil {
 return nil, err
 }

 // Invalidate cache
 cacheKey := fmt.Sprintf("org:%s", org.PublicID)
 go s.cache.Delete(context.Background(), cacheKey)

 return updated, nil
}
```

### Context Timeouts

```go
// Set timeouts for external calls
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

resp, err := httpClient.Get(ctx, url)
```

### Batch Operations

```go
// Good: Batch insert
db.CreateInBatches(users, 100) // Insert in batches of 100

// Good: Bulk update with IN clause
u := query.Use(db).User
u.WithContext(ctx).
 Where(u.ID.In(ids...)).
 Update(u.Active, false)
```

---

## Common Patterns Reference

### Request -> Domain Conversion

```go
type CreateOrganizationRequest struct {
 Name string `json:"name" binding:"required"`
}

func (r *CreateOrganizationRequest) ToDomain() *organization.Organization {
 return &organization.Organization{
 Name: r.Name,
 Active: true, // Default
 }
}
```

### Domain -> Response Conversion

```go
type OrganizationResponse struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Active bool `json:"active"`
 CreateAt string `json:"created_at"`
}

func BuildOrganizationResponse(org *organization.Organization) *OrganizationResponse {
 return &OrganizationResponse{
 ID: org.PublicID,
 Name: org.Name,
 Active: org.Active,
 CreatedAt: org.CreatedAt.Format(time.RFC3339),
 }
}
```

---

**See also:**

- [architecture-patterns.md](architecture-patterns.md) - Structure & layers
- [workflow.md](workflow.md) - Git, testing, deployment
- [conventions.md](conventions.md) - Quick TL;DR reference
