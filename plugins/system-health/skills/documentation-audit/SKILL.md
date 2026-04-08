---
name: documentation-audit
description: >
  Systematic process for auditing project documentation to detect and correct
  drift -- the gap between what docs say and what the code actually does.
  Cross-references every document against the actual codebase to classify
  findings as Accurate, Drifted, Missing, or Orphaned. Use when documentation
  feels stale, after major releases, before onboarding new contributors, or
  on a recurring schedule. Trigger words: documentation drift, outdated docs,
  stale README, docs audit, missing documentation, broken links, doc review.
triggers:
  - "audit the documentation for drift"
  - "check if the docs match the actual code"
  - "find stale or outdated documentation"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - health-agent
---

# Documentation Audit Skill

**Version:** 1.0
**Created:** 2026-02-11
**Purpose:** Provide a structured, repeatable process for auditing project documentation, combating documentation drift, and ensuring all documents remain a reliable source of truth.

---

## I. The Philosophy: Tending the Garden of Knowledge

Project documentation is a garden. When tended with care, it is a source of clarity, guidance, and shared understanding. When neglected, it becomes overgrown with outdated information, broken links, and misleading instructions -- a phenomenon known as **documentation drift**. This drift erodes trust and creates confusion.

The documentation audit is the practice of tending the garden. It is a recurring ritual: walking through documentation, pulling the weeds of inaccuracy, and pruning the branches of irrelevance. It is an act of stewardship, ensuring that the shared garden of knowledge remains a welcoming and reliable resource.

