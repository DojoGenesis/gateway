---
name: zenflow-prompt-writer
description: Write clear, comprehensive prompts for the Zenflow autonomous development agent, ensuring high-quality and predictable implementation
version: "1.0"
lastUpdated: "2026-04-06"
triggers:
  - "write zenflow prompt"
  - "create ai prompt"
  - "generate development prompt"
metadata:
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
---

# Zenflow Prompt Writer

## I. Philosophy

A prompt to Zenflow is not a command; it is a **commission**. It is a formal request for a work of craftsmanship. The quality of the commission directly determines the quality of the work. A vague, incomplete, or ambiguous prompt invites confusion, rework, and failure. A clear, comprehensive, and well-grounded prompt is an act of respect for the builder's time and capability.

This skill transforms prompt writing from a hopeful guess into a deliberate and rigorous engineering discipline. By following this structure, we provide Zenflow with everything it needs to succeed, enabling it to work with precision, autonomy, and a deep understanding of the existing codebase.

---

## II. When to Use

- **Always** use this skill when creating a new development task for Zenflow.
- Use it after a specification has been finalized and has passed the `pre-implementation-checklist`.
- Use it to break down a large specification into smaller, manageable implementation chunks for Zenflow.

---

## III. Workflow

### Step 1: Ground the Prompt in Context

Before writing, gather all necessary context. Zenflow has full access to the repository, so leverage this. Your primary job is to be an excellent librarian, pointing Zenflow to the right information.

- **Link to the Specification:** The prompt must always link to the final, approved specification document.
- **Identify Key Patterns:** Find 2-3 existing files in the codebase that Zenflow should use as a pattern for its work.
- **Gather Relevant Files:** List any other files Zenflow will need to read or modify.

### Step 2: Write the Prompt Using the Template

Create a new markdown file for the prompt (e.g., `prompts/v0.0.26/01_implement_breadcrumb.md`) and fill out the template below. Be precise and thorough.

```markdown
# Zenflow Commission: [Brief, Descriptive Title of Task]

**Objective:** [A single sentence describing the high-level goal of this task.]

---

## 1. Context & Grounding

**Primary Specification:**
- [Link to the final specification document]

**Pattern Files (Follow these examples):**
- `[path/to/existing_file_1.tsx]`: Use this for component structure and styling.
- `[path/to/existing_file_2.go]`: Use this for backend API endpoint structure and error handling.

**Files to Read/Modify:**
- [List of all files that will be touched by this task.]

---

## 2. Detailed Requirements

[Step-by-step, unambiguous implementation requirements.]

**Backend (Go):**
1. In `[path/to/file.go]`, create a new function `[FunctionName]` that...
2. Add a new API endpoint `GET /api/v1/[resource]` that...

**Frontend (React/TypeScript):**
1. Create a new component at `[path/to/new_component.tsx]` named `[ComponentName]`.
2. The component must fetch data from the `GET /api/v1/[resource]` endpoint.

---

## 3. File Manifest

**Create:**
- `[path/to/new_file_1.ts]`

**Modify:**
- `[path/to/existing_file_1.go]`

---

## 4. Success Criteria

- [ ] The new `[ComponentName]` component renders correctly at the `/page` route.
- [ ] The backend returns a `200 OK` status with the correct JSON payload.
- [ ] All new code is covered by unit tests with at least 80% coverage.

---

## 5. Constraints & Non-Goals

- **DO NOT** modify any files outside of the File Manifest.
- **DO NOT** introduce any new third-party dependencies.
- **DO NOT** address [related feature], as it is out of scope for this task.
```

### Step 3: Review the Prompt Against the Checklist

Before sending the prompt to Zenflow, review it against the quality checklist in Section V. Ensure every item is addressed. This is the final quality gate.

### Step 4: Execute the Zenflow Task

With a high-quality prompt in hand, confidently commission Zenflow to perform the work.

---

## IV. Best Practices

