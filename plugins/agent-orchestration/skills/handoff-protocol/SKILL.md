---
name: handoff-protocol
model: sonnet
description: "Produces a structured handoff package (markdown document) containing objective, required context file list, task definition, definition of done, constraints, and next steps for the receiving agent. Use when: 'hand off to the next agent', 'prepare this for handoff', 'is this handoff package complete', 'ensure nothing is lost in the transition'."
category: agent-orchestration
triggers:
  - "hand off to the next agent"
  - "prepare this for handoff"
  - "is this handoff package complete"
  - "ensure nothing is lost in the transition"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: objective
    type: string
    description: The objective being handed off to the receiving agent
    required: true
  - name: context_files
    type: string[]
    description: List of context file paths the receiving agent needs
    required: false
outputs:
  - name: handoff_package
    type: string
    description: Structured handoff markdown document with objective, context files, task definition, definition of done, and constraints
---

# Agent Handoff Protocol Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a clear, repeatable protocol for handing off work between agents (e.g., Manus to implementation agents, Manus to Cipher), ensuring no loss of context and a high probability of success.

---

## I. The Philosophy: The Sacred Relay

In a multi-agent ecosystem, a handoff is a **sacred relay**. It is the moment where one agent passes the baton of responsibility to another. A fumbled handoff leads to dropped context, wasted effort, and a broken workflow. A clean handoff is a moment of trust and shared understanding, where the receiving agent is not just given a task, but is fully equipped to carry it forward.

This protocol transforms the handoff from a hopeful transfer into a rigorous, verifiable exchange. It is an act of respect for the receiving agent, acknowledging that their ability to succeed is entirely dependent on the quality of the context they are given.

---

## II. When to Use This Skill

-   **Always** use this skill when one agent's work is complete and another agent must begin the next phase.
-   **Manus → Implementation Agent:** When a specification is complete and ready for implementation.
-   **Manus → Cipher:** When research is complete and a creative or divergent perspective is needed.
-   **Any Agent → Any Agent:** Whenever responsibility for a task is transferred.

---

## III. The Handoff Workflow

### Step 1: Prepare the Handoff Package

The sending agent is responsible for preparing a complete and self-contained "handoff package." This is a markdown document that contains everything the receiving agent needs to know. Use the template from Section IV.

### Step 2: Verify the Package with the Checklist

Before initiating the handoff, the sending agent must verify the package against the Handoff Checklist in Section V. This is a self-assessment to ensure quality.

### Step 3: Initiate the Handoff

The sending agent initiates the handoff by creating a task for the receiving agent, with the handoff package as the primary input.

**Example Handoff Task:**
-   **Agent:** Implementation Agent (Zenflow, Claude Code, etc.)
-   **Task:** Implement the v0.0.26 Breadcrumb feature.
-   **Input:** `handoffs/v0.0.26/01_breadcrumb_handoff.md`

### Step 4: The Receiving Agent's Acceptance

The receiving agent's first action is to read the handoff package and confirm that it has everything it needs to proceed. If the package is incomplete, the receiving agent must immediately return the task to the sender with a request for more information.

**Example Acceptance:** "Handoff package received and verified. I have all necessary context to proceed with the implementation. Work will now begin."

**Example Rejection:** "Handoff package is incomplete. The link to the specification is broken. Please correct and resubmit."

---

## IV. Handoff Package Template

```markdown
# Agent Handoff Package

**From:** [Sending Agent Name]
**To:** [Receiving Agent Name]
**Date:** [YYYY-MM-DD]
**Subject:** Handoff of [Task/Project Name]

---

## 1. Objective

> [A single, clear sentence describing the receiving agent's primary goal.]

---

## 2. Required Context

[A comprehensive list of all files, documents, and resources the receiving agent MUST read before starting work. Use full paths.]

**Core Documents:**
-   **Specification:** `[path/to/specification.md]`
-   **Status:** `[path/to/STATUS.md]`

**Key Conversation Summaries:**
-   `[path/to/conversation_summary_1.md]`
-   `[path/to/conversation_summary_2.md]`

**Relevant Seeds or Skills:**
-   `[path/to/seed.md]`
-   `[path/to/skill.md]`

**Pattern Files / Code Examples:**
-   `[path/to/pattern_file_1.tsx]`
-   `[path/to/pattern_file_2.go]`

---

## 3. Task Definition

[A clear and unambiguous definition of the task to be performed. If this is an implementation agent handoff, this section should contain the full implementation prompt.]

---

## 4. Definition of Done

[How will we know the receiving agent's work is complete? This must be a list of binary, testable success criteria.]

-   [ ] [Success Criterion 1]
-   [ ] [Success Criterion 2]
-   [ ] [Success Criterion 3]

---

## 5. Constraints & Boundaries

[What are the explicit constraints on the receiving agent's work?]

-   **DO NOT** modify files outside of the specified scope.
-   **DO NOT** make architectural decisions without consulting [Sending Agent Name].
-   **MUST** follow the patterns established in the provided pattern files.

---

## 6. Next Steps (After Completion)

[What happens after the receiving agent is done? Who takes the next handoff?]

-   Upon completion, notify [Next Agent Name] and hand off the results for the next phase (e.g., QA, documentation).
```

---

## V. Handoff Checklist (for Sending Agent)

-   [ ] **Is the Objective a single, clear sentence?**
-   [ ] **Are all links in the Required Context section valid and pointing to the correct files?**
-   [ ] **Is the Task Definition unambiguous and complete?**
-   [ ] **Is the Definition of Done a list of testable, binary criteria?**
-   [ ] **Are the Constraints clear and explicit?**
-   [ ] **Is the next step after completion clearly defined?**

---

## VI. Best Practices

-   **No Implicit Context:** If it's not in the handoff package, it doesn't exist. Never assume the receiving agent knows something.
-   **Over-communicate:** It is better to provide too much context than too little.
-   **The Receiver is the Gatekeeper:** Empower the receiving agent to reject incomplete handoffs. This maintains quality across the ecosystem.
-   **Standardize Handoff Locations:** Create a `handoffs/` directory in each project to store these packages, creating a clear audit trail.

## Output

- A `handoffs/` directory entry containing the completed handoff package as a markdown file (e.g., `handoffs/v0.0.26/01_feature_handoff.md`)
- The package includes: objective sentence, required context file list with full paths, task definition, definition-of-done checklist, explicit constraints, and next-steps attribution
- A completed self-assessment against the Handoff Checklist (Section V)

## Examples

**Scenario 1:** Specification writing is complete and an implementation agent must build the feature → skill produces a handoff package at `handoffs/v0.0.26/breadcrumb_handoff.md` with the spec path, pattern file paths, definition of done as binary checkboxes, and a constraint list preventing architectural changes.

**Scenario 2:** Research phase is complete and a creative agent must synthesize findings → skill produces a handoff package naming all research files, framing the synthesis objective in one sentence, and defining what a complete synthesis looks like.

## Edge Cases

- If the receiving agent is unknown at handoff time, leave the "To" field as `[TBD]` and note that the package will be routed by the orchestrator — do not hold the package waiting for that information.
- If required context files do not yet exist (e.g., a spec that is still being written), flag them as `[PENDING — do not begin until available]` rather than omitting them from the list.

## Anti-Patterns

- Writing the handoff package as a prose summary rather than a structured template — the receiving agent cannot programmatically check what it has and what it is missing.
- Marking the handoff complete before verifying all links in the Required Context section resolve to actual files — broken paths silently block the receiving agent.
