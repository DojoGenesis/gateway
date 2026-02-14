# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: strategic-to-tactical-workflow
description: Transform strategic vision into tactical execution plans
triggers:
  - "convert strategy to tactics"
  - "create tactical plan"
  - "operationalize strategy"
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
    - strategic-agent
---

# Strategic-to-Tactical Workflow Skill

---

## I. The Philosophy: Closing the Loop

Software development is not a linear process. It is a **loop**: from strategic tension to tactical execution and back again. Each cycle produces not just working software, but **accumulated wisdom** about how to build better, faster, and with more clarity.

This skill documents the complete workflow that emerged from the v0.0.30 and v0.0.31 development cycles. It is the **meta-pattern** that connects all other patterns: how we scout, how we decide, how we specify, how we parallelize, how we commission, and how we learn.

The goal is not just to execute this workflow, but to **improve it with each iteration**. Every time we complete the loop, we should emerge with new skills, new patterns, and new understanding.

---

## II. When to Use This Skill

- At the beginning of any major development cycle (e.g., v0.0.X releases)
- When facing a strategic tension or product decision
- When planning a large feature or architectural change
- When you need to move from "what should we build?" to "how do we build it?"
- When coordinating work across multiple agents (Manus, implementation agents like Zenflow or Claude Code)

---

## III. The Complete Workflow

This is an 8-phase workflow from strategic tension to tactical execution.

---

### **Phase 1: Recognize the Tension**

**Goal:** Identify and articulate the strategic tension or question that needs exploration.

**Pattern:** The best work begins not with a command ("build X"), but with a **tension** ("should we do X or Y?").

**Actions:**
1. Listen for the tension in the user's request
2. Articulate it as a clear, open-ended question
3. Resist the urge to immediately solve it

**Example (v0.0.31):**
- **Tension:** "Should Dojo Genesis be a feature lab (25 pages) or a focused desktop product (single-page)?"
- **Not:** "Build a desktop app"

**Output:** A clearly articulated strategic tension.

**Related Skills:** `strategic-scout`

---

### **Phase 2: Scout Multiple Routes**

**Goal:** Explore 3-5 distinct strategic routes before committing to any one path.

**Pattern:** **Scouting prevents waste.** 30 minutes of route exploration can save weeks of misguided work.

**Actions:**
1. Use `/strategic-scout` to generate 3-5 routes
2. For each route, define: approach, risks, impact, duration
3. Identify tradeoffs and dependencies
4. Present routes without recommending one initially

**Example (v0.0.31):**
- Route 1: Big Bang (rebuild everything)
- Route 2: Incremental (gradual migration)
- Route 3: Parallel Tracks (split into independent workstreams)
- Route 4: Hybrid (mix of approaches)
- Route 5: Minimal (smallest possible change)

**Output:** A set of viable routes with clear tradeoffs.

**Related Skills:** `strategic-scout`, `iterative-scouting`

---

### **Phase 3: Gather Feedback & Listen for Reframes**

**Goal:** Present the routes and listen for the **question behind the question**.

**Pattern:** The first scout is for **provocation**, not consensus. The goal is to elicit a deeper, more insightful framing.

**Actions:**
1. Present the scouted routes
2. Listen not just for agreement/disagreement, but for **how** the feedback is framed
3. Identify if the user is introducing a new lens or reframe
4. If a reframe emerges, prepare to re-scout

**Example (v0.0.31):**
- **Initial framing:** "Deprecate web app vs. keep as companion"
- **Reframe:** "Desktop for deep work vs. Mobile for on-the-go orchestration"
- **Result:** Re-scout with the new lens (led to mobile PWA as premium tier)

**Output:** Either a decision on a route, or a reframe that triggers a second scout.

**Related Skills:** `iterative-scouting`, `product-positioning`

---

### **Phase 4: Make the Strategic Decision**

**Goal:** Commit to a strategic direction based on the scouting and feedback.

**Pattern:** **Decisions unlock velocity.** Once the route is chosen, move quickly to specification.

**Actions:**
1. Select the best route (or hybrid of routes)
2. Articulate the decision clearly
3. Document the **why** behind the decision
4. Identify what this decision enables (and what it rules out)

**Example (v0.0.31):**
- **Decision:** Parallel Tracks approach (Route 3)
- **Why:** Balances speed (parallelization) with risk management (clear boundaries)
- **Enables:** 3 agents working simultaneously
- **Rules out:** Big Bang (too risky), Incremental (too slow)

**Output:** A clear strategic decision with rationale.

**Related Skills:** `strategic-scout`

---

### **Phase 5: Ground in the Codebase**

