package trace

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTraceTestDB(t *testing.T) (*TraceStorage, *sql.DB, string) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpfile := tmpDir + "/test_trace.db"

	// Use connection string parameters for WAL mode and busy timeout.
	// modernc.org/sqlite supports _pragma parameters in the DSN.
	dsn := tmpfile + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Limit to one open connection to serialize SQLite writes and avoid SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	ts, err := NewTraceStorage(db)
	if err != nil {
		t.Fatalf("failed to create trace storage: %v", err)
	}

	return ts, db, tmpfile
}

func teardownTraceTestDB(t *testing.T, db *sql.DB, dbPath string) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Errorf("failed to close database: %v", err)
	}
	// dbPath is inside t.TempDir() which is auto-cleaned, but remove explicitly too
	os.Remove(dbPath)
}

func TestTraceStorage_TraceOperations(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID:  "test-session",
		StartTime:  time.Now(),
		RootSpanID: "root-span-1",
		Status:     "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	if trace.TraceID == "" {
		t.Errorf("expected trace ID to be generated")
	}

	retrieved, err := ts.RetrieveTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("failed to retrieve trace: %v", err)
	}

	if retrieved.SessionID != trace.SessionID {
		t.Errorf("expected session_id '%s', got '%s'", trace.SessionID, retrieved.SessionID)
	}

	if retrieved.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", retrieved.Status)
	}

	if retrieved.RootSpanID != "root-span-1" {
		t.Errorf("expected root_span_id 'root-span-1', got '%s'", retrieved.RootSpanID)
	}
}

func TestTraceStorage_ListTraces(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	traces := []Trace{
		{
			SessionID: "test-session",
			StartTime: time.Now().Add(-2 * time.Hour),
			Status:    "completed",
		},
		{
			SessionID: "test-session",
			StartTime: time.Now().Add(-1 * time.Hour),
			Status:    "active",
		},
		{
			SessionID: "test-session",
			StartTime: time.Now(),
			Status:    "active",
		},
	}

	for i := range traces {
		if err := ts.StoreTrace(ctx, &traces[i]); err != nil {
			t.Fatalf("failed to store trace: %v", err)
		}
	}

	retrieved, err := ts.ListTraces(ctx, "test-session", 10)
	if err != nil {
		t.Fatalf("failed to list traces: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("expected 3 traces, got %d", len(retrieved))
	}

	if !retrieved[0].StartTime.After(retrieved[1].StartTime) {
		t.Errorf("traces should be ordered by start_time descending")
	}
}

func TestTraceStorage_UpdateTraceStatus(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "test-session",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	endTime := time.Now()
	if err := ts.UpdateTraceStatus(ctx, trace.TraceID, "completed", endTime); err != nil {
		t.Fatalf("failed to update trace status: %v", err)
	}

	retrieved, err := ts.RetrieveTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("failed to retrieve trace: %v", err)
	}

	if retrieved.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", retrieved.Status)
	}

	if retrieved.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}
}

func TestTraceStorage_SpanOperations(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "test-session",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	span := &Span{
		TraceID:   trace.TraceID,
		Name:      "test_span",
		StartTime: time.Now(),
		Inputs: map[string]interface{}{
			"query": "test query",
			"param": 123,
		},
		Metadata: map[string]interface{}{
			"model": "test-model",
		},
		Status: "running",
	}

	if err := ts.StoreSpan(ctx, span); err != nil {
		t.Fatalf("failed to store span: %v", err)
	}

	if span.SpanID == "" {
		t.Errorf("expected span ID to be generated")
	}

	retrieved, err := ts.RetrieveSpan(ctx, span.SpanID)
	if err != nil {
		t.Fatalf("failed to retrieve span: %v", err)
	}

	if retrieved.Name != "test_span" {
		t.Errorf("expected name 'test_span', got '%s'", retrieved.Name)
	}

	if retrieved.TraceID != trace.TraceID {
		t.Errorf("expected trace_id '%s', got '%s'", trace.TraceID, retrieved.TraceID)
	}

	if retrieved.Inputs["query"] != "test query" {
		t.Errorf("expected input query 'test query', got '%v'", retrieved.Inputs["query"])
	}

	if retrieved.Metadata["model"] != "test-model" {
		t.Errorf("expected metadata model 'test-model', got '%v'", retrieved.Metadata["model"])
	}
}

func TestTraceStorage_ListSpansByTrace(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "test-session",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	spans := []Span{
		{
			TraceID:   trace.TraceID,
			Name:      "span1",
			StartTime: time.Now().Add(-2 * time.Second),
			Status:    "completed",
		},
		{
			TraceID:   trace.TraceID,
			ParentID:  "parent-span",
			Name:      "span2",
			StartTime: time.Now().Add(-1 * time.Second),
			Status:    "running",
		},
		{
			TraceID:   trace.TraceID,
			Name:      "span3",
			StartTime: time.Now(),
			Status:    "running",
		},
	}

	for i := range spans {
		if err := ts.StoreSpan(ctx, &spans[i]); err != nil {
			t.Fatalf("failed to store span: %v", err)
		}
	}

	retrieved, err := ts.ListSpansByTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("failed to list spans: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("expected 3 spans, got %d", len(retrieved))
	}

	if !retrieved[0].StartTime.Before(retrieved[1].StartTime) {
		t.Errorf("spans should be ordered by start_time ascending")
	}

	if retrieved[1].ParentID != "parent-span" {
		t.Errorf("expected parent_id 'parent-span', got '%s'", retrieved[1].ParentID)
	}
}

