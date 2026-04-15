---
name: session-lifecycle-automation
model: opus
description: Designs and configures all three loops of solo-operator session automation — session bookends, background maintenance agents, and a delegation flywheel. Use when setting up a new workspace, recovering from the 44% zero-artifact session leak, or preparing to scale delegation across multiple concurrent agents.
category: wisdom-garden
version: 1.0.0
tags: [session, automation, lifecycle, scheduling]
triggers:
  - "set up session automation"
  - "configure session bookends"
  - "delegation flywheel"
  - "session lifecycle setup"
  - "recovering from zero-artifact session leak"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: workspace_path
    type: string
    description: Root path of the workspace to audit and configure automation for
    required: true
outputs:
  - name: automation_config
    type: ref
    format: markdown
    description: Configured session bookend commands, scheduled agent definitions, and commission directory with template files
---
# Session Lifecycle Automation Skill

## Philosophy

Analysis of 117 sessions revealed the core problem: 44% of sessions produced zero lines of code. Their insights, decisions, and unresolved questions lived only in the chat transcript — and were lost at session close. A solo operator's only scarce resource is attention. Every session that closes without harvesting its output is a compound leak: the next session re-learns, re-decides, and re-discovers.

The solution is not discipline — discipline without hooks drifts to 0% compliance. The solution is three automation loops that run mechanically regardless of whether the operator remembers to invoke them.

All three loops reinforce each other. Background agents discover stale state → they create commission files → commissions produce findings → session bookends deposit findings to memory → background agents tend that memory. The flywheel spins on its own once initialized.

## When to Use

- When setting up a new workspace or onboarding a new project to the Dojo ecosystem
- When noticing re-learning across sessions — spending time at session start rediscovering decisions made in prior sessions
- When background maintenance (supply chain sync, data freshness, memory tending) is being done manually and inconsistently
- When delegation to agents is happening ad hoc (paste-the-context style) rather than from a structured commission directory
- When reviewing productivity and finding that sessions close without clear artifacts

Do not use this skill to run a single wrap-up session — use `/wrap-up` or `compression-ritual` for that. This skill is for configuring the infrastructure that makes those tools automatic.

## Workflow

### Phase 1: Audit Current Automation State

**Objective:** Understand what is and is not already automated before adding anything.

1. Check `~/.claude/settings.json` for existing hooks on `PostToolUse`, `PreToolUse`, `Stop`, and `UserPromptSubmit` events.
2. List scheduled tasks (cron-style triggers) already configured for this workspace.
3. Check whether a commission directory exists (typically `{workspace}/commissions/` or `AgenticStackOrchestration/commissions/`).
4. Identify the dead zones in the operator's schedule — times when no sessions are active and background agents can safely run without contention.
5. Produce an audit summary: Loop 1 configured (Y/N), Loop 2 configured (Y/N + which agents), Loop 3 configured (Y/N + commission count).

### Phase 2: Configure Loop 1 — Session Bookends

**Objective:** Ensure every session >15 minutes ends with a harvest of decisions, surprises, and open questions.

1. Identify the wrap-up command available in the workspace (`/wrap-up`, `compression-ritual`, or a custom variant).
2. Add a `Stop` hook in `settings.json` that fires when a session ends — it should remind or automatically trigger the wrap-up:
   ```json
   {
     "hooks": {
       "Stop": [{
         "matcher": "",
         "hooks": [{
           "type": "command",
           "command": "echo '[SESSION END] Run /wrap-up to harvest decisions before closing.'"
         }]
       }]
     }
   }
   ```
