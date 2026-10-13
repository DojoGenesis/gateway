package server

// Tests for the tool-invoke gate (tool_invoke_gate.go).
//
// The defect was that any authenticated principal, however low its role, could
// run a shell command, execute code, and read or write arbitrary files by
// POSTing to a tool-invoke route. Registration is open by default, so
// "authenticated" amounted to "has an email address".
//
// Two properties shape how these tests are written:
//
//  1. They drive the REAL setupMiddleware() + setupRoutes() wiring through
//     newRoutedTestServer, never a handler on a hand-built gin.New(). The
//     property under test is "which middleware chain does this route actually
//     get", and the policy is bound to the request context by a middleware, so a
//     handler-only test cannot observe an ordering or wiring mistake at all.
//
//  2. Where the harm is a side effect, they assert on the side effect. A refused
//     run_command must leave no marker file behind and a refused fetch_url must
//     not reach the network: a 403 alone would still pass if the command ran and
//     the handler then merely reported a refusal.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/apps"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/provider"
	// The built-in tools in server/apptools register from init() and are
	// otherwise linked only into the main binary; TestEveryRegisteredToolIsClassified
	// must see the same registry production has.
	_ "github.com/DojoGenesis/gateway/server/apptools"
	"github.com/DojoGenesis/gateway/server/middleware"
	serverorchestration "github.com/DojoGenesis/gateway/server/orchestration"
	"github.com/DojoGenesis/gateway/tools"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// roleToken mints a real, signature-valid session token for role. An empty role
// yields "" (no Authorization header at all).
func roleToken(t *testing.T, role string) string {
	t.Helper()
	if role == "" {
		return ""
	}
	now := time.Now()
	claims := middleware.GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "gate-test-" + role,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Role: role,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(middleware.GetJWTSecret())
	require.NoError(t, err)
	return signed
}

// setGateEnv pins the two inputs the policy reads. E2B_API_KEY is cleared so an
// execute_python call that passes the gate fails locally instead of reaching a
// remote sandbox from a developer's machine.
func setGateEnv(t *testing.T, environment, runCommandOptIn string) {
	t.Helper()
	t.Setenv("ENVIRONMENT", environment)
	t.Setenv(EnvWorkflowRunCommand, runCommandOptIn)
	t.Setenv("E2B_API_KEY", "")
}

func gatePost(t *testing.T, s *Server, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// invokeRoute describes one of the two routes that invoke a tool by name. They
// live in different handler packages with different request and error shapes,
// which is exactly why both are exercised by every case.
type invokeRoute struct {
	name string
	// call performs the invocation through the real router.
	call func(t *testing.T, s *Server, token, tool string, params map[string]interface{}) *httptest.ResponseRecorder
	// refusalCode extracts the machine-readable code from a refusal body.
	refusalCode func(t *testing.T, body []byte) string
}

var invokeRoutes = []invokeRoute{
	{
		name: "POST /v1/tools/:name/invoke",
		call: func(t *testing.T, s *Server, token, tool string, params map[string]interface{}) *httptest.ResponseRecorder {
			return gatePost(t, s, "/v1/tools/"+tool+"/invoke", token, map[string]interface{}{"inputs": params})
		},
		refusalCode: func(t *testing.T, body []byte) string {
			var env struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(body, &env), "refusal body: %s", body)
			return env.Error.Code
		},
	},
	{
		name: "POST /v1/tools/invoke",
		call: func(t *testing.T, s *Server, token, tool string, params map[string]interface{}) *httptest.ResponseRecorder {
			return gatePost(t, s, "/v1/tools/invoke", token, map[string]interface{}{"tool_name": tool, "params": params})
		},
		refusalCode: func(t *testing.T, body []byte) string {
			var env struct {
				Code string `json:"code"`
			}
			require.NoError(t, json.Unmarshal(body, &env), "refusal body: %s", body)
			return env.Code
		},
	},
}

// ─── The matrix the fix is accountable for ───────────────────────────────────

