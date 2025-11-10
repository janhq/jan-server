# Observability Guide

## Metrics
- **Prometheus** (http://localhost:9090)
  - Scrapes Go services via `/metrics`.
  - Includes default dashboards for request rate, latency, error ratio.
- **Service metrics**:
  - LLM API: request duration, token usage, provider latency.
  - Response API: tool execution counts, depth histogram, orchestration latency.
  - Media API: upload size, S3 latency, resolution cache hits.
  - MCP Tools: tool success/failure, backend latency.

## Tracing
- **OpenTelemetry Collector** listens on `otel-collector:4317`.
- Services enable tracing by setting `OTEL_ENABLED=true` and `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`.
- **Jaeger** UI (http://localhost:16686):
  - Search by `service.name`.
  - Correlate request IDs (`X-Request-Id`) between services.

## Logging
- All services use structured JSON logs (zerolog).
- Docker Compose aggregates stdout/stderr; use `make logs-<service>` targets:
  - `make logs-api`, `make logs-media-api`, `make logs-mcp`, `make logs-infra`.
- For Kubernetes, send logs to Loki or your aggregator via sidecars/DaemonSets.

## Dashboards
- **Grafana** (http://localhost:3001, admin/admin by default).
- Import dashboards from `monitoring/grafana/provisioning/dashboards/`.
- Suggested panels:
  - Request/response duration per service.
  - Database connection pool usage.
  - MCP tool success/error counts.
  - Media upload throughput and storage utilisation.

## Alerts
- Configure Alertmanager rules inside `monitoring/prometheus/alerting-rules.yml` (add file as needed).
- Recommended alerts:
  - High error rate (>5% for 5 minutes)
  - Slow LLM responses (>5s p95)
  - Media API S3 failures
  - MCP tool timeout spikes

## Developer Workflow
1. Start the monitoring stack: `make monitor-up`.
2. Hit the APIs (curl/Postman/Newman).
3. Inspect metrics/traces/logs using the URLs above.
4. Tear down with `make monitor-down` (if defined) or `docker compose down` for the monitoring profile.

Update this file if ports or dashboards change.
