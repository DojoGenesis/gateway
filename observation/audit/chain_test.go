package audit_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/observation/audit"
)

func openFileLog(t *testing.T, path string) audit.AuditLog {
	t.Helper()
	log, err := audit.NewSQLiteAuditLog(path)
	if err != nil {
		t.Fatalf("NewSQLiteAuditLog: %v", err)
	}
	return log
}

func newRecorder(t *testing.T, log audit.AuditLog, opts ...audit.RecorderOption) *audit.Recorder {
	t.Helper()
	r, err := audit.NewRecorder(context.Background(), log, opts...)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	return r
}

func mustVerify(t *testing.T, log audit.AuditLog) audit.VerifyReport {
	t.Helper()
	rep, err := audit.Verify(context.Background(), log)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return rep
}

func TestRecorderChainsEntries(t *testing.T) {
	log := newTestLog(t)
	r := newRecorder(t, log)
	ctx := context.Background()

	var stored []audit.AuditEntry
	for _, tool := range []string{"a", "b", "c"} {
		e, err := r.Record(ctx, audit.AuditEntry{Action: audit.ActionToolExecution, Tool: tool})
		if err != nil {
			t.Fatalf("Record: %v", err)
		}
		stored = append(stored, e)
	}

	for i, e := range stored {
		if e.Seq != int64(i+1) {
			t.Errorf("entry %d: seq %d", i, e.Seq)
		}
		if e.ID == "" || e.Timestamp.IsZero() || e.AgentID != audit.DefaultAgentID {
			t.Errorf("entry %d: defaults not filled: %+v", i, e)
		}
		if i == 0 && e.PrevHash != "" {
			t.Errorf("first entry has prev_hash %q", e.PrevHash)
		}
		if i > 0 && e.PrevHash != stored[i-1].EntryHash {
			t.Errorf("entry %d: prev_hash does not link to entry %d", i, i-1)
		}
	}

	rep := mustVerify(t, log)
	if !rep.OK || rep.Chained != 3 || rep.HeadSeq != 3 || rep.HeadHash != stored[2].EntryHash {
		t.Fatalf("Verify: %+v", rep)
	}
	if seq, head := r.Head(); seq != 3 || head != stored[2].EntryHash {
		t.Errorf("Head: %d %q", seq, head)
	}
}

func TestRecorderResumesAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.db")
	ctx := context.Background()

	log := openFileLog(t, path)
	r := newRecorder(t, log)
	_, _ = r.Record(ctx, audit.AuditEntry{Action: audit.ActionAgentSpawn, AgentID: "agent-1"})
	second, _ := r.Record(ctx, audit.AuditEntry{Action: audit.ActionAgentSpawn, AgentID: "agent-2"})
	_ = log.Close()

	log = openFileLog(t, path)
	t.Cleanup(func() { _ = log.Close() })
	r = newRecorder(t, log)
	if seq, head := r.Head(); seq != 2 || head != second.EntryHash {
		t.Fatalf("resumed head: %d %q, want 2 %q", seq, head, second.EntryHash)
	}
	third, err := r.Record(ctx, audit.AuditEntry{Action: audit.ActionMemoryAccess})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if third.Seq != 3 || third.PrevHash != second.EntryHash {
		t.Fatalf("third entry not linked: %+v", third)
	}
	if rep := mustVerify(t, log); !rep.OK || rep.Chained != 3 {
		t.Fatalf("Verify after reopen: %+v", rep)
	}
}

// tamper opens the audit database directly and runs stmt, as an attacker with
// file access would.
func tamper(t *testing.T, path, stmt string, args ...any) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(stmt, args...); err != nil {
		t.Fatalf("tamper %q: %v", stmt, err)
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	cases := []struct {
		name string
		stmt string
		want string
	}{
		{"edited field", `UPDATE audit_log SET tool = 'something_else' WHERE seq = 2`, "seq 2: entry_hash does not match"},
		{"deleted entry", `DELETE FROM audit_log WHERE seq = 2`, "seq 3: entries 2..2 missing"},
		{"swapped order", `UPDATE audit_log SET seq = CASE seq WHEN 2 THEN 3 WHEN 3 THEN 2 END WHERE seq IN (2, 3)`, "prev_hash does not match"},
		{"rehashed edit", `UPDATE audit_log SET tool = 'x', entry_hash = 'f00d' WHERE seq = 2`, "seq 3: prev_hash does not match"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "audit.db")
			ctx := context.Background()
			log := openFileLog(t, path)
			t.Cleanup(func() { _ = log.Close() })
			r := newRecorder(t, log)
			for _, tool := range []string{"a", "b", "c", "d"} {
				if _, err := r.Record(ctx, audit.AuditEntry{Action: audit.ActionToolExecution, Tool: tool}); err != nil {
					t.Fatalf("Record: %v", err)
				}
			}
			if rep := mustVerify(t, log); !rep.OK {
				t.Fatalf("clean log failed: %+v", rep)
			}

			tamper(t, path, tc.stmt)

			rep := mustVerify(t, log)
			if rep.OK {
				t.Fatalf("tampered log verified OK")
			}
			if !strings.Contains(strings.Join(rep.Problems, "\n"), tc.want) {
				t.Errorf("problems %q do not mention %q", rep.Problems, tc.want)
			}
		})
	}
}