// TestToolInvokeGate_RunCommand drives run_command through every combination of
// role, environment and opt-in on BOTH routes. "ran" is the marker file the
// shell would create.
func TestToolInvokeGate_RunCommand(t *testing.T) {
	cases := []struct {
		name       string
		role       string
		env        string
		optIn      string
		wantStatus int
		wantCode   string
		wantRan    bool
	}{
		{"anonymous", "", "development", "", http.StatusUnauthorized, "", false},
		{"anonymous in production", "", "production", "true", http.StatusUnauthorized, "", false},

		// The incident: a user-role caller on a host where code execution is on.
		{"user, development default", "user", "development", "", http.StatusForbidden, ToolErrRequiresAdmin, false},
		{"user, development opt-in", "user", "development", "true", http.StatusForbidden, ToolErrRequiresAdmin, false},
		{"user, production opt-in", "user", "production", "true", http.StatusForbidden, ToolErrRequiresAdmin, false},
		{"user, production default", "user", "production", "", http.StatusForbidden, ToolErrRequiresAdmin, false},

		// Admin alone is not enough in production.
		{"admin, production default", "admin", "production", "", http.StatusForbidden, ToolErrCodeExecDisabled, false},
		{"admin, production unparseable opt-in", "admin", "production", "yes-please", http.StatusForbidden, ToolErrCodeExecDisabled, false},
		{"admin, production explicit false", "admin", "production", "false", http.StatusForbidden, ToolErrCodeExecDisabled, false},
		{"admin, development explicit false", "admin", "development", "false", http.StatusForbidden, ToolErrCodeExecDisabled, false},

		// Admin plus the shared opt-in is the only way through.
		{"admin, production opt-in", "admin", "production", "true", http.StatusOK, "", true},
		{"admin, development opt-in", "admin", "development", "true", http.StatusOK, "", true},
		{"admin, development default", "admin", "development", "", http.StatusOK, "", true},
	}

	for _, route := range invokeRoutes {
		for _, tc := range cases {
			t.Run(route.name+"/"+tc.name, func(t *testing.T) {
				setGateEnv(t, tc.env, tc.optIn)
				s := newRoutedTestServer(t)

				marker := filepath.Join(t.TempDir(), "ran")
				const canary = "CANARY-7f3a91"
				// The canary rides in the command so a refusal that echoed its
				// input would show up in the body.
				w := route.call(t, s, roleToken(t, tc.role), "run_command", map[string]interface{}{
					"command": "touch " + marker + " # " + canary,
				})

				require.Equal(t, tc.wantStatus, w.Code, "body=%s", w.Body.String())
				if tc.wantStatus == http.StatusForbidden {
					assert.Equal(t, tc.wantCode, route.refusalCode(t, w.Body.Bytes()))
					assert.NotContains(t, w.Body.String(), canary, "a refusal must not echo the caller's inputs")
					assert.NotContains(t, w.Body.String(), marker, "a refusal must not echo the caller's inputs")
				}

				_, statErr := os.Stat(marker)
				if tc.wantRan {
					assert.NoError(t, statErr, "an allowed call must really run the command")
				} else {
					assert.True(t, os.IsNotExist(statErr), "the command ran although the call was refused (status %d)", w.Code)
				}
			})
		}
	}
}

// TestToolInvokeGate_SafeToolStaysOpenToUsers is the other half of the bargain:
// the fix must not turn the endpoint into an admin-only wall.
func TestToolInvokeGate_SafeToolStaysOpenToUsers(t *testing.T) {
	for _, route := range invokeRoutes {
		for _, role := range []string{"user", "admin"} {
			for _, env := range []string{"development", "production"} {
				t.Run(route.name+"/"+role+"/"+env, func(t *testing.T) {
					setGateEnv(t, env, "")
					s := newRoutedTestServer(t)

					w := route.call(t, s, roleToken(t, role), "calculate", map[string]interface{}{"expression": "6 * 7"})

					require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
					assert.Contains(t, w.Body.String(), "42", "the tool must actually have run")
				})
			}
		}
	}
}

// TestToolInvokeGate_UnauthenticatedIs401 pins that authentication still comes
// first and is unchanged on both routes, for a safe and a dangerous tool alike.
func TestToolInvokeGate_UnauthenticatedIs401(t *testing.T) {
	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	for _, route := range invokeRoutes {
		for _, tool := range []string{"calculate", "run_command", "no_such_tool"} {
			t.Run(route.name+"/"+tool, func(t *testing.T) {
				w := route.call(t, s, "", tool, map[string]interface{}{})
				assert.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
			})
		}
	}
}

