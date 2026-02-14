# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: repo-status
description: Generate comprehensive repository status reports
triggers:
  - "generate repo status"
  - "create status report"
  - "check repository health"
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
    - health-agent
---

# Repo Status Skill

**Version:** 1.0
**Created:** 2026-02-08
**Author:** Cruz + Manus (Cowork)
**Origin:** Codified from a live repo-context-sync + strategic-scout + status-writing session on Dojo Genesis.
**Lineage:** status-writing + file-management + health-audit, distilled into a single repeatable workflow.

---

## I. The Philosophy: See the Whole Before Touching a Part

Most codebases are explored piecemeal — someone reads a file, then another, then another, building a fragmentary picture that drifts as the project evolves. This skill takes the opposite approach: **see the whole system first, as a living organism**, then record what you see in a structured artifact that any future agent or human can use as a starting point.

The output is a `.status.md` file — a comprehensive, versioned snapshot that bridges three perspectives:

- **Status Writer**: Emoji-driven health indicators, workstream tracking, blockers, next steps. The "heartbeat" view.
- **File Management**: Annotated directory tree with per-folder status. The "anatomy" view.
- **Health Supervisor**: Critical/security/sustainability assessment. The "checkup" view.

A single document. One place to look. Updated over time like a living record.

---

## II. When to Use This Skill

- **Onboarding to a new project:** Before writing a single line, run this skill to build your mental model.
- **Periodic health check:** Monthly or per-release, re-run to track drift and detect decay.
- **Before a major refactor:** Understand what exists before deciding what to change.
- **When handing off context:** Give another agent or team member a single file that tells them everything.
- **When a project feels "big" or "messy":** Structure reduces anxiety. This skill imposes structure on chaos.

---

## III. The Workflow

This is a 5-phase process. Each phase builds on the previous one.

### Phase 1: Grounding (Read Before You Judge)

**Objective:** Understand what the project *is* before evaluating how well it works.

