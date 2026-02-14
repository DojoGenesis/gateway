# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: pre-implementation-checklist
description: Verify readiness before starting implementation work
triggers:
  - "check implementation readiness"
  - "verify pre-implementation"
  - "validate before coding"
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

# Pre-Implementation Checklist Skill

**Version:** 1.1  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a final quality gate before handing a specification to an autonomous development agent, ensuring all prerequisites are met for a successful implementation.

---

## I. The Philosophy of Preparing the Way

Handing a specification to an autonomous implementation agent is an act of trust. It is a request for a great work to be done. The Pre-Implementation Checklist is an **act of care** that precedes this request. It is the ritual of preparing the way, of ensuring that the path for the builder is clear, the materials are ready, and the destination is unambiguous.

By completing this checklist, we honor the builder's time and energy. We minimize the risk of confusion, rework, and failure. We transform the handoff from a hopeful toss over the wall into a deliberate and respectful commissioning.

---

## II. When to Use This Skill

-   **Always** use this skill as the final step before committing a `Final` specification document.
-   Use it as a final review gate before commissioning an implementation agent.
-   Use it to sanity-check a development plan before execution begins.

---

## III. The Checklist Workflow

### Step 0: Track 0 — Pre-Commission Alignment Check

Before appending the quality checklist, run a pre-commission alignment check. This catches spec-to-codebase drift that accumulates between when a spec was written and when implementation begins.

**When to run:** Always. Before every commission. Especially when there is a time gap between writing the spec and commissioning the work, or when the codebase has changed since the spec was authored.

**The Track 0 process:**

1. **Read the actual codebase** — not assumptions from the spec. Run `repo-context-sync` or manually inspect:
   - Types and interfaces that the spec references
   - API endpoints and their current signatures
   - File structure and import patterns
   - Existing test infrastructure

2. **Read the implementation prompts** — compare what the prompt assumes exists vs. what actually exists.

3. **Identify every mismatch** — between spec assumptions and codebase reality. Common mismatches:
   - Types referenced in the spec that don't exist yet
   - APIs the spec assumes that have different signatures
   - File paths or import patterns that have changed
   - Dependencies the spec assumes that aren't installed

4. **Fix mismatches** — Write a focused Track 0 remediation prompt that creates the missing infrastructure before parallel work begins. Track 0 runs sequentially; parallel tracks follow only after Track 0 is complete.

**Evidence:** In the v0.2.x era, Track 0 cost ~1 day but saved 3-5 days of debugging spec-to-code drift during parallel implementation.

**Key triggers:** "pre-commission check", "track 0", "verify before commissioning", "spec-to-code alignment"

### Step 1: Append the Checklist to the Specification

Once a specification is considered feature-complete and ready for final review, copy the checklist template from Section IV of this skill and append it to the very end of the specification document.

### Step 2: Collaboratively Review Each Item

Go through each item on the checklist. For each item, confirm that it has been addressed in the specification. Mark it with a `✅`.

-   If an item is not addressed, pause and update the specification accordingly.
-   If an item is not applicable, mark it as `N/A`.

### Step 3: Do Not Proceed Until All Items are Checked

The checklist is a gate. Do not proceed with the handoff to the implementation agent until every applicable item is marked with a `✅`. This discipline is crucial for maintaining a high standard of quality and ensuring successful autonomous implementation.

### Step 4: Commission the Implementation Agent

Once the checklist is complete, you can confidently commission the implementation agent, linking to the now-verified specification document.

---

## IV. Pre-Implementation Checklist Template

```markdown
---

## X. Pre-Implementation Checklist

**Instructions:** Before handing this specification to the implementation agent, ensure every item is checked. Do not proceed until the checklist is complete.

### 1. Vision & Goals

-   [ ] **Clarity of Purpose:** The "Vision" and "Core Insight" are clear, concise, and unambiguous.
-   [ ] **Measurable Goals:** The "Primary Goals" are specific and measurable.
-   [ ] **Testable Success Criteria:** The "Success Criteria" are concrete and can be objectively tested.
-   [ ] **Scope is Defined:** The "Non-Goals" clearly define what is out of scope for this release.

### 2. Technical Readiness

-   [ ] **Architecture is Sound:** The "Technical Architecture" is well-defined, and diagrams are clear.
-   [ ] **Code is Production-Ready:** All code examples are complete, correct, and follow existing patterns.
-   [ ] **APIs are Specified:** All API endpoints are fully specified (Method, Endpoint, Request, Response).
-   [ ] **Database Schema is Final:** The database schema is complete, including tables, columns, types, and indexes.
-   [ ] **Dependencies are Met:** All prerequisites and dependencies on other systems are identified and resolved.

### 3. Implementation Plan

-   [ ] **Plan is Actionable:** The week-by-week breakdown consists of specific, actionable tasks.
-   [ ] **Timeline is Realistic:** The timeline has been reviewed and is considered achievable.
-   [ ] **Testing Strategy is Comprehensive:** The testing strategy covers unit, integration, and E2E tests with clear targets.

### 4. Risk & Quality

-   [ ] **Risks are Mitigated:** Major risks have been identified, and clear mitigation strategies are in place.
-   [ ] **Rollback Plan is Clear:** A clear, step-by-step rollback procedure is documented.
-   [ ] **Feature Flags are Defined:** Necessary feature flags for a safe rollout are specified.

### 5. Handoff

-   [ ] **Final Review Complete:** This checklist has been fully completed and reviewed.
-   [ ] **Specification is Final:** The document status is marked as "Final".
-   [ ] **Implementation Ready:** You are now ready to commission the implementation agent.

### 0. Track 0 — Pre-Commission Alignment

-   [ ] **Codebase Verified:** The actual codebase has been read (not assumed) and matches the spec's assumptions.
-   [ ] **Types Verified:** All types, interfaces, and data structures referenced in the spec exist in the codebase.
-   [ ] **APIs Verified:** All API endpoints referenced have the expected signatures.
-   [ ] **File Structure Verified:** File paths and import patterns match expectations.
-   [ ] **Remediation Complete:** Any mismatches have been fixed via a Track 0 remediation prompt before parallel tracks begin.
```

---

## V. Best Practices

-   **Be Rigorous:** Do not check an item if it is only partially complete. The goal is 100% readiness.
-   **Treat it as a Conversation:** Use the checklist as a final opportunity to discuss any lingering uncertainties.
-   **Empower the Gatekeeper:** The agent responsible for the handoff (typically Manus) is empowered to halt the process if the checklist is not complete.
-   **Adapt as Needed:** If you find that items are consistently `N/A` or that new checks are needed, propose an update to this skill.
