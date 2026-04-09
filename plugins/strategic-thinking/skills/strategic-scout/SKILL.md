---
name: strategic-scout
model: opus
description: Produces a scout document containing a stated tension, 3-5 distinct routes with tradeoffs, and a recommended direction with rationale — the first artifact in a four-step pipeline (scout → spec → prompts → commission). Use when: "we don't know which direction to take", "scout our options before we commit", "what are the tradeoffs here", "we're stuck between competing priorities", "I need routes before writing a spec".
category: strategic-thinking

inputs:
  - name: tension
    type: string
    description: The strategic tension or decision point to scout routes for
    required: true
outputs:
  - name: scout_document
    type: ref
    format: cas-ref
    description: Scout document with stated tension, 3-5 distinct routes with tradeoffs, and recommended direction with rationale
---

# Strategic Scout Skill

**Version:** 2.1
**Created:** 2026-02-07
**Author:** Manus AI
**Purpose:** To provide a structured, repeatable process for navigating strategic uncertainty, exploring multiple possible futures, and aligning on a clear, actionable plan.

---

## I. The Philosophy: From Problem to Possibility

Strategic thinking is not about finding the right answer to a problem; it is about exploring the landscape of possibility that a tension reveals. This skill transforms the act of planning from a linear process of problem-solving to a creative process of possibility-seeking.

By beginning with a tension, scouting multiple routes, and continuously aligning on a shared vision, we can navigate uncertainty with clarity and confidence, and arrive at solutions that are both innovative and robust.

---

## II. When to Use This Skill

-   **At the beginning of a new project or major release.**
-   **When facing a significant strategic decision with no clear answer.**
-   **When a project feels stuck or lacks a clear direction.**
-   **When there are multiple competing priorities or stakeholder interests.**

---

## III. The Workflow

This is a 4-step workflow for strategic scouting and decision-making.

### Step 1: Identify the Tension

**Goal:** To frame the strategic challenge as a tension to be held, not a problem to be solved.

1.  **Articulate the Tension:** Clearly state the core strategic challenge as a tension between two competing ideas (e.g., "feature lab vs. focused product").
2.  **Hold the Question Open:** Resist the urge to immediately choose a side or find a solution. The goal is to create a space for exploration.

### Step 2: Scout Multiple Routes

**Goal:** To map the landscape of possibility by exploring multiple distinct paths forward.

1.  **Generate 3-5 Routes:** Use Scout mode to generate a diverse set of potential routes.
2.  **Define Tradeoffs:** For each route, clearly articulate the risk profile, potential impact, and estimated duration.
3.  **Present the Options:** Present the routes and their tradeoffs to the user for review and discussion.

### Step 3: Synthesize and Refine

**Goal:** To create a hybrid approach that combines the best aspects of multiple routes.

1.  **Gather Feedback:** Actively listen to the user's feedback on the scouted routes.
2.  **Look for Connections:** Identify opportunities to combine elements from different routes into a more robust solution.
3.  **Propose a Hybrid Plan:** Present a new, synthesized plan that incorporates the user's feedback and the best aspects of the scouted routes.

### Step 4: Align on Vision

**Goal:** To ensure that the plan is fully aligned with the user's true strategic vision.

1.  **Check for Alignment:** Continuously ask clarifying questions to ensure that the plan is meeting the user's underlying goals.
2.  **Be Prepared to Reframe:** If the user's feedback reveals a deeper or different vision, be prepared to reframe the entire plan.
3.  **Confirm the Vision:** Before moving to execution, get explicit confirmation from the user that the plan is fully aligned with their vision.

---

## IV. Best Practices

-   **Begin with a Tension, Not a Solution:** The quality of your strategic thinking is determined by the quality of your questions.
-   **Scouting is an Act of Humility:** The first idea is rarely the best idea. Be patient and explore multiple possibilities.
-   **Synthesis is a Creative Act:** The best solutions often come from combining existing ideas in new ways.
-   **Alignment is an Ongoing Process:** Don't assume you understand the user's vision. Continuously check for alignment.

---

## VII. The Full Pipeline: Scout → Spec → Prompts → Commission

The strategic scout is not standalone — it is phase 1 of a 4-step workflow. Each step produces a persistent artifact:

### Phase 1: Scout (this skill)
- **Output:** A scout document with tension, 3-5 routes, tradeoffs, and selected direction with rationale
- **Artifact:** `thinking/[topic]_strategic_scout.md`

