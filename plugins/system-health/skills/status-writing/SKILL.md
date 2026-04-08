---
name: status-writing
description: >
  Write and maintain STATUS.md files that provide at-a-glance visibility into
  project health, progress, risks, and next milestones. Integrates behavioral
  architecture summaries from semantic clusters and health dashboard data from
  health audits into a single, honest status document. Use when starting a new
  project, at the beginning or end of work sessions, during weekly syncs, or
  whenever there is a significant change in project status. Trigger words:
  status report, project status, STATUS.md, project health, sprint summary,
  weekly update, project visibility, what's the state of.
triggers:
  - "write a STATUS.md for this project"
  - "update the project status document"
  - "what is the current state of this project"
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

# Status Writing Skill

**Version:** 1.0
**Created:** 2026-02-11
**Purpose:** Provide a structured, repeatable process for writing and updating STATUS.md files, turning project tracking into a practice of radical honesty and transparency.

---

## I. The Philosophy: The Ritual of Bearing Witness

A STATUS.md file is more than a report; it is a **Ritual of Bearing Witness**. It is a practice of radical honesty about where a project truly is -- not where we wish it were, or where it is supposed to be. It is a pause to see clearly, without judgment, the current state of our work.

This ritual combats the natural tendency for entropy and confusion to creep into complex projects. It provides a single, trusted source of truth that grounds conversations and decisions in reality. By maintaining this document with care, we cultivate transparency, accountability, and a shared understanding.

A status document is a **dashboard, not a novel**. It should be scannable in 30 seconds. Every word earns its place by providing actionable information. Aspirational language ("we plan to soon...") is replaced by concrete states ("blocked on X", "3 of 7 tasks complete").

---

## II. When to Use This Skill

- **At the beginning of a new project:** To establish the initial state and vision.
- **At the start and end of a work session:** To frame the day's work and document its outcome.
- **During a weekly sync:** To facilitate a high-level review of all active projects.
- **Whenever there is a significant change in project status:** A new blocker, a milestone reached, a risk materialized.
- **After completing a health audit:** To reflect the latest health assessment in the status document.
- **Before a stakeholder meeting:** To prepare an honest, data-backed status summary.

**When NOT to use:** For detailed technical documentation (use architecture docs). For tracking individual tasks (use a project tracker). For historical audit records (those go in `docs/audits/`).

---

## III. The Status Writing Workflow

### Step 1: Gather Context

**Goal:** Collect current project state from all available sources.

1. **Repository data:** Read recent commits, open pull requests, CI status, branch activity. If `~~repository` is connected, use it.
2. **Project tracker data:** Pull sprint progress, active issues, upcoming milestones. If `~~project tracker` is connected, use it.
3. **Prior audits:** Check `docs/audits/` for the most recent health audit and behavioral architecture map.
4. **Existing STATUS.md:** If one exists, read it to understand what has changed since the last update.
5. **Direct observation:** Navigate the codebase and assess the current state firsthand.

### Step 2: Write the STATUS.md

**Goal:** Produce a concise, honest status document with the following sections.

#### Section 1: Header

```markdown
# [Project Name] Project Status

**Status:** [Active & Evolving | On Hold | Complete]
**Last Updated:** [YYYY-MM-DD]
```

#### Section 2: Current State

A one-paragraph summary of where the project stands right now. Be honest and specific. Use concrete metrics where available (e.g., "12 of 18 API endpoints implemented", not "good progress on the API").

#### Section 3: Recent Progress

What happened in the last 1-2 weeks. Bullet points only. Include:
- Features shipped or completed
- Bugs fixed
- Infrastructure improvements
- Documentation updates
- Key decisions made

#### Section 4: Behavioral Architecture

An abbreviated semantic cluster summary showing the project's main capabilities. If a `/map-system` output exists in `docs/audits/`, reference it. Otherwise, provide a brief list:

```markdown
| Capability | Status | Components |
|---|---|---|
| CONVERSE | GREEN | Chat UI, WebSocket handler, message store |
| PERSIST | YELLOW | PostgreSQL, migrations (3 pending) |
| PROTECT | RED | Auth middleware (incomplete), no rate limiting |
```

#### Section 5: Health Dashboard

Rate 5 dimensions using the health audit framework:

```markdown
| Dimension | Status | Notes |
|---|---|---|
| Critical Issues | [GREEN/YELLOW/RED] | [one-line summary] |
| Security | [GREEN/YELLOW/RED] | [one-line summary] |
| Testing | [GREEN/YELLOW/RED] | [one-line summary] |
| Technical Debt | [GREEN/YELLOW/RED] | [one-line summary] |
| Documentation | [GREEN/YELLOW/RED] | [one-line summary] |
```

If a recent `/health-audit` exists, use those results. Otherwise, do a quick assessment.

#### Section 6: Active Risks

What could go wrong. Be ruthlessly honest:
- Technical risks (fragile code, missing tests, security gaps)
- Process risks (blocked dependencies, unclear ownership)
- External risks (third-party service changes, API deprecations)

Each risk should include: what it is, likelihood (high/medium/low), and impact if it materializes.

