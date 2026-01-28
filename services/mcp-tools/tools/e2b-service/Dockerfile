FROM python:3.13-slim

WORKDIR /app

# Install curl for healthcheck
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

# Copy application code first (needed for pip install)
COPY pyproject.toml .
COPY cmd/ cmd/
COPY internal/ internal/

# Copy alembic for migrations
COPY alembic.ini .
COPY alembic/ alembic/

# Install package (this makes 'internal' and 'cmd' importable)
RUN pip install --no-cache-dir .

# Expose port (internal only - not mapped to host in docker-compose)
EXPOSE 8095

# Run the application
CMD ["python", "-m", "cmd.server.main"]