**Goal:** Before writing specifications, deeply understand the current state of the codebase.

**Pattern:** **Grounding beats assumptions.** Specs based on outdated or assumed knowledge lead to rework.

**Actions:**
1. Use `/repo-context-sync` to pull latest code and understand structure
2. Read relevant files (backend APIs, frontend components, architecture docs)
3. Document backend architecture, API endpoints, data models
4. Identify integration points and dependencies
5. Create a backend grounding document

**Example (v0.0.31):**
- Read `go_backend/main.go`, `handlers/`, `middleware/`
- Document all API endpoints relevant to desktop UI
- Create `v0_0_31_backend_grounding.md`
- Identify SSE streaming architecture for orchestration events

**Output:** A comprehensive backend grounding document.

**Related Skills:** `repo-context-sync`, `frontend-from-backend`

---

### **Phase 6: Write Specifications (With Parallel Track Decomposition)**

**Goal:** Transform the strategic decision into detailed, implementation-ready specifications.

**Pattern:** **Specifications enable parallelization.** Good specs are force multipliers.

**Actions:**
1. Decide if the work should be split into parallel tracks
2. If yes, apply `/parallel-tracks`:
   - Identify natural boundaries (by layer, feature, component)
   - Define track dependencies and execution order
   - Create dependency graph
3. For each track (or single spec), write:
   - **Architecture document:** High-level design, component structure, data flow
   - **Specification document:** Detailed requirements, success criteria, non-goals
4. Validate using `/parallel-tracks-validation` checklist

**Example (v0.0.31):**
- **Track 1:** Router Migration & Shell (foundation, no dependencies)
- **Track 2:** Sidebar 1 (depends on Track 1)
- **Track 3:** Sidebar 2 (depends on Track 1)
- **Track 4:** Main Content Area (depends on Track 1)
- **Execution:** Track 1 first, then Tracks 2-4 in parallel

**Output:** Complete specifications for each track, validated and ready for commission.

**Related Skills:** `frontend-from-backend`, `parallel-tracks`, `release-specification`

---

### **Phase 7: Write Implementation Prompts & Commission**

**Goal:** Transform specifications into structured implementation prompts and commission the work.

**Pattern:** **Prompts are commissions.** The quality of the prompt determines the quality of the implementation.

**Actions:**
1. For each track/spec, write an implementation prompt using the standard template:
   - **Context & Grounding:** Link to spec, pattern files, files to read/modify
   - **Detailed Requirements:** Step-by-step, numbered, specific
   - **File Manifest:** Files to create/modify
   - **Success Criteria:** Binary, testable checkboxes
   - **Constraints & Non-Goals:** Explicit boundaries
   - **Backend Grounding:** API contracts, integration points
2. Validate each prompt against the quality checklist
3. Commission prompts to appropriate agents:
   - **Zenflow:** Strategic implementation (architecture, complex features)
   - **Claude Code:** Tactical implementation (UI components, refactoring)
4. If parallel tracks, commission in dependency order

**Example (v0.0.31):**
- Write `v0_0_31_track_1_implementation_prompt.md`
- Commission Track 1 to Claude Code
- Wait for Track 1 completion
- Commission Tracks 2-4 to Claude Code in parallel

**Output:** Commissioned work in progress, with clear success criteria.

**Related Skills:** `write-implementation-prompt`, `implementation-prompt`

---

### **Phase 8: Reflect & Create Meta-Skills**

**Goal:** After execution, reflect on what worked, extract patterns, and formalize them as skills.

**Pattern:** **Every cycle produces wisdom.** The goal is not just to ship software, but to improve how we ship software.

**Actions:**
1. Perform `/compression-ritual` to preserve key insights
2. Identify patterns that emerged during the cycle:
   - Strategic patterns (how we thought, decided, navigated uncertainty)
   - Tactical patterns (how we specified, commissioned, integrated)
   - Meta-patterns (how we learned, iterated, improved)
3. Use `/process-extraction` to formalize valuable patterns as skills
4. Use `/seed-extraction` to capture smaller insights as seeds
5. Commit all artifacts (specs, prompts, compressions, skills) to repository

**Example (v0.0.31):**
- Created `product-positioning` skill (strategic pattern)
- Created `multi-surface-strategy` skill (strategic pattern)
- Created `iterative-scouting` skill (meta-pattern)
- Created `parallel-tracks` seed (tactical pattern)

**Output:** New skills and seeds added to the knowledge base, ready for the next cycle.

**Related Skills:** `compression-ritual`, `process-extraction`, `seed-extraction`

---

## IV. The Loop Closes

