---
name: claude-md-guardian
model: sonnet
description: "Produces a CLAUDE.md health report listing conflicts, redundancies, and stale rules across the global, project, and subdirectory hierarchy — and optionally installs a PreToolUse hook to block unauthorized modifications. Use when: \"agent behavior is inconsistent\", \"just merged branches that touched CLAUDE.md\", \"onboarding a new agent\", \"after applying learnings\", \"weekly maintenance\"."
license: proprietary
category: system-health
triggers:
  - "agent behavior is inconsistent"
  - "merged branches that touched CLAUDE.md"
  - "onboarding a new agent"
  - "after applying learnings"
  - "weekly maintenance"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: context
    type: string
    description: Trigger context — branch merge, onboarding, weekly maintenance, or behavior inconsistency
    required: false
outputs:
  - name: health_report
    type: string
    description: CLAUDE.md health report listing conflicts, redundancies, and stale rules across global, project, and subdirectory hierarchy
---
# CLAUDE.md Guardian

## I. Philosophy

CLAUDE.md is the agent's behavioral contract. When rules drift, conflict, or
accumulate without review, agent behavior becomes unpredictable. The guardian
skill treats CLAUDE.md as a living document that requires the same rigor as
source code: validation, consistency checks, and protection against unreviewed
modification.

Three failure modes threaten CLAUDE.md health:
1. **Drift** -- Rules added to one CLAUDE.md but not propagated to related files.
2. **Conflict** -- Contradictory rules across global, project, and subdirectory files.
3. **Bloat** -- Accumulated rules that are no longer relevant or are redundant.

Mechanical enforcement (hooks) prevents unauthorized modification. Periodic
audits catch semantic drift that mechanics cannot detect.

## II. When to Use

- After applying learnings from `reflect-and-learn` to verify no conflicts were introduced.
- During periodic maintenance (weekly or sprint boundaries).
- When agent behavior deviates unexpectedly (first suspect: CLAUDE.md conflict).
- Before onboarding a new team member or agent to a project.
- After merging branches that may have modified CLAUDE.md independently.

Do not use this skill for writing new CLAUDE.md rules (use `reflect-and-learn`
for correction-based rules, or edit directly). Do not use it for auditing
documentation broadly (use `documentation-audit`).

## III. Workflow

**Step 1 -- Inventory all CLAUDE.md files.**

Locate every behavioral configuration file:
- `~/.claude/CLAUDE.md` (global)
- `./CLAUDE.md` (project root)
- `./subdirectory/CLAUDE.md` (subdirectory overrides)
- `./AGENTS.md` (cross-tool compatibility)

**Step 2 -- Extract and normalize rules.**

Parse each file into discrete behavioral rules. Normalize phrasing to detect
semantic duplicates and contradictions. Flag:
- Duplicate rules across files (redundancy)
- Contradictory rules (conflict)
- Rules that reference deprecated tools, paths, or patterns (staleness)

**Step 3 -- Check inheritance hierarchy.**

CLAUDE.md files form a cascade: global < project < subdirectory. Verify:
- Subdirectory rules do not contradict project rules unintentionally
- Project rules do not repeat global rules unnecessarily
- Override intent is explicit, not accidental

**Step 4 -- Configure mechanical protection (optional).**

Set up a PreToolUse hook that intercepts Write/Edit operations targeting
CLAUDE.md files. The hook can:
- Log all modification attempts for audit trail
- Block modifications from sub-agents (only primary agent may modify)
- Require a specific confirmation pattern before allowing changes

**Step 5 -- Report findings.**

Produce a health report listing:
- Total rule count per file
- Conflicts found (with file locations and rule text)
- Redundancies identified
- Staleness candidates
- Recommended actions (merge, delete, promote, demote)

## IV. Best Practices

- Run the guardian after every `reflect-and-learn` session.
- Keep global CLAUDE.md under 50 rules. Overflow to project-level files.
- Use explicit override markers when subdirectory rules intentionally contradict
  project rules ("OVERRIDE: ...").
- Version CLAUDE.md in git to track rule evolution over time.
- Cross-reference CLAUDE.md rules with AGENTS.md for cross-tool consistency.

## V. Quality Checklist

- [ ] All CLAUDE.md files inventoried (global, project, subdirectory)
- [ ] No unintentional contradictions between hierarchy levels
- [ ] Redundant rules identified and consolidated
- [ ] Stale rules flagged for removal
- [ ] Mechanical protection hook installed (if appropriate)
- [ ] Health report generated with actionable recommendations

## VI. Common Pitfalls

- **Treating all contradictions as bugs.** Some subdirectory overrides are intentional.
- **Over-guarding.** Blocking all CLAUDE.md modifications prevents legitimate learning.
- **Ignoring AGENTS.md.** Cross-tool files can contradict CLAUDE.md silently.
- **Manual-only audits.** Without hooks, modifications between audits go undetected.
- **Rule count as quality metric.** More rules does not mean better behavior.

## VII. Related Skills

- `reflect-and-learn` -- Applies learnings that this skill then validates
- `documentation-audit` -- Broader documentation health (not CLAUDE.md-specific)
- `hooks-reference` -- Configure the mechanical protection hooks
- `health-audit` -- Repository-wide health check (includes CLAUDE.md as one dimension)

## Output
- Health report: total rule count per file, list of conflicts with file locations and rule text, redundancies identified, staleness candidates, and recommended actions (merge, delete, promote, demote).
- Optional: a PreToolUse hook script and settings.json registration blocking unauthorized CLAUDE.md edits from sub-agents.

## Examples
**Scenario 1:** "Agent started ignoring my no-emoji rule after a merge" → Guardian inventories all CLAUDE.md files, finds a subdirectory rule that overrides the project-level rule unintentionally, and recommends adding an explicit OVERRIDE marker or removing the duplicate.
**Scenario 2:** "Weekly CLAUDE.md maintenance" → Identifies 3 rules referencing a deprecated tool path, 2 redundant rules between global and project files, and recommends consolidation. No conflicts found. Hook already installed from prior run.

## Edge Cases
- Intentional subdirectory overrides that genuinely contradict project rules are not bugs — confirm intent before flagging as conflicts. Check for explicit "OVERRIDE:" markers.
- AGENTS.md files can contradict CLAUDE.md silently; always include them in the inventory even if the user only mentions CLAUDE.md.

## Anti-Patterns
- Treating rule count as a quality metric — more rules does not mean better agent behavior; favor fewer, precise rules.
- Blocking all CLAUDE.md modifications via hook without a human override path — prevents legitimate learning and creates operational friction.
