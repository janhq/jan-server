# Architecture & Structure Conventions

> Use this file to understand how Jan Server is organised today. Every example below references the real repository structure under `services/<service>`.

---

## Repository Layout (Top Level)

```
jan-server/
├── tools/jan-cli/              # jan-cli sources + wrappers (jan-cli.sh / jan-cli.ps1)
├── config/                     # Shared configuration defaults and templates
├── infra/docker/               # Compose fragments (infra, services, observability)
├── docs/                       # Documentation (guides, conventions, templates, etc.)
├── k8s/                        # Helm chart + Kubernetes manifests
├── services/
│   ├── llm-api/
│   ├── media-api/
│   ├── response-api/
│   ├── mcp-tools/
│   └── template-api/
├── tests/                      # jan-cli api-test collections
├── Makefile                    # Canonical automation entry point
├── docker-compose.yml          # Root compose file wired to profiles
└── infra/docker/dev-full.yml   # Dev-Full overrides (host routing)
```

Each service folder contains the same structure:

```
services/<service>/
├── cmd/
│   ├── server/                 # Service entrypoint
│   └── gormgen/                # (llm-api) schema generator
├── config/                     # Service-specific configuration helpers
├── internal/
│   ├── domain/                 # Business logic (no HTTP/DB imports)
│   ├── infrastructure/         # Repositories, cache, provider clients
│   └── interfaces/httpserver/  # Gin routes, requests, responses, middlewares
├── migrations/                 # SQL migrations
├── swagger/ or docs/swagger/   # Generated OpenAPI files
├── scripts/                    # Service utilities (optional)
├── Makefile                    # Service-local helpers (e.g., `make gormgen`)
└── go.mod / go.sum             # Module definition
```

> Paths in the other convention documents are relative to `services/<service>/`.

---

## Clean Architecture Layers

```
Interfaces (routes, cron, event consumers)
        |
        v
Domain (entities, services, validation)
        |
        v
Infrastructure (repositories, cache, providers)
```

**Rules:**

- Domain packages only import other domain packages plus injected interfaces (e.g., repository interfaces).
- Infrastructure implements those interfaces and may import external drivers (PostgreSQL, Redis, provider SDKs, etc.).
- Interfaces (HTTP) bind requests to domain services. Do not place business logic in Gin handlers.

---

## File Placement Cheat Sheet

| Task                      | Location                                                             | Example                                                                    |
| ------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| New domain aggregate      | `services/<svc>/internal/domain/<aggregate>/`                        | `services/llm-api/internal/domain/conversation/`                           |
| New HTTP endpoint         | `services/<svc>/internal/interfaces/httpserver/routes/<area>/`       | `services/llm-api/internal/interfaces/httpserver/routes/v1/conversations/` |
| New schema/table          | `services/<svc>/internal/infrastructure/database/dbschema/`          |
| Repository implementation | `services/<svc>/internal/infrastructure/database/repository/<name>/` |
| Cache / provider client   | `services/<svc>/internal/infrastructure/<cache or provider>/`        |
| Shared helper             | `services/<svc>/internal/utils/<category>/`                          |

### Domain Entity Package

```
services/<svc>/internal/domain/<entity>/
├── <entity>.go            # Entity struct + methods
├── service.go             # Business logic / orchestrations
├── filter.go              # Query filters (optional)
└── dto.go                 # Converters if needed
```

### Infrastructure Repository Package

```
services/<svc>/internal/infrastructure/database/
├── dbschema/              # Schema structs + EtoD/DtoE helpers
├── repository/
│   └── <entity>repo/      # Repository implementation
└── gormgen/               # Generated query builders (llm-api)
```

### HTTP Interface Package

```
services/<svc>/internal/interfaces/httpserver/
├── routes/v1/<group>/     # Route registration + handlers
├── requests/<group>/      # Request DTOs + validation
├── responses/<group>/     # Response DTOs
└── middlewares/           # Shared middleware
```

---

## When to Add New Packages

1. **New domain concept** -> create `internal/domain/<concept>` with entity + service.
2. **New transport handler** -> add to `internal/interfaces/httpserver/routes/v1/<area>` and create `requests/` and `responses/` entries as needed.
3. **New persistence logic** -> add schema file under `dbschema/` and repository under `repository/<concept>repo/`. Run `make gormgen` afterwards.
4. **New provider client** -> add package under `internal/infrastructure/<provider>/` and inject through the service constructors.

---

## Anti-Patterns To Avoid

- **Direct DB access from handlers**: always go through domain services.
- **Fat handlers**: route handlers should validate input, call domain services, and return responses---nothing more.
- **Storing business logic in `internal/utils`**: keep helpers generic; domain rules belong in domain services.
- **Creating interfaces "just in case"**: only introduce an interface when multiple implementations exist or tests require it.

---

## Quick Layer Checklist

- Domain packages import only standard library and other domain packages.
- Infrastructure packages never import HTTP routes.
- Requests/responses convert to domain types immediately (`req.ToDomain()` / builders for responses).
- GORM pointer rules enforced in `dbschema/` structs.

See `design-patterns.md` for concrete code samples and `workflow.md` for the commands that keep code generation and testing in sync.
