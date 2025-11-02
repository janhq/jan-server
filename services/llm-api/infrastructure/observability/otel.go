package observability

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"jan-server/services/llm-api/infrastructure/config"
)

// Setup initialises OpenTelemetry tracing and metrics exporters. It returns a shutdown function that must be invoked on exit.
func Setup(ctx context.Context, cfg *config.Config, logger zerolog.Logger) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceNamespace(cfg.ServiceNamespace),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	var (
		tracerProvider *sdktrace.TracerProvider
		meterProvider  *sdkmetric.MeterProvider
	)

	if cfg.OTLPEndpoint != "" {
		// Normalize endpoint: allow values like "otel-collector:4318" or full URLs like "http://otel-collector:4318"
		endpoint := cfg.OTLPEndpoint
		insecure := true
		if strings.HasPrefix(endpoint, "http://") {
			endpoint = strings.TrimPrefix(endpoint, "http://")
			insecure = true
		} else if strings.HasPrefix(endpoint, "https://") {
			endpoint = strings.TrimPrefix(endpoint, "https://")
			insecure = false
		}

		traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}
		traceOpts = append(traceOpts, headerOptions(cfg.OTLPHeaders)...)
		metricOpts = append(metricOpts, metricHeaderOptions(cfg.OTLPHeaders)...)

		traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			return nil, err
		}

		meterExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			return nil, err
		}

		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
		)

		reader := sdkmetric.NewPeriodicReader(meterExporter, sdkmetric.WithInterval(30*time.Second))
		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(reader),
			sdkmetric.WithResource(res),
		)
	} else {
		tracerProvider = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithResource(res))
	}

	otel.SetTracerProvider(tracerProvider)

	shutdown := func(ctx context.Context) error {
		var shutdownErr error
		if err := meterProvider.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("shutdown meter provider")
			shutdownErr = err
		}
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("shutdown tracer provider")
			if shutdownErr == nil {
				shutdownErr = err
			}
		}
		return shutdownErr
	}

	return shutdown, nil
}

func headerOptions(raw string) []otlptracehttp.Option {
	if raw == "" {
		return nil
	}
	opts := make([]otlptracehttp.Option, 0)
	headers := parseHeaders(raw)
	if len(headers) == 0 {
		return nil
	}
	opts = append(opts, otlptracehttp.WithHeaders(headers))
	return opts
}

func metricHeaderOptions(raw string) []otlpmetrichttp.Option {
	if raw == "" {
		return nil
	}
	headers := parseHeaders(raw)
	if len(headers) == 0 {
		return nil
	}
	return []otlpmetrichttp.Option{otlpmetrichttp.WithHeaders(headers)}
}

func parseHeaders(raw string) map[string]string {
	result := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

// NoopTracer returns a noop tracer when telemetry is disabled.
func NoopTracer() trace.Tracer {
	return otel.Tracer("noop")
}
