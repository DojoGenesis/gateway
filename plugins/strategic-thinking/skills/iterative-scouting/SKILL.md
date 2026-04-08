---
name: iterative-scouting
description: >
  Formalize the meta-process of strategic scouting as an iterative loop:
  scout, gather feedback, reframe, re-scout. Use when the initial framing
  of a problem feels too narrow, after an initial strategic scout has been
  completed, or when facing a complex strategic decision where the first
  answer is unlikely to be the best answer. The reframe is the prize.
triggers:
  - "reframe and re-scout this strategic question"
  - "iterate on the scouting with new feedback"
  - "the first scout was too narrow, widen the lens"
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

# Iterative Scouting Pattern Skill

**Version:** 1.0
**Purpose:** To formalize the meta-process of strategic scouting, emphasizing its iterative nature and the importance of reframing. Scouting is not a single pass -- it's a conversation with the strategic landscape.

---

## I. The Philosophy: Scouting as a Conversation

Strategic scouting is not a linear process of finding the "right" answer. It is a **conversation** with the strategic landscape. The goal of the first scout is not to produce a final decision, but to generate a set of provocative routes that will elicit a deeper, more insightful response.

The real prize is not the initial answer, but the **reframe** of the original question.

This skill operationalizes the pattern of **scout -> feedback -> reframe -> re-scout**, turning a simple exploration into a powerful engine for strategic discovery. Where the `strategic-scout` skill handles a single pass, this skill handles the multi-pass loop that produces the deepest insights.

---

## II. When to Use This Skill

- When facing a complex strategic decision with no obvious answer
- When the initial framing of a problem feels too narrow or binary
- After an initial strategic scout has been completed and feedback has been gathered
- When the user's feedback reveals a deeper question than the one originally asked
- When you need to teach or demonstrate the process of strategic thinking
- When the first round of scouting produced useful routes but none feel quite right

**When NOT to use:**
- When the first scout produced a clear winner that the user is confident about
- When the decision is time-sensitive and a second round would delay action unacceptably
- For simple, well-understood decisions that don't benefit from iteration

---

## III. The Workflow

This is a 4-step iterative workflow. Steps 2-3 may repeat multiple times.

### Step 1: Initial Scout

**Goal:** Explore the initial strategic tension and propose a set of viable routes.

**Actions:**
1. Identify the initial tension (e.g., "Deprecate vs. Companion")
2. Use the `strategic-scout` workflow to explore 3-5 diverse routes
3. Present the routes without recommending one
4. Synthesize a provocative starting point -- not to commit to, but to react against

**Output:** An initial set of routes and a provocative synthesis.

**Key insight:** The first scout is for provocation, not consensus. Make routes provocative enough to elicit the real conversation.

### Step 2: Gather Feedback and Listen for the Reframe

**Goal:** Present the initial scouting results and listen for the deeper question hidden in the feedback.

**Actions:**
1. Present the initial routes
2. Listen not just for agreement or disagreement, but for the *way* the feedback is framed
3. Identify the "question behind the question":
   - Does the user's reaction reveal a different tension than the one you started with?
   - Is the user introducing a new lens or context you hadn't considered?
   - Does the feedback suggest the original framing was too narrow?
4. Articulate the reframe explicitly: "It sounds like the real question isn't [original], but [reframed]."

**Output:** Either confirmation of the current direction (skip to Step 4) or a clearly articulated reframe.

**Key insight:** Listen for how feedback is framed, not just what is said. The framing often contains the reframe.

### Step 3: Re-Scout with the New Lens

**Goal:** Conduct a second (or subsequent) round of scouting using the new, more powerful framing.

**Actions:**
1. Articulate the reframed tension clearly (e.g., "Deep Work vs. On-the-Go")
2. Scout 3-5 new routes that are native to the reframed tension -- not recycled from the first round
3. Present the new routes alongside a brief summary of how the framing shifted
4. Listen again for feedback. If another reframe emerges, repeat Steps 2-3.

**Output:** A new set of routes native to the reframed tension.

**Key insight:** New routes should feel genuinely different from the first round. If they feel like minor variations, the reframe wasn't deep enough.

### Step 4: Synthesize and Align on the Final Vision

**Goal:** Synthesize the results of the final scouting round into a coherent direction.

**Actions:**
1. Select the best route (or hybrid) from the most recent round
2. Define the final direction with clear rationale
3. Document the journey: initial tension, reframe(s), and how the final direction emerged
4. Confirm alignment with the user before moving to execution

