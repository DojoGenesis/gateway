---
name: implementation-prompt
model: sonnet
description: Produces a self-contained implementation prompt document with objective, context grounding, numbered requirements, file manifest, binary success criteria, constraints, and backend grounding — ready for autonomous agent execution. Use when 'write an implementation prompt', 'commission this spec', or 'convert spec to prompt'.
category: specification-driven-development
triggers:
  - "write an implementation prompt"
  - "commission this spec"
  - "convert spec to prompt"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec
    type: string
    description: The specification or feature description to convert into an implementation prompt
    required: true
  - name: file_manifest
    type: string[]
    description: List of files the implementation agent is authorized to touch
    required: false
outputs:
  - name: implementation_prompt
    type: ref
    format: cas-ref
    description: Self-contained implementation prompt with objective, context grounding, requirements, file manifest, and binary success criteria
---

# Transform Spec to Implementation Prompt Skill

---

## I. The Philosophy: From Vision to Execution

A specification describes **what** should be built and **why**. An implementation prompt describes **how** to build it, **where** to build it, and **what success looks like**. The gap between these two documents is where implementation clarity lives or dies.

This skill bridges that gap by providing a systematic process for transforming high-level specifications into execution-ready implementation prompts. The result is a prompt that gives the implementation agent everything it needs to work autonomously: context, requirements, file paths, success criteria, and explicit boundaries.

This skill builds on the foundational `write-implementation-prompt` skill but adds the critical step of **specification grounding**—ensuring that every prompt is deeply connected to both the strategic vision (the spec) and the technical reality (the backend architecture and existing codebase).

**Core Insight:** A well-written implementation prompt is an act of translation. You are translating strategic intent into tactical execution, abstract vision into concrete file paths, and implicit knowledge into explicit instructions.

---

## II. When to Use This Skill

- **After completing a comprehensive specification document** (architectural or feature-level)
- **When preparing to commission implementation agents (Zenflow, Claude Code, etc.)** for implementation work
- **When breaking down a large specification** into parallel tracks or sequential chunks
- **After completing backend grounding** (see `frontend-from-backend` skill)
- **During Phase 7 of the strategic-to-tactical workflow** (Write Implementation Prompts & Commission)

---

## III. The Workflow

This is a 6-step workflow for transforming a specification into implementation prompts.

---

### **Step 1: Validate Specification Readiness**

**Goal:** Ensure the specification is complete and implementation-ready before writing prompts.

**When:** Before starting any prompt writing.

**Actions:**
1. Verify the specification has passed the `pre-implementation-checklist` (if applicable)
2. Confirm backend grounding is complete (API contracts, data models, integration points)
3. Check that all architectural decisions are documented
4. Verify there are no missing dependencies or unclear requirements
5. Identify if the work should be split into multiple tracks or sequential chunks

**Output:** A validated specification and a decision on track structure (single prompt vs. multiple parallel tracks).

**Key Insight:** Never write an implementation prompt from an incomplete specification. The prompt quality is directly determined by the specification quality.

---

### **Step 2: Identify Track Boundaries (If Applicable)**

**Goal:** If the specification is large, identify which track this prompt addresses and what its boundaries are.

**When:** After validating the specification, if it covers multiple tracks.

**Actions:**
1. Review the parallel tracks decomposition (if applicable)
2. Identify the scope of THIS track:
   - What components/features does it include?
   - What does it explicitly NOT include?
3. Confirm track dependencies:
   - What must be complete before this track can start?
   - What will other tracks depend on from this track?
4. Verify track integration points:
   - What props, APIs, state shapes will this track produce?
   - What will this track consume from other tracks?

**Output:** Clear understanding of track scope, boundaries, and integration points.

**Key Insight:** When working with parallel tracks, be crystal clear about what THIS track does and doesn't do. Explicit non-goals prevent scope creep.

---

### **Step 3: Ground in the Codebase**

**Goal:** Identify specific files, patterns, and integration points in the existing codebase that this track will use or modify.

**When:** After understanding track boundaries, before writing the prompt.

**Actions:**
1. **Identify Pattern Files:** Find 2-3 existing files that demonstrate the desired structure, style, and patterns
   - For frontend: existing components, layouts, contexts
   - For backend: existing handlers, middleware, API endpoints
2. **List Files to Read/Modify:** Enumerate all files that will be touched by this track
3. **Identify Files to Create:** List new files that will be created
4. **Map Integration Points:** Define the props, APIs, state shapes, and context that this track will consume or produce
5. **Reference Backend Grounding:** Link to the backend integration guide for API contracts and data models

**Output:** A complete "Context & Grounding" section for the prompt.

**Key Insight:** Pattern files are force multipliers. Implementation agents work best when you point it to existing examples of the desired structure and style.

---

### **Step 4: Write the Prompt Using the Template**

**Goal:** Transform the specification into a structured implementation prompt using the standard 6-section template.

**When:** After grounding in the codebase.

**Template Structure:**

