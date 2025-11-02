# jan-server Quickstart

1. Copy the environment template and review required secrets:

   ```sh
   cp .env.example .env
   ```

2. Choose a vLLM profile and start the stack:

   ```sh
   make up-gpu    # or make up-cpu on machines without GPUs
   ```

3. Wait for llm-api to apply the embedded database migrations (watch with `docker compose logs llm-api`).

4. Generate (or merge) the Swagger documents:

   ```sh
   make swag
   ```

5. Verify connectivity through Kong:

   ```sh
   curl http://localhost:8000/v1/models
   ```

6. Request a chat completion via Kong:

   ```sh
   curl --request POST \
     --url http://localhost:8000/v1/chat/completions \
     --header "Authorization: Bearer <jwt-or-token>" \
     --header "Content-Type: application/json" \
     --data '{"model":"jan-v1-2509","messages":[{"role":"user","content":"Hello"}]}'
   ```

7. For streaming responses append `?stream=true` and consume the SSE stream.

8. Open the consolidated Swagger UI at <http://localhost:8000/v1/swagger/index.html>.

To stop the environment (and remove volumes) run:

```sh
make down
```
