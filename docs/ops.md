# Operations Guide

## API Keys and Kong Consumers

Kong runs in DB-less mode. To create additional consumers or API keys:

1. Add entries to `kong/kong.yml` under the `consumers` section (and optionally `key-auth` credentials).
2. Apply the new configuration using `deck` or by restarting the Kong container with the updated declarative file.

Example deck command:

```sh
deck sync --kong-addr http://localhost:8001 --state kong/kong.yml
```

## Key Rotation

- **API keys**: update `kong.yml` with new credentials, sync with deck, then communicate the updated key to clients. Because Kong hides the credential upstream, `llm-api` continues to operate without changes.
- **Internal vLLM key**: set `VLLM_INTERNAL_KEY` in `.env` and restart `llm-api` plus the vLLM container(s).

## Moving to DB-backed Kong

1. Provision Postgres for Kong metadata.
2. Switch the Kong image configuration to `KONG_DATABASE=postgres` with the appropriate DSN.
3. Migrate the declarative config using `deck dump` followed by `deck sync` pointing at the new database.

## Token Exchange Bootstrap

The `keycloak/init/enable-token-exchange.sh` helper enables token-exchange permissions and assigns the `realm-management/impersonation` role to the `backend` service account. Run it after Keycloak is healthy:

```sh
docker compose exec keycloak bash /opt/keycloak/init/enable-token-exchange.sh
```

Ensure the `.env` file carries consistent values for `BACKEND_CLIENT_ID`, `BACKEND_CLIENT_SECRET`, and `TARGET_CLIENT_ID` before executing the script.