```markdown
# Implementation Commission: [Track Name or Feature Name]

**Objective:** [One sentence describing the goal of this track/prompt]

---

## 1. Context & Grounding

**Primary Specification:**
-   `docs/[version]/[spec_file].md`

**Pattern Files (Follow these examples):**
-   `[path/to/pattern_file_1]`: Use this for [specific pattern, e.g., component structure]
-   `[path/to/pattern_file_2]`: Use this for [specific pattern, e.g., API endpoint structure]

**Files to Read/Modify:**
-   [List all files that will be touched]

---

## 2. Detailed Requirements

[Step-by-step, numbered requirements. Be ruthlessly specific.]

1.  [Requirement 1]
2.  [Requirement 2]
3.  [Requirement 3]

---

## 3. File Manifest

**Create:**
-   `[path/to/new_file_1]`
-   `[path/to/new_file_2]`

**Modify:**
-   `[path/to/existing_file_1]`
-   `[path/to/existing_file_2]`

---

## 4. Success Criteria

-   [ ] [Binary, testable criterion 1]
-   [ ] [Binary, testable criterion 2]
-   [ ] [Binary, testable criterion 3]

---

## 5. Constraints & Non-Goals

-   **DO NOT** [Explicit boundary 1]
-   **DO NOT** [Explicit boundary 2]

---

## 6. Backend Grounding

-   **Backend Integration Guide:** `docs/[version]/backend_integration_guide.md`
-   **API Endpoints Used:** [List specific endpoints this track will call]
-   **Data Models:** [List TypeScript interfaces or Go structs this track will use]
```

**Actions:**
1. Fill out each section systematically
2. Ensure requirements are step-by-step and specific
3. Make success criteria binary (yes/no, no ambiguity)
4. Explicitly state what NOT to do in constraints
5. Keep the objective to a single, clear sentence
6. Reference backend grounding even if not calling APIs yet

**Output:** A complete implementation prompt document.

**Key Insight:** The 6-section template ensures nothing is forgotten. Each section serves a specific purpose in enabling autonomous execution.

---

### **Step 5: Validate the Prompt**

**Goal:** Ensure the prompt is complete, clear, and ready for execution.

**When:** After writing the prompt, before commissioning.

**Quality Checklist:**
- [ ] Is the Objective a single, clear sentence?
- [ ] Is the link to the specification correct and complete?
- [ ] Are there at least 1-2 relevant Pattern Files listed with explanations of what to learn from each?
- [ ] Are the Requirements specific, step-by-step, and unambiguous?
- [ ] Is the File Manifest complete and accurate (all files to create/modify listed)?
- [ ] Are the Success Criteria binary and testable (yes/no, no subjective judgment)?
- [ ] Are the Constraints clear about what *not* to do?
- [ ] Does the prompt reference the backend grounding document?
- [ ] If this is part of a parallel track structure, are dependencies and integration points clearly defined?
- [ ] Is the prompt self-contained (can the implementation agent execute without asking clarifying questions)?

**Output:** A validated, execution-ready implementation prompt.

**Key Insight:** The quality checklist catches gaps before commissioning. A prompt that passes the checklist has a much higher success rate.

---

### **Step 6: Commission and Monitor**

**Goal:** Commission the prompt to the appropriate agent and monitor progress.

**When:** After validation is complete.

**Actions:**
1. Save the prompt to the appropriate location (e.g., `docs/v0.0.X/track_N_implementation_prompt.md`)
2. Commission to the appropriate agent:
   - **Zenflow:** Strategic implementation (architecture, complex features)
   - **Claude Code:** Tactical implementation (UI components, refactoring)
3. Monitor progress through success criteria
4. If parallel tracks, verify completion before commissioning dependent tracks

**Output:** Commissioned work in progress, with clear success criteria for verification.

**Key Insight:** A well-written prompt enables autonomous execution. The agent should complete the work without needing clarification or back-and-forth.

---

## IV. Best Practices

### From Specification to Prompt: The Translation

The specification describes the **what** and **why**. The prompt describes the **how** and **where**. Your job is to translate strategic intent into tactical execution.

**Example Transformation:**

**Specification says:**  
> "The desktop shell should have a three-column layout with two collapsible sidebars."

**Prompt says:**  
> "Create a new component at `frontend/src/app/desktop/components/DesktopShell.tsx`. Use a CSS Grid layout with three columns: 240px (sidebar 1), 200px (sidebar 2), and 1fr (main content). Add local state for `sidebar1Collapsed` and `sidebar2Collapsed` using `useState`. Follow the layout pattern in `frontend/src/app/layout.tsx`."

### Pattern Files are Force Multipliers

Pointing the implementation agent to 2-3 existing files that demonstrate the desired structure and style is more effective than writing detailed style guides. Implementation agents learn by example.

**Good:** "Follow the component structure in `src/components/Header.tsx`"  
**Bad:** "Use functional components with TypeScript, export as default, use Tailwind for styling..."

### The File Manifest Prevents Surprises

Explicitly listing every file to be created or modified sets clear expectations and makes it easy to verify completion. It also helps identify potential merge conflicts in parallel tracks.

### Success Criteria Must Be Binary

