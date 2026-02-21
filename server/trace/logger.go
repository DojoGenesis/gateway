package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type CostCalculator interface {
	GetCost(model string, tokens int) float64
}

type TraceLogger struct {
	storage     *TraceStorage
	eventChan   chan<- events.StreamEvent
	costTracker CostCalculator
	mu          sync.RWMutex
	activeSpans map[string]*Span
	otelTracer  oteltrace.Tracer
	otelSpans   map[string]oteltrace.Span
}

func NewTraceLogger(storage *TraceStorage, eventChan chan<- events.StreamEvent) *TraceLogger {
	return &TraceLogger{
		storage:     storage,
		eventChan:   eventChan,
		activeSpans: make(map[string]*Span),
		otelTracer:  otel.Tracer("agentic-gateway"),
		otelSpans:   make(map[string]oteltrace.Span),
	}
}

func NewTraceLoggerWithCostTracker(storage *TraceStorage, eventChan chan<- events.StreamEvent, costTracker CostCalculator) *TraceLogger {
	return &TraceLogger{
		storage:     storage,
		eventChan:   eventChan,
		costTracker: costTracker,
		activeSpans: make(map[string]*Span),
		otelTracer:  otel.Tracer("agentic-gateway"),
		otelSpans:   make(map[string]oteltrace.Span),
	}
}

func NewTraceLoggerWithoutEvents(storage *TraceStorage) *TraceLogger {
	return &TraceLogger{
		storage:     storage,
		activeSpans: make(map[string]*Span),
		otelTracer:  otel.Tracer("agentic-gateway"),
		otelSpans:   make(map[string]oteltrace.Span),
	}
}

func (tl *TraceLogger) StartTrace(ctx context.Context, sessionID string) (string, error) {
	traceID := uuid.New().String()

	trace := &Trace{
		TraceID:   traceID,
		SessionID: sessionID,
		StartTime: time.Now(),
		Status:    "active",
	}

	if tl.storage != nil {
		if err := tl.storage.StoreTrace(ctx, trace); err != nil {
			return "", fmt.Errorf("failed to store trace: %w", err)
		}
	}

	return traceID, nil
}

func (tl *TraceLogger) EndTrace(ctx context.Context, traceID string, status string) error {
	endTime := time.Now()

	if tl.storage != nil {
		if err := tl.storage.UpdateTraceStatus(ctx, traceID, status, endTime); err != nil {
			return fmt.Errorf("failed to end trace: %w", err)
		}
	}

	return nil
}

func (tl *TraceLogger) StartSpan(ctx context.Context, traceID string, name string, inputs map[string]interface{}) (*Span, error) {
	sb := NewSpanBuilder(traceID, "")

	parent := CurrentSpan(ctx)
	var span *Span

	if parent != nil {
		span = sb.NewChildSpan(parent, name, inputs)
	} else {
		span = sb.NewSpan(name, inputs)
	}

	if tl.storage != nil {
		if err := tl.storage.StoreSpan(ctx, span); err != nil {
			return nil, fmt.Errorf("failed to store span: %w", err)
		}
	}

	tl.mu.Lock()
	tl.activeSpans[span.SpanID] = span

	// Create OTEL span if tracer is available
	if tl.otelTracer != nil {
		otelCtx, otelSpan := tl.otelTracer.Start(ctx, name)
		tl.otelSpans[span.SpanID] = otelSpan

		// Add all input attributes to OTEL span
		AddLLMAttributes(otelSpan, inputs)

		// Update context with OTEL span
		ctx = otelCtx
	}
	tl.mu.Unlock()

	if tl.eventChan != nil {
		tl.sendSpanStartEvent(span)
	}

	return span, nil
}

