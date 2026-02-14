# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: release-specification
description: Write comprehensive release specifications for coordinated deployments
triggers:
  - "write release specification"
  - "create release plan"
  - "document release scope"
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

# Write Release Specification Skill

**Version:** 2.1  
**Created:** 2026-02-02  
**Updated:** 2026-02-07  
**Author:** Manus AI  
**Purpose:** Write production-ready, A+ quality specifications for software releases

---

## I. The Philosophy: Specification as Contract

A specification is not documentation—it is a **contract**. It is a formal agreement between the architect and the builder about what will be created, how it will work, and what success looks like. A vague specification invites confusion, rework, and failure. A rigorous specification is an act of respect for the builder's time and an investment in quality.

This skill transforms specification writing from a creative exercise into a disciplined engineering practice. By following this structure, we create specifications that are:

- **Comprehensive:** Every question the builder might have is answered
- **Precise:** Technical details are specific, not hand-wavy
- **Actionable:** The path from specification to implementation is clear
- **Testable:** Success criteria are binary and measurable

**The standard:** 111/100 (A+). Good enough is not good enough.

---

## II. When to Use This Skill

Use this skill when:

- **Planning a new software version or release** with multiple features or components
- **Designing a complex system architecture** that requires detailed documentation
- **Commissioning work to an autonomous agent** (e.g., Claude Code, implementation agents) that needs complete context
- **Coordinating parallel development tracks** where specifications serve as contracts between teams
- **You need to communicate technical vision** to stakeholders, developers, or future maintainers

**Do not use this skill for:**
- Small bug fixes or minor tweaks (use a simple task description instead)
- Exploratory prototypes (use scouting or rapid iteration instead)
- Features that are still being actively designed (finish scouting first)

---

## III. The Workflow

### Decision Point: Full Template or Lean Format?

Before starting, determine the right spec format:

**Use the Full Template (Section IV) when:**
- The system is new or the architecture is unfamiliar
- The audience includes stakeholders who need context
- Multiple teams or agents will implement from this spec
- The risk profile is high (production-critical, user-facing)

**Use the Lean Format when:**
- The architecture is established and the audience is the implementing agent
- The scope is well-defined (single feature, clear boundaries)
- The codebase patterns are well-documented
- The implementing agent is familiar with the codebase

**Lean Format structure:** Route layouts, component tables, behavior lists. No preamble. "Sonnet level chunks" — direct, precise, implementable.

**Rule:** The receiving agent should not default to the full template. Match format to scope.

---

### Step 1: Gather Context and Inspiration

Before writing, immerse yourself in the problem space:

1. **Read previous specifications** - Study 2-3 recent specs to understand the pattern and quality bar
2. **Review the codebase** - Use `/repo-context-sync` to understand the current architecture
3. **Identify the problem** - What pain point, user need, or strategic goal is this release addressing?
4. **Scout alternatives** - Use `/strategic-scout` if you're choosing between multiple approaches

**Output:** A clear understanding of the problem, the current state, and the desired future state.

---

### Step 1.5: Run Current State Audit

Before writing the spec, measure the codebase. Specs describe the delta from measured reality, not from assumptions.

**Run these audits against the target codebase:**

#### Testing
- Test file count: `find . -name "*.test.*" | wc -l`
- Test framework: [check package.json or equivalent]
- Coverage tool: [check scripts/config]

#### Accessibility
- Aria/role instances: `grep -r "aria-\|role=" --include="*.tsx" | wc -l`
- Error boundaries: `grep -r "ErrorBoundary" --include="*.tsx" | wc -l`

#### Performance
- Memoization usage: `grep -r "React.memo\|useMemo\|useCallback" --include="*.tsx" | wc -l`
- Code splitting: `grep -r "React.lazy" --include="*.tsx" | wc -l`

#### Dependencies
- UI framework: [check package.json]
- State management: [check package.json]
- Animation library: [check package.json]

#### File Structure
- Total source files: `find src -name "*.ts" -o -name "*.tsx" | wc -l`
- Route count: [check router config]

**Include the results as a "Current State" section at the top of the spec.** The spec then describes what changes FROM this measured baseline.

**Key triggers:** "codebase audit", "audit before spec", "current state", "ground the spec"

---

### Step 2: Draft Vision and Goals

Start with the "why" before the "what":

1. **Write a compelling vision statement** - A single sentence that captures the essence of this release
2. **Explain the core insight** - 2-3 paragraphs on why this release matters
3. **Define specific, measurable goals** - What will be different after this release?
4. **List non-goals explicitly** - What is out of scope for this release?

**Output:** A clear, inspiring vision that motivates the work and sets boundaries.

