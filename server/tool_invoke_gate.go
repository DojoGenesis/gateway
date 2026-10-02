package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/DojoGenesis/gateway/tools"
)

// This file decides ONE thing: may THIS caller invoke THIS tool?
//
// WHY IT EXISTS:
// the tool registry holds run_command, execute_python, write_file, read_file,
// fetch_url, api_call and friends. POST /v1/tools/:name/invoke and
// POST /v1/tools/invoke sat behind AuthMiddleware only, which accepts ANY
// authenticated role, and handed the request straight to tools.InvokeTool. And
// registration (POST /auth/register) is open by default, so "authenticated"
// meant "typed an email address": one extra HTTP request turned the public
// internet into a shell, a file reader/writer and a server-side request
// forger running as the gateway process, with JWT_SECRET one `cat` away.
//
// The same registry is reachable by an agent loop on behalf of a chat user,
// by an orchestration plan a caller submits, and by an app's tool-call proxy.
// Gating two routes would have left every one of those open, so the policy is
// bound to the request context (tools.WithAuthorizer) by toolAccessMiddleware
// and enforced where tools actually run (tools.InvokeToolWithTimeout), plus at
// the few call sites that invoke a tool function directly.
//
// THE RULE:
//   - A tool on the user list below may be called by any authenticated caller.
//   - Everything else requires the admin role. That includes every tool this
//     table does not name: an MCP-hosted tool ("server:tool"), a plugin, or a
//     new built-in nobody has classified yet. Default-deny is deliberate; the
//     cost of forgetting to list a harmless tool is an admin-only tool, and the
//     cost of forgetting to list a dangerous one is this incident.
//     TestEveryRegisteredToolIsClassified fails the build when a registered
//     tool is not in the table, so "not classified" is never the steady state.
//   - Tools that execute caller-supplied code (run_command, execute_python)
//     additionally require the same explicit opt-in the workflow gate uses
//     (workflowRunCommandEnabled): off in production unless an operator set
//     WORKFLOW_RUN_COMMAND_ENABLED=true. Admin plus opt-in is required; neither
//     is enough alone.
//
// This is a capability and authorization control, not an authentication one,
// and it is independent of both: it holds if registration is closed, and it
// holds if a token is stolen from a user-role account.

// Machine-readable refusal codes returned to the caller.
const (
	// ToolErrRequiresAdmin: the tool needs the admin role and the caller has not
	// got it.
	ToolErrRequiresAdmin = "tool_requires_admin"
	// ToolErrCodeExecDisabled: the caller is an admin, but this gateway has not
	// enabled caller-supplied code execution (see EnvWorkflowRunCommand).
	ToolErrCodeExecDisabled = "code_execution_disabled"
)

// toolPolicy is the access rule for one tool.
type toolPolicy struct {
	// adminOnly requires the admin role.
	adminOnly bool
	// codeExec marks a tool that runs caller-supplied code. It is subject to the
	// production opt-in in addition to adminOnly.
	codeExec bool
	// reason records why the tool is classified as it is. It is for reviewers and
	// is not shown to callers.
	reason string
}

func userTool(reason string) toolPolicy { return toolPolicy{reason: reason} }

func adminTool(reason string) toolPolicy { return toolPolicy{adminOnly: true, reason: reason} }

func codeExecTool(reason string) toolPolicy {
	return toolPolicy{adminOnly: true, codeExec: true, reason: reason}
}

