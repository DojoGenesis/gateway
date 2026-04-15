---
name: reflect-and-learn
model: sonnet
description: "Captures user corrections from a session, validates them semantically, and writes approved learnings to the most specific persistence target (skill file, project CLAUDE.md, or global CLAUDE.md). Use when: \"remember this for next time\", \"don't do that again\", \"update your behavior\", \"capture this correction\", \"reflect on what we learned\"."
license: proprietary
category: wisdom-garden
triggers:
  - "remember this for next time"
  - "don't do that again"
  - "update your behavior"
  - "capture this correction"
  - "reflect on what we learned"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: corrections
    type: string[]
    description: User corrections or learnings from the session to validate and persist
    required: true
outputs:
  - name: persisted_learnings
    type: string
    description: Approved learnings written to the most specific persistence target — skill file, project CLAUDE.md, or global CLAUDE.md
---
# Reflect and Learn

## I. Philosophy

Every user correction is a gift -- it reveals the gap between what the agent
does and what the operator needs. But corrections are ephemeral: they live in
conversation context that compacts away. This skill transforms ephemeral
corrections into permanent behavioral improvements by capturing them in
real-time, validating them with semantic analysis, and routing them to the
right persistence target (CLAUDE.md, skill files, or AGENTS.md).

The key insight from claude-reflect: corrections made during skill execution
should route back to the skill file, not just global CLAUDE.md. A correction
during `/deploy` improves the deploy skill; a correction about code style
improves the global config.

## II. When to Use

- At the end of a productive session where several corrections were made.
- When the user says "remember this for next time" or "don't do that again."
- Periodically (weekly) to discover recurring workflow patterns across sessions.
- After a skill is used multiple times and the user has refined its behavior
  through corrections.
- When onboarding to a new project and accumulating project-specific preferences.

Do not use this skill for extracting strategic seeds from experiences (use
`seed-extraction`). Do not use it for compressing conversation history (use
`compression-ritual`).

## III. Workflow

**Step 1 -- Review captured corrections.**

Examine the correction queue (populated by hooks or manual review). For each
captured correction, assess:

- **Confidence level** (0.60-0.95): Is this a genuine correction or conversational noise?
- **Scope**: Global (all sessions), project-specific, or skill-specific?
- **Actionability**: Can this be expressed as a clear behavioral rule?

Correction signals to look for:
- Direct negations: "no, use X" / "don't use Y"
- Clarifications: "actually..." / "that's wrong"
- Explicit markers: "remember:" (highest confidence)

**Step 2 -- Validate with semantic analysis.**

For each candidate correction:
- Process through semantic filter to eliminate false positives
- Extract distilled, actionable statement from noisy context
- Assign final confidence score: max(regex_score, semantic_score)
- Discard anything below 0.60 confidence threshold

**Step 3 -- Route to persistence target.**

Determine the correct destination for each approved learning:

| Signal | Target |
|--------|--------|
| General preference (formatting, style) | Global CLAUDE.md |
| Project-specific behavior | Project CLAUDE.md |
| Correction during skill execution | The skill's SKILL.md or command file |
| Cross-tool compatibility need | AGENTS.md |

**Step 4 -- Apply and verify.**

Write the learning to the target file. Verify it does not conflict with existing
rules. If a conflict exists, surface it to the user for resolution.

**Step 5 -- Discover recurring patterns (optional).**

Analyze session history (14+ days) for semantically similar requests:
- Group related workflows by project context
- Identify candidates for new skill creation
- Generate draft skill files with evidence trails and embedded corrections

## IV. Best Practices

- Capture is automatic; application is always manual with human review.
- Route corrections to the most specific target (skill file > project CLAUDE.md > global).
- Queue must survive compaction -- back up before PreCompact events.
- Session start should display pending (unreviewed) learnings.
- After git commits, prompt for reflection (natural pause point).
- Keep confidence threshold at 0.60 minimum to filter noise.

## V. Quality Checklist

- [ ] Corrections are captured with confidence scores
- [ ] Semantic validation eliminates false positives
- [ ] Routing targets the most specific persistence location
- [ ] No conflicts with existing CLAUDE.md rules
- [ ] Queue persists across session boundaries
- [ ] User reviewed and approved each applied learning

## VI. Common Pitfalls

- **Auto-applying without review.** Regex captures sarcasm and hypotheticals.
- **Only routing to global CLAUDE.md.** Skill-specific corrections get diluted.
- **Skipping semantic validation.** False positive rate from regex alone is too high.
- **Waiting too long to reflect.** Correction context degrades after compaction.
- **Treating affirmations as corrections.** "Perfect!" is reinforcement, not change.

## VII. Related Skills

- `seed-extraction` -- Extract strategic patterns (seeds), not behavioral corrections
- `compression-ritual` -- Compress conversation history into memory artifacts
- `memory-garden` -- Write structured memory entries for context management
- `hooks-reference` -- Configure capture hooks for automatic correction detection
- `claude-md-guardian` -- Protect CLAUDE.md integrity during learning application

## Output

- One or more corrections written to the appropriate target file (skill SKILL.md, project CLAUDE.md, or global CLAUDE.md) as new behavioral rules
- A short review summary listing: corrections evaluated, confidence scores, routing decisions, and any conflicts surfaced for user resolution

## Examples

**Scenario 1:** User says "remember — always dry-run scripts before running them" → correction assigned confidence 0.90 (explicit marker), routed to global CLAUDE.md, written as a new behavioral rule, summary returned.

**Scenario 2:** During a `/deploy` skill execution the user says "don't add the --force flag here" → correction identified as skill-specific (deploy context), routed to the deploy skill's SKILL.md, written under a "Corrections" section, summary returned.

## Edge Cases

- If the correction conflicts with an existing rule in the target file, surface the conflict to the user for resolution before writing — do not overwrite silently.
- If confidence is below 0.60 (sarcasm, hypotheticals, conversational noise), discard the candidate and note it in the summary.

## Anti-Patterns

- Routing all corrections to global CLAUDE.md — skill-specific corrections must go to the skill file or they get diluted across all contexts.
- Auto-applying corrections without user review — capture is automatic but application always requires explicit approval.