// TestToolInvokeGate_UnknownToolIsNotAPolicyRefusal pins that the gate does not
// change how an unregistered name is reported. Both routes answer such a call
// with their existing execution-failure response (500), which other tests pin;
// the gate must not turn it into a 403 for a user, because that would let a
// caller probe which names are registered by the difference.
func TestToolInvokeGate_UnknownToolIsNotAPolicyRefusal(t *testing.T) {
	setGateEnv(t, "development", "")
	s := newRoutedTestServer(t)

	for _, route := range invokeRoutes {
		for _, role := range []string{"user", "admin"} {
			t.Run(route.name+"/"+role, func(t *testing.T) {
				w := route.call(t, s, roleToken(t, role), "no_such_tool_xyz", map[string]interface{}{})
				assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
			})
		}
	}
}

// TestToolInvokeGate_EveryDangerousToolRefusedForUsers walks the whole table.
// Parameters are empty on purpose: the refusal must precede parameter
// validation and execution, so it does not depend on the call being well-formed.
func TestToolInvokeGate_EveryDangerousToolRefusedForUsers(t *testing.T) {
	for name, p := range toolPolicies {
		for _, route := range invokeRoutes {
			t.Run(route.name+"/"+name, func(t *testing.T) {
				// A server per case: the global per-IP rate limiter would
				// otherwise start answering 429 partway through the table.
				setGateEnv(t, "development", "true")
				s := newRoutedTestServer(t)

				user := route.call(t, s, roleToken(t, "user"), name, map[string]interface{}{})
				admin := route.call(t, s, roleToken(t, "admin"), name, map[string]interface{}{})

				if p.adminOnly {
					require.Equal(t, http.StatusForbidden, user.Code, "user must be refused %q; body=%s", name, user.Body.String())
					assert.Equal(t, ToolErrRequiresAdmin, route.refusalCode(t, user.Body.Bytes()))
				} else {
					assert.NotEqual(t, http.StatusForbidden, user.Code, "user must not be refused %q; body=%s", name, user.Body.String())
				}
				// Admin in development with execution enabled is never refused by
				// policy; an empty call may still fail validation, which is not a 403.
				assert.NotEqual(t, http.StatusForbidden, admin.Code, "admin must not be refused %q; body=%s", name, admin.Body.String())
			})
		}
	}
}

// TestToolInvokeGate_SearchToolsNeverLeakTheOperatorKey drives the tools that
// spend the operator's search credential as a user-role caller, on both routes,
// with a query that used to make the request builder fail with an error text
// containing the whole request URL, key included. The caller must get a refusal
// and the key must appear nowhere in the body. A stub transport guarantees that
// nothing leaves the machine even if the gate were wrong.
func TestToolInvokeGate_SearchToolsNeverLeakTheOperatorKey(t *testing.T) {
	const canary = "canary-search-key-2b8d4e"
	searchTools := map[string]string{
		"web_search":     "query",
		"gather_sources": "topic",
		"fact_check":     "claim",
	}
	for tool, field := range searchTools {
		for _, route := range invokeRoutes {
			t.Run(route.name+"/"+tool, func(t *testing.T) {
				setGateEnv(t, "development", "")
				t.Setenv("SERPAPI_KEY", canary)

				var sent atomic.Int32
				orig := http.DefaultTransport
				http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					sent.Add(1)
					return nil, fmt.Errorf("no network in tests: %s", r.URL.String())
				})
				t.Cleanup(func() { http.DefaultTransport = orig })

				s := newRoutedTestServer(t)
				w := route.call(t, s, roleToken(t, "user"), tool, map[string]interface{}{field: "x\n"})

				require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
				assert.Equal(t, ToolErrRequiresAdmin, route.refusalCode(t, w.Body.Bytes()))
				assert.NotContains(t, w.Body.String(), canary)
				assert.Zero(t, sent.Load(), "a refused call must not issue a search request")
			})
		}
	}
}

