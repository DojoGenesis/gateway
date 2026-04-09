---
name: status-writing
model: sonnet
description: "Produces an updated STATUS.md file as a single source of truth for project health — covering current state, active workstreams, blockers, and next steps. Use when: 'create a status file', 'update the project status', 'document where we are', 'what is the current state', 'create a status dashboard'."
category: system-health
triggers:
  - "create a status file"
  - "update the project status"
  - "document where we are"
  - "what is the current state"
  - "create a status dashboard"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: project_context
    type: string
    description: Current project context — what was done, what is active, what is blocked
    required: false
outputs:
  - name: status_document
    type: ref
    format: cas-ref
    description: Updated STATUS.md as a single source of truth covering current state, active workstreams, blockers, and next steps
---

# Status Writer Skill

## I. Philosophy

A `STATUS.md` file is more than a report; it is a Ritual of Bearing Witness. It is a practice of radical honesty about where a project truly is — not where we wish it were, or where it is supposed to be. It is a pause to see clearly, without judgment, the current state of our work.

This ritual combats the natural tendency for entropy and confusion to creep into complex projects. It provides a single, trusted source of truth that grounds conversations and decisions in reality. By maintaining this document with care, we cultivate transparency, accountability, and a shared understanding of our journey.

## II. When to Use This Skill

- At the beginning of a new project: to establish the initial state and vision.
- At the start and end of a work session: to frame the day's work and document its outcome.
- During a weekly sync: to facilitate a high-level review of all active projects.
- Whenever there is a significant change in project status (a new blocker emerges, a major milestone is reached).

For the full 10-section comprehensive schema used in handoffs and onboarding, see the `status-template` skill.

## III. The Status Update Workflow

### Step 1: Locate or Create the STATUS.md File

Navigate to the root of the project repository. If a `STATUS.md` file does not exist, create one.

**Decision: is this a new file or an update?**
- New file: initialize all sections with placeholders and fill them using the steps below.
- Existing file: proceed to Step 2, updating only what has changed.

### Step 2: Update the Header

Change the `Last Updated` date to the current date. Do not change the `Author` or `Status` fields unless there is a concrete reason to do so.

### Step 3: Review and Update Each Section

Go through each section with these decision rules:

**Vision and Purpose:** Re-read it. If the purpose has changed, update it. If not, leave it untouched — this section is an anchor, not a changelog.

**Current State:** This is the most important section. For each area, ask:
- Has it moved from its last emoji status? If yes, update the emoji and the note.
- Has a new area emerged that isn't listed? Add a row.
- Has an area been completed or archived? Remove or mark it done.
  Use the emoji key strictly: `✅` complete, `🔄` in progress, `⏸️` paused, `❌` blocked.

**Active Workstreams:** List only what is being worked on right now — not what is planned. If a workstream from last time is complete, remove it. If a new one has started, add it with the current task and progress percentage or description.

**Blockers and Dependencies:** Be ruthlessly honest. If a blocker has been resolved, remove it immediately. If a new one has appeared, add it with:
- What it is blocking
- Who owns resolution
- What the next step is to unblock

**Next Steps:** These should be concrete and actionable — specific enough that a different person (or agent) could pick them up without asking questions. Replace vague items ("improve performance") with specific ones ("run the p95 latency benchmark and open an issue if > 500ms").

### Step 4: Commit the Changes

Commit the updated `STATUS.md` with a clear, descriptive commit message.

**Commit Message Convention:**
`docs(status): Update [Project Name] status for [Date]`

**Example:**
`docs(status): Update Gateway status for 2026-04-08`

## IV. STATUS.md Structure

See the `status-template` skill for the full 10-section `.status.md` schema used for comprehensive handoffs and onboarding.

For the lightweight `STATUS.md` maintained by this skill, the document has 5 sections:

1. **Vision and Purpose** — One sentence + core principles. Rarely changes.
2. **Current State** — Emoji status table of major areas. Updated every session.
3. **Active Workstreams** — What is being worked on right now. Updated every session.
4. **Blockers and Dependencies** — What is preventing progress. Updated when blockers appear or resolve.
5. **Next Steps** — Concrete, actionable items. Updated every session.

## V. Best Practices

- **Be Honest:** The value of this document is its truthfulness. Do not sugarcoat bad news.
- **Be Concise:** Use bullet points and short sentences. This is a dashboard, not a novel.
- **Use the Emoji Key:** The emojis provide an instant visual summary of project health.
- **Update Regularly:** A stale status document is worse than no status document. Make it a habit.
- **Focus on the 'What', Not the 'Who':** The status is about the project, not the people. Frame blockers and issues impersonally.

## VI. Quality Checklist

- [ ] `Last Updated` date reflects today's date
- [ ] Current State table has been reviewed — every row either confirmed current or updated
- [ ] Active Workstreams reflects only what is being worked on right now (not plans)
- [ ] Blockers section is truthful — resolved blockers removed, new ones added
- [ ] Next Steps are concrete and specific enough for handoff
- [ ] Changes committed with the standard commit message convention

## Output
- An updated `STATUS.md` at the project root with 5 sections reflecting the current state of the project.
- A git commit with the `docs(status): Update [Project Name] status for [Date]` message convention.

## Examples
**Scenario 1:** "Update the Gateway status at the end of today's session" → Existing `STATUS.md` updated: one workstream marked complete (removed), one new blocker added (missing deployment key), next steps updated to reflect tomorrow's priorities. Committed.
**Scenario 2:** "Create a status file for the new HTMLCraft Studio project" → New `STATUS.md` initialized with vision statement, empty current state table (all areas at `🔄`), first workstream ("Architecture setup: 0% complete"), no blockers, and 3 concrete next steps.

## Edge Cases
- If the project has no existing STATUS.md and the user needs a comprehensive onboarding document rather than a lightweight status file, redirect to the `status-template` skill for the full 10-section schema.
- If the Vision and Purpose has genuinely changed (e.g., a pivot), update it but note the change in the commit message — vision drift should be visible in the git history.

## Anti-Patterns
- Keeping resolved blockers in the document "for historical reference" — STATUS.md is a live dashboard, not a changelog. Move historical context to CHANGELOG.md or a retrospective.
- Writing aspirational next steps ("Eventually improve test coverage") instead of concrete ones — if it can't be handed off to another agent as-is, it's not specific enough.
