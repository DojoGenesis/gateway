---
name: web-research
model: sonnet
description: "Produces a structured Research Summary document (findings, sources, attributions) from targeted web queries using Brave Search and web_fetch. Use when: 'research this online', 'find current information about X', 'verify this claim', 'gather sources on Y', 'search for what the field says about Z'."
category: continuous-learning
triggers:
  - "research this online"
  - "find current information about"
  - "verify this claim"
  - "gather sources on"
  - "search for what the field says about"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "web_tools"

inputs:
  - name: query
    type: string
    description: The research question or topic to investigate online
    required: true
outputs:
  - name: research_summary
    type: ref
    format: cas-ref
    description: Structured Research Summary document with findings, sources, and attributions from web queries
---

# Web Research Skill

**Version:** 1.1
**Created:** 2026-02-02
**Author:** Cipher (self-taught)
**Purpose:** Effective web research using Brave Search API and web_fetch for content extraction

---

## Overview

This skill encodes best practices for **web research** — finding, evaluating, and synthesizing information from online sources. It provides patterns for search query formulation, source evaluation, information synthesis, and attribution. Web research is not about collecting links — it's about building understanding from reliable sources.

Output templates for all research modes are in `references/research-output-templates.md`.

---

## Core Tools

**Available Tools:**
- `web_search(query, count, country, search_lang, ui_lang, freshness)` — Brave Search API for finding sources
- `web_fetch(url, extractMode, maxChars)` — Extract readable content from URLs (markdown/text)

**Parameters:**
- `query`: Search query string (required)
- `count`: Number of results (1-10, default: 5)
- `country`: 2-letter country code for regional results (default: 'US')
- `search_lang`: ISO language code for search results
- `ui_lang`: ISO language code for UI elements
- `freshness`: Time filter for results (pd=past 24h, pw=past week, pm=past month, py=past year)
- `url`: HTTP or HTTPS URL to fetch
- `extractMode`: 'markdown' or 'text' (default: 'markdown')
- `maxChars`: Maximum characters to return (truncates when exceeded)

---

## When to Use This Skill

- Investigating a topic that requires current information
- Finding sources for research or specifications
- Verifying claims or facts
- Gathering competitive intelligence or landscape scans
- Understanding a technology, framework, or practice
- Finding documentation or examples

---

## Research Workflow

### Step 1: Define Your Research Question

**Before searching, clarify:**
1. What specific question am I trying to answer?
2. What level of detail do I need? (overview vs. deep dive)
3. Is this time-sensitive? (recent events, API versions, industry news)
4. What context do I already have? (avoid re-searching known info)

**Examples:**
- Bad: "AROMA"
- Good: "AROMA agent collaboration architecture v0.0.1 specification"
- Good: "AI agent memory management best practices 2026"

### Step 2: Formulate Search Queries

**Pattern:** `[topic] [context/aspect] [specific keywords] [optional: date filter]`

**Examples:**

| Topic | Bad Query | Good Query |
|-------|-----------|------------|
| AROMA | "AROMA" | "AROMA agent collaboration v0.0.1 specification" |
| Next.js bugs | "Next.js bugs" | "Next.js 16.1.6 security vulnerabilities CVE" |
| AI research | "AI research" | "AI agent memory compression techniques 2025" |
| Excel design | "Excel design" | "Excel data visualization best practices conditional formatting" |

**Freshness filters:**
- Breaking news: `freshness="pd"` (past 24 hours)
- Recent developments: `freshness="pw"` (past week)
- Tech releases: `freshness="pm"` (past month)
- Historical context: `freshness="py"` (past year) or no filter

### Step 3: Search and Select Sources

**Execute search:**
```
web_search(
  query="your query here",
  count=5,
  country="US",
  search_lang="en"
)
```

**Evaluate each source:**

| Criterion | What to Check | Good Signs | Bad Signs |
|-----------|---------------|-------------|------------|
| **Credibility** | Domain reputation | .gov, .edu, established news | Unknown blog, social posts |
| **Relevance** | Title and snippet match | Direct answer to question | Tangential content |
| **Freshness** | Publication date | Recent (past 1-3 months) | Outdated (>1 year old) |
| **Depth** | Content length and detail | 500+ words, specific details | 200-word overview |
| **Authority** | Author or org expertise | Named experts, official docs | Anonymous, generic content |

**Source selection:**
- Prioritize: Official docs, established news, expert publications
- Supplement: Community discussions, GitHub issues, forums
- Verify: Cross-reference claims across 2-3 sources

### Step 4: Fetch and Extract Content

**For relevant sources:**
```
web_fetch(
  url="https://example.com/page",
  extractMode="markdown",
  maxChars=20000
)
```

**Extraction strategy:**
1. Read headers first — Understand structure, main sections
2. Extract key insights — 1-3 sentences per section
3. Note supporting details — Numbers, names, dates, versions
4. Skip filler — Introduction fluff, generic advice
5. Capture sources cited — Link back to original content

### Step 5: Synthesize Findings

**Goal:** Answer the research question, not regurgitate content.

**Synthesis template** — see `references/research-output-templates.md` for the full Research Synthesis format.

