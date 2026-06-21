# Jan Server — New Owner Handover & Setup Guide

> Purpose: everything a new owner needs to take over **Jan Server** — from a fresh clone to a running stack, which environment/secrets to use, how login & admin work, how it's operated, and the open questions to resolve during handover.
>
> Audience: a developer on **Windows 11** inheriting ownership. Commands are Windows-first; macOS/Linux equivalents are noted where they differ.
> Last written: 2026-06-19. Cross-check anything marked ⚠️ against the live code — parts of `docs/` are stale (see [§13 Doc drift](#13-known-doc-drift--gotchas)).

---

## Table of contents
1. [What this project is](#1-what-this-project-is)
2. [Prerequisites](#2-prerequisites)
3. [Fastest path: `make quickstart`](#3-fastest-path-make-quickstart)
4. [Manual setup (the explicit path)](#4-manual-setup-the-explicit-path)
5. [Which environment / `.env` to use](#5-which-environment--env-to-use)
6. [Secrets & accounts to obtain or rotate](#6-secrets--accounts-to-obtain-or-rotate)
7. [Google login (Keycloak IdP)](#7-google-login-keycloak-identity-provider)
8. [Admin access](#8-admin-access)
9. [Services & ports](#9-services--ports)
10. [Daily operations](#10-daily-operations)
11. [Testing](#11-testing)
12. [Architecture & auth flow](#12-architecture--auth-flow-quick-mental-model)
13. [Known doc drift & gotchas](#13-known-doc-drift--gotchas)
14. [Deployment & production ownership](#14-deployment--production-ownership-read-this)
15. [Handover checklist](#15-handover-checklist)
16. [Key docs index](#16-key-docs-to-read-next)

---

## 1. What this project is

Jan Server is an enterprise-grade, microservices **LLM API platform** with Model Context Protocol (MCP) tool integration. It exposes OpenAI-compatible APIs, multi-step tool orchestration, media management, and full observability.

- **Monorepo**: Go backend services (modules) + a React/Vite web app (pnpm/npm workspace).
- **Architecture**: Clean Architecture, services behind a Kong gateway, Keycloak for auth.
- The single best in-repo overview is **`CLAUDE.md`** at the repo root — read it alongside this guide.

---

## 2. Prerequisites

For a **Docker-only** bring-up on Windows 11 you only need the first three:

| Tool | Version | Needed for | Notes |
|------|---------|-----------|-------|
| **Docker Desktop** | 24+ (Compose v2) | Everything | The whole stack runs in containers. |
| **GNU Make** | any recent | Running `make` targets | **Not bundled with Windows** — `choco install make` or the GnuWin32 installer (README link). |
| **Git** | any | Cloning | |
| Go | **1.24+** (per `go.mod`; toolchain 1.24.7) | Native/hybrid runs, editing Go, `make swagger`/`make test-*`, building `jan-cli` | ⚠️ Docs that say "1.21+" are stale. Not needed for a pure Docker run. |
| Node.js + pnpm (9.15.4) | — | Running the web app *outside* Docker only | The Docker path needs neither. |
| NVIDIA GPU + drivers | — | Local vLLM inference only | Skip by using a remote LLM provider. |

RAM: 8 GB min, 12 GB recommended.

> You do **not** need WSL — every OS-divergent `make` target already branches to PowerShell on Windows.

---

## 3. Fastest path: `make quickstart`

```powershell
git clone https://github.com/janhq/server.git jan-server
cd jan-server
make quickstart
```

`make quickstart` runs the interactive wizard (`tools/jan-cli.ps1 setup-and-run` on Windows). It copies `.env.template` → `.env` (if missing) and walks these prompts. **Recommended answers for a Windows machine with no GPU:**

| # | Prompt | Recommended answer |
|---|--------|--------------------|
| 1 | (only if `.env` exists) `Do you want to update it? (y/N)` | `N` to keep, `y` to re-run the wizard |
| 2 | LLM provider: `1. Local vLLM / 2. Remote API` | **`2` (Remote)** — then enter an OpenAI-compatible URL + API key. (`1` needs a GPU + `HF_TOKEN`.) |
| 3 | Search provider: `1. Serper / 2. SearXNG / 3. None` | **`2` (SearXNG, local, no key)** — or `1` if you have a [serper.dev](https://serper.dev) key |
| 4 | Media API / storage | **Local file system** (no credentials). Only pick S3 if you have bucket creds. |
| 5 | Sandbox provider: `1. None / 2. AIO / 3. E2B` | **`1` (None)** unless you need code-execution tools |
| 6 | `Set up monitoring? (y/N)` | `N` (enable later with `make monitor-up`) |
| 7 | `Start Platform app? (y/N)` | **`N`** — ⚠️ the platform app was removed from the codebase |

The wizard then runs `make setup` and `make up-full`, waits ~30s, and prints access URLs. First run takes 1–2 minutes (image pulls/builds).

> ⚠️ **What we fixed this session** so quickstart actually completes: the web image build (TanStack route tree) and Keycloak login (missing OIDC scopes, Google IdP, token-exchange CORS). Those fixes are in PR [#494](https://github.com/janhq/server/pull/494) — make sure that's merged, or you'll hit the same failures on a clean machine.

---

## 4. Manual setup (the explicit path)

If you'd rather not use the wizard:

```powershell
git clone https://github.com/janhq/server.git jan-server
cd jan-server

make setup            # copies .env.template -> .env, creates docker networks + dirs
#  (or copy it yourself:  copy .env.template .env  )

# edit .env  (see §5 and §6 below)

make up-full          # docker compose up -d, honoring COMPOSE_PROFILES from .env
make health-check     # verify everything is healthy
```

Start subsets instead of the whole stack:

| Target | Brings up |
|--------|-----------|
| `make up-infra` | PostgreSQL (5432), Keycloak (8085), Kong (8000) |
| `make up-api` | LLM API (8080), Media API (8285), Response API (8082) |
| `make up-mcp` | MCP Tools (8091) |
| `make up-web` | Web app (3001) + infra |
| `make up-full` | Everything in `COMPOSE_PROFILES` |

### The `jan-cli` tool
A Go CLI under `tools/jan-cli/` that the Makefile delegates to. Invoke it directly with:
- **Windows**: `.\tools\jan-cli.ps1 <cmd>` &nbsp; (e.g. `.\tools\jan-cli.ps1 service logs llm-api --follow`)
- **macOS/Linux**: `./tools/jan-cli.sh <cmd>`

The wrappers auto-build the binary (needs Go). `make cli-install` puts `jan-cli` on your PATH. Useful subcommands: `service list|logs|status`, `dev setup|run <svc>|scaffold <name>`, `config validate|export`, `monitor up`, `swagger generate`, `api-test run`.

> ⚠️ The wrapper scripts live under **`tools/`**, not the repo root. README/CLAUDE.md shorthand like `./jan-cli.sh` and references to a `scripts/` directory are inaccurate — there is no `scripts/` dir; scaffold new services with `jan-cli dev scaffold`.

---

## 5. Which environment / `.env` to use

**There is one real env file: the root `.env`**, created from `.env.template` by `make setup`/`make quickstart`. Services receive it two ways (see `infra/docker/*.yml`): `env_file: ../../.env` loads the *entire* file into each container, and an `environment:` block re-maps key vars with `${VAR:-default}` fallbacks.

> ⚠️ `.env.template` and `config/README.md` mention `make env-create` / `make env-switch ENV=hybrid` and per-env files (`config/development.env`, `config/hybrid.env`, …). **Those targets/files do not exist** in the current repo. Ignore them — use `make setup` / `make quickstart` and edit the single `.env`.

### `COMPOSE_PROFILES` decides which services start
Comma-separated, set in `.env` (and respected automatically by `docker compose`):

| Profile | Services |
|---------|----------|
| `infra` | PostgreSQL, Keycloak (+ its DB), Kong |
| `api` | llm-api, media-api, response-api |
| `mcp` | mcp-tools |
| `web` | web app (chat UI, :3001) |
| `full` | superset **including GPU vLLM** (`vllm-jan-gpu`) |
| `gpu` / `cpu` | vLLM GPU / CPU only |
| `aio` / `e2b` | code-execution sandboxes |

**Recommended profile strings:**
- **Local dev on Windows, no GPU (use a remote LLM):** `COMPOSE_PROFILES=infra,api,mcp,web`
- **Full local with GPU inference:** `COMPOSE_PROFILES=infra,api,mcp,web,full`

> ⚠️ The shipped `.env.template` default is `infra,api,mcp,full` (assumes a GPU). On a no-GPU machine, drop `full`, set `VLLM_ENABLED=false`, and configure a remote provider: `REMOTE_LLM_ENABLED=true` + `REMOTE_LLM_PROVIDER_URL` + `REMOTE_API_KEY` (or use the wizard's "Remote" option).

`media-api` defaults to `MEDIA_STORAGE_BACKEND=local`, so **no S3 credentials are needed for local dev**.

---

## 6. Secrets & accounts to obtain or rotate

`.env` is **git-ignored** (`.gitignore` whitelists only `*.env.example`/`*.env.template`). Real secrets live only in your local `.env`; for shared/prod use a secret manager (see [§14](#14-deployment--production-ownership-read-this)).

### Needed to make features work locally (by profile)
| Variable | Get it from | Needed when |
|----------|-------------|-------------|
| `REMOTE_API_KEY` (+ `REMOTE_LLM_PROVIDER_URL`) | your LLM provider | Using a remote model (no GPU) |
| `HF_TOKEN` | huggingface.co/settings/tokens | Local vLLM only (`full`/`gpu`/`cpu`) |
| `SERPER_API_KEY` (+ `SERPER_ENABLED=true`) | serper.dev | Serper web-search tool (else use SearXNG) |
| `GOOGLE_IDP_CLIENT_ID` / `GOOGLE_IDP_CLIENT_SECRET` | Google Cloud Console | "Sign in with Google" — see [§7](#7-google-login-keycloak-identity-provider) |

### Ship with weak defaults — **rotate before any shared/non-local use**
`POSTGRES_PASSWORD` (`jan_password`), `KEYCLOAK_ADMIN_PASSWORD` (`admin`), `BACKEND_CLIENT_SECRET` (`backend-secret`), `MODEL_PROVIDER_SECRET`, `VLLM_INTERNAL_KEY` (`changeme`), `MEDIA_SERVICE_KEY`/`MEDIA_API_KEY` (`changeme-media-key`), `GRAFANA_ADMIN_PASSWORD` (`admin`). Generate with `openssl rand -base64 32` (hex: `openssl rand -hex 32`).

### Only if you enable connectors (Gmail/Drive/GitHub data access)
`GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` **(distinct from the IdP ones above — different purpose!)**, `GITHUB_CLIENT_ID`/`SECRET`, plus `CONNECTOR_TOKEN_ENCRYPTION_KEY` (32-byte hex) and `OAUTH_STATE_SECRET` (32-byte hex).

> 🔒 **Security flag for handover:** `.env.template` lines ~338–339 contain commented-out but **real-looking S3 access key + secret** (`MEDIA_S3_ACCESS_KEY_ID=7N33WPTUI1KN99MFILQS`, `MEDIA_S3_SECRET_ACCESS_KEY=…`). Since the template **is** committed, verify these are dead/rotated and scrub them. (I can do this on request.)

---

## 7. Google login (Keycloak identity provider)

The web app's "Login / Register" buttons send `kc_idp_hint=google`, so a Google IdP must exist in the `jan` realm. This is now wired into the realm import (PR #494):

1. In [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials), create an **OAuth client (Web application)**.
2. Add **Authorized redirect URI** exactly: `http://localhost:8085/realms/jan/broker/google/endpoint`
3. If the consent screen is in **Testing** mode, add each user's Google account as a **test user**.
4. Put the credentials in `.env`:
   ```
   GOOGLE_IDP_CLIENT_ID=...apps.googleusercontent.com
   GOOGLE_IDP_CLIENT_SECRET=GOCSPX-...
   ```
5. Keycloak substitutes `${GOOGLE_IDP_CLIENT_ID}` / `${GOOGLE_IDP_CLIENT_SECRET}` into `integrations/keycloak/import/realm-jan.json` **at import time** — so no secret is committed.

**Realm import only runs when the realm doesn't exist.** To apply realm/IdP changes to a running stack, reset just the Keycloak DB:
```powershell
docker compose rm -sf keycloak keycloak-db
docker volume rm jan-server_keycloak-db-data
docker compose up -d keycloak-db keycloak
```
Guest login (`POST /llm/auth/guest-login`) works without Google.

---

## 8. Admin access

"Admin" = the Keycloak **realm role `admin`** (the app checks `realm_access.roles`). To grant it to a user who has already logged in once (so the user exists in Keycloak):

```powershell
# get an admin token, find the user, assign the 'admin' realm role
# (replace the email). Uses Keycloak Admin API on :8085 (admin/admin in dev)
```
Easiest via the **Keycloak admin console** → http://localhost:8085 (admin/admin) → realm **jan** → Users → *user* → Role mapping → assign `admin`. Or the app's own admin UI once you have one admin.

> ⚠️ The role is embedded in the JWT, so the user must **sign out and sign back in** for admin to take effect — a server restart does **not** refresh existing tokens.

Example done this session: `nguyen@jan.ai` was granted `admin` directly in the running Keycloak (lives in the `keycloak-db` volume; not committed). If you want fresh setups to auto-grant an admin, ask and I'll add a configurable bootstrap (`KEYCLOAK_BOOTSTRAP_ADMIN_EMAILS`) or seed it in the realm import.

---

## 9. Services & ports

| Service | Port | Responsibility |
|---------|------|----------------|
| **Kong** (gateway) | **8000** | Sole public entry point; JWT/API-key auth, rate limiting, routing |
| LLM API | 8080 | OpenAI-compatible chat completions; conversations; **all auth endpoints**; model catalog |
| Response API | 8082 | Multi-step tool orchestration (calls MCP Tools, then LLM API) |
| Media API | 8285 | Upload/storage, opaque `jan_*` IDs → presigned URLs |
| MCP Tools | 8091 | MCP JSON-RPC: web search (Serper→Exa→Tavily→SearXNG), scrape, code exec |
| Web app | 3001 | React/Vite chat UI |
| Keycloak | 8085 | OIDC, realm `jan` (admin/admin in dev) |
| PostgreSQL (app) | 5432 | App data (pgvector) |
| PostgreSQL (keycloak) | 5433 | Keycloak's own DB |
| Grafana / Prometheus / Jaeger | 3331 / 9090 / 16686 | After `make monitor-up` |

> ⚠️ Ignore **Memory Tools (8090)**, **Realtime API (8186)**, and **Platform app** in older docs — those were removed from the codebase.
> ⚠️ `GRAFANA_PORT` in `.env.template` is `3001` (collides with the web app); docs/tooling use **3331**. Verify before enabling monitoring next to the web app.

---

## 10. Daily operations

```powershell
make up-full           # start everything (per COMPOSE_PROFILES)
make health-check      # probe all services
make logs              # all logs   (also: make logs-api / logs-mcp)
docker compose ps      # status

make stop              # stop containers (fastest restart; keeps everything)
make down              # remove containers, KEEP volumes/data
make down-clean        # remove containers AND volumes (full wipe)
make restart-api / restart-keycloak / restart-kong   # targeted restarts
```

Access after start: app **http://localhost:3001**, gateway **http://localhost:8000**, Swagger **http://localhost:8000/api/swagger/index.html**, Keycloak **http://localhost:8085**.

Quick end-to-end smoke test:
```bash
curl -X POST http://localhost:8000/llm/auth/guest-login        # -> token
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"model":"<your-model>","messages":[{"role":"user","content":"Hello!"}]}'
```

---

## 11. Testing

```powershell
make test-all          # all integration suites (Postman collections via jan-cli)
make test-auth         # guest login, API keys, token refresh
make test-conversation # conversations
make test-response     # response API
make test-media        # media API
make test-mcp          # MCP tools
go test ./services/<svc>/...   # Go unit tests
```
Full Docker integration tests are expected to run on **Ubuntu CI**; Windows CI focuses on build + CLI verification.

---

## 12. Architecture & auth flow (quick mental model)

```
Browser / API client
        │
        ▼
   Kong  :8000   ──(JWT  OR  X-API-Key)──>  validates against Keycloak / llm-api
        │  routes:  /v1/* /llm/* -> llm-api      /responses/* -> response-api
        │           /media/*     -> media-api    /mcp         -> mcp-tools
        ▼
   Go services (Gin) ── share one PostgreSQL ── Keycloak issues/validates JWTs
```

- **Two auth methods**, OR-gated per route: Keycloak **JWT** (RS256) or **`X-API-Key: sk_…`** (validated by llm-api). Kong injects `X-User-*` headers downstream.
- ⚠️ Kong pins Keycloak's **RSA public key + issuer statically** in `integrations/kong/kong.yml` — there's **no dynamic JWKS refresh**, so rotating Keycloak signing keys requires editing/redeploying Kong.
- Guest users are real (temporary) Keycloak users (`guest-…@temp.jan.ai`); they can be upgraded via `POST /auth/upgrade`.

---

## 13. Known doc drift & gotchas

The `docs/` tree was last reviewed ~Nov 2025 and is partly stale. **Trust code/config over prose.** Specifics:

- **Removed services** still described in docs: memory-tools, realtime-api, platform app, slide_creator agent.
- **Wrapper scripts** are at `tools/jan-cli.{ps1,sh}`, not repo root; **no `scripts/` dir** (use `jan-cli dev scaffold`).
- **Go version** is 1.24+ (per `go.mod`), not 1.21.
- **`make env-create`/`env-switch`** and `config/<env>.env` files don't exist — single root `.env` only.
- **Grafana port** collision (`.env.template` 3001 vs docs 3331).
- **Committed-looking S3 creds** in `.env.template` (§6) — scrub.
- **This session's fixes** (PR #494) were required for `make quickstart` to complete at all: web route-tree build, Keycloak OIDC scopes, Google IdP, token-exchange CORS. Confirm it's merged.

---

## 14. Deployment & production ownership (read this)

This is the **biggest knowledge gap to close during handover.**

- **CI (`.github/workflows/`) is build-and-push only.** There is **no automated deploy**, no GitOps (no Argo CD/Flux), no `helm upgrade` in any workflow. A human deploys after images are pushed.
- **Backend images** → private registry **`registry.menlo.ai`** (`jan-server/<svc>:dev-<sha>` / `:prod-<tag>`).
- **Web app prod is NOT a container** — `ci-production-release.yml` deploys it to **Cloudflare Pages** (project `jan-server-web`; prod endpoints api.jan.ai / chat.jan.ai / auth.menlo.ai).
- **Release tags**: `vX.Y.Z` (full), `…NN1` (backend-only), `…NN2` (web-only), `vX.Y.Z-interfaces` (npm publish of `@janhq/interfaces`).
- **Kubernetes/Helm** chart exists at `infra/k8s/jan-server/` (Bitnami Postgres + Redis deps) but is **manual and incompletely configured** (`values-production.yaml` has `CHANGE_ME_*` placeholders; secrets/extra DBs created by hand; no rollback/verify runbook) and **not wired into CI**.
- **CI secrets used**: `REGISTRY_USERNAME/PASSWORD`, `DOCKERHUB_*`, `CLOUDFLARE_API_TOKEN`/`ACCOUNT_ID`, `POSTHOG_API_KEY`, `NPM_TOKEN`.

**❓ Confirm with the current team before you own prod:** which cluster/host actually runs the production **backend**, who deploys it, how secrets are injected there, and where DB backups live. None of that is captured in the repo.

---

## 15. Handover checklist

- [ ] Get added to / take ownership of: the GitHub org (`janhq/server`), the **`registry.menlo.ai`** registry, **Cloudflare** account (Pages project `jan-server-web`), Google Cloud project (OAuth clients), and any LLM-provider account.
- [ ] Collect/rotate all secrets in [§6](#6-secrets--accounts-to-obtain-or-rotate); confirm where prod secrets are stored.
- [ ] Confirm the **production backend deployment** target & process ([§14](#14-deployment--production-ownership-read-this)).
- [ ] Verify/scrub the S3 creds in `.env.template`.
- [ ] Get a clean machine through `make quickstart` end-to-end (depends on PR #494 being merged).
- [ ] Run `make test-all` and confirm CI is green.
- [ ] Review the Dependabot alerts on the repo's default branch (a large count was flagged on push).
- [ ] Decide on an admin-bootstrap mechanism if fresh environments need a default admin.

---

## 16. Key docs to read next

| Topic | Path |
|-------|------|
| Best single overview | `CLAUDE.md` |
| System design | `docs/architecture/system-design.md` |
| Services | `docs/architecture/services.md` |
| Data/request flows | `docs/architecture/data-flow.md` |
| Auth (Kong + Keycloak + API keys) | `docs/guides/authentication.md` |
| Development (3 dev modes, hybrid debug) | `docs/guides/development.md` |
| Endpoint inventory | `docs/api/endpoint-matrix.md` |
| Testing | `docs/guides/testing.md` |
| Observability | `docs/architecture/observability.md`, `docs/guides/monitoring.md` |
| Kubernetes/Helm (manual prod path) | `infra/k8s/README.md`, `infra/k8s/SETUP.md` |
| Conventions / patterns | `docs/conventions/`, `AGENTS.md` |

Ground-truth config files (trust these over prose): `integrations/kong/kong.yml`, `docker-compose.yml` + `infra/docker/*.yml`, `integrations/keycloak/import/realm-jan.json`, `integrations/monitoring/prometheus.yml`, `.github/workflows/`.
