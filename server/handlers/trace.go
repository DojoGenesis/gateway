package handlers

import (
	"log/slog"
	"net/http"

	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/gin-gonic/gin"
)

// TraceHandler handles trace-related HTTP requests.
type TraceHandler struct {
	storage *trace.TraceStorage
}

// NewTraceHandler creates a new TraceHandler.
func NewTraceHandler(ts *trace.TraceStorage) *TraceHandler {
	return &TraceHandler{storage: ts}
}

type ListTracesRequest struct {
	SessionID string `form:"session_id"`
	Limit     int    `form:"limit"`
}

func (h *TraceHandler) ListTraces(c *gin.Context) {
	if h.storage == nil {
		respondInternalErrorWithSuccess(c, "trace storage not initialized")
		return
	}

	var req ListTracesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid query parameters")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = c.Query("session_id")
	}

	if sessionID == "" {
		respondBadRequestWithSuccess(c, "session_id is required")
		return
	}

	traces, err := h.storage.ListTraces(c.Request.Context(), sessionID, limit)
	if err != nil {
		slog.Error("failed to list traces", "error", err, "session_id", sessionID)
		respondInternalErrorWithSuccess(c, "Failed to list traces")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(traces),
		"traces":  traces,
	})
}

func (h *TraceHandler) GetTrace(c *gin.Context) {
	if h.storage == nil {
		respondInternalErrorWithSuccess(c, "trace storage not initialized")
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		respondBadRequestWithSuccess(c, "trace_id is required")
		return
	}

	trace, err := h.storage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		respondNotFoundWithSuccess(c, "Trace not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"trace":   trace,
	})
}

func (h *TraceHandler) GetTraceReplay(c *gin.Context) {
	if h.storage == nil {
		respondInternalErrorWithSuccess(c, "trace storage not initialized")
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		respondBadRequestWithSuccess(c, "trace_id is required")
		return
	}

	trace, err := h.storage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		respondNotFoundWithSuccess(c, "Trace not found")
		return
	}

	spans, err := h.storage.ListSpansByTrace(c.Request.Context(), traceID)
	if err != nil {
		slog.Error("failed to retrieve spans for replay", "error", err, "trace_id", traceID)
		respondInternalErrorWithSuccess(c, "Failed to retrieve spans")
		return
	}

	spansTree := buildSpanHierarchy(spans)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"trace":      trace,
		"spans":      spans,
		"spans_tree": spansTree,
	})
}

func buildSpanHierarchy(spans []trace.Span) []map[string]interface{} {
	spanMap := make(map[string]*trace.Span)
	for i := range spans {
		spanMap[spans[i].SpanID] = &spans[i]
	}

	var rootSpans []map[string]interface{}
	childrenMap := make(map[string][]map[string]interface{})

	for i := range spans {
		span := &spans[i]
		spanData := spanToMap(span)

		if span.ParentID == "" {
			rootSpans = append(rootSpans, spanData)
		} else {
			childrenMap[span.ParentID] = append(childrenMap[span.ParentID], spanData)
		}
	}

	attachChildren(rootSpans, childrenMap)

	return rootSpans
}

func spanToMap(span *trace.Span) map[string]interface{} {
	result := map[string]interface{}{
		"span_id":    span.SpanID,
		"trace_id":   span.TraceID,
		"parent_id":  span.ParentID,
		"name":       span.Name,
		"start_time": span.StartTime,
		"end_time":   span.EndTime,
		"inputs":     span.Inputs,
		"outputs":    span.Outputs,
		"metadata":   span.Metadata,
		"status":     span.Status,
	}

	if span.EndTime != nil {
		duration := span.EndTime.Sub(span.StartTime)
		result["duration_ms"] = duration.Milliseconds()
	}

	return result
}

func attachChildren(spans []map[string]interface{}, childrenMap map[string][]map[string]interface{}) {
	for _, span := range spans {
		spanID, _ := span["span_id"].(string)
		if children, ok := childrenMap[spanID]; ok {
			span["children"] = children
			attachChildren(children, childrenMap)
		}
	}
}

func (h *TraceHandler) GetSpan(c *gin.Context) {
	if h.storage == nil {
		respondInternalErrorWithSuccess(c, "trace storage not initialized")
		return
	}

	spanID := c.Param("span_id")
	if spanID == "" {
		respondBadRequestWithSuccess(c, "span_id is required")
		return
	}

	span, err := h.storage.RetrieveSpan(c.Request.Context(), spanID)
	if err != nil {
		respondNotFoundWithSuccess(c, "Span not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"span":    span,
	})
}

func (h *TraceHandler) GetTraceStats(c *gin.Context) {
	if h.storage == nil {
		respondInternalErrorWithSuccess(c, "trace storage not initialized")
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		respondBadRequestWithSuccess(c, "trace_id is required")
		return
	}

	trace, err := h.storage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		respondNotFoundWithSuccess(c, "Trace not found")
		return
	}

	spans, err := h.storage.ListSpansByTrace(c.Request.Context(), traceID)
	if err != nil {
		slog.Error("failed to retrieve spans for stats", "error", err, "trace_id", traceID)
		respondInternalErrorWithSuccess(c, "Failed to retrieve spans")
		return
	}

	stats := calculateTraceStats(trace, spans)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func calculateTraceStats(t *trace.Trace, spans []trace.Span) map[string]interface{} {
	stats := map[string]interface{}{
		"trace_id":   t.TraceID,
		"span_count": len(spans),
		"status":     t.Status,
	}

	if t.EndTime != nil {
		totalDuration := t.EndTime.Sub(t.StartTime)
		stats["total_duration_ms"] = totalDuration.Milliseconds()
	}

	spansByName := make(map[string]int)
	spansByStatus := make(map[string]int)
	var totalSpanDuration int64

	for _, span := range spans {
		spansByName[span.Name]++
		spansByStatus[span.Status]++

		if span.EndTime != nil {
			duration := span.EndTime.Sub(span.StartTime)
			totalSpanDuration += duration.Milliseconds()
		}
	}

	stats["spans_by_name"] = spansByName
	stats["spans_by_status"] = spansByStatus
	stats["total_span_duration_ms"] = totalSpanDuration

	return stats
}