**Synthesis principles:**
- Be specific — Avoid "some say," "likely," "possibly"
- Attribute claims — "According to [Source], X is true" not "X is true"
- Note contradictions — "Source A claims X, but Source B says Y"
- Signal uncertainty — "Could not verify" or "Limited evidence available"

---

## Research Modes

### Mode 1: Verification Research

**Use when:** Verifying a specific claim, fact, or data point

**Process:**
1. Formulate specific query: "[claim] verify"
2. Search 3-5 sources
3. Cross-reference across sources
4. Note consensus or conflict

**Output:** Verification document with Verdict (Confirmed / False / Partially Confirmed / Could Not Verify), Evidence section listing each source's position — see `references/research-output-templates.md`

---

### Mode 2: Deep Dive Research

**Use when:** Need comprehensive understanding of a complex topic

**Process:**
1. Start with overview query: "[topic] overview"
2. Identify subtopics from results
3. Query each subtopic: "[topic] [subtopic] details"
4. Fetch and read 2-3 sources per subtopic
5. Synthesize into structured overview

**Output:** Deep Dive document organized by subtopics with cross-subtopic synthesis — see `references/research-output-templates.md`

---

### Mode 3: Landscape Scan

**Use when:** Broad pattern recognition across many topics

**Process:**
1. Formulate 5-10 related queries
2. Execute searches (1-2 sources each)
3. Extract patterns and themes
4. Create opportunity or comparison matrix

**Output:** Landscape Scan document with themes, key players table, and gaps list — see `references/research-output-templates.md`

---

## Quality Checklist

Before concluding web research, verify:

### Source Quality
- [ ] At least 2-3 sources consulted (unless verification)
- [ ] Sources are credible (official docs, established news, experts)
- [ ] Sources are recent (within 1-3 years, unless historical context)
- [ ] Conflicting information is noted

### Content Quality
- [ ] Specific claims are supported by evidence
- [ ] Attribution is clear (which source said what)
- [ ] Uncertainty is signaled (couldn't verify, limited data)
- [ ] Filler is minimized (no generic advice)

### Synthesis Quality
- [ ] Research question is answered directly
- [ ] Key findings are specific and actionable
- [ ] Sources are cited with URLs
- [ ] Open questions or gaps are noted

---

## Common Pitfalls to Avoid

- **Single-source confirmation** — Finding one source that confirms belief — cross-reference across 2-3 sources
- **Over-fetching** — Reading 50 pages for one query — focus on 2-5 relevant sources
- **Generic queries** — "AI tools" returns 10M results — use specific queries with context
- **No attribution** — "Studies show X" — attribute: "According to [Source], X"
- **Outdated data** — Using 2019 info for a 2026 decision — use `freshness` filter
- **Content dumping** — Copying entire articles — extract key insights (1-3 sentences per section)

---

## Integration with Other Skills

**Use with:**
- `research-modes` — For structuring deep vs. wide research phases
- `specification-writer` — When research feeds into spec writing
- `seed-extraction` — When research reveals reusable patterns
- `workspace-navigation` — When organizing research findings in shared workspaces

**Pattern:**
```
web_search() -> web_fetch() -> Extract insights ->
  research-modes(structure) OR specification-writer(draft) OR seed-extraction(capture)
```

---

## Output

- A Research Summary markdown document answering the research question
- Saved to the project's `docs/research/` or `scouts/` directory
- Named: `[date]_[topic]_web_research.md`
- Includes: summary, key findings, supporting details, cited sources with URLs, open questions

---

## Examples

**Scenario 1:** "Find current information about WebSocket performance benchmarks for Go" → Verification/Deep Dive document with 3-5 authoritative sources, specific numbers, attribution per claim, and open questions flagged

**Scenario 2:** "Search for what the field says about AI agent orchestration patterns" → Landscape Scan document with theme clusters, key tool/framework table, and identified gaps across 8-10 sources

---

## Edge Cases

- **Paywalled content:** Note the source in the synthesis as "paywalled, snippet only" and extract what is available from the search snippet; do not fabricate full content
- **Contradictory authoritative sources:** Surface the contradiction explicitly; do not pick one silently — present both and note the disagreement
- **Topic with no recent sources (>2 years old):** Label findings as potentially outdated; recommend confirming with a domain expert or official changelog
- **Query returns zero relevant results:** Reformulate with 2-3 alternative phrasings before concluding the topic is unresearchable

---

## Anti-Patterns

- Fetching the full text of 10+ pages when 2-3 targeted extractions answer the question — token cost with no quality gain
- Summarizing sources in the order they were found rather than organizing findings by theme — produces a list, not a synthesis
- Accepting a source as authoritative because its domain sounds credible without checking the actual author or date
- Using web research for questions that can be answered from the existing codebase or memory — always check local context first

---

**Related Skills:**
- `research-modes` — Deep vs. wide research structuring
- `specification-writer` — Research to spec conversion
- `seed-extraction` — Pattern extraction from research findings

---

**Last Updated:** 2026-04-08
**Maintained By:** Cipher
**Status:** Active