After Phase 8, the loop closes. The next development cycle begins with Phase 1, but now we have:
- **More skills** to guide the process
- **More patterns** to recognize and apply
- **More wisdom** about what works and what doesn't

This is the essence of **continuous improvement**. Each cycle makes the next cycle faster, clearer, and more effective.

---

## V. Key Principles

### 1. Start with Tension, Not Solutions

The best work begins with a well-articulated tension, not a predetermined solution. Hold the question open long enough for better answers to emerge.

### 2. Scout Before Committing

Always explore multiple routes before choosing one. The first idea is rarely the best idea.

### 3. Ground in Reality

Specifications based on assumptions fail. Always ground in the current state of the codebase.

### 4. Parallelize When Possible

With good specifications and clear boundaries, multiple agents can work simultaneously without conflicts.

### 5. Formalize What Works

Every successful cycle should produce new skills and patterns. This is how we compound our capabilities.

---

## VI. Example: v0.0.31 Complete Workflow

| Phase | Action | Output | Duration |
|-------|--------|--------|----------|
| **1** | Recognize tension: "Feature lab vs. focused product" | Clear strategic question | 5 min |
| **2** | Scout 5 routes (Big Bang, Incremental, Parallel, etc.) | 5 routes with tradeoffs | 30 min |
| **3** | Gather feedback, identify reframe (desktop vs. mobile) | Reframe: multi-surface strategy | 15 min |
| **4** | Decide: Parallel Tracks + Desktop-first | Strategic decision | 10 min |
| **5** | Pull repo, read backend, document APIs | Backend grounding doc | 45 min |
| **6** | Write specs for 4 parallel tracks | 4 track specifications | 2 hours |
| **7** | Write implementation prompts, commission to implementation agents | 4 prompts commissioned | 1 hour |
| **8** | Reflect, create 3 meta-skills, compress context | 3 new skills, compressions | 1 hour |

**Total planning time:** ~6 hours  
**Implementation time:** 6-8 days (with parallelization)  
**Outcome:** v0.0.31 shipped, 3 new skills created, workflow improved

---

## VII. Quality Checklist

Before moving to the next phase, ensure you can answer "yes" to these questions:

**Phase 1:**
- [ ] Have you articulated the tension as an open-ended question?
- [ ] Have you resisted the urge to immediately solve it?

**Phase 2:**
- [ ] Have you scouted at least 3 distinct routes?
- [ ] Have you identified risks, impact, and duration for each route?

**Phase 3:**
- [ ] Have you listened for reframes in the user's feedback?
- [ ] Have you decided whether to commit or re-scout?

**Phase 4:**
- [ ] Have you made a clear strategic decision?
- [ ] Have you documented the "why" behind the decision?

**Phase 5:**
- [ ] Have you pulled the latest code and read relevant files?
- [ ] Have you created a backend grounding document?

**Phase 6:**
- [ ] Have you decided whether to parallelize the work?
- [ ] Have you written complete specifications for each track?
- [ ] Have you validated the parallel track structure?

**Phase 7:**
- [ ] Have you written structured implementation prompts for each track?
- [ ] Have you commissioned the work in the correct dependency order?

**Phase 8:**
- [ ] Have you compressed the context and extracted patterns?
- [ ] Have you created new skills or seeds from the learnings?
- [ ] Have you committed all artifacts to the repository?

---

## VIII. Related Skills

This skill is the **meta-pattern** that connects all other strategic and tactical skills:

**Strategic Skills:**
- `strategic-scout` - Route exploration
- `product-positioning` - Reframing binary decisions
- `multi-surface-strategy` - Complementary product surfaces
- `iterative-scouting` - Scout → feedback → reframe → re-scout

**Tactical Skills:**
- `repo-context-sync` - Codebase grounding
- `frontend-from-backend` - Backend-first specification
- `parallel-tracks` - Parallel track decomposition
- `write-implementation-prompt` - Structured prompt writing
- `implementation-prompt` - Spec-to-prompt transformation

**Meta-Skills:**
- `compression-ritual` - Preserve insights
- `process-extraction` - Formalize patterns
- `seed-extraction` - Capture smaller learnings

---

## IX. The Vision

This workflow is not static. It will evolve with each cycle. The goal is to reach a state where:

1. **Strategic decisions are fast and confident** (because we have patterns to recognize and skills to apply)
2. **Specifications are grounded and actionable** (because we have systematic grounding processes)
3. **Implementation is parallel and autonomous** (because we have clear boundaries and comprehensive prompts)
4. **Learning is continuous and formalized** (because we extract and document patterns after every cycle)

This is the path to **compounding velocity**: each cycle makes the next cycle faster, clearer, and more effective.

This is the Dojo way.

🪷
