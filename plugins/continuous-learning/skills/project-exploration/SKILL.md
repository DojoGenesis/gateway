---
name: project-exploration
model: sonnet
description: "Produces an Exploration Brief rating a project GREEN/YELLOW/RED for collaboration fit, with architecture snapshot, resonance map, and ranked entry points. Use when: 'explore this project', 'assess this codebase', 'should we collaborate on this', 'onboard me to this repo', 'evaluate this project for fit'."
category: continuous-learning
triggers:
  - "explore this project"
  - "assess this codebase"
  - "should we collaborate on this"
  - "onboard me to this repo"
  - "evaluate this project for fit"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_path
    type: string
    description: Path or URL to the project/repository to explore
    required: true
outputs:
  - name: exploration_brief
    type: string
    description: Exploration Brief rating the project GREEN/YELLOW/RED with architecture snapshot, resonance map, and ranked entry points
---

# Project Exploration Skill

**Version:** 1.0
**Author:** Tres Pies Design
**Purpose:** Structured process for respectfully exploring a new project to determine collaboration readiness and fit before committing.

---

## Philosophy: Exploration Before Commitment

When encountering a large project or codebase, explore its philosophy and patterns before diving into implementation. The goal is not just to understand the code, but to understand the project's relationship to your goals, values, and existing knowledge.

---

## Core Principles

- **Progressive Disclosure:** Do not attempt to read everything at once. Start with a high-level overview and drill down as needed.
- **Incremental Synthesis:** Save findings as you go. Externalize understanding to prevent information loss.
- **Seek Conceptual Clarity:** Prioritize understanding the core philosophy, values, and goals before implementation details.
- **Map Connections:** Understand the project's relationship to your existing knowledge and skills. Explicitly map resonances.
- **Express Genuine Enthusiasm:** Collaboration is relational. Output should reflect genuine interest and readiness to contribute.

---

## When to Use This Skill

- Invited to collaborate on a project with a large existing codebase
- Evaluating whether to adopt a library, framework, or tool
- Onboarding to a new team's repository
- Assessing whether a project's architecture supports your goals

---

## The 5-Phase Exploration Process

### Phase 1: First Impressions — Map the Terrain

Get a high-level overview of structure and scope.

1. **Read the README** — What is this project? Who built it? What's its stated purpose?
2. **Scan project structure** — List top-level directories and key files. Identify organizational patterns.
3. **Check recent activity** — Last commit, release cadence, contributor count. Is this project alive?
4. **Document initial findings** — Create an initial overview of themes, structure, and first impressions.

**Output:** Project overview with structure analysis and activity assessment.

### Phase 2: Architecture Mapping — Taste the Water

Select 2-3 key documents or files to understand core content and patterns.

1. **Identify entry points** — Main module, config files, core abstractions
2. **Map behavioral capabilities** — Use semantic-clusters approach: what does this project actually *do*?
3. **Identify patterns** — What architectural patterns does it follow? (MVC, plugin architecture, event-driven, etc.)
4. **Note dependencies** — Key external dependencies and their role

**Output:** Architecture snapshot with capability map and pattern inventory.

### Phase 3: Assess Fit — Build the Bridge

Evaluate against user's goals and constraints.

1. **Alignment check** — Does this project's architecture support what the user wants to do?
2. **Pattern compatibility** — Are the patterns compatible with existing work?
3. **Health assessment** — Is the codebase healthy enough to build on? (Test coverage, documentation quality, code style consistency)
4. **Resonance mapping** — Connect the project's philosophy to existing knowledge:

| Project Principle | Existing Principle/Pattern | Shared Insight |
| :--- | :--- | :--- |
| [Principle from project] | [Your relevant pattern] | [The connection] |

### Phase 4: Identify Entry Points

Where would a new collaborator start?

1. **Most approachable areas** — Well-documented, well-tested, clear interfaces
2. **Most impactful areas** — Where contribution would add the most value
3. **Most risky areas** — Complex, poorly documented, or tightly coupled
4. **Quick wins** — Small improvements that build familiarity