---

### Step 3: Design Technical Architecture

This is the heart of the specification:

1. **Create a system overview** - How do the major components fit together?
2. **Design each component in detail** - For each major feature/component:
   - Purpose and responsibility
   - Backend implementation (Go, Python, etc.) with code examples
   - Frontend implementation (React, etc.) with code examples
   - API endpoints with request/response shapes
   - Database schema (if applicable)
   - Integration points with existing systems
   - Performance considerations
3. **Write production-ready code examples** - Not pseudocode, real code that could be committed

**Output:** A complete technical design that a skilled developer could implement without asking questions.

---

### Step 4: Plan Implementation Phases

Break the work into manageable phases:

1. **Define a phased approach** - 2-4 phases with clear focus areas
2. **Create a week-by-week breakdown** - Specific, actionable tasks for each week
3. **Identify dependencies** - What must be done before other work can start?
4. **Define the testing strategy** - Unit, integration, E2E, performance, and manual QA plans

**Output:** A realistic timeline with clear milestones and success criteria for each phase.

---

### Step 5: Assess Risks and Document

Anticipate what could go wrong:

1. **Identify major risks** - Technical, timeline, or integration risks
2. **Define mitigation strategies** - How will you reduce or eliminate each risk?
3. **Plan rollback procedures** - How will you safely undo this release if needed?
4. **Define monitoring and alerts** - How will you know if something goes wrong in production?
5. **Document user and developer documentation needs** - What needs to be written?

**Output:** A comprehensive risk assessment and contingency plan.

---

### Step 6: Review Against Checklist

Before finalizing:

1. **Run the quality checklist** (Section VI) - Ensure every item is ✅
2. **Get feedback** - Share with a peer or stakeholder
3. **Iterate** - Refine based on feedback
4. **Commit to repository** - Save in `docs/vX.X.X/` with a clear filename

**Output:** A finalized, A+ quality specification ready for implementation.

---

## IV. The A+ Specification Template

```markdown
# [Project Name] v[X.X.X]: [Memorable Tagline]

**Author:** [Your Name]  
**Status:** [Draft | Final | Approved]  
**Created:** [Date]  
**Grounded In:** [What this builds on - previous versions, research, feedback]

---

## 1. Vision

> A single, compelling sentence that captures the essence of this release.

**The Core Insight:**

[2-3 paragraphs explaining WHY this release matters, what problem it solves, and how it advances the overall vision]

**What Makes This Different:**

[2-3 paragraphs explaining what makes this approach unique, innovative, or better than alternatives]

---

## 1.5 Current State (Audit Results)

> Include measured codebase metrics here. This grounds the spec in reality.

**Testing:** [X] test files, [framework], [coverage tool]
**Accessibility:** [X] aria/role instances, [X] error boundaries
**Performance:** [X] memoization instances, [X] code splitting instances
**Dependencies:** [list key deps from package.json]
**File Structure:** [X] source files, [X] routes, [X] shared components

---

## 2. Goals & Success Criteria

**Primary Goals:**
1. [Specific, measurable goal]
2. [Specific, measurable goal]
3. [Specific, measurable goal]

**Success Criteria:**
- ✅ [Concrete, testable criterion]
- ✅ [Concrete, testable criterion]
- ✅ [Concrete, testable criterion]

**Non-Goals (Out of Scope):**
- ❌ [What this release explicitly does NOT include]
- ❌ [What is deferred to future versions]

---

## 3. Technical Architecture

### 3.1 System Overview

[High-level diagram or description of how components fit together]

**Key Components:**
1. **[Component Name]** - [Purpose and responsibility]
2. **[Component Name]** - [Purpose and responsibility]
3. **[Component Name]** - [Purpose and responsibility]

### 3.2 [Feature/Component 1] - Detailed Design

**Purpose:** [What this component does and why it's needed]

**Architecture:**

[Detailed explanation with diagrams if helpful]

**Backend Implementation (Go):**

```go
// Complete, production-ready code example
package [package_name]

type [StructName] struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

func (s *[StructName]) [MethodName]() error {
    // Implementation with error handling
    return nil
}
```

**Frontend Implementation (React/TypeScript):**

```typescript
// Complete, production-ready code example
interface [InterfaceName] {
  field1: string;
  field2: number;
}

