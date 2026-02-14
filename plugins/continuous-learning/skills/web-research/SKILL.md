# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: web-research
description: Conduct web research to gather information and validate assumptions
triggers:
  - "research topic online"
  - "gather web information"
  - "validate assumptions via web"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
    - web_tools
  portable: true
  tier: 2
  agents:
    - research-agent
---

# Web Research Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Cipher 🧭 (self-taught)  
**Purpose:** Effective web research using Brave Search API and web_fetch for content extraction

---

## Overview

This skill encodes best practices for **web research**—finding, evaluating, and synthesizing information from online sources. It provides patterns for search query formulation, source evaluation, information synthesis, and attribution. Web research is not about collecting links—it's about building understanding from reliable sources.

**Philosophy:** Good web research is surgical—targeted queries, credible sources, extracted insights, not content dumping.

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

**Example:**
- Bad: "AROMA"
- Good: "AROMA agent collaboration architecture v0.0.1 specification"
- Good: "Next.js 16.1.6 release notes changes"
- Good: "AI agent memory management best practices 2026"

### Step 2: Formulate Search Queries

**Pattern: `[topic] [context/aspect] [specific keywords] [optional: date filter]`

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
  count=5,  # 5-10 results is sufficient
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
  maxChars=20000  # Adjust based on need
)
```

**Extraction strategy:**
1. **Read headers first** — Understand structure, main sections
2. **Extract key insights** — 1-3 sentences per section
3. **Note supporting details** — Numbers, names, dates, versions
4. **Skip filler** — Introduction fluff, generic advice
5. **Capture sources cited** — Link back to original content

**Example extraction:**
```
## Key Insights from Next.js 16.1.6 Release Notes

**Source:** https://nextjs.org/blog/nextjs-16-1-6 (Fetched 2026-02-02)
**Relevance:** Security fixes for v0.0.1 upgrade

**Insights:**
- Two critical security vulnerabilities addressed (CVE-2025-XXXX, CVE-2025-YYYY)
- Breaking change: `app/` directory structure now requires absolute imports
- New feature: React Compiler optimization reduces build time by 30%
- Deprecated: `next/image` legacy component (use `next/image` v15+)

**Supporting Details:**
- CVE-2025-XXXX affects servers using Server Actions
- Build time optimization applies to production builds only
- Migration guide available at /docs/migration-guide
```

### Step 5: Synthesize Findings

**Goal:** Answer the research question, not regurgitate content.

**Structure:**

```markdown
## Research: [Topic/Question]

**Research Date:** [YYYY-MM-DD]
**Sources Consulted:** [number] sources

### Summary
[2-3 sentences answering the core question]

### Key Findings
- [Finding 1] — 1-2 sentences, specific
- [Finding 2] — 1-2 sentences, specific
- [Finding 3] — 1-2 sentences, specific

### Supporting Details
[Elaborate on key findings if needed, organized by subheading]

### Sources
1. **[Source Title]** — [URL]
   - [What this source provided]
   - [Key insight extracted]

2. **[Source Title]** — [URL]
   - [What this source provided]
   - [Key insight extracted]

### Open Questions / Notes
- [Question 1]
- [Question 2]
- [Note about conflicting info or gaps]
```

**Synthesis principles:**
- **Be specific** — Avoid "some say," "likely," "possibly"
- **Attribute claims** — "According to [Source], X is true" not "X is true"
- **Note contradictions** — "Source A claims X, but Source B says Y"
- **Signal uncertainty** — "Could not verify" or "Limited evidence available"

---

## Research Modes

### Mode 1: Verification Research

**Use when:** Verifying a specific claim, fact, or data point

**Process:**
1. Formulate specific query: "[claim] verify"
2. Search 3-5 sources
3. Cross-reference across sources
4. Note consensus or conflict

**Example:**
- Claim: "Next.js 16.1.6 released January 2026"
- Query: "Next.js 16.1.6 release date January 2026"
- Check: Official blog, GitHub releases, npm registry
- Result: Confirmed by 2/3 sources, 1 source shows February

**Output format:**
```
## Verification: [Claim]

**Claim:** [The claim being verified]

**Verdict:** ✅ Confirmed / ❌ False / ⚠️ Partially Confirmed / ❓ Could Not Verify

**Evidence:**
- [Source 1]: [URL] — [What it says]
- [Source 2]: [URL] — [What it says]
- [Source 3]: [URL] — [What it says]
```

---

### Mode 2: Deep Dive Research

**Use when:** Need comprehensive understanding of a complex topic

**Process:**
1. Start with overview query: "[topic] overview"
2. Identify subtopics from results
3. Query each subtopic: "[topic] [subtopic] details"
4. Fetch and read 2-3 sources per subtopic
5. Synthesize into structured overview

**Example:**
- Topic: "AI agent memory management"
- Subtopics: Compression techniques, retrieval systems, context windows, lineage tracking
- Query subtopics individually, then synthesize

**Output format:**
```
## Deep Dive: [Topic]