// TestToolInvokeGate_AdminSearchErrorsNeverCarryTheKey covers the second line of
// defence: even a caller allowed to run the tool must not be able to read the
// provider key out of an error, whatever the transport reports.
func TestToolInvokeGate_AdminSearchErrorsNeverCarryTheKey(t *testing.T) {
	const canary = "canary-search-key-9c1f06"
	for _, route := range invokeRoutes {
		t.Run(route.name, func(t *testing.T) {
			setGateEnv(t, "development", "")
			t.Setenv("SERPAPI_KEY", canary)

			orig := http.DefaultTransport
			http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("transport down")
			})
			t.Cleanup(func() { http.DefaultTransport = orig })

			s := newRoutedTestServer(t)
			w := route.call(t, s, roleToken(t, "admin"), "web_search", map[string]interface{}{"query": "x\n"})

			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
			assert.NotContains(t, w.Body.String(), canary)
			assert.Contains(t, w.Body.String(), "search request failed")
		})
	}
}

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestToolInvokeGate_RefusedFetchNeverReachesTheNetwork asserts on the request
// that would have left the host, not on the status code.
func TestToolInvokeGate_RefusedFetchNeverReachesTheNetwork(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = fmt.Fprint(w, "internal-only")
	}))
	defer target.Close()

	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	for _, route := range invokeRoutes {
		for _, tool := range []string{"fetch_url", "api_call", "web_navigate", "web_extract_links"} {
			t.Run(route.name+"/"+tool, func(t *testing.T) {
				w := route.call(t, s, roleToken(t, "user"), tool, map[string]interface{}{"url": target.URL})
				require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
				assert.NotContains(t, w.Body.String(), "internal-only")
			})
		}
	}
	assert.Zero(t, hits.Load(), "a refused call still made a server-side request")
}

// TestToolInvokeGate_RefusedFileToolsTouchNothing covers the filesystem tools:
// a user must neither read a secret nor create a file.
func TestToolInvokeGate_RefusedFileToolsTouchNothing(t *testing.T) {
	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	dir := t.TempDir()
	secret := filepath.Join(dir, "env")
	require.NoError(t, os.WriteFile(secret, []byte("SECRET-VALUE-91c2"), 0o600))
	created := filepath.Join(dir, "created")

	for _, route := range invokeRoutes {
		t.Run(route.name+"/read_file", func(t *testing.T) {
			w := route.call(t, s, roleToken(t, "user"), "read_file", map[string]interface{}{"file_path": secret})
			require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
			assert.NotContains(t, w.Body.String(), "SECRET-VALUE-91c2")
		})
		t.Run(route.name+"/write_file", func(t *testing.T) {
			w := route.call(t, s, roleToken(t, "user"), "write_file", map[string]interface{}{"file_path": created, "content": "x"})
			require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
			_, err := os.Stat(created)
			assert.True(t, os.IsNotExist(err), "write_file created a file for a user-role caller")
		})
	}
}

// TestToolInvokeGate_UnclassifiedToolDefaultsToAdmin pins default-deny with a
// tool registered the way an MCP host or plugin registers one: under a
// namespaced name this table has never heard of.
func TestToolInvokeGate_UnclassifiedToolDefaultsToAdmin(t *testing.T) {
	setGateEnv(t, "development", "")
	var ran atomic.Int32
	const name = "gatetest:probe"
	require.NoError(t, tools.RegisterTool(&tools.ToolDefinition{
		Name:       name,
		Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
			ran.Add(1)
			return map[string]interface{}{"ok": true}, nil
		},
	}))
	t.Cleanup(func() { _ = tools.UnregisterTool(name) })

	s := newRoutedTestServer(t)

	// The :name route takes the segment verbatim, colon included.
	w := gatePost(t, s, "/v1/tools/"+name+"/invoke", roleToken(t, "user"), map[string]interface{}{"inputs": map[string]interface{}{}})
	require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.Zero(t, ran.Load(), "an unclassified tool ran for a user-role caller")

	w = gatePost(t, s, "/v1/tools/invoke", roleToken(t, "user"), map[string]interface{}{"tool_name": name, "params": map[string]interface{}{}})
	require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.Zero(t, ran.Load())

	w = gatePost(t, s, "/v1/tools/"+name+"/invoke", roleToken(t, "admin"), map[string]interface{}{"inputs": map[string]interface{}{}})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, int32(1), ran.Load())
}

