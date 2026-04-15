---
name: tool-intercept-logger
model: sonnet
description: "Produces structured OTEL-compatible log entries and a session summary (call counts, success rate, cost, top tools by cost and frequency) for all tool executions within a session. Use when: \"log tool calls for this session\", \"debug unexpected tool selection\", \"audit token cost by tool\", \"validate a new MCP integration\", \"build a session summary for handoff\"."
license: proprietary
category: system-health
triggers:
  - "log tool calls for this session"
  - "debug unexpected tool selection"
  - "audit token cost by tool"
  - "validate a new MCP integration"
  - "build a session summary for handoff"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: session_context
    type: string
    description: Session identifier or context for the logging run
    required: false
outputs:
  - name: session_log
    type: string
    description: Structured OTEL-compatible log entries and session summary with call counts, success rate, cost, and top tools
---
## I. Philosophy

You cannot improve what you cannot see. Tool call logging is not just
debugging infrastructure — it is the foundation of agent observability.
Understanding which tools are called, how often, at what cost, and with what
success rate is prerequisite knowledge for any meaningful performance
optimization or behavioral audit.

OTEL-compatible attribute naming ensures that logs produced here can be
consumed by any standards-compliant observability pipeline without translation.
Dual-layer storage (SQLite + OTEL) means the data is queryable locally and
exportable to external systems without re-instrumentation.

## II. When to Use

Use this skill when:

- Starting a new session and establishing a baseline of tool call patterns
  before any optimization work.
- Debugging unexpected agent behavior — especially looping, repeated calls,
  or unexpected tool selections.
- Auditing token cost attributable to specific tools or namespaces.
- Validating that a new MCP server or tool integration is being called as
  expected and returning correct results.
- Building a session summary for handoff to another agent or for human review.

Do not use this skill as a replacement for the `agent-performance-report`
skill. Tool intercept logging captures raw, per-call data during a session.
Performance reporting aggregates historical span data across sessions. Use
both in sequence when doing a comprehensive system health review.

## III. Workflow

**Step 1 — Define logging scope.**

Before any tool executions begin (or at the point logging is activated), define
the scope of capture. Three scope modes are supported:

- `all` — capture every tool execution in the current session regardless of
  namespace or name
- `namespace` — capture all tools matching a namespace prefix, e.g.,
  `mcp_by_dojo:*` or `gateway:*`
- `named` — capture a specific list of tool names only

If scope is not specified by the user, default to `all` and note this in the
log header.

**Step 2 — Capture pre-execution context.**

For each tool execution that falls within scope, record the following before
the call is dispatched:

```
timestamp_start: <ISO 8601 with milliseconds>
llm.tool_name:   <fully qualified tool name>
parameters:      <sanitized argument map — redact any value matching patterns
                  for API keys, passwords, tokens, or PII>
estimated_tokens: <token estimate for input arguments>
node_id:         <DAG node ID if this call is part of an orchestration plan>
dependencies:    <list of node IDs this call depends on, if known>
```

The `node_id` and `dependencies` fields are populated from the orchestration
engine span context (`node.<toolName>`) when available. Leave them null for
ad-hoc calls outside a DAG plan.

**Step 3 — Capture post-execution results.**

After the tool call completes (or times out or errors), record:

```
timestamp_end:          <ISO 8601 with milliseconds>
llm.tool_duration_ms:   <elapsed time in milliseconds>
llm.input_tokens:       <actual input tokens consumed>
llm.output_tokens:      <actual output tokens produced>
llm.estimated_cost:     <cost in USD or normalized units if USD not available>
result_summary:         <first 200 characters of result, or "[binary]" for
                         non-text output>
status:                 <"success" | "error" | "timeout" | "blocked">
error_message:          <full error string if status is error, else null>
```

For SSE-delivered results (via `tool_invoked` and `tool_completed` events),
capture the event payload directly. The `tool_completed` event provides
`tool_name`, `result`, and `duration_ms` — use these as the authoritative
post-execution values rather than reconstructing them.

**Step 4 — Format as structured log entry.**

Combine pre- and post-execution context into a single OTEL-compatible span
record using the attribute naming conventions from the Gateway TraceLogger:

```
span_name:            "node.<llm.tool_name>"
llm.model:            <model ID if this call was made by a specific model>
llm.tool_name:        <tool name>
llm.input_tokens:     <int>
llm.output_tokens:    <int>
llm.latency_ms:       <same as tool_duration_ms>
llm.estimated_cost:   <float>
llm.tool_duration_ms: <int>
node_id:              <string or null>
dependencies:         <array or null>
status:               <success|error|timeout|blocked>
```

