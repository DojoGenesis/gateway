package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// denyNamed refuses exactly the named tools.
func denyNamed(names ...string) Authorizer {
	deny := map[string]bool{}
	for _, n := range names {
		deny[n] = true
	}
	return func(name string) error {
		if deny[name] {
			return &ToolAccessError{Tool: name, Code: "denied", Message: fmt.Sprintf("tool %q refused", name)}
		}
		return nil
	}
}

func registerProbe(t *testing.T, name string, ran *atomic.Int32) {
	t.Helper()
	require.NoError(t, RegisterTool(&ToolDefinition{
		Name:       name,
		Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
			ran.Add(1)
			return map[string]interface{}{"ran": true}, nil
		},
	}))
}

func TestInvokeTool_RefusalPrecedesExecution(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()
	var ran atomic.Int32
	registerProbe(t, "probe", &ran)

	ctx := WithAuthorizer(context.Background(), denyNamed("probe"))

	for name, call := range map[string]func() (map[string]interface{}, error){
		"InvokeTool": func() (map[string]interface{}, error) {
			return InvokeTool(ctx, "probe", map[string]interface{}{})
		},
		"InvokeToolWithTimeout": func() (map[string]interface{}, error) {
			return InvokeToolWithTimeout(ctx, "probe", map[string]interface{}{}, time.Second)
		},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := call()
			require.Error(t, err)
			assert.Nil(t, res)
			denied, ok := AsToolAccessError(err)
			require.True(t, ok, "the refusal must stay a *ToolAccessError so handlers can answer 403")
			assert.Equal(t, "probe", denied.Tool)
			assert.Zero(t, ran.Load(), "the tool function ran although the call was refused")
		})
	}
}

func TestInvokeTool_AllowedCallsRunAndNoAuthorizerIsUnrestricted(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()
	var ran atomic.Int32
	registerProbe(t, "probe", &ran)

	_, err := InvokeTool(WithAuthorizer(context.Background(), denyNamed("something_else")), "probe", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), ran.Load())

	_, err = InvokeTool(context.Background(), "probe", map[string]interface{}{})
	require.NoError(t, err, "a context with no authorizer is unrestricted")
	assert.Equal(t, int32(2), ran.Load())
}

// An unregistered name must stay "not found" rather than become a policy
// refusal, or a refusal would tell a caller which names exist.
func TestInvokeTool_UnknownToolIsStillNotFound(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()

	ctx := WithAuthorizer(context.Background(), denyNamed("no_such_tool"))
	_, err := InvokeTool(ctx, "no_such_tool", map[string]interface{}{})
	require.Error(t, err)
	_, isAccess := AsToolAccessError(err)
	assert.False(t, isAccess)
	assert.Contains(t, err.Error(), "tool not found")
}

func TestWithAuthorizer_NilCannotLiftARestriction(t *testing.T) {
	ctx := WithAuthorizer(context.Background(), denyNamed("probe"))
	ctx = WithAuthorizer(ctx, nil)
	assert.Error(t, AuthorizeCall(ctx, "probe"))
}

// A later authorizer replaces an earlier one (the innermost caller binding
// wins), which is what lets an entry point bind exactly one policy.
func TestWithAuthorizer_InnermostWins(t *testing.T) {
	ctx := WithAuthorizer(context.Background(), denyNamed("a"))
	ctx = WithAuthorizer(ctx, denyNamed("b"))
	assert.NoError(t, AuthorizeCall(ctx, "a"))
	assert.Error(t, AuthorizeCall(ctx, "b"))
}

func TestInheritAuthorizer(t *testing.T) {
	src := WithAuthorizer(context.Background(), denyNamed("probe"))
	dst := InheritAuthorizer(context.Background(), src)
	assert.Error(t, AuthorizeCall(dst, "probe"), "the detached context must carry the policy")

	// Inheriting from an unbound context binds nothing, and never clears an
	// authorizer already on dst.
	assert.NoError(t, AuthorizeCall(InheritAuthorizer(context.Background(), context.Background()), "probe"))
	bound := WithAuthorizer(context.Background(), denyNamed("x"))
	assert.Error(t, AuthorizeCall(InheritAuthorizer(bound, context.Background()), "x"))
}

