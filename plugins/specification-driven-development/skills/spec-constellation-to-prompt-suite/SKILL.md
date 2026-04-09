---
name: spec-constellation-to-prompt-suite
model: sonnet
description: Produces a suite of coordinated, self-contained implementation prompts and a master integration contract table from a set of interconnected specs. Use when 'convert specs to prompts', 'ready to commission these specs', or 'coordinate these parallel tracks into executable prompts'.
category: specification-driven-development
triggers:
  - "convert specs to prompts"
  - "ready to commission these specs"
  - "coordinate these parallel tracks into executable prompts"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec_files
    type: string[]
    description: Paths to interconnected specification files to convert into prompts
    required: true
outputs:
  - name: prompt_suite
    type: ref
    format: cas-ref
    description: Suite of coordinated, self-contained implementation prompts and master integration contract table
---

## I. The Philosophy

When you move from one specification to one implementation prompt, you're writing a focused brief. But when you move from four interconnected specifications to four parallel implementation prompts, you're solving a **coordination problem**.

The danger: each prompt looks coherent in isolation, but the tracks execute independently. Track B (Go backend) produces the API that Track C (frontend) consumes—but if the two prompts define the contract differently, integration fails. Track A (scaffold) defines design tokens that Track D (dock + home) must use—but inconsistent naming breaks everything. Type definitions (DojoEntity, PipelineContent) must be identical across tracks, not compatible or "close enough."

This skill exists because:
- Writing N prompts from N specs is not N repetitions of "write one prompt"
- The coordination happens at the integration boundaries, not within prompts
- Shared types and contracts must be defined BEFORE prompts are written
- Cross-validation is the difference between "tracks that exist" and "tracks that integrate"

## II. When to Use This Skill

- You have 2+ interconnected specifications covering different aspects of a release
- Prompts for different tracks must agree on APIs, types, exports, or design tokens
- You're moving from "specs documented" to "ready to commission parallel work"
- One track's output becomes another track's input (explicit handoff required)
- You need a master plan that ties implementation work together
- Risk: Tracks ship independently without realizing their contracts don't align

## III. Prerequisites

- A master plan or release plan defining the release scope
- 2+ specifications covering different aspects of the release
- A parallel tracks decomposition (or the intent to create one)
- Codebase patterns extracted for grounding (handler patterns, response formats, type definitions)
- At least one existing codebase to extract patterns from

## IV. The Workflow

### Step 1: Map the Spec Constellation

List all specs that contribute to this release. For each spec, identify:
- Which aspects map to which track(s)
- Which other specs it depends on or feeds
- What decisions or scouts inform it

Create a matrix:
| Spec | Contributes To | Dependencies | Key Decision |
|------|---|---|---|
| Entity Data Model | Track B, C, D | — | Entity CRUD shape |
| Tauri Shell | Track A | — | Platform scaffold |
| Horizon Dock | Track D | Tauri Shell, Entity Model | UI layout |
| Home State | Track D | Entity Model | State shape |

**Key insight:** Specs don't map 1:1 to tracks. One spec feeds multiple tracks; multiple specs feed one track.

### Step 2: Define Integration Interfaces

For each pair of tracks that share data, define the contract in this format:

**Integration Contract Table:**
| Interface | Produced By | Consumed By | Shape | Verification |
|---|---|---|---|---|
| EntityAPI | Track B | Track C | `listEntities(route): Promise<DojoEntity[]>` | Type check against export |
| DojoEntity type | Track B | Track C, D | TypeScript interface | Shared in all 3 prompts identically |
| Route handlers | Track A | Track D | Path format: `/entity/:id` | Navigation code type-checks |

Write each contract in the Integration Contract Template below. **This table becomes the spine of the master plan.**

### Step 3: Extract Codebase Patterns

For each track that touches existing code:

- **Backend tracks:** Extract handler patterns, request/response shapes, middleware hooks
- **Frontend tracks:** Extract component folder structure, hook patterns, state management shapes
- **All tracks:** Response formats, error handling, logging conventions

Store patterns as reference files in the prompt (or link to them). Ground each prompt in "this is how we actually do it," not assumptions.

### Step 4: Write Prompts in Dependency Order

Begin with the track that has **no dependencies** (often scaffold or backend). For each prompt, include:

1. **Context & Grounding:** Which specs to read, which pattern files to reference
2. **Detailed Requirements:** Numbered, specific, unambiguous
3. **File Manifest:** Create vs. modify; where files go
4. **Success Criteria:** Binary, testable (not "looks good")
5. **Constraints & Non-Goals:** What NOT to do
6. **Integration Contracts:** What this track produces; what it depends on (reference Step 2)

**When writing a consuming track's prompt**, cite the producing track's contract explicitly:
> "Track B produces the DojoEntity type (Integration Contract: DojoEntity type). Use that type exactly in Track C's API client."

