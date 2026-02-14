package trace

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// LLM-specific span attribute keys for OpenTelemetry
const (
	// LLMModel is the model name (e.g., "claude-opus-4.6")
	LLMModel = "llm.model"

	// LLMInputTokens is the number of input tokens
	LLMInputTokens = "llm.input_tokens"

	// LLMOutputTokens is the number of output tokens
	LLMOutputTokens = "llm.output_tokens"

	// LLMLatencyMs is the latency in milliseconds
	LLMLatencyMs = "llm.latency_ms"

	// LLMEstimatedCost is the estimated cost in dollars
	LLMEstimatedCost = "llm.estimated_cost"

	// LLMToolName is the name of the tool if span is from tool invocation
	LLMToolName = "llm.tool_name"

	// LLMToolDurationMs is the tool duration in milliseconds
	LLMToolDurationMs = "llm.tool_duration_ms"
)

// AddLLMAttributes adds all attributes from the map to an OTEL span.
// This includes LLM-specific attributes and any other attributes from the internal span.
// The helper function converts map values to appropriate OTEL attribute types.
func AddLLMAttributes(span oteltrace.Span, attrs map[string]interface{}) {
	if span == nil || attrs == nil {
		return
	}

	var otelAttrs []attribute.KeyValue

	for key, value := range attrs {
		// Convert value to appropriate OTEL attribute type
		switch v := value.(type) {
		case string:
			otelAttrs = append(otelAttrs, attribute.String(key, v))
		case int:
			otelAttrs = append(otelAttrs, attribute.Int64(key, int64(v)))
		case int64:
			otelAttrs = append(otelAttrs, attribute.Int64(key, v))
		case float64:
			otelAttrs = append(otelAttrs, attribute.Float64(key, v))
		case bool:
			otelAttrs = append(otelAttrs, attribute.Bool(key, v))
		// For other types, convert to string
		default:
			otelAttrs = append(otelAttrs, attribute.String(key, fmt.Sprint(v)))
		}
	}

	if len(otelAttrs) > 0 {
		span.SetAttributes(otelAttrs...)
	}
}
