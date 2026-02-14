# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: status-writing
description: Write clear, concise status updates following established templates
triggers:
  - "write status update"
  - "create progress report"
  - "document current state"
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
    - health-agent
---

# Status Writer Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a structured, repeatable process for writing and updating `STATUS.md` files, turning project tracking into a mindful and transparent practice.

---

## I. The Philosophy: The Ritual of Bearing Witness

A `STATUS.md` file is more than a report; it is a **Ritual of Bearing Witness**. It is a practice of radical honesty about where a project truly is—not where we wish it were, or where it is supposed to be. It is a pause to see clearly, without judgment, the current state of our work.

This ritual combats the natural tendency for entropy and confusion to creep into complex projects. It provides a single, trusted source of truth that grounds our conversations and decisions in reality. By maintaining this document with care, we cultivate transparency, accountability, and a shared understanding of our journey.

---

## II. When to Use This Skill

-   **At the beginning of a new project:** To establish the initial state and vision.
-   **At the start and end of a work session:** To frame the day's work and document its outcome.
-   **During a weekly sync:** To facilitate a high-level review of all active projects.
-   **Whenever there is a significant change in project status:** (e.g., a new blocker emerges, a major milestone is reached).

---

## III. The Status Update Workflow

### Step 1: Locate or Create the `STATUS.md` File

Navigate to the root of the project repository. If a `STATUS.md` file does not exist, create one using the template from Section IV of this skill.

### Step 2: Update the Header

Change the `Last Updated` date to the current date.

### Step 3: Review and Update Each Section

Go through each section of the `STATUS.md` file and update it to reflect the current reality.

-   **Vision & Purpose:** This should rarely change, but it is good to re-read it to stay grounded.
-   **Current State:** This is the most important section. Update the status of each area using the emoji key (✅, 🔄, ⏸️, ❌). Add or remove items as the project evolves.
-   **Active Workstreams:** What are you actively working on *right now*? Update the tasks and progress.
-   **Blockers & Dependencies:** Be ruthlessly honest about what is preventing progress. If a blocker is resolved, remove it.
-   **Next Steps:** What is the immediate next action? This should be concrete and actionable.

### Step 4: Commit the Changes

Commit the updated `STATUS.md` file with a clear and descriptive commit message.

**Commit Message Convention:**
`docs(status): Update [Project Name] status for [Date]`

**Example:**
`docs(status): Update AROMA status for 2026-02-04`

---

## IV. `STATUS.md` Template

```markdown
# [Project Name] Project Status

**Author:** [Your Name]
**Status:** [Active & Evolving | On Hold | Complete]
**Last Updated:** [YYYY-MM-DD]

---

## 1. Vision & Purpose

> [A single, compelling sentence that captures the essence of this project.]

**Core Principles:**
-   [Principle 1]
-   [Principle 2]
-   [Principle 3]

---

## 2. Current State

[A brief, high-level summary of the project's current phase or condition.]

| Area | Status | Notes |
| :--- | :--- | :--- |
| **[Key Area 1]** | [✅/🔄/⏸️/❌] | [Brief, relevant notes.] |
| **[Key Area 2]** | [✅/🔄/⏸️/❌] | [Brief, relevant notes.] |
| **[Key Area 3]** | [✅/🔄/⏸️/❌] | [Brief, relevant notes.] |

**Status Key:**
-   ✅ **Complete:** The work is done and verified.
-   🔄 **In Progress:** Actively being worked on.
-   ⏸️ **Paused:** Intentionally on hold.
-   ❌ **Blocked:** Progress is halted due to a dependency or issue.

---

## 3. Active Workstreams

[What are we actively working on right now? Be specific.]

1.  **[Workstream 1]:**
    -   **Task:** [Description of the current task.]
    -   **Progress:** [e.g., 75% complete, waiting for review]

2.  **[Workstream 2]:**
    -   **Task:** [Description of the current task.]
    -   **Progress:** [e.g., Just started, gathering requirements]

---

## 4. Blockers & Dependencies

[What is preventing progress? Be ruthlessly honest.]

-   **[Blocker 1]:** [Description of the blocker and its impact.]
    -   **Owner:** [Who is responsible for resolving this?]
    -   **Next Step:** [What is the next action to resolve this?]

---

## 5. Next Steps

[What are the immediate next actions for this project?]

1.  [A concrete, actionable task.]
2.  [A concrete, actionable task.]
3.  [A concrete, actionable task.]
```

---

## V. Best Practices

-   **Be Honest:** The value of this document is its truthfulness. Do not sugarcoat bad news.
-   **Be Concise:** Use bullet points and short sentences. This is a dashboard, not a novel.
-   **Use the Emoji Key:** The emojis provide an instant visual summary of project health.
-   **Update Regularly:** A stale status document is worse than no status document. Make it a habit.
-   **Focus on the 'What', Not the 'Who':** The status is about the project, not the people. Frame blockers and issues impersonally.
