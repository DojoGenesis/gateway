# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: workspace-navigation
description: Navigate multi-workspace environments and locate relevant project files
triggers:
  - "navigate workspace"
  - "find project files"
  - "locate workspace resources"
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

# Agent Workspace Navigator Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Efficiently navigate and interact with shared agent workspaces and discussion rooms

---

## Overview

This skill encodes best practices for working in **shared agent workspaces**—structured repositories where multiple agents collaborate, think together, and build shared context. It provides patterns for reading, writing, and organizing content in a way that maximizes clarity, minimizes token waste, and enables effective collaboration.

**Philosophy:** A shared workspace is not a file dump—it's a thinking room. Structure enables clarity; clarity enables collaboration.

---

## When to Use This Skill

- Working in a shared private repository with other agents
- Contributing to or reading from a collaborative discussion space
- Organizing thoughts, specifications, or research in a structured way
- Coordinating work across multiple agents or sessions
- Building shared context without unstructured file chaos

---

## Workspace Structure

A well-structured agent workspace follows the **Planning with Files** philosophy and organizes content by purpose, not by time or author.

### Standard Directory Structure

```
workspace/
├── README.md                    # Workspace overview and navigation guide
├── 00_Active/                   # Current work in progress
│   ├── discussions/             # Active discussions and threads
│   ├── drafts/                  # Work in progress (not final)
│   └── decisions/               # Decisions made (with rationale)
├── 01_Specifications/           # Finalized specs and designs
│   ├── architecture/            # System architecture docs
│   ├── features/                # Feature specifications
│   └── protocols/               # Protocols and standards
├── 02_Research/                 # Research findings and synthesis
│   ├── deep-dives/              # Deep research on specific topics
│   ├── landscape-scans/         # Wide research across domains
│   └── references/              # External sources and citations
├── 03_Memory/                   # Shared memory and knowledge base
│   ├── seeds/                   # Reusable patterns and insights
│   ├── learnings/               # Lessons learned
│   └── context/                 # Shared context for continuity
├── 04_Artifacts/                # Generated artifacts and outputs
│   ├── code/                    # Code examples or prototypes
│   ├── diagrams/                # Visual artifacts
│   └── documents/               # Final documents
└── 05_Archive/                  # Completed or deprecated work
    ├── by-date/                 # Time-based archive
    └── by-topic/                # Topic-based archive
```

### File Naming Conventions

**Format:** `YYYY-MM-DD_topic-name_author.md`

**Examples:**
- `2026-02-02_memory-garden-design_manus.md`
- `2026-02-02_context-compression-strategy_cipher.md`
- `2026-02-02_collaboration-protocol_manus-cipher.md`

**Why:**
- Date prefix enables chronological sorting
- Topic name provides context
- Author attribution enables tracking
- Underscore separators are filesystem-friendly

---

## Reading from the Workspace

### 1. Start with the Index

**Always read `README.md` first** to understand:
- Workspace purpose and scope
- Directory structure and conventions
- Active discussions and priorities
- How to contribute

### 2. Navigate by Purpose, Not Time

**Don't:**
- Browse chronologically through all files
- Read everything in a directory

**Do:**
- Identify your purpose (e.g., "understand memory architecture")
- Navigate to the relevant directory (e.g., `01_Specifications/architecture/`)
- Read the most recent or relevant file

### 3. Use Grep for Targeted Search

**Pattern:**
```bash
# Search for a specific topic across all markdown files
grep -r "memory compression" workspace/ --include="*.md"

# Search within a specific directory
grep -r "context window" workspace/01_Specifications/ --include="*.md"

# Search for discussions by a specific agent
grep -r "Author: Cipher" workspace/00_Active/discussions/ --include="*.md"
```

### 4. Read Metadata First

**Every document should have frontmatter:**

```markdown
---
title: Memory Garden Design
author: Manus
date: 2026-02-02
status: Draft | Active | Final | Archived
tags: [memory, architecture, compression]
related: [context-compression-strategy, seed-extraction]
---
```

**Read the metadata to decide:**
- Is this relevant to my current task?
- Is this the most recent version?
- Who authored this, and should I coordinate with them?
- What related documents should I read?

### 5. Extract Key Insights Efficiently

**Don't:**
- Read every word of every document
- Copy entire documents into your context

**Do:**
- Skim for structure (headings, lists, tables)
- Extract key insights (1-3 sentences per section)
- Note open questions or decisions
- Link to the full document for reference

**Template:**

```markdown
## Insights from [Document Title]

**Author:** [Name]  
**Date:** [YYYY-MM-DD]  
**Status:** [Draft/Active/Final]  
**Link:** [Path to document]

**Key Insights:**
- [Insight 1]
- [Insight 2]
- [Insight 3]

**Open Questions:**
- [Question 1]
- [Question 2]

**Decisions:**
- [Decision 1: Rationale]

**Relevance to Current Task:**
[How this informs what I'm working on]
```

---

## Writing to the Workspace

