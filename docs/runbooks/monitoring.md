# Jan Server Monitoring Runbook

> **Status: TEMPLATE / DRAFT.** The alert names below match
> `integrations/monitoring/prometheus-alerts.yml`, but the remediation steps are
> written for a Kubernetes deployment (`kubectl`, PagerDuty escalation, etc.)
> that is **not** wired up in this repository---the local stack runs via Docker
> Compose. Treat the `kubectl`/cloud-specific commands as illustrative until a
> Kubernetes deployment exists; adapt them (e.g., `docker compose`, `make`
> targets) for the environment you are operating.

## Quick Reference

| Alert              | Severity | MTTR Target | On-Call Action                          |
| ------------------ | -------- | ----------- | --------------------------------------- |
| HighLLMLatency     | Warning  | 15min       | [§1](#1-high-llm-latency)               |
| QueueBacklog       | Critical | 5min        | [§2](#2-queue-backlog)                  |
| CollectorDown      | Critical | 2min        | [§3](#3-collector-outage)               |
| StorageFailure     | Critical | 10min       | [§4](#4-media-api-storage-failure)      |
| TraceExportFailure | Warning  | 30min       | [§5](#5-trace-export-failure)           |
| ClassifierErrors   | Warning  | 20min       | [§6](#6-conversation-classifier-errors) |

---

## 1. High LLM Latency

**Alert:** `HighLLMLatency`  
**Triggered when:** P95 LLM API latency >2s for 5min  
**Impact:** Degraded user experience, potential timeouts, increased abandonment rate

### Investigation Steps

1. **Check LLM Provider Dashboard**

   ```bash
   # Open Grafana
   open https://grafana/d/llm-overview
   ```

   - Review latency by model (GPT-4 vs GPT-3.5)
   - Check error rates per provider

2. **Verify Upstream Provider Status**
   - OpenAI: https://status.openai.com
   - Anthropic: https://status.anthropic.com
   - Azure: https://status.azure.com

3. **Check Recent Deployments**

   ```bash
   git log --since="1 hour ago" --oneline
   kubectl rollout history deployment/llm-api
   ```

4. **Inspect Token Queue Depth**

   ```bash
   curl localhost:8080/metrics | grep queue_depth
   ```

5. **Review Jaeger Traces**
   - Find slow traces: `http://jaeger:16686/search?service=llm-api&minDuration=2s`
   - Look for database queries, external API calls taking >1s

### Remediation

**If Provider Issue:**

```bash
# Enable fallback provider
jan-cli config set llm.fallback_enabled=true
jan-cli config set llm.fallback_provider=anthropic
```

**If Jan Server Issue:**

```bash
# Scale replicas
kubectl scale deployment/llm-api --replicas=5

# If memory exhaustion
kubectl top pod -l app=llm-api
kubectl set resources deployment/llm-api --limits=memory=2Gi
```

**If Database Bottleneck:**

```sql
-- Check connection pool
psql jan_server -c "SELECT COUNT(*), state FROM pg_stat_activity GROUP BY state;"

-- Check slow queries
psql jan_server -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

### Escalation

- **After 30min:** Page SRE team lead via PagerDuty
- **After 1h:** Engage vendor support (OpenAI/Anthropic)
- **If P0:** Notify customer success team for user communication

---

## 2. Queue Backlog

**Alert:** `ResponseAPIQueueBacklog`  
**Triggered when:** Response API queue depth >100 for 10min  
**Impact:** Processing delays, webhook failures, incomplete conversations

### Root Causes

- Background worker pool exhausted
- LLM API latency spike
- Media API unavailable
- Database connection pool exhausted

### Investigation

```bash
# Check worker status
curl http://response-api:8082/metrics | grep workers_active
curl http://response-api:8082/metrics | grep workers_idle

# View queue contents
psql jan_server -c "SELECT COUNT(*), status, error_message FROM background_jobs GROUP BY status, error_message ORDER BY COUNT(*) DESC;"

# Check dependent services
make health-check

# View recent job failures
psql jan_server -c "SELECT id, status, error_message, created_at FROM background_jobs WHERE status='failed' ORDER BY created_at DESC LIMIT 20;"
```

### Remediation

1. **Increase Worker Pool**

   ```bash
   kubectl set env deployment/response-api WORKER_POOL_SIZE=20
   kubectl rollout status deployment/response-api
   ```

2. **Purge Old Jobs**

   ```bash
   jan-cli jobs purge --older-than=1h --status=failed
   jan-cli jobs retry --status=failed --max-retries=3
   ```

3. **Restart Service (Last Resort)**
   ```bash
   kubectl rollout restart deployment/response-api
   kubectl rollout status deployment/response-api
   ```

---

## 3. Collector Outage

**Alert:** `OTELCollectorDown`  
**Triggered when:** Collector unreachable for 2min  
**Impact:** Loss of observability (no new traces/metrics), blind operations

### Symptoms

- Grafana dashboards flatline
- Jaeger UI shows no recent traces
- Services log OTLP export errors

### Investigation

```bash
# Check collector health
curl http://otel-collector:13133/

# View collector logs
kubectl logs -l app=otel-collector --tail=100

# Check resource usage
kubectl top pod -l app=otel-collector

# Verify connectivity from services
kubectl run -it --rm debug --image=curlimages/curl --restart=Never \
  -- curl -v http://otel-collector:4318/v1/traces
```

### Remediation

1. **Restart Collector**

   ```bash
   kubectl rollout restart deployment/otel-collector
   kubectl rollout status deployment/otel-collector
   ```

2. **If Resource Exhaustion**

   ```bash
   # Increase memory
   kubectl set resources deployment/otel-collector --limits=memory=1Gi

   # Check Jaeger backend
   curl http://jaeger-query:16686/api/services
   ```

3. **If Configuration Error**

   ```bash
   # Validate config
   kubectl get configmap otel-collector-config -o yaml | yq '.data'

   # Revert to last known good config
   kubectl rollout undo deployment/otel-collector
   ```

### Fallback Mode

Services continue operating without telemetry until collector is restored. No user impact.

---

## 4. Media API Storage Failure

**Alert:** `MediaAPIStorageFailure`  
**Triggered when:** S3 error rate >10% for 2min  
**Impact:** Upload/download failures, broken media references

### Investigation

```bash
# Check S3 metrics
curl http://media-api:8080/metrics | grep s3_errors

# View recent errors
kubectl logs -l app=media-api --tail=50 | grep -i s3

# Check AWS status
open https://health.aws.amazon.com/health/status

# Verify credentials
kubectl get secret media-api-s3-credentials -o yaml
```

### Remediation

1. **Verify S3 bucket exists and is accessible**

   ```bash
   aws s3 ls s3://jan-media-bucket/
   ```

2. **Check IAM permissions**

   ```bash
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::ACCOUNT:role/media-api-role \
     --action-names s3:PutObject s3:GetObject
   ```

3. **Enable fallback storage**
   ```bash
   kubectl set env deployment/media-api STORAGE_FALLBACK_ENABLED=true
   ```

---

## 5. Trace Export Failure

**Alert:** `TraceExportFailure`  
**Triggered when:** Jaeger export failing >10 spans/sec for 5min  
**Impact:** Partial trace loss, incomplete observability

### Investigation

```bash
# Check collector export metrics
curl http://otel-collector:8889/metrics | grep exporter_send_failed

# Check Jaeger ingestion
curl http://jaeger-collector:14269/metrics | grep spans_received

# View collector logs
kubectl logs -l app=otel-collector | grep -i error
```

### Remediation

1. **Verify Jaeger collector is running**

   ```bash
   kubectl get pods -l app=jaeger
   kubectl logs -l app=jaeger --tail=50
   ```

2. **Check network connectivity**

   ```bash
   kubectl run -it --rm debug --image=curlimages/curl --restart=Never \
     -- curl -v http://jaeger-collector:14268/api/traces
   ```

3. **Increase collector retry settings**
   - Edit `monitoring/otel-collector.yaml`
   - Increase `max_elapsed_time` from 5m to 10m
   - Increase `queue_size` from 5000 to 10000
   - Apply config: `kubectl apply -f monitoring/otel-collector.yaml`

4. **Temporary: Reduce sampling rate**
   ```bash
   kubectl set env deployment/llm-api OTEL_TRACES_SAMPLER_ARG=0.1
   kubectl set env deployment/response-api OTEL_TRACES_SAMPLER_ARG=0.1
   ```

---

## 6. Conversation Classifier Errors

**Alert:** `ConversationInsightFailure`  
**Triggered when:** Classifier error rate >5% for 5min  
**Impact:** Missing conversation metadata, incomplete analytics

### Investigation

```bash
# View classifier metrics
curl http://response-api:8082/metrics | grep classifier_errors

# Review error logs
kubectl logs -l app=response-api | grep classifier
```

### Remediation

1. **Check for malformed prompt data**

   ```bash
   # Review recent requests
   kubectl logs -l app=response-api --tail=100 | grep -A5 "classifier error"
   ```

2. **Review recent classifier configuration changes**

   ```bash
   git log --since="1 day ago" --grep="classifier" --oneline
   kubectl describe configmap response-api-config
   ```

3. **Disable classifier temporarily (if persistent)**
   ```bash
   kubectl set env deployment/response-api CLASSIFIER_ENABLED=false
   ```

---

## Appendix A: Common Commands

### Health Checks

```bash
# All services
make health-check

# Individual service
curl http://SERVICE:PORT/health

# Monitoring stack
make monitor-test
```

### Viewing Logs

```bash
# Recent logs
kubectl logs -l app=SERVICE --tail=100

# Follow logs
kubectl logs -l app=SERVICE -f

# Logs with timestamp
kubectl logs -l app=SERVICE --timestamps=true
```

### Metrics Queries

```bash
# Service metrics
curl http://SERVICE:8080/metrics

# Prometheus query
curl 'http://localhost:9090/api/v1/query?query=METRIC_NAME'

# Alert status
curl http://localhost:9090/api/v1/rules
```

### Trace Queries

```bash
# Recent traces for service
curl 'http://localhost:16686/api/traces?service=SERVICE&limit=10'

# Specific trace
curl 'http://localhost:16686/api/traces/TRACE_ID'

# Slow traces
curl 'http://localhost:16686/api/traces?service=SERVICE&minDuration=2s'
```

---

## Appendix B: Escalation Contacts

| Severity      | Contact      | Response Time     | Channel            |
| ------------- | ------------ | ----------------- | ------------------ |
| P0 (Critical) | SRE On-Call  | &lt;5min          | PagerDuty          |
| P1 (High)     | Team Lead    | &lt;15min         | Slack #incidents   |
| P2 (Medium)   | Dev Team     | &lt;1h            | Slack #engineering |
| P3 (Low)      | Ticket Queue | Next business day | Jira               |

---

## Appendix C: Useful Links

- **Grafana:** http://localhost:3331
- **Jaeger:** http://localhost:16686
- **Prometheus:** http://localhost:9090
- **Monitoring Guide:** [docs/guides/monitoring.md](../guides/monitoring.md)
- **Architecture Overview:** [docs/architecture/services.md](../architecture/services.md)
- **Security Policy:** [docs/architecture/security.md](../architecture/security.md)
