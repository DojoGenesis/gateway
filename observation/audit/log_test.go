package audit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/observation/audit"
)

func newTestLog(t *testing.T) audit.AuditLog {
	t.Helper()
	log, err := audit.NewSQLiteAuditLog(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteAuditLog: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestNewSQLiteAuditLog(t *testing.T) {
	log, err := audit.NewSQLiteAuditLog(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteAuditLog returned error: %v", err)
	}
	if log == nil {
		t.Fatal("NewSQLiteAuditLog returned nil")
	}
	if err := log.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestRecordAndQuery(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	entry := audit.AuditEntry{
		ID:        "test-1",
		Timestamp: time.Now().UTC(),
		AgentID:   "agent-1",
		Action:    audit.ActionToolExecution,
		Tool:      "web_search",
		ToolArgs:  map[string]interface{}{"query": "test"},
		Duration:  150 * time.Millisecond,
	}

	if err := log.Record(ctx, entry); err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := log.Query(ctx, audit.AuditFilter{AgentID: "agent-1"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Query: got %d entries, want 1", len(entries))
	}
	if entries[0].ID != "test-1" {
		t.Errorf("Query entry ID: got %q, want %q", entries[0].ID, "test-1")
	}
	if entries[0].Action != audit.ActionToolExecution {
		t.Errorf("Query entry action: got %q, want %q", entries[0].Action, audit.ActionToolExecution)
	}
}

func TestQueryByAction(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	log.Record(ctx, audit.AuditEntry{ID: "1", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionToolExecution, Tool: "t1"})
	log.Record(ctx, audit.AuditEntry{ID: "2", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionAgentSpawn})
	log.Record(ctx, audit.AuditEntry{ID: "3", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionToolExecution, Tool: "t2"})

	entries, err := log.Query(ctx, audit.AuditFilter{Actions: []audit.Action{audit.ActionToolExecution}})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Query by action: got %d entries, want 2", len(entries))
	}
}

func TestQueryByTool(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	log.Record(ctx, audit.AuditEntry{ID: "1", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionToolExecution, Tool: "web_search"})
	log.Record(ctx, audit.AuditEntry{ID: "2", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionToolExecution, Tool: "file_read"})

	entries, err := log.Query(ctx, audit.AuditFilter{Tool: "web_search"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Query by tool: got %d entries, want 1", len(entries))
	}
}

func TestQueryWithLimit(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		log.Record(ctx, audit.AuditEntry{
			ID:        "entry-" + string(rune('0'+i)),
			Timestamp: time.Now().UTC(),
			AgentID:   "a",
			Action:    audit.ActionToolExecution,
		})
	}

	entries, err := log.Query(ctx, audit.AuditFilter{Limit: 3})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Query with limit: got %d entries, want 3", len(entries))
	}
}

func TestExportJSON(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	log.Record(ctx, audit.AuditEntry{
		ID:        "export-1",
		Timestamp: time.Now().UTC(),
		AgentID:   "agent-1",
		Action:    audit.ActionToolExecution,
		Tool:      "test_tool",
	})

	var buf bytes.Buffer
	if err := log.Export(ctx, audit.AuditFilter{}, audit.ExportJSON, &buf); err != nil {
		t.Fatalf("Export JSON: %v", err)
	}

	var entries []audit.AuditEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("Unmarshal JSON export: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("JSON export: got %d entries, want 1", len(entries))
	}
}

func TestExportCSV(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()

	log.Record(ctx, audit.AuditEntry{
		ID:        "export-1",
		Timestamp: time.Now().UTC(),
		AgentID:   "agent-1",
		Action:    audit.ActionToolExecution,
		Tool:      "test_tool",
	})

	var buf bytes.Buffer
	if err := log.Export(ctx, audit.AuditFilter{}, audit.ExportCSV, &buf); err != nil {
		t.Fatalf("Export CSV: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 { // header + 1 row
		t.Fatalf("CSV export: got %d lines, want 2", len(lines))
	}
}