### 1. Choose the Right Location

**Ask:**
- Is this work in progress or finalized?
- Is this a discussion, specification, or research?
- Who is the intended audience?

**Decision Tree:**

```
Is this finalized?
├─ No → 00_Active/
│   ├─ Is this a discussion? → discussions/
│   ├─ Is this a draft spec? → drafts/
│   └─ Is this a decision? → decisions/
└─ Yes → Where does it belong?
    ├─ Specification → 01_Specifications/
    ├─ Research → 02_Research/
    ├─ Knowledge/Pattern → 03_Memory/
    ├─ Artifact/Output → 04_Artifacts/
    └─ Completed/Deprecated → 05_Archive/
```

### 2. Use Structured Templates

**Every document should follow a template** to ensure consistency and clarity.

#### Discussion Template

```markdown
---
title: [Discussion Topic]
author: [Your Name]
date: [YYYY-MM-DD]
status: Active
tags: [tag1, tag2, tag3]
participants: [agent1, agent2]
---

# [Discussion Topic]

## Context

[What is the background or situation that prompted this discussion?]

## Question / Problem

[What are we trying to decide or solve?]

## Perspectives

### Perspective 1: [Name]

[Description of this perspective]

**Pros:**
- [Pro 1]
- [Pro 2]

**Cons:**
- [Con 1]
- [Con 2]

### Perspective 2: [Name]

[Repeat structure]

## Open Questions

- [ ] [Question 1]
- [ ] [Question 2]

## Next Steps

- [ ] [Action 1: Owner]
- [ ] [Action 2: Owner]

## References

- [Link to related discussion]
- [Link to relevant spec]
```

#### Decision Template

```markdown
---
title: [Decision Title]
author: [Your Name]
date: [YYYY-MM-DD]
status: Final
tags: [tag1, tag2, tag3]
decision_id: [Unique ID, e.g., DEC-001]
---

# Decision: [Title]

## Context

[What situation led to this decision?]

## Decision

[What was decided? Be specific and actionable.]

## Rationale

[Why was this decision made?]

**Factors Considered:**
- [Factor 1]
- [Factor 2]
- [Factor 3]

**Alternatives Considered:**
- [Alternative 1: Why rejected]
- [Alternative 2: Why rejected]

## Implications

**Immediate:**
- [Implication 1]
- [Implication 2]

**Long-term:**
- [Implication 1]
- [Implication 2]

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| [Risk 1] | Low/Med/High | Low/Med/High | [How to address] |

## Review Criteria

[How will we know if this decision was correct?]

- [ ] [Criterion 1]
- [ ] [Criterion 2]

## References

- [Link to discussion]
- [Link to research]
```

#### Specification Template

See `specification-writer` skill for full template.

#### Research Template

See `research-modes` skill for full template.

### 3. Write for Clarity and Scannability

**Principles:**
- Use headings to create structure
- Use lists for enumerations
- Use tables for comparisons
- Use blockquotes for key insights
- Use code blocks for examples

**Don't:**
- Write long paragraphs without structure
- Bury key insights in prose
- Use vague language ("maybe", "possibly", "might")

**Do:**
- Make headings descriptive ("Memory Compression Strategy" not "Strategy")
- Put key insights at the top (executive summary)
- Use specific language ("Use 3-month rule" not "Consider time-based compression")

### 4. Link to Related Content

**Always include a "References" or "Related" section** to help others navigate.

**Format:**

```markdown
## Related Content

**Discussions:**
- [Discussion: Memory Architecture](../00_Active/discussions/2026-02-01_memory-architecture_manus.md)

**Specifications:**
- [Spec: Context Compression](../01_Specifications/architecture/context-compression.md)

**Research:**
- [Research: Semantic Compression Techniques](../02_Research/deep-dives/semantic-compression.md)

**Seeds:**
- [Seed: 3-Month Rule](../03_Memory/seeds/3-month-rule.md)
```

### 5. Update the Index

**After creating or updating a document**, update the workspace `README.md` to reflect:
- New active discussions
- Completed decisions
- Finalized specifications
- Key research findings

---

## Collaboration Patterns

### Pattern 1: Threaded Discussions

**Use when:** Multiple agents are exploring a topic together

**Process:**
1. Agent A creates initial discussion document in `00_Active/discussions/`
2. Agent B reads and adds their perspective to the same document
3. Agent C reads both perspectives and synthesizes
4. When consensus is reached, move to `00_Active/decisions/`

**Template:**

```markdown
## Discussion Thread

### [Agent A] - [Date]

[Perspective or question]

### [Agent B] - [Date]

[Response or alternative perspective]

### [Agent C] - [Date]

[Synthesis or recommendation]
```

### Pattern 2: Specification Review

**Use when:** One agent drafts a spec, another reviews

**Process:**
1. Agent A creates draft in `00_Active/drafts/`
2. Agent A notifies Agent B (via discussion or shared log)
3. Agent B reviews and adds inline comments
4. Agent A addresses comments and updates draft
5. When finalized, move to `01_Specifications/`

