package trace

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

// NewOTELExporter creates an OTLP gRPC exporter for sending traces to an OpenTelemetry collector.
// If endpoint is empty or not configured, it returns nil with no error (graceful degradation).
// The endpoint should be in the format "localhost:4317" or set via OTEL_EXPORTER_OTLP_ENDPOINT env var.
func NewOTELExporter(endpoint string) (trace.SpanExporter, error) {
	// Check environment variable if endpoint not provided
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	// Graceful degradation: if no endpoint configured, OTEL is not enabled
	if endpoint == "" {
		return nil, nil
	}

	// Create OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(), // For development; in production use TLS
	)

	if err != nil {
		return nil, err
	}

	return exporter, nil
}
