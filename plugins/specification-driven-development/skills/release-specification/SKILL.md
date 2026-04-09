---
name: release-specification
model: opus
description: Produces a production-ready release specification document (Full or Lean format) including current-state audit, technical architecture with code examples, implementation plan, and risk assessment. Use when 'write a release spec', 'create a release specification for vX.X.X', or 'ground this spec in the codebase'.
category: specification-driven-development
triggers:
  - "write a release spec"
  - "create a release specification"
  - "ground this spec in the codebase"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"
  - "bash"

inputs:
  - name: release_context
    type: string
    description: Description of what the release should accomplish (version, goals, scope)
    required: true
  - name: format
    type: string
    description: Specification format — Full, Lean, or Minimal
    required: false
outputs:
  - name: release_spec
    type: ref
    format: cas-ref
    description: Production-ready release specification document with current-state audit, architecture, implementation plan, and risk assessment
---

# Write Release Specification Skill

**Version:** 2.1  
**Created:** 2026-02-02  
**Updated:** 2026-02-07  
**Author:** Manus AI  
**Purpose:** Write production-ready, A+ quality specifications for software releases

---

## I. The Philosophy: Specification as Contract

A specification is not documentation—it is a **contract**. A vague specification invites confusion, rework, and failure. A rigorous specification is an act of respect for the builder's time.

Specifications created with this skill are:
- **Comprehensive:** Every question the builder might have is answered
- **Precise:** Technical details are specific, not hand-wavy
- **Actionable:** The path from specification to implementation is clear
- **Testable:** Success criteria are binary and measurable

**The standard:** 111/100 (A+). Good enough is not good enough.

---

## II. When to Use This Skill

**Use when:**
- Planning a new software version or release with multiple features
- Designing a complex system architecture requiring detailed documentation
- Commissioning work to an autonomous agent (Claude Code, etc.) that needs complete context
- Coordinating parallel development tracks

**Do not use for:**
- Small bug fixes or minor tweaks (use a simple task description)
- Exploratory prototypes (use scouting first)
- Features still being actively designed (finish scouting first)

---

## III. The Workflow

### Decision Point: Full Template or Lean Format?

**Use the Full Template (Section IV) when:** System is new, audience includes stakeholders, multiple teams implement, risk is high.

**Use the Lean Format when:** Architecture is established, audience is the implementing agent, scope is well-defined.

**Lean Format structure:** Route layouts, component tables, behavior lists. No preamble. "Sonnet level chunks" — direct, precise, implementable.

**Rule:** Match format to scope. Don't default to full template when lean will do.

---

### Step 1: Gather Context and Inspiration

1. **Read previous specifications** — Study 2-3 recent specs for pattern and quality bar
2. **Review the codebase** — Use `/repo-context-sync` to understand current architecture
3. **Identify the problem** — What pain point or strategic goal is this release addressing?
4. **Scout alternatives** — Use `/strategic-scout` if choosing between approaches

---

### Step 1.5: Run Current State Audit

Specs describe the delta from measured reality, not from assumptions. Run before writing.

**Testing:** `find . -name "*.test.*" | wc -l` | framework from package.json | coverage tool  
**Accessibility:** `grep -r "aria-\|role=" --include="*.tsx" | wc -l` | error boundaries count  
**Performance:** `grep -r "React.memo\|useMemo\|useCallback" --include="*.tsx" | wc -l` | lazy splits  
**File Structure:** `find src -name "*.ts" -o -name "*.tsx" | wc -l` | route count

**Include results as "Current State" section at the top of the spec.**

**Key triggers:** "codebase audit", "audit before spec", "current state", "ground the spec"

---

### Step 2: Draft Vision and Goals

1. Write a compelling vision statement (single sentence)
2. Explain the core insight (2-3 paragraphs on why this release matters)
3. Define specific, measurable goals
4. List non-goals explicitly (what is out of scope)

---

### Step 3: Design Technical Architecture

For each major component:
- Purpose and responsibility
- Backend implementation with production-ready code examples
- Frontend implementation with production-ready code examples
- API endpoints with request/response shapes
- Database schema (if applicable)
- Integration points with existing systems
- Performance considerations

Write real code, not pseudocode. A skilled developer should be able to implement without asking questions.

---

### Step 4: Plan Implementation Phases

1. Define 2-4 phases with clear focus areas
2. Create week-by-week task breakdown with specific, actionable items
3. Identify dependencies and blocking work
4. Define testing strategy: unit, integration, E2E, performance, manual QA

