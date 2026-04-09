---
name: context-ingestion
model: sonnet
description: "Produces a phased, file-grounded action plan from 1-2 uploaded documents (specs, meeting notes, design docs, or code files) with concrete deliverables and binary success criteria per phase. Use when: 'create a plan from this spec', 'plan the refactoring from these files', 'what are the next steps from these docs'."
category: specification-driven-development

inputs:
  - name: documents
    type: string[]
    description: Paths to 1-2 uploaded documents (specs, meeting notes, design docs, or code files)
    required: true
outputs:
  - name: action_plan
    type: string
    description: Phased, file-grounded action plan with concrete deliverables and binary success criteria per phase
---

# Context Ingestion Workflow

**Version:** 1.0  
**Created:** 2026-02-08  
**Author:** Manus AI  
**Purpose:** To create plans that are deeply informed by uploaded content, ensuring that recommendations are specific, actionable, and aligned with the provided context.

---

## I. The Philosophy: Grounding is Everything

The quality of a plan is directly proportional to how well it is grounded in the available context. This skill transforms file uploads from passive attachments into active participants in the planning process, ensuring that every recommendation is rooted in the specifics of the provided files.

---

## II. When to Use This Skill

- **When you have 1-2 files** and a general planning request (e.g., "create a plan from this spec").
- **When you need to refactor a codebase** and have uploaded the relevant files.
- **When you want to create action items** from meeting notes or a design document.
- **When the `planning-with-files` meta-skill routes to this mode.**

---

## III. The Workflow

This is a 5-step workflow for creating grounded plans from uploaded files.

### Step 1: File Ingestion and Cataloging

**Goal:** Read all uploaded files and create a structured catalog of their content.

1.  **Read Files:** Use the `file` tool to read each uploaded file (use `view` for images/PDFs).
2.  **Extract Content:** For PDFs, extract text. For code, identify structure. For images, describe visual content.
3.  **Create Catalog:** Create an internal catalog document listing all files and their key content.

### Step 2: Context Synthesis

**Goal:** Synthesize the file content into a coherent understanding of the current state.

1.  **Identify Patterns:** Look for recurring themes, architectural decisions, and coding conventions across files.
2.  **Extract Constraints:** Note any explicit or implicit constraints (e.g., "must maintain backward compatibility").
3.  **Identify Opportunities:** Note areas for improvement or refactoring.
4.  **Note Contradictions:** If files contradict each other, call this out explicitly.

### Step 3: Plan Creation

**Goal:** Create a detailed plan grounded in the synthesized context.

1.  **Define Phases:** Break the plan into clear phases with estimated durations.
2.  **Specify Actions:** For each phase, list specific actions grounded in the uploaded files (reference file names and line numbers where relevant).
3.  **Define Deliverables:** Specify concrete deliverables for each phase.
4.  **Set Success Criteria:** Define binary, testable success criteria.
5.  **Identify Risks:** List potential risks and their mitigations.

### Step 4: Validation and Refinement

**Goal:** Ensure the plan is complete, actionable, and aligned with the user's intent.

1.  **Review for Completeness:** Check that all phases have clear deliverables and success criteria.
2.  **Verify Grounding:** Ensure that all recommendations are directly tied to the uploaded files.
3.  **Check for Actionability:** Verify that the plan can be executed immediately without needing additional context.

### Step 5: Delivery

**Goal:** Deliver the plan to the user with clear next steps.

1.  **Send Plan:** Use the `message` tool to send the plan as an attachment.
2.  **Summarize:** Briefly summarize the key phases and deliverables.
3.  **Offer Refinement:** Offer to answer questions or refine the plan based on feedback.

---

## IV. Best Practices

- **Reference Specifics:** Always reference specific files, functions, or sections in the plan.
- **Make Constraints Explicit:** If you find constraints in the files, list them in the plan.
- **Write Actionable Plans:** Use clear phases, concrete deliverables, and binary success criteria.
- **Synthesize Before Planning:** Create an internal synthesis document before writing the plan to ensure deep understanding.

---

## V. Quality Checklist

- [ ] Does the plan reference specific files and line numbers?
- [ ] Does the plan explicitly list any constraints found in the files?
- [ ] Does the plan have clear phases with concrete deliverables and binary success criteria?
- [ ] Is the plan actionable without needing additional context?

## Output

- A phased action plan (typically 2-4 phases) with explicit deliverables, binary success criteria, and identified risks per phase
- An internal synthesis document summarizing key patterns, constraints, and contradictions found across the uploaded files
- A brief routing explanation stating which mode was used and why

## Examples

**Scenario 1:** Two uploaded files — a backend architecture doc and a design brief — with the request "plan the next sprint." → A 3-phase plan referencing specific endpoints from the architecture doc and components from the design brief, with a success criterion per phase that can be verified by running a build or opening a route.

**Scenario 2:** A single meeting-notes markdown file with action items scattered throughout, with the request "extract the next steps." → A prioritized, phased action list with owners and binary done-states drawn directly from the meeting notes, with a note on any contradictions found between stated priorities.

## Edge Cases

- When uploaded files contradict each other (e.g., one doc says use PostgreSQL, another says SQLite), call the contradiction out explicitly in the synthesis before writing the plan — do not silently choose one.
- When a file is a code file rather than a doc, extract structure (file paths, exported functions, types) rather than narrative content, and reference those in the plan.
- When the planning request is ambiguous (e.g., "help me think through this"), default to a short synthesis first and ask whether the user wants a phased plan or a strategic overview.

## Anti-Patterns

- Writing a generic plan that could apply to any project without referencing the specific files — if the plan would be identical without the uploads, the grounding step was skipped.
- Treating contradictions between files as resolved when they are not — surface them; the user needs to decide, not the agent.
- Producing a plan with more than 5 phases for a 1-2 file input — over-scoping signals the plan is not grounded in the files but in general process templates.
