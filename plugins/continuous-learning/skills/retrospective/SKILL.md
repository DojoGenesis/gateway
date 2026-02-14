# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: retrospective
description: Reflect on completed work to extract learnings and improvement opportunities
triggers:
  - "conduct retrospective"
  - "reflect on completed work"
  - "extract project learnings"
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
    - research-agent
---

# Retrospective Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a structured, repeatable process for conducting sprint retrospectives, harvesting learnings, and feeding them back into our collaborative practice.

---

## I. The Philosophy of the Harvest

A retrospective is not a post-mortem. It is not about blame or judgment. It is a **harvest**. After a season of hard work, we pause to gather the fruits of our labor—not just the features we shipped, but the wisdom we gained in the process. It is a practice of gratitude, honesty, and a commitment to continuous learning.

This skill turns the informal act of looking back into a formal ritual, ensuring that the valuable lessons from each sprint are not lost, but are instead integrated into our shared memory and future workflows.

---

## II. When to Use This Skill

-   **After a major release:** (e.g., after Dojo Genesis v0.0.25 is fully shipped)
-   **After a significant milestone:** (e.g., after completing a major refactor)
-   **When a project feels stuck or has encountered significant friction.**
-   **At regular intervals** (e.g., monthly) to maintain a cadence of reflection.

---

## III. The Retrospective Workflow

### Step 1: Initiate the Retrospective

When a trigger event occurs, state the intention to conduct a retrospective. Frame it as a positive and necessary part of the workflow.

**Example:** "Now that v0.0.25 is shipped, let's conduct a retrospective to harvest the learnings from this sprint. I'll use the Retrospective skill to guide us."

### Step 2: Create the Retrospective Document

Create a new markdown file in the relevant project's `docs/retrospectives/` directory (e.g., `/home/ubuntu/dojo-genesis/docs/retrospectives/v0.0.25_retrospective.md`). Use the template provided in Section IV of this skill.

### Step 3: Answer the Three Core Questions

Collaboratively, answer the three core questions of the retrospective. This should be a brainstorming phase. Capture all thoughts without judgment.

1.  **What went well?** (What should we amplify?)
2.  **What was hard?** (What were the sources of friction or difficulty?)
3.  **What would we do differently next time?** (What are the concrete, actionable changes we can make?)

### Step 4: Synthesize and Extract Learnings

Analyze the answers to the three questions and synthesize them into key themes and actionable insights. This is the most critical step.

-   **Identify Patterns:** Look for recurring themes across the different answers.
-   **Distill Actionable Insights:** For each theme, identify a concrete action that can be taken to improve the workflow.
-   **Create Seeds for the Memory Garden:** If a lesson is particularly profound or reusable, distill it into a Dojo Seed.

### Step 5: Commit and Share

Commit the completed retrospective document to the repository. Share the key findings and action items to ensure they are integrated into the next sprint's planning.

---

## IV. Retrospective Document Template

```markdown
# Retrospective: [Sprint/Release Name]

**Date:** [Date]
**Participants:** [e.g., Cruz, Manus]
**Context:** A reflection on the [Sprint/Release Name] sprint, which focused on [brief description of the sprint's goals].

---

## 1. The Three Core Questions

### What Went Well?

- *[Observation 1]*
- *[Observation 2]*
- *[Observation 3]*

### What Was Hard?

- *[Friction Point 1]*
- *[Friction Point 2]*
- *[Friction Point 3]*

### What Would We Do Differently Next Time?

- *[Actionable Suggestion 1]*
- *[Actionable Suggestion 2]*
- *[Actionable Suggestion 3]*

---

## 2. Key Themes & Insights

| Theme | Analysis | Action Item |
| :--- | :--- | :--- |
| **[Theme 1]** | [A summary of the observations related to this theme.] | [A concrete action to be taken in the next sprint.] |
| **[Theme 2]** | [A summary of the observations related to this theme.] | [A concrete action to be taken in the next sprint.] |
| **[Theme 3]** | [A summary of the observations related to this theme.] | [A concrete action to be taken in the next sprint.] |

---

## 3. Seeds for the Memory Garden

-   **Seed:** [Seed Name] — *Why it matters:* [Explanation] — *Revisit trigger:* [When to remember this lesson]

---

## 4. Closing

[A brief closing statement summarizing the key takeaway from the retrospective and expressing gratitude for the practice.]
```

---

## V. Best Practices

-   **Be Honest and Gentle:** The goal is learning, not blame.
-   **Focus on Process, Not People:** Analyze the workflow, not the individuals.
-   **Be Specific and Concrete:** Avoid vague statements. Use specific examples.
-   **End with Action:** Every retrospective should result in at least one concrete action item for the next sprint.
-   **Keep it Lightweight:** The process should feel energizing, not burdensome.
