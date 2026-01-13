package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "jan-server/response-api"
)

// GetTracer returns the tracer for the response-api service.
func GetTracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// PlanAttributes returns common attributes for plan spans.
func PlanAttributes(planID, responseID, agentType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("plan.id", planID),
		attribute.String("plan.response_id", responseID),
		attribute.String("plan.agent_type", agentType),
	}
}

// TaskAttributes returns common attributes for task spans.
func TaskAttributes(taskID, planID, taskType string, sequence int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("task.id", taskID),
		attribute.String("task.plan_id", planID),
		attribute.String("task.type", taskType),
		attribute.Int("task.sequence", sequence),
	}
}

// StepAttributes returns common attributes for step spans.
func StepAttributes(stepID, taskID, action string, sequence, retryCount int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("step.id", stepID),
		attribute.String("step.task_id", taskID),
		attribute.String("step.action", action),
		attribute.Int("step.sequence", sequence),
		attribute.Int("step.retry_count", retryCount),
	}
}

// ArtifactAttributes returns common attributes for artifact spans.
func ArtifactAttributes(artifactID, contentType, title string, version int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("artifact.id", artifactID),
		attribute.String("artifact.content_type", contentType),
		attribute.String("artifact.title", title),
		attribute.Int("artifact.version", version),
	}
}

// StartPlanSpan starts a new span for plan execution.
func StartPlanSpan(ctx context.Context, planID, responseID, agentType string) (context.Context, trace.Span) {
	ctx, span := GetTracer().Start(ctx, "plan.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(PlanAttributes(planID, responseID, agentType)...),
	)
	return ctx, span
}

// StartTaskSpan starts a new span for task execution.
func StartTaskSpan(ctx context.Context, taskID, planID, taskType string, sequence int) (context.Context, trace.Span) {
	ctx, span := GetTracer().Start(ctx, "task.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(TaskAttributes(taskID, planID, taskType, sequence)...),
	)
	return ctx, span
}

// StartStepSpan starts a new span for step execution.
func StartStepSpan(ctx context.Context, stepID, taskID, action string, sequence, retryCount int) (context.Context, trace.Span) {
	ctx, span := GetTracer().Start(ctx, "step.execute."+action,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(StepAttributes(stepID, taskID, action, sequence, retryCount)...),
	)
	return ctx, span
}

// StartArtifactSpan starts a new span for artifact operations.
func StartArtifactSpan(ctx context.Context, operation, artifactID, contentType string) (context.Context, trace.Span) {
	ctx, span := GetTracer().Start(ctx, "artifact."+operation,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("artifact.id", artifactID),
			attribute.String("artifact.content_type", contentType),
		),
	)
	return ctx, span
}

// RecordError records an error on a span.
func RecordError(span trace.Span, err error, severity string) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(attribute.String("error.severity", severity))
}

// AddProgressEvent adds a progress event to a span.
func AddProgressEvent(span trace.Span, progress float64, message string) {
	span.AddEvent("progress",
		trace.WithAttributes(
			attribute.Float64("progress.percent", progress),
			attribute.String("progress.message", message),
		),
	)
}

// AddStatusTransition adds a status transition event to a span.
func AddStatusTransition(span trace.Span, fromStatus, toStatus string) {
	span.AddEvent("status.transition",
		trace.WithAttributes(
			attribute.String("status.from", fromStatus),
			attribute.String("status.to", toStatus),
		),
	)
}

// AddRetryEvent adds a retry event to a span.
func AddRetryEvent(span trace.Span, attempt int, reason string) {
	span.AddEvent("retry",
		trace.WithAttributes(
			attribute.Int("retry.attempt", attempt),
			attribute.String("retry.reason", reason),
		),
	)
}

// AddUserInputEvent adds a user input required event to a span.
func AddUserInputEvent(span trace.Span, prompt string) {
	span.AddEvent("user_input.required",
		trace.WithAttributes(
			attribute.String("user_input.prompt", prompt),
		),
	)
}
