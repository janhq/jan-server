# Observability & Monitoring Stack

This document describes the optional observability stack for Jan Server, which provides comprehensive monitoring, metrics, and distributed tracing capabilities.

## Overview

The monitoring stack is completely optional and runs separately from the main Jan Server services. It consists of:

- **OpenTelemetry Collector**: Telemetry data collection and forwarding
- **Prometheus**: Metrics storage and querying
- **Jaeger**: Distributed tracing backend
- **Grafana**: Unified visualization dashboard

## Quick Start

### Start Monitoring Stack

```bash
make monitor-up
```

This command will:

1. Start all monitoring services (Prometheus, Jaeger, Grafana, OpenTelemetry Collector)
2. Display access URLs for each dashboard
3. Run in the background

### Access Dashboards

- **Grafana** (Unified Dashboard): http://localhost:3331
- Username: `admin`
- Password: `admin`
- Pre-configured with Prometheus and Jaeger datasources

- **Prometheus** (Metrics): http://localhost:9090
- Direct PromQL queries
- Service discovery status
- Target health monitoring

- **Jaeger** (Traces): http://localhost:16686
- Distributed trace search
- Service dependency graph
- Performance analysis

### Stop Monitoring Stack

```bash
# Stop but keep data
make monitor-down

# Stop and remove all data volumes (fresh start)
make monitor-clean
```

### View Logs

```bash
make monitor-logs
```

## Architecture

```
+-------------------------------------------------------------+
| Jan Server Services |
| (llm-api, mcp-tools, etc.) |
+----------------+--------------------------------------------+
 | OpenTelemetry Protocol (OTLP)
 | Ports: 4318 (HTTP), 4317 (gRPC)
 v
+-------------------------------------------------------------+
| OpenTelemetry Collector |
| - Receives metrics and traces from services |
| - Processes and enriches telemetry data |
| - Exports to Prometheus (metrics) and Jaeger (traces) |
| - Uses OTLP exporter for Jaeger (not deprecated Jaeger) |
+------------+------------------------------+-----------------+
 | |
 | Metrics | Traces (OTLP)
 v v
+------------------------+ +--------------------------------+
| Prometheus | | Jaeger |
| - Time-series DB | | - Trace storage |
| - 15s scrape interval | | - Service dependency graph |
| - PromQL queries | | - Performance insights |
+------------+-----------+ +------------+-------------------+
 | |
 +--------------+---------------+
 v
 +------------------------+
 | Grafana |
 | - Unified dashboards |
 | - Metrics + Traces |
 | - Alerting |
 +------------------------+
```

## Configuration

### Environment Variables

Set these in your root `.env` file (the single env file used by Compose and the services):

```bash
# Prometheus
PROMETHEUS_PORT=9090

# Jaeger
JAEGER_UI_PORT=16686

# Grafana
GRAFANA_PORT=3331
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# OpenTelemetry
OTEL_GRPC_PORT=4317
OTEL_HTTP_PORT=4318
```

### Enable Telemetry in Services

To send metrics and traces from Jan Server services:

```bash
# In llm-api environment
OTEL_ENABLED=true
OTEL_SERVICE_NAME=llm-api
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

### Prometheus Configuration

The `integrations/monitoring/prometheus.yml` file defines scrape targets:

```yaml
scrape_configs:
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:8889']

  - job_name: 'llm-api'
    static_configs:
      - targets: ['llm-api:8080']
    metrics_path: '/metrics'

  - job_name: 'response-api'
    static_configs:
      - targets: ['response-api:8082']
    metrics_path: '/metrics'

  - job_name: 'media-api'
    static_configs:
      - targets: ['media-api:8285']
    metrics_path: '/metrics'

  - job_name: 'mcp-tools'
    static_configs:
      - targets: ['mcp-tools:8091']
    metrics_path: '/metrics'