// TestServiceTokenIsNeverAnAdminForTools: a machine credential must not unlock
// admin-only tools even if it somehow carries role=admin.
func TestServiceTokenIsNeverAnAdminForTools(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)
	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	token, _, err := middleware.IssueServiceToken("gate-test", 0)
	require.NoError(t, err)

	marker := filepath.Join(t.TempDir(), "ran")
	for _, route := range invokeRoutes {
		t.Run(route.name, func(t *testing.T) {
			w := route.call(t, s, token, "run_command", map[string]interface{}{"command": "touch " + marker})
			require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
			_, statErr := os.Stat(marker)
			assert.True(t, os.IsNotExist(statErr))
		})
	}
}

// ─── Classification invariants ───────────────────────────────────────────────

// TestEveryRegisteredToolIsClassified is the drift guard. Default-deny keeps an
// unlisted tool safe, but "safe because nobody looked" should never be the
// steady state: adding a built-in tool must force someone to say whether a
// user-role caller may run it.
func TestEveryRegisteredToolIsClassified(t *testing.T) {
	registered := map[string]bool{}
	for _, d := range tools.GetAllTools() {
		registered[d.Name] = true
		_, listed := toolPolicies[d.Name]
		assert.True(t, listed, "tool %q is registered but absent from toolPolicies (it would default to admin-only); classify it", d.Name)
	}
	for name := range toolPolicies {
		assert.True(t, registered[name], "toolPolicies names %q, which no longer registers: remove the stale entry", name)
	}
}

// TestToolPolicyShape pins the facts a reviewer should not have to rediscover.
func TestToolPolicyShape(t *testing.T) {
	mustBeAdmin := []string{
		"run_command", "execute_python",
		"read_file", "write_file", "list_directory", "search_files",
		"fetch_url", "api_call",
		"web_navigate", "web_scrape_structured", "web_screenshot", "web_monitor", "web_extract_links",
		"web_search", "gather_sources", "fact_check",
		"export_diagram", "export_chart",
	}
	for _, name := range mustBeAdmin {
		assert.True(t, policyForTool(name).adminOnly, "%s must be admin-only", name)
	}

	for name, p := range toolPolicies {
		assert.NotEmpty(t, p.reason, "%s has no recorded reason", name)
		if p.codeExec {
			assert.True(t, p.adminOnly, "%s executes code and must be admin-only", name)
		}
	}
	assert.True(t, policyForTool("run_command").codeExec)
	assert.True(t, policyForTool("execute_python").codeExec)
	assert.False(t, policyForTool("read_file").codeExec, "only code-executing tools take the production opt-in")

	assert.True(t, policyForTool("some_server:some_tool").adminOnly, "unknown tools default to admin-only")
	assert.True(t, policyForTool("").adminOnly)
}

// TestAuthorizeToolCall_Matrix is the unit-level statement of the rule.
func TestAuthorizeToolCall_Matrix(t *testing.T) {
	cases := []struct {
		tool, env, optIn string
		admin            bool
		wantCode         string // "" = allowed
	}{
		{"calculate", "production", "", false, ""},
		{"calculate", "production", "", true, ""},
		{"read_file", "production", "", false, ToolErrRequiresAdmin},
		{"read_file", "production", "", true, ""}, // admin-only, not a code-exec tool: the opt-in does not apply
		{"fetch_url", "production", "", true, ""},
		{"run_command", "production", "", false, ToolErrRequiresAdmin}, // admin check wins, so a user learns nothing about config
		{"run_command", "production", "true", false, ToolErrRequiresAdmin},
		{"run_command", "production", "", true, ToolErrCodeExecDisabled},
		{"run_command", "production", "true", true, ""},
		{"execute_python", "production", "", true, ToolErrCodeExecDisabled},
		{"execute_python", "production", "true", true, ""},
		{"execute_python", "development", "", true, ""},
		{"execute_python", "development", "false", true, ToolErrCodeExecDisabled},
		{"unknown:tool", "development", "", false, ToolErrRequiresAdmin},
		{"unknown:tool", "development", "", true, ""},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%s/%q/admin=%v", tc.tool, tc.env, tc.optIn, tc.admin), func(t *testing.T) {
			setGateEnv(t, tc.env, tc.optIn)
			got := authorizeToolCall(tc.tool, tc.admin)
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.Code)
			assert.Equal(t, tc.tool, got.Tool)
		})
	}
}

