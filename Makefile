COMPOSE ?= docker compose
VLLM_COMPOSE ?= docker compose -f docker-compose.yml -f docker-compose.vllm.yml
VLLM_COMPOSE_ONLY ?= docker compose -f docker-compose.vllm.yml
MONITOR_COMPOSE ?= docker compose -f docker-compose.monitor.yml
NEWMAN ?= newman
NEWMAN_AUTH_COLLECTION ?= tests/automation/auth-postman-scripts.json
NEWMAN_CONVERSATION_COLLECTION ?= tests/automation/conversations-postman-scripts.json

.PHONY: up up-gpu up-cpu down down-db reset-db logs swag curl-chat fmt lint test newman newman-debug up-full-local up-full-docker restart-kong monitor-up monitor-down monitor-logs up-mcp-tools

ifeq ($(OS),Windows_NT)
define compose_full_with_env
	set "ENV_FILE=$(1)" && $(COMPOSE) --env-file $(1) --profile full up -d --build
endef
else
define compose_full_with_env
	ENV_FILE=$(1) $(COMPOSE) --env-file $(1) --profile full up -d --build
endef
endif

up:
	$(COMPOSE) up -d --build

up-gpu:
	$(VLLM_COMPOSE) --profile gpu up -d --build

up-cpu:
	$(VLLM_COMPOSE) --profile cpu up -d --build

up-gpu-only:
	$(VLLM_COMPOSE_ONLY) --profile gpu up -d --build

up-cpu-only:
	$(VLLM_COMPOSE_ONLY) --profile cpu up -d --build

up-infra:
	$(COMPOSE) up -d --build

up-llm-api:
	$(COMPOSE) --profile llm-api up -d --build

up-mcp-tools:
	$(COMPOSE) --profile mcp-tools up -d --build

up-kong:
	$(COMPOSE) --profile kong up -d

up-full:
	$(COMPOSE) --profile full up -d --build

up-full-local:
	$(call compose_full_with_env,.env.local)

up-full-docker:
	$(call compose_full_with_env,.env.docker)

up-gpu-infra:
	$(VLLM_COMPOSE) --profile gpu up -d --build

up-gpu-llm-api:
	$(VLLM_COMPOSE) --profile gpu --profile llm-api up -d --build

up-gpu-kong:
	$(VLLM_COMPOSE) --profile gpu --profile kong up -d

up-gpu-full:
	$(VLLM_COMPOSE) --profile gpu --profile full up -d --build

down:
	$(COMPOSE) down -v

down-db:
	$(COMPOSE) down -v api-db

reset-db:
	@echo "Stopping and removing API database to fix migration issues..."
	$(COMPOSE) stop api-db
	$(COMPOSE) rm -f api-db
	docker volume rm jan-server_api-db-data || true
	@echo "Database reset complete. Run 'make up-llm-api' or 'make up-full' to restart."

restart-kong:
	@echo "Restarting Kong to reload configuration..."
	$(COMPOSE) restart kong
	@echo "Kong restarted. Waiting for it to be ready..."
	@sleep 3
	@echo "Kong is ready."

monitor-up:
	@echo "Starting Observability Stack (Prometheus, Jaeger, Grafana, OpenTelemetry Collector)..."
	$(MONITOR_COMPOSE) up -d
	@echo ""
	@echo "Observability Stack is starting. Access dashboards at:"
	@echo "  - Grafana:    http://localhost:3001 (admin/admin)"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Jaeger:     http://localhost:16686"
	@echo ""

monitor-down:
	@echo "Stopping Observability Stack..."
	$(MONITOR_COMPOSE) down
	@echo "Observability Stack stopped."

monitor-down-v:
	@echo "Stopping Observability Stack and removing volumes..."
	$(MONITOR_COMPOSE) down -v
	@echo "Observability Stack stopped and data volumes removed."

monitor-logs:
	$(MONITOR_COMPOSE) logs -f

logs:
	$(COMPOSE) logs -f

swag:
	go run ./tools/swagger-merge -in docs/openapi/llm-api.json -out docs/openapi/combined.json

curl-chat:
	curl -s -H "Authorization: Bearer $$TOKEN" -H "Content-Type: application/json" \
	  -d '{"model":"jan-v1-4b","messages":[{"role":"user","content":"Hello"}]}' \
	  http://localhost:8001/v1/chat/completions | jq

fmt:
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

lint:
	go vet ./...

test:
	go test ./...

newman:
	@echo "Running Newman tests..."
	@$(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--reporters cli && \
	$(NEWMAN) run $(NEWMAN_CONVERSATION_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--reporters cli
	@echo ""
	@echo "All Newman tests completed successfully!"

newman-debug:
	NODE_DEBUG=request $(NEWMAN) run $(NEWMAN_AUTH_COLLECTION) \
		--env-var "kong_url=http://localhost:8000" \
		--env-var "llm_api_url=http://localhost:8000" \
		--env-var "keycloak_base_url=http://localhost:8085" \
		--env-var "keycloak_admin=admin" \
		--env-var "keycloak_admin_password=admin" \
		--env-var "realm=jan" \
		--env-var "client_id_public=llm-api" \
		--verbose \
		--reporter-cli-no-banner \
		--reporter-cli-no-summary \
		--reporter-cli-show-timestamps
