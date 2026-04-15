---
name: hooks-reference
model: sonnet
description: "Produces a hook script scaffold and settings.json registration block by looking up the correct lifecycle event, exit code pattern, and configuration for a Claude Code hook. Use when: \"write a new hook\", \"hook is not firing\", \"block a tool call mechanically\", \"configure sub-agent coordination\", \"implement a security block pattern\"."
license: proprietary
category: system-health
triggers:
  - "write a new hook"
  - "hook is not firing"
  - "block a tool call mechanically"
  - "configure sub-agent coordination"
  - "implement a security block pattern"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: hook_requirement
    type: string
    description: Description of what the hook should do — block a tool, coordinate agents, implement a security gate, etc.
    required: true
outputs:
  - name: hook_scaffold
    type: string
    description: Hook script scaffold and settings.json registration block for the specified lifecycle event and exit code pattern
---
# Hooks Reference

## I. Philosophy

Hooks are the mechanical enforcement layer of agent safety and observability.
Unlike CLAUDE.md instructions, which guide agent behavior through documentation,
hooks execute code at lifecycle boundaries where the agent cannot override them.
A well-designed hook system makes dangerous operations impossible rather than
merely discouraged.

This skill is a reference card, not a tutorial. It assumes you know what hooks
are and need to quickly find the right event, the right exit code, or the right
configuration pattern. For architectural understanding, read the
`seed_hooks_lifecycle_mastery` and `seed_mechanical_safety_enforcement` seeds.

## II. When to Use

- Writing a new hook and need to know which event type to target.
- Debugging why a hook is not firing (wrong event name, wrong exit code).
- Reviewing the full lifecycle to understand execution order.
- Implementing a security block pattern (exit code 2).
- Configuring hooks in `.claude/settings.json` or plugin `hooks/hooks.json`.
- Building sub-agent coordination via SubagentStart/SubagentStop hooks.

Do not use this skill for designing observability dashboards (use
`observability-dashboard` instead) or for understanding the correction-capture
pipeline (use `reflect-and-learn` instead).

## III. All 26 Lifecycle Events

Events are organized by category. The "Block?" column indicates whether exit 2
halts execution.

### Session Events

| Event | Block? | Notes |
|-------|--------|-------|
| SessionStart | No | Fires on session open. Stdout added to Claude context. Write to $CLAUDE_ENV_FILE to persist env vars. |
| SessionEnd | No | Fires on session termination. Output ignored. |
| InstructionsLoaded | No | Fires after CLAUDE.md and system prompt load. |
| CwdChanged | No | Fires when working directory changes. Write to $CLAUDE_ENV_FILE to inject env vars for subsequent Bash calls. |

### User / Prompt Events

| Event | Block? | Notes |
|-------|--------|-------|
| UserPromptSubmit | Yes (exit 2) | Fires before prompt is processed. Stdout added to Claude context. Exit 2 blocks submission. |
| Notification | No | Fires when Claude surfaces a notification. |

### Tool Execution Events

| Event | Block? | Notes |
|-------|--------|-------|
| PreToolUse | Yes (exit 2) | Fires before every tool call. Stdout JSON can rewrite tool input (`updatedInput`) or inject context (`additionalContext`). Use `if` field to scope without spawning a process. |
| PostToolUse | No | Fires after successful tool call. Stdout JSON can replace MCP tool output via `updatedMCPToolOutput`. |
| PostToolUseFailure | No | Fires after a tool call that errored. |
| PermissionRequest | Yes (exit 2 or `{"permissionDecision":"deny"}`) | Fires when a tool requires a permission check. |
| PermissionDenied | No | Fires after permission is denied. Cannot block. |

### SubAgent Events

| Event | Block? | Notes |
|-------|--------|-------|
| SubagentStart | No | Fires when a subagent starts. Cannot block. |
| SubagentStop | Yes (exit 2) | Fires when a subagent finishes. Exit 2 forces continuation. |

### Completion Events

