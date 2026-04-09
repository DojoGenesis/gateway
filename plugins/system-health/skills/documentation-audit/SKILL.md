---
name: documentation-audit
model: sonnet
description: "Produces a committed audit log file enumerating documentation drift — inaccuracies, broken links, missing sections, and outdated information — with severity ratings and action taken. Use when: 'docs seem out of date', 'audit the documentation', 'is the README accurate', 'check for broken links', 'before onboarding a new team member'."
category: system-health
triggers:
  - "docs seem out of date"
  - "audit the documentation"
  - "is the README accurate"
  - "check for broken links"
  - "before onboarding a new team member"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"
  - "bash"

inputs:
  - name: docs_path
    type: string
    description: Path to the documentation directory or file to audit
    required: true
outputs:
  - name: audit_log
    type: ref
    format: cas-ref
    description: Committed audit log enumerating documentation drift — inaccuracies, broken links, missing sections — with severity ratings and action taken
---

# Documentation Auditor Skill

## I. Philosophy

Project documentation is a garden. When tended with care, it is a source of clarity, guidance, and shared understanding. When neglected, it becomes overgrown with outdated information, broken links, and misleading instructions — a phenomenon known as documentation drift. This drift erodes trust and creates confusion.

The Documentation Auditor skill is the practice of tending the garden. It is a recurring ritual where we mindfully walk through our documentation, pulling the weeds of inaccuracy and pruning the branches of irrelevance. It is an act of stewardship, ensuring that our shared garden of knowledge remains a welcoming and reliable resource for all agents and collaborators.

## II. When to Use This Skill

- After a major release: to ensure all documentation reflects the new features and changes.
- Before a new team member or agent is onboarded: to ensure they are given accurate information.
- As a scheduled, recurring task (e.g., on the first day of each month) to maintain a regular cadence of review.
- When you have a feeling that the documentation is out of sync with the code.

## III. The Audit Workflow

### Step 1: Initiate the Audit

Announce the intention to perform a documentation audit. Define the scope of the audit (e.g., "a full audit of the Dojo Genesis repo" or "a targeted audit of the AROMA README").

### Step 2: Create an Audit Log

Create a new markdown file to log the findings of the audit (e.g., `docs/audits/YYYY-MM-DD_documentation_audit.md`).

Use the template from `references/audit-log-template.md` in this skill directory. The template provides the table schema and severity definitions.

### Step 3: Systematically Review Each Document

Using the checklist in Section V, go through each key document in the repository. For each document, check for:

- **Accuracy:** Does the information reflect the current state of the code?
- **Completeness:** Is anything missing?
- **Clarity:** Is the language clear, concise, and easy to understand?
- **Broken Links:** Do all internal and external links still work?

For each issue found, create an entry in the audit log.

### Step 4: Prioritize and Address the Issues

Once the review is complete, review the audit log and prioritize the issues. Address the high-priority issues immediately by creating pull requests to update the documentation.

### Step 5: Commit and Share the Findings

Commit the audit log and all documentation fixes to the repository. Share a summary of the findings, highlighting the key improvements made.

## IV. Core Documentation Checklist

**`README.md`**
- [ ] Is the project purpose clear?
- [ ] Are the installation and quickstart instructions accurate and functional?
- [ ] Does it link to other key documents (e.g., `CONTRIBUTING.md`, `ARCHITECTURE.md`)?
- [ ] Is the status badge (if any) correct?

**`CONTRIBUTING.md`**
- [ ] Are the guidelines for contributing clear?
- [ ] Is the process for submitting a pull request well-defined?
- [ ] Does it link to the code of conduct?

**`ARCHITECTURE.md`**
- [ ] Does the high-level overview reflect the current system architecture?
- [ ] Are all major components and their interactions documented?
- [ ] Are diagrams up-to-date?

**`docs/` Directory**
- [ ] Are all specifications for past releases present?
- [ ] Are all retrospective documents present?
- [ ] Is there any outdated information in the guides or tutorials?

**`SKILLS/` Directory**
- [ ] Does each skill have a clear `SKILL.md` file?
- [ ] Is the description and purpose of each skill accurate?

## V. Best Practices

- **Audit in Small, Regular Batches:** It is less daunting to audit one section of the documentation each week than to audit the entire repository once a year.
- **Automate Where Possible:** Use tools like `lychee` to check for broken links automatically.
- **Link, Don't Copy:** When information needs to exist in multiple places, link to a single source of truth rather than copying and pasting it. This makes updates much easier.
- **Every Fix is a Good Fix:** Even fixing a small typo improves the quality of the garden.

## VI. Quality Checklist

- [ ] Scope defined before starting (full repo or targeted area)
- [ ] Audit log file created using the template from `references/audit-log-template.md`
- [ ] Every key document reviewed against the checklist in Section IV
- [ ] Each finding has a severity rating (High / Medium / Low)
- [ ] High-priority issues addressed immediately or assigned with clear next steps
- [ ] Audit log committed to the repository
- [ ] Summary shared with stakeholders

## Output
- A committed audit log file at `docs/audits/YYYY-MM-DD_documentation_audit.md` containing a findings table (file, line, issue, severity, action taken) and a summary (total issues found, resolved, deferred).
- Pull requests or direct fixes for high-severity findings made during the audit.

## Examples
**Scenario 1:** "Audit the README before we onboard a new contributor" → Audit log created with 4 findings: outdated installation command (High), missing link to CONTRIBUTING.md (High), stale architecture diagram reference (Medium), minor typo (Low). High-severity issues fixed in the same session.
**Scenario 2:** "Monthly documentation maintenance" → Targeted audit of `docs/` directory reveals 2 specs that reference deprecated API endpoints. Findings logged, issues created for follow-up in the next sprint.

## Edge Cases
- If the repo has no `docs/` directory yet, note this in the audit log as a structural gap rather than a per-file finding — recommend creating the directory.
- External links may be behind auth or rate-limited; note failed verification rather than marking all links broken by default.

## Anti-Patterns
- Copying the audit log template inline into the SKILL.md itself — the template lives in `references/audit-log-template.md` to avoid duplication and enable independent updates.
- Auditing only the README — documentation drift typically hides in architecture docs, changelogs, and skill files, not just the front-door document.
