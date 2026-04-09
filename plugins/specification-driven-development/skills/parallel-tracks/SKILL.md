---
name: parallel-tracks
model: sonnet
description: Produces a parallel-tracks execution plan with phased structure, self-contained track specifications, integration contracts, and a mandatory wiring gate. Use when 'split into parallel tracks', 'organize these tracks into phases', or 'plan parallel development for this release'.
category: specification-driven-development
triggers:
  - "split into parallel tracks"
  - "organize these tracks into phases"
  - "plan parallel development for this release"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec
    type: string
    description: The release specification or feature set to split into parallel tracks
    required: true
outputs:
  - name: parallel_tracks_plan
    type: ref
    format: cas-ref
    description: Parallel-tracks execution plan with phased structure, self-contained track specs, integration contracts, and mandatory wiring gate
---

# Parallel Tracks Pattern Skill

**Version:** 1.1  
**Created:** February 7, 2026  
**Author:** Manus AI  
**Purpose:** To provide a structured, repeatable process for planning and executing large development tasks in parallel, significantly reducing timelines while improving focus and architectural discipline.

---

## I. The Philosophy: From Sequence to Simultaneity

In complex software development, the default is often sequential execution: one task must finish before the next can begin. This creates bottlenecks, extends timelines, and reduces the cognitive focus of the development team. The Parallel Tracks Pattern is a shift in mindset from **sequence to simultaneity**.

This skill provides a framework for identifying natural boundaries within a large body of work and splitting it into independent, self-contained tracks that can be executed concurrently. It is not merely about doing things at the same time; it is a disciplined practice of **upfront architectural planning, rigorous specification, and clear dependency management** that makes parallel execution possible. By investing in this discipline, we transform development from a linear relay race into a coordinated, multi-pronged advance, multiplying velocity without sacrificing quality.

---

## II. When to Use This Skill

This skill is most effective when planning a major release, a new feature with multiple components, or any large-scale development effort. Use this skill when the following conditions are met:

-   **The task is large enough to benefit from parallelization.** A good rule of thumb is any work estimated to take more than two weeks if executed sequentially.
-   **Clear separation of concerns exists.** The work can be cleanly divided by layer (frontend vs. backend), by feature (auth vs. orchestration), or by component (header vs. chat area).
-   **Multiple agents or developers are available** to work on the tracks simultaneously.
-   **The tracks have minimal dependencies on each other.** While some dependencies are expected, the work should not be so tightly coupled that parallel execution is impossible.
-   **You are committed to writing clear, self-contained specifications** for each track.

---

## III. The Workflow

### Step 1: Identify Natural Boundaries

Begin by analyzing the total scope of work. Look for clean separation points that allow you to divide the project into 2-4 substantial tracks. Avoid over-parallelization; each track should represent a meaningful chunk of work (e.g., 500+ lines of code or 3+ days of effort).

**Common Boundaries:**
-   **By Layer:** `frontend`, `backend`, `database`, `ci-cd`
-   **By Feature:** `authentication`, `orchestration-engine`, `user-interface`
-   **By Component:** `desktop-foundation`, `orchestration-ui`, `essential-features`

### Step 2: Define Track Dependencies

Create a dependency graph to visualize the relationships between the tracks. This will determine the execution order.

1.  **Identify Independent Tracks:** These have no dependencies and can begin immediately.
2.  **Identify Dependent Tracks:** These must wait for another track to be completed.

**Example Execution Plan (from v0.0.31):**

| Phase | Track(s) | Status |
| :--- | :--- | :--- |
| **1** | Track 1: Desktop Foundation | Start Immediately |
| **2** | Track 2: Orchestration UI<br>Track 3: Essential Features | Start after Track 1 is complete |

### Step 2.5: Organize Tracks into Phases

When tracks have dependencies, organize them into execution phases:

**Phase 0 — Foundation (Sequential):**
Track 0 remediation and shared infrastructure. This phase runs BEFORE any parallel work. Use it to:
- Close gaps identified by the pre-implementation checklist
- Create shared types, interfaces, or utilities that multiple tracks need
- Set up test infrastructure or CI/CD changes

**Phase 1 — Parallel Execution (Independent):**
Tracks with no cross-dependencies execute simultaneously. Each track has its own self-contained specification. No track needs output from another Phase 1 track.

**Phase 2 — Integration (If Needed):**
Tracks that depend on Phase 1 output. These handle cross-cutting concerns like integration testing, shared state connections, or UI composition that combines output from multiple Phase 1 tracks.

