# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: skill-maintenance
description: Update existing skills to maintain quality standards and relevance
triggers:
  - "update skill documentation"
  - "maintain skill quality"
  - "refresh skill content"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
---

# Skill Maintenance Ritual

---

## I. The Philosophy: Maintenance is Not Just Fixing—It's Keeping What Works Working Well

A skills directory is a living knowledge base. As the project evolves, skill names may become unclear, terminology may need updating, and cross-references may drift out of sync. Without systematic maintenance, the knowledge base degrades: skills become hard to find, references break, and terminology becomes inconsistent.

This skill transforms maintenance from a reactive chore into a proactive ritual. By following a structured process (read, propose, execute, verify, document), we ensure the skills directory remains clear, consistent, and valuable.

**Core Insight:** Good maintenance prevents future problems. A well-maintained knowledge base is easier to search, easier to understand, and easier to extend.

---

## II. When to Use This Skill

Use this skill when:

- **Skill names become unclear or outdated** - Names no longer describe what the skill does
- **Terminology needs updating** - Tool-specific language should be generalized (e.g., "zenflow" → "implementation")
- **Adding new skills** - New skills reference existing skills and need validation
- **Deprecating or merging skills** - Old skills need to be removed or consolidated
- **Conducting periodic audits** - Regular health checks of the skills directory
- **Onboarding new contributors** - Ensuring the knowledge base is accessible and consistent

---

## III. The Workflow

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

---

## IV. Best Practices

### **Pause and Clarify Scope**

Before any large refactor, pause and clarify with the user:
- What exactly needs to change?
- What should stay the same?
- What's the scope (skills directory only, or entire repository)?

### **Read Before Proposing**

Never suggest changes without reading the actual content. Understanding reality prevents proposing changes that don't make sense.

### **Use Systematic Search**

Don't rely on memory to find all references. Use `grep` to systematically search and catalog all instances.

### **Preserve Contextually Appropriate References**

Not all references should be changed. Preserve tool-specific mentions when they're contextually appropriate (e.g., when listing multiple tools).

### **Batch Edits for Efficiency**

Use the `file` tool's multiple edits feature with `all: true` to replace all occurrences in a single operation.

### **Comprehensive Commit Messages**

Write commit messages that explain:
- **What** changed (file-by-file breakdown)
- **Why** it changed (rationale)
- **What** was preserved (and why)

### **Document Immediately**

Create summary documents right after completing the maintenance. Details fade quickly, and future maintainers will thank you.

---

## V. Quality Checklist

Before considering the maintenance complete, ensure you can answer "yes" to all of the following:

- [ ] Have you clarified the scope with the user?
- [ ] Have you read all affected skills to understand their purpose?
- [ ] Have you proposed changes and received user confirmation?
- [ ] Have you renamed directories and updated internal metadata?
- [ ] Have you cataloged all references that need updating?
- [ ] Have you determined a refactoring strategy (what changes, what stays)?
- [ ] Have you executed the refactor with batch edits?
- [ ] Have you verified no unintended references remain?
- [ ] Have you committed with a comprehensive commit message?
- [ ] Have you pushed changes to the remote repository?
- [ ] Have you created summary documentation?

---

## VI. Common Pitfalls to Avoid

❌ **Over-refactoring:** Changing references that should stay (e.g., historical docs, tool-specific mentions)  
✅ **Scope appropriately:** Only change what needs changing

❌ **Proposing without reading:** Suggesting changes based on assumptions  
✅ **Read first:** Understand reality before proposing

❌ **Missing references:** Relying on memory instead of systematic search  
✅ **Use grep:** Catalog all references systematically

❌ **Vague commit messages:** "Updated skills" without explanation  
✅ **Be comprehensive:** Explain what, why, and what was preserved

❌ **No documentation:** Completing maintenance without recording the process  
✅ **Document immediately:** Create summary docs while details are fresh

---

## VII. Example: Skill Rename and Terminology Refactor (Feb 7, 2026)

**Context:** After creating 5 new skills, we needed to rename 4 existing skills and refactor "zenflow" to "implementation" across 32 references.

**Process:**
1. User requested renames: "let's rename our specification and prompt skills to simply describe what they do"
2. Read all 4 skills to understand their purpose
3. Proposed renames following "verb-object" pattern
4. User refined names (e.g., "write-zenflow-prompt" → "write-implementation-prompt")
5. Renamed directories and updated metadata
6. User requested terminology refactor: "complete this knowledge refactor to rename everything else with zenflow in it to implementation"
7. Searched and found 530 total "zenflow" references
8. User scoped down: "skills directory only" (32 references)
9. Read context and determined strategy (what to change, what to preserve)
10. Executed refactor with batch edits (22 + 7 + 2 + 1 replacements)
11. Verified and committed with comprehensive message
12. Created summary documentation

**Outcome:**
- 4 skills renamed with clear, descriptive names
- 32 references updated to generic "implementation" terminology
- 6 tool-specific "Zenflow" references preserved
- All cross-references updated
- Comprehensive documentation created

**Time:** ~2 hours total

---

## VIII. Related Skills

- **`compression-ritual`** - For preserving insights before large refactors
- **`process-extraction`** - For creating new skills from maintenance processes
- **`health-audit`** - For comprehensive repository health audits
- **`documentation-audit`** - For identifying documentation drift

---

## IX. Maintenance Schedule Recommendation

**Suggested frequency:**
- **Quarterly:** Full audit of skills directory
- **After major additions:** When 5+ new skills are added
- **On user request:** When terminology or naming issues are identified
- **Before major releases:** Ensure consistency before public releases

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active
