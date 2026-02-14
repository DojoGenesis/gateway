package trace

import (
	"context"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewOTELExporter_WithEndpoint(t *testing.T) {
	// Test with explicit endpoint
	exporter, err := NewOTELExporter("localhost:4317")
	if err != nil {
		t.Fatalf("expected no error when creating OTEL exporter with endpoint, got: %v", err)
	}
	if exporter == nil {
		t.Fatal("expected non-nil exporter when endpoint is provided")
	}

	// Clean up
	if err := exporter.Shutdown(context.Background()); err != nil {
		t.Logf("warning: failed to shutdown exporter: %v", err)
	}
}

func TestNewOTELExporter_WithoutEndpoint(t *testing.T) {
	// Ensure no env var is set
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Test with no endpoint (graceful degradation)
	exporter, err := NewOTELExporter("")
	if err != nil {
		t.Fatalf("expected no error when endpoint is empty, got: %v", err)
	}
	if exporter != nil {
		t.Fatal("expected nil exporter when no endpoint is provided (graceful degradation)")
	}
}

func TestNewOTELExporter_FromEnvVar(t *testing.T) {
	// Set environment variable
	testEndpoint := "localhost:4317"
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", testEndpoint)
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Test with env var
	exporter, err := NewOTELExporter("")
	if err != nil {
		t.Fatalf("expected no error when using env var, got: %v", err)
	}
	if exporter == nil {
		t.Fatal("expected non-nil exporter when env var is set")
	}

	// Clean up
	if err := exporter.Shutdown(context.Background()); err != nil {
		t.Logf("warning: failed to shutdown exporter: %v", err)
	}
}

func TestSpanExport_WithMockExporter(t *testing.T) {
	// Create a mock exporter
	mockExporter := tracetest.NewInMemoryExporter()

	// Create a TracerProvider with the mock exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(mockExporter),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Logf("warning: failed to shutdown tracer provider: %v", err)
		}
	}()

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Create a tracer
	tracer := otel.Tracer("test-tracer")

	// Create a span
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")

	// Add attributes
	attrs := map[string]interface{}{
		LLMModel:         "claude-opus-4.6",
		LLMInputTokens:   1000,
		LLMOutputTokens:  500,
		LLMLatencyMs:     250,
		LLMEstimatedCost: 0.05,
	}
	AddLLMAttributes(span, attrs)

	// End the span
	span.End()

	// Force export
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush spans: %v", err)
	}

	// Wait a bit for export to complete
	time.Sleep(100 * time.Millisecond)

	// Verify span was exported
	spans := mockExporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span to be exported, got none")
	}

	// Verify span name
	if spans[0].Name != "test-span" {
		t.Errorf("expected span name 'test-span', got '%s'", spans[0].Name)
	}

	// Verify attributes were added
	exportedAttrs := spans[0].Attributes
	foundModel := false
	foundTokens := false

	for _, attr := range exportedAttrs {
		if string(attr.Key) == LLMModel && attr.Value.AsString() == "claude-opus-4.6" {
			foundModel = true
		}
		if string(attr.Key) == LLMInputTokens && attr.Value.AsInt64() == 1000 {
			foundTokens = true
		}
	}

	if !foundModel {
		t.Error("expected to find llm.model attribute in exported span")
	}
	if !foundTokens {
		t.Error("expected to find llm.input_tokens attribute in exported span")
	}
}
