# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: specification-writer
description: Write clear, actionable specifications following established patterns
triggers:
  - "write specification"
  - "create spec document"
  - "document requirements"
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

# Specification Writer Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Write production-ready, A+ quality specifications for software releases

---

## Overview

This skill encodes the pattern for writing comprehensive, technically rigorous specifications that match the quality of Dojo Genesis v0.0.17-v0.0.23. Use this skill when creating specifications for new features, releases, or system components.

**Quality Standard:** 111/100 (A+)

---

## When to Use This Skill

- Creating specifications for new software versions or releases
- Documenting complex system architectures
- Planning major feature implementations
- Communicating technical vision to development teams
- Ensuring consistency across multiple specifications

---

## The A+ Specification Template

### Part 1: Vision & Context (10-15% of document)

```markdown
# [Project Name] v[X.X.X]: [Memorable Tagline]

**Author:** [Your Name]  
**Status:** [Draft | Final | Approved]  
**Created:** [Date]  
**Grounded In:** [What this builds on - previous versions, research, feedback]

## 1. Vision

> A single, compelling sentence that captures the essence of this release.

**The Core Insight:**
[2-3 paragraphs explaining WHY this release matters, what problem it solves, and how it advances the overall vision]

**What Makes This Different:**
[2-3 paragraphs explaining what makes this approach unique, innovative, or better than alternatives]

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
```

### Part 2: Technical Architecture (40-50% of document)

```markdown
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

### 3.4 [Feature/Component 3] - Detailed Design

[Repeat structure for each major component]
```

### Part 3: Implementation Plan (20-25% of document)

```markdown
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
```

### Part 4: Risk & Quality (10-15% of document)

```markdown
## 5. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation Strategy |
|------|------------|--------|---------------------|
| [Risk description] | High/Med/Low | High/Med/Low | [Specific mitigation] |
| [Risk description] | High/Med/Low | High/Med/Low | [Specific mitigation] |

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
```

### Part 5: Appendices (5-10% of document)

```markdown
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

## Quality Checklist

Before finalizing a specification, verify:

### Vision & Clarity
- [ ] The tagline is memorable and captures the essence
- [ ] The vision is compelling and explains WHY this matters
- [ ] Goals are specific and measurable
- [ ] Success criteria are concrete and testable
- [ ] Non-goals are explicitly stated

### Technical Depth
- [ ] All major components have detailed designs
- [ ] Code examples are complete and production-ready
- [ ] API endpoints are fully specified (method, path, request, response)
- [ ] Database schema is included (if applicable)
- [ ] Integration points with existing systems are documented
- [ ] Performance considerations are addressed

### Implementation Rigor
- [ ] Timeline is realistic and grounded in complexity
- [ ] Week-by-week breakdown is specific and actionable
- [ ] Dependencies and prerequisites are identified
- [ ] Testing strategy covers unit, integration, E2E, and performance
- [ ] Manual QA scenarios are defined

### Risk & Quality
- [ ] Major risks are identified with mitigation strategies
- [ ] Rollback procedure is documented
- [ ] Feature flags are defined for gradual rollout
- [ ] Monitoring and alerting strategy is specified

### Documentation
- [ ] User-facing documentation plan is included
- [ ] Developer documentation updates are listed
- [ ] Release notes structure is defined

### Readability
- [ ] Headings are clear and hierarchical
- [ ] Tables are used for structured data
- [ ] Code examples are syntax-highlighted
- [ ] Diagrams or visual aids are included (if helpful)
- [ ] Language is precise and professional

---

## Examples of A+ Specifications

**From Dojo Genesis:**
- `Dojo_Genesis_v0.0.17_Final_Specification.md` - The Thoughtful System
- `Dojo_Genesis_v0.0.18_Final_Specification.md` - The Creative Studio
- `Dojo_Genesis_v0.0.19_Specification.md` - The Surgical Mind
- `Dojo_Genesis_v0.0.20_Specification.md` - The Compassionate Companion
- `Dojo_Genesis_v0.0.22_Specification_Expanded.md` - The Living Interface
- `Dojo_Genesis_v0.0.23_Specification.md` - The Collaborative Calibration

**Study these for:**
- How to structure complex technical architecture
- How to balance vision with implementation detail
- How to write production-ready code examples
- How to create realistic timelines
- How to identify and mitigate risks

---

## Common Pitfalls to Avoid

❌ **Vague Goals:** "Improve user experience" → ✅ "Reduce context loading time by 50%"  
❌ **Missing Code Examples:** High-level description only → ✅ Complete, runnable code  
❌ **Unrealistic Timelines:** "2 days for full backend" → ✅ "2 weeks with phased approach"  
❌ **No Risk Assessment:** Assumes everything will work → ✅ Identifies risks and mitigations  
❌ **Incomplete Testing:** "We'll test it" → ✅ Specific test cases and coverage targets  
❌ **No Integration Points:** Treats feature as isolated → ✅ Documents how it connects to existing system

---

## Usage Instructions

1. **Read this skill** before writing any specification
2. **Copy the template** structure to your new document
3. **Fill in each section** with specific, detailed content
4. **Run the quality checklist** before finalizing
5. **Study example specifications** for patterns and depth
6. **Iterate** until every checklist item is ✅

---

## Skill Metadata

**Token Savings:** ~10,000-15,000 tokens per specification (no need to re-read old specs for patterns)  
**Quality Impact:** Ensures consistency across all specifications  
**Maintenance:** Update when new patterns emerge from successful releases  

**Related Skills:**
- `memory-garden` - For documenting learnings from implementation
- `seed-extraction` - For extracting reusable patterns from specs
- `dual-track-orchestrator` - For coordinating parallel development

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
