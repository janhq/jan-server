# Repository Naming Conventions

> Naming standards for the Jan monorepo structure

---

## 1. Directory Naming

### 1.1 Top-Level Directories

Use lowercase with hyphens for multi-word names:

```
server/
├── apps/              # Frontend applications
├── services/          # Backend microservices
├── packages/          # Shared libraries
├── infra/             # Infrastructure as Code
├── tools/             # Development tools
├── integrations/      # Third-party integrations
├── docs/              # Documentation
└── tests/             # Test suites
```

**Rules:**

- Use descriptive, plural nouns where appropriate (`apps`, `services`, `packages`)
- Single word preferred (`tools`, `docs`)
- Hyphenate multi-word directories (`infra-scripts` if needed)
- No underscores in directory names

### 1.2 Application Directories

Frontend apps in `apps/` use kebab-case:

```
apps/
├── web/               # Main web application
├── admin/             # Admin dashboard
├── mobile/            # Mobile app (future)
└── chrome-extension/  # Browser extension (example)
```

**Rules:**

- Lowercase only
- Hyphenate multi-word apps (`chrome-extension`, `user-portal`)
- Descriptive and concise
- Avoid abbreviations unless widely known

### 1.3 Service Directories

Backend services in `services/` use kebab-case with suffix pattern:

```
services/
├── llm-api/           # LLM API service
├── mcp-tools/         # MCP tools service
├── media-api/         # Media API service
├── memory-tools/      # Memory tools service
├── realtime-api/      # Realtime API service
└── response-api/      # Response API service
```

**Rules:**

- Lowercase kebab-case
- End with `-api` or `-tools` or `-service` for clarity
- Match Docker image naming
- Consistent across documentation

### 1.4 Package Directories

Shared packages in `packages/` use kebab-case with prefix pattern:

```
packages/
├── shared-ui/         # Shared React components
├── shared-types/      # TypeScript type definitions
├── shared-utils/      # Common utilities
├── shared-config/     # Shared configuration
└── go-common/         # Shared Go packages
```

**Rules:**

- Prefix with `shared-` for cross-app packages
- Language-specific packages use language name (`go-common`, `py-utils`)
- Descriptive suffix (`-ui`, `-types`, `-utils`)

---

## 2. File Naming

### 2.1 Configuration Files

```
.env                   # Root environment file
.env.example           # Example environment template
.gitignore             # Git ignore rules
docker-compose.yml     # Main compose file
pnpm-workspace.yaml    # pnpm workspace config
turbo.json             # Turborepo config (if used)
```

**Rules:**

- Lowercase
- Use standard config file names
- Hyphenate multi-word configs (`docker-compose.yml`)

### 2.2 Documentation Files

```
docs/
├── README.md          # Documentation index
├── quickstart.md      # Getting started guide
├── repo-naming.md     # This document
└── api/
    ├── README.md
    └── llm-api.md
```

**Rules:**

- Lowercase kebab-case with `.md` extension
- `README.md` (uppercase) for index files
- Descriptive names (`authentication.md`, `deployment-guide.md`)
- Match directory structure where relevant

### 2.3 Source Code Files

**TypeScript/JavaScript:**

- Components: `PascalCase.tsx` (e.g., `UserProfile.tsx`)
- Utilities: `camelCase.ts` (e.g., `formatDate.ts`)
- Hooks: `camelCase.ts` with `use` prefix (e.g., `useAuth.ts`)
- Types: `PascalCase.types.ts` (e.g., `User.types.ts`)

**Go:**

- Files: `snake_case.go` (e.g., `cmd_setup.go`, `utils.go`)
- Packages: lowercase single word (e.g., `config`, `telemetry`)

---

## 3. Docker & Infrastructure

### 3.1 Docker Compose Files

```
infra/docker/
├── infrastructure.yml    # Core infrastructure
├── services-api.yml      # API services
├── services-mcp.yml      # MCP services
├── services-memory.yml   # Memory services
├── services-realtime.yml # Realtime services
└── observability.yml     # Monitoring stack
```