```

### Grafana Datasources

Datasources are auto-provisioned from `integrations/monitoring/grafana/provisioning/datasources/datasources.yml`:

- **Prometheus**: Default datasource for metrics
- **Jaeger**: Datasource for distributed traces

## Usage

### Viewing Metrics in Prometheus

1. Navigate to http://localhost:9090
2. Use the "Graph" tab for queries
3. Example PromQL queries:

```text
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Response time (95th percentile)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Viewing Traces in Jaeger

1. Navigate to http://localhost:16686
2. Select a service (e.g., `llm-api`)
3. Search for traces by:

- Time range
- Duration
- Tags

4. Click on a trace to view:

- Span timeline
- Service dependencies
- Tags and logs

### Creating Grafana Dashboards

1. Navigate to http://localhost:3331 (admin/admin)
2. Click "+" -> "Create Dashboard"
3. Add panels with queries from Prometheus or Jaeger
4. Save the dashboard

To persist dashboards:

1. Export as JSON
2. Save to `integrations/monitoring/grafana/provisioning/dashboards/json/`
3. Restart Grafana: `make monitor-down && make monitor-up`

## Data Persistence

The monitoring stack uses Docker volumes for data persistence:

- `prometheus-data`: Stores metrics time-series data
- `grafana-data`: Stores dashboards, users, and settings

### Backup Data

```bash
# Backup Prometheus data
docker run --rm -v jan-server_prometheus-data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz -C /data.

# Backup Grafana data
docker run --rm -v jan-server_grafana-data:/data -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz -C /data.
```

### Restore Data

```bash
# Restore Prometheus data
docker run --rm -v jan-server_prometheus-data:/data -v $(pwd):/backup alpine sh -c "cd /data && tar xzf /backup/prometheus-backup.tar.gz"

# Restore Grafana data
docker run --rm -v jan-server_grafana-data:/data -v $(pwd):/backup alpine sh -c "cd /data && tar xzf /backup/grafana-backup.tar.gz"
```

## Troubleshooting

### Monitoring Stack Won't Start

```bash
# Check if services are running
docker compose -f infra/docker/observability.yml ps

# View logs
make monitor-logs

# Restart with fresh data
make monitor-clean && make monitor-up
```

### No Metrics in Prometheus

1. Check if OpenTelemetry Collector is running:

```bash
docker compose -f infra/docker/observability.yml ps otel-collector
```

2. Verify Prometheus targets are healthy:

- Navigate to http://localhost:9090/targets
- All targets should show "UP" status

3. Ensure services are exporting metrics:

- Set `OTEL_ENABLED=true` in service environment
- Restart the service

### No Traces in Jaeger

1. Check Jaeger is receiving data:

```bash
make monitor-logs | grep jaeger
```

2. Verify OpenTelemetry Collector is exporting to Jaeger:

```bash
make monitor-logs | grep "jaeger.*exporter"
```

3. Ensure services are generating traces:

- Check service logs for trace IDs
- Verify OTLP endpoint is correct

### Grafana Datasources Not Working

1. Check datasource configuration:

- Login to Grafana
- Go to Configuration -> Data Sources
- Test each datasource

2. Verify provisioning:

```bash
docker compose -f infra/docker/observability.yml exec grafana ls -la /etc/grafana/provisioning/datasources
```

3. Restart Grafana:

```bash
docker compose -f infra/docker/observability.yml restart grafana
```

## Advanced Configuration

### Custom Prometheus Retention

Edit `infra/docker/observability.yml`:

```yaml
prometheus:
  command:
    - "--storage.tsdb.retention.time=30d" # Keep data for 30 days
    - "--storage.tsdb.retention.size=10GB" # Max 10GB storage
```

### Custom Grafana Plugins

Edit `infra/docker/observability.yml`:

```yaml
grafana:
  environment:
  GF_INSTALL_PLUGINS: "grafana-clock-panel,grafana-simple-json-datasource"
```

### Enable Jaeger Sampling

Edit `infra/docker/observability.yml`:

```yaml
jaeger:
  environment:
  COLLECTOR_OTLP_ENABLED: "true"
  SAMPLING_STRATEGIES_FILE: /etc/jaeger/sampling.json
  volumes: -./docs/jaeger-sampling.json:/etc/jaeger/sampling.json:ro
```

