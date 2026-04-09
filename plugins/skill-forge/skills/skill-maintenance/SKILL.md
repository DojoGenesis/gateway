---
name: skill-maintenance
model: sonnet
description: "Produces a maintained skills directory with updated names, unified terminology, repaired cross-references, and a git commit documenting every change and what was preserved. Use when: 'update these skill names', 'refactor the skills directory', 'clean up skill references', 'rename this skill', 'audit the skills ecosystem', 'deprecate this skill', 'add version history to a skill'."
category: skill-forge
triggers:
  - "update these skill names"
  - "refactor the skills directory"
  - "clean up skill references"
  - "rename this skill"
  - "audit the skills ecosystem"
  - "deprecate this skill"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"
  - "bash"

inputs:
  - name: skills_directory
    type: string
    description: Path to the skills directory to maintain and audit
    required: true
outputs:
  - name: maintenance_report
    type: string
    description: Maintained skills directory with updated names, unified terminology, repaired cross-references, and git commit documentation
---

# Skill Maintenance Ritual

## I. The Workflow

This is a 9-step workflow for maintaining the skills directory.

---

### **Step 1: Recognize the Need for Maintenance**

**Goal:** Identify when skill maintenance is needed.

**Triggers:**
- User requests skill renames or refactors
- You notice unclear or inconsistent skill names
- New skills are added that reference outdated names
- Terminology changes in the broader ecosystem (e.g., tool names, industry standards)
- Periodic audit schedule (e.g., quarterly)

**Actions:**
1. Pause and clarify the scope with the user
2. Ask: What specifically needs to change?
3. Ask: What should stay the same?
4. Document the maintenance goal clearly

**Output:** Clear understanding of maintenance scope

**Key Insight:** Always pause and clarify scope before large refactors. Avoid over-refactoring by understanding what actually needs to change.

---

### **Step 2: Read Before Proposing**

**Goal:** Understand what the skills actually do before suggesting changes.

**Actions:**
1. Read all skills that will be affected by the maintenance
2. Understand the actual purpose and workflow of each skill
3. Note any cross-references between skills
4. Identify patterns in naming or terminology

**Tools:**
- `file` tool (read action) for each skill
- Take notes on what each skill does

**Output:** Deep understanding of affected skills

**Key Insight:** Never propose renames or refactors without reading the actual content first. Names should reflect reality, not assumptions.

---

### **Step 3: Propose Clear, Descriptive Changes**

**Goal:** Suggest changes that improve clarity and consistency.

**Actions:**
1. For renames: Propose names following the "verb-object" pattern
   - Examples: `release-specification`, `implementation-prompt`
2. For terminology refactors: Identify what should change and what should stay
3. Create a table showing old → new with rationale
4. Get user confirmation before proceeding

**Tools:**
- `message` tool (ask type) to propose and get feedback

**Output:** Agreed-upon changes

**Key Insight:** Good naming is good documentation. Descriptive names reduce cognitive load and make skills immediately understandable.

---

### **Step 4: Execute Renames Systematically**

**Goal:** Rename skill directories and update internal references.

**Actions:**
1. Rename skill directories using `mv` command
2. Update `name` field in each SKILL.md frontmatter
3. Update title (H1 heading) in each SKILL.md
4. Verify renames with `ls` command

**Tools:**
- `shell` tool for directory renames
- `file` tool (edit action) for content updates

**Output:** Renamed skills with updated metadata

**Key Insight:** Rename both the directory and the internal metadata. Inconsistency between directory name and skill name causes confusion.

---

### **Step 5: Search and Catalog References**

**Goal:** Find all instances of terminology or names that need updating.

**Actions:**
1. Use `grep -r -i "<term>"` to find all references
2. Count references by directory: `grep -r -i "<term>" | cut -d: -f1 | sed 's|^\./||' | cut -d/ -f1 | sort | uniq -c | sort -rn`
3. Identify which references need updating (skills directory) vs. which should stay (historical docs)
4. Confirm scope with user if needed

**Tools:**
- `shell` tool with grep for searching
- `wc -l` to count references
- `cut`, `sort`, `uniq -c` to categorize

**Output:** Catalog of references to update

**Key Insight:** Always catalog before refactoring. Understanding the scope prevents over-refactoring or missing references.

---

### **Step 6: Read Context and Determine Strategy**

**Goal:** Understand which references should change and which should stay.

**Actions:**
1. Use `match` tool (grep action) to view references with context
2. Analyze each usage to determine if it should be updated
3. Create a refactoring strategy:
   - What should be replaced?
   - What should be preserved?
   - What replacement text should be used?
4. Document the strategy

**Tools:**
- `match` tool (grep action) with leading/trailing context

