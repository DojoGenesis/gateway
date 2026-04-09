---
name: workflow-router
model: sonnet
description: "Routes user goals to the appropriate specialist agent workflow with resource allocation confirmation. Use when: the user wants to start a task but has not specified a workflow, asks 'how should I approach this?', or needs orchestration across research → plan → build → fix stages."
category: agent-orchestration
triggers:
  - "how should I approach this"
  - "route this to the right workflow"
  - "orchestrate across research plan build fix"
  - "start a task but not sure which workflow"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "meta_skill"
  - "file_system"

inputs:
  - name: goal
    type: string
    description: User's stated goal or task description — the router will classify and route it
    required: false
outputs:
  - name: routed_workflow
    type: string
    description: Confirmation of the agent(s) dispatched and their assigned tasks, with handoff paths
---

# Workflow Router

Routes user goals to the appropriate specialist agent workflow with explicit resource allocation and confirmation before dispatch.

## Description

Classifies an incoming goal into one of four categories (Research, Plan, Build, Fix), maps it to the appropriate specialist agent(s), confirms resource allocation with the user, and dispatches. Handles the Fix workflow as a two-stage process (investigate first, then implement). Preserves context across agent handoffs using shared handoff documents. This skill is the entry point for multi-agent orchestration when the user knows what they want to accomplish but not which agent or workflow to use.

## When to Use

- User wants to start a non-trivial task without specifying a workflow
- User asks "how should I approach this?" or "what's the best way to tackle X?"
- A task requires multiple sequential agents (Research → Plan → Build)
- You need explicit resource allocation agreement before spawning agents

## Workflow

### Step 1: Classify the Goal

If the goal is not provided or is ambiguous, ask:

```
What's your primary goal for this task?

- Research — Understand/explore something (unfamiliar code, libraries, concepts)
- Plan — Design/architect a solution (implementation plans, problem decomposition)
- Build — Implement/code something (new features, components, implementation from a plan)
- Fix — Debug/fix an issue (investigate and resolve bugs, failing tests)
```

If intent is clear from context (e.g., "this test is failing" → Fix), infer without asking.

### Step 2: Check for Existing Plans

```bash
ls thoughts/shared/plans/*.md 2>/dev/null || ls docs/plans/*.md 2>/dev/null
```

- **Build goal:** Ask if implementing an existing plan; load it to avoid duplication
- **Plan goal:** Surface existing plans to prevent redundant work
- **Research / Fix:** Proceed without plan check

### Step 3: Confirm Resource Allocation

Present options and confirm before spawning:

```
How would you like to allocate resources?

- Conservative — 1-2 agents, sequential (minimal context, simple tasks)
- Balanced (recommended) — appropriate agents for the task, some parallelism
- Aggressive — max parallel agents (time-critical tasks)
- Auto — system decides based on complexity
```

Default to **Balanced** for unspecified or Auto selection.

### Step 4: Map Goal to Agent

| Goal | Primary Agent Type | Description |
|------|-------------------|-------------|
| Research | general-purpose (research mode) | Multi-source research using web and documentation tools |
| Plan | Plan | Create phased implementation plans with architectural trade-offs |
| Build | general-purpose (implementation mode) | Coding tasks, feature implementation, component creation |
| Fix | general-purpose (debug mode) → then implementation | Investigate first, implement fix after diagnosis |

**Fix workflow (two stages):**
1. Spawn debug agent to investigate: produce diagnosis and recommended fix
2. If fix requires code changes, spawn implementation agent with the diagnosis as context

### Step 5: Show Execution Summary and Confirm

```
## Execution Summary

Goal: [Research / Plan / Build / Fix]
Resource Allocation: [Conservative / Balanced / Aggressive]
Agent(s) to dispatch: [agent descriptions]

What will happen:
- [Brief description of each agent's task]
- [Expected output / deliverable and where it will be written]

Proceed? [Yes / Adjust settings]
```

Wait for confirmation before dispatching. If "Adjust settings", return to the relevant step.

### Step 6: Dispatch Agents

#### Research

```
Dispatch: general-purpose agent (research mode)

Prompt:
  Research: [topic]
  Scope: [what to investigate — libraries, APIs, patterns, concepts]
  Output: Write findings to research/[topic-slug].md and summarize key points
```

#### Plan

```
Dispatch: Plan agent

Prompt:
  Create implementation plan for: [feature/task]
  Context: [relevant codebase context, constraints, goals]
  Output: Save plan to docs/plans/[plan-slug].md with phased breakdown and trade-offs
```

#### Build

If a plan exists, summarize risks before dispatching:
- Identify any HIGH severity blockers (missing dependencies, ambiguous interfaces)
- Surface them to the user before proceeding

```
Dispatch: general-purpose agent (implementation mode)

Prompt:
  Implement: [task]
  Plan: [path to plan, if applicable]
  Constraints: [language, framework, test requirements]
  Output: Working implementation with tests passing; write handoff to handoffs/[session]/
```

#### Fix (two stages)

```
Stage 1 — Dispatch: general-purpose agent (debug mode)

Prompt:
  Investigate: [issue description]
  Symptoms: [what's failing, error messages, test output]
  Output: Diagnosis document with root cause and recommended fix path

Stage 2 — (after diagnosis) Dispatch: general-purpose agent (implementation mode)

Prompt:
  Fix: [issue as diagnosed]
  Diagnosis: [diagnosis document path]
  Output: Working fix with tests passing
```

## Handoff Preservation

Each dispatched agent should write a handoff before completing:
```
handoffs/{session-id}/{agent-type}-{timestamp}.md
```

The handoff captures: what was accomplished, what was found, decisions made, open questions, and what the next agent needs to know.

## Output

A confirmation message showing:
- Which agent(s) were dispatched
- Their assigned prompts
- Expected output locations
- Status after dispatch completes