### Phase 2: Specify
- **Input:** Scout decisions + codebase audit
- **Tool:** `release-specification` skill
- **Output:** A production-ready specification grounded in scout decisions AND measured codebase state
- **Artifact:** `docs/vX.X.X/[release]_specification.md`

### Phase 3: Prompts
- **Input:** Specification sections
- **Tool:** `implementation-prompt` or `zenflow-prompt-writer` skill
- **Output:** Self-contained implementation prompts, one per parallel track
- **Artifact:** `docs/vX.X.X/prompts/track_[N]_prompt.md`

### Phase 4: Commission
- **Input:** Implementation prompts + actual codebase
- **Tool:** `pre-implementation-checklist` skill (with Track 0)
- **Output:** Verified prompts with Track 0 remediation complete, then parallel execution
- **Gate:** Track 0 must pass before parallel tracks begin

The scout is the "why." The spec is the "what." The prompts are the "how." Track 0 is the "verify."

**Key triggers:** "scout to spec", "spec pipeline", "full workflow from tension to implementation"

---

## VI. Quality Checklist

Before moving to execution, ensure you can answer "yes" to all of the following questions:

-   [ ] Have you clearly articulated the core strategic tension?
-   [ ] Have you scouted at least 3-5 distinct routes with clear tradeoffs?
-   [ ] Have you looked for opportunities to synthesize the best aspects of multiple routes?
-   [ ] Have you gotten explicit confirmation from the user that the plan is fully aligned with their vision?

---

## VIII. Perspectives from Past Scouting Sessions

### Perspective 1: The Great Simplification (v0.0.31)

-   **Tension:** Additive complexity vs. subtractive simplification.
-   **Insight:** A lean, focused product is often more powerful than a feature-rich but bloated one.
-   **Lesson:** Don't be afraid to pivot and start fresh if the current path is leading to complexity.

### Perspective 2: Orchestration Visibility (v0.0.31)

-   **Tension:** Backend power vs. frontend visibility.
-   **Insight:** A powerful feature is useless if the user can't see it or understand it.
-   **Lesson:** Make the invisible visible. The user experience of a feature is as important as the feature itself.

### Perspective 3: Parallel Tracks (v0.0.30)

-   **Tension:** Speed vs. quality.
-   **Insight:** With clear specifications and separation of concerns, you can have both.
-   **Lesson:** Good governance multiplies velocity. Invest in planning and specification to enable parallel execution.

---

## Output

- Scout document saved to `thinking/[topic]_strategic_scout.md` containing: tension statement, 3-5 routes (each with approach, risks, estimated impact, and duration), and a recommended direction with explicit rationale
- If the user is proceeding to Phase 2, a handoff note naming the spec artifact and the relevant skill to invoke next

## Examples

**Scenario 1:** "Should we build a mobile app or focus on improving the desktop experience?" → Scout document with routes including (1) mobile-first PWA, (2) desktop-only deepening, (3) parallel mobile + desktop, (4) web-responsive as mobile substitute — each with risk profile and timeline, plus a recommendation with stated tradeoffs.

**Scenario 2:** "We need to decide whether to open-source our core library." → Scout document exploring (1) full open-source with community model, (2) open-core (free tier + paid extensions), (3) source-available license, (4) stay closed — with business model implications and competitive risk for each. Recommended direction identifies which option fits the current revenue stage.

## Edge Cases

- If the user already knows their direction and just wants help specifying it, skip scouting and route directly to `release-specification` or `implementation-prompt` — running a full scout on a decided question wastes time and can introduce unnecessary doubt.
- If the tension involves a third-party constraint (regulatory requirement, contractual obligation, hard deadline), surface that constraint upfront and remove any routes that violate it before presenting options.
- If the tension is primarily interpersonal or organizational (e.g., disagreement between two team members), note that this skill addresses the strategic dimension only and flag that alignment work is a separate step.

## Anti-Patterns

- Presenting routes that are not genuinely distinct — if Routes 2 and 3 differ only in implementation detail rather than strategic approach, collapse them into one route.
- Recommending a direction before presenting all routes — the recommendation must come after the options are on the table, not before, to avoid anchoring the user prematurely.
- Scouting without naming the tension explicitly — a scout that jumps straight to routes without a stated tension gives users no way to evaluate whether the routes actually address their situation.
- Treating scout output as final — the scout document is a starting point for conversation, not a decision. Always check for reframes before moving to Phase 2.
