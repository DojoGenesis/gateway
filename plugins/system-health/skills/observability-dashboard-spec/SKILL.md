---
name: observability-dashboard-spec
model: sonnet
description: "Produces an architectural specification document mapping Gateway SSE events and OTEL spans to named dashboard widgets, a data flow diagram, and an MCP App config — ready to hand to an implementation agent. Use when: \"design the observability dashboard\", \"spec the monitoring app\", \"map events to widgets\", \"plan the agent dashboard\", \"extend the dashboard with new widget types\"."
license: Complete terms in LICENSE.txt
category: system-health
triggers:
  - "design the observability dashboard"
  - "spec the monitoring app"
  - "map events to widgets"
  - "plan the agent dashboard"
  - "extend the dashboard with new widget types"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: events_to_surface
    type: string[]
    description: Gateway SSE events and OTEL spans to map to dashboard widgets
    required: false
outputs:
  - name: dashboard_spec
    type: ref
    format: cas-ref
    description: Architectural specification document mapping events to named dashboard widgets, data flow diagram, and MCP App config
---
# Observability Dashboard Spec

Prompt-only skill (Tier 0) that produces an MCP App dashboard specification.

## I. Philosophy

Observability is not logging with a UI. It is the ability to answer questions about system behavior that you did not anticipate when building the system. A well-designed dashboard does not show you every data point — it shows you the right data at the right granularity to support the three fundamental questions: What happened? Why did it happen? What should I do about it?

The Dojo Platform emits 20+ event types across three layers (SSE, OTEL spans, orchestration events). The challenge is not collecting data — it is designing views that make multi-agent DAG execution legible at a glance.

## II. When to Use

- When planning the first observability MCP App for the Gateway
- When extending the dashboard with new widget types
- When the event catalog has expanded and the dashboard spec needs updating
- When designing dashboard views for specific personas (operator, developer, auditor)

## III. Workflow

### Step 1: Inventory the Event Catalog

Map all available data sources:

**SSE Events (real-time, push):**

| Event | Key Fields | Widget Candidate |
|-------|-----------|-----------------|
| intent_classified | intent, confidence | Intent distribution pie chart |
| provider_selected | provider, model | Model usage breakdown |
| tool_invoked | tool, arguments | Live tool call feed |
| tool_completed | tool, result, duration_ms | Tool latency histogram |
| orchestration_plan_created | plan_id, node_count, estimated_cost | DAG visualization (new plan) |
| orchestration_node_start | node_id, tool_name | DAG node state (running) |
| orchestration_node_end | node_id, state, duration_ms | DAG node state (complete/failed) |
| orchestration_replanning | reason, failed_nodes | Replanning alert |
| orchestration_complete | total_nodes, success/failed, duration_ms | DAG summary card |
| trace_span_start | trace_id, span_id, name | Trace waterfall (new span) |
| trace_span_end | duration_ms, status, metadata | Trace waterfall (close span) |
| memory_retrieved | memories_found | Memory hit rate |

**OTEL Span Attributes (queryable):**

| Attribute | Type | Widget Candidate |
|-----------|------|-----------------|
| llm.model | string | Model usage by time |
| llm.input_tokens | int | Token burn-down chart |
| llm.output_tokens | int | Token burn-down chart |
| llm.estimated_cost | float | Cost accumulation line |
| llm.tool_name | string | Tool frequency bar chart |
| llm.tool_duration_ms | int | Tool latency percentiles |

**BudgetTracker State (poll):**

| Field | Widget Candidate |
|-------|-----------------|
| query remaining | Budget gauge (per-query) |
| session remaining | Budget gauge (session) |
| monthly remaining | Budget gauge (monthly) |

### Step 2: Design the Dashboard Layout

Three-panel layout optimized for the operator persona:

