---
name: health-audit
model: opus
description: Produces engineering commissions — structured implementation prompts stored in `docs/audits/` and `prompts/` — by auditing repositories for critical issues, security gaps, and sustainability risks. Use when: "audit this repo", "what are the blockers", "run a full health check", "identify technical debt before a sprint", "onboarding to a new codebase".
category: system-health

inputs:
  - name: repo_path
    type: string
    description: Path to the repository to audit for critical issues and technical debt
    required: true
outputs:
  - name: engineering_commissions
    type: ref
    format: cas-ref
    description: Structured implementation prompts stored in docs/audits/ and prompts/ covering critical issues, security gaps, and sustainability risks
---

# Health Audit Skill

## I. Philosophy

Health is not merely the absence of bugs; it is the presence of practices that ensure security, sustainability, and alignment with a project's core purpose. A health audit is a gardener's walk, not a mechanic's checklist — the role is to tend the ecosystem, protect the conditions for growth, and ensure the space can fulfill its intended purpose.

This skill produces two types of artifacts that distinguish it from `repo-status` (which writes status documents) and `documentation-audit` (which logs documentation drift): a permanent audit log recording what was found, and actionable engineering commissions that give implementation agents everything they need to fix the issues autonomously.

## II. When to Use This Skill

- As a scheduled, recurring task (e.g., on the first day of each month) for critical repositories.
- When onboarding to a new set of repositories: to establish a baseline health assessment.
- After a major release or architectural change: to audit the impact on the system's overall health.
- When a project feels "at risk" due to technical debt or process gaps.

Do not use this skill to write status documents (use `repo-status`) or to audit documentation accuracy (use `documentation-audit`). Health audit is specifically about generating engineering commissions.

## III. The Health Supervision Workflow

### Phase 1: Grounding

**Objective:** Understand what the repository is for before evaluating how well it works.

1. Use `repo-context-sync` to extract directory structure, recent diffs, and file patterns.
2. Read core philosophy, architecture, and purpose documents (`README.md`, `PHILOSOPHY.md`, `ARCHITECTURE.md`, etc.).
3. Identify the tech stack, active branches, and CI/CD status before assessing anything.

### Phase 2: Health Audit and Logging

**Objective:** Systematically assess the repository's health and create a permanent record.

1. Ensure `docs/audits/` directory exists. Create it if not.
2. Create a new audit log: `docs/audits/YYYY-MM-DD_health_audit.md`.
3. Conduct assessment using the Health Assessment Framework (Section IV):
   - Critical Issues: blockers, build failures, critical dependency vulnerabilities.
   - Security Issues: encryption gaps, secret management flaws, authentication vulnerabilities.
   - Sustainability Issues: technical debt, manual processes, documentation drift, testing gaps, paused development.
4. For each finding, document: issue, severity, impact, and a placeholder for the corresponding implementation commission.

### Phase 3: Generate Engineering Commissions

**Objective:** Translate audit findings into clear, specific, executable engineering tasks.

1. For each repository's set of findings, create a single consolidated implementation prompt.
2. Structure each commission as a cohesive "health sprint" that addresses the highest-priority issues together.
3. Each commission must contain:
   - A clear objective statement.
   - Links to the audit log and grounding documents.
   - Specific file paths, code snippets, and commands.
   - A complete file manifest (files to create or modify).
   - Binary, testable success criteria.
   - Explicit constraints and non-goals.
4. Store the commission under `prompts/` in the repository, following the `file-management` skill conventions.

### Phase 4: Summarize and Deliver

**Objective:** Provide a high-level executive summary and deliver all artifacts.

1. Write a single executive summary document explaining overall findings and linking to audit logs and commissions.
2. Deliver the summary and attach all created files (audit logs and implementation commissions) for user review.

## IV. Core Health Assessment Framework

| Category | Areas to Investigate |
| :--- | :--- |
| **Critical Issues** | Is the project buildable? (`npm run build`, `go build`) Are there critical dependency vulnerabilities (`npm audit --critical`)? Is the main branch in a broken state? |
| **Security** | Is sensitive data encrypted at rest? Are secrets managed securely (not hardcoded)? Is authentication/authorization implemented correctly? |
| **Sustainability** | Testing: is there a testing framework and adequate coverage? CI/CD: are tests and builds automated? Technical Debt: significant complexity, duplication, or outdated patterns? Manual Processes: required for setup, testing, or deployment? Documentation: accurate, complete, and current? |

## V. Best Practices

- **Balance is Key:** The goal is not to be purely mechanical. Hold both the philosophical purpose and the engineering reality in balance. Output should be empathetic in tone but rigorous in detail.
- **Audit Trail is Non-Negotiable:** Every health assessment must result in a committed audit log in the repository.
- **Commissions, Not Commands:** Frame implementation prompts as well-specified commissions. Provide all the context and detail the agent needs to succeed autonomously.
- **One Sprint per Repo:** Consolidate findings for a single repository into a single, cohesive health sprint prompt.
- **Close the Loop:** After the implementation agent completes a health sprint, schedule a follow-up audit to verify success criteria were met.

## VI. Quality Checklist

- [ ] Core documents read before any assessment begins
- [ ] `docs/audits/` directory exists and audit log created
- [ ] All three categories assessed: Critical, Security, Sustainability
- [ ] Each finding has severity, impact, and commission placeholder
- [ ] Implementation commission is ruthlessly specific (file paths, commands, success criteria)
- [ ] Commission stored in `prompts/` directory
- [ ] Executive summary written and delivered

## Output
- `docs/audits/YYYY-MM-DD_health_audit.md` — permanent findings record with severity ratings and commission cross-references.
- `prompts/health-sprint-YYYY-MM-DD.md` — executable engineering commission for an implementation agent.
- Executive summary delivered to the user linking both artifacts.

## Examples
**Scenario 1:** "Run a full health check on the Gateway repo" → Audit log created with 2 Critical (build failing on main, hardcoded API key in tests), 1 Security (no encryption at rest for session tokens), 3 Sustainability findings. Single health sprint commission generated with binary success criteria for each.
**Scenario 2:** "Onboard to the HTMLCraft repo before the next sprint" → Baseline audit reveals no critical issues, one security concern (missing RLS policies), and two sustainability gaps (no CI/CD, 12% test coverage). Commission generated for the CI/CD and test coverage gaps; RLS issue escalated to Critical for immediate fix.

## Edge Cases
- If a repo is a prototype or marked explicitly as non-production, adjust sustainability expectations accordingly — flag the discrepancy if critical production safeguards are missing anyway.
- If `docs/audits/` already has a recent audit (within 2 weeks), compare findings against the prior audit rather than starting fresh.

## Anti-Patterns
- Producing a long list of findings without grouping them into a single executable commission — implementation agents need a sprint, not a bug list.
- Conflating health-audit with repo-status: health-audit produces commissions; repo-status produces status documents. The two are complementary, not interchangeable.