// ─── Other paths that reach a tool function ──────────────────────────────────

// recordingRegistry is a gateway.ToolRegistry holding a single tool, so a test
// can prove a handler did or did not call its function.
type recordingRegistry struct {
	*testToolRegistry
	ran atomic.Int32
}

func newRecordingRegistry(toolName string) *recordingRegistry {
	r := &recordingRegistry{testToolRegistry: newTestToolRegistry()}
	_ = r.Register(context.Background(), &tools.ToolDefinition{
		Name: toolName,
		Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
			r.ran.Add(1)
			return map[string]interface{}{"ran": true}, nil
		},
	})
	return r
}

// TestAppsToolCallProxy_AppliesCallerPolicy: a launched app holds a wildcard
// tool grant, so the only thing standing between a user-role caller and
// run_command on this route is the caller policy.
func TestAppsToolCallProxy_AppliesCallerPolicy(t *testing.T) {
	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	reg := newRecordingRegistry("run_command")
	mgr := apps.NewAppManager(apps.AppManagerConfig{DefaultToolTimeout: 5 * time.Second}, reg)
	require.NoError(t, mgr.RegisterResource(&apps.ResourceMeta{URI: "ui://gate/test.html", MimeType: "text/html", CacheKey: "k"}))
	inst, err := mgr.LaunchApp("ui://gate/test.html", "sess")
	require.NoError(t, err)
	s.appManager = mgr

	body := map[string]interface{}{"app_id": inst.ID, "tool_name": "run_command", "arguments": map[string]interface{}{}}

	w := gatePost(t, s, "/v1/gateway/apps/tool-call", roleToken(t, "user"), body)
	require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	assert.Zero(t, reg.ran.Load(), "the app proxy ran run_command for a user-role caller")

	w = gatePost(t, s, "/v1/gateway/apps/tool-call", roleToken(t, "admin"), body)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, int32(1), reg.ran.Load())

	// A safe tool is unaffected for a user.
	safe := newRecordingRegistry("calculate")
	mgr2 := apps.NewAppManager(apps.AppManagerConfig{DefaultToolTimeout: 5 * time.Second}, safe)
	require.NoError(t, mgr2.RegisterResource(&apps.ResourceMeta{URI: "ui://gate/test.html", MimeType: "text/html", CacheKey: "k"}))
	inst2, err := mgr2.LaunchApp("ui://gate/test.html", "sess")
	require.NoError(t, err)
	s.appManager = mgr2
	w = gatePost(t, s, "/v1/gateway/apps/tool-call", roleToken(t, "user"),
		map[string]interface{}{"app_id": inst2.ID, "tool_name": "calculate", "arguments": map[string]interface{}{}})
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, int32(1), safe.ran.Load())
}

