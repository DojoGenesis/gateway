---
name: context-ingestion
description: A skill for general-purpose planning grounded in uploaded files. Creates deeply informed plans from specs, docs, code, research, or images. Includes intent-based routing to select the right planning mode automatically.
triggers:
  - "create a plan from these uploaded files"
  - "ingest this context and build an action plan"
  - "ground a plan in these specs and docs"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - strategic-agent
    - implementation-agent
---

# Context Ingestion Skill

**Version:** 1.0
**Author:** Tres Pies Design
**Purpose:** Create plans that are deeply informed by uploaded content, ensuring that recommendations are specific, actionable, and aligned with the provided context. Includes intent-based routing for automatic mode selection.

---

## I. The Philosophy: Grounding is Everything

The quality of a plan is directly proportional to how well it is grounded in the available context. This skill transforms file uploads from passive attachments into active participants in the planning process, ensuring that every recommendation is rooted in the specifics of the provided files.

**Core principles:**
- Every recommendation must reference a specific file, function, or section
- Constraints found in files must be listed explicitly
- Contradictions between files must be flagged, not silently resolved
- Plans must be actionable without needing additional context

---

## II. When to Use This Skill

- When you have 1-2 files and a general planning request (e.g., "create a plan from this spec")
- When you need to refactor a codebase and have uploaded the relevant files
- When you want to create action items from meeting notes or a design document
- When the intent-based router selects this mode
- As the default mode when file types or intent are ambiguous

---

## III. Intent-Based Routing

Before executing the context ingestion workflow, analyze the user's request to determine if another mode is a better fit.

### Routing Table

| File Types & Quantity | Intent Keywords | Selected Mode |
|---|---|---|
| 1-2 files (any type) | "plan", "refactor", "next steps", "action items" | **Context Ingestion** (this skill) |
| Spec, requirements doc | "spec", "prompt", "implement" | Route to `/write-spec` or `/write-prompt` |
| 3+ research files | "synthesize", "research", "patterns", "compare" | **Research Synthesis** |
| Ambiguous | (default) | **Context Ingestion** (this skill) |

### Routing Behavior

- **If routing to another command:** Explain the routing decision and hand off. Example: "You uploaded a spec file and asked for an implementation prompt — routing to `/write-prompt`."
- **If staying in context ingestion:** Proceed with the 5-step workflow below.
- **User override:** If the router picks the wrong mode, the user can invoke the correct command directly.

---

## IV. The 5-Step Workflow

### Step 1: File Ingestion and Cataloging

**Goal:** Read all uploaded files and create a structured catalog.

1. **Read files:** Read each uploaded file completely
2. **Extract content:** For code, identify structure (functions, classes, exports). For docs, identify sections and key decisions. For images, describe visual content.
3. **Create catalog:** Build an internal catalog listing all files and their key content:

```markdown
## File Catalog

### File 1: `feature_spec.md`
- **Type:** Specification
- **Key Content:** Defines authentication feature with OAuth2 flow
- **Entities:** UserAuth interface, /api/v1/auth endpoint, token refresh flow
- **Constraints:** Must support SSO, must not break existing session management

### File 2: `auth_handler.go`
- **Type:** Backend code
- **Key Content:** Current authentication handler implementation
- **Entities:** AuthHandler struct, LoginHandler func, ValidateToken func
- **Patterns:** Uses middleware pattern, returns JSON responses
```

### Step 2: Context Synthesis

**Goal:** Synthesize file content into a coherent understanding of the current state.

1. **Identify patterns:** Recurring themes, architectural decisions, coding conventions across files
2. **Extract constraints:** Both explicit ("must maintain backward compatibility") and implicit (existing patterns that shouldn't be broken)
3. **Identify opportunities:** Areas for improvement or refactoring
4. **Note contradictions:** If files contradict each other, flag it explicitly:

```markdown
### Contradiction Detected
- `spec.md` line 45 states: "All endpoints must use JWT authentication"
- `auth_handler.go` line 12 uses: Session-based authentication
- **Impact:** Migration strategy needed before implementation
```

### Step 3: Plan Creation

**Goal:** Create a detailed plan grounded in the synthesized context.

1. **Define phases:** Break the plan into clear phases with estimated durations
2. **Specify actions:** For each phase, list specific actions grounded in the uploaded files:
   - Reference file names and line numbers
   - Reference specific functions, types, or sections
3. **Define deliverables:** Concrete, tangible outputs for each phase
4. **Set success criteria:** Binary, testable criteria (pass/fail)
5. **Identify risks:** Potential problems and their mitigations

### Step 4: Validation

**Goal:** Ensure the plan is complete, grounded, and actionable.

Run these checks:

- [ ] **Completeness:** All phases have deliverables and success criteria
- [ ] **Grounding:** All recommendations are tied to specific files (no ungrounded suggestions)
- [ ] **Actionability:** The plan can execute without needing additional context beyond the uploaded files
- [ ] **Constraints:** All constraints found in files are listed explicitly in the plan
- [ ] **Contradictions:** Any contradictions between files are flagged and addressed

### Step 5: Delivery

**Goal:** Deliver the plan and offer refinement.

1. **Save plan:** As `[date]_plan_[topic].md` in the workspace
2. **Summarize:** Briefly present key phases and deliverables
3. **Explain routing:** Tell the user why this mode was selected
4. **Offer refinement:** Ask if any phases need adjustment

---

## V. Plan Document Template

```markdown
# Plan: [Topic]

**Created:** [Date]
**Source Files:** [List of uploaded files]
**Mode:** Context Ingestion (routed because: [reason])

---

## File Catalog

[Internal catalog from Step 1]

---

## Constraints

[All constraints extracted from files]

1. [Constraint from file X, line Y]
2. [Constraint from file Z, line W]

---

## Contradictions (if any)

[Any contradictions between files, flagged explicitly]

---

## Plan

### Phase 1: [Name] — [Duration estimate]

**Actions:**
1. [Action referencing file:line]
2. [Action referencing file:function]

**Deliverables:**
- [Concrete output]

**Success Criteria:**
- [ ] [Binary criterion]

### Phase 2: [Name] — [Duration estimate]

[Repeat structure]

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| [Description] | High/Med/Low | High/Med/Low | [Strategy] |
```

---

## VI. Best Practices

- **Reference specifics:** Always reference specific files, functions, or sections in the plan. Never make ungrounded recommendations.
- **Make constraints explicit:** If you find constraints in the files, list them prominently in the plan.
- **Flag contradictions:** Files that disagree are a planning hazard. Surface them, don't hide them.
- **Write actionable plans:** Clear phases, concrete deliverables, binary success criteria.
- **Synthesize before planning:** Create the internal file catalog and context synthesis before writing the plan.

---

## VII. Quality Checklist

- [ ] Does the plan reference specific files and line numbers?
- [ ] Does the plan explicitly list all constraints found in the files?
- [ ] Are contradictions between files flagged and addressed?
- [ ] Does the plan have clear phases with concrete deliverables?
- [ ] Are success criteria binary and testable?
- [ ] Is the plan actionable without needing additional context?
- [ ] Was the routing decision explained to the user?
