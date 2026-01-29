package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the default tracer name
const TracerName = "jan-server"

// GetTracer returns a tracer with the given name
func GetTracer(name string) trace.Tracer {
	if name == "" {
		name = TracerName
	}
	return otel.Tracer(name)
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer(TracerName).Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
	}
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, code trace.SpanStatus) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		// Note: trace.SpanStatus is not directly usable; use SetStatus with codes
	}
}