- **Chunk your prompts.** Break down large features into smaller, logical implementation chunks. A single prompt should ideally take Zenflow 1-2 hours to complete.
- **Reference, don't re-explain.** Leverage Zenflow's ability to read the repo. Point it to existing patterns instead of re-explaining them.
- **Be a good librarian.** The most important part of the prompt is the Context & Grounding section. Good inputs lead to good outputs.
- **Specify file paths.** Always use full, explicit file paths. Never say "in the utils directory."
- **Define the done state.** The Success Criteria are the definition of "done." Make them crystal clear.
- **One concern per prompt.** A prompt that tries to do everything will do nothing well. Scope tightly.

---

## V. Quality Checklist

- [ ] **Is the Objective a single, clear sentence?**
- [ ] **Is the link to the specification correct?**
- [ ] **Are there at least 1-2 relevant Pattern Files listed?**
- [ ] **Are the Requirements specific, step-by-step, and unambiguous?**
- [ ] **Is the File Manifest complete and accurate?**
- [ ] **Are the Success Criteria binary and testable?**
- [ ] **Are the Constraints clear about what NOT to do?**
- [ ] **Does the prompt respect existing codebase patterns?**

---

## VI. Example

**Prompt file:** `prompts/v0.0.26/01_implement_breadcrumb.md`

```markdown
# Zenflow Commission: Implement Breadcrumb Navigation

**Objective:** Add breadcrumb navigation to the dashboard that reflects the user's current position in the page hierarchy.

## 1. Context & Grounding

**Primary Specification:**
- `docs/v0.0.26_specification.md`, Section 3.2

**Pattern Files:**
- `src/components/Sidebar.tsx`: Follow this for component structure, TailwindCSS usage, and state management patterns.
- `internal/api/routes.go`: Follow this for any new route registration.

**Files to Read/Modify:**
- `src/components/Dashboard.tsx` (modify -- add breadcrumb component)
- `src/components/Breadcrumb.tsx` (create)
- `src/hooks/useLocation.ts` (read -- extract current route info)

## 2. Detailed Requirements

**Frontend (React/TypeScript):**
1. Create `src/components/Breadcrumb.tsx` that accepts a `path: string[]` prop.
2. Each segment renders as a clickable link except the last (current page).
3. Use the `useLocation` hook to derive the path segments.
4. Style using TailwindCSS following the `Sidebar.tsx` pattern.

## 3. File Manifest

**Create:** `src/components/Breadcrumb.tsx`
**Modify:** `src/components/Dashboard.tsx`

## 4. Success Criteria

- [ ] Breadcrumb renders on all dashboard pages.
- [ ] Clicking a breadcrumb segment navigates to that route.
- [ ] Current page segment is not clickable and visually distinct.
- [ ] Unit tests cover all path segment rendering scenarios.

## 5. Constraints & Non-Goals

- DO NOT modify routing logic in `useLocation.ts`.
- DO NOT add new dependencies.
```

---

## VII. Common Pitfalls

- **Omitting pattern files.** Without explicit examples, Zenflow invents its own patterns that may conflict with the codebase.
- **Vague requirements.** "Add error handling" is not actionable. "Return a 400 status with `{ error: string }` body when input validation fails" is.
- **Incomplete file manifests.** Zenflow may miss files it needs to modify, or worse, modify files it should not touch.
- **Non-binary success criteria.** "Works well" is not testable. "Returns 200 with `{ id: string }` body" is.
- **Overscoped prompts.** Trying to implement an entire spec in one prompt leads to partial, low-quality results. Chunk aggressively.
- **Re-explaining patterns.** Writing paragraphs about coding style when a single pattern file reference communicates the same thing in zero tokens.

---

## VIII. Related Skills

- `specification-writer` -- Write the specs that these prompts implement
- `pre-implementation-checklist` -- Gate to run before writing prompts
- `implementation-prompt` -- Alternative prompt format for non-Zenflow agents
- `parallel-tracks` -- Split prompts across parallel execution tracks
- `pre-commission-alignment` -- Quality gate before commissioning parallel tracks
