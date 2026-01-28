# API Test Collections (jan-cli api-test)

Domain-scoped Postman collections for `jan-cli api-test`. Collections rely on `--auto-auth` and `--auto-models` flags; no auth bootstrap requests are required.

## Collections

- `collections/auth.postman.json` – auth flows (guest/admin) + full legacy scenarios.
- `collections/conversation.postman.json` – conversation flows (create/chat/list/delete) + legacy scenarios.
- `collections/model.postman.json` – model listing plus admin/model-management scenarios; prompt templates in `collections/model-prompt-templates.postman.json`.
- `collections/response.postman.json` – response service scenarios (health/create/fetch plus legacy cases).
- `collections/media.postman.json` – media health and legacy media flows.
- `collections/mcp-runtime.postman.json` – MCP runtime tooling flows.
- `collections/mcp-admin.postman.json` – public/admin MCP tooling.
- `collections/mcp-agent.postman.json` - MCP agent + sandbox tool validation.
- `collections/user-management.postman.json` – user management scenarios.
- `collections/model-prompt-templates.postman.json` – prompt template scenarios.
- `collections/messages.postman.json` – Anthropic Messages API compatibility (Claude Code API).

## Running

- `make test-all` – runs the core collections (memory/response excluded for now).
- `make test-<domain>` – run a single collection (`auth`, `conversation`, `response`, `model`, `memory`, `media`, `mcp`, `dev` fail-fast).
- `make test-mcp-agents` - run MCP agent sandbox validation (E2B).

## Variables

- Canonical: `gateway_url`, `kong_url`, `memory_url`, `media_url`, `mcp_url`, `embedding_url`, `keycloak_base_url`, `keycloak_admin`, `keycloak_admin_password`.
- Defaults live in `tests/automation/.env`; CLI flags can override (see Makefile `API_TEST_FLAGS`).

## Notes

- Auth headers use `{{access_token}}`; tokens are fetched automatically by jan-cli when `--auto-auth` is provided.
- Model IDs are auto-fetched via `--auto-models`; collections only reference `{{model_id}}`/`{{default_model_id}}`.