func TestTraceStorage_UpdateSpanStatus(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "test-session",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	span := &Span{
		TraceID:   trace.TraceID,
		Name:      "test_span",
		StartTime: time.Now(),
		Status:    "running",
	}

	if err := ts.StoreSpan(ctx, span); err != nil {
		t.Fatalf("failed to store span: %v", err)
	}

	endTime := time.Now()
	if err := ts.UpdateSpanStatus(ctx, span.SpanID, "completed", endTime); err != nil {
		t.Fatalf("failed to update span status: %v", err)
	}

	retrieved, err := ts.RetrieveSpan(ctx, span.SpanID)
	if err != nil {
		t.Fatalf("failed to retrieve span: %v", err)
	}

	if retrieved.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", retrieved.Status)
	}

	if retrieved.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}
}

func TestTraceStorage_SchemaInitialization(t *testing.T) {
	tmpfile := "/tmp/test_trace_schema_init.db"

	db, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		t.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	trace := &Trace{
		SessionID: "schema-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Errorf("schema tables not properly initialized: %v", err)
	}

	db2, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db2.Close()

	ts2, err := NewTraceStorage(db2)
	if err != nil {
		t.Fatalf("failed to initialize trace storage on existing db: %v", err)
	}

	if trace.TraceID == "" {
		t.Fatalf("trace ID was not generated")
	}

	retrieved, err := ts2.RetrieveTrace(ctx, trace.TraceID)
	if err != nil {
		t.Errorf("failed to retrieve trace after reinitializing: %v", err)
	}

	if retrieved.SessionID != trace.SessionID {
		t.Errorf("data persisted incorrectly across reopening")
	}
}