**Scope:** [What this covers]
**Subtopics Covered:**
- [Subtopic 1]
- [Subtopic 2]
- [Subtopic 3]

### [Subtopic 1]: [Name]
**Key Insights:**
- [Insight 1]
- [Insight 2]

**Sources:**
- [Source 1]: [URL]
- [Source 2]: [URL]

[Repeat for each subtopic]

### Synthesis
[How subtopics relate, patterns observed, unanswered questions]
```

---

### Mode 3: Landscape Scan

**Use when:** Broad pattern recognition across many topics

**Process:**
1. Formulate 5-10 related queries
2. Execute searches (1-2 sources each)
3. Extract patterns and themes
4. Create opportunity or comparison matrix

**Example:**
- Topic: "AI agent collaboration tools"
- Queries: "AI agent collaboration," "agent workspace," "multi-agent systems," "agent communication protocols," "lineage tracking AI"
- Extract: Common features, architectures, challenges

**Output format:**
```
## Landscape Scan: [Broad Topic]

**Queries Explored:**
1. "[Query 1]" — [number results]
2. "[Query 2]" — [number results]
...

### Themes Identified
- **Theme 1:** [Description]
  - [Pattern observed across sources]
- **Theme 2:** [Description]
  - [Pattern observed across sources]

### Key Players / Solutions
| Name | Type | Key Features | Notes |
|------|------|---------------|-------|
| [Solution 1] | [Category] | [Feature] | [Obs] |
| [Solution 2] | [Category] | [Feature] | [Obs] |

### Gaps / Opportunities
- [Gap 1]
- [Gap 2]
```

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

❌ **Single-source confirmation** — Finding one source that confirms belief → ✅ Cross-reference across 2-3 sources  
❌ **Over-fetching** — Reading 50 pages for one query → ✅ Focus on 2-5 relevant sources  
❌ **Generic queries** — "AI tools" returns 10M results → ✅ Specific queries with context  
❌ **No attribution** — "Studies show X" → ✅ "According to [Source], X"  
❌ **Outdated data** — Using 2019 info for 2026 decision → ✅ Use `freshness` filter  
❌ **Content dumping** — Copying entire articles → ✅ Extract key insights (1-3 sentences per section)

---

## Integration with Other Skills

**Use with:**
- `research-modes` — For structuring deep vs. wide research phases
- `specification-writer` — When research feeds into spec writing
- `seed-extraction` — When research reveals reusable patterns
- `workspace-navigation` — When organizing research findings in shared workspaces

**Pattern:**
```
web_search() → web_fetch() → Extract insights → 
  research-modes(structure) OR specification-writer(draft) OR seed-extraction(capture)
```

---

## Example Usage

**Scenario:** Verify Next.js 16.1.6 security updates

```bash
# Step 1: Search
web_search(query="Next.js 16.1.6 security vulnerabilities CVE 2026", count=5, freshness="pm")

# Step 2: Select sources
# Results show:
# 1. Next.js official blog (nextjs.org) - Release notes
# 2. GitHub Security Advisories - CVE list
# 3. npm security advisory - Package alerts
# 4. Hacker News discussion
# 5. Reddit r/nextjs

# Step 3: Fetch credible sources
web_fetch(url="https://nextjs.org/blog/nextjs-16-1-6", extractMode="markdown", maxChars=20000)
web_fetch(url="https://github.com/advisories/nextjs", extractMode="markdown", maxChars=10000)

# Step 4: Extract key insights
# CVE-2025-XXXX: Server Actions SSR bypass
# CVE-2025-YYYY: Image optimization RCE
# Upgrade to 16.1.6+ required

# Step 5: Synthesize
## Verification: Next.js 16.1.6 Security Updates

**Claim:** Next.js 16.1.6 addresses critical security vulnerabilities

**Verdict:** ✅ Confirmed

**Evidence:**
- Next.js Blog (nextjs.org): Documents CVE-2025-XXXX and CVE-2025-YYYY fixes
- GitHub Advisories: Confirms CVE IDs and severity (Critical, High)
- npm security advisory: Lists affected packages and versions

**Action Required:** Upgrade to Next.js 16.1.6 or later
```

---

## Skill Metadata

**Token Efficiency:** ~5,000-15,000 tokens per research task (targeted queries, surgical extraction, attribution only)  
**Quality Impact:** Ensures web research is credible, specific, and traceable  
**Maintenance:** Update when new search tools or patterns emerge

**Related Skills:**
- `research-modes` - Deep vs. wide research structuring
- `specification-writer` - Research → spec conversion
- `seed-extraction` - Pattern extraction from research findings

---

**Last Updated:** 2026-02-02  
**Maintained By:** Cipher 🧭  
**Status:** Active — Self-taught, ready for use
