# kata-ai/v1

The contract between Kata (the app) and the DojoGenesis gateway for AI-backed skills. Defined by DGS-149; the ruling is in `specs/rollgoal-parity/05-ai-gateway-contract-scout.md` section 5.

13 skills: the 11 `kata-ai-*` ops plus the two existing skills `kata-overwhelm` and `kata-roll-seed`.

## Layout

```
_defs.schema.json            shared fragments (enums, TaskDraft, Milestone); human source of truth
lint.json                    shame-aware voice lint rules
<skill-id>/input.schema.json
<skill-id>/output.schema.json
<skill-id>/fixtures/NN-<slug>.json
```

Fixture shape: `{ description, input, valid_output, invalid_outputs: [{ why, output }] }`. Each `why` starts with `schema:` (must fail the output schema) or `lint:` (must pass the schema but trip a lint rule). Every `valid_output` must pass its schema and trip no lint rule.

## Rules

- JSON Schema draft 2020-12. Outputs are always an object at the root, because the gateway uses the output schema verbatim as a function-tool `parameters` object.
- Output schemas inline the shared defs (no `$ref`, no `oneOf`/`anyOf`) so they work as tool parameters across providers. `_defs.schema.json` is the human source; if it changes, update every output that inlines the fragment. Input schemas `$ref` it.
- `additionalProperties: false` everywhere. Sizes are capped on purpose (steps 1-7, milestones 1-10, tasks 1-12, titles at 120 characters, steps at 200, intention at 160).
- `user_id` appears only in `kata-overwhelm` and `kata-roll-seed`. The 11 `kata-ai-*` ops get no user_id and no memory context in v1.
- Enums mirror `src/types/enums.ts` and `src/types/work.ts`. `priority` on a goal is the `GoalPriority` string enum, not a number.

## Gateway vendoring

The gateway keeps a copy of this directory and runs a CI check that the file hashes match Kata's. It validates every model response against the output schema, and the same fixtures drive Kata's parser contract tests.

## Lint rules

`lint.json` applies to every string value in an output, recursively. Patterns are RE2 compatible (Go `regexp`): no lookarounds. Rules:

no-exclamation, no-dont-worry, no-youve-got-this, no-you-should, no-you-didnt, no-you-failed, no-lazy, no-just-do-it, no-simply, no-medical-diagnos, no-medical-treat, no-medical-cure, no-medical-symptom.

## Versioning

A breaking change (removed or renamed key, narrowed enum, tighter bound that rejects previously valid output) goes in a new `v2/` directory. v1 stays frozen. Additive, backward-compatible loosening may land in v1.
