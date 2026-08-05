package server

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DojoGenesis/gateway/server/middleware"
)

// This file decides ONE thing: may a workflow step turn an API-supplied string
// into a process on this host?
//
// WHY THIS GATE EXISTS (DGS-108):
// handleWorkflowExecute constructs CommandSkillRunner unconditionally, and a
// workflow step with skill="run_command" reaches
// exec.CommandContext(shell, "-c", cmd) in tools.RunCommand. The only thing
// standing between a stored workflow definition and a shell is
// tools.validateCommand, a five-pattern substring denylist (rm -rf /, mkfs,
// dd if=/dev/zero, forkbomb, chmod 777/666 /). `cat /etc/dojo/env` does not
// match any of them. Neither does an outbound curl. The denylist is not, and
// cannot be, a security boundary on a string handed to `sh -c`.
//
// WHY THE FIX IS HERE AND NOT IN THE AUTH LAYER:
// the obvious reading of DGS-108 is "/api/* has no auth, so add auth". That
// does not close this. POST /auth/register is open (DGS-101), so an anonymous
// caller can mint themselves a role=user token and walk the identical path
// with one extra HTTP request. Authentication answers "who are you"; this is a
// capability question — "may anyone at all turn this API into a shell on this
// box" — and on a production gateway the answer is no by default. The two
// fixes are complementary, not alternatives: DGS-100 still owes /api/* an auth
// story, and this gate still holds if that story is ever bypassed.
//
// WHY NOT SIMPLY DELETE run_command:
// it has a real consumer. commissions/atlas-pipeline-workflow.json is a
// six-step DAG built entirely from run_command. Its working_directory is
// /Users/alfonsomorales/ZenflowProjects/CWD/atlas — a local macOS path, not
// anything on the deployed host. That is the whole distinction this gate
// encodes: shelling out is a legitimate LOCAL developer capability and an
// illegitimate PRODUCTION one. Defaulting off in production keeps that
// workflow working for whoever runs it on their own machine while removing the
// primitive from the internet-facing deployment.
//
// This mirrors EnsureJWTSecretConfigured in server/middleware/jwt_secret.go:
// development is deliberately unaffected and needs no configuration, and
// production fails closed unless an operator has said otherwise in writing.

// EnvWorkflowRunCommand is the environment variable controlling whether
// workflow steps may dispatch skill="run_command" to a real shell.
//
// Unset is the case that matters, and it is resolved by environment:
// enabled outside production, disabled in production. Setting it explicitly
// overrides that in either direction, on any host.
const EnvWorkflowRunCommand = "WORKFLOW_RUN_COMMAND_ENABLED"

// runCommandDecision records what was decided and why, so startup can say so
// out loud without the caller re-deriving the rule.
type runCommandDecision struct {
	// Enabled reports whether run_command steps may reach a shell.
	Enabled bool
	// Explicit reports that an operator set EnvWorkflowRunCommand themselves,
	// rather than inheriting the environment-derived default.
	Explicit bool
	// Production reports that this process considers itself production.
	Production bool
}

// resolveRunCommand applies the precedence rule: an explicit, parseable value
// wins; otherwise production disables and everything else enables.
//
// A value that is set but unparseable (WORKFLOW_RUN_COMMAND_ENABLED=yes-please)
// is deliberately treated as ABSENT rather than as true. An operator who
// fat-fingers this variable in production gets the safe default, not a shell.
func resolveRunCommand() runCommandDecision {
	production := isProductionEnvironment()

	raw, ok := lookupNonBlankEnvVar(EnvWorkflowRunCommand)
	if ok {
		if v, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
			return runCommandDecision{Enabled: v, Explicit: true, Production: production}
		}
	}

	return runCommandDecision{Enabled: !production, Explicit: false, Production: production}
}

// workflowRunCommandEnabled is the hot-path predicate consulted by
// CommandSkillRunner before any shell is spawned.
//
// It reads the environment on every call rather than caching. This runs once
// per workflow STEP, not per request, so the cost is irrelevant, and reading
// live means a test can flip the variable with t.Setenv and observe the change
// without a process restart or an exported reset hook.
func workflowRunCommandEnabled() bool {
	return resolveRunCommand().Enabled
}

// isProductionEnvironment delegates to the one definition of "production" in
// the tree (middleware.IsProductionEnvironment). It is not re-implemented here:
// DGS-112 was two functions in the same package disagreeing about what counts
// as production, and a third copy is how that comes back.
func isProductionEnvironment() bool {
	return middleware.IsProductionEnvironment()
}

// lookupNonBlankEnvVar treats unset, empty and whitespace-only as absent, so
// `WORKFLOW_RUN_COMMAND_ENABLED=` cannot pass for a configured value.
func lookupNonBlankEnvVar(key string) (string, bool) {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

// errRunCommandDisabled is what a blocked step reports. It names the variable,
// so an operator who hits this while legitimately trying to run a pipeline can
// act on the message without reading this file.
func errRunCommandDisabled() error {
	return fmt.Errorf(
		"run_command is disabled on this gateway: a workflow step asked to execute a shell "+
			"command, and %s is not enabled. This is the default in production because a "+
			"workflow definition arriving over /api/workflows would otherwise become a process "+
			"on this host, running as the gateway and inheriting its environment (including "+
			"JWT_SECRET). To allow it anyway — only on a host where every caller that can store "+
			"a workflow is already trusted with a shell — set %s=true and restart",
		EnvWorkflowRunCommand, EnvWorkflowRunCommand)
}

// RunCommandStartupWarning returns a log line when this process will allow
// workflow steps to shell out in production, and "" otherwise.
//
// Enabling this in production is a legitimate operator choice on a host whose
// /api surface is genuinely closed, but it should never be a quiet one — the
// combination of an enabled run_command and an unauthenticated /api/* is
// exactly the DGS-108 chain.
func RunCommandStartupWarning() string {
	d := resolveRunCommand()
	if !d.Production || !d.Enabled {
		return ""
	}
	return fmt.Sprintf(
		"%s=true in production: workflow steps may execute shell commands on this host. "+
			"Anyone who can POST a workflow definition to /api/workflows can run a command as "+
			"the gateway process. Confirm /api/* is authenticated at the edge or in code before "+
			"leaving this enabled (DGS-108)", EnvWorkflowRunCommand)
}
