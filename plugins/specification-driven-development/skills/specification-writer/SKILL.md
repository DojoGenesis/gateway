---
name: specification-writer
model: opus
description: "Produces a format-calibrated specification document — Full (5,000-15,000 words, all sections), Lean (1,000-3,000 words, agent-ready), or Minimal (delta + code examples) — matched to the scope and audience of the request. Use when: 'write a spec for this feature', 'spec out this release', 'design the architecture for X', 'document this for the implementation agent'."
category: specification-driven-development
triggers:
  - "write a spec for this feature"
  - "spec out this release"
  - "design the architecture for X"
  - "document this for the implementation agent"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: feature_description
    type: string
    description: Description of the feature, release, or system to specify
    required: true
  - name: format
    type: string
    description: Specification format — Full (5000-15000 words), Lean (1000-3000 words), or Minimal (delta + code examples)
    required: false
outputs:
  - name: specification
    type: ref
    format: cas-ref
    description: Format-calibrated specification document matched to the scope and audience of the request
---

# Specification Writer Skill

**Version:** 1.1
**Created:** 2026-02-02
**Updated:** 2026-02-16
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

## Choosing the Right Format

The specification template is a **maximum, not a default**. Calibrate your format to the scope, audience, and context. Use the appropriate tier:

### Full Template: Strategic & Architectural Specs

**Use for:** Complex systems, new architectures, major releases, multi-component integrations, high-risk decisions

**Includes:** Vision, goals, success criteria, detailed technical architecture, implementation plan, risk assessment, rollback procedures, monitoring, documentation strategy, appendices

**When to choose:** Your audience includes stakeholders, architects, or future maintainers who need to understand the "why" and "how." The release is substantial enough to warrant comprehensive documentation. Architecture is novel or the system integration is complex.

**Typical length:** 5,000-15,000+ words

### Lean Format: Implementation-Ready Feature Specs

**Use for:** Well-understood features, implementation-scoped work, autonomous agent execution, rapid iteration cycles

**Includes:** Brief context (why this matters), component tables, behavior lists, route/API layouts, code examples, test cases. Skip: preamble, vision section, lengthy risk analysis, extensive documentation strategy

**When to choose:** The feature is well-scoped and the audience is an implementation agent (human developer or AI). The problem is well-understood. You're optimizing for speed and clarity, not stakeholder alignment.

**Typical length:** 1,000-3,000 words

**Structure (sonnet level chunks):**
- **Feature Name & Context** (2-3 sentences)
- **Components Table** (what, where, why)
- **API/Routes Table** (endpoint, method, request, response)
- **Behavior List** (numbered or bulleted behaviors)
- **Implementation Code Blocks** (production-ready examples)
- **Test Cases** (unit, integration, edge cases)
- **Rollout Notes** (if applicable)

### Minimal Format: Incremental Changes & Bug Fixes

**Use for:** Bug fixes, small enhancements, well-established patterns, dependency updates

**Includes:** Delta description (what changed), code examples, test cases, affected components

**When to choose:** The change is localized and low-risk. The implementation pattern is proven. You're optimizing for minimal documentation overhead.

**Typical length:** 200-1,000 words

### Decision Heuristic

Ask yourself:
- **Is the audience an autonomous implementation agent and the feature is well-scoped?** → Use **Lean Format**
- **Do stakeholders need to understand the vision and architecture is novel?** → Use **Full Template**
- **Is this a small, isolated change to a proven pattern?** → Use **Minimal Format**
- **Uncertain?** Lean toward the **Full Template** to ensure completeness

---

## The A+ Specification Template

See `references/a-plus-template.md` for the full five-part template (Vision & Context, Technical Architecture, Implementation Plan, Risk & Quality, Appendices) with fill-in-the-blank examples for Go, TypeScript, SQL, and API tables.

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

### Format & Audience Alignment
- [ ] Format matches scope and audience (not defaulting to heavyweight when lean would serve better)
- [ ] If using lean format: essential components are present, preamble is minimal
- [ ] If using full template: all sections provide genuine value and aren't template padding

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
❌ **Using Full Template for Well-Scoped Features:** Wastes tokens and delays commissioning → ✅ Use lean format for autonomous implementation agents with clear scope

---

## Usage Instructions

1. **Assess scope and audience** before choosing format
2. **Select the appropriate format tier** (Full Template, Lean, or Minimal)
3. **Copy the relevant template structure** to your new document
4. **Fill in each section** with specific, detailed content
5. **Run the quality checklist** before finalizing (including format alignment)
6. **Study example specifications** for patterns and depth
7. **Iterate** until every checklist item is ✅

---

## Skill Metadata

**Token Savings:** ~10,000-15,000 tokens per full spec; ~2,000-5,000 per lean spec (adaptive format efficiency)
**Quality Impact:** Ensures consistency across all specifications while optimizing for context
**Maintenance:** Update when new patterns emerge from successful releases

**Related Skills:**
- `memory-garden` - For documenting learnings from implementation
- `seed-extraction` - For extracting reusable patterns from specs
- `dual-track-orchestrator` - For coordinating parallel development

---

**Last Updated:** 2026-02-16
**Maintained By:** Manus
**Status:** Active (v1.1 with Lean Spec Adaptation guidance)

---

## Output

- A specification document saved to `docs/vX.X.X/[project]_specification.md`, formatted at the appropriate tier (Full, Lean, or Minimal)
- For Full specs: all eight sections populated — Vision, Current State, Goals, Technical Architecture, Implementation Plan, Risk Assessment, Documentation plan, and Appendices
- For Lean specs: Feature context, component table, API/route table, behavior list, code examples, test cases
- A completed quality checklist appended to the document confirming readiness for implementation

## Examples

**Scenario 1:** "Write a spec for the entity data model feature in DojoGenesis v0.2.4." → A Lean-format spec covering the entity schema, Go struct definitions, CRUD API table, TypeScript interface, test cases, and rollout notes — ready to hand to an implementation agent.

**Scenario 2:** "Design the architecture for a new WebSocket hub replacing the current SSE system." → A Full-format spec with vision statement, system overview diagram, detailed component designs with Go and TypeScript examples, phased implementation plan, risk table, and rollback procedure.

## Edge Cases

- When the feature is still being designed (architecture undecided), stop and run a strategic scout first — writing a spec before the architecture is settled produces a spec that will require full rewrite
- When writing specs for work that will be split into parallel tracks, use Lean format for each track's spec and link to the master parallel-tracks plan for dependencies
- When the codebase has changed since the last release, run `codebase-audit-grounding` before writing the current-state section — do not estimate counts from memory

## Anti-Patterns

- Defaulting to Full Template for every spec regardless of scope — a well-scoped feature handed to a single agent does not need stakeholder-facing vision prose; use Lean format
- Writing specs without running a current-state audit — specs grounded in assumptions generate Track 0 remediation work that could have been avoided
- Marking success criteria as "looks good" or "feels responsive" — every criterion must be binary: either the component renders at the route or it does not