### Step 5: Cross-Validate Type Consistency

For every shared type (DojoEntity, PipelineContent, etc.):
- Check that the definition appears identically in every prompt that references it
- If a type evolves, update it in ALL prompts and the Integration Contract Table

For every API contract:
- Verify producer's response shape matches consumer's expected input shape
- Check HTTP method, status codes, error responses

For shared resources (CSS variables, design tokens, route patterns):
- Verify naming is consistent across all prompts

**Catch-and-fix:** This step prevents integration bugs.

### Step 6: Write the Master Implementation Plan

Summarize all tracks, dependencies, and integration points:

1. List all tracks in dependency order
2. Include the Integration Contract Table from Step 2
3. Define the day-by-day execution sequence
4. List aggregate success criteria spanning tracks
5. Note risk mitigation (e.g., "Track B ships Friday; Track C unblocked Monday")

## V. Integration Contract Template

```markdown
### Integration Contract: [Name]

- **Producer:** Track [X] — [brief description of what produces this]
- **Consumer:** Track [Y] — [brief description of what consumes this]
- **Interface:** [API endpoint / TypeScript type / CSS variable / Go struct / etc.]
- **Language:** [TypeScript / Go / CSS / etc.]
- **Shape:**
  ```typescript
  // or Go, or CSS — whatever the interface language is
  interface EntityAPI {
    listEntities(route: string): Promise<DojoEntity[]>
    getEntity(id: string): Promise<DojoEntity>
  }
  ```
- **Constraints:** [Any limits on the interface — e.g., "Must serialize to JSON", "Case-sensitive names"]
- **Verification:** [How to verify the contract is met — e.g., "Track Y's API client type-checks against Track X's response"]
- **Owner:** Track [X] (responsible for backward compatibility)
```

## VI. Best Practices

- **Specs don't map 1:1 to tracks.** Build the mapping explicitly in Step 1.
- **Integration contracts are the most important output.** Prompts without contracts produce tracks that don't integrate.
- **Write the most-depended-on track's prompt first.** Its contracts shape everything downstream.
- **Cross-validation catches bugs before execution.** Type mismatches, response shape mismatches, and naming inconsistencies surface now, not during integration.
- **If a spec needs updating mid-stream,** do it in the prompt, not by rewriting the spec. Prompts are executable; specs are reference.
- **Each prompt must be self-contained.** An agent should execute it without reading other prompts. Integration contracts embedded in the prompt provide necessary context.

## VII. Quality Checklist

- [ ] All specs contributing to this release are mapped to tracks (Step 1)
- [ ] Every cross-track dependency has an explicit integration contract (Step 2)
- [ ] Shared types (DojoEntity, etc.) are consistent across all prompts that reference them (Step 5)
- [ ] Each prompt is self-contained and executable independently
- [ ] Prompts are ordered by dependency (foundation tracks first) (Step 4)
- [ ] Integration Contract Table is complete and unambiguous (Step 2)
- [ ] Codebase patterns extracted and grounded in prompts (Step 3)
- [ ] Master implementation plan ties all tracks together (Step 6)
- [ ] Success criteria span tracks, not just per-track (Step 6)
- [ ] Risk mitigation plan addresses integration dependencies

---

## Output

- One implementation prompt file per track (markdown), each self-contained and executable by an autonomous agent without reading sibling prompts
- A master implementation plan document listing all tracks in dependency order, the Integration Contract Table, and aggregate success criteria
- An Integration Contract Table naming every shared type, API, and design token along with producer track, consumer track, shape, and verification method

## Examples

**Scenario 1:** "We have four specs covering scaffold, backend, frontend, and dock. Turn them into prompts." → Four prompt files written in dependency order (scaffold first), with a shared DojoEntity type definition appearing identically in every prompt that references it, plus a master plan tying the execution sequence together.

**Scenario 2:** "The tracks shipped but didn't integrate cleanly — types were different." → Run this skill before commissioning: the cross-validation step (Step 5) would have caught the type mismatch between Track B's response shape and Track C's expected input before any agent wrote a line of code.

## Edge Cases

- When a spec feeds more than one track, explicitly list the mapping in Step 1 rather than assigning it to one track and hoping the other tracks infer the overlap
- When shared types evolve mid-commission (one track refines a shape), update the Integration Contract Table and re-issue all prompts that reference the changed type before other agents reach that section
- When only one spec exists, use `implementation-prompt` instead — this skill's value is coordination across multiple interconnected specs

## Anti-Patterns

- Writing all prompts simultaneously before defining integration contracts — prompts written without contracts will define contracts independently and produce incompatible interfaces
- Treating the spec-to-prompt mapping as 1:1 — one spec often feeds multiple tracks; skipping Step 1's matrix causes gaps and duplicate work
- Embedding cross-track context inline in each prompt instead of referencing contracts — inline duplication drifts; the contract table is the single source of truth