**Output:** Refactoring strategy document

**Example Strategy:**
- Replace "Zenflow prompt" → "implementation prompt"
- Replace "Zenflow" (as executor) → "implementation agent"
- Preserve "Zenflow" when listing multiple tools: "Zenflow, Claude Code, etc."
- Preserve "Zenflow" in routing decisions: "Zenflow: Strategic implementation"

**Key Insight:** Not all references should be changed. Preserve tool-specific references when contextually appropriate.

---

### **Step 7: Execute Refactor with Batch Edits**

**Goal:** Update all references systematically using batch edits.

**Actions:**
1. For each affected skill file:
   - Create a list of find/replace pairs
   - Use `file` tool (edit action) with multiple edits
   - Set `all: true` to replace all occurrences
2. Verify changes with `grep` after each file
3. Keep a count of replacements per file

**Tools:**
- `file` tool (edit action) with multiple edits
- `shell` tool with grep to verify

**Output:** Updated skill files

**Example Edit:**
```json
{
  "edits": [
    {"all": true, "find": "Zenflow prompt", "replace": "implementation prompt"},
    {"all": true, "find": "Zenflow", "replace": "implementation agent"}
  ]
}
```

**Key Insight:** Batch edits are more efficient than one-by-one replacements. Use `all: true` to replace all occurrences in a single operation.

---

### **Step 8: Verify and Commit**

**Goal:** Ensure all changes are correct and commit with comprehensive documentation.

**Actions:**
1. Verify no unintended references remain: `grep -i "<old term>" <directory>`
2. Check git status: `git status`
3. Stage changes: `git add skills/`
4. Write comprehensive commit message:
   - Summary of changes
   - File-by-file breakdown
   - Rationale for changes
   - What was preserved and why
5. Commit: `git commit -m "<message>"`
6. Push to remote: `git push origin main`

**Tools:**
- `shell` tool with git commands

**Output:** Committed and pushed changes

**Commit Message Template:**
```
<Action> in skills directory

<Summary paragraph>

Changes:
- <file1> (<N> replacements)
  - <change 1>
  - <change 2>
  
- <file2> (<N> replacements)
  - <change 1>

Kept <term> only when:
- <context 1>
- <context 2>

<Rationale paragraph>
```

**Key Insight:** Comprehensive commit messages are documentation. Future maintainers need to understand *why* changes were made, not just *what* changed.

---

### **Step 9: Document the Process**

**Goal:** Create summary documents for future reference.

**Actions:**
1. Create a summary document with:
   - Overview of changes
   - Refactoring strategy
   - Files modified
   - Benefits of the changes
   - Reflection on the process
2. Deliver summary to user with `message` tool

**Tools:**
- `file` tool (write action)
- `message` tool (result type)

**Output:** Documentation for future reference

**Key Insight:** Document the process immediately after completing it. Details fade quickly from memory.

## Output

- Renamed skill directories with updated `name` field and H1 heading in each SKILL.md
- All cross-references updated to the new names or terminology
- Zero stale references remaining (verified by grep)
- A git commit with a structured message: what changed, why, what was preserved
- Optional: a summary document delivered to the user if the scope was large

## Examples

**Scenario 1:** "Rename `zenflow-prompt` to `implementation-prompt` and update all references" → Read the skill, propose the rename following verb-object pattern, execute directory mv + metadata update, grep for all cross-references, batch-edit and verify, commit with per-file replacement counts.

**Scenario 2:** "Deprecate `old-pipeline` skill — it's been replaced by `batch-normalize-and-package`" → Add a `deprecated: true` field to frontmatter, add a deprecation notice at the top of the body pointing to the replacement, update any skills that link to it, commit.

## Edge Cases

- User requests a rename but the new name is already taken by another skill — surface the conflict, propose a resolution, and get confirmation before executing
- Terminology refactor touches historical documents (retrospectives, plans) outside the skills directory — catalog them but do not change them unless the user explicitly expands the scope
- A skill has no cross-references in the rest of the directory — rename is safe; note this in the commit message
- User wants to "clean up" a skill without changing its name — treat as in-place refactoring: read, propose specific changes, get confirmation, then edit

## Anti-Patterns

- **Over-refactoring:** Changing references that are contextually appropriate (e.g., tool-specific mentions in routing docs) because they happen to match the search term — always read the surrounding context before replacing
- **Proposing without reading:** Suggesting renames based on the directory name alone without reading what the skill actually does — names should reflect reality, not assumptions
- **Relying on memory for references:** Skipping grep and trusting recall to find all instances — always catalog systematically before refactoring
- **Vague commit messages:** Writing "updated skills" without a per-file breakdown — commit messages are the only audit trail for future maintainers