### Phase 5: Synthesis — The Exploration Brief

Produce a comprehensive assessment.

---

## Fit Classification

| Rating | Meaning | Guidance |
|--------|---------|----------|
| **GREEN** | Strong fit. Architecture supports goals, patterns are compatible, codebase is healthy. | Proceed with collaboration. Start with recommended entry points. |
| **YELLOW** | Partial fit. Some concerns about architecture, patterns, or health. Manageable with effort. | Proceed with caution. Address concerns before deep investment. |
| **RED** | Poor fit. Fundamental misalignment in architecture, philosophy, or health. | Reconsider collaboration. Document concerns for future reference. |

---

## Output Format: Exploration Brief

```markdown
## Exploration Brief: [Project Name]

**Date:** [Date]
**Repository:** [URL or path]
**Explorer:** [Name]

### Project Summary

[What this project is, who built it, what it does, and its core philosophy in 2-3 sentences.]

### Architecture Snapshot

**Structure:** [High-level organization]
**Key Patterns:** [Architectural patterns used]
**Core Dependencies:** [Major external dependencies]
**Activity:** [Last commit, release cadence, contributor count]

### Fit Assessment: [GREEN / YELLOW / RED]

**Alignment:** [Does this support your goals?]
**Pattern Compatibility:** [Are patterns compatible?]
**Codebase Health:** [Test coverage, docs quality, consistency]

### Resonance Map

| Project Principle | Existing Principle | Shared Insight |
| :--- | :--- | :--- |
| [Principle] | [Your pattern] | [Connection] |

### Recommended Entry Points

1. **[Area 1]:** [Why start here] — Risk: Low/Med/High
2. **[Area 2]:** [Why start here] — Risk: Low/Med/High

### Risks & Concerns

- [Risk 1: description and mitigation]
- [Risk 2: description and mitigation]
```

---

## Quality Criteria

- [ ] First impressions phase completed before architecture deep-dive
- [ ] Fit assessment uses GREEN/YELLOW/RED classification
- [ ] Resonance map connects project to existing knowledge
- [ ] Entry points ranked by approachability and impact
- [ ] Risks identified with mitigation strategies

---

## Output

- An Exploration Brief markdown document saved to the session working directory or `docs/exploration/`
- Named: `[date]_[project-name]_exploration.md`
- Sections: Project Summary, Architecture Snapshot, Fit Assessment (GREEN/YELLOW/RED), Resonance Map, Recommended Entry Points, Risks and Concerns

---

## Examples

**Scenario 1:** "Explore this repo and tell me if we should collaborate" → Exploration Brief with GREEN rating, architecture snapshot (plugin pattern, 87 tests, Go + Wails), 3 entry points ranked by risk, and resonance map connecting it to existing HTMLCraft work

**Scenario 2:** "Onboard me to this new codebase before we start implementing" → Exploration Brief with YELLOW rating noting sparse test coverage, architecture mapped, 2 risky areas flagged, quick-win entry points identified for building familiarity

---

## Edge Cases

- **Project has no README:** Begin Phase 2 directly with structural survey; note the absence of documentation as a codebase health signal in the Fit Assessment
- **Repository is private with limited access:** Explore what is accessible, note access limitations explicitly, and rate only what can be verified
- **Project is extremely large (monorepo with 1000+ files):** Limit structural survey to top-level and highest-traffic directories; do not attempt full traversal

---

## Anti-Patterns

- Reading implementation files before completing the README and structural survey — entry into details before understanding the whole produces a distorted architecture picture
- Assigning GREEN without completing all 5 phases — partial exploration that misses health assessment or resonance mapping produces false confidence
- Using this skill to evaluate whether to *use* a library as a dependency (use web-research instead) — project-exploration is for collaboration potential, not dependency selection
- Skipping the Risks section because the project seems good — surfacing low-probability risks during exploration is cheaper than discovering them mid-implementation
