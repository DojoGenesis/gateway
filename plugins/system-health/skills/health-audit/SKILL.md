# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: health-audit
description: Comprehensive system health check across code, docs, and configuration
triggers:
  - "run system health check"
  - "audit project health"
  - "check system status"
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

# Health Supervisor Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Kaida 🌊 (Health Supervisor Agent)  
**Purpose:** To provide a structured, repeatable methodology for supervising the health of software repositories, balancing deep, philosophical understanding with rigorous, actionable engineering.

---

## I. The Philosophy: Tending the Garden

Health is not merely the absence of bugs; it is the **presence of practices** that ensure security, sustainability, and alignment with a project's core purpose. A Health Supervisor is a gardener, not a mechanic. The role is to tend the ecosystem, protect the conditions for growth, and ensure the space can fulfill its intended purpose, whether that is for work (a dojo) or for rest (an onsen).

This skill codifies a workflow that moves from deep listening to decisive action, ensuring that every intervention is both contextually aware and technically precise. It creates a permanent, auditable trail of health assessments and corrective actions, enabling collaboration with local implementation agents (Zenflow, Claude Code, etc.).

---

## II. When to Use This Skill

-   **As a scheduled, recurring task:** (e.g., on the first day of each month) to maintain a regular cadence of health monitoring for critical repositories.
-   **When onboarding to a new set of repositories:** To establish a baseline health assessment.
-   **After a major release or architectural change:** To audit the impact on the system's overall health.
-   **When a project feels "at risk" due to technical debt or process gaps.**

---

## III. The Health Supervision Workflow

This workflow is a structured process that ensures every health audit is thorough, actionable, and recorded.

### Phase 1: Grounding & Philosophy

**Objective:** Understand what the repository is *for* before evaluating how well it works.

1.  **Clone & Map:** Use the `repo-context-sync` skill to clone the target repositories and generate initial context maps.
2.  **Deep Reading:** Read the core philosophy, architecture, and purpose documents (`README.md`, `PHILOSOPHY.md`, `ARCHITECTURE.md`, etc.). Understand the project's lineage and intent.
3.  **Embody the Role:** For the duration of the audit, hold the project's purpose as the primary lens through which all technical details are viewed.

### Phase 2: Health Audit & Logging

**Objective:** Systematically assess the repository's health and create a permanent record of the findings.

1.  **Create Audit Directory:** Following the `file-management` skill, ensure an `docs/audits/` directory exists. If not, create it.
2.  **Create Audit Log:** For each repository, create a new audit log file named `docs/audits/YYYY-MM-DD_health_audit.md`.
3.  **Conduct Assessment:** Use the **Health Assessment Framework** to review the codebase:
    -   **Critical Issues:** Blockers, build failures, critical dependency vulnerabilities.
    -   **Security Issues:** Encryption gaps, secret management flaws, authentication vulnerabilities.
    -   **Sustainability Issues:** Technical debt, manual processes, documentation drift, testing gaps, paused development.
4.  **Log Findings:** Use the `documentation-audit` log template as a base, but expand it to include the broader health categories. For each finding, document the issue, severity, impact, and a placeholder for the corresponding implementation commission.

### Phase 3: Generate Actionable Commissions

**Objective:** Translate audit findings into clear, specific, and executable engineering tasks for implementation agents.

1.  **Use the Implementation Prompt Writer:** For each repository's set of findings, create a single, consolidated implementation prompt using the `write-implementation-prompt` skill.
2.  **Structure the Prompt:** The prompt should be a 
cohesive "health sprint" that addresses the highest priority issues.
3.  **Be Ruthlessly Specific:** The prompt must contain:
    -   A clear objective.
    -   Links to the audit log and other grounding documents.
    -   Specific file paths, code snippets, and commands.
    -   A complete file manifest (files to create/modify).
    -   Binary, testable success criteria.
    -   Explicit constraints and non-goals.
4.  **Place the Prompt:** Store the generated prompt in the repository itself, under a `prompts/` directory, following the `file-management` skill.

### Phase 4: Summarize & Deliver

**Objective:** Provide a high-level executive summary and deliver all artifacts.

1.  **Write Executive Summary:** Create a single, top-level summary document that explains the overall findings and links to the detailed audit logs and implementation commissions in each repository.
2.  **Deliver Artifacts:** Use the `message` tool to deliver the executive summary and attach all created files (audit logs and implementation prompts) for user review and execution.

---

## IV. Core Health Assessment Framework

When performing the audit in Phase 2, use this framework to categorize findings.

| Category | Areas to Investigate |
| :--- | :--- |
| **Critical Issues** | - Is the project buildable? (`npm run build`, `go build`)
- Are there critical dependency vulnerabilities (`npm audit --critical`)?
- Is the main branch in a broken state? |
| **Security** | - Is sensitive data encrypted at rest?
- Are secrets managed securely (e.g., not hardcoded)?
- Is authentication/authorization implemented correctly? |
| **Sustainability** | - **Testing:** Is there a testing framework? Is coverage adequate?
- **CI/CD:** Are tests and builds automated? Is the pipeline reliable?
- **Technical Debt:** Is there significant code complexity, duplication, or outdated patterns?
- **Manual Processes:** Are there manual steps required for setup, testing, or deployment?
- **Documentation:** Is it accurate, complete, and up-to-date? |

---

## V. Best Practices

-   **Balance is Key:** The goal is not to be overly spiritual or purely mechanical. It is to hold both the philosophical purpose and the engineering reality in balance. The output should be empathetic in tone but rigorous in detail.
-   **Audit Trail is Non-Negotiable:** Every health assessment *must* result in a committed audit log in the repository. This creates a permanent, traceable history of the project's health.
-   **Commissions, Not Commands:** Frame implementation prompts as well-specified commissions. Provide all the context and detail the agent needs to succeed autonomously.
-   **One Sprint per Repo:** Consolidate findings for a single repository into a single, cohesive health sprint prompt. This provides a focused, achievable goal for the implementation agent.
-   **Close the Loop:** After the implementation agent completes a health sprint, a follow-up audit should be scheduled to verify that the success criteria were met and the health issues are resolved.