func TestTraceStorage_ForeignKeyConstraint(t *testing.T) {
	tmpfile := "/tmp/test_trace_fk.db"

	db, err := sql.Open("sqlite", tmpfile+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		t.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	span := &Span{
		TraceID:   "non-existent-trace",
		Name:      "orphan_span",
		StartTime: time.Now(),
		Status:    "running",
	}

	err = ts.StoreSpan(ctx, span)
	if err == nil {
		t.Errorf("expected foreign key constraint error when storing span with non-existent trace")
	}
}

func TestTraceStorage_SpanWithComplexMetadata(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "test-session",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	span := &Span{
		TraceID:   trace.TraceID,
		Name:      "complex_span",
		StartTime: time.Now(),
		Inputs: map[string]interface{}{
			"query":  "complex query",
			"params": []string{"param1", "param2"},
			"nested": map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		Outputs: map[string]interface{}{
			"result": "success",
			"data":   []interface{}{1, 2, 3},
		},
		Metadata: map[string]interface{}{
			"model":       "test-model",
			"tokens":      1000,
			"cost":        0.01,
			"provider":    "test-provider",
			"temperature": 0.7,
		},
		Status: "completed",
	}

	endTime := time.Now()
	span.EndTime = &endTime

	if err := ts.StoreSpan(ctx, span); err != nil {
		t.Fatalf("failed to store complex span: %v", err)
	}

	retrieved, err := ts.RetrieveSpan(ctx, span.SpanID)
	if err != nil {
		t.Fatalf("failed to retrieve complex span: %v", err)
	}

	if retrieved.Inputs["query"] != "complex query" {
		t.Errorf("inputs not properly persisted")
	}

	if retrieved.Outputs["result"] != "success" {
		t.Errorf("outputs not properly persisted")
	}

	if retrieved.Metadata["model"] != "test-model" {
		t.Errorf("metadata not properly persisted")
	}

	nestedMap, ok := retrieved.Inputs["nested"].(map[string]interface{})
	if !ok {
		t.Errorf("nested map structure not preserved")
	} else {
		if nestedMap["key2"] != float64(42) {
			t.Errorf("nested values not preserved correctly")
		}
	}
}

func TestTraceStorage_CascadeDelete(t *testing.T) {
	tmpfile := "/tmp/test_trace_cascade.db"

	db, err := sql.Open("sqlite", tmpfile+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		t.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	trace := &Trace{
		SessionID: "cascade-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	span := &Span{
		TraceID:   trace.TraceID,
		Name:      "test_span",
		StartTime: time.Now(),
		Status:    "running",
	}

	if err := ts.StoreSpan(ctx, span); err != nil {
		t.Fatalf("failed to store span: %v", err)
	}

	_, err = db.ExecContext(ctx, "DELETE FROM traces WHERE trace_id = ?", trace.TraceID)
	if err != nil {
		t.Fatalf("failed to delete trace: %v", err)
	}

	_, err = ts.RetrieveSpan(ctx, span.SpanID)
	if err == nil {
		t.Errorf("expected span to be deleted via CASCADE")
	}
}

func BenchmarkTraceStorage_StoreTrace(b *testing.B) {
	tmpfile := "/tmp/bench_store_trace.db"

	db, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		b.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trace := &Trace{
			SessionID: "bench-session",
			StartTime: time.Now(),
			Status:    "active",
		}
		if err := ts.StoreTrace(ctx, trace); err != nil {
			b.Fatalf("failed to store trace: %v", err)
		}
	}
}

func BenchmarkTraceStorage_StoreSpan(b *testing.B) {
	tmpfile := "/tmp/bench_store_span.db"

	db, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		b.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	trace := &Trace{
		SessionID: "bench-session",
		StartTime: time.Now(),
		Status:    "active",
	}
	if err := ts.StoreTrace(ctx, trace); err != nil {
		b.Fatalf("failed to store trace: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := &Span{
			TraceID:   trace.TraceID,
			Name:      "bench_span",
			StartTime: time.Now(),
			Inputs: map[string]interface{}{
				"query": "test query",
			},
			Status: "running",
		}
		if err := ts.StoreSpan(ctx, span); err != nil {
			b.Fatalf("failed to store span: %v", err)
		}
	}
}

func BenchmarkTraceStorage_RetrieveTrace(b *testing.B) {
	tmpfile := "/tmp/bench_retrieve_trace.db"

	db, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(tmpfile)

	ts, err := NewTraceStorage(db)
	if err != nil {
		b.Fatalf("failed to create trace storage: %v", err)
	}

	ctx := context.Background()

	trace := &Trace{
		SessionID: "bench-session",
		StartTime: time.Now(),
		Status:    "active",
	}
	if err := ts.StoreTrace(ctx, trace); err != nil {
		b.Fatalf("failed to store trace: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ts.RetrieveTrace(ctx, trace.TraceID)
		if err != nil {
			b.Fatalf("failed to retrieve trace: %v", err)
		}
	}
}

func TestTraceStorage_MalformedJSON(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "malformed-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO spans (span_id, trace_id, parent_id, name, start_time, inputs, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "malformed-span-1", trace.TraceID, "", "test_span", time.Now(), "invalid json {{{", "running")

	if err != nil {
		t.Fatalf("failed to insert malformed span: %v", err)
	}

	_, err = ts.ListSpansByTrace(ctx, trace.TraceID)
	if err == nil {
		t.Errorf("expected error when reading malformed JSON, got nil")
	}

	if err != nil && !contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestTraceStorage_ContextCancellation(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx, cancel := context.WithCancel(context.Background())

	trace := &Trace{
		SessionID: "cancel-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	cancel()

	_, err := ts.RetrieveTrace(ctx, trace.TraceID)
	if err == nil {
		t.Errorf("expected error when context is cancelled")
	}
}

func TestTraceStorage_ConcurrentWrites(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "concurrent-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			span := &Span{
				TraceID:   trace.TraceID,
				Name:      fmt.Sprintf("concurrent_span_%d", idx),
				StartTime: time.Now(),
				Status:    "running",
			}
			errChan <- ts.StoreSpan(ctx, span)
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("concurrent write failed: %v", err)
		}
	}

	spans, err := ts.ListSpansByTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("failed to list spans: %v", err)
	}

	if len(spans) != numGoroutines {
		t.Errorf("expected %d spans, got %d", numGoroutines, len(spans))
	}
}

func TestTraceStorage_EmptyResults(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	traces, err := ts.ListTraces(ctx, "non-existent-session", 10)
	if err != nil {
		t.Errorf("expected no error for empty result set, got: %v", err)
	}

	if len(traces) != 0 {
		t.Errorf("expected 0 traces, got %d", len(traces))
	}

	spans, err := ts.ListSpansByTrace(ctx, "non-existent-trace")
	if err != nil {
		t.Errorf("expected no error for empty result set, got: %v", err)
	}

	if len(spans) != 0 {
		t.Errorf("expected 0 spans, got %d", len(spans))
	}
}

func TestTraceStorage_LargeJSON(t *testing.T) {
	ts, db, dbPath := setupTraceTestDB(t)
	defer teardownTraceTestDB(t, db, dbPath)

	ctx := context.Background()

	trace := &Trace{
		SessionID: "large-json-test",
		StartTime: time.Now(),
		Status:    "active",
	}

	if err := ts.StoreTrace(ctx, trace); err != nil {
		t.Fatalf("failed to store trace: %v", err)
	}

	largeInputs := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeInputs[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_lots_of_data_%s", i, string(make([]byte, 1000)))
	}

	span := &Span{
		TraceID:   trace.TraceID,
		Name:      "large_span",
		StartTime: time.Now(),
		Inputs:    largeInputs,
		Status:    "running",
	}

	if err := ts.StoreSpan(ctx, span); err != nil {
		t.Fatalf("failed to store large span: %v", err)
	}

	retrieved, err := ts.RetrieveSpan(ctx, span.SpanID)
	if err != nil {
		t.Fatalf("failed to retrieve large span: %v", err)
	}

	if len(retrieved.Inputs) != 100 {
		t.Errorf("expected 100 input keys, got %d", len(retrieved.Inputs))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
