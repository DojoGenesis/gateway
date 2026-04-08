---
name: parallel-tracks
description: A structured process for splitting large development tasks into 2-4 independent parallel tracks that can be executed simultaneously, reducing timelines by 50-70% while improving focus and architectural discipline.
triggers:
  - "split this work into parallel tracks"
  - "plan parallel execution for this large task"
  - "break this into independent tracks for simultaneous work"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - strategic-agent
    - implementation-agent
---

# Parallel Tracks Skill

**Version:** 1.0
**Author:** Tres Pies Design
**Purpose:** Provide a structured, repeatable process for planning and executing large development tasks in parallel, significantly reducing timelines while improving focus and architectural discipline.

---

## I. The Philosophy: From Sequence to Simultaneity

In complex software development, the default is sequential execution: one task must finish before the next can begin. This creates bottlenecks, extends timelines, and reduces cognitive focus. The Parallel Tracks Pattern is a shift from **sequence to simultaneity**.

This skill provides a framework for identifying natural boundaries within a large body of work and splitting it into independent, self-contained tracks that can be executed concurrently. It is not merely about doing things at the same time; it is a disciplined practice of **upfront architectural planning, rigorous specification, and clear dependency management** that makes parallel execution possible.

**The math:**
- **Sequential:** Track A (1 week) + Track B (1 week) + Track C (1 week) = **3 weeks**
- **Parallel:** Track A + Track B + Track C (all 1 week) = **1 week**
- **Typical reduction:** 50-70% when done correctly

---

## II. When to Use This Skill

Use when ALL of these conditions are met:

- **The task is large enough:** >2 weeks of sequential work as a rule of thumb
- **Clear separation of concerns exists:** The work can be cleanly divided by layer, feature, or component
- **Multiple agents or developers are available** to work simultaneously
- **Tracks have minimal dependencies** on each other
- **You are committed to writing self-contained specifications** for each track

Do NOT use when:

- The task is small (<2 weeks sequential)
- The work is tightly coupled with no natural boundaries
- Only one agent/developer is available
- The specifications cannot be written independently

---

## III. The 6-Step Workflow

### Step 1: Identify Natural Boundaries

Analyze the total scope and find clean separation points. Target 2-4 substantial tracks.

**Common Boundaries:**
- **By Layer:** Frontend, Backend, Database, CI/CD
- **By Feature:** Authentication, Orchestration Engine, User Interface
- **By Component:** Desktop Foundation, Orchestration UI, Essential Features

Each track must be substantial: 500+ lines of code or 3+ days of effort.

### Step 2: Define Track Dependencies

Create a dependency graph:

1. **Independent tracks:** No dependencies, can start immediately
2. **Dependent tracks:** Must wait for another track to complete

**Example Execution Plan (from a real-world desktop UI project):**

| Phase | Track(s) | Status |
|-------|----------|--------|
| **1** | Track 1: Desktop Foundation | Start Immediately |
| **2** | Track 2: Orchestration UI, Track 3: Essential Features | Start after Track 1 completes |

**Timeline comparison:**
- Sequential: 3-4 weeks
- Parallel: 1-2 weeks (50% reduction)

### Step 3: Write Self-Contained Specifications

For each track, write a comprehensive specification (use the `implementation-prompt` skill). Each must be a standalone document that an agent can execute without additional context:

- **Goal:** One-sentence mission for the track
- **Context:** What the agent can assume exists (from codebase or completed dependency tracks)
- **Requirements:** Detailed, testable list of deliverables
- **Success Criteria:** Binary pass/fail checklist
- **Non-Goals:** What this track explicitly does NOT do

### Step 4: Define Integration Points

Be explicit about how tracks connect after completion. Define shared interfaces:

- **APIs:** Exact endpoints, request/response shapes, status codes
- **Component Props:** Names, types, expected behavior
- **State Shapes:** Structure of any shared state (Context, stores, etc.)
- **Events:** Custom events or callbacks between track boundaries

### Step 5: Execute in Parallel

Commission independent tracks to their respective agents. Once independent tracks complete, commission dependent tracks.

