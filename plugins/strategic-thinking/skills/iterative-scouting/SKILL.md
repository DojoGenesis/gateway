# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: iterative-scouting
description: Progressive market and competitive research through iterative cycles
triggers:
  - "conduct iterative research"
  - "scout market progressively"
  - "research competitors iteratively"
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
    - research-agent
---

# Iterative Scouting Pattern Skill

**Version:** 1.0
**Created:** 2026-02-07
**Author:** Manus AI
**Purpose:** To formalize the meta-process of strategic scouting, emphasizing its iterative nature and the importance of reframing.

---

## I. The Philosophy: Scouting as a Conversation

Strategic scouting is not a linear process of finding the "right" answer. It is a **conversation** with the strategic landscape. The goal of the first scout is not to produce a final decision, but to generate a set of provocative routes that will elicit a deeper, more insightful response. The real prize is not the initial answer, but the **reframe** of the original question.

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

### Step 3: Re-Scout with the New Lens

**Goal:** To conduct a second round of scouting using the new, more powerful framing.

**Actions:**
-   Articulate the new, reframed tension (e.g., "Deep Work vs. On-the-Go").
-   Use `/strategic-scout` again with this new lens.
-   Explore routes that are native to the new framing.

### Step 4: Synthesize and Align on the Final Vision

**Goal:** To synthesize the results of the second scouting round into a final, coherent product vision.

**Actions:**
-   Select the best route from the second round.
-   Define the final product positioning, timeline, and business model.
-   Confirm alignment and commit to the vision.

---

## IV. Best Practices

-   **The Two-Scout Rule:** For any non-trivial strategic decision, assume you will need at least two rounds of scouting.
-   **The Reframe is the Prize:** The most valuable output of the process is the new, more powerful question you discover.
-   **Scout for Provocation, Not for Consensus:** The goal of the first scout is to provoke a better conversation, not to find an answer that everyone agrees with.

---

## V. Quality Checklist

Before concluding the process, ensure you can answer "yes" to all of the following questions:

-   [ ] Have you completed at least two rounds of scouting?
-   [ ] Can you clearly articulate the initial tension and the reframed tension?
-   [ ] Did the reframe lead to a richer, more insightful set of strategic options?
-   [ ] Is the final vision aligned with the insights from the second scouting round?
-   [ ] Is the final decision documented and socialized?