**Output:** A strategic direction document that includes the full scouting journey (initial tension, reframes, final synthesis).

---

## IV. Best Practices

### 1. The Two-Scout Rule

**Why:** For any non-trivial strategic decision, the first scout reveals the real question and the second scout answers it.

**How:** Plan for at least two rounds of scouting. Set expectations with the user that the first round is exploratory.

### 2. The Reframe is the Prize

**Why:** The most valuable output of the process is not a chosen route but a new, more powerful question.

**How:** After each round of feedback, explicitly name the reframe. Write it down. Compare it to the original tension.

### 3. Scout for Provocation, Not for Consensus

**Why:** Safe, consensus-friendly routes don't generate the friction needed for reframes.

**How:** Include at least one provocative route per round -- something that challenges assumptions or inverts the question.

### 4. Don't Recycle Routes

**Why:** The reframe should produce genuinely new options, not minor variations of previous routes.

**How:** After a reframe, generate routes from scratch using the new lens. If old routes still apply, they'll re-emerge naturally.

### 5. Know When to Stop

**Why:** Infinite scouting is analysis paralysis. The loop should converge, not diverge.

**How:** If the second round produces alignment, stop. If a third round is needed, that's fine, but ask: "Are we refining or avoiding a decision?"

---

## V. Quality Checklist

Before concluding, verify:

- [ ] Have you completed at least two rounds of scouting?
- [ ] Can you clearly articulate the initial tension and the reframed tension?
- [ ] Did the reframe lead to a richer, more insightful set of strategic options?
- [ ] Are the second-round routes genuinely different from the first-round routes?
- [ ] Is the final vision aligned with the insights from the final scouting round?
- [ ] Is the full journey (initial tension -> reframe -> synthesis) documented?
- [ ] Has the user confirmed alignment with the final direction?

---

## VI. Common Pitfalls

### Pitfall 1: Treating the First Scout as Final

**Problem:** Committing to a route from the first round without testing whether a deeper question exists.

**Solution:** Always gather feedback after the first round. Ask: "Does this feel like the right question, or is there something deeper?"

### Pitfall 2: Recycling Routes Across Rounds

**Problem:** The second scout produces the same routes with minor tweaks, missing the point of the reframe.

**Solution:** Generate routes from scratch using the new lens. Old routes that still apply will re-emerge naturally.

### Pitfall 3: Infinite Scouting

**Problem:** Using the iterative loop to avoid making a decision. Each round produces a new reframe, and the process never converges.

**Solution:** Set a maximum of 3 rounds. If alignment hasn't emerged by the third round, the tension may need to be split into smaller, more tractable questions.

### Pitfall 4: Losing the Thread

**Problem:** After multiple rounds, forgetting the original tension and how the conversation evolved.

**Solution:** Document each round's tension, routes, feedback, and reframe. The journey is as valuable as the destination.

---

## VII. Example: Web App to Multi-Surface Strategy

**Initial Tension:** "Should we deprecate the web app or ship it as a companion to desktop?"

**Round 1 Scout:** 5 routes explored (Full Deprecation, Simplified Companion, Web as Beta, PWA Bridge, Freemium Split). Recommended hybrid: Simplified Web Companion as Pre-Alpha.

**Feedback and Reframe:** User's response shifted the conversation: "What if the web app is for mobile-first, on-the-go orchestration?" This revealed the real question wasn't about the web app's fate -- it was about what mobile users need.

**Reframed Tension:** "Desktop for Deep Work vs. Mobile for On-the-Go Orchestration."

**Round 2 Scout:** New routes native to the reframe (Mobile-First PWA, Native Companion App, Hybrid PWA-then-Native, Desktop-Only with Mobile Notifications). Selected: Hybrid PWA Now, Native Later.

**Final Synthesis:**
- Desktop: Core product for deep work ($20/month)
- Mobile PWA: Premium tier for on-the-go orchestration (separate subscription, 4-6 weeks after desktop)
- Web: Free tier for discovery and onboarding

**The Journey:** Started with "deprecate or companion?" and ended with a three-tier multi-surface strategy. The reframe from "web app fate" to "mobile context" was the transformative insight.

---

## VIII. Related Skills

- **`strategic-scout`** -- The single-pass scouting workflow used within each round
- **`product-positioning`** -- Use when the reframe reveals a binary positioning question
- **`multi-surface-strategy`** -- Use when the reframe reveals a multi-surface opportunity