#### Section 7: Next Milestones

Upcoming deliverables with dates (if known). Be specific about what "done" looks like for each milestone. Use measurable criteria, not vague descriptions.

### Step 3: Save and Commit

Save as `STATUS.md` at the project root. If a previous STATUS.md exists, overwrite it -- status documents are living documents, not historical records.

**Commit message convention:** `docs(status): Update [Project Name] status for [YYYY-MM-DD]`

---

## IV. Best Practices

### 1. Be Honest

The value of this document is its truthfulness. Do not sugarcoat bad news. A STATUS.md that hides problems is worse than no STATUS.md at all.

### 2. Be Concise

Use bullet points and short sentences. This is a dashboard, not a novel. If a section takes more than 30 seconds to read, it's too long.

### 3. Use Visual Status Indicators

The GREEN/YELLOW/RED system provides instant visual summary. Use it consistently across all status sections so readers can scan for trouble spots.

### 4. Update Regularly

A stale status document is worse than no status document. Make it a habit -- at minimum, update weekly. Daily updates during active development sprints are ideal.

### 5. Focus on State, Not Effort

Report what IS, not what you DID. "Authentication is 70% complete, missing password reset flow" is more useful than "Spent 3 days working on authentication."

### 6. Reference, Don't Duplicate

If a health audit exists, reference it ("See `docs/audits/2026-02-11_health_audit.md`") rather than copying its contents into the status document.

---

## V. Quality Checklist

Before delivering a STATUS.md, confirm:

- [ ] Current State section is honest and specific, not aspirational
- [ ] Recent Progress covers the last 1-2 weeks with concrete items
- [ ] Behavioral Architecture section reflects actual system capabilities
- [ ] Health Dashboard uses GREEN/YELLOW/RED for all 5 dimensions
- [ ] Active Risks section exists and is not empty
- [ ] Next Milestones are specific with measurable "done" criteria
- [ ] The entire document is scannable in under 60 seconds
- [ ] Status emoji key is consistent throughout
- [ ] If prior audits exist, they are referenced rather than duplicated
- [ ] Last Updated date is today's date

---

## VI. Common Pitfalls

### Pitfall 1: Aspirational Instead of Honest

**Problem:** Writing "we're on track" when 3 of 5 milestones are overdue and testing is RED.

**Solution:** Report what IS, not what you wish. The status document's value is its honesty. Use data: "3 of 5 milestones overdue. Testing coverage at 23% (target: 80%)."

### Pitfall 2: Too Long

**Problem:** The STATUS.md is 3 pages of detailed explanations that nobody reads.

**Solution:** Each section should be scannable in 10-15 seconds. Total document should be scannable in 60 seconds. Move detail to referenced audit documents.

### Pitfall 3: Stale Status

**Problem:** The STATUS.md was last updated 6 weeks ago and no longer reflects reality.

**Solution:** Make updating STATUS.md part of your workflow rhythm. At minimum: update after every health audit, every sprint completion, and every major status change.

### Pitfall 4: Missing Risks Section

**Problem:** Everything looks green and rosy because risks are omitted.

**Solution:** Always include at least 2-3 risks, even when things are going well. "No known risks" is almost never true -- it usually means risks haven't been identified yet.

### Pitfall 5: No Health Dashboard

**Problem:** The status document describes features but doesn't assess health dimensions.

**Solution:** The health dashboard (5 dimensions with GREEN/YELLOW/RED) is required. It provides the system-level view that feature lists miss. A project can be "feature complete" and still unhealthy.

---

## VII. Example: Project Status Report

**The Problem:** A 3-month-old project needs a status report for a stakeholder meeting. No STATUS.md exists yet.

**The Process:**

1. **Gathered context:** Read git log (142 commits over 3 months), checked CI (green for 2 weeks), read the most recent health audit from `docs/audits/`.

2. **Wrote STATUS.md:**
   - Current State: "MVP complete. 12 of 18 planned API endpoints implemented. Frontend deployed to staging."
   - Recent Progress: 4 features shipped, 2 bugs fixed, CI pipeline stabilized.
   - Behavioral Architecture: 7 active clusters (CONVERSE, REASON, PERSIST, PRESENT, PROTECT, BUILD, ACT).
   - Health Dashboard: Critical GREEN, Security YELLOW (missing rate limiting), Testing YELLOW (62% coverage), Tech Debt GREEN, Documentation YELLOW (API docs incomplete).
   - Active Risks: 3 items including "Authentication rate limiting not implemented -- vulnerability to brute force."
   - Next Milestones: 2 items with dates and acceptance criteria.

3. **Saved:** `STATUS.md` at project root.

**The Outcome:** Stakeholder meeting had a clear, honest foundation. The YELLOW security rating prompted immediate prioritization of rate limiting implementation.

---

## VIII. Related Skills

- **`health-audit`** -- Produces the health dashboard data that feeds into STATUS.md
- **`semantic-clusters`** -- Produces the behavioral architecture summary for Section 4
- **`documentation-audit`** -- STATUS.md is one of the documents that should be maintained and audited
