# Contributing to Jan Server

Thanks for taking the time to improve Jan Server! This guide explains how to propose changes, run the required checks, and keep the documentation aligned with the codebase.

## Ways to Contribute
- **Report issues**: use GitHub Issues with reproduction steps, logs, and the commit hash you tested.
- **Feature proposals**: outline the use case, affected services, and expected APIs before opening a pull request.
- **Code changes**: bug fixes, new functionality, refactors, and automation scripts.
- **Docs and examples**: clarify setup steps, add API samples, or improve troubleshooting guides.

## Development Workflow
1. **Sync local environment**
   ```bash
   git checkout main
   git pull origin main
   ```
2. **Create a feature branch**
   ```bash
   git checkout -b feature/<short-description>
   ```
3. **Bootstrap tooling**
   ```bash
   make env-create           # copies .env.template -> .env (idempotent)
   make setup                # dependency check + docker network
   ```
4. **Pick a target service**
   - Run everything in Docker: `make up-full`
   - Hybrid mode for local debugging: `make hybrid-dev-api` / `make hybrid-dev-mcp`

## Coding Standards
- **Language**: Go 1.21+ across services. Use `go fmt ./...` or `make fmt` before committing.
- **Static analysis**: run `make lint` to execute vet, golangci-lint, and other configured linters.
- **Swagger/OpenAPI**: update specs with `make swagger` after changing HTTP handlers.
- **Configuration**: add new env vars to `.env.template`, `config/defaults.env`, and mention them in `config/README.md`.
- **Documentation**: update relevant guides plus `docs/INDEX.md` when adding or moving features.

## Required Test Matrix
Run the smallest set that covers your change:

| Change Type | Minimum Commands |
|-------------|------------------|
| Library or helper updates | `make test` |
| API surface changes | `make test` + targeted Postman suite (for example `make test-conversations`) |
| Cross-service or infra updates | `make test-all` |
| Docker/Kubernetes manifests | `make up-full` (smoke) + `make health-check` |
| Documentation-only | `make lint-docs` *(if available)* or spell/markdown checker of your choice |

For MCP tooling, also run:
```bash
make test-mcp-integration
```

Before pushing, ensure the tree is clean:
```bash
go fmt ./...
make lint
make test
git status -sb         # no unexpected files
```

## Commit and PR Guidelines
- Keep commits focused; split large work into logical chunks.
- Write descriptive messages (for example `feat(response-api): add SSE streaming`).
- Reference the related issue in the pull request body (`Fixes #123`).
- Include screenshots or log excerpts when they clarify behaviour.
- For documentation-heavy PRs, mention which guides or runbooks were updated.

## Documentation Expectations
- `README.md` must stay aligned with the default Docker Compose workflow.
- `docs/getting-started/README.md` is the canonical setup guide; keep it in sync with the Makefile targets.
- `docs/INDEX.md` acts as the sitemap; add or move entries there whenever you add documentation elsewhere.
- If you introduce a new service or API, create or update:
  - `docs/services.md`
  - `docs/api/<service>/README.md`
  - Per-service `services/<name>/README.md`

## Testing Secrets
Do **not** commit real keys or tokens. Place new variables in `.env.template` and document how to obtain them. For CI-only secrets, describe the expectation inside `config/secrets.env.example`.

## Opening the Pull Request
1. Push your branch: `git push origin feature/<short-description>`
2. Create a PR against `main`
3. Fill out the PR template, including:
   - Motivation / context
   - Testing evidence (commands + output summary)
   - Docs updated checklist
4. Respond to review feedback promptly; squash or rebase only when requested.

## Code of Conduct
Be respectful, stay constructive, and follow project maintainers' guidance. By participating you agree to uphold the community standards.