---

### Step 5: Assess Risks and Document

1. Identify major technical, timeline, and integration risks
2. Define mitigation strategies for each risk
3. Plan rollback procedures
4. Define monitoring and alerts
5. Document user and developer documentation needs

---

### Step 6: Review Against Checklist

Run the quality checklist (Section VI). Get feedback. Iterate. Commit to `docs/vX.X.X/`.

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
[2-3 paragraphs explaining WHY this release matters]

**What Makes This Different:**
[2-3 paragraphs on unique approach vs. alternatives]

---

## 1.5 Current State (Audit Results)

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

**Success Criteria:**
- ✅ [Concrete, testable criterion]
- ✅ [Concrete, testable criterion]

**Non-Goals (Out of Scope):**
- ❌ [What this release explicitly does NOT include]

---

## 3. Technical Architecture

### 3.1 System Overview
[High-level description of how components fit together]

**Key Components:**
1. **[Component Name]** - [Purpose and responsibility]
2. **[Component Name]** - [Purpose and responsibility]

### 3.2 [Feature/Component 1] - Detailed Design

**Purpose:** [What this component does and why it's needed]

**Backend Implementation (Go):**

```go
package [package_name]

type [StructName] struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

func (s *[StructName]) [MethodName]() error {
    return nil
}
```

**Frontend Implementation (React/TypeScript):**

```typescript
interface [InterfaceName] {
  field1: string;
  field2: number;
}

export const [ComponentName]: React.FC<Props> = ({ prop1, prop2 }) => {
  return <div className="...">{/* JSX */}</div>;
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_[field] ON [table_name]([field]);
```

**Integration Points:**
- Integrates with [existing component] via [method]
- Extends [existing pattern] from v[X.X.X]

**Performance Considerations:**
- [Caching strategy or database indexing]
- [Expected latency or throughput]

### 3.3 [Feature/Component 2] - Detailed Design
[Repeat structure for each major component]

---

## 4. Implementation Plan

### 4.1 Phased Approach

| Phase | Duration | Focus | Deliverables |
|-------|----------|-------|--------------|
| 1 | Week 1-2 | [Focus area] | [Specific deliverables] |
| 2 | Week 3-4 | [Focus area] | [Specific deliverables] |

### 4.2 Week-by-Week Breakdown

**Week 1: [Focus]**
- [ ] Task 1: [Specific, actionable task]
- [ ] Task 2: [Specific, actionable task]

**Success Criteria:** [What "done" looks like for this week]

[Continue for all weeks]

### 4.3 Dependencies & Prerequisites

**Required Before Starting:** ✅ [Prerequisite 1]  
**Parallel Work:** [What can be developed simultaneously]  
**Blocking:** [What must complete before other work starts]

### 4.4 Testing Strategy

**Unit:** [Component to test] — Target coverage: [X]%  
**Integration:** [Integration point to test]  
**E2E:** [User flow to test]  
**Performance:** [Metric] — Target: [Number]  
**Manual QA:** [Scenario] — [Edge cases]

---

## 5. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation Strategy |
|------|------------|--------|---------------------|
| [Risk description] | High/Med/Low | High/Med/Low | [Specific mitigation] |

---

## 6. Rollback & Contingency

**Feature Flags:** `[flag_name]`: Controls [feature], default: `false`

**Rollback Procedure:** 1. [Step 1] 2. [Step 2] 3. [Step 3]

**Monitoring & Alerts:** [Metric]: Alert if [condition]

---

## 7. Documentation & Communication

- [ ] Update user guide with [new feature]
- [ ] Update API documentation
- [ ] Prepare changelog with breaking changes and migration guide (if needed)

---

## 8. Appendices

### 8.1 Future Considerations
**v[X+1] Candidates:** [Features deferred to next release]

### 8.2 Open Questions
- [ ] [Question needing resolution before or during implementation]

### 8.3 References
1. [Link to related spec]
2. [Link to GitHub issue or discussion]
```

---

## V. Best Practices

| Practice | Why | How |
|----------|-----|-----|
| Start with vision, not features | Features without vision are a list of tasks | Write the vision statement first — if you can't state why in one sentence, you're not ready |
| Write production-ready code examples | Pseudocode leaves room for misinterpretation | Code that could be committed; include types and error handling |
| Use realistic timelines | Underestimating leads to rushed work and debt | Use past releases as benchmarks; 1,000-line feature = 1-2 weeks |
| Document integration points explicitly | Most bugs happen at system boundaries | For every component, document how it connects to existing systems |
| Include risk mitigation from the start | Identifying risks after implementation is too late | Ask "What could go wrong?" during architecture phase |
| Make success criteria binary and testable | Ambiguous criteria lead to scope creep | "User can create a project" ✅ vs. "UI is intuitive" ❌ |
| Reference existing patterns | Consistency reduces cognitive load | "Follow the structure of `ComponentX`" |

---

## VI. Quality Checklist

Before finalizing a specification, verify:

**Vision & Goals**
- [ ] Vision statement is a single, compelling sentence
- [ ] Goals are specific, measurable, and achievable
- [ ] Non-goals explicitly stated to prevent scope creep

**Technical Architecture**
- [ ] Every major component has detailed design with code examples
- [ ] All API endpoints fully specified (method, path, request, response)
- [ ] Integration points with existing systems documented
- [ ] Performance considerations addressed

**Implementation Plan**
- [ ] Timeline is realistic based on complexity
- [ ] Week-by-week breakdown has specific, actionable tasks
- [ ] Testing strategy is comprehensive (unit, integration, E2E, performance)

**Risk & Documentation**
- [ ] Major risks identified with mitigation strategies
- [ ] Rollback procedure defined
- [ ] User and developer documentation needs documented

**If you cannot answer "yes" to all 13 questions, the specification is not ready.**

---

## VII. Common Pitfalls to Avoid

❌ **Vague Goals:** "Improve user experience" → ✅ "Reduce context loading time by 50%"  
❌ **Missing Code Examples:** High-level description only → ✅ Complete, runnable code  
❌ **Unrealistic Timelines:** "2 days for full backend" → ✅ "2 weeks with phased approach"  
❌ **No Risk Assessment:** Assumes everything will work → ✅ Identifies risks and mitigations  
❌ **Incomplete Testing:** "We'll test it" → ✅ Specific test cases and coverage targets  
❌ **No Integration Points:** Treats feature as isolated → ✅ Documents how it connects to existing system

---

## VIII. Related Skills

- **`strategic-to-tactical-workflow`** — Complete workflow from scouting to implementation (this skill is Phase 6)
- **`frontend-from-backend`** — For frontend specs needing deep backend grounding
- **`implementation-prompt`** — For converting this spec into implementation prompts
- **`parallel-tracks`** — For splitting large specs into parallel execution tracks
- **`repo-context-sync`** — For gathering codebase context before writing specs
- **`memory-garden`** — For documenting learnings from implementation

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active

---

## Output

- A release specification document saved to `docs/vX.X.X/[release]_specification.md`
- Includes a "Current State (Audit Results)" section with measured counts, not estimates
- Technical architecture section with production-ready Go and TypeScript code examples, API endpoint tables, and database schema
- A complete quality checklist appended confirming all 13 readiness criteria are met before commissioning

## Examples

**Scenario 1:** "Write a spec for DojoGenesis v0.2.5 adding the channel bridge and WebSocket hub." → A Full-format spec with a current-state audit measuring existing test files and API endpoints, detailed design sections for each new component with Go struct definitions and TypeScript interfaces, a two-week phased implementation plan, and a risk table covering WebSocket connection stability.

**Scenario 2:** "Spec the new entity CRUD feature for the v0.3.0 release." → A Lean-format spec with a route layout table (GET/POST/PUT/DELETE for `/api/v1/entities`), a component table mapping each UI widget to its handler, behavior list, test cases, and rollout notes — no preamble, ready to commission.

## Edge Cases

- When scouting is not complete and the architecture is still undecided, stop and complete the scout before writing the spec — a spec written from open architecture questions will need to be substantially rewritten
- When the spec covers work for parallel tracks, write a master release spec first and then use `spec-constellation-to-prompt-suite` to derive the per-track prompts
- When the codebase has changed since the previous spec, run Step 1.5 (current state audit) against the latest main branch — stale metrics produce grounding errors

## Anti-Patterns

- Writing the spec from memory of the codebase instead of running the actual audit commands — even a 5-day-old mental model will have drifted; always run the grep commands
- Skipping "Non-Goals" — implementation agents will fill undefined scope with adjacent features; explicit non-goals prevent this
- Marking a spec "Final" before running the 13-item quality checklist — the checklist exists precisely because authors miss items when reviewing their own work
