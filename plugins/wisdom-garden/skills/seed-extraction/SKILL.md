# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: seed-extraction
description: Extract wisdom seeds from conversations and experiences
triggers:
  - "extract wisdom seeds"
  - "capture knowledge nuggets"
  - "identify key insights"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - script_execution
    - file_system
    - bash
  portable: true
  tier: 2
  agents:
    - research-agent
---

# Seed Reflector Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Extract, document, and apply reusable "seeds" (patterns, insights, principles) from experiences

---

## Overview

This skill encodes the practice of **seed extraction and reflection** — identifying reusable patterns from experiences and documenting them in a way that makes them easy to apply in future contexts. Inspired by the Dojo Seed Patches and Cipher's practice of reflection.

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
- "Knowing When to Shut Up" (restraint as wisdom)

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
- ✅ **General enough** to apply in multiple contexts
- ✅ **Specific enough** to be actionable
- ✅ **Grounded in experience** (not abstract)
- ✅ **Has a clear trigger** (you know when to apply it)

**A bad seed is:**
- ❌ Too specific ("Use Mermaid.js for diagrams in Dojo Genesis")
- ❌ Too vague ("Be thoughtful")
- ❌ Not grounded ("I think this might work")
- ❌ No trigger ("Apply this... sometime?")

### Step 3: Document the Seed

Use the **Seed Template** (see below)

### Step 4: Test the Seed

**Apply it in a new context:**
- Does the trigger work? (Do you recognize when to use it?)
- Is it actionable? (Can you actually apply it?)
- Does it produce value? (Does it improve outcomes?)

**If yes:** Keep and refine  
**If no:** Revise or discard

---

## Seed Template

```markdown
## Seed: [Name]

**Pattern:** [One-sentence description of the reusable insight]

**Origin:** [Where this came from - project, experience, date]

**Why It Matters:** [The value or benefit of applying this seed]

**Trigger:** [When to apply this seed]
- [Context or situation 1]
- [Context or situation 2]
- [Keywords or signals that indicate this seed is relevant]

**How to Apply:**
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Example (From Origin):**
[Concrete example from the experience where this seed emerged]

**Example (Applied):**
[Concrete example of applying this seed in a new context]

**Related Seeds:**
- [Seed that complements this one]
- [Seed that contrasts with this one]

**Cautions:**
- [When NOT to apply this seed]
- [Common misapplications]

**Evidence:**
- [Instance 1: date, context, outcome]
- [Instance 2: date, context, outcome]
- [Instance 3: date, context, outcome]

**Metadata:**
- **Created:** YYYY-MM-DD
- **Last Applied:** YYYY-MM-DD
- **Usage Count:** [Number]
- **Success Rate:** [X]% (if measurable)
- **Status:** Active | Experimental | Deprecated
```

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
- Understanding is Love
- Knowing When to Shut Up
- Honesty is Wisdom

### 5. Technical Seeds
**Pattern:** Implementation patterns and best practices

**Examples:**
- Surgical Context (memory_search → memory_get)
- Graceful Degradation (resilience patterns)
- Semantic Compression (content-based, not positional)

---

## Reflection Practice

### Daily Reflection (5-10 minutes)

**Questions:**
1. What worked well today?
2. What didn't work as expected?
3. What pattern did I notice?
4. What would I do differently?
5. Is there a seed here?

**Output:** 1-2 candidate seeds for deeper reflection

### Weekly Reflection (20-30 minutes)

**Questions:**
1. What patterns emerged across this week?
2. Which candidate seeds are actually reusable?
3. Which seeds did I apply this week?
4. Which seeds need refinement?
5. Which seeds should be deprecated?

**Output:** Refined seed library, updated usage counts

### Monthly Reflection (1-2 hours)

**Questions:**
1. Which seeds have proven most valuable?
2. Which seeds have I stopped using?
3. What new categories of seeds are emerging?
4. How has my seed library evolved?
5. What seeds should I share with others?

**Output:** Curated seed collection, reflection document

---

## Seed Library Structure

```
seeds/
├── README.md (Overview and index)
├── architectural/
│   ├── three-tiered-governance.md
│   ├── harness-trace.md
│   └── context-iceberg.md
├── process/
│   ├── planning-with-files.md
│   ├── dual-track-orchestration.md
│   └── compression-cycle.md
├── decision/
│   ├── 3-month-rule.md
│   ├── cost-guard.md
│   └── safety-switch.md
├── wisdom/
│   ├── beginners-mind.md
│   ├── understanding-is-love.md
│   └── knowing-when-to-shut-up.md
└── technical/
    ├── surgical-context.md
    ├── graceful-degradation.md
    └── semantic-compression.md
```

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
- **Origin:** Dataiku research synthesis (v0.0.17)

**Seed: 3-Month Rule**
- **Pattern:** If it wouldn't matter in 3 months → compress or discard
- **Trigger:** When compressing memory or deciding what to keep
- **Origin:** Cipher's feedback on semantic compression (v0.0.19)

**Seed: Knowing When to Shut Up**
- **Pattern:** Power without judgment is dangerous; restraint is wisdom
- **Trigger:** When building proactive or adaptive systems
- **Origin:** Cipher's v0.0.20 specification (Judgment Layer)

### From Agent Collaboration

**Seed: Honesty is Wisdom**
- **Pattern:** Acknowledge gaps openly; it builds trust and enables mutual learning
- **Trigger:** When reviewing work, sharing specifications, or collaborating with other agents
- **Origin:** Manus-Cipher exchange (v0.0.19-v0.0.23)

**Seed: Mutual Evolution**
- **Pattern:** Collaboration isn't one-way teaching; it's mutual recognition of what's possible
- **Trigger:** When working with other agents or receiving feedback
- **Origin:** Lineage transmission practice (Manus-Cipher)

---

## Common Pitfalls to Avoid

❌ **Hoarding Seeds:** Keeping every insight → ✅ Curate ruthlessly  
❌ **Vague Patterns:** "Be thoughtful" → ✅ "Apply 3-month rule when compressing"  
❌ **No Trigger:** Seed without context → ✅ Clear "when to apply"  
❌ **Not Testing:** Extract and forget → ✅ Apply, reflect, refine  
❌ **Over-Abstracting:** Theory without grounding → ✅ Concrete examples from experience

---

## Usage Instructions

1. **Read this skill** before extracting seeds
2. **Identify candidate patterns** from recent experiences
3. **Test for reusability** (general enough, specific enough, grounded, has trigger)
4. **Document using the template**
5. **Apply in a new context** to test
6. **Reflect and refine** based on outcomes
7. **Share with others** when seeds prove valuable

---

## Skill Metadata

**Token Savings:** ~3,000-5,000 tokens per session (offload learnings to structured seeds instead of re-explaining)  
**Quality Impact:** Ensures reusable patterns are captured and applied consistently  
**Maintenance:** Update when new seeds emerge or existing seeds are refined  

**Related Skills:**
- `specification-writer` - Seeds inform architectural decisions
- `memory-garden` - Seeds are extracted during memory compression
- `workspace-navigation` - Seeds are stored in shared workspace for collaboration

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
