---
name: web-research-external
model: sonnet
description: "Produces structured research documents or implementation handoffs by searching external sources — library docs, web, APIs — for a given topic. Use when: 'research how to use a library', 'find best practices for a topic', 'compare options', 'look up API or framework documentation'. NOT for codebase exploration (use project-exploration for that)."
category: continuous-learning
triggers:
  - "research how to use a library"
  - "find best practices for"
  - "compare options"
  - "look up API documentation"
  - "look up framework documentation"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "web_tools"
  - "file_system"

inputs:
  - name: topic
    type: string
    description: The library, concept, or question to research
    required: true
  - name: focus
    type: string
    description: Research mode — "library" (API docs), "best-practices" (recommended approaches), or "general" (all sources)
    required: false
  - name: depth
    type: string
    description: Search depth — "shallow" (essentials only) or "thorough" (multi-source, examples, edge cases)
    required: false
  - name: output_format
    type: string
    description: Output type — "chat" (summary in conversation), "doc" (write research file), or "handoff" (implementation-ready context)
    required: false
outputs:
  - name: research_output
    type: ref
    format: cas-ref
    description: Research document (markdown) or implementation handoff (YAML) with findings, code examples, best practices, pitfalls, and sources
---

# Web Research External

Researches external sources — library documentation, web search, API references — and produces structured findings for decisions or implementation.

## Description

Covers three research modes: `library` (API docs and usage patterns for a specific package), `best-practices` (recommended approaches, comparisons, anti-patterns via web search), and `general` (all available sources for comprehensive coverage). Output is either a chat summary, a dated research document, or an implementation handoff ready for a coding agent. This skill is distinct from `project-exploration` (local codebase) and `research-synthesis` (processing existing notes).

## When to Use

- Looking up how to use a library or framework (npm, PyPI, crates.io, Go modules)
- Finding current best practices for a technology decision
- Comparing two or more approaches or tools
- Building implementation context before writing code
- Producing a research document to share with teammates or agents

Do not use for local codebase exploration — use `project-exploration` for that.

## Workflow

### Step 1: Clarify Intent

If focus, topic, or depth are not provided, guide the user through:

**Research type:**
- "How to use a library/package" → `library` focus
- "Best practices for a task" → `best-practices` focus
- "General topic research" → `general` focus
- "Compare options/alternatives" → `best-practices` with comparison framing

**Topic:** The specific library, concept, or question (e.g., "Prisma ORM connection pooling", "error handling in Python async").

**Depth:**
- "Quick answer" → shallow (essentials only)
- "Thorough research" → thorough (multiple sources, examples, edge cases)

**Output format:**
- "Summary in chat" → `chat`
- "Research document" → `doc` (write to file)
- "Handoff for implementation" → `handoff` (structured YAML for coding agent)

Show a confirmation summary before executing:
```
Focus: library | Topic: "Prisma ORM" | Depth: thorough | Output: doc
Proceed?
```

### Step 2: Execute Research by Focus

#### Focus: `library`

Use WebFetch to pull official documentation, README, and API reference. Use WebSearch for usage examples.

```
1. WebFetch: official docs URL for the package
2. WebSearch: "{library} {topic} examples site:github.com OR site:stackoverflow.com"
3. WebSearch: "{library} changelog OR migration guide" (if version-specific)
```

For thorough depth: add multiple semantic queries, grep for specific function/class names in examples, scrape additional documentation pages.

#### Focus: `best-practices`

Use WebSearch with current-year framing.

```
1. WebSearch: "{topic} best practices 2025"
2. WebSearch: "{topic} vs alternatives comparison"
3. WebSearch: "{topic} common mistakes anti-patterns"
```

For thorough depth: add chain-of-thought comparison queries, recent developments search (last 3 months), and cross-reference findings from multiple sources.

#### Focus: `general`

Use all available research tools in sequence:
1. WebSearch for library documentation and API references
2. WebSearch for best practices and comparisons
3. WebFetch specific documentation pages found in step 2 results

For thorough depth: run all three with expanded queries; cross-reference findings between sources; follow links for deeper context.

### Step 3: Synthesize Findings

Combine results across all sources into:

1. **Key Concepts** — Core ideas and terminology
2. **Code Examples** — Working examples from documentation
3. **Best Practices** — Recommended approaches
4. **Pitfalls** — Common mistakes to avoid
5. **Alternatives** — Other options considered
6. **Sources** — URLs for all citations

If a source is unavailable, continue with remaining sources and note the gap explicitly.

### Step 4: Write Output

#### Output: `doc`

Write to: `research/YYYY-MM-DD-{topic-slug}.md`

```markdown
---
date: {ISO timestamp}
type: external-research
topic: "{topic}"
focus: {focus}
depth: {depth}
sources: [count]
status: complete | partial | failed
---

# Research: {Topic}

## Summary
{2-3 sentence summary of findings}

## Key Findings

### {Source category}
{Findings from each source}

### Code Examples
{Working examples found}

## Best Practices
- {Practice 1}
- {Practice 2}

## Pitfalls to Avoid
- {Pitfall 1}

## Alternatives Considered
| Option | Pros | Cons |
|--------|------|------|

## Sources
- [{Source 1}]({url1})
```

#### Output: `handoff`

Write to: `research/{topic-slug}-handoff.yaml`

```yaml
type: research-handoff
topic: "{topic}"
status: complete

findings:
  key_concepts: [concept1, concept2]
  code_examples:
    - pattern: "{pattern name}"
      code: |
        // example code
  best_practices: [practice1, practice2]
  pitfalls: [pitfall1]

recommendations: [rec1, rec2]

sources:
  - title: "{Source 1}"
    url: "{url1}"
    type: documentation | article | reference

implementation_notes: |
  Recommended approach: ...
  Key libraries: ...
  Avoid: ...
```

#### Output: `chat`

Return a structured summary in the conversation with key findings, code examples, and source URLs. No file written.

### Step 5: Return Completion Summary

```
Research complete.

Topic: {topic}
Focus: {focus} | Depth: {depth}
Output: {path or "chat summary above"}
Sources: {N} sources cited

Key findings:
- {Finding 1}
- {Finding 2}
- {Finding 3}
```

## Output

A research document (`.md`) or implementation handoff (`.yaml`) with:
- Synthesized findings across all sources
- Working code examples
- Explicit source citations with URLs
- Status indicator (complete / partial / failed)
- Gap notes for any unavailable sources