```
┌─────────────────────────────────────────────────────────┐
│ HEADER: Agent Status Bar                                │
│ [Active Agents: 3] [DAGs Running: 1] [Budget: 72%]     │
├──────────────────────┬──────────────────────────────────┤
│ LEFT PANEL           │ CENTER PANEL                     │
│ (Navigation + KPIs)  │ (Primary visualization)          │
│                      │                                  │
│ - Session KPIs       │ - DAG Visualization (default)    │
│   - Total calls      │   OR                             │
│   - Total tokens     │ - Trace Waterfall                │
│   - Total cost       │   OR                             │
│   - Error rate       │ - Tool Latency Distribution      │
│   - Avg latency      │                                  │
│                      │                                  │
│ - Budget Gauges      │                                  │
│   - Query            │                                  │
│   - Session          │                                  │
│   - Monthly          │                                  │
│                      │                                  │
│ - Active Tools       │                                  │
│   (sorted by freq)   │                                  │
│                      │                                  │
├──────────────────────┼──────────────────────────────────┤
│ BOTTOM PANEL: Live Event Feed                           │
│ [tool_invoked] bash: ls -la                    2ms ago  │
│ [tool_completed] bash: success                 1ms ago  │
│ [orchestration_node_end] node_3: 245ms         0ms ago  │
└─────────────────────────────────────────────────────────┘
```

### Step 3: Specify Each Widget

**Widget 1: DAG Visualization (center panel default)**
- Source: orchestration_plan_created + node_start + node_end events
- Rendering: directed graph with nodes colored by state
  - Gray: pending, Blue: running, Green: completed, Red: failed, Yellow: replanning
- Interaction: click node to see inputs/outputs in a tooltip
- Update: real-time via SSE push (no polling)
- Layout: left-to-right, dependency edges as arrows, parallel nodes at same horizontal level

**Widget 2: Token Burn-Down Chart (left panel)**
- Source: OTEL spans with llm.input_tokens + llm.output_tokens
- Rendering: area chart, X=time, Y=cumulative tokens
- Three lines: input tokens, output tokens, budget remaining
- Alert: red zone when budget < 10% remaining
- Update: on each trace_span_end event

**Widget 3: Tool Latency Histogram (center panel, switchable)**
- Source: tool_completed events, duration_ms field
- Rendering: horizontal bar chart, tools sorted by p95 latency
- Color: green (< 500ms), yellow (500ms-2s), red (> 2s)
- Tooltip: min, p50, p95, max, call count
- Update: rolling window (last 100 calls or last 5 minutes)

**Widget 4: Budget Gauges (left panel)**
- Source: BudgetTracker state (poll every 5s, or on tool_completed event)
- Rendering: three circular gauges (query, session, monthly)
- Color: green (< 70%), yellow (70-90%), red (> 90%)
- Label: remaining tokens + percentage

**Widget 5: Live Event Feed (bottom panel)**
- Source: all SSE events
- Rendering: scrolling log, newest at bottom, auto-scroll
- Format: `[event_type] summary_text    {relative_time}`
- Filter: dropdown to filter by event type or tool namespace
- Capacity: last 200 events in DOM, older events discarded

**Widget 6: Trace Waterfall (center panel, switchable)**
- Source: trace_span_start + trace_span_end events
- Rendering: horizontal waterfall chart (like Chrome DevTools Network tab)
- X=time, each row=one span, width=duration
- Color by span type: model_invocation (blue), tool execution (green), orchestration (orange)
- Nested spans indented under parent (using ParentID)

### Step 4: Define the Data Flow

```
Gateway SSE Endpoint (/v1/events/stream)
    |
    v
EventSource (browser) ────────────────────────┐
    |                                          |
    v                                          v
Event Router                              Event Store
(dispatches by type                    (in-memory ring buffer,
 to widget handlers)                    last 1000 events)
    |                                          |
    ├─> DAG Widget Handler                     |
    ├─> Token Chart Handler                    |
    ├─> Latency Histogram Handler              |
    ├─> Budget Gauge Handler (+ 5s poll)       |
    └─> Live Feed Handler <────────────────────┘
```

**State management:**
- Each widget maintains its own state slice (no global store)
- Widget handlers are pure functions: (currentState, event) -> newState
- DOM updates via targeted element replacement (no full re-render)

### Step 5: Specify MCP App Integration

The dashboard is hosted as an MCP App via the Gateway's Apps infrastructure:

