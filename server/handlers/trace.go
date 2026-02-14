package handlers

import (
	"net/http"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/gin-gonic/gin"
)

var traceStorage *trace.TraceStorage

func InitializeTraceHandlers(ts *trace.TraceStorage) {
	traceStorage = ts
}

type ListTracesRequest struct {
	SessionID string `form:"session_id"`
	Limit     int    `form:"limit"`
}

func HandleListTraces(c *gin.Context) {
	if traceStorage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "trace storage not initialized",
		})
		return
	}

	var req ListTracesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "session_id is required",
		})
		return
	}

	traces, err := traceStorage.ListTraces(c.Request.Context(), sessionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list traces",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(traces),
		"traces":  traces,
	})
}

func HandleGetTrace(c *gin.Context) {
	if traceStorage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "trace storage not initialized",
		})
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "trace_id is required",
		})
		return
	}

	trace, err := traceStorage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Trace not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"trace":   trace,
	})
}

func HandleGetTraceReplay(c *gin.Context) {
	if traceStorage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "trace storage not initialized",
		})
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "trace_id is required",
		})
		return
	}

	trace, err := traceStorage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Trace not found",
			"details": err.Error(),
		})
		return
	}

	spans, err := traceStorage.ListSpansByTrace(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve spans",
			"details": err.Error(),
		})
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
		spanID := span["span_id"].(string)
		if children, ok := childrenMap[spanID]; ok {
			span["children"] = children
			attachChildren(children, childrenMap)
		}
	}
}

func HandleGetSpan(c *gin.Context) {
	if traceStorage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "trace storage not initialized",
		})
		return
	}

	spanID := c.Param("span_id")
	if spanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "span_id is required",
		})
		return
	}

	span, err := traceStorage.RetrieveSpan(c.Request.Context(), spanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Span not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"span":    span,
	})
}

func HandleGetTraceStats(c *gin.Context) {
	if traceStorage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "trace storage not initialized",
		})
		return
	}

	traceID := c.Param("trace_id")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "trace_id is required",
		})
		return
	}

	trace, err := traceStorage.RetrieveTrace(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Trace not found",
			"details": err.Error(),
		})
		return
	}

	spans, err := traceStorage.ListSpansByTrace(c.Request.Context(), traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve spans",
			"details": err.Error(),
		})
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
