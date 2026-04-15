---
name: voice-before-structure
model: sonnet
description: "Read the project's design language and philosophy before writing structural artifacts. Ground manifests, READMEs, and descriptions in the project's voice, not boilerplate. Use when creating manifests or READMEs for established projects, writing ecosystem-level documentation, building plugin descriptions or marketplace metadata, or authoring metadata that carries a project's identity. Trigger phrases: 'write the manifest', 'create a README', 'describe this plugin', 'ground yourself in the philosophy', 'ecosystem-level work'."
category: wisdom-garden
triggers:
  - "write the manifest"
  - "create a README"
  - "describe this plugin"
  - "ground yourself in the philosophy"
  - "ecosystem-level work"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: project_context
    type: string
    description: Description of the project whose design language and philosophy to read before writing
    required: true
  - name: artifact_type
    type: string
    description: Type of artifact to produce — manifest, README, description, or metadata
    required: false
outputs:
  - name: voiced_artifact
    type: string
    description: Project-voice-grounded artifact (manifest, README, description, or metadata) written in the established design language
---
# Voice Before Structure

## Philosophy

Structure without voice is generic cargo. A manifest read aloud should unmistakably belong to *this* project, not any project. The voice — the project's design language, founding tensions, philosophy — informs the naming, descriptions, and framing of everything you build.

When you write structural artifacts (manifests, READMEs, descriptions) in a vacuum, you get rewrite cycles. When you ground them in the philosophy first, they land. The difference between "strategic planning tools" and "begin with a tension, not a solution" is the difference between cargo and cargo with identity.

## When to Use

- **Starting ecosystem-level work**: plugin builds, marketplace setup, documentation overhauls
- **Creating identity-bearing documents**: manifests, READMEs, descriptions, schemas
- **Writing for established projects**: ones with recognizable design language or founding philosophy
- **Metadata authoring**: anything that carries the project's identity outward

Avoid using this when working on utility code or implementation details that don't need to express the project's voice.

## Workflow

### Before Writing Structure

1. **Identify philosophy documents** in the project (README, design language docs, founding reflections, core principles, vision statements)

2. **Read 2-3 documents** — enough to *hear* the voice, not so many that you stall execution
   - What tensions does this project address?
   - What language does it use repeatedly?
   - What does it refuse to do?
   - What's the underlying philosophy?

3. **Absorb the voice** — not just facts, but *how* the project thinks
   - Naming conventions and style
   - Metaphors it uses
   - Values it prioritizes
   - Problems it frames differently than standard industry language

4. **Draft the structural artifact** with voice as a constraint, not an afterthought
   - Use the project's language for concepts
   - Match the tone and cadence
   - Reference the founding tensions or philosophy where relevant

5. **Read aloud** — the quality check
   - Does it sound like it belongs to this project?
   - Could it belong to any project (generic = failed)?
   - Would someone familiar with the philosophy recognize these descriptions?

## Best Practices

- **Read before writing**, not while writing. You need the voice internalized, not consulted.
- **Read selectively**. 2-3 documents is the sweet spot — enough context, not analysis paralysis.
- **Listen for what's unique**, not just what's stated. Tone, metaphors, and refusals reveal philosophy.
- **Check specificity**. If your descriptions could apply to any project, you haven't grounded them yet.
- **Test for recognition**. Imagine someone who built this project reading what you wrote. Would they nod?
- **Use concrete language** from the project. Borrow the vocabulary; don't invent parallel terminology.

## Quality Checklist

- [ ] Philosophy documents identified and read (at least 2-3)
- [ ] Project's core tensions or design principles understood
- [ ] Artifact uses language/metaphors from philosophy docs
- [ ] Descriptions are specific to *this* project, not generic
- [ ] Read-aloud test: sounds like it belongs here
- [ ] Someone familiar with the project would recognize it
- [ ] Tone and cadence match the project's established voice

## Common Pitfalls

**Generic boilerplate**: Writing a description that could apply to any project. Test: remove the project name. Does it still make sense? If yes, it's too generic.

**Over-reading**: Spending hours absorbing philosophy instead of drafting. 2-3 documents, then write. Iteration refines the voice further.

**Mismatching tone**: Writing formally when the project is conversational, or vice versa. Listen for how the philosophy docs talk, not just what they say.

**Forcing the voice**: Trying to sound like the project when you don't understand it yet. Return to the philosophy docs; let understanding come first.

**Treating structure as separate**: Writing the manifest first, then trying to add voice afterward. Voice must inform structure from the start.

## Related Skills

- **memory-garden**: Preserves the philosophy and design language that this skill reads
- **seed-extraction**: Extracts patterns that become part of the project's voice
- **specification-writer**: Benefits from voice grounding when writing identity-bearing sections
- **compression-ritual**: Preserves the "why" that feeds future voice grounding

---

## Output

- The identity-bearing artifact itself (manifest, README, plugin description, marketplace metadata) written in the project's established voice
- A brief voice-grounding note (1-2 sentences in-line or as a comment) citing which philosophy documents were read and what language was adopted

## Examples

**Scenario 1:** User says "write the manifest for this plugin" on a project that has a PHILOSOPHY.md → reads PHILOSOPHY.md and one other founding doc, then writes `plugin.json` descriptions that use the project's own vocabulary and framing rather than generic boilerplate.

**Scenario 2:** User says "describe this plugin for the marketplace" → reads 2-3 design language docs, identifies core tensions and metaphors, drafts the description, and passes the read-aloud test before finalizing.

## Edge Cases

- If no philosophy documents exist in the project, ask the user for 2-3 sentences describing the project's design philosophy before drafting. Do not invent the voice.
- If the project has conflicting philosophy documents (e.g., an old VISION.md and a newer README with different framing), surface the conflict to the user before choosing which to ground in.

## Anti-Patterns

- Writing the manifest first and then "adding voice" afterward — voice must inform structure from the start, not be layered on top.
- Reading too many documents before drafting — more than 3 documents causes analysis paralysis. 2-3 is the ceiling; start drafting.

---

**Origin**: Marketplace Build Sprint. Cruz's mid-sprint interrupt — "before you write the manifest make sure you ground yourself in the philosophy" — was the single highest-leverage moment of the sprint.
