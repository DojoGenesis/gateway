package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func setupTraceTestRouter(t *testing.T) (*gin.Engine, *trace.TraceStorage, *sql.DB) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	storage, err := trace.NewTraceStorage(db)
	if err != nil {
		t.Fatalf("Failed to create trace storage: %v", err)
	}

	InitializeTraceHandlers(storage)

	r := gin.New()
	return r, storage, db
}

func TestHandleListTraces(t *testing.T) {
	r, storage, db := setupTraceTestRouter(t)
	defer db.Close()

	now := time.Now()
	testTrace := &trace.Trace{
		TraceID:   "trace-1",
		SessionID: "session-1",
		StartTime: now,
		Status:    "active",
	}

	if err := storage.StoreTrace(nil, testTrace); err != nil {
		t.Fatalf("Failed to store test trace: %v", err)
	}

	r.GET("/api/v1/traces", HandleListTraces)

	req, _ := http.NewRequest("GET", "/api/v1/traces?session_id=session-1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	if response["count"].(float64) != 1 {
		t.Errorf("Expected count=1, got %v", response["count"])
	}
}

func TestHandleListTraces_MissingSessionID(t *testing.T) {
	r, _, db := setupTraceTestRouter(t)
	defer db.Close()

	r.GET("/api/v1/traces", HandleListTraces)

	req, _ := http.NewRequest("GET", "/api/v1/traces", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}
}

func TestHandleGetTrace(t *testing.T) {
	r, storage, db := setupTraceTestRouter(t)
	defer db.Close()

	now := time.Now()
	testTrace := &trace.Trace{
		TraceID:   "trace-1",
		SessionID: "session-1",
		StartTime: now,
		Status:    "active",
	}

	if err := storage.StoreTrace(nil, testTrace); err != nil {
		t.Fatalf("Failed to store test trace: %v", err)
	}

	r.GET("/api/v1/traces/:trace_id", HandleGetTrace)

	req, _ := http.NewRequest("GET", "/api/v1/traces/trace-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	traceData := response["trace"].(map[string]interface{})
	if traceData["trace_id"] != "trace-1" {
		t.Errorf("Expected trace_id=trace-1, got %v", traceData["trace_id"])
	}
}

func TestHandleGetTrace_NotFound(t *testing.T) {
	r, _, db := setupTraceTestRouter(t)
	defer db.Close()

	r.GET("/api/v1/traces/:trace_id", HandleGetTrace)

	req, _ := http.NewRequest("GET", "/api/v1/traces/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleGetTraceReplay(t *testing.T) {
	r, storage, db := setupTraceTestRouter(t)
	defer db.Close()

	now := time.Now()
	testTrace := &trace.Trace{
		TraceID:   "trace-1",
		SessionID: "session-1",
		StartTime: now,
		Status:    "active",
	}

	if err := storage.StoreTrace(nil, testTrace); err != nil {
		t.Fatalf("Failed to store test trace: %v", err)
	}

	testSpan := &trace.Span{
		SpanID:    "span-1",
		TraceID:   "trace-1",
		Name:      "test_operation",
		StartTime: now,
		Status:    "completed",
	}

	if err := storage.StoreSpan(nil, testSpan); err != nil {
		t.Fatalf("Failed to store test span: %v", err)
	}

	r.GET("/api/v1/traces/:trace_id/replay", HandleGetTraceReplay)

	req, _ := http.NewRequest("GET", "/api/v1/traces/trace-1/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	spans := response["spans"].([]interface{})
	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}
}

func TestHandleGetTraceStats(t *testing.T) {
	r, storage, db := setupTraceTestRouter(t)
	defer db.Close()

	now := time.Now()
	endTime := now.Add(1 * time.Second)
	testTrace := &trace.Trace{
		TraceID:   "trace-1",
		SessionID: "session-1",
		StartTime: now,
		EndTime:   &endTime,
		Status:    "completed",
	}

	if err := storage.StoreTrace(nil, testTrace); err != nil {
		t.Fatalf("Failed to store test trace: %v", err)
	}

	spanEndTime := now.Add(500 * time.Millisecond)
	testSpan := &trace.Span{
		SpanID:    "span-1",
		TraceID:   "trace-1",
		Name:      "test_operation",
		StartTime: now,
		EndTime:   &spanEndTime,
		Status:    "completed",
	}

	if err := storage.StoreSpan(nil, testSpan); err != nil {
		t.Fatalf("Failed to store test span: %v", err)
	}

	r.GET("/api/v1/traces/:trace_id/stats", HandleGetTraceStats)

	req, _ := http.NewRequest("GET", "/api/v1/traces/trace-1/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	stats := response["stats"].(map[string]interface{})
	if stats["span_count"].(float64) != 1 {
		t.Errorf("Expected span_count=1, got %v", stats["span_count"])
	}

	if stats["status"] != "completed" {
		t.Errorf("Expected status=completed, got %v", stats["status"])
	}
}

func TestHandleGetSpan(t *testing.T) {
	r, storage, db := setupTraceTestRouter(t)
	defer db.Close()

	now := time.Now()
	testTrace := &trace.Trace{
		TraceID:   "trace-1",
		SessionID: "session-1",
		StartTime: now,
		Status:    "active",
	}

	if err := storage.StoreTrace(nil, testTrace); err != nil {
		t.Fatalf("Failed to store test trace: %v", err)
	}

	testSpan := &trace.Span{
		SpanID:    "span-1",
		TraceID:   "trace-1",
		Name:      "test_operation",
		StartTime: now,
		Status:    "running",
		Inputs: map[string]interface{}{
			"query": "test query",
		},
	}

	if err := storage.StoreSpan(nil, testSpan); err != nil {
		t.Fatalf("Failed to store test span: %v", err)
	}

	r.GET("/api/v1/spans/:span_id", HandleGetSpan)

	req, _ := http.NewRequest("GET", "/api/v1/spans/span-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	spanData := response["span"].(map[string]interface{})
	if spanData["span_id"] != "span-1" {
		t.Errorf("Expected span_id=span-1, got %v", spanData["span_id"])
	}
}

func TestTraceHandlersWithoutInitialization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	traceStorage = nil

	r := gin.New()

	tests := []struct {
		name     string
		method   string
		path     string
		handler  gin.HandlerFunc
		expected int
	}{
		{"ListTraces", "GET", "/api/v1/traces?session_id=test", HandleListTraces, http.StatusInternalServerError},
		{"GetTrace", "GET", "/api/v1/traces/trace-1", HandleGetTrace, http.StatusInternalServerError},
		{"GetTraceReplay", "GET", "/api/v1/traces/trace-1/replay", HandleGetTraceReplay, http.StatusInternalServerError},
		{"GetTraceStats", "GET", "/api/v1/traces/trace-1/stats", HandleGetTraceStats, http.StatusInternalServerError},
		{"GetSpan", "GET", "/api/v1/spans/span-1", HandleGetSpan, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.Handle(tt.method, tt.path, tt.handler)

			req, _ := http.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, w.Code)
			}
		})
	}
}

