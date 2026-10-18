package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/observation/audit"
	"github.com/DojoGenesis/gateway/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The audit package existed for months with nothing outside its tests calling
// it. This pins the wiring main() does: tool calls (run and refused) and memory
// operations reach the hash-chained log, and the chain verifies.
func TestAuditWiringRecordsToolAndMemoryAccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AUDIT_DB_PATH", filepath.Join(dir, "audit.db"))

	rec := initAuditRecorder()
	require.NotNil(t, rec)
	t.Cleanup(func() { _ = rec.Close() })

	tools.SetCallObserver(rec.ToolExecution)
	t.Cleanup(func() { tools.SetCallObserver(nil) })

	const toolName = "audit_wiring_echo"
	require.NoError(t, tools.RegisterTool(&tools.ToolDefinition{
		Name: toolName,
		Function: func(ctx context.Context, p map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"echo": p["text"]}, nil
		},
	}))
	t.Cleanup(func() { _ = tools.UnregisterTool(toolName) })

	mm, err := memory.NewMemoryManager(filepath.Join(dir, "memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = mm.Close() })
	mm.SetAccessObserver(rec.MemoryAccess)

	ctx := audit.WithAgentID(context.Background(), "agent-under-test")

	_, err = tools.InvokeTool(ctx, toolName, map[string]interface{}{"text": "hello"})
	require.NoError(t, err)

	denied := tools.WithAuthorizer(ctx, func(name string) error {
		return &tools.ToolAccessError{Tool: name, Code: "forbidden", Message: "not for you"}
	})
	_, err = tools.InvokeTool(denied, toolName, map[string]interface{}{"text": "again"})
	require.Error(t, err)

	require.NoError(t, mm.StoreMemory(ctx, &memory.Memory{ID: "m1", Type: "note", Content: "private text"}))
	_, err = mm.GetMemory(ctx, "m1")
	require.NoError(t, err)

	entries, err := rec.Log().Query(context.Background(), audit.AuditFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 4)

	var tool, refused, memOps int
	for _, e := range entries {
		assert.Equal(t, "agent-under-test", e.AgentID)
		switch e.Action {
		case audit.ActionToolExecution:
			if e.Metadata["outcome"] == "ok" {
				tool++
			} else {
				refused++
				assert.Equal(t, "not for you", e.Metadata["error"])
			}
		case audit.ActionMemoryAccess:
			memOps++
			for _, v := range e.Metadata {
				assert.NotEqual(t, "private text", v, "memory content must not be logged")
			}
		}
	}
	assert.Equal(t, 1, tool)
	assert.Equal(t, 1, refused)
	assert.Equal(t, 2, memOps)

	rep, err := audit.Verify(context.Background(), rec.Log())
	require.NoError(t, err)
	assert.True(t, rep.OK, "problems: %v", rep.Problems)
	assert.Equal(t, int64(4), rep.HeadSeq)
}

func TestAuditCanBeDisabled(t *testing.T) {
	t.Setenv("AUDIT_ENABLED", "false")
	t.Setenv("AUDIT_DB_PATH", filepath.Join(t.TempDir(), "audit.db"))
	assert.Nil(t, initAuditRecorder())
}
