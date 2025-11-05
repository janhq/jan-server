# OpenTelemetry Implementation Guide

This document describes how OpenTelemetry is implemented in the llm-api service and how to use it.

## Overview

The llm-api service uses OpenTelemetry for distributed tracing and metrics collection. Traces and metrics are exported to an OpenTelemetry Collector, which then forwards them to Prometheus (metrics) and Jaeger (traces).

## Architecture

```
llm-api → OpenTelemetry SDK → OTLP Exporter → OTel Collector → Prometheus + Jaeger → Grafana
```

## Configuration

OpenTelemetry is configured via environment variables:

```bash
# Enable/disable by providing an endpoint
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318  # HTTP endpoint
# OR
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317  # gRPC endpoint (auto-detected)

# Optional: Custom headers
OTEL_EXPORTER_OTLP_HEADERS=key1=value1,key2=value2

# Service identification
SERVICE_NAME=llm-api
SERVICE_NAMESPACE=jan
ENVIRONMENT=development
```

## Automatic Instrumentation

### HTTP Request Tracing

All HTTP requests are automatically traced via the `TracingMiddleware`. This middleware:

- Creates a span for each HTTP request
- Extracts trace context from incoming headers (for distributed tracing)
- Adds standard HTTP attributes (method, route, status, etc.)
- Records errors for 4xx and 5xx responses
- Injects trace context into the request context

**Attributes captured:**
- `http.method`
- `http.route`
- `http.url`
- `http.target`
- `http.scheme`
- `net.host.name`
- `http.user_agent`
- `http.client_ip`
- `http.status_code`
- `request.id` (if present)

### Logging Integration

The `LoggingMiddleware` automatically correlates logs with traces by adding:

- `trace_id`: OpenTelemetry trace ID
- `span_id`: Current span ID
- `request_id`: Request ID from middleware

This allows you to find all logs related to a specific trace in your log aggregation system.

## Manual Instrumentation

### Adding Spans in Business Logic

Use the helper functions from `internal/infrastructure/observability/tracing.go`:

```go
package mypackage

import (
    "context"
    "jan-server/services/llm-api/internal/config"
    "jan-server/services/llm-api/internal/infrastructure/observability"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

func MyBusinessLogic(ctx context.Context) error {
    cfg := config.GetGlobal()
    
    // Start a new span
    ctx, span := observability.StartSpan(ctx, cfg.ServiceName, "MyBusinessLogic.ProcessData")
    defer span.End()
    
    // Add custom attributes
    observability.AddSpanAttributes(ctx,
        attribute.String("user.id", "12345"),
        attribute.Int("batch.size", 100),
    )
    
    // Add events
    observability.AddSpanEvent(ctx, "validation_started")
    
    // Do some work...
    if err := doWork(); err != nil {
        // Record errors
        observability.RecordError(ctx, err)
        return err
    }
    
    // Set success status
    observability.SetSpanStatus(ctx, codes.Ok, "completed successfully")
    
    return nil
}
```

### Nested Spans

Create child spans for sub-operations:

```go
func ProcessBatch(ctx context.Context, items []Item) error {
    cfg := config.GetGlobal()
    
    // Parent span
    ctx, span := observability.StartSpan(ctx, cfg.ServiceName, "ProcessBatch")
    defer span.End()
    
    observability.AddSpanAttributes(ctx, attribute.Int("batch.size", len(items)))
    
    for i, item := range items {
        // Child span for each item
        _, itemSpan := observability.StartSpan(ctx, cfg.ServiceName, "ProcessItem")
        observability.AddSpanAttributes(ctx,
            attribute.Int("item.index", i),
            attribute.String("item.id", item.ID),
        )
        
        if err := processItem(ctx, item); err != nil {
            observability.RecordError(ctx, err)
            itemSpan.End()
            continue
        }
        
        itemSpan.End()
    }
    
    return nil
}
```

### Adding Trace Context to Logs

Get trace IDs for manual logging:

```go
import (
    "jan-server/services/llm-api/internal/infrastructure/logger"
    "jan-server/services/llm-api/internal/infrastructure/observability"
)

func MyFunction(ctx context.Context) {
    log := logger.GetLogger()
    
    // Get trace context
    traceID := observability.GetTraceID(ctx)
    spanID := observability.GetSpanID(ctx)
    
    log.Info().
        Str("trace_id", traceID).
        Str("span_id", spanID).
        Msg("Processing request")
}
```

## Example: Tracing a Use Case

