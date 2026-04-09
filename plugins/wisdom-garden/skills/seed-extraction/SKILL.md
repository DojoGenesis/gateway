---
name: seed-extraction
model: opus
description: "Produces a seed file — a YAML-fronted markdown document capturing a reusable pattern with trigger, evidence, and application steps — from a conversation or project experience. Use when: 'extract the learnings', 'turn this into a seed', 'what can we learn from this', 'capture this insight', 'document this pattern'."
category: wisdom-garden
triggers:
  - "extract the learnings"
  - "turn this into a seed"
  - "what can we learn from this"
  - "capture this insight"
  - "document this pattern"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: experience
    type: string
    description: Description of the conversation, project, or experience to extract a reusable pattern from
    required: true
outputs:
  - name: seed_file
    type: ref
    format: cas-ref
    description: Seed file capturing a reusable pattern with trigger, evidence, and application steps
---

# Seed Reflector Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Extract, document, and apply reusable "seeds" (patterns, insights, principles) from experiences

---

## Overview

This skill encodes the practice of **seed extraction and reflection** — identifying reusable patterns from experiences and documenting them in a way that makes them easy to apply in future contexts.

**Philosophy:** Every experience contains seeds. The practice is learning to see them, extract them, and plant them where they'll grow.

---

## When to Use This Skill

- After completing a major project or release
- When you notice a pattern emerging across multiple experiences
- During memory maintenance (Tier A → Tier B compression)
- When preparing to share knowledge with other agents
- When you want to reflect on what you've learned

---

## What Is a "Seed"?

A **seed** is a reusable pattern, insight, or principle that:

1. **Emerged from experience** (not abstract theory)
2. **Can be applied in future contexts** (not one-time specific)
3. **Has a clear trigger** (you know when to use it)
4. **Captures wisdom** (not just information)

**Examples of Seeds:**
- "Three-Tiered Governance" (from Dataiku research)
- "Harness Trace" (traceability pattern)
- "Context Iceberg" (hierarchical context management)
- "3-Month Rule" (semantic compression heuristic)

---

## Seed Extraction Process

### Step 1: Identify Candidate Patterns

