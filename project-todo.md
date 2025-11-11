# Project TODO

Mirror the project behavior so projects can group conversations and inherit instructions, similar to the workspace pattern but simplified.

## Key requirements

- Projects are created via POST `/llm/projects`, and include optional `instruction` text that defines the project persona/context.
- Each project is scoped to the authenticated user and exposed through GET `/llm/projects`; we need to retain `name`, `instruction`, and `created_at`/`updated_at` metadata.
- Updates use PATCH (name or instruction) and DELETE per the route middleware.
- Projects group conversations by linking `project_id` to conversation entities.
- Unique constraint on `(user_id, name)` to prevent duplicate project names per user.
- Soft-delete with `deleted_at` to avoid accidental data loss.
- Support archiving with `archived_at` to hide without deleting.
- Cursor-based pagination for list endpoints.

## Implementation Plan

### 1. Domain Layer
- Create `services/llm-api/internal/domain/project/project.go`
  - Define `Project` entity with: ID, PublicID, UserID, Name, Instruction (optional text), Favorite, ArchivedAt, DeletedAt, LastUsedAt, timestamps
  - Factory method `NewProject()` for creation

### 2. Database Schema
- Create `services/llm-api/internal/infrastructure/database/dbschema/project.go`
  - `Project` table schema with GORM annotations
  - Fields: id, public_id, user_id, name, instruction, favorite, archived_at, deleted_at, last_used_at, created_at, updated_at
  - Unique constraint on (user_id, name)
  - Conversion methods: `EtoD()` (Entity to Domain) and `DtoE()` (Domain to Entity)
- Create migration `migrations/000X_create_projects.up.sql`
  - `projects` table with user_id index and composite unique index
  - Index: `idx_projects_user_updated_at` on (user_id, updated_at DESC)
- Update `Conversation` schema to include optional `project_id` foreign key (ON DELETE SET NULL)
  - Add `instruction_version` INT NOT NULL DEFAULT 1
  - Add `effective_instruction_snapshot` TEXT NULL for reproducibility
  - Index: `idx_conversations_project_updated_at` on (project_id, updated_at DESC)

### 3. Repository Layer
- Create `services/llm-api/internal/infrastructure/database/repository/projectrepo/project_repository.go`
  - `Create()` - Create new project
  - `GetByPublicID()` - Retrieve single project
  - `ListByUserID()` - List all projects for user
  - `Update()` - Update name or instruction
  - `Delete()` - Delete project

### 4. HTTP Routes & Handlers
- Create `services/llm-api/internal/interfaces/httpserver/routes/v1/projects/route.go`
  - POST `/v1/projects` - Create project
  - GET `/v1/projects` - List user's projects
  - GET `/v1/projects/:project_id` - Get single project
  - PATCH `/v1/projects/:project_id` - Update project
  - DELETE `/v1/projects/:project_id` - Delete project
- All routes verify user ownership via auth middleware

### 5. Request/Response DTOs
- Create `services/llm-api/internal/interfaces/httpserver/requests/projectreq/requests.go`
  - `CreateProjectRequest` - name (required), instruction (optional)
  - `UpdateProjectRequest` - name (optional), instruction (optional), archived (optional)
  - Validation: name ≤ 120 chars, instruction ≤ 32k chars, trim whitespace, reject control chars
- Create `services/llm-api/internal/interfaces/httpserver/responses/projectres/responses.go`
  - `ProjectResponse` - Single project response with object="project"
  - `ProjectListResponse` - List response with object="list", has_more, next_cursor

### 6. Wire Dependencies
- Update `services/llm-api/cmd/server/wire.go` to inject ProjectRepository and ProjectRoute
- Run `make wire` to generate provider code

### 7. Conversation Grouping
- Update conversation creation to accept optional `project_id` parameter
- When listing conversations, filter by `project_id` if provided
- Project's `instruction` text is inherited by conversations in that project
- Allow moving conversations out of project by setting `project_id` to null via PATCH
- When a project is deleted (soft-delete), set conversations' `project_id` to NULL (ON DELETE SET NULL)

### 8. Instruction Inheritance Semantics
- **Resolution order**: `system > project.instruction > conversation.instruction`
- On conversation create:
  - Compute `effective_instruction` by merging project and conversation instructions
  - Store `instruction_version` (increment on project instruction change)
  - Store `effective_instruction_snapshot` for reproducibility
- On project instruction update:
  - Increment project version
  - Do not rewrite existing conversations automatically
  - New conversations inherit the latest instruction
  - Provide endpoint to manually recompute: `POST /conversations/:id/recompute-instruction`

## Next Steps

1. Create domain entity and database schema
2. Implement repository layer with CRUD operations
3. Generate GORM code: `make gormgen`
4. Create and run database migration
5. Implement HTTP routes and handlers
6. Create request/response DTOs
7. Update Wire dependencies: `make wire`
8. Update Swagger documentation: `make doc`
9. Link conversations to projects via `project_id` field
10. Test all endpoints with authentication

## API Endpoints

- `POST /v1/projects` - Create project with name and optional instruction
- `GET /v1/projects` - List all projects for authenticated user with cursor pagination
  - Query params: `?limit=50&cursor=...&sort=updated_at&archived=false&search=...`
- `GET /v1/projects/:project_id` - Get project details
- `PATCH /v1/projects/:project_id` - Update project name, instruction, or archived status
- `DELETE /v1/projects/:project_id` - Soft-delete project
- `POST /v1/conversations` - Create conversation with optional `project_id`
- `GET /v1/conversations?project_id=xxx` - List conversations in project with cursor pagination
- `PATCH /v1/conversations/:id` - Move conversation to/from project by updating `project_id`
- `POST /v1/conversations/:id/recompute-instruction` - Manually recompute effective instruction

## Database Schema

### projects table
```sql
CREATE TABLE projects (
  id BIGSERIAL PRIMARY KEY,
  public_id TEXT NOT NULL UNIQUE,
  user_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  instruction TEXT,
  favorite BOOLEAN NOT NULL DEFAULT false,
  archived_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_projects_user_name UNIQUE (user_id, name)
);

CREATE INDEX idx_projects_user_updated_at ON projects (user_id, updated_at DESC);
```

### conversations table updates
```sql
ALTER TABLE conversations
  ADD COLUMN project_id BIGINT NULL REFERENCES projects(id) ON DELETE SET NULL,
  ADD COLUMN instruction_version INT NOT NULL DEFAULT 1,
  ADD COLUMN effective_instruction_snapshot TEXT NULL;

CREATE INDEX idx_conversations_project_updated_at
  ON conversations (project_id, updated_at DESC);
```

## Response Formats

### Single project response
```json
{
  "object": "project",
  "id": "proj_8x2y3...",
  "name": "Growth Experiments",
  "instruction": "You are a growth-focused assistant...",
  "favorite": false,
  "archived_at": null,
  "created_at": "2025-11-11T08:12:00Z",
  "updated_at": "2025-11-11T08:12:00Z"
}
```

### List response (cursor-paged)
```json
{
  "object": "list",
  "data": [
    {
      "object": "project",
      "id": "proj_...",
      "name": "...",
      "instruction": "...",
      "favorite": false,
      "archived_at": null,
      "created_at": "2025-11-11T08:12:00Z",
      "updated_at": "2025-11-11T08:12:00Z"
    }
  ],
  "has_more": true,
  "next_cursor": "c3VyZ..."
}
```