func (tl *TraceLogger) EndSpan(ctx context.Context, span *Span, outputs map[string]interface{}) error {
	if span == nil {
		return fmt.Errorf("cannot end nil span")
	}

	span.Complete(outputs)

	if tl.costTracker != nil && span.Name == "model_invocation" {
		if model, ok := span.Inputs["model"].(string); ok {
			inputTokens := 0
			outputTokens := 0

			if it, ok := outputs["input_tokens"].(int); ok {
				inputTokens = it
			}
			if ot, ok := outputs["output_tokens"].(int); ok {
				outputTokens = ot
			}

			totalTokens := inputTokens + outputTokens
			if totalTokens > 0 {
				cost := tl.costTracker.GetCost(model, totalTokens)
				if span.Metadata == nil {
					span.Metadata = make(map[string]interface{})
				}
				span.Metadata["estimated_cost_usd"] = cost
			}
		}
	}

	if tl.storage != nil {
		if err := tl.storage.StoreSpan(ctx, span); err != nil {
			return fmt.Errorf("failed to update span: %w", err)
		}
	}

	tl.mu.Lock()
	// Complete OTEL span if it exists
	if otelSpan, ok := tl.otelSpans[span.SpanID]; ok {
		// Add all output attributes to OTEL span
		AddLLMAttributes(otelSpan, outputs)
		// Add metadata if available
		if span.Metadata != nil {
			AddLLMAttributes(otelSpan, span.Metadata)
		}
		otelSpan.SetStatus(codes.Ok, "completed")
		otelSpan.End()
		delete(tl.otelSpans, span.SpanID)
	}

	delete(tl.activeSpans, span.SpanID)
	tl.mu.Unlock()

	if tl.eventChan != nil {
		tl.sendSpanEndEvent(span)
	}

	return nil
}

func (tl *TraceLogger) FailSpan(ctx context.Context, span *Span, errorMsg string) error {
	if span == nil {
		return fmt.Errorf("cannot fail nil span")
	}

	span.Fail(errorMsg)

	if tl.storage != nil {
		if err := tl.storage.StoreSpan(ctx, span); err != nil {
			return fmt.Errorf("failed to update span: %w", err)
		}
	}

	tl.mu.Lock()
	// Fail OTEL span if it exists
	if otelSpan, ok := tl.otelSpans[span.SpanID]; ok {
		otelSpan.SetStatus(codes.Error, errorMsg)
		otelSpan.RecordError(fmt.Errorf("%s", errorMsg))
		otelSpan.End()
		delete(tl.otelSpans, span.SpanID)
	}

	delete(tl.activeSpans, span.SpanID)
	tl.mu.Unlock()

	if tl.eventChan != nil {
		tl.sendSpanEndEvent(span)
	}

	return nil
}

func (tl *TraceLogger) GetActiveSpans() []*Span {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	spans := make([]*Span, 0, len(tl.activeSpans))
	for _, span := range tl.activeSpans {
		spans = append(spans, span.Clone())
	}

	return spans
}

func (tl *TraceLogger) GetTraceSpans(ctx context.Context, traceID string) ([]Span, error) {
	if tl.storage == nil {
		return nil, nil
	}
	spans, err := tl.storage.ListSpansByTrace(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve trace spans: %w", err)
	}

	return spans, nil
}

func (tl *TraceLogger) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	if tl.storage == nil {
		return nil, fmt.Errorf("trace storage not initialized")
	}
	trace, err := tl.storage.RetrieveTrace(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve trace: %w", err)
	}

	return trace, nil
}

func (tl *TraceLogger) sendSpanStartEvent(span *Span) {
	if tl.eventChan == nil {
		return
	}

	event := events.NewTraceSpanStartEvent(
		span.TraceID,
		span.SpanID,
		span.ParentID,
		span.Name,
		span.StartTime,
		span.Inputs,
	)

	select {
	case tl.eventChan <- event:
	default:
	}
}

func (tl *TraceLogger) sendSpanEndEvent(span *Span) {
	if tl.eventChan == nil {
		return
	}

	event := events.NewTraceSpanEndEvent(
		span.TraceID,
		span.SpanID,
		span.ParentID,
		span.Name,
		span.StartTime,
		span.EndTime,
		span.Inputs,
		span.Outputs,
		span.Metadata,
		span.Status,
		span.Duration().Milliseconds(),
	)

	select {
	case tl.eventChan <- event:
	default:
	}
}
