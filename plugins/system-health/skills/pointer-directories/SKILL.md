---
name: pointer-directories
model: sonnet
description: "Produces a provenance determination — pointer or gap — for empty or apparently incomplete directories, with a documented rationale and recommended action. Use when: \"this directory is empty\", \"missing SKILL.md\", \"coverage metric shows a gap\", \"why is this directory here with nothing in it\", \"before creating content to fill an apparent void\"."
category: system-health
triggers:
  - "this directory is empty"
  - "missing SKILL.md"
  - "coverage metric shows a gap"
  - "why is this directory here with nothing in it"
  - "before creating content to fill an apparent void"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: directory_path
    type: string
    description: Path to the empty or apparently incomplete directory to evaluate
    required: true
outputs:
  - name: provenance_determination
    type: string
    description: Provenance determination — pointer or gap — with documented rationale and recommended action
---
# Pointer Directories

## Philosophy

Empty directories are not failures of the system. They are signals. In well-designed registries, an empty directory often means: "The content you're looking for exists elsewhere by design." The instinct to fill every gap produces duplication and masks the actual provenance of content.

Before creating content, understand why something is empty. This distinction separates accidental incompleteness from intentional reference architecture.

## When to Use

- You encounter empty directories during a registry audit or reorganization
- A coverage metric shows "gaps" that nobody has reported as problems
- You find a skill, component, or module in a registry with no local content
- A project registry reports missing SKILL.md, README, or component files
- You're about to create content to fill an apparent void
- You're building changelog entries and need to account for "missing" content

## Workflow

### 1. Identify the Empty Directory

Locate the directory that appears incomplete. Note its location and expected content type (skill, component, configuration, etc.).

### 2. Ask the Provenance Question

**"Is this directory a pointer or a gap?"**

A pointer is intentional. A gap is accidental. You need to know which one before proceeding.

### 3. Check Three Levels

Search for the content at other levels:

- **Source Level**: Original authoring location (e.g., dojo-genesis/skills/)
- **System-Installed Level**: Deployed to runtime (e.g., .skills/skills/, system packages)
- **Platform Level**: Provided by host platform (e.g., Cowork system skills, external dependencies)

### 4. Determine Status

- **Found at another level?** → It's a pointer. Document the provenance.
- **Found nowhere?** → It's a gap. Create content or mark for later.
- **Unclear?** → Ask the team or check the changelog.

### 5. Document Your Finding

If it's a pointer:

- Record which level holds the authoritative content
- Add an entry to the changelog explaining the provenance
- Consider whether a symlink, reference, or comment in the empty directory would help future maintainers

If it's a gap:

- Decide whether to create content now or defer
- If deferring, add a note explaining why the gap exists

## Best Practices

**Verify before Creating**: Always check multiple locations before assuming content is missing. Empty directories are often intentional.

**Copy from Authoritative Source**: If content exists elsewhere, copy from the authoritative source rather than recreating it. This preserves accuracy and reduces drift.

**Document Provenance**: When a directory is a pointer, make that relationship visible. Use comments, changelogs, or README entries to explain why the directory is empty and where content lives.

**Treat Coverage Metrics as Signals, Not Rules**: A coverage gap may reflect reality or reflect the structure of your registry. Investigate before accepting it as a problem.

**Ask Before Filling**: When you encounter an empty directory, the first question is "Why is this empty?" not "What content should go here?"

## Quality Checklist

- [ ] Searched at least two other levels (source, system, platform) before concluding content is missing
- [ ] Can articulate why this directory is empty (if empty by design)
- [ ] If content exists elsewhere, identified the authoritative source
- [ ] Checked the changelog or team documentation for context
- [ ] If creating content, confirmed it's because a gap exists, not because a pointer looks incomplete
- [ ] Documented the provenance decision for future maintainers

## Common Pitfalls

**Mistaking pointers for gaps**: You see an empty skill directory and assume someone forgot to write the skill. Investigate first.

**Creating duplicate content**: You write a new SKILL.md for a skill that already exists at the system level. Check system-installed locations.

**Ignoring the "nobody complained" signal**: A gap has existed for months and nobody has filed an issue. This often means it's not actually a gap.

**Treating all empty directories the same**: Some empty directories are intentional pointers; others are true gaps. Each one needs investigation.

**Skipping the changelog**: The history of why something is empty often lives in the changelog. Check there before deciding.

## Related Skills

- **health-audit**: Surface-level audits may flag pointers as gaps; this skill provides deeper investigation
- **documentation-audit**: Documentation gaps and pointer directories follow the same investigation pattern
- **skill-audit-upgrade**: When auditing skills, pointer directories need investigation not automatic content creation
- **repo-context-sync**: Understanding registry architecture helps distinguish pointers from gaps

## Output
- A written provenance determination: "pointer" or "gap", which level holds the authoritative content (source / system-installed / platform), and the recommended action (document, symlink, create, or defer).
- If a pointer: a changelog entry explaining provenance for future maintainers.
- If a gap: a decision on whether to create content now or defer with a documented reason.

## Examples
**Scenario 1:** "The `hooks-reference` directory in the registry is empty — should I write content?" → Investigation finds the skill exists at the system-installed level (.skills/skills/hooks-reference/SKILL.md). Provenance documented: pointer, not a gap. No content created.
**Scenario 2:** "Coverage audit shows `semantic-clusters` skill missing from the plugin" → Investigation finds no content at any level (source, system, platform). Confirmed gap. Content creation initiated.

## Edge Cases
- A directory that has existed for months without anyone filing an issue is probably a pointer — the "nobody complained" signal is evidence, not proof. Verify at all three levels before concluding.
- If the content is found at multiple levels with different versions, flag this as a divergence requiring reconciliation rather than treating either as authoritative.

## Anti-Patterns
- Creating a new SKILL.md for content that already exists at the system-installed level — produces duplicate content that will diverge from the authoritative source over time.
- Treating all coverage metrics as actionable gaps without investigation — coverage gaps may reflect the registry structure, not missing content.