## Production Recommendations

1. **Change default Grafana password**:

```bash
GRAFANA_ADMIN_PASSWORD=<secure-password>
```

2. **Configure retention policies**:

- Prometheus: Set appropriate retention based on storage
- Jaeger: Configure sampling to reduce data volume

3. **Set up alerting**:

- Configure Prometheus alert rules
- Set up Grafana alert notifications (email, Slack, etc.)

4. **Secure access**:

- Use reverse proxy (nginx/traefik) with TLS
- Implement authentication/authorization
- Restrict network access to monitoring ports

5. **Scale for production**:

- Use external storage for Prometheus (remote write)
- Use production-grade Jaeger backend (Elasticsearch, Cassandra)
- Enable Grafana HA mode

## Service Health Monitoring

All services expose a `/healthz` endpoint (Kong's dev-full upstream health checks also use `/healthz`):

```bash
curl http://localhost:8080/healthz   # LLM API
curl http://localhost:8082/healthz   # Response API
curl http://localhost:8285/healthz   # Media API
curl http://localhost:8091/healthz   # MCP Tools
```

`make health-check` runs the full-stack health summary across infrastructure and API/MCP services.

## Key Metrics

Each service exposes Prometheus metrics on `/metrics` (scraped per `integrations/monitoring/prometheus.yml`). Representative series:

**LLM API**

```
llm_api_requests_total                     # Total API requests (labelled by status)
llm_api_request_duration_seconds           # Request latency histogram
llm_api_time_to_first_token_seconds        # Streaming time-to-first-token histogram
llm_api_tokens_total                       # Tokens processed
llm_api_provider_health                    # Provider health (1 = healthy, 0 = unhealthy)
llm_api_active_connections                 # Concurrent connections
llm_api_conversations_created_total        # Conversations created
```

**Response API**

```
response_api_requests_total                # Total API requests (labelled by status)
response_api_queue_depth                   # Background job queue depth
response_api_workers_active                # Active background workers
response_api_workers_idle                  # Idle background workers
response_api_db_query_duration_seconds     # DB query latency histogram
```

**Media API**

```
media_api_requests_total                   # Total API requests (labelled by status)
media_api_s3_errors_total                  # S3 operation errors
```

**MCP Tools**

```
mcp_tools_calls_total                      # Tool calls (labelled by status)
mcp_tools_circuit_state                    # Circuit breaker state (1 = open)
```

## Alerting

Prometheus alert rules live in `integrations/monitoring/prometheus-alerts.yml` (loaded via the `rule_files` directive). They are grouped into `jan_server_critical`, `jan_server_performance`, `jan_server_capacity`, and `jan_server_business`. A few representative rules:

```yaml
groups:
  - name: jan_server_critical
    interval: 30s
    rules:
      - alert: ServiceDown
        expr: up{job=~"llm-api|response-api|media-api|mcp-tools"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.job }} is down"

      - alert: HighLLMLatency
        expr: histogram_quantile(0.95, rate(llm_api_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
          service: llm-api
        annotations:
          summary: "LLM API P95 latency {{ $value }}s exceeds 2s"

      - alert: ResponseAPIQueueBacklog
        expr: response_api_queue_depth > 100
        for: 10m
        labels:
          severity: critical
          service: response-api
        annotations:
          summary: "Response API queue depth {{ $value }}"

  - name: jan_server_performance
    interval: 1m
    rules:
      - alert: HighErrorRate
        expr: rate(llm_api_requests_total{status=~"5.."}[5m]) / rate(llm_api_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
          service: llm-api
        annotations:
          summary: "High error rate {{ $value | humanizePercentage }}"
```

Alert annotations reference on-call steps in `docs/runbooks/monitoring.md`. To add or change alerts, edit `integrations/monitoring/prometheus-alerts.yml` and reload Prometheus (`make monitor-down && make monitor-up`).

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
