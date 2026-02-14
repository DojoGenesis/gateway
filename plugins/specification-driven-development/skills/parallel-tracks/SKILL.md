# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: parallel-tracks
description: Manage parallel development tracks for independent feature work
triggers:
  - "manage parallel tracks"
  - "coordinate parallel development"
  - "organize concurrent work"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
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

### Step 6: Integrate and Test

As tracks are completed, merge them into the main branch. After all tracks are merged, run a full suite of integration tests to verify that the defined integration points are working correctly. Fix any interface mismatches or regressions.

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

If you cannot answer "yes" to all of these, revisit the planning phase before proceeding.