| Event | Block? | Notes |
|-------|--------|-------|
| Stop | Yes (exit 2) | Fires when turn ends normally. Exit 2 forces Claude to continue. Guard with `stop_hook_active` flag. |
| StopFailure | No | Fires when turn ends due to API or runtime error. Output ignored. |

### Context Events

| Event | Block? | Notes |
|-------|--------|-------|
| PreCompact | No | Fires before context compaction. Stdout injected into compaction instructions. |
| PostCompact | No | Fires after context compaction. Cannot block. |

### Config / File Events

| Event | Block? | Notes |
|-------|--------|-------|
| ConfigChange | Yes (exit 2) | Fires when a config file changes mid-session. Matcher values: `"user_settings"`, `"project_settings"`, `"local_settings"`, `"policy_settings"`, `"skills"`. Exit 2 blocks the change. |
| FileChanged | No | Fires when a watched file changes. Matcher = filename basename. Cannot block. |

### Worktree Events

| Event | Block? | Notes |
|-------|--------|-------|
| WorktreeCreate | Yes (any non-zero) | Fires when a worktree is being created. Any non-zero exit aborts creation. |
| WorktreeRemove | No | Fires when a worktree is removed. Cannot block. |

### Agent-Team Events

| Event | Block? | Notes |
|-------|--------|-------|
| TeammateIdle | Yes (exit 2) | Fires when an agent-team teammate is about to go idle. Exit 2 prevents idle state. |
| TaskCreated | Yes (exit 2) | Fires when a task is being created. Exit 2 rolls back the task. |

### MCP Events

| Event | Block? | Notes |
|-------|--------|-------|
| Elicitation | Yes (exit 2) | Fires when an MCP server requests user input. Exit 2 denies the request. |
| ElicitationResult | Yes (exit 2) | Fires after user responds to MCP elicitation. Exit 2 blocks the response from being sent. |

**NOTE:** `TaskCompleted` does NOT exist in Claude Code hooks. Do not register or reference it.

## IV. Exit Code Reference

| Exit Code | Behavior |
|-----------|----------|
| 0 | Allow. Stdout may be parsed as JSON. For UserPromptSubmit and SessionStart, stdout is added to Claude context. |
| 2 | BLOCK. Stderr is fed back to Claude as error feedback. ONLY exit 2 blocks — not exit 1. |
| 1 or other | Non-blocking error. Shows one-line error notice in UI. Execution continues. |

**Critical:** exit 1 does NOT block. Only exit 2 blocks. This is the single most common hook debugging mistake.

## V. Four Hook Types

### `command` (default)
Runs a shell script or executable. Receives event JSON on stdin.

```json
{
  "hooks": {
    "PreToolUse": [{"matcher": "Write", "hooks": [{"type": "command", "command": "python3 .claude/hooks/guard.py"}]}]
  }
}
```

### `prompt`
Single-turn Haiku LLM call. Returns `{"ok": true/false, "reason": "..."}`. Low latency but requires network. Best for Stop/SubagentStop task-completion enforcement.

```json
{"type": "prompt", "prompt": "Did the agent complete all required steps? Check for missing tests."}
```

### `agent`
Spawns a subagent with Read/Grep/Glob for codebase verification. Returns `{"ok": true/false, "reason": "..."}`. 60-second timeout, 50 tool turns. Use on Stop for deep multi-file verification.

```json
{"type": "agent", "prompt": "Verify all modified Go files compile. Run go build ./... and report."}
```

### `http`
POSTs event JSON to an HTTP endpoint. Response uses the same format as `command` hooks. Headers support `$VAR` interpolation (requires `allowedEnvVars` list).

```json
{"type": "http", "url": "https://hooks.example.com/events", "headers": {"Authorization": "Bearer $HOOK_TOKEN"}, "allowedEnvVars": ["HOOK_TOKEN"]}
```

## VI. Advanced Features

### `if` Field (PreToolUse / PostToolUse scoping)
Scopes a hook to specific tool+argument patterns without spawning a process. Only works on: PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, PermissionDenied.

