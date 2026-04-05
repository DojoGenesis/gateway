package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock D1 server for audit_log
// ---------------------------------------------------------------------------

type auditMockDB struct {
	rows []auditMockRow
}

type auditMockRow struct {
	ID           string
	Timestamp    string
	AgentID      string
	Action       string
	Tool         string
	ToolArgs     string
	Capabilities string
	ResultHash   string
	DurationNs   float64
	Metadata     string
}

func newAuditMockDB() *auditMockDB {
	return &auditMockDB{}
}

func newAuditMockServer(db *auditMockDB) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			SQL    string `json:"sql"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		rows := dispatchAudit(db, req.SQL, req.Params)

		resp := map[string]any{
			"success": true,
			"errors":  []any{},
			"result": []map[string]any{
				{
					"results": rows,
					"success": true,
				},
			},
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
}

// dispatchAudit handles the exact SQL patterns emitted by d1_log.go.
func dispatchAudit(db *auditMockDB, sql string, params []any) []map[string]any {
	str := func(i int) string {
		if i >= len(params) {
			return ""
		}
		s, _ := params[i].(string)
		return s
	}
	num := func(i int) float64 {
		if i >= len(params) {
			return 0
		}
		switch v := params[i].(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		case json.Number:
			f, _ := v.Float64()
			return f
		}
		return 0
	}

	switch {
	// Record: INSERT INTO audit_log ...
	case hasAuditPrefix(sql, "INSERT INTO audit_log"):
		row := auditMockRow{
			ID:           str(0),
			Timestamp:    str(1),
			AgentID:      str(2),
			Action:       str(3),
			Tool:         str(4),
			ToolArgs:     str(5),
			Capabilities: str(6),
			ResultHash:   str(7),
			DurationNs:   num(8),
			Metadata:     str(9),
		}
		// Upsert: replace existing entry with same ID.
		for i, existing := range db.rows {
			if existing.ID == row.ID {
				db.rows[i] = row
				return nil
			}
		}
		db.rows = append(db.rows, row)
		return nil

	// Query: SELECT id, timestamp, agent_id, ... FROM audit_log [WHERE ...] ORDER BY ...
	case hasAuditPrefix(sql, "SELECT id, timestamp, agent_id"):
		// The mock returns all rows — filter logic is tested via the SQLite store.
		// D1 tests focus on HTTP transport and marshaling correctness.
		var rows []map[string]any
		for _, r := range db.rows {
			rows = append(rows, map[string]any{
				"id":                    r.ID,
				"timestamp":             r.Timestamp,
				"agent_id":              r.AgentID,
				"action":                r.Action,
				"tool":                  r.Tool,
				"tool_args":             r.ToolArgs,
				"capabilities_granted":  r.Capabilities,
				"result_hash":           r.ResultHash,
				"duration_ns":           r.DurationNs,
				"metadata":              r.Metadata,
			})
		}
		return rows
	}

	return nil
}

func hasAuditPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newD1TestAuditLog(t *testing.T) (AuditLog, *auditMockDB, *httptest.Server) {
	t.Helper()
	db := newAuditMockDB()
	srv := newAuditMockServer(db)
	log, err := NewD1AuditLog(D1Config{
		AccountID:  "test-account",
		DatabaseID: "test-db",
		APIToken:   "test-token",
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("NewD1AuditLog: %v", err)
	}
	t.Cleanup(srv.Close)
	return log, db, srv
}

func sampleEntry(id, agentID string, action Action) AuditEntry {
	return AuditEntry{
		ID:         id,
		Timestamp:  time.Now().UTC(),
		AgentID:    agentID,
		Action:     action,
		Tool:       "test_tool",
		ToolArgs:   map[string]interface{}{"key": "value"},
		ResultHash: "abc123",
		Duration:   50 * time.Millisecond,
		Metadata:   map[string]interface{}{"env": "test"},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestD1AuditLog_RecordAndQuery(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	ctx := context.Background()

	e := sampleEntry("entry-1", "agent-a", ActionToolExecution)
	if err := log.Record(ctx, e); err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := log.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Query: got %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.ID != "entry-1" {
		t.Errorf("ID: got %q, want %q", got.ID, "entry-1")
	}
	if got.AgentID != "agent-a" {
		t.Errorf("AgentID: got %q, want %q", got.AgentID, "agent-a")
	}
	if got.Action != ActionToolExecution {
		t.Errorf("Action: got %q, want %q", got.Action, ActionToolExecution)
	}
	if got.ResultHash != "abc123" {
		t.Errorf("ResultHash: got %q, want %q", got.ResultHash, "abc123")
	}
	if got.Duration != 50*time.Millisecond {
		t.Errorf("Duration: got %v, want %v", got.Duration, 50*time.Millisecond)
	}
}

func TestD1AuditLog_RecordMissingID(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	err := log.Record(context.Background(), AuditEntry{AgentID: "x", Action: ActionToolExecution})
	if err == nil {
		t.Error("Record with empty ID: expected error")
	}
}

func TestD1AuditLog_QueryEmpty(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	entries, err := log.Query(context.Background(), AuditFilter{})
	if err != nil {
		t.Fatalf("Query empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Query empty: got %d entries, want 0", len(entries))
	}
}

func TestD1AuditLog_MultipleEntries(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	ctx := context.Background()

	for i, id := range []string{"e1", "e2", "e3"} {
		e := sampleEntry(id, "agent-x", ActionToolExecution)
		_ = i
		if err := log.Record(ctx, e); err != nil {
			t.Fatalf("Record %s: %v", id, err)
		}
	}

	entries, err := log.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Query: got %d entries, want 3", len(entries))
	}
}

func TestD1AuditLog_ToolArgsMarshal(t *testing.T) {
	// Verify ToolArgs survive the JSON round-trip through the mock.
	log, _, _ := newD1TestAuditLog(t)
	ctx := context.Background()

	e := AuditEntry{
		ID:        "ta-1",
		Timestamp: time.Now().UTC(),
		AgentID:   "agent-b",
		Action:    ActionToolExecution,
		Tool:      "complex_tool",
		ToolArgs:  map[string]interface{}{"nested": map[string]interface{}{"x": float64(42)}},
	}
	if err := log.Record(ctx, e); err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := log.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Query: no entries returned")
	}
	if entries[0].ToolArgs == nil {
		t.Error("ToolArgs: expected non-nil after round-trip")
	}
}

func TestD1AuditLog_ExportJSON(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	ctx := context.Background()

	if err := log.Record(ctx, sampleEntry("json-1", "agent-c", ActionAgentSpawn)); err != nil {
		t.Fatalf("Record: %v", err)
	}

	var buf bytes.Buffer
	if err := log.Export(ctx, AuditFilter{}, ExportJSON, &buf); err != nil {
		t.Fatalf("Export JSON: %v", err)
	}

	var entries []AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("Unmarshal JSON export: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("JSON export: got %d entries, want 1", len(entries))
	}
}

func TestD1AuditLog_ExportCSV(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	ctx := context.Background()

	if err := log.Record(ctx, sampleEntry("csv-1", "agent-d", ActionSkillInvocation)); err != nil {
		t.Fatalf("Record: %v", err)
	}

	var buf bytes.Buffer
	if err := log.Export(ctx, AuditFilter{}, ExportCSV, &buf); err != nil {
		t.Fatalf("Export CSV: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 { // header + 1 data row
		t.Errorf("CSV export: got %d lines, want 2", len(lines))
	}
	if !strings.HasPrefix(lines[0], "id,") {
		t.Errorf("CSV header: got %q, want prefix %q", lines[0], "id,")
	}
	if !strings.Contains(lines[1], "csv-1") {
		t.Errorf("CSV row: got %q, want entry ID csv-1", lines[1])
	}
}

func TestD1AuditLog_ExportUnknownFormat(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	err := log.Export(context.Background(), AuditFilter{}, ExportFormat("xml"), &bytes.Buffer{})
	if err == nil {
		t.Error("Export unknown format: expected error")
	}
}

func TestD1AuditLog_Close(t *testing.T) {
	log, _, _ := newD1TestAuditLog(t)
	if err := log.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNewD1AuditLog_MissingConfig(t *testing.T) {
	_, err := NewD1AuditLog(D1Config{})
	if err == nil {
		t.Error("NewD1AuditLog with empty config: expected error")
	}
}
