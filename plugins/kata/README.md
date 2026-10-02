# kata — first-party plugin

The prompts behind Kata's AI features (ADHD / executive-function scaffolding, DojoGenesis). There are 13 hidden, structured-output skills, one per operation of the `kata-ai/v1` contract:

| Skill | Output envelope |
|---|---|
| `kata-ai-decompose-goal` | `{milestones}` |
| `kata-ai-draft-goal` | `{goal}` |
| `kata-ai-tasks-from-goal` | `{tasks}` |
| `kata-ai-task-chain` | `{tasks}` |
| `kata-ai-goal-ladder` | `{order}` |
| `kata-ai-unstick-breakdown` | `{steps}` |
| `kata-ai-sprint-intention` | `{intention}` |
| `kata-ai-optimize-project` | `{suggestions}` |
| `kata-ai-celebration-pack` | `{pack, celebrations}` |
| `kata-ai-draft-routine` | `{routine, tasks}` |
| `kata-ai-triage-brain-dump` | `{items}` |
| `kata-overwhelm` | `{steps}` |
| `kata-roll-seed` | `{seed}` |

Each `SKILL.md` body is the system prompt, sent verbatim. The user message is the op's input JSON. The model answers by calling a single function, `emit_result`, whose parameters are the op's output schema.

## Contract

The schemas are owned by Kata at `Kata/contracts/kata-ai/v1/` (input + output JSON Schema per op, plus golden fixtures). `plugins/kata/contracts/` is a vendored copy; the integrator syncs it, so don't edit it here. Each skill's `input_schema` / `output_schema` frontmatter points into that copy.

## These prompts are product copy

Shame-aware wording is Kata's product, and it lives in these files. Every change is reviewed against Kata's brand voice and non-negotiables (`Kata/CLAUDE.md`). Read every line the model could produce and ask: would a user in the middle of overwhelm feel shamed by this? Bans that apply to every prompt: exclamation marks, the brand-voice banned phrases, streak or guilt framing, and medical or diagnostic language. User-supplied text is always treated as data, never as instructions.

## Versioning

- **Wording change** (same output shape): bump the skill's `version:` by a patch or minor step, e.g. `1.0.0` → `1.0.1` for a fix or `1.1.0` for a reworked prompt.
- **Output shape change** (a field added, removed, renamed, or retyped): it's a new contract version (`kata-ai/v2`) with new schemas in Kata, and v1 keeps being served for binaries already shipped. Never change a v1 shape in place.

Every skill name here is reserved (`reserved-names.txt`, ADR-020) and packages only in first-party mode.