**Comment Format:**

```markdown
<!-- COMMENT [Agent B]: This section needs more detail on error handling -->
```

### Pattern 3: Parallel Research

**Use when:** Multiple agents research different aspects of the same problem

**Process:**
1. Agents agree on research scope and division of labor
2. Each agent creates their own research document in `02_Research/`
3. Agents read each other's findings
4. One agent synthesizes into a unified document
5. Synthesized document is moved to `03_Memory/` as shared knowledge

### Pattern 4: Seed Sharing

**Use when:** An agent extracts a reusable pattern

**Process:**
1. Agent extracts seed using `seed-extraction` skill
2. Agent documents seed in `03_Memory/seeds/`
3. Agent updates seed index in `03_Memory/seeds/README.md`
4. Other agents can discover and apply the seed

---

## Token Efficiency Strategies

### 1. Surgical Reading

**Don't:**
- Read entire workspace into context
- Load all files in a directory

**Do:**
- Read README.md first (500-1000 tokens)
- Identify 2-3 relevant files (2000-5000 tokens each)
- Extract key insights into a summary (500-1000 tokens)
- Total: ~5,000-10,000 tokens vs. 50,000+ tokens

### 2. Incremental Context Building

**Pattern:**
1. Start with minimal context (README + current task)
2. Add context as needed (specific files)
3. Compress context after each major milestone
4. Archive completed work

### 3. Reference, Don't Copy

**Don't:**
- Copy entire documents into your working memory
- Duplicate content across multiple files

**Do:**
- Link to source documents
- Extract key insights (1-3 sentences)
- Reference by path and section

**Example:**

```markdown
As documented in [Memory Architecture Spec](../01_Specifications/architecture/memory.md#compression-strategy), we use the 3-month rule for semantic compression.
```

### 4. Use Metadata for Filtering

**Before reading a file**, check metadata:
- Is this relevant? (tags, title)
- Is this current? (date, status)
- Is this authoritative? (author, decision_id)

**This saves:**
- Reading time (minutes)
- Token cost (thousands of tokens)
- Context pollution (irrelevant information)

---

## Maintenance Practices

### Weekly Review (30 minutes)

**Tasks:**
1. Review `00_Active/` directory
   - Move finalized drafts to appropriate directories
   - Archive completed discussions
   - Update decision log
2. Update workspace README.md
   - Reflect current priorities
   - Update navigation guide
3. Compress old discussions
   - Extract key insights into memory
   - Archive full discussions

### Monthly Audit (1-2 hours)

**Tasks:**
1. Review `01_Specifications/` for outdated specs
2. Review `03_Memory/` for unused seeds
3. Archive deprecated content to `05_Archive/`
4. Update directory structure if needed
5. Refactor for clarity and efficiency

---

## Quality Checklist

Before committing content to the workspace, verify:

### Structure
- [ ] File is in the correct directory
- [ ] File follows naming convention
- [ ] Metadata (frontmatter) is complete
- [ ] Headings create clear structure

### Content
- [ ] Purpose is clear (why this document exists)
- [ ] Key insights are at the top
- [ ] Content is scannable (headings, lists, tables)
- [ ] Links to related content are included

### Collaboration
- [ ] Author is identified
- [ ] Status is set (Draft/Active/Final/Archived)
- [ ] Tags enable discovery
- [ ] Workspace README is updated (if needed)

### Token Efficiency
- [ ] Content is concise (no unnecessary prose)
- [ ] Structure enables surgical reading
- [ ] Metadata enables filtering
- [ ] References are used instead of duplication

---

## Common Pitfalls to Avoid

❌ **File Dumping:** Creating files without structure → ✅ Use directory hierarchy  
❌ **Orphan Documents:** No links to/from other content → ✅ Always link related content  
❌ **Stale Content:** Old drafts never finalized or archived → ✅ Weekly review and cleanup  
❌ **Vague Naming:** "thoughts.md", "notes.md" → ✅ Descriptive names with dates  
❌ **No Metadata:** Can't filter or discover → ✅ Always include frontmatter

---

## Usage Instructions

1. **Read this skill** before working in a shared workspace
2. **Read the workspace README.md** to understand structure and conventions
3. **Navigate by purpose** (use directory structure, not chronological browsing)
4. **Use templates** when creating new content
5. **Link to related content** to enable discovery
6. **Update the index** when adding significant content
7. **Review and maintain** regularly (weekly/monthly)

---

## Skill Metadata

**Token Savings:** ~10,000-20,000 tokens per session (surgical reading, structured navigation, metadata filtering)  
**Quality Impact:** Ensures workspace remains organized, discoverable, and collaborative  
**Maintenance:** Update when workspace structure evolves  

**Related Skills:**
- `specification-writer` - Use for writing specs in the workspace
- `research-modes` - Use for conducting research in the workspace
- `seed-extraction` - Use for extracting and documenting seeds
- `memory-garden` - Use for documenting memory entries

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
