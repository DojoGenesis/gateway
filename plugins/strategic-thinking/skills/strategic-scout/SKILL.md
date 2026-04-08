---
name: strategic-scout
description: >
  A structured process for exploring strategic tensions, scouting multiple routes,
  and aligning on a clear vision before committing to a plan. Use when facing a
  significant strategic decision, when a project feels stuck, when there are competing
  priorities, or at the beginning of a new project or major release. Produces a
  strategic decision document with tension statement, routes, synthesis, and next steps.
triggers:
  - "scout the strategic landscape for this decision"
  - "explore routes before we commit to a plan"
  - "we are stuck, run a strategic scout"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - strategic-agent
---

# Strategic Scout Skill

**Version:** 1.0
**Purpose:** To provide a structured, repeatable process for navigating strategic uncertainty, exploring multiple possible futures, and aligning on a clear, actionable direction.

---

## I. The Philosophy: From Problem to Possibility

Strategic thinking is not about finding the right answer to a problem; it is about exploring the landscape of possibility that a tension reveals.

Three principles guide this skill:

1. **Begin with a tension, not a solution.** The quality of your strategic thinking is determined by the quality of your questions. A tension held open reveals more than a problem quickly solved.

2. **Scouting is an act of humility.** The first idea is rarely the best idea. By generating multiple routes, you resist premature commitment and let the landscape teach you what's possible.

3. **Synthesis is a creative act.** The best solutions often come from combining existing ideas in new ways. Routes are raw material for synthesis, not a menu to pick from.

4. **Alignment is an ongoing process.** Don't assume you understand the user's vision. Continuously check for alignment. The goal is a shared understanding, not a delivered answer.

---

## II. When to Use This Skill

- At the beginning of a new project or major release
- When facing a significant strategic decision with no clear answer
- When a project feels stuck or lacks a clear direction
- When there are multiple competing priorities or stakeholder interests
- When a binary decision feels limiting (pair with `product-positioning`)
- When moving from "what should we build?" to exploring options before committing

**When NOT to use:**
- When the decision is purely tactical and well-understood
- When the user has already committed to a direction and needs execution help
- When the question can be answered with a quick lookup or factual analysis

---

## III. The Workflow

This is a 4-step workflow for strategic scouting and decision-making.

### Step 1: Identify the Tension

**Goal:** Frame the strategic challenge as a tension to be held, not a problem to be solved.

**Actions:**
1. Ask the user to describe the tension. Accept any form: a binary decision ("should we X or Y?"), a strategic question ("how should we approach Z?"), or a feeling of being stuck ("something isn't right about our Q direction").
2. Help articulate the tension as a clean statement: "The tension is between [A] and [B]."
3. Hold the question open. Resist the urge to immediately choose a side or propose a solution.

**Output:** A clearly articulated tension statement.

**Key insight:** The way you frame the tension determines the quality of the routes you'll discover. Spend time here.

### Step 2: Scout Multiple Routes

**Goal:** Map the landscape of possibility by exploring multiple distinct paths forward.

**Actions:**
1. Generate 3-5 distinct routes. Each route is a complete strategic direction, not a variation on the same theme.
2. For each route, define:
   - **Name:** A memorable, descriptive title
   - **Thesis:** One sentence explaining the core idea
   - **Tradeoffs:** Risk profile, potential impact, estimated duration
   - **Optimizes for:** What this route prioritizes
   - **Sacrifices:** What this route gives up
3. Present routes as a table. Do NOT recommend one. Ask the user which routes resonate and which feel wrong.

**Output:** A route table with 3-5 distinct options and their tradeoffs.

**Key insight:** Routes should be genuinely different directions, not minor variations. If routes feel too similar, you haven't explored far enough.

### Step 3: Synthesize and Refine

**Goal:** Create a hybrid approach that combines the best aspects of what resonated.

**Actions:**
1. Gather user feedback on the routes. Listen for:
   - Which routes resonate and why
   - Which routes feel wrong and why
   - Whether any route sparked a new idea or reframe
2. Look for connections between routes that resonated. Where do they overlap? What do they share?
3. Propose a synthesized direction that incorporates the user's feedback. Name the reframe -- give the new direction a clear identity.

**Output:** A synthesized direction with a name, rationale, and clear differentiation from the original routes.

**Key insight:** The synthesis is often better than any individual route. The reframe that emerges from the conversation is the real prize.

### Step 4: Align on Vision

**Goal:** Ensure the synthesized direction is fully aligned with the user's true strategic vision.

**Actions:**
1. Present the synthesized direction and check for alignment. Ask: "Does this capture what you're really after?"
2. Be prepared to reframe. If the user's feedback reveals a deeper or different vision, return to Step 2 with the new lens.
3. Confirm the vision. Before moving to execution, get explicit confirmation.
4. Produce a strategic decision document with:
   - Tension Statement
   - Routes Explored (table)
   - What We Heard (user feedback summary)
   - Synthesized Direction
   - Next Steps
   - Open Questions