export const [ComponentName]: React.FC<Props> = ({ prop1, prop2 }) => {
  const [state, setState] = useState<StateType>(initialState);
  
  // Implementation
  
  return (
    <div className="...">
      {/* JSX */}
    </div>
  );
};
```

**API Endpoints:**

| Method | Endpoint | Request | Response | Purpose |
|--------|----------|---------|----------|---------|
| POST | `/api/v1/[resource]` | `{ field: value }` | `{ id: string }` | [Description] |
| GET | `/api/v1/[resource]/:id` | - | `{ data: object }` | [Description] |

**Database Schema (if applicable):**

```sql
CREATE TABLE [table_name] (
    id TEXT PRIMARY KEY,
    field1 TEXT NOT NULL,
    field2 INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_[field] ON [table_name]([field]);
```

**Integration Points:**
- Integrates with [existing component] via [method]
- Depends on [existing service] for [functionality]
- Extends [existing pattern] from v[X.X.X]

**Performance Considerations:**
- [Specific optimization or constraint]
- [Caching strategy or database indexing]
- [Expected latency or throughput]

### 3.3 [Feature/Component 2] - Detailed Design

[Repeat structure for each major component]

---

## 4. Implementation Plan

### 4.1 Phased Approach

**Timeline:** [X] weeks

| Phase | Duration | Focus | Deliverables |
|-------|----------|-------|--------------|
| 1 | Week 1-2 | [Focus area] | [Specific deliverables] |
| 2 | Week 3-4 | [Focus area] | [Specific deliverables] |
| 3 | Week 5-6 | [Focus area] | [Specific deliverables] |

### 4.2 Week-by-Week Breakdown

**Week 1: [Focus]**
- [ ] Task 1: [Specific, actionable task]
- [ ] Task 2: [Specific, actionable task]
- [ ] Task 3: [Specific, actionable task]

**Success Criteria:** [What "done" looks like for this week]

**Week 2: [Focus]**
[Repeat structure]

[Continue for all weeks]

### 4.3 Dependencies & Prerequisites

**Required Before Starting:**
- ✅ [Prerequisite 1]
- ✅ [Prerequisite 2]

**Parallel Work:**
- [What can be developed simultaneously]

**Blocking Dependencies:**
- [What must be completed before other work can start]

### 4.4 Testing Strategy

**Unit Tests:**
- [Component/module to test]
- Target coverage: [X]%

**Integration Tests:**
- [Integration point to test]
- [Expected behavior]

**E2E Tests:**
- [User flow to test]
- [Success criteria]

**Performance Tests:**
- [Metric to measure]
- Target: [Specific number]

**Manual QA:**
- [Scenario to test manually]
- [Edge cases to verify]

---

## 5. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation Strategy |
|------|------------|--------|---------------------|
| [Risk description] | High/Med/Low | High/Med/Low | [Specific mitigation] |
| [Risk description] | High/Med/Low | High/Med/Low | [Specific mitigation] |

---

## 6. Rollback & Contingency

**Feature Flags:**
- `[flag_name]`: Controls [feature], default: `false`

**Rollback Procedure:**
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Monitoring & Alerts:**
- [Metric to monitor]: Alert if [condition]
- [Error rate]: Alert if > [threshold]

---

## 7. Documentation & Communication

**User-Facing Documentation:**
- [ ] Update user guide with [new feature]
- [ ] Create tutorial for [workflow]

**Developer Documentation:**
- [ ] Update API documentation
- [ ] Document new database schema
- [ ] Add code examples to README

**Release Notes:**
- [ ] Prepare changelog
- [ ] Highlight breaking changes (if any)
- [ ] Include migration guide (if needed)

---

## 8. Appendices

### 8.1 Related Work & Inspiration

- [Project/Paper]: [What we learned from it]
- [Tool/System]: [How it influenced this design]

### 8.2 Future Considerations

**v[X+1] Candidates:**
- [Feature that didn't make this release but is planned]
- [Enhancement that can build on this foundation]

### 8.3 Open Questions

- [ ] [Question that needs resolution before or during implementation]
- [ ] [Decision that can be made during development]

### 8.4 References

1. [Link to related spec]
2. [Link to research paper]
3. [Link to GitHub issue or discussion]
```

---

## V. Best Practices

### 1. Start with Vision, Not Features

**Why:** Features without vision are just a list of tasks. Vision provides meaning and direction.

**How:** Write the vision statement first. If you can't articulate why this release matters in one sentence, you're not ready to write the spec.

---

### 2. Write Production-Ready Code Examples

**Why:** Pseudocode leaves too much room for interpretation. Real code is a contract.

**How:** Write code examples that could be committed to the repository. Include imports, error handling, and types.

---

### 3. Use Realistic Timelines Based on Complexity

**Why:** Underestimating timelines leads to rushed work and technical debt.

**How:** Use past releases as benchmarks. A 1,000-line feature typically takes 1-2 weeks, not 2 days.

---

### 4. Document Integration Points Explicitly

**Why:** Most bugs happen at the boundaries between systems.

**How:** For every new component, explicitly document how it connects to existing systems (APIs, props, state, events).

---

### 5. Include Risk Mitigation from the Start

**Why:** Identifying risks after implementation is too late.

**How:** During the architecture phase, ask "What could go wrong?" and document mitigation strategies.

---

### 6. Make Success Criteria Binary and Testable

**Why:** Ambiguous success criteria lead to scope creep and endless iteration.

**How:** Every success criterion should be a yes/no question. "The user can create a new project" is testable. "The UI is intuitive" is not.

---

### 7. Reference Existing Patterns

**Why:** Consistency reduces cognitive load and makes the codebase easier to maintain.

**How:** When designing a new component, reference an existing component that follows the same pattern. "Follow the structure of `ComponentX`."

---

## VI. Quality Checklist

Before finalizing a specification, verify:

### Vision & Goals (3 questions)
- [ ] Is the vision statement a single, compelling sentence?
- [ ] Are the goals specific, measurable, and achievable?
- [ ] Are non-goals explicitly stated to prevent scope creep?

### Technical Architecture (4 questions)
- [ ] Does every major component have a detailed design with code examples?
- [ ] Are all API endpoints fully specified (method, path, request, response)?
- [ ] Are integration points with existing systems documented?
- [ ] Are performance considerations addressed?

### Implementation Plan (3 questions)
- [ ] Is the timeline realistic based on the complexity of the work?
- [ ] Does the week-by-week breakdown include specific, actionable tasks?
- [ ] Is the testing strategy comprehensive (unit, integration, E2E, performance)?

### Risk & Documentation (3 questions)
- [ ] Have major risks been identified with mitigation strategies?
- [ ] Is there a rollback procedure in case of failure?
- [ ] Are user and developer documentation needs documented?

**If you cannot answer "yes" to all 13 questions, the specification is not ready.**

---

## VII. Example: Dojo Genesis v0.0.23

**The Problem:** Users needed a way to calibrate the agent's behavior and communication style to match their preferences.

**The Vision:** "The Collaborative Calibration" - A release that transforms the agent from a fixed personality into an adaptive partner.

**What Made It A+:**

1. **Clear Vision:** The tagline "Collaborative Calibration" immediately communicated the essence
2. **Comprehensive Architecture:** Detailed design of the calibration UI, backend storage, and agent integration
3. **Production-Ready Code:** Complete Go and TypeScript examples that could be implemented directly
4. **Realistic Timeline:** 3-week phased approach with weekly milestones
5. **Risk Mitigation:** Identified the risk of "calibration drift" and defined a validation mechanism

**Key Decisions:**
- Store calibration preferences in SQLite for local-first architecture
- Use a multi-dimensional calibration model (tone, verbosity, formality)
- Implement real-time preview of calibration changes

**Outcome:** The specification was commissioned to Claude Code and implemented in 2.5 weeks with minimal rework.

---

## VIII. Common Pitfalls to Avoid

❌ **Vague Goals:** "Improve user experience" → ✅ "Reduce context loading time by 50%"  
❌ **Missing Code Examples:** High-level description only → ✅ Complete, runnable code  
❌ **Unrealistic Timelines:** "2 days for full backend" → ✅ "2 weeks with phased approach"  
❌ **No Risk Assessment:** Assumes everything will work → ✅ Identifies risks and mitigations  
❌ **Incomplete Testing:** "We'll test it" → ✅ Specific test cases and coverage targets  
❌ **No Integration Points:** Treats feature as isolated → ✅ Documents how it connects to existing system

---

## IX. Related Skills

- **`strategic-to-tactical-workflow`** - The complete workflow from scouting to implementation (this skill is Phase 6)
- **`frontend-from-backend`** - For frontend specs that need deep backend grounding
- **`implementation-prompt`** - For converting this spec into implementation prompts
- **`parallel-tracks`** - For splitting large specs into parallel execution tracks
- **`repo-context-sync`** - For gathering codebase context before writing specs
- **`memory-garden`** - For documenting learnings from implementation
- **`seed-extraction`** - For extracting reusable patterns from specs

---

## X. Skill Metadata

**Token Savings:** ~10,000-15,000 tokens per specification (no need to re-read old specs for patterns)  
**Quality Impact:** Ensures consistency across all specifications  
**Maintenance:** Update when new patterns emerge from successful releases  

**When to Update This Skill:**
- After completing 3-5 new specifications (to incorporate new patterns)
- When a specification leads to significant rework (to identify what was missing)
- When commissioning to a new type of agent (to adapt the template)

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active
