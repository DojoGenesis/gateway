---
name: health-audit
description: >
  Conduct a comprehensive 4-phase health audit on any software repository.
  Evaluates 5 dimensions (critical issues, security, testing, technical debt,
  documentation) with GREEN/YELLOW/RED classification, generates actionable
  engineering tasks with file paths and effort estimates, and produces a
  permanent dated audit artifact. Use when onboarding to a new codebase, after
  major releases, on a recurring schedule, or whenever a project feels at risk.
  Trigger words: health check, audit, code review, technical debt, security
  review, project health, sustainability, code quality.
triggers:
  - "run a health audit on this repo"
  - "check the project health and technical debt"
  - "assess this codebase for security and quality"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - health-agent
---

# Health Audit Skill

**Version:** 1.0
**Created:** 2026-02-11
**Purpose:** Provide a structured, repeatable methodology for auditing the health of software repositories, generating actionable engineering tasks, and creating a permanent audit trail.

---

## I. The Philosophy: Tending the Garden

Health is not merely the absence of bugs; it is the **presence of practices** that ensure security, sustainability, and alignment with a project's core purpose. A health auditor is a gardener, not a mechanic. The role is to tend the ecosystem, protect the conditions for growth, and ensure the space can fulfill its intended purpose.

This skill codifies a workflow that moves from deep listening to decisive action, ensuring that every intervention is both contextually aware and technically precise. It creates a permanent, auditable trail of health assessments and corrective actions.

The core principle: **maintenance is not just fixing what's broken -- it's keeping what works working well.** Regular audits compound quality. A single audit is a snapshot; a series of audits reveals a trajectory. That trajectory -- improving or degrading -- is the truest measure of project health.

---

## II. When to Use This Skill

- **As a scheduled, recurring task** (e.g., monthly or quarterly) to maintain regular health monitoring for critical repositories.
- **When onboarding to a new codebase** to establish a baseline health assessment before making changes.
- **After a major release or architectural change** to audit the impact on overall system health.
- **When a project feels "at risk"** due to accumulating technical debt, missing tests, or process gaps.
- **Before a sprint planning session** to prioritize health improvement tasks alongside feature work.
- **When inheriting a project from another team** to understand the current state before assuming ownership.