**Pattern**: `{category}.yml` or `services-{type}.yml`

**Rules:**

- Lowercase kebab-case
- Group by function/category
- Prefix with `services-` for service definitions
- Single word for infrastructure types

### 3.2 Docker Images

**Format**: `registry.domain.com/server/{service}:{tag}`

**Examples:**

- `registry.menlo.ai/server/llm-api:dev-abc123`
- `registry.menlo.ai/server/mcp-tools:prod-v1.2.3`

**Rules:**

- Use kebab-case for service names
- Tag format: `{env}-{identifier}` (e.g., `dev-sha`, `prod-v1.0.0`)

### 3.3 Kubernetes Resources

```
infra/k8s/
├── namespace.yaml
├── llm-api-deployment.yaml
├── mcp-tools-service.yaml
└── ingress.yaml
```

**Pattern**: `{resource-name}-{type}.yaml` or `{type}.yaml`

**Rules:**

- Lowercase kebab-case
- Suffix with resource type (`-deployment`, `-service`, `-configmap`)
- Standalone types use singular (`namespace.yaml`, `ingress.yaml`)

---

## 4. Git Branch Naming

### 4.1 Branch Types

**Format**: `{type}/{scope}-{description}` or `{type}/{description}` (for cross-cutting changes)

**Types:**

- `feat/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation updates
- `chore/` - Maintenance tasks
- `test/` - Test additions/fixes

**Scopes** (optional but recommended for monorepo):

- `web` - Web app changes
- `admin` - Admin app changes
- `platform` - Platform app changes
- `services` - Backend services (general)
- `llm-api`, `mcp-tools`, etc. - Specific service
- `shared` - Shared packages
- `infra` - Infrastructure changes

**Examples:**

_With scope (recommended for monorepo):_

- `feat/web-user-authentication` - New auth feature in web app
- `fix/llm-api-memory-leak` - Bug fix in LLM API service
- `feat/admin-dashboard-redesign` - Admin app feature
- `refactor/shared-refactor-types` - Shared package refactoring
- `chore/infra-update-k8s-configs` - Infrastructure update

_Without scope (for cross-cutting changes):_

- `docs/update-api-guide` - Documentation affecting multiple services
- `chore/upgrade-dependencies` - Dependency updates across monorepo
- `refactor/monorepo-structure` - Structural changes

### 4.2 Environment Branches

- `main` - Main development branch (triggers dev deployments)
- `dev-test` - Testing branch (triggers dev deployments)
- `release` - Production branch (triggers prod deployments)

---

## 5. Package Naming

### 5.1 NPM Packages

**Format**: `@jan/{package-name}`

**Examples:**

- `@jan/shared-ui`
- `@jan/shared-types`
- `@jan/shared-utils`

### 5.2 Docker Images

**Format**: `server/{service-name}:{tag}`

**Examples:**

- `server/llm-api:latest`
- `server/mcp-tools:dev-abc123`

### 5.3 Go Modules

**Format**: `github.com/janhq/server/{path}`

**Examples:**

- `github.com/janhq/server/tools/jan-cli`
- `github.com/janhq/server/pkg/config`

---

## 6. GitHub Actions Workflows

### 6.1 Workflow Structure

All workflows live in `.github/workflows/` with the following organization:

```
.github/workflows/
├── ci-backend-dev.yml           # Backend services (dev/main branches)
├── ci-backend-prod.yml          # Backend services (release branch)
├── ci-app-web-dev.yml           # Web app dev/main
├── ci-app-web-prod.yml          # Web app release
├── ci-app-admin-dev.yml         # Admin app dev/main
├── ci-app-admin-prod.yml        # Admin app release
├── ci-app-platform-dev.yml      # Platform app dev/main
├── ci-app-platform-prod.yml     # Platform app release
├── ci-packages.yml              # Shared packages validation
├── config-drift.yml             # Config validation
└── _reusable-docker.yml         # Reusable Docker build template
```

### 6.2 Naming Pattern

**Format**: `ci-{component}-{environment}.yml`

**Components:**

- `backend` - Backend microservices
- `app-{name}` - Frontend applications (web, admin, platform)
- `packages` - Shared packages/libraries
- Component-specific names (e.g., `config-drift`)

**Environments:**

- `dev` - Development/staging (triggers on `main`, `dev-test` branches)
- `prod` - Production (triggers on `release` branch)
- No suffix - Environment-agnostic validation workflows

**Special Prefixes:**

- `_` (underscore) - Reusable workflow templates (e.g., `_reusable-docker.yml`)

### 6.3 Examples

#### ✅ Correct:

- `ci-backend-dev.yml` - Backend services for dev environment
- `ci-app-web-prod.yml` - Web app production deployment
- `ci-app-checkout-dev.yml` - New checkout app dev workflow
- `ci-packages.yml` - Shared packages validation
- `_reusable-node-build.yml` - Reusable Node.js build template

#### ❌ Incorrect:

- `dev.yml` - Too generic, unclear what it builds
- `web-app-ci.yml` - Wrong order, should be `ci-app-web-*.yml`
- `ci-web.yml` - Missing `app-` prefix for frontend apps
- `backend.yml` - Missing `ci-` prefix and environment suffix
- `reusable-docker.yml` - Missing underscore prefix

### 6.4 Adding New App Workflows

When adding a new frontend app (e.g., `mobile`), create:

```
.github/workflows/
├── ci-app-mobile-dev.yml        # Mobile app dev/main
└── ci-app-mobile-prod.yml       # Mobile app release
```

**Template for dev workflow:**

```yaml
name: CI - Mobile App (Dev)