func TestFilterAuthorized(t *testing.T) {
	defs := []*ToolDefinition{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	assert.Equal(t, defs, FilterAuthorized(context.Background(), defs), "no authorizer: everything is offered")

	got := FilterAuthorized(WithAuthorizer(context.Background(), denyNamed("b")), defs)
	var names []string
	for _, d := range got {
		names = append(names, d.Name)
	}
	assert.Equal(t, []string{"a", "c"}, names)
}

func TestAsToolAccessError_UnwrapsWrappedErrors(t *testing.T) {
	inner := &ToolAccessError{Tool: "t", Code: "c", Message: "m"}
	wrapped := fmt.Errorf("adapter: %w", inner)
	got, ok := AsToolAccessError(wrapped)
	require.True(t, ok)
	assert.Equal(t, "c", got.Code)

	_, ok = AsToolAccessError(errors.New("plain"))
	assert.False(t, ok)
}

// ─── Path-segment validation on tools that join a caller-supplied id ────────

func TestIsPlainFileID(t *testing.T) {
	good := []string{"abc", "0b1f6c1e-7a8e-4d6a-9d1b-2f2f2f2f2f2f", "plan_1", "doc.v2", "a..b"}
	bad := []string{"", ".", "..", "../x", "a/b", `a\b`, "/abs", "x\x00y", "../../etc/passwd"}
	for _, id := range good {
		assert.True(t, isPlainFileID(id), "%q should be accepted", id)
	}
	for _, id := range bad {
		assert.False(t, isPlainFileID(id), "%q should be rejected", id)
	}
}

// A plan_id that walks out of the plans directory must neither read nor write
// outside it. The sibling file is a valid plan document, so without the
// validation update_plan would load it and write it back modified.
func TestPlanTools_RejectPathTraversalInPlanID(t *testing.T) {
	plansDir := setupTestPlansDir(t)

	outside := filepath.Join(filepath.Dir(plansDir), "outside.json")
	original := []byte(`{"id":"outside","title":"not yours","status":"pending","goals":[],"milestones":[],"progress":0}`)
	require.NoError(t, os.WriteFile(outside, original, 0o600))

	traversal := "../outside" // <plansDir>/../outside.json is the sibling above
	for name, fn := range map[string]func(context.Context, map[string]interface{}) (map[string]interface{}, error){
		"update_plan":      UpdatePlan,
		"track_progress":   TrackProgress,
		"validate_plan":    ValidatePlan,
		"create_milestone": CreateMilestone,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := fn(context.Background(), map[string]interface{}{
				"plan_id": traversal,
				"title":   "m",
				"status":  "completed",
			})
			// The tools report failures either as an error or as success=false.
			failed := err != nil
			if res != nil {
				if ok, has := res["success"].(bool); has && !ok {
					failed = true
				}
			}
			assert.True(t, failed, "a traversing plan_id must be rejected; got res=%v err=%v", res, err)
			if res != nil {
				assert.NotContains(t, fmt.Sprint(res), "not yours", "content from outside the plans directory was returned")
			}
		})
	}

	after, err := os.ReadFile(outside) //nolint:gosec // test reads a path it created
	require.NoError(t, err)
	assert.Equal(t, original, after, "a file outside the plans directory was modified")
}

func TestGetDocument_RejectsPathTraversalInDocumentID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	docs := filepath.Join(home, ".zen-sci", "documents")
	require.NoError(t, os.MkdirAll(docs, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(docs, "ok.json"), []byte(`{"title":"fine"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, "secret.json"), []byte(`{"title":"outside the documents dir"}`), 0o600))

	res, err := getDocument(context.Background(), map[string]interface{}{"document_id": "ok"})
	require.NoError(t, err, "a plain id must still work")
	assert.Equal(t, "fine", res["title"])

	_, err = getDocument(context.Background(), map[string]interface{}{"document_id": "../../secret"})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "outside the documents dir")
}
