# Security Architecture

## Identity and Access
- **OAuth2/OIDC** via Keycloak (`keycloak/` Dockerfile).
- **Kong gateway** (`http://localhost:8000`) protects every `/llm/*` route using the built-in `jwt` plugin (validating Keycloak tokens) plus the custom `keycloak-apikey` plugin (`X-API-Key: sk_*` -> `POST /auth/validate-api-key`).
- **Clients** obtain tokens using:
  - Guest endpoint (`POST /llm/auth/guest-login` via Kong) for quick local access; the LLM API coordinates with Keycloak.
  - OAuth2 (code/password/device) flows against the `jan` realm in Keycloak for registered users.
- **Services** validate tokens with:
  - `AUTH_ENABLED=true`
  - `AUTH_ISSUER`, `AUTH_AUDIENCE`, `AUTH_JWKS_URL`
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
- **jan_* identifiers**: act as opaque references; actual S3 URLs are short lived.
- **Logs**: structured JSON, avoid logging secrets (token middleware redacts sensitive headers).

## Secrets Lifecycle
1. Add new variables to `.env.template` with clear comments.
2. Mirror them in `config/secrets.env.example`.
3. Document usage in `config/README.md` and relevant service README.
4. For production, load values from secret managers or Kubernetes secrets instead of `.env`.

## Threat Mitigations
- **JWT validation**: services reject expired or mismatched tokens and refresh their JWKS cache periodically.
- **Tool execution**: SandboxFusion isolates python code; `SANDBOX_FUSION_REQUIRE_APPROVAL` can force manual approval.
- **Web fetches**: SearXNG provides result filtering; Response API enforces depth/time budgets.
- **Media uploads**: requests require a Bearer token plus `MEDIA_MAX_BYTES`/content-type validation before accepting bytes.
- **Rate limits**: configure Kong plugins per route; Response API also throttles multi-step workflows internally.

## Incident Response
- Capture request IDs from response headers to trace calls across services.
- Use Jaeger + Prometheus dashboards for triage.