func TestVerifyRoundTripsTypedValues(t *testing.T) {
	log := newTestLog(t)
	r := newRecorder(t, log, audit.WithToolArgs(true))
	ctx := context.Background()

	_, err := r.Record(ctx, audit.AuditEntry{
		Action:              audit.ActionCapabilityGrant,
		ToolArgs:            map[string]interface{}{"n": 3, "big": int64(1) << 53, "f": 0.1, "nested": map[string]int{"k": 1}},
		CapabilitiesGranted: []string{},
		Metadata:            map[string]interface{}{},
		Duration:            1500 * time.Microsecond,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if rep := mustVerify(t, log); !rep.OK {
		t.Fatalf("Verify: %+v", rep)
	}
}

func TestVerifyCountsUnchainedEntries(t *testing.T) {
	log := newTestLog(t)
	ctx := context.Background()
	_ = log.Record(ctx, audit.AuditEntry{ID: "legacy", Timestamp: time.Now().UTC(), AgentID: "a", Action: audit.ActionToolExecution})
	r := newRecorder(t, log)
	if _, err := r.Record(ctx, audit.AuditEntry{Action: audit.ActionToolExecution}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	rep := mustVerify(t, log)
	if !rep.OK || rep.Chained != 1 || rep.Unchained != 1 {
		t.Fatalf("Verify: %+v", rep)
	}
}

func TestToolExecutionHashesArgsByDefault(t *testing.T) {
	log := newTestLog(t)
	r := newRecorder(t, log)
	ctx := audit.WithAgentID(context.Background(), "agent-7")

	r.ToolExecution(ctx, "write_file", map[string]interface{}{"path": "/tmp/x", "content": "secret"}, map[string]interface{}{"ok": true}, nil, 2*time.Millisecond)
	r.ToolExecution(ctx, "run_command", nil, nil, errors.New("refused"), 0)

	entries, err := log.Query(context.Background(), audit.AuditFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries", len(entries))
	}
	byTool := map[string]audit.AuditEntry{}
	for _, e := range entries {
		byTool[e.Tool] = e
	}

	ok := byTool["write_file"]
	if ok.AgentID != "agent-7" || ok.ToolArgs != nil || ok.ResultHash == "" {
		t.Errorf("write_file entry: %+v", ok)
	}
	if ok.Metadata["args_sha256"] == "" || ok.Metadata["outcome"] != "ok" {
		t.Errorf("write_file metadata: %v", ok.Metadata)
	}
	for _, v := range ok.Metadata {
		if s, isStr := v.(string); isStr && strings.Contains(s, "secret") {
			t.Errorf("raw argument value leaked into metadata: %v", ok.Metadata)
		}
	}

	bad := byTool["run_command"]
	if bad.Metadata["outcome"] != "error" || bad.Metadata["error"] != "refused" || bad.ResultHash != "" {
		t.Errorf("run_command entry: %+v", bad)
	}
	if rep := mustVerify(t, log); !rep.OK {
		t.Fatalf("Verify: %+v", rep)
	}
}

func TestNilRecorderIsNoop(t *testing.T) {
	var r *audit.Recorder
	ctx := context.Background()
	r.ToolExecution(ctx, "t", nil, nil, nil, 0)
	r.AgentSpawn(ctx, "a", nil)
	r.MemoryAccess(ctx, "get", nil, nil, 0)
	if _, err := r.Record(ctx, audit.AuditEntry{}); err != nil {
		t.Fatalf("nil Record: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("nil Close: %v", err)
	}
}

func TestRecorderConcurrentWritesStayChained(t *testing.T) {
	log := newTestLog(t)
	r := newRecorder(t, log)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				r.MemoryAccess(ctx, "search", map[string]interface{}{"results": j}, nil, 0)
			}
		}()
	}
	wg.Wait()

	rep := mustVerify(t, log)
	if !rep.OK || rep.Chained != 80 || rep.HeadSeq != 80 {
		t.Fatalf("Verify: %+v", rep)
	}
}