```json
{"if": "Write(**/SKILL.md)", "hooks": [{"type": "command", "command": "skill-validate.sh"}]}
{"if": "Bash(git *)", "hooks": [{"type": "command", "command": "git-guard.py"}]}
```

### `async: true`
Runs the hook in the background. Exit code and stdout are ignored. Use for logging and notifications where latency matters.

```json
{"type": "command", "command": "logger.py", "async": true}
```

### `updatedInput` (PreToolUse)
PreToolUse hooks can rewrite tool arguments before the tool executes via stdout JSON:

```json
{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "allow", "updatedInput": {"content": "...modified content..."}}}
```

### `additionalContext` (PreToolUse)
Inject context without blocking. Claude sees this as additional background:

```json
{"hookSpecificOutput": {"hookEventName": "PreToolUse", "additionalContext": "Note: this file is under active review."}}
```

### `updatedMCPToolOutput` (PostToolUse)
PostToolUse hooks can replace MCP tool output before Claude sees it:

```json
{"hookSpecificOutput": {"hookEventName": "PostToolUse", "updatedMCPToolOutput": "...filtered output..."}}
```

### `stop_hook_active`
Field present in Stop hook stdin JSON when a Stop hook already forced continuation this turn. Check this to prevent infinite loops:

```python
event = json.loads(sys.stdin.read())
if event.get("stop_hook_active"):
    sys.exit(0)  # Do not force continuation again
```

### `$CLAUDE_ENV_FILE`
Write `export VAR=val` lines to this file path (available in env) during SessionStart or CwdChanged hooks to persist environment variables across all subsequent Bash calls in the session.

### Plugin Hooks
Plugin hooks live at `{plugin-root}/hooks/hooks.json` with the same format as the `hooks` object in settings.json. Two extra env vars are available inside plugin hooks:
- `$CLAUDE_PLUGIN_ROOT` — absolute path to the plugin directory
- `$CLAUDE_PLUGIN_DATA` — writable data directory for the plugin

## VII. Workflow

**Step 1 -- Identify the lifecycle stage.**
Use the event tables in Section III to find the right event.

**Step 2 -- Choose the enforcement pattern.**
- Block (exit 2): halts the operation. Only for PreToolUse, PermissionRequest, UserPromptSubmit, Stop, SubagentStop, TeammateIdle, TaskCreated, ConfigChange, Elicitation, ElicitationResult, WorktreeCreate.
- Observe (exit 0): event logged, execution continues. Use for all non-blocking events.
- Modify (stdout JSON): rewrites inputs or injects context. Use updatedInput, additionalContext, updatedMCPToolOutput.

**Step 3 -- Write the hook script.**

UV single-file Python with embedded dependencies:

```python
# /// script
# dependencies = ["requests"]
# ///
import json, sys

event = json.loads(sys.stdin.read())
tool_name = event.get("tool_name", "")
tool_input = event.get("tool_input", {})

# Guard: block writes to protected paths
if tool_input.get("file_path", "").endswith("CLAUDE.md"):
    print("Blocked: CLAUDE.md is write-protected", file=sys.stderr)
    sys.exit(2)

sys.exit(0)
```