**Look for:**
- Decisions that worked well (or didn't)
- Patterns that emerged across multiple instances
- Insights that changed how you think
- Principles that guided successful outcomes
- Tensions or tradeoffs you navigated

**Questions to ask:**
- What did I learn that I didn't know before?
- What pattern did I notice repeating?
- What decision framework did I use?
- What would I do differently next time?
- What would I tell someone else facing this situation?

### Step 2: Test for Reusability

**A good seed is:**
- General enough to apply in multiple contexts
- Specific enough to be actionable
- Grounded in experience (not abstract)
- Has a clear trigger (you know when to apply it)

**A bad seed is:**
- Too specific ("Use Mermaid.js for diagrams in Dojo Genesis")
- Too vague ("Be thoughtful")
- Not grounded ("I think this might work")
- No trigger ("Apply this... sometime?")

### Step 3: Document the Seed

Use the seed template at [references/seed-template.md](references/seed-template.md).

### Step 4: Test the Seed

**Apply it in a new context:**
- Does the trigger work? (Do you recognize when to use it?)
- Is it actionable? (Can you actually apply it?)
- Does it produce value? (Does it improve outcomes?)

**If yes:** Keep and refine  
**If no:** Revise or discard

---

## Seed Categories

### 1. Architectural Seeds
**Pattern:** Design decisions and system structures

**Examples:**
- Three-Tiered Governance
- Harness Trace
- Context Iceberg
- Agent Connect (routing-first, not swarm-first)

### 2. Process Seeds
**Pattern:** Workflows and methodologies

**Examples:**
- Planning with Files
- Backend-First, Chunked Development
- Dual-Track Orchestration
- Compression Cycle (every 3-7 days)

### 3. Decision Seeds
**Pattern:** Frameworks for making choices

**Examples:**
- 3-Month Rule (semantic compression)
- Cost Guard (token budget management)
- Safety Switch (feature flags and rollback)

### 4. Wisdom Seeds
**Pattern:** Principles and philosophies

**Examples:**
- Beginner's Mind
- Knowing When to Shut Up
- Honesty is Wisdom

### 5. Technical Seeds
**Pattern:** Implementation patterns and best practices

**Examples:**
- Surgical Context (memory_search → memory_get)
- Graceful Degradation (resilience patterns)
- Semantic Compression (content-based, not positional)

---

## Seed Application Workflow

### 1. Recognize the Trigger

**Ask:** Does this situation match a seed's trigger?

**Check:**
- Context matches seed's "when to apply"
- Keywords or signals are present
- Problem pattern is similar to seed's origin

### 2. Retrieve the Seed

**Methods:**
- Search seed library by keyword
- Browse category (architectural, process, decision, wisdom, technical)
- Recall from memory (if seed is well-practiced)

### 3. Apply the Seed

**Follow:**
- Read "How to Apply" steps
- Adapt to current context
- Check "Cautions" to avoid misapplication

### 4. Reflect on Outcome

**Document:**
- Did the seed work? (Yes/No/Partially)
- What was the outcome?
- What would you adjust?
- Should the seed be refined?

**Update:**
- Increment usage count
- Add new example (if successful)
- Refine "How to Apply" (if needed)
- Update success rate

---

## Quality Checklist

Before finalizing a seed, verify:

### Clarity
- [ ] Name is memorable and descriptive
- [ ] Pattern is stated in one clear sentence
- [ ] Origin is documented
- [ ] Why it matters is explicit

### Reusability
- [ ] Trigger is specific and recognizable
- [ ] "How to Apply" steps are actionable
- [ ] Examples demonstrate the pattern
- [ ] Related seeds are identified

### Grounding
- [ ] Emerged from real experience (not theory)
- [ ] Evidence includes 3+ instances
- [ ] Examples are concrete (not abstract)
- [ ] Cautions address misapplication

### Metadata
- [ ] Created date is recorded
- [ ] Usage count is tracked
- [ ] Status is set (Active/Experimental/Deprecated)
- [ ] Category is assigned

---

## Examples of Seeds

### From Dojo Genesis

**Seed: Three-Tiered Governance**
- **Pattern:** Governance multiplies velocity by providing clear decision frameworks at strategic, tactical, and operational levels
- **Trigger:** When building complex systems that need both flexibility and control
- **Origin:** Dataiku research synthesis

**Seed: 3-Month Rule**
- **Pattern:** If it wouldn't matter in 3 months → compress or discard
- **Trigger:** When compressing memory or deciding what to keep
- **Origin:** Cipher's feedback on semantic compression

---

## Common Pitfalls to Avoid

- **Hoarding Seeds:** Keeping every insight — curate ruthlessly
- **Vague Patterns:** "Be thoughtful" — write "Apply 3-month rule when compressing"
- **No Trigger:** Seed without context — include clear "when to apply"
- **Not Testing:** Extract and forget — apply, reflect, refine
- **Over-Abstracting:** Theory without grounding — use concrete examples from experience

---

## Output

- A seed file saved to the project's `seeds/` directory (or `memory/seeds/` if no dedicated location exists)
- File format: Markdown with YAML frontmatter (`seed_id`, `name`, `category`, `status`, `created`)
- File name convention: `[category-prefix]_[short-name].md` (e.g., `03_context_iceberg.md`)

---

## Examples

**Scenario 1:** User says "turn this into a seed — always dry-run new scripts before committing" → produces `process_dry_run_gate.md` with pattern, 2 trigger phrases, one concrete example from the session, and "Cautions" noting it does not apply to read-only scripts.

**Scenario 2:** User says "what can we learn from the three failed migrations?" → reads session context, identifies the common root cause as a missing pre-flight validation step, and produces `decision_preflight_validation.md` with evidence from all three instances.

---

## Edge Cases

- If the candidate pattern is too specific to one project (e.g., a config path), document it as a project note, not a seed — seeds must be transferable.
- If the user provides fewer than 2 concrete instances, mark the seed as `status: experimental` until more evidence accumulates.
- If a seed already exists in the library with the same core pattern, compare and merge rather than duplicating.

---

## Anti-Patterns

- Writing a seed from a single anecdote without checking if it generalizes — check against at least 2 instances before promoting to `active`.
- Creating a seed whose trigger phrase matches every situation — overly broad triggers make seeds useless for retrieval.

---

**Related Skills:**
- `specification-writer` - Seeds inform architectural decisions
- `memory-garden` - Seeds are extracted during memory compression
- `workspace-navigation` - Seeds are stored in shared workspace for collaboration

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