1. **List the root directory.** Get the top-level shape.
2. **Read core documents.** In priority order:
   - `README.md` (what it is, how to run it)
   - `STATUS.md` or equivalent (where it is right now)
   - `CHANGELOG.md` or release notes (where it's been)
   - `package.json`, `go.mod`, `Cargo.toml`, `pyproject.toml` (tech stack, deps)
   - `ARCHITECTURE.md`, `PHILOSOPHY.md` if they exist (intent, design)
3. **Identify the tech stack.** Language(s), framework(s), database(s), infra.
4. **Note the project's self-stated purpose.** This becomes Section 1 of the output.

*Time budget: ~10% of total effort. Resist the urge to explore deeply here.*

### Phase 2: Deep Inventory (Parallel Exploration)

**Objective:** Build a complete picture of everything in the repository.

Launch parallel exploration across the major layers. For most projects, these are:

| Layer | What to Inventory |
|-------|-------------------|
| **Frontend** | Components, routes, state management, styling, tests, LOC estimates |
| **Backend** | Packages, handlers, services, models, LOC estimates |
| **Data** | Migrations, schemas, seed data, vector stores |
| **Knowledge** | Docs, skills, prompts, thinking artifacts, compressions |
| **Infra** | CI/CD, Docker, scripts, deployment configs |
| **Tests** | Unit, integration, e2e, coverage estimates |

For each layer, collect:
- File count
- Approximate LOC (use `wc -l` or `cloc` if available)
- Package/module count
- Status (active, legacy, deprecated, empty)

*Time budget: ~30% of total effort. Parallelize aggressively.*

### Phase 3: Semantic Clustering

**Objective:** Group everything by *what the system does*, not where files happen to live.

This is the distinctive step. Instead of just listing directories, map every feature to an **action verb** — a cluster that describes a behavioral capability of the system.

Read `references/semantic-clusters.md` for the full cluster framework and examples.

**The process:**

1. Review the inventory from Phase 2.
2. For each significant component, ask: "What verb describes what this does?"
3. Assign it to a cluster. Create new clusters if needed.
4. For each cluster, build a component table: `Component | Location | Status | LOC`
5. Write a health assessment per cluster.
6. Identify cross-cluster components (things that serve multiple verbs).
7. Identify orphans (components that don't fit any cluster) — these may signal architectural confusion.

*Time budget: ~25% of total effort. This is where the insight lives.*

### Phase 4: Write the .status.md

**Objective:** Assemble everything into a single, structured document.

Read `references/status-template.md` for the complete output template.

The document has 10 sections:

1. **Vision & Purpose** — One sentence + core principles (from Phase 1)
2. **Current State** — Emoji table of major areas (from Phase 2)
3. **Directory Structure** — Annotated tree with per-folder status (from Phase 2)
4. **Semantic Clusters** — One subsection per verb, with component tables (from Phase 3)
5. **File Importance Ranking** — 4 tiers (Critical, Important, Supporting, Knowledge)
6. **Health Assessment** — Critical issues, security, sustainability (health-audit framework)
7. **Active Workstreams** — What's being worked on right now
8. **Blockers & Dependencies** — What's preventing progress
9. **Next Steps** — Immediate actionable items
10. **Aggregate Statistics** — Summary numbers table

Save to `.status.md` at the project root (dot-prefixed to stay out of the way but discoverable).

*Time budget: ~25% of total effort.*

### Phase 5: Verification

**Objective:** Catch errors, fill gaps, and harden the document.

1. **Cross-check statistics programmatically.** Run `find` and `wc -l` to verify LOC claims. Check file counts against actual filesystem.
2. **Identify unmapped components.** Walk the component directories and confirm every significant directory appears in at least one semantic cluster.
3. **Validate status emojis.** Ensure every area marked "complete" actually has evidence of completeness.
4. **Apply corrections.** Update the `.status.md` with any discrepancies found.
5. **Note confidence level.** If you couldn't verify something, say so.

*Time budget: ~10% of total effort. This step prevents the document from being fiction.*

---

## IV. The Semantic Cluster Framework

The clusters are action verbs that describe what a system *does*. For the full reference with starter verbs and mapping guidance, read:

```
references/semantic-clusters.md
```

**Starter set (adapt to your project):**

| Verb | Meaning | Example Components |
|------|---------|-------------------|
| CONVERSE | Chat, messaging, streaming | Chat UI, SSE handlers, message models |
| REASON | Thinking, planning, deciding | Agent, intent classifier, orchestration |
| REMEMBER | Storage, recall, learning | Memory system, embeddings, seeds |
| OBSERVE | Monitoring, tracing, metrics | Trace logger, event bus, cost tracker |
| LEARN | Adaptation, feedback, calibration | Calibration engine, preference system |
| ACT | Tool execution, side effects | Tool registry, tool implementations |
| PROTECT | Auth, security, boundaries | Middleware, secure storage, auth context |
| CONNECT | External integrations | Plugins, APIs, bots, webhooks |
| PRESENT | UI, rendering, layout | Shell, sidebars, panels, components |
| PERSIST | Database, migrations, storage | DB manager, migrations, schemas |
| BUILD | CI/CD, testing, deployment | Workflows, Docker, scripts, tests |
| THINK | Meta-cognition, knowledge | Skills, prompts, documentation |
| ORCHESTRATE | Multi-step coordination | DAG engine, task decomposition |

Not every project will use all 13. Some projects may need verbs not on this list. The framework is a starting point, not a constraint.

---

## V. File Importance Ranking

Rank files into 4 tiers based on runtime criticality and development importance:

| Tier | Criteria | Typical Count |
|------|----------|---------------|
| **Tier 1: Critical** | System won't function without these. Core agent, main state, primary API endpoint, database manager. | 10 files |
| **Tier 2: Important** | Core features break without these. Supporting services, secondary state, routing. | 10 files |
| **Tier 3: Supporting** | System degrades gracefully without these. Integrations, utilities, background tasks. | 20-30 files |
| **Tier 4: Knowledge** | Essential for development, not runtime. Docs, specs, skills, prompts. | Variable |

For each file in Tiers 1-2, include `Rank | File | Why` — a one-line justification.

---

## VI. Health Assessment Framework

Borrowed directly from the health-audit skill. For each category, use a table with `Area | Status | Notes`.

| Category | What to Check |
|----------|--------------|
| **Critical Issues** | Can it build? Critical dependency vulnerabilities? Main branch broken? |
| **Security** | Secrets hardcoded? Auth implemented? RLS policies set? Encryption at rest? |
| **Sustainability** | Test coverage adequate? CI/CD automated? Technical debt managed? Docs current? Manual processes documented? |

Be honest. Use emoji status indicators: ✅ good, ⚠️ concern, ❌ blocked.

---

## VII. Best Practices

- **Dot-prefix the output file.** `.status.md` stays discoverable but doesn't clutter the root alongside README and STATUS.
- **Separate from STATUS.md.** The lightweight `STATUS.md` (per status-writing skill) is the human-facing dashboard. `.status.md` is the comprehensive agent-facing audit. They complement each other.
- **Update incrementally.** After the first generation, subsequent runs should diff against the existing `.status.md` and update rather than regenerate from scratch.
- **Statistics drift is normal.** LOC counts change every commit. The goal is "accurate enough to be useful" — within 10% is fine.
- **Semantic clusters are the crown jewel.** The directory tree tells you *where* things are. The clusters tell you *what* things do. Prioritize cluster quality over tree completeness.
- **Parallelize Phase 2.** The deep inventory is the slowest phase. Use subagents or parallel exploration to cut time.
- **Verification is not optional.** A beautiful document full of wrong numbers is worse than no document. Always verify.

---

## VIII. Quality Checklist

Before delivering the `.status.md`, confirm:

- [ ] Vision statement is present and accurate
- [ ] Every major directory appears in the annotated tree
- [ ] Every significant component maps to at least one semantic cluster
- [ ] No orphan directories (unmapped components) remain unexplained
- [ ] File importance ranking has at least 10 Tier 1-2 files
- [ ] Health assessment covers critical, security, and sustainability
- [ ] Aggregate statistics are programmatically verified (within 10%)
- [ ] Active workstreams reflect actual current work
- [ ] Next steps are concrete and actionable
- [ ] The document reads coherently from top to bottom
