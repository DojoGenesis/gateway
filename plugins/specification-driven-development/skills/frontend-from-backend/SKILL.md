---
name: frontend-from-backend
model: opus
description: Produces a backend-grounded frontend specification document covering API contracts, integration guide, auth flows, and updated track prompts. Use when 'spec the frontend for this API', 'write frontend spec from backend', or 'ground the UI in this existing backend'.
category: specification-driven-development

inputs:
  - name: backend_spec
    type: string
    description: Path or content of the backend API specification or code to derive from
    required: true
outputs:
  - name: frontend_spec
    type: ref
    format: cas-ref
    description: Backend-grounded frontend specification document covering API contracts, integration guide, auth flows, and track prompts
---

# Write Frontend Spec From Backend Skill

**Version:** 1.0
**Created:** 2026-02-07
**Author:** Manus AI
**Purpose:** To provide a structured, repeatable process for writing high-quality frontend specifications that are deeply integrated with an existing backend, ensuring seamless development and reducing integration friction.

---

## I. The Philosophy: Grounding Before Building

Frontend development in a full-stack application does not happen in a vacuum. The most common source of bugs, delays, and rework is a disconnect between the frontend implementation and the backend reality. This skill is built on a simple but powerful principle: **grounding before building**.

By deeply understanding the existing backend architecture, APIs, and data models *before* writing a single line of frontend specification, we can prevent entire classes of integration problems. This process transforms specification writing from a creative exercise into a disciplined engineering practice, ensuring that what we design is not just beautiful, but buildable.

---

## II. When to Use This Skill

-   **When planning a new frontend feature** that will interact with an existing backend.
-   **When writing specifications for a UI redesign** of a feature that has a backend component.
-   **When commissioning frontend work to an autonomous agent** like Claude Code, Zenflow, or other implementation agents.
-   **When you feel a disconnect** between the frontend vision and the backend reality.
-   **At the beginning of any major frontend development cycle**.

---

## III. The Workflow

This workflow is a 5-step process that takes you from a high-level feature idea to a production-ready, backend-grounded specification.

### Step 1: Deep Backend Analysis

**Goal:** Achieve a comprehensive understanding of the existing backend architecture.

1.  **Run `/repo-context-sync`:** Generate a comprehensive context map of the repository.
2.  **Read Key Backend Files:**
    -   `main.go` (or equivalent): Understand route registration and server setup.
    -   `handlers/`: Read the handlers for the relevant feature areas.
    -   `middleware/`: Understand the authentication and request lifecycle.
3.  **Document APIs:** Map all relevant API endpoints, including methods, authentication requirements, request bodies, and success/error responses.
4.  **Identify Integration Points:** For each part of the new frontend feature, identify the specific backend endpoint it will interact with.

### Step 2: Comprehensive Feature Specification

**Goal:** Write a production-ready specification for the new feature that leverages the existing backend.

1.  **Write a Full Feature Spec:** Create a detailed document covering:
    -   Executive summary and problem statement
    -   Goals, non-goals, and user stories
    -   Technical architecture (how frontend and backend will interact)
    -   UI/UX wireframes and interaction flows
    -   API contracts with request/response examples
    -   Implementation plan (if multi-phased)
    -   Security considerations
2.  **Leverage Existing Backend:** Design the feature to use existing backend infrastructure wherever possible. Avoid proposing new backend features unless absolutely necessary.

### Step 3: Integration Guide Creation

**Goal:** Create a practical guide for developers on how to wire the new frontend to the backend.

1.  **Create a Track-by-Track Guide:** If the feature is being built in parallel tracks, create a guide for each track.
2.  **Document Authentication Flow:** Explain how the frontend should handle authentication (guest mode, API key mode, cloud sync).
3.  **Explain Streaming Architecture:** If the feature uses streaming, document the SSE or WebSocket connection and event handling.
4.  **Provide Code Examples:** Include frontend code snippets for making API calls.
5.  **Document Error Handling:** Specify how the frontend should handle different backend error codes.

### Step 4: Track Prompt Enhancement

**Goal:** Update all development prompts (e.g., for implementation agents like Zenflow or Claude Code) with backend grounding.

1.  **Add a "Backend Grounding" Section:** In each prompt, add a dedicated section that explains the backend context.
2.  **Document Specific Endpoints:** For each part of the implementation, specify the exact API endpoint to use.
3.  **Reference the Integration Guide:** Link to the full integration guide for more details.
4.  **Ensure Prompts Use Existing Patterns:** Explicitly instruct the development agent to follow existing backend patterns.

### Step 5: Audit and Deliver

**Goal:** Ensure all documentation is complete, consistent, and ready for commissioning.

1.  **Run a Final Audit:** Review all documents for gaps, inconsistencies, or missing information.
2.  **Write Missing Documentation:** If any gaps are found (e.g., design system, API contracts), write the missing documents.
3.  **Push to Repository:** Commit all documentation to a dedicated directory (e.g., `docs/vX.X.X/`).
4.  **Confirm Readiness:** Announce that the specifications are complete and ready for commissioning.

---

## IV. Best Practices

-   **No Backend Changes is the Goal:** The primary goal of this process is to build a frontend that works with the *existing* backend. Only propose backend changes as a last resort.
-   **The Backend is the Source of Truth:** If there is a discrepancy between the frontend design and the backend API, the backend API is correct. The frontend design must adapt.
-   **Over-Document the Integration:** It is better to provide too much detail on how the frontend and backend should connect than too little.
-   **Use this Skill Before Writing Code:** This process should be completed *before* any significant frontend development begins.

---

## V. Quality Checklist

Before commissioning the work, ensure you can answer "yes" to all of the following questions:

-   [ ] Have you read the main entrypoint of the backend application?
-   [ ] Have you read the handlers for all relevant API endpoints?
-   [ ] Have you written a comprehensive feature specification that includes API contracts?
-   [ ] Have you created a backend integration guide with code examples?
-   [ ] Have all development prompts been updated with a "Backend Grounding" section?
-   [ ] Have you audited all documentation for completeness and pushed it to the repository?

---

## Output

- A frontend feature specification document grounded in actual backend endpoints and data models (saved to `docs/vX.X.X/frontend_spec.md`)
- A backend integration guide with code examples for each API call the frontend makes
- Updated implementation prompts for all affected tracks, each containing a "Backend Grounding" section
- An audit summary confirming no new backend routes are required

## Examples

**Scenario 1:** "We have a working Go API and need to spec the settings page for it." → A feature spec mapping each settings field to the exact backend endpoint, with TypeScript fetch examples and error-handling patterns drawn from the actual handler code.

**Scenario 2:** "The agent keeps building frontend that doesn't match the API." → A backend integration guide extracted from `handlers/` and `middleware/`, then injected into each track prompt so agents reference real endpoint signatures instead of assumptions.

## Edge Cases

- When the backend has not been built yet, this skill should not be invoked — use `specification-writer` to design both layers together instead
- When backend APIs are behind an authentication layer, document the full auth handshake in the integration guide before speccing any authenticated endpoints
- When the backend uses streaming (SSE or WebSocket), add a dedicated "Streaming Architecture" section; generic fetch patterns are not sufficient

## Anti-Patterns

- Proposing new backend endpoints inside a frontend spec — the goal is to build a frontend against the existing backend; backend changes belong in a separate release spec
- Writing frontend specs from API documentation instead of reading actual handler code — docs drift; the handler is ground truth
- Skipping the integration guide and embedding API details only in the track prompt — the guide is the canonical reference; prompts should link to it, not duplicate it inline