// toolPolicies classifies every tool this gateway registers. A tool absent from
// the map is admin-only; see the rule above.
var toolPolicies = map[string]toolPolicy{
	// ── Host execution ────────────────────────────────────────────────────
	"run_command":    codeExecTool("runs a caller-supplied string through a shell as the gateway process; the built-in denylist is not a boundary"),
	"execute_python": codeExecTool("runs caller-supplied code in a remote sandbox billed to the operator's API key"),

	// ── Host filesystem ───────────────────────────────────────────────────
	"read_file":      adminTool("reads any path the gateway process can read, including its own env file"),
	"write_file":     adminTool("writes any path the gateway process can write"),
	"list_directory": adminTool("enumerates any directory the gateway process can read"),
	"search_files":   adminTool("searches file contents under any path the gateway process can read"),

	// ── Server-side network reach (caller chooses the destination) ───────
	"fetch_url":             adminTool("server-side GET of a caller-chosen URL: loopback, link-local metadata and internal services"),
	"api_call":              adminTool("server-side request with caller-chosen URL, method, headers and body"),
	"web_navigate":          adminTool("headless browser on the host navigating to a caller-chosen URL"),
	"web_scrape_structured": adminTool("fetches and parses a caller-chosen URL"),
	"web_screenshot":        adminTool("headless browser on the host rendering a caller-chosen URL"),
	"web_monitor":           adminTool("repeatedly fetches a caller-chosen URL"),
	"web_extract_links":     adminTool("fetches a caller-chosen URL"),

	// ── Rendering with host side effects ─────────────────────────────────
	"export_diagram": adminTool("interpolates caller text into a page run in a headless browser on the host (script and network reach) and writes the output to the host temp dir"),
	"export_chart":   adminTool("writes caller-derived output to the host temp dir"),

	// ── Pure computation ─────────────────────────────────────────────────
	"calculate":         userTool("expression evaluator with no custom functions and no I/O"),
	"prepare_diagram":   userTool("validates diagram syntax; no I/O"),
	"prepare_chart":     userTool("validates chart data; no I/O"),
	"generate_image":    userTool("placeholder that returns metadata only; no I/O"),
	"synthesize_info":   userTool("text summarisation; no I/O"),
	"analyze_sentiment": userTool("text analysis; no I/O"),
	"extract_entities":  userTool("text analysis; no I/O"),

	// ── Registry metadata (already served by GET /v1/tools) ──────────────
	"search_tools":  userTool("read-only view of the tool registry, same data as GET /v1/tools"),
	"get_tool_info": userTool("read-only view of one registry entry, same data as GET /v1/tools/:name"),

	// ── Planning (a fixed plans directory; plan_id is validated) ─────────
	"create_plan":      userTool("writes a gateway-generated plan file under the fixed plans directory"),
	"update_plan":      userTool("rewrites a plan file; plan_id is validated as a single path segment"),
	"track_progress":   userTool("reads a plan file; plan_id is validated as a single path segment"),
	"validate_plan":    userTool("reads a plan file; plan_id is validated as a single path segment"),
	"create_milestone": userTool("rewrites a plan file; plan_id is validated as a single path segment"),

	// ── Spends the operator's search credential ─────────────────────────
	// The destination is fixed, but the request is built with the operator's
	// provider key, and the search code runs caller text through that request.
	// A credential that a user-role caller can spend, or provoke into an error
	// message, is not pure computation, so these stay admin-only.
	"web_search":     adminTool("issues a search request authenticated with the operator's provider key and spends its quota"),
	"gather_sources": adminTool("calls web_search with a caller-supplied topic; same classification"),
	"fact_check":     adminTool("calls web_search with a caller-supplied claim; same classification"),

	// ── Gateway state: database-backed or a fixed directory ──────────────
	// Known gap, deliberately out of scope here: these are not scoped per user,
	// so one authenticated user can read or change another's records. That is a
	// tenancy defect, not host reach, and wants its own change.
	"get_document":           userTool("reads <home>/.zen-sci/documents/<id>.json; document_id is validated as a single path segment"),
	"create_project":         userTool("database row plus a directory under the fixed projects base; name is validated"),
	"list_projects":          userTool("database read"),
	"get_project":            userTool("database read"),
	"switch_project":         userTool("database read"),
	"list_templates":         userTool("database read"),
	"update_project":         userTool("database write; the project name is immutable"),
	"delete_project":         userTool("database write plus removal of the project directory under the fixed projects base"),
	"create_artifact":        userTool("database write"),
	"update_artifact":        userTool("database write"),
	"get_artifact":           userTool("database read"),
	"list_artifacts":         userTool("database read"),
	"list_artifact_versions": userTool("database read"),
	"export_artifact":        userTool("database read re-encoded in memory"),
	"memory_search":          userTool("database read"),
	"memory_get":             userTool("database read by memory id; no file path is read"),
}

// unclassifiedToolPolicy applies to any tool the table does not name.
var unclassifiedToolPolicy = adminTool("not classified: default-deny")

// policyForTool returns the policy for a tool, defaulting to admin-only.
func policyForTool(name string) toolPolicy {
	if p, ok := toolPolicies[name]; ok {
		return p
	}
	return unclassifiedToolPolicy
}

// authorizeToolCall applies the policy to one (caller, tool) pair. It returns
// nil to allow and a *tools.ToolAccessError to refuse. The error names the tool
// and the reason and carries no call arguments.
//
// The admin check comes first on purpose: a non-admin is told only that the
// tool needs admin, never whether code execution happens to be enabled here.
func authorizeToolCall(name string, admin bool) *tools.ToolAccessError {
	p := policyForTool(name)

	if p.adminOnly && !admin {
		return &tools.ToolAccessError{
			Tool:    name,
			Code:    ToolErrRequiresAdmin,
			Message: fmt.Sprintf("tool %q requires administrator privileges", name),
		}
	}

	if p.codeExec && !workflowRunCommandEnabled() {
		return &tools.ToolAccessError{
			Tool: name,
			Code: ToolErrCodeExecDisabled,
			Message: fmt.Sprintf("tool %q is disabled on this gateway: caller-supplied code execution is not enabled (%s)",
				name, EnvWorkflowRunCommand),
		}
	}

	return nil
}

// newToolAuthorizer binds the policy to one caller. admin is decided once, when
// the request is authenticated, and travels with the context: work that
// outlives the request (an orchestration, a streamed agent turn) keeps the
// authority it started with instead of re-deriving it from a token that may
// have expired mid-run.
//
// It does not log. tools.AuthorizeCall logs the refusals that are actually
// enforced; tools.FilterAuthorized probes every tool and must stay quiet.
func newToolAuthorizer(admin bool) tools.Authorizer {
	return func(name string) error {
		if denied := authorizeToolCall(name, admin); denied != nil {
			return denied
		}
		return nil
	}
}

// toolAccessMiddleware binds the caller's tool policy to the request context.
//
// It MUST run after AuthMiddleware, because it reads the verified token that
// middleware leaves on the context; without one, IsAdminRequest reports false
// and every admin-only tool is refused. It is installed in the same Use() call
// as AuthMiddleware on the /v1 group, so the two cannot be separated by a later
// edit, and attached by hand to /dispatch, which sits outside that group. A
// context with no authorizer is unrestricted, so any route that can reach the
// tool registry must have this middleware in front of it.
func toolAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.IsAdminRequest(c)
		a := newToolAuthorizer(admin)
		c.Request = c.Request.WithContext(tools.WithAuthorizer(c.Request.Context(), a))
		c.Next()
	}
}
