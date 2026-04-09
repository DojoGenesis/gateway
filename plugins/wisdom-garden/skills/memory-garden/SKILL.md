---
name: memory-garden
model: sonnet
description: "Writes a structured memory entry to the garden — daily note (Tier A), curated wisdom update (Tier B), or monthly compressed archive (Tier C) — based on conversation insights. Use when: 'remember this pattern', 'save this insight', 'add to memory garden', 'write a memory entry', 'record this learning'."
category: wisdom-garden
triggers:
  - "remember this pattern"
  - "save this insight"
  - "add to memory garden"
  - "write a memory entry"
  - "record this learning"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: insight
    type: string
    description: The insight, pattern, or learning to write as a memory entry
    required: true
  - name: tier
    type: string
    description: Memory tier — A (daily note), B (curated wisdom), or C (monthly compressed archive)
    required: false
outputs:
  - name: memory_entry
    type: ref
    format: cas-ref
    description: Structured memory entry written to the garden at the appropriate tier
---

# Memory Garden Writer Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Write structured, semantically rich memory entries for efficient context management

---

## Overview

This skill encodes the pattern for writing memory entries that follow the **4-Tier Context Iceberg** and **Hierarchical Memory** principles. Use this skill to create memory entries that are easy to search, retrieve, and compress.

**Philosophy:** Memory should be a garden, not a landfill. Cultivate what matters, compost what doesn't.

---

## When to Use This Skill

- Creating daily memory notes after a session
- Writing compressed summaries of conversations
- Extracting "seeds" (reusable insights) from experiences
- Documenting decisions and their rationale
- Maintaining the memory hierarchy (Tier A → Tier B → Tier C)

---

## Memory Hierarchy

### Tier A: Raw Daily Notes
- **Location:** `memory/YYYY-MM-DD.md`
- **Purpose:** Capture everything from today's session
- **Lifespan:** 1-3 days before compression
- **Format:** Timestamped entries with context

### Tier B: Curated Wisdom
- **Location:** `MEMORY.md` (root level)
- **Purpose:** Distilled insights, decisions, patterns
- **Lifespan:** Permanent, but evolves
- **Format:** Structured sections with triggers

### Tier C: Compressed Archive
- **Location:** `memory/archive/YYYY-MM.md`
- **Purpose:** Historical record, rarely accessed
- **Lifespan:** Permanent, read-only
- **Format:** Semantic summaries

See [references/memory-templates.md](references/memory-templates.md) for the full templates for each tier.

---

## The "3-Month Rule"

**Rule:** If it wouldn't matter in 3 months → compress or discard.

**Keep:**
- Decisions and their rationale
- Lessons learned and patterns discovered
- Seeds (reusable insights)
- Significant events and outcomes

**Compress:**
- Routine activities ("worked on X")
- Pleasantries and confirmations
- Detailed step-by-step logs (keep summary only)

**Discard:**
- Duplicate information
- Temporary notes that were resolved
- Irrelevant tangents

---

## Semantic Compression Guidelines

### What to Keep (Verbatim)

1. **Decisions:** The choice, rationale, and context
2. **Insights:** Novel patterns or principles
3. **Seeds:** Reusable knowledge with triggers
4. **Failures:** What didn't work and why
5. **Breakthroughs:** Moments of clarity or innovation

### What to Summarize

1. **Activities:** "Worked on X, Y, Z" → "Implemented feature X"
2. **Discussions:** Long back-and-forth → Key points and outcome
3. **Research:** Detailed findings → Summary and conclusion
4. **Iterations:** Multiple attempts → Final approach and why

### What to Discard

1. **Pleasantries:** "Great work!" "Thank you!" (unless significant)
2. **Confirmations:** "Got it" "Understood" "Proceeding"
3. **Redundant logs:** Repeated information
4. **Resolved questions:** Questions that were answered and no longer relevant

---

## Memory Maintenance Cycle

**Every 3-7 Days:**

1. **Review Tier A (Daily Notes):**
   - Identify seeds to extract
   - Identify decisions to document
   - Identify patterns to record

2. **Update Tier B (Curated Wisdom):**
   - Add new seeds, decisions, patterns
   - Update existing entries if needed
   - Remove deprecated information

3. **Compress to Tier C (Archive):**
   - Apply 3-month rule
   - Create semantic summary
   - Move to archive folder

4. **Prune:**
   - Delete raw daily notes older than 7 days (after compression)
   - Keep only what matters

---

## Quality Checklist

Before finalizing a memory entry, verify:

### Daily Notes (Tier A)
- [ ] Timestamped entries with context
- [ ] Clear "What, Why, Outcome" structure
- [ ] Insights and learnings captured
- [ ] Decisions documented with rationale
- [ ] Seeds extracted with triggers
- [ ] Tags and metadata included

### Curated Memory (Tier B)
- [ ] Principles are clear and actionable
- [ ] Decisions include context and rationale
- [ ] Patterns have evidence (3+ instances)
- [ ] Seeds have clear triggers
- [ ] Compression history is updated
- [ ] Maintenance date is set

### Compressed Archive (Tier C)
- [ ] Summary captures key events
- [ ] Decisions and lessons are preserved
- [ ] Seeds are extracted
- [ ] Compression ratio is calculated
- [ ] Original files are deleted after compression

---

## Common Pitfalls to Avoid

- **Hoarding Everything:** Keeping every detail — compress ruthlessly
- **Vague Insights:** "This was useful" — write "This pattern applies when X"
- **Missing Triggers:** Seed without context — include clear "when to apply"
- **No Maintenance:** Letting Tier A grow forever — compress every 3-7 days
- **Duplicate Information:** Same thing in multiple places — single source of truth

---

## Output

- **Tier A:** `memory/YYYY-MM-DD.md` — new or appended daily note with timestamped sections
- **Tier B:** `MEMORY.md` — updated curated wisdom entry (principle, decision, or seed block)
- **Tier C:** `memory/archive/YYYY-MM.md` — new monthly archive file with compression log

---

## Examples

**Scenario 1:** User says "remember this pattern — always read before editing" → writes a Tier A seed entry under today's date with pattern, trigger, and one concrete example from the session.

**Scenario 2:** User says "compress last week's notes into memory" → reads all Tier A files from the week, applies 3-month rule, writes a Tier B update to MEMORY.md with seeds and decisions retained, and marks Tier A files for deletion.

---

## Edge Cases

- If no tier is specified, default to Tier A (daily note) and state the assumption.
- If the user asks to "save" something that is already in MEMORY.md verbatim, note the duplicate and skip.
- If Tier A is more than 7 days old without compression, prompt the user before writing more raw notes.

---

## Anti-Patterns

- Writing a Tier B entry directly from raw conversation without distillation — always compress first.
- Adding a seed without a trigger phrase — a seed without "when to apply" has no retrieval value.
- Using this skill to store code or file content verbatim — memory entries hold insights and decisions, not raw code.

---

**Related Skills:**
- `specification-writer` - For documenting technical decisions
- `seed-extraction` - For extracting reusable insights
- `workspace-navigation` - For managing memory files efficiently

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