```go
package usecases

import (
    "context"
    "jan-server/services/llm-api/internal/config"
    "jan-server/services/llm-api/internal/domain"
    "jan-server/services/llm-api/internal/infrastructure/observability"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

type ChatCompletionUseCase struct {
    repo domain.ConversationRepository
    cfg  *config.Config
}

func (uc *ChatCompletionUseCase) Execute(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
    // Start span for the entire use case
    ctx, span := observability.StartSpan(ctx, uc.cfg.ServiceName, "ChatCompletionUseCase.Execute")
    defer span.End()
    
    observability.AddSpanAttributes(ctx,
        attribute.String("model", req.Model),
        attribute.Int("message.count", len(req.Messages)),
    )
    
    // Database operation (create child span)
    observability.AddSpanEvent(ctx, "fetching_conversation")
    ctx, dbSpan := observability.StartSpan(ctx, uc.cfg.ServiceName, "FetchConversation")
    conversation, err := uc.repo.FindByID(ctx, req.ConversationID)
    dbSpan.End()
    
    if err != nil {
        observability.RecordError(ctx, err)
        return nil, err
    }
    
    // LLM call (create child span)
    observability.AddSpanEvent(ctx, "calling_llm")
    ctx, llmSpan := observability.StartSpan(ctx, uc.cfg.ServiceName, "LLMInference")
    observability.AddSpanAttributes(ctx,
        attribute.String("llm.provider", "vllm"),
        attribute.String("llm.model", req.Model),
    )
    
    response, err := uc.callLLM(ctx, req)
    llmSpan.End()
    
    if err != nil {
        observability.RecordError(ctx, err)
        return nil, err
    }
    
    // Save result
    ctx, saveSpan := observability.StartSpan(ctx, uc.cfg.ServiceName, "SaveResponse")
    if err := uc.repo.Save(ctx, response); err != nil {
        observability.RecordError(ctx, err)
        saveSpan.End()
        return nil, err
    }
    saveSpan.End()
    
    observability.SetSpanStatus(ctx, codes.Ok, "completed")
    return response, nil
}
```

## Viewing Traces

### In Jaeger UI

1. Start the monitoring stack: `make monitor-up`
2. Navigate to http://localhost:16686
3. Select service: `llm-api`
4. Search for traces
5. Click on a trace to see:
   - Span timeline
   - Parent-child relationships
   - Attributes and events
   - Errors

### In Grafana

1. Navigate to http://localhost:3001 (admin/admin)
2. Go to Explore
3. Select Jaeger datasource
4. Query traces
5. Correlate with metrics from Prometheus

## Best Practices

1. **Always defer span.End()**
   ```go
   ctx, span := observability.StartSpan(ctx, serviceName, "operation")
   defer span.End()  // Ensures span is closed even if panic occurs
   ```

2. **Use meaningful span names**
   ```go
   // Good
   "ChatCompletion.Execute"
   "UserRepository.FindByID"
   "LLMProvider.GenerateResponse"
   
   // Bad
   "process"
   "doStuff"
   "handler"
   ```

3. **Add relevant attributes**
   ```go
   observability.AddSpanAttributes(ctx,
       attribute.String("user.id", userID),
       attribute.String("model.name", model),
       attribute.Int("batch.size", len(items)),
       attribute.Bool("cache.hit", cacheHit),
   )
   ```

4. **Record errors properly**
   ```go
   if err != nil {
       observability.RecordError(ctx, err)
       return err
   }
   ```

5. **Use events for significant milestones**
   ```go
   observability.AddSpanEvent(ctx, "validation_completed")
   observability.AddSpanEvent(ctx, "cache_miss")
   observability.AddSpanEvent(ctx, "model_loaded")
   ```

6. **Don't create spans for trivial operations**
   - Avoid spans for simple getters/setters
   - Focus on I/O operations, external calls, and business logic

## Troubleshooting

### No traces appearing in Jaeger

1. Check OTEL_EXPORTER_OTLP_ENDPOINT is set:
   ```bash
   docker logs jan-server-llm-api-1 | grep -i otel
   ```

2. Verify OTel Collector is receiving data:
   ```bash
   docker logs jan-server-otel-collector-1
   ```

3. Check llm-api is creating spans:
   - Look for trace_id in logs
   - Verify TracingMiddleware is registered

### Traces not correlating with logs

1. Ensure LoggingMiddleware is after TracingMiddleware
2. Check that context is properly passed through the call chain
3. Verify trace_id appears in log output

### High overhead

1. Reduce sampling rate (if needed, add sampler to otel.go)
2. Avoid creating too many child spans
3. Use span events instead of separate spans for lightweight operations

## References

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
