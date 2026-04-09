---
name: iterative-scouting
model: opus
description: Produces a follow-up scout document that reframes the original question based on what feedback revealed (a reframe = a new strategic lens that changes which problem is worth solving). Use when: 'scout this again with fresh eyes', 'the first scout raised deeper questions', 'what is the real question here', 'iterate the scout'
category: strategic-thinking

inputs:
  - name: original_scout
    type: string
    description: The original scout document or question that raised deeper issues
    required: true
  - name: feedback
    type: string
    description: Feedback or new insight that triggers the re-scout
    required: true
outputs:
  - name: follow_up_scout
    type: ref
    format: cas-ref
    description: Follow-up scout document reframing the original question with a new strategic lens
---

# Iterative Scouting Pattern Skill

**Version:** 1.1
**Created:** 2026-02-07
**Author:** Manus AI
**Purpose:** To formalize the meta-process of strategic scouting, emphasizing its iterative nature and the importance of reframing.

---

## I. The Philosophy: Scouting as a Conversation

Strategic scouting is not a linear process of finding the "right" answer. It is a **conversation** with the strategic landscape. The goal of the first scout is not to produce a final decision, but to generate a set of provocative routes that will elicit a deeper, more insightful response. The real prize is not the initial answer, but the **reframe** of the original question.

A **reframe** is a new strategic lens that changes which problem is worth solving — not a refinement of the original question, but a replacement of it with a better one.

This skill operationalizes the pattern of **scout → feedback → reframe → re-scout**, turning a simple exploration into a powerful engine for strategic discovery.

---

## II. When to Use This Skill

-   When facing a complex strategic decision with no obvious answer.
-   When the initial framing of a problem feels too narrow or binary.
-   After an initial strategic scout has been completed and feedback has been gathered.
-   When you need to teach or demonstrate the process of strategic thinking.

---

## III. The Workflow

This is a 4-step workflow for the iterative scouting pattern.

### Step 1: Initial Scout

**Goal:** To explore the initial strategic tension and propose a set of viable routes.

**Actions:**
-   Identify the initial tension (e.g., "Deprecate vs. Companion").
-   Use `/strategic-scout` to explore a diverse set of routes.
-   Synthesize the routes and propose a provocative starting point.

### Step 2: Gather Feedback & Listen for the Reframe

**Goal:** To present the initial scouting results and listen for the deeper question hidden in the feedback.

**Actions:**
-   Present the initial routes.
-   Listen not just for agreement or disagreement, but for the *way* the feedback is framed.
-   Identify the "question behind the question" (e.g., the shift from "what to do with the web app" to "what is the mobile experience for?").
-   Explicitly ask: "Did this feedback confirm the original question, or reveal a different one?"

### Step 3: Re-Scout with the New Lens

**Goal:** To conduct a second round of scouting using the new, more powerful framing.

**Actions:**
-   Articulate the new, reframed tension (e.g., "Deep Work vs. On-the-Go").
-   Use `/strategic-scout` again with this new lens.
-   Explore routes that are native to the new framing.
-   Document the before/after: original tension → reframe → second scout headline.

### Step 4: Synthesize and Align on the Final Vision

**Goal:** To synthesize the results of the second scouting round into a final, coherent product vision.

**Actions:**
-   Select the best route from the second round.
-   Define the final product positioning, timeline, and business model.
-   Confirm alignment and commit to the vision.

---

## IV. Best Practices

-   **Plan for a second scouting pass:** For any non-trivial strategic decision, assume you will need at least two rounds of scouting. The first pass surfaces routes; the second pass answers the real question.
-   **The Reframe is the Prize:** The most valuable output of the process is the new, more powerful question you discover.
-   **Scout for Provocation, Not for Consensus:** The goal of the first scout is to provoke a better conversation, not to find an answer that everyone agrees with.
-   **Name the reframe explicitly:** When a reframe occurs, write it down as "Original question: X → Reframed question: Y" before re-scouting. This makes the pivot visible and prevents reverting to the original framing mid-execution.

---

## V. Quality Checklist

Before concluding the process, ensure you can answer "yes" to all of the following questions:

-   [ ] Have you completed at least two rounds of scouting?
-   [ ] Can you clearly articulate the initial tension and the reframed tension?
-   [ ] Did the reframe lead to a richer, more insightful set of strategic options?
-   [ ] Is the final vision aligned with the insights from the second scouting round?
-   [ ] Is the final decision documented and socialized?

---

## Output

- A second scout document saved to `thinking/[topic]_scout_v2.md` (or equivalent project thinking directory)
- The document includes: original tension, reframe statement, second-pass routes with tradeoffs, and selected direction with rationale

---

## Examples

**Scenario 1:** "The first scout on mobile vs. web raised more questions than it answered — scout this again with fresh eyes." → Produces a second scout document that leads with the reframe ("Original: deprecate web. Reframed: what is mobile uniquely for?") and explores routes native to that new question.

**Scenario 2:** "What is the real question here? We keep debating the same two options." → Runs the feedback-listening step explicitly, surfaces the hidden reframe ("we're actually deciding who the primary user is"), and produces a targeted second scout scoped to that question.

---

## Edge Cases

- **Feedback confirms the original scout** — no reframe is needed. Document this explicitly ("second pass confirms original framing") and move to decision rather than forcing a reframe that isn't there.
- **Multiple competing reframes emerge from feedback** — pick the one that changes the most about the original question, not the one that feels safest. Scout the boldest reframe first.
- **The second scout also raises new questions** — limit to two scouting passes unless the stakes are very high. A third pass usually signals the question is not yet well-formed enough to scout; step back and clarify scope instead.

---

## Anti-Patterns

- **Iterating without capturing the reframe decision** — running a second scout without documenting what changed between pass 1 and pass 2. This causes the reframe to disappear in later conversations and the team reverts to the original framing.
- **Mistaking refinement for reframing** — adjusting route details from the first scout is not a reframe. A reframe changes which problem is being solved, not how a solution is described.
- **Forcing a reframe when feedback confirms the original** — not every first scout needs a second pass. If feedback aligns with the original question, commit and move on rather than manufacturing a reframe to justify the process.
