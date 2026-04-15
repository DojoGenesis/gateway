---
name: observability-dashboard
model: sonnet
description: "Produces a running multi-agent observability dashboard by deploying the hook capture pipeline, SQLite WAL store, and WebSocket-streaming Vue client — then validates it with a live test session. Use when: \"set up monitoring for multi-agent workflows\", \"debug parallel agent coordination\", \"audit tool call history across a session\", \"build visibility for a demo or review\"."
license: proprietary
category: system-health
triggers:
  - "set up monitoring for multi-agent workflows"
  - "debug parallel agent coordination"
  - "audit tool call history across a session"
  - "build visibility for a demo or review"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec_path
    type: string
    description: Path to the observability dashboard specification to implement
    required: false
outputs:
  - name: dashboard
    type: string
    description: Running multi-agent observability dashboard with hook capture pipeline, SQLite WAL store, and WebSocket-streaming client deployed and validated
---
# Observability Dashboard

## I. Philosophy

Observability is the ability to answer questions about system behavior that you
did not anticipate when building the system. For multi-agent workflows, the
fundamental questions are: What happened? Why did it happen? What should I do
about it?

This skill covers the operational side -- deploying, configuring, and using the
dashboard. For designing new dashboard specs, use `observability-dashboard-spec`.
For the underlying hook taxonomy, use `hooks-reference`.

## II. When to Use

- Setting up a monitoring dashboard for a multi-agent workflow for the first time.
- Debugging coordination failures between parallel agents.
- Auditing which tools were called, how often, and with what results across a session.
- Building real-time visibility into agent execution for demonstrations or reviews.
- Comparing agent behavior across sessions (before/after a CLAUDE.md change).

Do not use this skill for designing the dashboard architecture (use
`observability-dashboard-spec`) or for writing individual hooks (use
`hooks-reference`).

## III. Workflow

**Step 1 -- Deploy the event pipeline.**

Set up the three-layer architecture:

```
Hook Scripts --> HTTP POST --> Bun/Node Server --> SQLite (WAL mode)
                                    |
                                    +--> WebSocket --> Dashboard Client
```

- Hook scripts: One per event type, capturing contextual data as JSON payloads.
- Server: Bun or Node.js process that validates events, persists to SQLite, and
  broadcasts to WebSocket clients simultaneously.
- SQLite: WAL mode mandatory for concurrent writes from multiple agents.

**Step 2 -- Configure hook capture.**

Install hooks for the 12 observable event types:
PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, Notification,
UserPromptSubmit, Stop, SubagentStart, SubagentStop, PreCompact, SessionStart,
SessionEnd.

Each hook gathers event-specific fields (tool_name, agent_id, session_id,
notification_type) and transmits via HTTP POST.

**Step 3 -- Launch the dashboard.**

Start the Vue 3 (or equivalent) client connecting via WebSocket to the server.
Configure the five core views:

- **Timeline** -- Chronological event rows with auto-scroll and dual-color coding
  (app color on left border, session color on secondary border).
- **Filtering** -- Multi-criteria selection by app, session, event type.
- **Pulse chart** -- Canvas-based activity density with session-specific colors
  and event-type visual indicators.
- **Transcript modal** -- Chat history viewer for Stop/SubagentStop events.
- **Time windows** -- 1-min, 3-min, 5-min adaptive aggregation.

**Step 4 -- Validate with a test session.**

Run a simple multi-agent workflow (2 agents in parallel Tmux panes). Verify:
- Both agents appear as distinct sessions in the dashboard.
- PreToolUse and PostToolUse events pair correctly.
- Filtering by session isolates one agent's events.
- Transcripts are captured on Stop events.

**Step 5 -- Tune and extend.**

Add custom event-specific field forwarding. Implement tool-type visual encoding
for rapid scanning. Configure time-based filtering windows for the team's
preferred monitoring cadence.

## IV. Best Practices

- Always use SQLite WAL mode when multiple agents write simultaneously.
- Color-code by session, not by agent name (names can collide).
- Capture transcripts on Stop and SubagentStop for post-hoc analysis.
- Use the `stop_hook_active` guard to prevent infinite loops.
- Keep event payloads under 10KB to avoid WebSocket congestion.
- Run automatic schema migrations to handle evolution across versions.

## V. Quality Checklist

- [ ] SQLite is in WAL mode
- [ ] All 12 event types have corresponding hooks installed
- [ ] WebSocket broadcasts events within 100ms of receipt
- [ ] Dashboard distinguishes parallel sessions visually
- [ ] Filtering works across app, session, and event type dimensions
- [ ] Transcripts are captured and viewable for Stop events
- [ ] Time-window filtering adapts data granularity correctly

## VI. Common Pitfalls

- **Polling instead of WebSocket push.** Polling adds latency and load.
- **Single-color dashboards.** Without session-specific colors, parallel agent
  events become indistinguishable.
- **Missing WAL mode.** SQLite defaults to rollback journal, which blocks on
  concurrent writes. Multiple agents will cause lock contention.
- **Logging without querying.** A database of events nobody examines is waste.

## VII. Related Skills

- `hooks-reference` -- Event taxonomy and hook authoring patterns
- `observability-dashboard-spec` -- Architectural spec for dashboard design
- `tool-intercept-logger` -- OTEL-compatible per-call logging
- `agent-performance-report` -- Aggregated metrics from historical spans

## Output
- A running Bun/Node.js server persisting events to SQLite (WAL mode) and broadcasting via WebSocket.
- Hook scripts installed for all 12 observable event types in `.claude/hooks/`.
- A Vue 3 (or vanilla JS) dashboard client connected to the server, showing Timeline, Pulse chart, Transcript modal, and Filtering views.
- A validation report from the test session confirming both agents appear as distinct sessions and PreToolUse/PostToolUse events pair correctly.

## Examples
**Scenario 1:** "Set up monitoring before our multi-agent sprint" → Three-layer pipeline deployed, hooks installed, dashboard launched. Test session with 2 parallel agents in Tmux panes confirms both sessions are visible and color-coded.
**Scenario 2:** "Debug why agent B stopped after agent A's tool call" → Dashboard filtering by session isolates agent B's event stream. Transcript modal on the SubagentStop event reveals the error message in the captured chat history.

## Edge Cases
- If multiple agents write simultaneously to SQLite without WAL mode, writes will block and events will be lost — always verify WAL mode is set before starting multi-agent sessions.
- Keep event payloads under 10KB; large result blobs (e.g., full file contents) should be truncated to a summary before WebSocket broadcast to avoid congestion.

## Anti-Patterns
- Polling the server for events instead of using WebSocket push — polling adds latency and defeats the real-time purpose of the dashboard.
- Using a single color for all sessions — without session-specific color coding, parallel agent events are indistinguishable in the timeline.
