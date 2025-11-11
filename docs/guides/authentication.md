# Authentication & Gateway

This guide describes the Kong + Keycloak solution that fronts every `/llm/*` request in Jan Server. The implementation uses Kong OSS plugins (`jwt` + `keycloak-apikey`) so the edge accepts Keycloak-issued JWTs or scoped API keys before requests reach the microservices.

## 1. Architectural Overview

- **Kong gateway** (`http://localhost:8000`) is the sole public endpoint for Jan Server. Every API (LLM, Response, Media, MCP, auth) is exposed through Kong routes that perform JWT/API-key validation, rate limiting, request transformation, and header sanitation.
- **Keycloak** (realm `jan`) issues OAuth2/OIDC tokens. Services and Kong both depend on the Keycloak JWKS endpoint (`http://keycloak:8085/realms/jan/protocol/openid-connect/certs`) for signature validation.
- **LLM API** is responsible for guest onboarding, API key lifecycle endpoints, and the `/auth/validate-api-key` callback consumed by the Kong plugin.
- **Custom auth plugin** (`keycloak-apikey`) replaces Kong consumers/credentials in DB-less mode by delegating API key validation to the service layer.

## 2. Kong Authentication Flow

| Plugin | Purpose | Key config |
| ------ | ------- | ---------- |
| `jwt` | Validates Keycloak JWTs | `key_claim_name: iss`, `claims_to_verify: ["exp","nbf"]`, `maximum_expiration: 3600`, `anonymous: kong-anon-jwt`, `secret_is_base64: false`, `run_on_preflight: false` |
| `keycloak-apikey` | Validates API keys via LLM API | `validation_url: http://llm-api:8080/auth/validate-api-key`, `hide_credentials: true`, `validation_timeout: 5000`, `run_on_preflight: false` |
| `request-termination` (anonymous fallback) | Returns 401 when neither plugin runs |

The routes define **OR logic**: requests are accepted if either the JWT or API key plugin succeeds. Kong also injects `X-Auth-Method` (value `jwt` or `apikey`) and user context headers (`X-User-ID`, `X-User-Subject`, `X-User-Email`, `X-User-Username`) so downstream services know who authenticated the call.

### Flowchart

```
Client
  └──> Kong Gateway (`/llm/*`)
        ├── JWT Plugin (Keycloak)
        │     └──> Valid token → Add `X-Auth-Method: jwt`, inject user headers → Upstream
        ├── API Key Plugin (`keycloak-apikey`)
        │     └──> Forward `X-API-Key` to `llm-api/auth/validate-api-key`
        │            └──> LLM API hashes key, consults Keycloak → Valid → Inject headers + `X-Auth-Method: apikey`
        └── Request-termination (fallback) → Return 401
```

## 3. Guest Tokens

- **Endpoint**: `POST /llm/auth/guest-login` exposed through Kong (`/llm/auth/guest-login` route). This endpoint creates a temporary Keycloak user and returns `access_token`, `refresh_token`, and metadata. Guest tokens are meant for quick local testing; they honor rate limits and expire around 5 minutes.
- **Usage**: Include `Authorization: Bearer <token>` on `/v1/*` calls or sent via Kong using `curl -X POST http://localhost:8000/llm/auth/guest-login`. Kong forwards the request to `llm-api` and enforces the auth plugin (JWT may succeed immediately after issuance).
- **Refresh**: Call `/llm/auth/refresh-token` or rely on Kong’s JWT verification for new tokens in production flows.

## 4. API Key Lifecycle

- **Format**: Keys use the `sk_` prefix plus 32 random characters. The shared secret is shown only once (on creation). Services store only the SHA-256 hash inside Keycloak user attributes and PostgreSQL (`api_keys` table from `000001_init_schema.up.sql`).
- **Endpoints** (require JWT auth):
  - `POST /auth/api-keys` – Create a new API key tied to the authenticated user.
  - `GET /auth/api-keys` – List active keys for the calling user.
  - `DELETE /auth/api-keys/{id}` – Revoke a key.
  - `POST /auth/validate-api-key` – Public validation endpoint called by Kong’s plugin.
- **Validation Flow**:
  1. Kong receives `X-API-Key` from the client.
  2. `keycloak-apikey` calls `http://llm-api:8080/auth/validate-api-key`.
  3. LLM API hashes the key, compares it against Keycloak attributes, and responds with user data (or `401` when invalid).
  4. Kong injects user headers and marks the request as authenticated (can now enforce rate limits per consumer).

## 5. Keycloak Integration Notes

- **JWKS**: The Kong `jwt` plugin fetches the Keycloak JWKS manually (no dynamic JWKS refresh). Rotate Keycloak signing keys via a manual Kong restart or redeploy the gateway.
- **Admin API**: Credentials (JWT secrets) live only in the Kong Admin API and are never committed to Git. The gateway does not create consumers dynamically in DB-less mode, which keeps configuration declarative (`kong.yml`).
- **Guest users**: Each guest login request creates a temporary Keycloak user flagged for easy cleanup. Upgrade and refresh flows use the same `jan` realm policies as regular users.

## 6. Environment & Deployment Guidance

- **Overlays**: Use environment-specific Kong overlays (`docker`, `k8s/jan-server/templates`, etc.) to toggle TLS verification (`ssl_verify: false` in development, `true` plus CA bundles in staging/prod).
- **Rate limiting**: Kong enforces per-IP limits at the gateway plus per-consumer bucketed limits where a consumer is resolved either from JWT claims (`iss` → `keycloak-issuer`) or from API key metadata.
- **Plugin loading**: Custom `keycloak-apikey` code lives in `kong/plugins/keycloak-apikey/` (handler + schema + README). Compose mounts `../kong/plugins:/usr/local/kong/plugins:ro` and sets `KONG_PLUGINS: bundled,keycloak-apikey`.
- **Credentials**: The plugin uses `hide_credentials: true` so backend services never see the raw `X-API-Key`.

## 7. Observability & Follow-up

- **Metrics**: Expose plugin-specific stats for auth method usage and failure reasons. Consider adding Redis caching for `validate-api-key` responses to reduce latency.
- **Logging**: Kong logs record which plugin succeeded; look for `X-Auth-Method` in `request-transformer`-injected headers.
- **Tests**: Newman/Postman suites verify `/auth/api-keys`, `/llm/auth/guest-login`, and the `validate-api-key` call. Run `make test-auth` in development.

## 8. Security Hardening Summary

- **Hashed secrets**: API key secrets are hashed with SHA-256 and stored inside Keycloak user attributes to avoid storing plaintext tokens.
- **Single-use visibility**: Keys are shown only once (creation response) to prevent accidental leaks.
- **Fallback response**: `request-termination` returns `401` when neither plugin authenticates, preventing unauthenticated requests from reaching services.
- **Anonymous consumer**: The `kong-anon-jwt` anonymous consumer is configured purely for the OR logic gate; it has no access beyond the gateway.

This document replaces the implementation roadmap formerly captured in `auth-todo.md`. Keep it updated whenever you add authentication routes, adjust Kong plugins, or change Keycloak realms.