// TestGatewayOrchestrate_PlanNodesRunWithTheCallersAuthority is the strongest
// of the other paths: the plan, and so every node's tool_name, is supplied by
// the caller directly, and the engine runs it on a context detached from the
// request.
func TestGatewayOrchestrate_PlanNodesRunWithTheCallersAuthority(t *testing.T) {
	cases := []struct {
		name    string
		role    string
		wantRan bool
		want    string
	}{
		{"user is refused", "user", false, "failed"},
		{"admin runs it", "admin", true, "complete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setGateEnv(t, "development", "true")
			s := newRoutedTestServer(t)

			cfg := orchestrationpkg.DefaultEngineConfig()
			cfg.MaxRetries = 0
			cfg.EnableAutoReplanning = false
			s.orchestrationEngine = orchestrationpkg.NewEngine(cfg, nil, serverorchestration.NewToolInvokerAdapter(), nil, nil, nil)

			marker := filepath.Join(t.TempDir(), "ran")
			plan := map[string]interface{}{
				"id":   "gate-plan",
				"name": "gate plan",
				"dag": []map[string]interface{}{{
					"id":         "n1",
					"tool_name":  "run_command",
					"input":      map[string]interface{}{"command": "touch " + marker},
					"depends_on": []string{},
				}},
			}
			w := gatePost(t, s, "/v1/gateway/orchestrate", roleToken(t, tc.role), map[string]interface{}{"plan": plan, "user_id": "u"})
			require.Equal(t, http.StatusAccepted, w.Code, "body=%s", w.Body.String())

			var resp struct {
				ExecutionID string `json:"execution_id"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

			var status string
			require.Eventually(t, func() bool {
				state, ok := s.orchestrations.Get(resp.ExecutionID)
				if !ok {
					return false
				}
				state.mu.Lock()
				status = state.Status
				state.mu.Unlock()
				return status == "complete" || status == "failed"
			}, 5*time.Second, 10*time.Millisecond)

			assert.Equal(t, tc.want, status)
			_, statErr := os.Stat(marker)
			if tc.wantRan {
				assert.NoError(t, statErr)
			} else {
				assert.True(t, os.IsNotExist(statErr), "a plan node ran run_command for a user-role caller")
			}
		})
	}
}

// TestGatewayAgentToolCall_AppliesCallerPolicy covers the agent loop behind
// POST /v1/gateway/agents/:id/chat, which invokes tool functions directly.
func TestGatewayAgentToolCall_AppliesCallerPolicy(t *testing.T) {
	setGateEnv(t, "development", "true")
	reg := newRecordingRegistry("run_command")
	s := &Server{toolRegistry: reg}

	call := provider.ToolCall{ID: "c1", Name: "run_command", Arguments: map[string]interface{}{}}

	userCtx := tools.WithAuthorizer(context.Background(), newToolAuthorizer(false))
	out := s.executeToolCall(userCtx, call)
	assert.Contains(t, out, "requires administrator privileges")
	assert.Zero(t, reg.ran.Load(), "the agent loop ran run_command for a user-role caller")

	adminCtx := tools.WithAuthorizer(context.Background(), newToolAuthorizer(true))
	out = s.executeToolCall(adminCtx, call)
	assert.Contains(t, out, "ran")
	assert.Equal(t, int32(1), reg.ran.Load())

	// And the model is not even offered what the caller cannot run.
	full := newRecordingRegistry("run_command")
	_ = full.Register(context.Background(), &tools.ToolDefinition{Name: "calculate", Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) { return nil, nil }})
	s2 := &Server{toolRegistry: full}
	var offered []string
	for _, tl := range s2.buildToolList(userCtx) {
		offered = append(offered, tl.Name)
	}
	assert.Equal(t, []string{"calculate"}, offered)
}

// TestWorkflowExecute_IsAdminOnly: executing a stored workflow is how a step
// reaches run_command, so it takes the same role the direct routes take.
func TestWorkflowExecute_IsAdminOnly(t *testing.T) {
	setGateEnv(t, "development", "true")
	s := newRoutedTestServer(t)

	w := gatePost(t, s, "/api/workflows/demo/execute", roleToken(t, "user"), map[string]interface{}{})
	assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())

	// An admin passes the role gate; newRoutedTestServer has no CAS, so the
	// handler itself answers 501, which is how we know it was reached.
	w = gatePost(t, s, "/api/workflows/demo/execute", roleToken(t, "admin"), map[string]interface{}{})
	assert.Equal(t, http.StatusNotImplemented, w.Code, "body=%s", w.Body.String())
}

// stubPlanner returns a one-node plan that runs a shell command, and records
// whether the context it was handed carried the caller's tool policy.
type stubPlanner struct {
	marker      string
	policySeen  atomic.Bool
	policyKnown atomic.Bool
}

func (p *stubPlanner) GeneratePlan(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
	// A context with no policy bound would let every call through, so a refusal
	// here proves the policy crossed the detach from the request.
	p.policySeen.Store(tools.AuthorizeCall(ctx, "run_command") != nil)
	p.policyKnown.Store(true)
	return &orchestrationpkg.Plan{
		ID:     "stub-plan",
		TaskID: task.ID,
		Nodes: []*orchestrationpkg.PlanNode{{
			ID:           "n1",
			ToolName:     "run_command",
			Parameters:   map[string]interface{}{"command": "touch " + p.marker},
			Dependencies: []string{},
			State:        orchestrationpkg.NodeStatePending,
		}},
		CreatedAt: time.Now(),
		Version:   1,
	}, nil
}

func (p *stubPlanner) RegeneratePlan(context.Context, *orchestrationpkg.Task, *orchestrationpkg.Plan, string) (*orchestrationpkg.Plan, error) {
	return nil, fmt.Errorf("not expected")
}

func newPlannerServer(t *testing.T) (*Server, *stubPlanner) {
	t.Helper()
	s := newRoutedTestServer(t)
	cfg := orchestrationpkg.DefaultEngineConfig()
	cfg.MaxRetries = 0
	cfg.EnableAutoReplanning = false
	planner := &stubPlanner{marker: filepath.Join(t.TempDir(), "ran")}
	s.planner = planner
	s.orchestrationEngine = orchestrationpkg.NewEngine(cfg, planner, serverorchestration.NewToolInvokerAdapter(), nil, nil, nil)
	return s, planner
}

func waitForOrchestration(t *testing.T, s *Server, id string) string {
	t.Helper()
	var status string
	require.Eventually(t, func() bool {
		state, ok := s.orchestrations.Get(id)
		if !ok {
			return false
		}
		state.mu.Lock()
		status = state.Status
		state.mu.Unlock()
		return status == "complete" || status == "failed"
	}, 5*time.Second, 10*time.Millisecond)
	return status
}

// TestOrchestrate_PlannerAndEngineRunWithTheCallersAuthority covers
// POST /v1/orchestrate, where an LLM writes the plan from the caller's prompt.
func TestOrchestrate_PlannerAndEngineRunWithTheCallersAuthority(t *testing.T) {
	for _, tc := range []struct {
		role    string
		wantRan bool
	}{{"user", false}, {"admin", true}} {
		t.Run(tc.role, func(t *testing.T) {
			setGateEnv(t, "development", "true")
			s, planner := newPlannerServer(t)

			w := gatePost(t, s, "/v1/orchestrate", roleToken(t, tc.role), map[string]interface{}{"task_description": "do a thing"})
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
			var resp OrchestrateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

			status := waitForOrchestration(t, s, resp.OrchestrationID)

			require.True(t, planner.policyKnown.Load())
			assert.Equal(t, !tc.wantRan, planner.policySeen.Load(), "the planner's context must carry the caller's policy")
			_, statErr := os.Stat(planner.marker)
			if tc.wantRan {
				assert.NoError(t, statErr)
				assert.Equal(t, "complete", status)
			} else {
				assert.True(t, os.IsNotExist(statErr), "an orchestration ran run_command for a user-role caller")
				assert.Equal(t, "failed", status)
			}
		})
	}
}

// TestStartOrchestrationForChat_KeepsTheCallersAuthority covers the seam the
// chat handler uses: it has no HTTP handler of its own for the orchestration,
// so the caller's request context is passed in explicitly.
func TestStartOrchestrationForChat_KeepsTheCallersAuthority(t *testing.T) {
	for _, tc := range []struct {
		name    string
		admin   bool
		wantRan bool
	}{{"user", false, false}, {"admin", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			setGateEnv(t, "development", "true")
			s, planner := newPlannerServer(t)

			reqCtx := tools.WithAuthorizer(context.Background(), newToolAuthorizer(tc.admin))
			id, err := s.StartOrchestrationForChat(reqCtx, "u", "do a thing", time.Minute)
			require.NoError(t, err)

			status := waitForOrchestration(t, s, id)

			_, statErr := os.Stat(planner.marker)
			if tc.wantRan {
				assert.NoError(t, statErr)
				assert.Equal(t, "complete", status)
			} else {
				assert.True(t, os.IsNotExist(statErr), "a chat-started orchestration ran run_command for a user-role caller")
				assert.Equal(t, "failed", status)
			}
		})
	}
}
