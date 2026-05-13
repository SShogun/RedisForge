package observability

// tracing_hooks.go initialises the OpenTelemetry TracerProvider and
// registers it globally. Uses a stdout exporter in dev so you can see
// traces without a collector. In ObsTower, swap for the OTLP exporter.

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracer sets up a stdout-exporting TracerProvider.
// Returns a shutdown function — call it in your graceful shutdown sequence.
func InitTracer(ctx context.Context, serviceName, version string) (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("observability.InitTracer: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability.InitTracer: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 100% in dev
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