**When NOT to use:** For a quick code review of a single PR (that's a code review, not a health audit). For understanding what a system does (use semantic clusters instead). For checking a single document (use documentation audit instead).

---

## III. The Health Audit Workflow

### Phase 1: Grounding

**Goal:** Understand what the repository is *for* before evaluating how well it works.

**Actions:**
1. Map the directory tree -- top-level shape, key directories, project structure.
2. Count files by type and estimate LOC to gauge scale.
3. Read foundational documents: README, package manifest (package.json, go.mod, Cargo.toml, pyproject.toml), CI configs.
4. Identify the tech stack: primary language, framework, runtime, infrastructure.
5. Read philosophy, architecture, or design documents if they exist. Hold the project's purpose as the primary lens for the entire audit.

**Outputs:**
- Project summary: name, purpose, tech stack, scale (files, LOC)
- Key directories and their roles
- Initial observations and hypotheses

**Key insight:** You cannot evaluate health without understanding purpose. A project with no tests might be healthy if it's a prototype; the same project in production is critical.

### Phase 2: Health Assessment

**Goal:** Systematically evaluate the repository across 5 dimensions.

For each dimension, assign a status:
- **GREEN** -- Healthy. Good practices in place, no immediate action needed.
- **YELLOW** -- Needs attention. Gaps or debt exist but nothing is broken yet.
- **RED** -- Critical. Immediate action required to prevent or fix failures.

**Dimension 1: Critical Issues**
- Build status: does the project compile/build without errors?
- Dependency health: any critical vulnerabilities in dependencies?
- Main branch state: is it clean, or are there broken commits?
- Missing dependencies or broken imports?

**Dimension 2: Security**
- Secrets management: are secrets hardcoded, or properly managed?
- Encryption: is sensitive data encrypted at rest and in transit?
- Authentication/authorization: are patterns correct and complete?
- Dependency vulnerabilities: any known CVEs in the dependency tree?

**Dimension 3: Sustainability -- Testing**
- Testing framework: is one configured and in use?
- Test coverage: approximate percentage and distribution?
- CI pipeline: are tests automated? Is the pipeline reliable?
- Test quality: are tests meaningful or just checking boxes?

**Dimension 4: Sustainability -- Technical Debt**
- Code smells: high-complexity functions, excessive duplication?
- TODO/FIXME/HACK density: how much acknowledged debt?
- Dead code: unused files, unreachable functions, stale feature flags?
- Manual processes: any setup, test, or deploy steps that should be automated?

**Dimension 5: Sustainability -- Documentation**
- README quality: accurate, complete, useful for a newcomer?
- Inline documentation: meaningful comments, not stale annotations?
- API documentation: does it exist and match actual endpoints?
- Architecture documentation: is the system's design recorded anywhere?

**Outputs:**
- 5-dimension health dashboard (GREEN/YELLOW/RED for each)
- Detailed findings for each dimension

### Phase 3: Generate Actionable Tasks

**Goal:** Translate every YELLOW or RED finding into a concrete engineering task.

For each finding, produce a task with:

| Field | Description |
|---|---|
| **Task title** | Verb-object format (e.g., "Fix frontend build failure", "Add integration tests for auth module") |
| **Priority** | P0 (do now -- blocking), P1 (this sprint), P2 (next sprint), P3 (backlog) |
| **File paths** | Specific files that need to change |
| **Estimated effort** | Time range in hours (e.g., "2-4 hours") |
| **Acceptance criteria** | Binary, testable condition (e.g., "`npm run build` exits with code 0") |

**Priority guidelines:**
- **P0:** Build failures, critical security vulnerabilities, broken main branch
- **P1:** High-severity security issues, missing critical tests, blocking tech debt
- **P2:** Documentation drift, moderate tech debt, missing non-critical tests
- **P3:** Code style improvements, nice-to-have documentation, minor debt

**Outputs:**
- Prioritized task list with all required fields
- Offer to create tickets if a project tracker is connected

### Phase 4: Produce Audit Report

**Goal:** Create a permanent, dated audit artifact.

Save as `docs/audits/[YYYY-MM-DD]_health_audit.md` with these sections:

1. **Executive Summary** -- 3-line verdict: overall health, biggest risk, highest-priority action.
2. **Health Dashboard** -- All 5 dimensions with status emoji and one-line summary.
3. **Findings** -- Grouped by dimension. Each finding: severity, description, impact, affected files.
4. **Action Items** -- Prioritized table of all generated tasks.
5. **Trend** -- If prior audits exist in `docs/audits/`, compare. Note improvements and regressions.

**Outputs:**
- Dated audit file committed to the repository
- Executive summary delivered to the user

---

## IV. Best Practices

### 1. Balance Purpose and Engineering

The goal is not to be overly philosophical or purely mechanical. Hold both the project's purpose and the engineering reality in balance. The output should be empathetic in tone but rigorous in detail.

### 2. Audit Trail is Non-Negotiable

Every health assessment *must* result in a committed audit log in the repository. This creates a permanent, traceable history. A health audit without a saved artifact is incomplete.

### 3. Actionable Tasks, Not Vague Advice

"Improve test coverage" is not actionable. "Add unit tests for `auth/middleware.go` functions `ValidateToken` and `RefreshSession` -- target: 80% coverage for auth package" is actionable. Every task must have file paths and acceptance criteria.

### 4. One Sprint, Not a Roadmap

Consolidate findings into a focused set of tasks that can be accomplished in a single sprint. Don't produce a 50-item backlog. Prioritize ruthlessly and present the top 5-10 items.

### 5. Close the Loop

After tasks are completed, schedule a follow-up audit to verify that findings were resolved and health improved. The trend between audits is the true measure of success.

### 6. Use Automation for Initial Assessment

Before manually reading files, use shell commands to quickly gauge scale and identify obvious issues: file counts, build status, dependency audits, test runs. Manual deep reading comes in Phase 2 for the areas that automation flags.

---

## V. Quality Checklist

Before completing an audit, confirm:

- [ ] All 5 health dimensions are assessed with GREEN/YELLOW/RED status
- [ ] Every YELLOW or RED finding has a corresponding actionable task
- [ ] Every task has title, priority, file paths, effort estimate, and acceptance criteria
- [ ] The audit report is saved as a dated file in `docs/audits/`
- [ ] The executive summary is 3 lines or fewer
- [ ] Tasks use verb-object naming
- [ ] Priority assignments follow the P0-P3 guidelines
- [ ] If prior audits exist, a trend comparison is included
- [ ] No vague recommendations -- every finding has concrete next steps
- [ ] The audit could be repeated by a different person and produce comparable results

---

## VI. Common Pitfalls

### Pitfall 1: Auditing Without Understanding Purpose

**Problem:** Flagging "no tests" as RED when the project is a 2-day prototype that was never meant for production.

**Solution:** Phase 1 (Grounding) exists for a reason. Understand the project's purpose and lifecycle stage before assigning severity. A prototype with no tests is YELLOW; a production service with no tests is RED.

### Pitfall 2: Producing Vague Advice Instead of Tasks

**Problem:** Writing "improve documentation" as a finding with no specifics.

**Solution:** Every finding must include the specific file, what's wrong, and what "fixed" looks like. If you can't name a file, you haven't audited deeply enough.

### Pitfall 3: Over-Scoping the Action Items

**Problem:** Generating 40 tasks that would take months to complete.

**Solution:** Prioritize ruthlessly. Present the top 5-10 items as a "health sprint." Everything else goes in the backlog section of the report. One focused sprint beats a 100-item wish list.

### Pitfall 4: Skipping the Trend Comparison

**Problem:** Each audit stands alone, making it impossible to know if health is improving.

**Solution:** Always check `docs/audits/` for prior audits. If they exist, compare dimensions. If they don't, note that this is the baseline audit.

### Pitfall 5: Conflating Health Audit with Code Review

**Problem:** Reviewing individual functions and code style instead of systemic health.

**Solution:** A health audit evaluates practices and systems (testing, CI, docs, security posture). A code review evaluates individual code changes. Stay at the system level.

---

## VII. Example: Repository Health Audit

**The Problem:** A production web application hasn't been audited in 3 months. The team suspects growing technical debt.

**The Process:**

1. **Grounding:** Mapped the repository -- Next.js frontend (142 files, ~18K LOC), Go backend (89 files, ~12K LOC), PostgreSQL database. Purpose: customer-facing SaaS platform.

2. **Assessment:**
   - Critical Issues: RED -- Frontend build fails due to missing env var in prerender.
   - Security: YELLOW -- Dependencies have 1 high-severity CVE. Secrets in `.env` files, no vault.
   - Testing: YELLOW -- Backend has 65% coverage, frontend has 12%. No E2E tests.
   - Technical Debt: YELLOW -- 47 TODOs, 3 functions over 200 LOC, unused migration files.
   - Documentation: RED -- README references deprecated API endpoints. No architecture docs.

3. **Tasks Generated:**
   - P0: Fix frontend build failure (1-2 hours)
   - P0: Upgrade `next` to patch CVE (1 hour)
   - P1: Add frontend unit tests for auth flows (4-6 hours)
   - P1: Implement secrets management with vault (3-4 hours)
   - P2: Update README API references (2 hours)
   - P2: Write architecture overview document (3-4 hours)

4. **Report:** Saved as `docs/audits/2026-02-11_health_audit.md` with executive summary, dashboard, findings, and prioritized action items.

**The Outcome:** Team fixed P0 items in the first day, completed P1 items in the sprint, and scheduled P2 items for the following sprint. Follow-up audit showed Critical Issues moved from RED to GREEN.

---

## VIII. Related Skills

- **`semantic-clusters`** -- Map behavioral architecture before or alongside a health audit for deeper system understanding
- **`documentation-audit`** -- Deep dive into documentation drift (Dimension 5 of the health audit)
- **`status-writing`** -- Use audit results to write an accurate STATUS.md
- **`skill-audit`** -- Similar methodology applied to skills ecosystems instead of codebases
