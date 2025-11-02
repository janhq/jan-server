COMPOSE ?= docker compose
VLLM_COMPOSE ?= docker compose -f docker-compose.yml -f docker-compose.vllm.yml
VLLM_COMPOSE_ONLY ?= docker compose -f docker-compose.vllm.yml
NEWMAN ?= newman
NEWMAN_COLLECTION ?= tests/automation/auth-postman-scripts.json

.PHONY: up up-gpu up-cpu down logs swag curl-chat fmt lint test newman newman-debug up-full-local up-full-docker

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
	$(NEWMAN) run $(NEWMAN_COLLECTION)

newman-debug:
	NODE_DEBUG=request $(NEWMAN) run $(NEWMAN_COLLECTION) --verbose --reporter-cli-no-banner --reporter-cli-no-summary --reporter-cli-show-timestamps