```yaml
app:
  id: "observability-dashboard"
  name: "Agent Observability"
  version: "1.0.0"
  entry_point: "index.html"
  permissions:
    - read_events        # SSE event stream
    - read_traces        # OTEL span query
    - read_budget        # BudgetTracker state
  tools:                 # App-only tools (hidden from LLM)
    - query_spans        # TraceStorage.QuerySpans(timeRange, filter)
    - get_budget_status  # BudgetTracker.GetRemaining(userID)
    - get_plan_details   # OrchestrationExecutor.GetPlan(planID)
```

**Sandbox model:** Two-layer iframe (per ADR-016):
- Outer iframe: Gateway origin, handles SSE connection + app-only tool calls
- Inner iframe: Separate origin, renders HTML dashboard, communicates via postMessage

### Step 6: Write the Implementation Spec

Produce a spec document that includes:
1. Widget inventory (from Step 3)
2. Data flow diagram (from Step 4)
3. MCP App config (from Step 5)
4. HTML/CSS/JS architecture (single-page, no framework, vanilla JS + CSS Grid)
5. Event handler signatures for each widget
6. Estimated bundle size (target: < 50KB gzipped)
7. Browser compatibility (modern browsers, no IE)

## IV. Best Practices

1. **Design for glanceability.** The operator should understand system state in < 3 seconds. KPIs in the left panel, primary visualization in the center.

2. **Push, do not poll.** SSE provides real-time updates. Only poll for BudgetTracker state (which has no event emission yet).

3. **Degrade gracefully.** If the SSE connection drops, show a "disconnected" banner, keep last-known state visible, and auto-reconnect with exponential backoff.

4. **Respect the iframe sandbox.** All data passes through postMessage. No direct DOM access across the iframe boundary.

5. **Keep it vanilla.** No React, no Vue, no build step. The MCP App must be a single HTML file that the Gateway serves directly. Alpine.js is acceptable if needed for reactivity.

## V. Quality Checklist

- [ ] All 20+ SSE event types mapped to at least one widget
- [ ] Dashboard layout fits in a single viewport (no scrolling for primary view)
- [ ] DAG visualization handles at least 10 concurrent nodes
- [ ] Token burn-down chart updates in real-time without jank
- [ ] Budget gauges reflect all 3 tiers (query, session, monthly)
- [ ] Live event feed handles 200+ events without memory leaks
- [ ] MCP App config specifies all required permissions
- [ ] Data flow diagram accounts for SSE disconnection/reconnection
- [ ] Spec is implementable without external dependencies (vanilla JS)
- [ ] Estimated bundle size documented and under 50KB gzipped

## Output
- A spec document containing: widget inventory (name, data source, rendering approach, update mechanism), data flow diagram (SSE to EventSource to widget handlers), MCP App YAML config (permissions, app-only tools, sandbox model), and HTML/CSS/JS architecture notes.
- Estimated bundle size and browser compatibility documented.

## Examples
**Scenario 1:** "Design the first observability MCP App for the Gateway" → Full spec document produced with 6 widgets (DAG visualization, token burn-down, tool latency histogram, budget gauges, live event feed, trace waterfall), data flow diagram, MCP App config with 3 permissions, and vanilla JS architecture targeting < 50KB gzipped.
**Scenario 2:** "Add a new widget for memory hit rate" → Incremental spec update: new `memory_retrieved` event mapped to a rolling hit-rate counter widget in the left panel, event handler signature defined, state slice specified.

## Edge Cases
- If the SSE connection drops, the spec must account for degraded state: show a "disconnected" banner, keep last-known state visible, and auto-reconnect with exponential backoff — design the spec to handle this explicitly.
- The two-layer iframe sandbox means all data must pass through postMessage; no direct DOM access across the iframe boundary — widget handlers must be designed accordingly.

## Anti-Patterns
- Specifying a framework dependency (React, Vue) in a spec destined for a single-file MCP App — the Gateway serves a single HTML file directly; Alpine.js is the maximum acceptable dependency.
- Designing for polling when SSE provides push — only BudgetTracker state (which has no event emission) should be polled; everything else is push-driven.
