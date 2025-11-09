package observability

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"jan-server/services/template-api/internal/config"
)

// Shutdown is a function that releases telemetry resources.
type Shutdown func(ctx context.Context) error

// Setup configures OpenTelemetry tracing if enabled.
func Setup(ctx context.Context, cfg *config.Config, log zerolog.Logger) (Shutdown, error) {
	if !cfg.EnableTracing || cfg.OTLPEndpoint == "" {
		log.Info().Msg("Tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	log.Info().Str("endpoint", cfg.OTLPEndpoint).Msg("Tracing enabled")

	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}
