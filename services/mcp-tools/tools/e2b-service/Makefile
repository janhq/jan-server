.PHONY: install dev run test lint clean docker-build

# Install for production
install:
	pip install .

# Install for development (editable + dev deps)
dev:
	pip install -e ".[dev]"

# Run locally (requires 'make dev' first)
run:
	python -m cmd.server.main

# Run with uvicorn reload
run-reload:
	uvicorn cmd.server.main:app --host 0.0.0.0 --port 8095 --reload

# Run tests
test:
	pytest tests/ -v

# Lint code
lint:
	ruff check .
	ruff format --check .

# Format code
format:
	ruff format .

# Clean build artifacts
clean:
	rm -rf build/ dist/ *.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete

# Build Docker image
docker-build:
	docker build -t e2b-api:latest .

# Run Docker container (for testing - normally use docker-compose)
docker-run:
	docker run --rm -p 8095:8095 \
		-e E2B_API_KEY=$${E2B_API_KEY} \
		-e E2B_TEMPLATE_ID=$${E2B_TEMPLATE_ID:-jan-browser-sandbox} \
		e2b-api:latest
