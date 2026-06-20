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
   make setup                # copies .env.template -> root .env (idempotent) + checks deps
   ```
4. **Pick a target service**
   - Run everything in Docker: `make up-full`
   - Hybrid mode for local debugging: `make dev-full`, then run a service natively
     with `jan-cli dev run <svc>` (for example `jan-cli dev run llm-api`)

## Coding Standards

- **Language**: Go 1.24.0 across services. Use `go fmt ./...` or `make fmt` before committing.
- **Static analysis**: run `make lint` to execute vet, golangci-lint, and other configured linters.
- **Swagger/OpenAPI**: update specs with `make swagger` after changing HTTP handlers.
- **Configuration**: add new env vars to the root `.env.template` and document them in `docs/configuration/README.md`.
- **Documentation**: update relevant guides plus `docs/README.md` when adding or moving features.

## Required Test Matrix

Run the smallest set that covers your change:

| Change Type                    | Minimum Commands                                                          |
| ------------------------------ | ------------------------------------------------------------------------- |
| Library or helper updates      | `go test ./services/<svc>/...`                                            |
| API surface changes            | targeted suite (for example `make test-conversation` or `make test-auth`) |
| Cross-service or infra updates | `make test-all`                                                           |
| Docker/Kubernetes manifests    | `make up-full` (smoke) + `make health-check`                              |

The integration suites run through jan-cli api-test collections. Available
targets include `make test-all`, `make test-auth`, `make test-conversation`,
`make test-response`, `make test-media`, and `make test-mcp`.

Before pushing, ensure the tree is clean:

```bash
go fmt ./...
make lint
go test ./services/<svc>/...
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
- `docs/quickstart.md` is the canonical setup guide; keep it in sync with the Makefile targets.
- `docs/README.md` acts as the sitemap; add or move entries there whenever you add documentation elsewhere.
- If you introduce a new service or API, create or update:
  - `docs/architecture/services.md`
  - `docs/api/<service>/README.md`
  - Per-service `services/<name>/README.md`

## Documentation standards

- Keep documentation in sync with the code; update docs in the same PR as the
  change that affects them.
- Single-source the service/port table in `docs/architecture/README.md` and link
  to it rather than duplicating ports across files.
- Single-source authentication details in `docs/guides/authentication.md`.
- Do not add per-file version or date stamps; let Git history track changes.

## Testing Secrets

Do **not** commit real keys or tokens. Place new variables in the root `.env.template` (with placeholder values) and document how to obtain them. The real `.env` is git-ignored; configure CI secrets through the pipeline's secret store.

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