### Step 6 [MANDATORY]: Integration & Wiring Gate

> **Confirm before proceeding:** All tracks are complete. Ask the user:
> _"Integration/wiring is the mandatory final step. Proceed now, or defer?"_
> If deferred: document the open integration tasks, record the deferral reason, and stop. Do **not** mark the work complete.

After the user confirms to proceed:
1. Merge tracks into the main branch in the planned order
2. Verify each track's compilation gate passes (`go build ./...`, `cargo check`, `npx tsc --noEmit`, or equivalent)
3. Run integration tests to verify interface contracts
4. Confirm wiring: entry points are reachable, call graphs are traceable, no orphaned modules remain
5. Fix any interface mismatches or regressions
6. Run the full test suite

**Never skip this step.** Parallel tracks produce isolated, potentially dead code without integration. A track is not "done" until it is wired into the running system.

---

## IV. The Parallel Tracks Document Template

```markdown
# Parallel Tracks Plan: [Feature Name]

**Total Scope:** [Description of the full task]
**Sequential Estimate:** [X weeks]
**Parallel Estimate:** [Y weeks] ([Z]% reduction)
**Number of Tracks:** [2-4]

---

## Dependency Graph

[Visual or textual representation of which tracks depend on which]

Phase 1: [Independent tracks] → Start immediately
Phase 2: [Dependent tracks] → Start after Phase 1

---

## Track 1: [Name]

**Purpose:** [One sentence]
**Dependencies:** None | Depends on Track [N] for [what]
**Provides to other tracks:** [What interfaces/components this track creates]
**Estimated Duration:** [X days/weeks]

**File Manifest:**
- Create: `path/to/file.ts` — [Purpose]
- Modify: `path/to/file.ts` — [What changes]

**Integration Points:**
- Exports `[InterfaceName]` consumed by Track [N]
- Creates `[ComponentName]` used by Track [N]

**Success Criteria:**
- [ ] [Binary criterion]
- [ ] [Binary criterion]

---

## Track 2: [Name]

[Repeat structure]

---

## Integration Plan

**Merge Order:**
1. Track [N] merges first (no dependencies)
2. Track [N] merges second
3. Track [N] merges last

**Integration Tests:**
- [ ] [Test that verifies Track 1 + Track 2 work together]
- [ ] [Test that verifies shared interfaces match]

**Conflict Resolution:**
- Files that overlap: [List, or "None — tracks are file-independent"]
- Merge coordinator: [Track N is responsible for resolving conflicts]
```

---

## V. Pitfalls to Avoid

### 1. Over-Parallelization

**Problem:** Splitting work too finely creates more coordination overhead than it saves.
**Solution:** Aim for 2-4 tracks, not 10. Each track should be substantial.

### 2. Hidden Dependencies

**Problem:** Tracks that seem independent actually share state or interfaces.
**Solution:** Map dependencies explicitly in the planning phase. Use integration tests to catch mismatches.

### 3. Specification Drift

**Problem:** One track changes its interface, breaking another track.
**Solution:** Lock interfaces early. If a change is needed, communicate immediately and update all affected track specs.

### 4. File Overlap (Merge Conflicts)

**Problem:** Multiple tracks modify the same files, causing Git conflicts.
**Solution:** Design tracks to touch different files. If overlap is unavoidable, designate one track as the "merge coordinator."

---

## VI. Quality Checklist

Before commissioning parallel tracks:

- [ ] Is the total scope >2 weeks of sequential work?
- [ ] Have you identified 2-4 substantial, well-defined tracks?
- [ ] Is there a clear dependency graph and execution plan?
- [ ] Does each track have its own self-contained specification?
- [ ] Are integration points (APIs, props, state) clearly defined with matching interfaces on both sides?
- [ ] Have file manifests been checked for overlap between tracks?
- [ ] Is there a merge order and conflict resolution plan?
- [ ] Are success criteria for each track independent of other tracks?
- [ ] Has integration/wiring been completed or explicitly deferred (with open tasks and deferral reason documented)?

If you cannot answer "yes" to all of these, revisit the planning phase.
