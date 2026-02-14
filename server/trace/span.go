package trace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type spanContextKeyType string

const (
	currentSpanKey spanContextKeyType = "current_span"
)

type SpanBuilder struct {
	traceID   string
	sessionID string
}

func NewSpanBuilder(traceID, sessionID string) *SpanBuilder {
	return &SpanBuilder{
		traceID:   traceID,
		sessionID: sessionID,
	}
}

func (sb *SpanBuilder) NewSpan(name string, inputs map[string]interface{}) *Span {
	return &Span{
		SpanID:    uuid.New().String(),
		TraceID:   sb.traceID,
		Name:      name,
		StartTime: time.Now(),
		Inputs:    inputs,
		Status:    "running",
	}
}

func (sb *SpanBuilder) NewChildSpan(parent *Span, name string, inputs map[string]interface{}) *Span {
	child := sb.NewSpan(name, inputs)
	if parent != nil {
		child.ParentID = parent.SpanID
	}
	return child
}

func WithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, currentSpanKey, span)
}

func CurrentSpan(ctx context.Context) *Span {
	if span, ok := ctx.Value(currentSpanKey).(*Span); ok {
		return span
	}
	return nil
}

func (s *Span) Complete(outputs map[string]interface{}) {
	now := time.Now()
	s.EndTime = &now
	s.Outputs = outputs
	s.Status = "completed"
}

func (s *Span) Fail(errorMsg string) {
	now := time.Now()
	s.EndTime = &now
	s.Status = "failed"
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata["error"] = errorMsg
}

func (s *Span) AddMetadata(key string, value interface{}) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata[key] = value
}

func (s *Span) Duration() time.Duration {
	if s.EndTime == nil {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

func (s *Span) IsCompleted() bool {
	return s.Status == "completed" || s.Status == "failed"
}

func (s *Span) Clone() *Span {
	clone := &Span{
		SpanID:    s.SpanID,
		TraceID:   s.TraceID,
		ParentID:  s.ParentID,
		Name:      s.Name,
		StartTime: s.StartTime,
		Status:    s.Status,
	}

	if s.EndTime != nil {
		endTimeCopy := *s.EndTime
		clone.EndTime = &endTimeCopy
	}

	if s.Inputs != nil {
		clone.Inputs = make(map[string]interface{})
		for k, v := range s.Inputs {
			clone.Inputs[k] = v
		}
	}

	if s.Outputs != nil {
		clone.Outputs = make(map[string]interface{})
		for k, v := range s.Outputs {
			clone.Outputs[k] = v
		}
	}

	if s.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range s.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}