**Output:** A strategic decision document saved as `[date]_strategic_scout_[topic].md`.

---

## IV. Best Practices

### 1. Begin with Tension, Not Solution

**Why:** The most powerful strategic moves come from holding the tension open long enough for better answers to emerge.

**How:** When a user says "we should do X," ask "what tension is X trying to resolve?" to open the space.

### 2. Scout for Provocation, Not Consensus

**Why:** The goal of scouting is not to find an answer everyone agrees with. It's to generate routes provocative enough to elicit a deeper conversation.

**How:** Include at least one route that challenges conventional thinking. If all routes feel "safe," push further.

### 3. Listen for the Question Behind the Question

**Why:** The user's feedback often reveals a deeper, more important question than the one initially asked.

**How:** Pay attention not just to which routes the user picks, but how they frame their reactions. The framing often contains the reframe.

### 4. Name the Reframe

**Why:** Unnamed reframes get lost. Named reframes become decision anchors that teams can rally around.

**How:** When a synthesis emerges, give it a memorable name. "The Two-Speed Strategy" or "Complement, Don't Compete" -- something that captures the essence.

### 5. Never Recommend a Single Route

**Why:** The scout's job is to present the landscape, not to choose the path. Premature recommendation short-circuits the conversation.

**How:** Present all routes equally. Ask which resonate. Let the user's reaction drive synthesis.

---

## V. Quality Checklist

Before delivering, verify:

- [ ] Have you clearly articulated the core strategic tension?
- [ ] Have you scouted at least 3 distinct routes (never fewer)?
- [ ] Does each route have a name, thesis, tradeoffs, and what it optimizes/sacrifices?
- [ ] Have you presented routes without recommending one?
- [ ] Have you gathered user feedback before synthesizing?
- [ ] Have you looked for opportunities to synthesize the best aspects of multiple routes?
- [ ] Have you named the reframe or synthesized direction?
- [ ] Have you gotten explicit confirmation that the direction is aligned with the user's vision?
- [ ] Is the output saved as a dated markdown file?

---

## VI. Common Pitfalls

### Pitfall 1: Jumping to Solutions

**Problem:** Skipping the tension identification and going straight to "here are some options."

**Solution:** Always start by naming the tension. If the user presents a solution, ask what tension it resolves.

### Pitfall 2: Routes That Are Too Similar

**Problem:** Generating 5 routes that are minor variations on the same theme.

**Solution:** Force diversity. Include at least one route that's a radical departure, one that's the status quo, and one that reframes the question entirely.

### Pitfall 3: Recommending a Favorite

**Problem:** Subtly or explicitly pushing the user toward one route before gathering feedback.

**Solution:** Present all routes with equal weight. Use a table format. Let the user react first.

### Pitfall 4: Skipping Synthesis

**Problem:** The user picks a route and you move straight to execution without synthesizing.

**Solution:** Even when a user has a clear preference, check whether elements from other routes could strengthen the chosen direction.

### Pitfall 5: One-Shot Scouting

**Problem:** Treating scouting as a single pass instead of an iterative conversation.

**Solution:** Plan for at least two rounds. The first scout reveals the real question; the second scout answers it. See `iterative-scouting` skill.

---

## VII. Example: Six-Route Navigation Architecture

**Tension:** "Should all six routes be visible from day one, or should they unlock progressively?"

**Routes Scouted:**
| Route | Thesis | Optimizes For | Sacrifices |
|-------|--------|--------------|------------|
| A: Horizon Bar | All routes visible, ambient suggestions | Discovery, completeness | Beginner simplicity |
| B: The River | No nav bar, content-edge transitions | Calm UX, user agency | Route discovery |
| C: Constellation | Fullscreen map, spatial navigation | Visual memory, exploration | Small screens, simplicity |
| D: Companion | AI opens routes during conversation | Beginner onboarding | Power user control |
| E: Thresholds | Three open, three closed, ritual unlock | Progressive disclosure, pedagogy | Immediate access |

**User Feedback:** Routes B and E resonated most. The calm, content-edge transitions of B combined with E's progressive revelation felt right. Route A's persistent bar was useful but needed simplification. Routes C and D felt too heavy.

**Synthesis -- "Show All, Open Three":** Three routes open and navigable for beginners. Three deep routes visible but distant (misty mountain shapes). Content-edge transitions instead of drag-and-drop. A home state landscape for returning users. Entities visible from routes (lenses), not contained by them.

**Outcome:** Transformed a binary unlock question into a hybrid navigation philosophy that preserved calm UX while enabling progressive discovery.

---

## VIII. Related Skills

- **`product-positioning`** -- Use before scouting when the tension is a binary product decision
- **`iterative-scouting`** -- Use to formalize the scout-feedback-reframe loop across multiple rounds
- **`multi-surface-strategy`** -- Use when the tension involves product surfaces or platforms
- **`strategic-to-tactical-workflow`** -- Use to move from scouting to specification and implementation
