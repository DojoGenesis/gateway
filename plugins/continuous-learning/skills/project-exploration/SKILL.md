---
name: project-exploration
description: Guides exploration of new, large-scale projects to assess collaboration potential. Explore philosophy and patterns before diving into implementation. Use when encountering a new codebase, repository, or project.
triggers:
  - "explore this new project before diving in"
  - "assess this codebase for collaboration potential"
  - "map the philosophy and patterns of this repo"
metadata:
  version: "1.1"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
---

# Project Exploration Skill

**Version:** 1.1
**Author:** Tres Pies Design
**Purpose:** Structured process for respectfully exploring a new project to determine collaboration readiness and fit before committing.

---

## I. Philosophy: Exploration Before Commitment

When encountering a large project or codebase, explore its philosophy and patterns before diving into implementation. The goal is not just to understand the code, but to understand the project's relationship to your goals, values, and existing knowledge.

**Core Principles:**

- **Progressive Disclosure:** Do not attempt to read everything at once. Start with a high-level overview and drill down as needed.
- **Incremental Synthesis:** Save findings as you go. Externalize understanding to prevent information loss.
- **Seek Conceptual Clarity:** Prioritize understanding the core philosophy, values, and goals before implementation details.
- **Map Connections:** Understand the project's relationship to your existing knowledge and skills. Explicitly map resonances.
- **Express Genuine Enthusiasm:** Collaboration is relational. Output should reflect genuine interest and readiness to contribute.

---

## II. When to Use This Skill

- Invited to collaborate on a project with a large existing codebase
- Evaluating whether to adopt a library, framework, or tool
- Onboarding to a new team's repository
- Assessing whether a project's architecture supports your goals

---

## III. The 5-Phase Exploration Workflow

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

Produce a comprehensive assessment using the Fit Classification and Output Format below.

---

## IV. Best Practices

1. **Start with README, not code** — The README reveals intent and philosophy. Code reveals implementation. Start with intent.
2. **Map resonances explicitly** — Don't just assess compatibility. Name the connections between the project's principles and your existing knowledge. Resonances accelerate collaboration.
3. **Time-box each phase** — Spend no more than 15-20 minutes per phase. Exploration should produce a decision, not an encyclopedia.
4. **Document as you go** — Don't wait until Phase 5. Capture findings at each phase to prevent information loss.
5. **Respect the project's norms** — Before contributing, understand the project's conventions (commit style, PR process, coding standards). Adopting norms shows good faith.

---

## V. Quality Checklist

- [ ] First impressions phase completed before architecture deep-dive
- [ ] Fit assessment uses GREEN/YELLOW/RED classification
- [ ] Resonance map connects project to existing knowledge
- [ ] Entry points ranked by approachability and impact
- [ ] Risks identified with mitigation strategies
- [ ] Time-boxed (< 90 minutes total for the full exploration)

---

## VI. Example: Dojo Supply Chain Repo Exploration (April 2026)

**The Problem:** Needed to evaluate 13 community GitHub repos for skill supply chain integration — determining which repos had compatible architectures, useful skills, and healthy codebases worth investing in.

**The Process:**

1. **First Impressions:** Scanned READMEs, commit histories, and star counts for all 13 repos. 4 repos eliminated immediately (dormant >6 months, no clear purpose).
2. **Architecture Mapping:** Mapped behavioral capabilities of 9 remaining repos using semantic-clusters. Found 3 distinct patterns: hooks-based, prompt-library, and full-agent-framework.
3. **Fit Assessment:** Assessed each against Dojo Gateway integration requirements. Classified: 5 GREEN (strong fit), 3 YELLOW (partial), 1 RED (incompatible architecture).
4. **Entry Points:** Identified specific skills and patterns to extract from each GREEN repo.
5. **Synthesis:** Produced exploration briefs that fed directly into the supply chain pipeline.

**The Outcome:** 495 discoverable community skills cataloged. 15 repos security-audited. 47 reusable seeds extracted. The exploration briefs prevented investing time in the 4 repos that would have been dead ends.

**Key Insight:** The resonance mapping in Phase 3 was the most valuable step — it revealed that 2 repos we almost skipped (low star count) had the highest philosophical alignment with Dojo's skill-first approach.

---

## VII. Common Pitfalls

### Pitfall 1: Diving Into Code Before Philosophy

**Problem:** Reading source code before understanding the project's intent leads to misinterpretation of design decisions.

**Solution:** Always complete Phase 1 (README, structure, activity) before Phase 2 (code). The README tells you *why*; the code tells you *how*.

### Pitfall 2: Skipping the Resonance Map

**Problem:** Without explicit resonance mapping, you assess projects as "good" or "bad" rather than "compatible" or "incompatible."

**Solution:** Always fill out the resonance table in Phase 3. Name specific connections between the project's principles and your existing patterns.

### Pitfall 3: Exploring Without a Time Box

**Problem:** Interesting projects can consume hours of exploration without producing a decision.

**Solution:** Set a 90-minute maximum for the full 5-phase process. If you can't assess fit in 90 minutes, the project is likely too complex for casual collaboration.

### Pitfall 4: Confusing Stars with Quality

**Problem:** High GitHub stars indicate popularity, not architectural compatibility or codebase health.

**Solution:** Phase 3 health assessment (test coverage, docs quality, consistency) is more predictive of collaboration success than star count.

### Pitfall 5: Producing Exploration Without Action

**Problem:** A beautiful exploration brief that doesn't lead to a clear GREEN/YELLOW/RED decision is wasted effort.

**Solution:** Every exploration must end with a classification and recommended next step. Exploration without decision is procrastination.

---

## VIII. Related Skills

- **`semantic-clusters`** — Used in Phase 2 to map behavioral capabilities of the project
- **`health-audit`** — Provides the 5-dimension assessment used in Phase 3 health checks
- **`research-modes`** — Deep research mode for investigating specific aspects that emerge during exploration
- **`handoff-protocol`** — When exploration leads to collaboration, use this to hand off findings to implementation agents
- **`seed-extraction`** — Extract reusable patterns discovered during exploration into seeds

---

## IX. Fit Classification Reference

| Rating | Meaning | Guidance |
|--------|---------|----------|
| **GREEN** | Strong fit. Architecture supports goals, patterns are compatible, codebase is healthy. | Proceed with collaboration. Start with recommended entry points. |
| **YELLOW** | Partial fit. Some concerns about architecture, patterns, or health. Manageable with effort. | Proceed with caution. Address concerns before deep investment. |
| **RED** | Poor fit. Fundamental misalignment in architecture, philosophy, or health. | Reconsider collaboration. Document concerns for future reference. |

---

## X. Output Format: Exploration Brief

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

**Last Updated:** 2026-04-06
**Status:** Active