Documentation drift is insidious because it happens silently. Code changes daily; documentation changes monthly (if you're lucky). The longer the gap between code changes and doc updates, the wider the drift. Regular audits catch drift early, when corrections are small and cheap.

---

## II. When to Use This Skill

- **After a major release:** To ensure all documentation reflects new features and changes.
- **Before onboarding a new contributor or agent:** To ensure they receive accurate information.
- **As a scheduled, recurring task:** (e.g., monthly) to maintain regular review cadence.
- **When documentation "feels" out of sync** with the code.
- **After a refactor or architectural change:** These are the highest-risk moments for drift.
- **As part of a broader health audit:** Dimension 5 (Documentation) of the health audit can trigger a deep documentation audit.

**When NOT to use:** For writing new documentation from scratch (that's a different skill). For reviewing a single document in isolation (just read and fix it). For auditing code quality (use the health audit skill instead).

---

## III. The Audit Workflow

### Step 1: Inventory All Documentation

**Goal:** Know what you're auditing before you start.

Locate and list every document in the repository:

| Document Type | Where to Look |
|---|---|
| Project overview | `README.md` at root |
| API documentation | OpenAPI specs, route docs, `docs/api/` |
| Architecture docs | `ARCHITECTURE.md`, `docs/architecture/`, design docs |
| Contribution guides | `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` |
| Changelog | `CHANGELOG.md`, release notes |
| Status | `STATUS.md` |
| Inline documentation | Code comments, docstrings, JSDoc, GoDoc |
| Guides and tutorials | `docs/guides/`, `docs/tutorials/` |
| Configuration docs | `.env.example`, config documentation |

**Output:** A complete list of documents to audit with file paths.

### Step 2: Cross-Reference Against the Codebase

**Goal:** Compare what docs say against what the code actually does.

For each document, systematically check:

| Check | What to Compare |
|---|---|
| **API accuracy** | Do documented endpoints match actual routes and handlers? |
| **Type accuracy** | Do documented types, interfaces, and schemas match actual definitions? |
| **Workflow accuracy** | Do documented workflows and processes match actual code paths? |
| **Dependency accuracy** | Are documented dependencies current with the lockfile and manifest? |
| **Configuration accuracy** | Do documented env vars and configs match actual usage in code? |
| **Example accuracy** | Do code examples and snippets actually run? |
| **Link accuracy** | Do internal and external links resolve? |

**Key principle:** Cross-reference docs against code, not just read the docs in isolation. A doc that reads well but describes a feature that was removed is worse than no doc at all.

### Step 3: Categorize Findings

**Goal:** Classify every finding into one of 4 categories.

| Category | Definition | Action | Severity Tendency |
|---|---|---|---|
| **Accurate** | Docs match code. No issues. | None needed | -- |
| **Drifted** | Docs are outdated but partially correct. The feature exists but details are wrong. | Update the documentation | Medium-High |
| **Missing** | Code has functionality with no corresponding documentation. | Write new documentation | Medium |
| **Orphaned** | Docs describe something that no longer exists in the codebase. | Remove or archive | High (actively misleading) |

**Severity levels:**
- **High:** Misleading -- someone following these docs would do the wrong thing.
- **Medium:** Incomplete -- someone would need to figure things out on their own.
- **Low:** Cosmetic -- typos, formatting, or minor inaccuracies that don't cause harm.

### Step 4: Produce Correction Tasks

**Goal:** Generate an actionable task for every non-accurate finding.

For each finding, produce:

| Field | Description |
|---|---|
| **File** | The documentation file affected |
| **Line(s)** | Specific line numbers if applicable |
| **Issue** | What is wrong |
| **Category** | Drifted / Missing / Orphaned |
| **Severity** | High / Medium / Low |
| **Correction** | What specifically should change |

### Step 5: Produce Audit Report

**Goal:** Create a permanent, dated artifact.

Save as `docs/audits/[YYYY-MM-DD]_documentation_audit.md` with:

1. **Summary** -- Total documents reviewed, issues by category, overall documentation health (GREEN/YELLOW/RED).
2. **Findings Table** -- All findings with file, lines, issue, category, severity, and correction.
3. **Documentation Health by Area** -- Rate each documentation area with GREEN/YELLOW/RED:
   - README and project overview
   - API documentation
   - Architecture documentation
   - Inline code documentation
   - Guides and tutorials
   - Configuration documentation
4. **Correction Tasks** -- Prioritized list of remaining fixes.

---

## IV. Best Practices

### 1. Audit in Small, Regular Batches

It is less daunting to audit one section of the documentation each week than to audit the entire repository once a year. Regular small audits prevent drift from compounding.

### 2. Compare Docs to Code, Not Docs to Memory

Never assess documentation accuracy from memory. Always open the actual source file and compare. Memory is unreliable; code is the source of truth.

### 3. Link, Don't Copy

When information needs to exist in multiple places, link to a single source of truth rather than copying. Copied information drifts independently. Linked information stays in sync.

### 4. Automate What You Can

Use tools for broken link detection, dependency version checking, and API spec validation. Manual review should focus on semantic accuracy -- the things automation can't check.

### 5. Prioritize Orphaned Documentation

Orphaned docs (describing features that no longer exist) are the most dangerous category because they actively mislead. Prioritize their removal or archival over other fixes.

### 6. Every Fix is a Good Fix

Even fixing a small typo improves the quality of the garden. Don't dismiss low-severity findings -- they add up over time.

---

## V. Quality Checklist

Before completing a documentation audit, confirm:

- [ ] Every document in the repository was inventoried
- [ ] Docs were cross-referenced against actual code, not just read in isolation
- [ ] Every finding is categorized as Accurate, Drifted, Missing, or Orphaned
- [ ] Severity is assigned to every non-accurate finding
- [ ] Correction tasks have specific file paths and descriptions of what to change
- [ ] The audit report is saved as a dated file in `docs/audits/`
- [ ] Overall documentation health uses GREEN/YELLOW/RED classification
- [ ] Health is rated for each documentation area separately
- [ ] Orphaned documentation is flagged as high priority

---

## VI. Common Pitfalls

### Pitfall 1: Reading Docs Without Comparing to Code

**Problem:** The audit just reads the documentation and checks if it "sounds right" without verifying against the actual codebase.

**Solution:** For every claim in the documentation, open the corresponding source file and verify. Does the documented API endpoint exist? Does the documented type match the actual definition?

### Pitfall 2: Only Auditing the README

**Problem:** The README is checked but inline docs, API docs, and architecture docs are ignored.

**Solution:** The inventory step (Step 1) exists to ensure completeness. Don't skip it. Some of the worst drift hides in architecture docs and API specifications that nobody reads regularly.

### Pitfall 3: Fixing Without Recording

**Problem:** Issues are fixed on the spot but never logged, so there's no record of what was found.

**Solution:** Always create the audit report first, then fix. The audit trail is as important as the fixes themselves -- it enables trend tracking across audits.

### Pitfall 4: Ignoring Missing Documentation

**Problem:** The audit only checks existing docs for accuracy but doesn't identify undocumented features.

**Solution:** Walk the codebase to find public APIs, configuration options, and workflows that have no corresponding documentation. Missing docs are a finding, not a non-finding.

### Pitfall 5: Treating All Findings as Equal Priority

**Problem:** Typos are given the same urgency as misleading API documentation.

**Solution:** Use the severity levels (High/Medium/Low) consistently. Orphaned and drifted high-severity findings should be fixed immediately. Low-severity cosmetic issues can be batched.

---

## VII. Example: Repository Documentation Audit

**The Problem:** A web application's documentation hasn't been updated since the last major refactor 2 months ago. New contributors are complaining about inaccurate setup instructions.

**The Process:**

1. **Inventory:** Found 8 documentation files: README.md, ARCHITECTURE.md, CONTRIBUTING.md, 3 API docs in `docs/api/`, .env.example, and inline JSDoc across 42 source files.

2. **Cross-reference:** Compared each document against the codebase:
   - README setup instructions reference a removed `npm run seed` command
   - API docs list 12 endpoints but the codebase has 15
   - ARCHITECTURE.md diagram missing the new caching layer
   - .env.example missing 3 environment variables added in the refactor

3. **Findings:**
   - Accurate: 2 (CONTRIBUTING.md, 1 API doc)
   - Drifted: 4 (README, ARCHITECTURE.md, .env.example, 1 API doc)
   - Missing: 3 (3 new API endpoints undocumented)
   - Orphaned: 1 (API doc for removed `/legacy/sync` endpoint)

4. **Report:** Saved as `docs/audits/2026-02-11_documentation_audit.md`. Overall health: YELLOW. Highest priority: remove orphaned API doc and fix README setup instructions.

**The Outcome:** 6 issues fixed in one session, 2 new API docs created in the following sprint. Follow-up audit 1 month later showed documentation health improved from YELLOW to GREEN.

---

## VIII. Related Skills

- **`health-audit`** -- The documentation dimension of a health audit can trigger a deep documentation audit
- **`status-writing`** -- STATUS.md is one of the documents that should be checked during an audit
- **`semantic-clusters`** -- Architecture docs should be compared against the behavioral architecture map
