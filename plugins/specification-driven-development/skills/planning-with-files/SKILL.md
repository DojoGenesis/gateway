# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: planning-with-files
description: Plan development work using file-based specifications and tracking
triggers:
  - "plan with specification files"
  - "create file-based plan"
  - "organize work in files"
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

# Planning with Files (Meta-Skill)

**Version:** 1.0  
**Created:** 2026-02-08  
**Author:** Manus AI  
**Purpose:** To provide a unified interface for all file-based planning workflows while maintaining the specialized quality of each mode.

---

## I. The Philosophy: One Entry, Many Paths

This skill acts as an intelligent router, analyzing the user's request and uploaded files to determine the most appropriate specialized workflow. It embodies the principle of **intent-based routing**, ensuring that the right tool is used for the right job without requiring the user to know the names of the individual skills.

---

## II. When to Use This Skill

- **When you need to create a plan** from uploaded files (specs, docs, code, research).
- **When you want to write a specification** for a new feature or refactoring task.
- **When you need to synthesize multiple research files** into actionable insights.
- **When you have files and a planning goal**, but you're not sure which skill to use.

---

## III. The Workflow

This is a 3-step workflow for routing and executing file-based planning tasks.

### Step 1: File and Intent Analysis

**Goal:** Determine which mode is most appropriate for the user's request.

1.  **Analyze Files:** Read all uploaded files to understand their type, content, and quantity.
2.  **Analyze Intent:** Analyze the user's request for keywords and intent signals.
3.  **Apply Routing Logic:** Use the routing table below to select the appropriate mode.

**Routing Table:**

| File Types & Quantity | User Intent Keywords | Selected Mode |
| --------------------- | -------------------- | ------------- |
| 1-2 files (any type) | "plan", "refactor", "next steps" | `context-ingestion` |
| Spec, requirements doc | "spec", "zenflow", "prompt" | `specification-driven-development` |
| 3+ research files (PDFs) | "synthesize", "research", "patterns" | `research-synthesis` |
| Ambiguous | (default) | `context-ingestion` |

### Step 2: Mode Routing and Execution

**Goal:** Hand off the task to the selected specialized mode and monitor execution.

1.  **Invoke Skill:** Load the selected skill's workflow (`context-ingestion`, `specification-driven-development`, or `research-synthesis`).
2.  **Pass Context:** Pass the original files and request to the selected skill.
3.  **Monitor Execution:** Track the progress of the specialized skill.

### Step 3: Delivery and Feedback

**Goal:** Deliver the results to the user and gather feedback for routing improvement.

1.  **Present Output:** Deliver the output file(s) from the specialized skill.
2.  **Explain Routing:** Briefly explain why the mode was selected (e.g., "I used the research synthesis engine because you uploaded multiple research papers.").
3.  **Gather Feedback:** Ask if the selected mode was appropriate and log feedback for future routing improvements.

---

## IV. Best Practices

- **Trust the Router:** The routing logic is designed to select the best mode for the job. Let the skill do its work.
- **Provide Clear Intent:** The more specific your request, the better the routing will be (e.g., "synthesize these papers" is better than "look at these files").
- **User Override:** If the skill selects the wrong mode, you can always explicitly invoke the correct skill.

---

## V. Quality Checklist

- [ ] Have you uploaded all relevant files for the planning task?
- [ ] Have you provided a clear request with intent keywords (plan, spec, synthesize)?
- [ ] Have you reviewed the output to ensure it matches your expectations?
- [ ] Have you provided feedback on the routing decision if it was incorrect?

---

## VI. Common Pitfalls

### Underspecified Intent

**Problem:** Vague requests like "look at these files" force the router to guess. Guessing defaults to `context-ingestion`, which may not be what you wanted.

**Solution:** Include an intent keyword: "plan from these files," "synthesize these papers," or "spec this feature." One word changes the routing.

### Mixing File Types Without Context

**Problem:** Uploading a spec, three research papers, and a code file together. The router sees conflicting signals and can't determine which workflow you need.

**Solution:** Either group files by purpose (research together, specs together) or state your intent explicitly. The router handles ambiguity, but clarity is faster.

### Skipping the Routing Explanation

**Problem:** Jumping straight to the output without telling the user which mode was selected. If the mode was wrong, the user won't know until the output is off.

**Solution:** Always state which mode was selected and why before delivering results. One sentence is enough: "I routed this to research-synthesis because you uploaded 4 research PDFs."

---

## VII. Example

**Context:** A user uploads two files — a backend architecture doc and a product brief — and says "help me plan the frontend for this."

**Routing decision:** The word "plan" + 2 files → `context-ingestion`. But "frontend" + "backend architecture doc" is a stronger signal for `frontend-from-backend`. The router selects `frontend-from-backend` based on the file type override.

**Outcome:** A frontend specification grounded in the actual backend endpoints, data models, and auth patterns from the uploaded architecture doc.

---

## VIII. Related Skills

- **context-ingestion** — Default routing target for general planning from files
- **research-synthesis** — Target for multi-paper synthesis
- **release-specification** — Target when files indicate a spec-writing workflow
- **frontend-from-backend** — Target when backend architecture docs are present with frontend intent