**Evidence:** v0.0.35 used 4 tracks in 2 phases. v0.2.2 used Track 0 as Phase 0 + 3 parallel tracks as Phase 1. Timeline reduction: 40-50% vs sequential.

**Key triggers:** "phased parallelism", "dependent tracks", "Phase 0 foundation", "organize into phases"

### Step 3: Write Self-Contained Specifications

For each track, write a comprehensive specification using the `/write-implementation-prompt` or similar skill. Each specification must be a standalone document that an agent can execute without needing additional context. It must include:

-   **Goal:** A clear, one-sentence mission for the track.
-   **Context:** What the agent can assume exists (from the current codebase or from completed dependency tracks).
-   **Requirements:** A detailed, testable list of deliverables.
-   **Success Criteria:** A checklist to verify completion.
-   **Non-Goals:** What the track is explicitly *not* responsible for, to prevent scope creep.

### Step 4: Define Integration Points

In the specifications, be explicit about how the tracks will connect after they are complete. Define the shared interfaces:

-   **APIs:** The exact endpoints, request/response shapes, and status codes.
-   **Component Props:** The names, types, and expected behavior of props passed between components from different tracks.
-   **State Shapes:** The structure of any shared state (e.g., in a React Context or Zustand store).

### Step 5: Execute in Parallel

Commission the independent tracks to their respective agents or developers. Once they are complete, commission the dependent tracks.

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

## IV. Best Practices

-   **Aim for 2-4 Tracks:** Over-parallelizing a task can create more coordination overhead than it saves. Find the right balance.
-   **Lock Interfaces Early:** Once specifications are written, treat the defined interfaces as a contract. If a change is needed, it must be communicated to all dependent tracks immediately.
-   **Design for Minimal File Overlap:** To reduce Git merge conflicts, design tracks to operate on different sets of files where possible. If overlap is unavoidable, designate one track as the "merge coordinator" responsible for resolving conflicts.
-   **Use a Dependency Graph:** A simple visual diagram can clarify the execution order and prevent misunderstandings.

---

## V. Quality Checklist

Before commissioning the tracks, ensure you can answer "yes" to all of the following questions:

1.  [ ] Is the total scope of work large enough to justify parallelization?
2.  [ ] Have you identified 2-4 substantial, well-defined tracks?
3.  [ ] Have you created a clear dependency graph and execution plan?
4.  [ ] Does each track have its own comprehensive, self-contained specification?
5.  [ ] Are the integration points (APIs, props, state) clearly defined in the specifications?
6.  [ ] Have you considered potential merge conflicts and planned for them?
7.  [ ] Has integration/wiring been completed or explicitly deferred (with open tasks and deferral reason documented)?

If you cannot answer "yes" to all of these, revisit the planning phase before proceeding.

---

## Output

- A parallel-tracks execution plan document (markdown), typically saved to `docs/vX.X.X/parallel_tracks_plan.md`
- One self-contained track specification per track (markdown files), each executable independently
- A dependency graph or phase table showing execution order
- An integration/wiring checklist confirming all tracks are merged and wired before the work is declared done

## Examples

**Scenario 1:** "We have a new desktop shell with three major pieces — scaffold, backend, and UI. How do we parallelize this?" → A three-track plan with Phase 0 (scaffold) running first, then Phase 1 (backend + UI in parallel), plus integration contracts defining the API the UI consumes from the backend.

**Scenario 2:** "I have four specs ready. Let's organize these tracks and get agents running." → A phased execution plan enumerating Track 0 remediation (if needed), Tracks 1-4 with dependency arrows, and a mandatory integration gate prompt for after all tracks complete.

## Edge Cases

- When the codebase has no existing patterns to ground track specs in, run `codebase-audit-grounding` first before writing specifications
- When fewer than two agents are available, collapse parallel tracks into a sequential plan rather than forcing parallelism that will block on a single executor
- When a track produces a shared type used by three or more other tracks, treat that track as Phase 0 regardless of its apparent independence

## Anti-Patterns

- Writing track specs that assume the output of sibling tracks — if Track B needs Track A's API, Track B is dependent and must run in a later phase, not in parallel
- Skipping the integration gate and calling tracks "done" once they compile individually — isolated tracks that are never wired produce dead code
- Over-parallelizing: creating five or more tracks adds coordination overhead that cancels the speed benefit; aim for two to four