Success criteria must be answerable with yes or no, with no subjective judgment.

**Good:** "The component renders at the /desktop route"  
**Bad:** "The component should look good"

### Constraints Prevent Scope Creep

Explicitly stating what NOT to do is as important as stating what to do. This prevents the implementation agent from "helpfully" implementing adjacent features that belong in other tracks.

**Example:** "DO NOT implement the sidebar content. This is only the shell."

### Backend Grounding Even When Not Calling APIs

Even if a track doesn't call backend APIs yet, reference the backend grounding document. This ensures the implementation agent understands future integration points and doesn't make decisions that would conflict with the backend architecture.

---

## V. Example: v0.0.31 Track 1

**Specification:** `docs/v0.0.31/v0_0_31_desktop_architecture.md`

**Prompt:** `docs/v0.0.31/v0_0_31_track_1_implementation_prompt.md`

**Key Elements:**
- **Objective:** "Replace Next.js router with React Router and create the main `DesktopShell` component"
- **Pattern Files:** 
  - `frontend/src/app/layout.tsx` (for provider setup)
  - `frontend/src/app/desktop/page.tsx` (for entry point)
- **Requirements:** 5 numbered, specific steps (install React Router, create router config, wrap app, create shell, update page)
- **File Manifest:** 2 files to create, 3 files to modify
- **Success Criteria:** 5 binary checkboxes
- **Constraints:** "DO NOT implement sidebar or main content area content"
- **Backend Grounding:** "No direct backend integration. Refer to backend integration guide for future tracks."

**Outcome:** Claude Code completed Track 1 autonomously without clarification, enabling Tracks 2-4 to start in parallel.

---

## VI. Quality Checklist

Before commissioning the prompt, ensure you can answer "yes" to all of the following:

- [ ] Have you validated that the specification is complete and implementation-ready?
- [ ] Have you identified track boundaries (if applicable) and integration points?
- [ ] Have you grounded the prompt in the existing codebase with pattern files and file paths?
- [ ] Have you written the prompt using the standard 6-section template?
- [ ] Have you validated the prompt against the quality checklist in Step 5?
- [ ] Have you referenced the backend grounding document?
- [ ] Are all success criteria binary and testable?
- [ ] Are all constraints explicit (what NOT to do)?
- [ ] Is the prompt self-contained (no missing information)?

---

## Output

- A single implementation prompt file (markdown) saved to `docs/vX.X.X/track_N_implementation_prompt.md`
- The prompt contains six sections: Objective, Context & Grounding, Detailed Requirements, File Manifest, Success Criteria, and Constraints & Non-Goals
- If the spec is multi-track, one prompt file per track, each independently executable

## Examples

**Scenario 1:** "We have a spec for the desktop shell. Write the Track 1 implementation prompt." → A prompt with a one-sentence objective, links to pattern files in `frontend/src/app/layout.tsx`, numbered step-by-step requirements for installing React Router and creating DesktopShell, a file manifest listing two new files and three modified files, and five binary success criteria.

**Scenario 2:** "Convert this backend spec section into a prompt for Claude Code." → A prompt grounding the agent in existing Go handler patterns, specifying exact API endpoint shapes, and listing `DO NOT implement authentication` as an explicit constraint to prevent scope creep into adjacent tracks.

## Edge Cases

- When the specification has not passed the pre-implementation checklist, stop and run `pre-implementation-checklist` before writing the prompt — a prompt written from an incomplete spec will produce an incomplete implementation
- When commissioning a consuming track whose producer track is not yet complete, stub the integration contract in the prompt with explicit placeholders rather than guessing the shape
- When a single spec covers more than one track, write separate prompts per track and use this skill iteratively, not once for the whole spec

## Anti-Patterns

- Writing requirements as high-level goals instead of step-by-step instructions — "add authentication" is not a requirement; "add JWT validation middleware to the `/api/v1/entities` handler in `handlers/entities.go`" is
- Omitting the File Manifest and letting the agent decide which files to create or modify — unexpected files created outside the manifest cause merge conflicts in parallel tracks
- Setting success criteria that require subjective judgment ("the UI should feel responsive") — every criterion must be answerable yes or no without the agent asking a follow-up question

---

## VII. Related Skills

- **`write-implementation-prompt`:** Foundational skill for writing implementation prompts (this skill builds on it)
- **`frontend-from-backend`:** Ensures backend architecture is documented before frontend specs
- **parallel-tracks:** Guides the decomposition of large specs into parallel tracks
- **pre-implementation-checklist:** Validates specification readiness before prompt writing
- **strategic-to-tactical-workflow:** The complete workflow that includes this skill as Phase 7

---

## VIII. The Vision

This skill is designed to create **autonomous execution**. When an implementation prompt is written following this process, the agent should be able to:

1. Understand exactly what to build
2. Know where to build it (file paths)
3. Follow existing patterns (pattern files)
4. Verify completion (success criteria)
5. Stay within boundaries (constraints)
6. Integrate cleanly (backend grounding)

The result is high-quality implementation with minimal back-and-forth, enabling parallel work and compounding velocity.
