---
name: zenflow-prompt-writer
model: sonnet
description: "Produces a structured Zenflow commission document (Objective, Context & Grounding with pattern files, step-by-step Requirements, File Manifest, binary Success Criteria, and explicit Constraints) from a finalized spec, saved to prompts/vX.X.X/. Use when: 'write a Zenflow prompt', 'commission Zenflow for this spec', 'turn this spec into a Zenflow prompt', 'create the implementation prompt for Zenflow'."
category: specification-driven-development
triggers:
  - "write a Zenflow prompt"
  - "commission Zenflow for this spec"
  - "turn this spec into a Zenflow prompt"
  - "create the implementation prompt for Zenflow"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec_path
    type: string
    description: Path to the finalized specification to convert into a Zenflow commission document
    required: true
  - name: version
    type: string
    description: Version string for the prompt (e.g. v3.1.0)
    required: false
outputs:
  - name: zenflow_prompt
    type: ref
    format: cas-ref
    description: Structured Zenflow commission document saved to prompts/vX.X.X/ with objective, context grounding, requirements, file manifest, and success criteria
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

---

## Output

- A Zenflow commission prompt file (markdown) saved to `prompts/vX.X.X/NN_[task_name].md`
- The prompt contains five sections: Objective, Context & Grounding (with spec link and pattern files), Detailed Requirements, File Manifest, and Constraints & Non-Goals
- Each requirement is a numbered, atomic instruction referencing exact file paths and function names

## Examples

**Scenario 1:** "Spec for the breadcrumb component is finalized. Write the Zenflow prompt." → A prompt saved to `prompts/v0.0.26/01_implement_breadcrumb.md`, with a one-sentence objective, two pattern files pointing to existing nav components, five numbered requirements specifying file paths and data sources, and a constraint preventing Zenflow from touching the sidebar.

**Scenario 2:** "Commission Zenflow to add the SSE streaming handler to the backend." → A prompt grounding Zenflow in the existing handler at `handlers/chat.go`, requirements numbered step-by-step from "add the route registration" through "write the SSE write loop", and success criteria checking that the `/api/v1/stream` endpoint returns `text/event-stream` content type.

## Edge Cases

- When the spec has not been finalized and approved, do not write a Zenflow prompt — ambiguous specs produce ambiguous implementations; complete the spec and run `pre-implementation-checklist` first
- When the task is too large to complete in 1-2 hours, split it into multiple prompts before commissioning — Zenflow works best on atomic, bounded tasks
- When Zenflow will work in parallel with another agent, add explicit file reservation constraints ("DO NOT modify `src/app/layout.tsx`") to prevent simultaneous edits to shared files

## Anti-Patterns

- Writing requirements that say "follow the spec" and link to a 15-page document — Zenflow needs the relevant requirements extracted and numbered in the prompt itself; the spec is context, not instructions
- Omitting pattern files and expecting Zenflow to infer the correct code style — pattern files eliminate ambiguity about structure, naming, and conventions without requiring lengthy style-guide prose
- Writing success criteria that require Zenflow to make judgment calls ("ensure the UI looks consistent") — every criterion must be verifiable without subjective assessment