**Step 4 -- Register in settings.json.**

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [{"type": "command", "command": "python3 .claude/hooks/pre-tool-use/guard.py"}]
      }
    ],
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "python3 .claude/hooks/stop/done-notify.py"}]
      }
    ]
  }
}
```

**Step 5 -- Test the hook.**
Trigger the target event in a real session. Verify side effects (log file, blocked operation, HTTP POST).

## VIII. Hooks Active in This Ecosystem

These hooks are registered in `~/.claude/settings.json` (global, fires in all projects):

| Hook Name | Event | Type | Behavior |
|-----------|-------|------|----------|
| go-build-check | PostToolUse | command | Runs `go build ./...` after Go file writes |
| go-test-check | PostToolUse | command | Runs `go test ./...` after *_test.go writes |
| svelte5-lint | PostToolUse | command | Lints Svelte files for `on:` directive usage |
| rust-check | PostToolUse | command | Runs `cargo check` after Rust file writes |
| ts-check | PostToolUse | command | Runs `tsc --noEmit` after TypeScript writes |
| skill-validate | PostToolUse | command | if: "Write\|Edit(**\/SKILL.md)" — validates skill structure |
| agent-log | PostToolUse | command | if: "Agent" — logs agent invocations to JSONL |
| destructive-guard | PreToolUse | command | Blocks rm -rf and destructive shell patterns |
| force-push-guard | PreToolUse | command | Blocks `git push --force` to main/master |
| pre-commit-build-gate | PreToolUse | command | if: "Bash(git commit *)" — runs go build before commit |
| done-notification | Stop | command | macOS notification when turn completes |
| context-inject | UserPromptSubmit | command | Injects project context into each prompt |

## IX. Quality Checklist

- [ ] Hook targets the correct lifecycle event (check Section III)
- [ ] Exit code matches intent (0 = observe, 2 = block, NOT 1)
- [ ] stop_hook_active guard present in all Stop hooks
- [ ] Script is idempotent (safe to re-execute)
- [ ] Performance acceptable for hot-path hooks (PreToolUse fires on every tool call)
- [ ] Hook registered in settings.json or plugin hooks.json
- [ ] async: true used for logging/notification hooks to avoid latency

## X. Related Skills

- `observability-dashboard` -- Visualize events captured by hooks
- `tool-intercept-logger` -- OTEL-compatible tool call logging
- `claude-md-guardian` -- Protect CLAUDE.md via hook enforcement
- `agent-performance-report` -- Aggregate hook data into performance metrics

## Output
- A UV single-file Python hook script (with embedded dependency declarations) ready to drop into `.claude/hooks/{event-type}/`.
- A `settings.json` snippet registering the hook under the correct event key.
- Exit code guidance specific to the chosen enforcement pattern (0 = observe, 2 = block — never 1).

## Examples
**Scenario 1:** "I want to block sub-agents from writing to CLAUDE.md" → PreToolUse hook with `if: "Write|Edit(**/CLAUDE.md)"` field. Script exits 2 when matched. Settings.json snippet with `if` scoping provided.
**Scenario 2:** "Log every tool call to a file for debugging" → PostToolUse hook (exit 0, async: true) that appends tool_name, duration_ms, and status to `.claude/hooks/tool-log.jsonl`. No blocking.
**Scenario 3:** "Verify all required steps are done before agent stops" → Stop hook using `prompt` or `agent` type. Guard with stop_hook_active check to prevent infinite loop.
**Scenario 4:** "Persist an API key across Bash calls after SessionStart" → SessionStart hook that writes `export MY_API_KEY=...` to $CLAUDE_ENV_FILE.

## Edge Cases
- Stop hooks that trigger agent re-runs MUST check `stop_hook_active` in stdin JSON and exit 0 if true — without this guard the hook will loop until the session is killed.
- If a PreToolUse hook has heavy imports, move them inside the function or use lazy loading — a 2-second import delay fires on every single tool call.
- WorktreeCreate blocks on any non-zero exit, not just exit 2. All other blocking events require specifically exit 2.
- CwdChanged cannot block, but $CLAUDE_ENV_FILE is the correct channel to propagate environment changes to subsequent Bash calls.
- `prompt` and `agent` hook types require network access; if the network is unavailable they will fail non-blockingly (treated as exit 1).

## Anti-Patterns
- Using the wrong event name casing ("pre_tool_use" instead of "PreToolUse") — the hook silently never fires.
- Using exit 1 intending to block — exit 1 shows a warning but does NOT block. Only exit 2 blocks.
- Referencing TaskCompleted — this event does not exist in Claude Code. Use Stop or SubagentStop instead.
- Assuming hooks survive context compaction — use Setup/SessionStart hooks for re-initialization rather than relying on context that may be compacted away.
- Writing heavy synchronous hooks on PostToolUse without async: true — blocks Claude's response loop for every tool execution.