func TestBuildSpanHierarchy(t *testing.T) {
	now := time.Now()

	spans := []trace.Span{
		{
			SpanID:    "root",
			TraceID:   "trace-1",
			Name:      "root_operation",
			StartTime: now,
			Status:    "completed",
		},
		{
			SpanID:    "child-1",
			TraceID:   "trace-1",
			ParentID:  "root",
			Name:      "child_operation_1",
			StartTime: now,
			Status:    "completed",
		},
		{
			SpanID:    "child-2",
			TraceID:   "trace-1",
			ParentID:  "root",
			Name:      "child_operation_2",
			StartTime: now,
			Status:    "completed",
		},
		{
			SpanID:    "grandchild",
			TraceID:   "trace-1",
			ParentID:  "child-1",
			Name:      "grandchild_operation",
			StartTime: now,
			Status:    "completed",
		},
	}

	tree := buildSpanHierarchy(spans)

	if len(tree) != 1 {
		t.Errorf("Expected 1 root span, got %d", len(tree))
	}

	root := tree[0]
	if root["span_id"] != "root" {
		t.Errorf("Expected root span_id=root, got %v", root["span_id"])
	}

	children := root["children"].([]map[string]interface{})
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}

	if children[0]["span_id"] == "child-1" {
		grandchildren := children[0]["children"].([]map[string]interface{})
		if len(grandchildren) != 1 {
			t.Errorf("Expected 1 grandchild, got %d", len(grandchildren))
		}
	}
}
