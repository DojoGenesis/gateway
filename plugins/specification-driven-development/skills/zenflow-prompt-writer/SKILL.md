# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: zenflow-prompt-writer
description: Write Zenflow-style prompts for AI-assisted development
triggers:
  - "write zenflow prompt"
  - "create ai prompt"
  - "generate development prompt"
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

# Zenflow Prompt Writer Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a structured methodology for writing effective prompts for Zenflow, maximizing the probability of a successful, high-quality, and autonomous implementation.

---

## I. The Philosophy: The Art of Commissioning

A prompt to Zenflow is not a command; it is a **commission**. It is a formal request for a work of craftsmanship. The quality of the commission directly determines the quality of the work. A vague, incomplete, or ambiguous prompt invites confusion, rework, and failure. A clear, comprehensive, and well-grounded prompt is an act of respect for the builder's time and capability.

This skill transforms prompt writing from a hopeful guess into a deliberate and rigorous engineering discipline. By following this structure, we provide Zenflow with everything it needs to succeed, enabling it to work with precision, autonomy, and a deep understanding of the existing codebase.

---

## II. When to Use This Skill

-   **Always** use this skill when creating a new development task for Zenflow.
-   Use it after a specification has been finalized and has passed the `pre-implementation-checklist`.
-   Use it to break down a large specification into smaller, manageable implementation chunks for Zenflow.

---

## III. The Prompt Writing Workflow

### Step 1: Ground the Prompt in Context

Before writing, gather all necessary context. Zenflow has full access to the repository, so leverage this. Your primary job is to be an excellent librarian, pointing Zenflow to the right information.

-   **Link to the Specification:** The prompt must always link to the final, approved specification document.
-   **Identify Key Patterns:** Find 2-3 existing files in the codebase that Zenflow should use as a pattern for its work.
-   **Gather Relevant Files:** List any other files Zenflow will need to read or modify.

### Step 2: Write the Prompt Using the Template

Create a new markdown file for the prompt (e.g., `prompts/v0.0.26/01_implement_breadcrumb.md`) and fill out the template from Section IV of this skill. Be precise and thorough.

### Step 3: Review the Prompt Against the Checklist

Before sending the prompt to Zenflow, review it against the quality checklist in Section V. Ensure every item is addressed. This is the final quality gate.

### Step 4: Execute the Zenflow Task

With a high-quality prompt in hand, you can now confidently commission Zenflow to perform the work.

---

## IV. Zenflow Prompt Template

```markdown
# Zenflow Commission: [Brief, Descriptive Title of Task]

**Objective:** [A single sentence describing the high-level goal of this task.]

---

## 1. Context & Grounding

**Primary Specification:**
-   [Link to the final specification document (e.g., `docs/v0.0.26_specification.md`)]

**Pattern Files (Follow these examples):**
-   `[path/to/existing_file_1.tsx]`: Use this for component structure and styling.
-   `[path/to/existing_file_2.go]`: Use this for backend API endpoint structure and error handling.

**Files to Read/Modify:**
-   [List of all files that will be touched by this task.]

---

## 2. Detailed Requirements

[Provide a step-by-step, unambiguous list of implementation requirements. Be ruthlessly specific.]

**Backend (Go):**
1.  In `[path/to/file.go]`, create a new function `[FunctionName]` that...
2.  Add a new API endpoint `GET /api/v1/[resource]` that...
3.  The endpoint must return a JSON object with the following structure: `[JSON structure]`

**Frontend (React/TypeScript):**
1.  Create a new component at `[path/to/new_component.tsx]` named `[ComponentName]`.
2.  The component must fetch data from the `GET /api/v1/[resource]` endpoint.
3.  It must render the data using the following JSX structure, following the styling patterns in the reference file.

---

## 3. File Manifest

[A complete list of all files to be created or modified. This helps Zenflow verify its work.]

**Create:**
-   `[path/to/new_file_1.ts]`
-   `[path/to/new_file_2.tsx]`

**Modify:**
-   `[path/to/existing_file_1.go]`
-   `[path/to/existing_file_2.go]`

---

## 4. Success Criteria

[How will we know this task is done? The criteria must be binary and testable.]

-   [ ] The new `[ComponentName]` component renders correctly at the `/page` route.
-   [ ] Clicking the component triggers a call to the `GET /api/v1/[resource]` endpoint.
-   [ ] The backend returns a `200 OK` status with the correct JSON payload.
-   [ ] All new code is covered by unit tests with at least 80% coverage.

---

## 5. Constraints & Non-Goals

[What should Zenflow explicitly *not* do?]

-   **DO NOT** modify any files outside of the File Manifest.
-   **DO NOT** introduce any new third-party dependencies.
-   **DO NOT** address [related feature], as it is out of scope for this task.
```

---

## V. Quality Checklist

-   [ ] **Is the Objective a single, clear sentence?**
-   [ ] **Is the link to the specification correct?**
-   [ ] **Are there at least 1-2 relevant Pattern Files listed?**
-   [ ] **Are the Requirements specific, step-by-step, and unambiguous?**
-   [ ] **Is the File Manifest complete and accurate?**
-   [ ] **Are the Success Criteria binary and testable?**
-   [ ] **Are the Constraints clear about what *not* to do?**
-   [ ] **Does the prompt respect existing codebase patterns?**

---

## VI. Best Practices

-   **Chunk Your Prompts:** Break down large features into smaller, logical implementation chunks. A single prompt should ideally take Zenflow 1-2 hours to complete.
-   **Reference, Don't Re-explain:** Leverage Zenflow's ability to read the repo. Point it to existing patterns instead of re-explaining them.
-   **Be a Good Librarian:** The most important part of the prompt is the Context & Grounding section. Good inputs lead to good outputs.
-   **Specify File Paths:** Always use full, explicit file paths. Never say "in the utils directory."
-   **Define the "Done" State:** The Success Criteria are the definition of "done." Make them crystal clear.