Write each record to the dual-layer store: SQLite for local query, OTEL exporter
for pipeline forwarding. If the OTEL exporter is not configured, write to
SQLite only and note the missing exporter in the session log header.

**Step 5 — Aggregate session summary.**

After the session ends (or when explicitly requested), compute session-level
aggregates:

```
total_calls:       <count of all logged executions>
success_rate:      <successful calls / total calls as percentage>
total_input_tokens:  <sum>
total_output_tokens: <sum>
total_cost:        <sum of llm.estimated_cost>
avg_duration_ms:   <mean of llm.tool_duration_ms>
p95_duration_ms:   <95th percentile of llm.tool_duration_ms>
error_rate:        <errors / total calls as percentage>
top_tools_by_cost: <top 5 tools ranked by cumulative estimated cost>
top_tools_by_calls: <top 5 tools ranked by call count>
```

**Step 6 — Output.**

Return to the caller:

```
Log scope: <all | namespace:<pattern> | named:<list>>
Session log header: <start timestamp, model, scope, exporter status>
Per-call entries: <formatted table or structured list>
Session summary: <aggregated metrics from Step 5>
```

If the session produced more than 50 log entries, summarize per-call entries
by tool namespace rather than listing each call individually.

## IV. Best Practices

- Always sanitize arguments before logging. API keys, bearer tokens, passwords,
  and PII should be redacted to `[REDACTED]` before any persistence or output.
  Use pattern matching (`Bearer `, `sk-`, `password`, `token`) as a first pass.
- Use `tool_completed` SSE event data as the authoritative source for
  post-execution values. Do not reconstruct duration from timestamps if the
  event payload provides `duration_ms`.
- When logging tools outside a DAG plan, set `node_id` to `null` and omit
  `dependencies`. Do not invent node IDs for ad-hoc calls.
- Timestamp precision matters for latency analysis. Always use ISO 8601 with
  millisecond resolution, not second resolution.
- Blocked calls (those stopped by `budget-guard`) should be logged with
  `status: "blocked"` and a `result_summary` of `"[budget guard: <decision>]"`.
  They are observable events even though they produced no tool output.
- The OTEL attribute keys defined here are stable contracts. Do not rename them
  for brevity or clarity — downstream consumers depend on the exact key names.

## V. Quality Checklist

Before completing this skill, verify:

- [ ] Scope defined before any logging begins (not retroactively applied)
- [ ] Argument sanitization applied — no secrets or PII in persisted records
- [ ] Both pre-execution and post-execution context captured for each entry
- [ ] OTEL attribute names match Gateway TraceLogger conventions exactly
- [ ] Blocked calls logged with status "blocked", not silently omitted
- [ ] Session summary computed with all six aggregate metrics
- [ ] Exporter status noted in session log header if OTEL pipeline not configured
- [ ] Large sessions (>50 entries) summarized by namespace in output

## Output
- Per-call log entries in OTEL-compatible format written to SQLite (local query) and forwarded to the OTEL exporter pipeline if configured.
- A session summary block: total calls, success rate, total input/output tokens, total cost, avg duration, p95 duration, error rate, top 5 tools by cost, top 5 tools by call count.
- A session log header noting scope, start timestamp, model, and exporter status.

## Examples
**Scenario 1:** "Log all tool calls for this session" → Scope set to `all`. Per-call entries captured for 23 tool calls. Session summary produced at session end showing `web_search` as top cost driver (42% of total cost) and `bash` as top by call count (11 calls).
**Scenario 2:** "Validate that the new GitHub MCP integration is being called correctly" → Scope set to `namespace:mcp__github__*`. 6 calls logged. Result summaries confirm correct arguments being passed. One call returned a 404 — error_message captured and surfaced.

## Edge Cases
- Blocked calls (stopped by `budget-guard`) must be logged with `status: "blocked"` and a result_summary of `"[budget guard: BLOCK]"` — they are observable events even though they produced no tool output.
- For SSE-delivered results, use the `tool_completed` event payload as the authoritative source for duration_ms and result rather than reconstructing from timestamps.

## Anti-Patterns
- Using this skill instead of `agent-performance-report` for historical analysis across sessions — tool intercept logging captures live, per-call data for the current session; performance reporting aggregates historical spans across sessions.
- Logging raw argument values without sanitization — API keys, bearer tokens, and PII must be redacted to `[REDACTED]` before any persistence or output.