on:
  push:
    branches: [main, dev-test]
    paths:
      - "apps/mobile/**"
      - "packages/shared-ui/**"
      - "packages/shared-types/**"
      - "packages/shared-utils/**"
  pull_request:
    paths:
      - "apps/mobile/**"

jobs:
  build-and-test:
    # ... implementation
```

---

## 7. Quick Reference

### Adding a New Frontend App

1. **Directory**: `apps/new-app/`
2. **Workflows**:
   - `ci-app-new-app-dev.yml`
   - `ci-app-new-app-prod.yml`
3. **Package**: `@jan/new-app`
4. **Branch**: `feat/add-new-app`

### Adding a New Backend Service

1. **Directory**: `services/new-service/`
2. **Workflow**: Include in `ci-backend-dev.yml` and `ci-backend-prod.yml`
3. **Docker**: `server/new-service:tag`
4. **Compose**: Add to `infra/docker/services-api.yml`

### Adding a New Shared Package

1. **Directory**: `packages/shared-new/`
2. **Workflow**: Include in `ci-packages.yml`
3. **NPM**: `@jan/shared-new`
4. **Workspace**: Auto-detected by `pnpm-workspace.yaml`

---

## 8. Validation Checklist

Before committing new files/directories, verify:

- [ ] Names follow kebab-case (lowercase with hyphens)
- [ ] Workflow files use `ci-{component}-{env}.yml` pattern
- [ ] Reusable workflows prefixed with underscore
- [ ] Directories use plural nouns where appropriate
- [ ] No spaces or special characters (except `-` and `_`)
- [ ] Consistent with existing naming patterns
- [ ] Git branch follows `{type}/{description}` format
- [ ] Package names use organization scope (`@jan/`)

---

## 9. Migration Guide

When migrating an existing app into the monorepo:

1. **Rename directory** to match `apps/` or `services/` convention
2. **Create workflows** following `ci-app-{name}-{env}.yml` pattern
3. **Update package.json** name to `@jan/{name}` if applicable
4. **Add to workspace** in `pnpm-workspace.yaml` (auto if in `apps/`)
5. **Update imports** to use monorepo package references
6. **Document** in `docs/` with matching kebab-case filename

---

## References

- [Monorepo Best Practices](https://monorepo.tools/)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Actions Naming Best Practices](https://docs.github.com/en/actions/learn-github-actions/workflow-syntax-for-github-actions)

---

**Questions?** Ask in #engineering or open an issue with the `documentation` label.
