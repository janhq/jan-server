# Security Architecture

## Identity and Access
- **OAuth2/OIDC** via Keycloak (`keycloak/` Dockerfile).
- **Clients** obtain tokens using:
  - Guest endpoint (`POST /auth/guest`) for local testing.
  - OAuth2 code flow (Keycloak realm `jan`) for real users.
- **Services** validate tokens with:
  - `AUTH_ENABLED=true`
  - `AUTH_ISSUER`, `AUTH_AUDIENCE`, `AUTH_JWKS_URL`
- **Service keys**: Media API requires `X-Media-Service-Key`; keep it secret and rotate regularly.
- **Kong plugins**: apply rate limiting, request size limits, and header sanitization at the edge.

## Network Boundaries
- **Public**: Kong (8000) and, optionally, Keycloak admin (8085) when protected.
- **Private**: LLM API (8080), Response API (8082), Media API (8285), MCP Tools (8091), vLLM (8001).
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
- **Media uploads**: requests require `X-Media-Service-Key` and enforce `MEDIA_MAX_BYTES`.
- **Rate limits**: configure Kong plugins per route; Response API also throttles multi-step workflows internally.

## Incident Response
- Capture request IDs from response headers to trace calls across services.
- Use Jaeger + Prometheus dashboards for triage.