3. Define what the wrap-up must capture at minimum:
   - **Key decisions made** (not just what was built, but why)
   - **Surprises** (anything that didn't go as expected — these are the highest-signal seeds)
   - **Open questions** (unresolved items that need next-session attention)
4. Verify the wrap-up output lands in the memory garden (typically `~/.claude/projects/{workspace}/memory/`) and is committed.
5. Test by running a short session and verifying the artifact appears in memory after the Stop hook fires.

### Phase 3: Configure Loop 2 — Background Maintenance

**Objective:** Run maintenance agents in dead zones so the operator never returns to stale state.

1. From the Phase 1 audit, identify the three most critical maintenance tasks that are currently manual and inconsistent. Common candidates:
   - **Memory tending** (midday MWF): reads recent session seeds, reconciles MEMORY.md index, prunes stale entries
   - **Supply chain sync** (Sunday mornings): audits skill versions, flags outdated community CAS entries
   - **Data freshness check** (Wednesday): pings data endpoints, flags APIs that have gone stale or changed schema
2. For each maintenance task:
   a. Write a commission file that the background agent can execute with no interactive input (see Phase 4 for format).
   b. Schedule the agent at the identified dead zone time using the scheduled task system.
   c. Configure the agent output to be deposited to memory or to a status file that the next session can read.
3. Use Sonnet for all background maintenance agents — they are parsing/bulk/template work, not strategic decisions.
4. Verify scheduling: confirm each task has a unique, non-overlapping window. Overlapping background agents contend on the same files and produce merge conflicts.

### Phase 4: Configure Loop 3 — Delegation Flywheel

**Objective:** Build a commission directory so every agent dispatch is fire-and-forget.

1. Create or verify the commission directory at `{workspace}/commissions/` (or the project-specific equivalent).
2. Each commission file must contain — and only contain — everything the receiving agent needs to complete the task autonomously:
   - **Objective:** one sentence
   - **File manifest:** exact paths the agent is allowed to read and write — no others
   - **Constraints:** what it must NOT do (do not modify tests, do not push to remote, etc.)
   - **Definition of done:** binary, testable criteria (build passes, N files created, specific grep matches)
   - **Model:** `sonnet` or `opus` — never inherit default
3. Never write commission files that rely on the receiving agent's memory or prior context — sub-agents start cold.
4. After any session that produces delegation candidates, write the commission files before closing. This is the most important habit change: commissions are written when the context is hot, not reconstructed cold next session.
5. Verify the flywheel: run one commission cold (no context given) and confirm the agent reaches definition-of-done without back-and-forth.

### Phase 5: Verify the Integrated Loop

**Objective:** Confirm all three loops reinforce each other rather than running in isolation.

1. Trace a complete cycle: background maintenance agent discovers stale data → writes a finding to a commission file → session bookend at next operator session picks it up and deposits it to memory → next background maintenance agent reads the updated memory.
2. If any step in the chain fails to hand off to the next, identify the gap and fix it before declaring the automation configured.
3. Document the cycle in a single diagram or bullet list in the workspace's CLAUDE.md under an "Automation" section.

## Output

- `~/.claude/settings.json` updated with Stop hook for session bookends (Loop 1).
- Scheduled task entries created for each background maintenance agent (Loop 2), with commission files backing each one.
- Commission directory populated with at minimum one template commission and any existing delegation candidates (Loop 3).
- CLAUDE.md updated with an "Automation" section describing the three loops and their cadences.

## Examples

**Example 1: New workspace onboarding**

Operator sets up a new project with 5 active repos and no automation. Phase 1 audit: no hooks, no scheduled tasks, no commission directory. Phase 2: adds Stop hook to settings.json pointing to `/wrap-up`. Phase 3: identifies three dead zones — MWF 2pm (memory tending), Sunday 10am (supply chain sync), Wednesday 11am (data freshness). Writes commission files for each; schedules Sonnet agents. Phase 4: creates `commissions/` directory with 3 existing delegation candidates (D1 migration, PDI BLS re-run, HTMLCraft v3.5 polish) that were sitting as open items in CLAUDE.md. Phase 5: traces the cycle — background memory agent reads new session seeds from Sunday → deposits summary → next morning session starts with clean MEMORY.md index instead of 40-session backlog.

**Example 2: Recovering from session leak**

After reviewing a week of sessions, the operator notices 6 of 14 sessions closed with no artifacts in memory. Phase 1 audit: Stop hook exists but is a passive echo only (no actual wrap-up trigger). Phase 2: upgrades Stop hook to auto-run `/wrap-up` and fail loudly if the output file is not created. Phase 3: adds a midday memory-tending agent that was previously skipped because "the session is only 3 hours away." Phase 4: reviews the 6 zero-artifact sessions and reconstructs the decisions from chat history into commission files retroactively — time-consuming once, never again. After one week: zero-artifact session rate drops from 44% to 6% (one session where the hook failed due to a connectivity issue).

## Edge Cases

- If the operator runs multiple concurrent sessions (e.g., parallel agent dispatches in separate terminal windows), the Stop hook fires for each. Ensure wrap-up commands are idempotent — appending to a daily file rather than creating a new file per session prevents duplicate entries.
- If a background maintenance agent discovers a critical issue (e.g., broken build in a repo it was only supposed to read), it should write a high-priority commission file and stop — not attempt the fix itself. Background agents should never have write access to production paths.
- If the commission directory grows beyond 20 files, run a triage pass: kill completed commissions, archive parking-lot items, ensure active commissions have their definition-of-done clearly defined. Commission bloat signals that the flywheel is accepting work faster than it is completing it.

## Anti-Patterns

- **Configuring Loop 1 without Loop 3** — Session bookends harvest decisions into memory, but without commission files, those decisions still require the operator to reconstruct context before delegating. The harvest is only half the value; the other half is making the harvested context immediately dispatchable.
- **Writing commission files during the delegation session itself** — Commission files written cold (at dispatch time) miss 70% of the relevant constraints and context. Write them while the context is hot (at the end of the session that surfaces the need), not when you are trying to fire the agent.
- **Using a single omnibus maintenance agent** — A single "do all maintenance" agent is slow, fragile, and hard to schedule around. Break maintenance into three independent agents with non-overlapping scopes and dead zones. When one fails, the other two still run.
- **Relying on the operator to remember to run wrap-up** — The 44% zero-artifact session leak was measured against operators who intended to run wrap-up. Intention without hooks is not automation. The Stop hook must run mechanically or the loop degrades to manual.
- **Giving background agents write access to source files** — Background maintenance agents read and report. They write only to designated output paths (memory files, commission files, status reports). Granting broader write access to a scheduled agent that runs unsupervised is how silent destructive changes accumulate overnight.
