# Security Architecture

## Identity and Access

- **OAuth2/OIDC** via Keycloak (`jan` realm).
- **Kong gateway** (`http://localhost:8000`) protects the `/llm`, `/v1`, `/responses`, `/media`, and `/mcp` route groups using the built-in `jwt` plugin (validating Keycloak tokens) plus the custom `keycloak-apikey` plugin (`X-API-Key: sk_live...` -> `POST /auth/validate-api-key`).
- **JWT validation is pinned, not dynamic**: Kong stores Keycloak's RSA public key and issuer **statically** in `integrations/kong/kong.yml` (`jwt_secrets` with `rsa_public_key` + `key: <issuer>`). There is no dynamic JWKS fetch at the gateway, so rotating Keycloak's signing keys requires updating `kong.yml` and reloading Kong.
- **Clients** obtain tokens using:
- Guest endpoint (`POST /llm/auth/guest-login` via Kong) for quick local access; the LLM API coordinates with Keycloak.
- OAuth2 (code/password/device) flows against the `jan` realm in Keycloak for registered users.
- API keys with the `sk_live` prefix, presented as `X-API-Key`, validated by the `keycloak-apikey` plugin.
- **Services** validate tokens with:
- `AUTH_ENABLED=true`
- `AUTH_ISSUER`, `ACCOUNT`, `AUTH_JWKS_URL`
- **Service auth**: Media API, Response API, and MCP Tools enforce Keycloak-issued JWTs via `AUTH_*` settings and inherit Kong headers when needed.
- **Kong plugins**: besides jwt/apikey, Kong applies rate limiting, request size limits, and header sanitization at the edge to keep unauthenticated traffic out.

## Network Boundaries

- **Public**: Kong (8000) and, optionally, Keycloak admin (8085) when protected.
- **Private**: LLM API (8080), Response API (8082), Media API (8285), MCP Tools (8091), vLLM (8101).
- **MCP network**: SearXNG, Redis, Vector Store, SandboxFusion run on `jan-server_mcp-network` and are not exposed externally.
- **Kubernetes**: use NetworkPolicies to isolate namespaces or rely on service mesh if available.

## Data Protection

- **Databases**: PostgreSQL instances run inside Docker/Kubernetes. Use managed services with TLS for production.
- **S3 credentials**: stored in `.env` or secret stores, mounted into Media API only.
- **jan\_\* identifiers**: act as opaque references; actual S3 URLs are short lived.
- **Logs**: structured JSON, avoid logging secrets (token middleware redacts sensitive headers).

## Secrets Lifecycle

1. Add new variables to `.env.template` with clear comments (a single root `.env` holds all configuration and secrets).
2. Document usage in the relevant service README and the configuration docs.
3. For production, load values from secret managers or Kubernetes secrets instead of `.env`.

## Threat Mitigations

- **JWT validation**: services reject expired or mismatched tokens. Kong validates signatures against the RSA public key pinned in `kong.yml` (static, not a periodic JWKS refresh); rotating Keycloak keys requires editing Kong's config.
- **Tool execution**: SandboxFusion isolates python code; `SANDBOX_FUSION_REQUIRE_APPROVAL` can force manual approval.
- **Web fetches**: SearXNG provides result filtering; Response API enforces depth/time budgets.
- **Media uploads**: requests require a Bearer token plus `MEDIA_MAX_BYTES`/content-type validation before accepting bytes.
- **Rate limits**: configure Kong plugins per route; Response API also throttles multi-step workflows internally.

## Incident Response

- Capture request IDs from response headers to trace calls across services.
- Use Jaeger + Prometheus dashboards for triage.
