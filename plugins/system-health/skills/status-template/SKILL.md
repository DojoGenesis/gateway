---
name: status-template
model: sonnet
description: Produces a comprehensive `.status.md` file using a 10-section schema that bridges heartbeat (current state), anatomy (directory structure), and checkup (health assessment) perspectives into one document. Use when: "write a status report", "give me the lay of the land", "where are we right now", "prepare a handoff document", "document the current codebase state".
category: system-health

inputs:
  - name: repo_path
    type: string
    description: Path to the repository or project to write status for
    required: true
outputs:
  - name: status_file
    type: ref
    format: cas-ref
    description: Comprehensive .status.md using a 10-section schema bridging heartbeat, anatomy, and checkup perspectives
---

# Status Template Skill

**Version:** 1.0
**Created:** 2026-02-08
**Author:** Cruz + Manus (Cowork)
**Origin:** Elevated from the `repo-status` skill's status template reference, which was itself codified from a live status-writing + file-management + health-audit session.
**Lineage:** status-writing (heartbeat) + file-management (anatomy) + health-audit (checkup) → repo-status → this standalone skill.

---

## I. The Philosophy: One Document, Three Perspectives

Project status lives in too many places — Slack threads, standup notes, README footers, someone's head. When a new person (or agent) arrives and asks "where are we?", the answer is usually scattered, stale, or both.

This skill produces a single `.status.md` file that answers that question by looking at the project from three angles:

- **Heartbeat (status-writing):** Is it alive? What's moving? What's stuck? Emoji-driven health indicators give an instant pulse check.
- **Anatomy (file-management):** Where is everything? An annotated directory tree with per-folder status turns the filesystem into a map.
- **Checkup (health-audit):** How healthy is it? Critical issues, security posture, and sustainability assessment tell you what needs attention.

One file. Three lenses. Updated over time.

---

## II. When to Use This Skill

- **Writing a status report:** Monthly, per-release, or ad-hoc — this template gives you the structure.
- **Onboarding someone to a project:** Instead of a 30-minute walkthrough, hand them a `.status.md`.
- **Preparing a handoff:** Moving a project between teams or agents? This is the context document.
- **Before a major decision:** Understanding the full state before deciding what to change.
- **When a project feels opaque:** Structure reduces confusion. The template imposes coherence on complexity.
- **After running the `repo-status` skill:** This template is Phase 4 of that workflow.

---

## III. The Template Structure

The status document has 10 sections. Each one has a specific job.

### Section 1: Vision & Purpose
**Job:** Anchor everything that follows. One sentence + core principles.

Why it matters: Without a stated purpose, status updates become lists of things. With one, they become a story about progress toward a goal.

```markdown
## 1. Vision & Purpose
> [One sentence capturing what the project is and why it exists.]
**Core Principles:** [3-5 comma-separated principles]
```

### Section 2: Current State
**Job:** Instant pulse check. A table of major areas with emoji status and brief notes.

Why it matters: Someone glancing at this document for 5 seconds should know if things are healthy or on fire. The emoji table achieves this.

```markdown
## 2. Current State
| Area | Status | Notes |
| :--- | :--- | :--- |
| **[Major Area]** | ✅/🔄/⚠️/❌ | [One-line note] |
```

**Status Key:**
- ✅ Complete/Healthy
- 🔄 In Progress
- ⏸️ Paused
- ⚠️ Concern
- ❌ Blocked/Critical

### Section 3: Directory Structure
**Job:** Annotated filesystem map. Every significant directory gets a status emoji and note.

Why it matters: Most codebases lack a visual map of what's where. An annotated tree is worth a thousand words for spatial orientation.

Use `📦` for legacy/deprecated directories. Use `—` for empty or minimal ones.

### Section 4: Semantic Clusters
**Job:** Group components by *what they do*, not where they live.

Why it matters: This is the behavioral architecture. A feature spans frontend + backend + database + tests. Clusters reveal the system's capabilities as cross-cutting concerns.

If using semantic clusters, invoke the `semantic-clusters` skill for the full framework. Otherwise, this section can be simplified to a feature list with health assessments.

For each cluster:
```markdown
### [emoji] [VERB] — [Short Description]
> [One sentence explaining this capability.]
| Component | Location | Status | LOC |
|-----------|----------|--------|-----|
**Health:** [emoji] [assessment]
**Audit Notes:** [key technical details]
```

### Section 5: File Importance Ranking
**Job:** Rank files by criticality so people know what not to break.

Why it matters: In a 500-file codebase, 10 files are critical, 10 are important, and the rest are supporting. Making this explicit prevents accidental damage to core files.

Four tiers: Critical (system breaks without these), Important (features break), Supporting (graceful degradation), Knowledge (development-time only).

### Section 6: Health Assessment
**Job:** Honest checkup. Critical issues, security, sustainability.

Why it matters: This is the health-audit perspective. Every project has blind spots — security gaps, test coverage holes, manual processes that should be automated. This section forces them into the light.

Three categories:
- **Critical Issues:** Can it build? Critical vulnerabilities? Main branch broken?
- **Security:** Secrets management, auth, encryption, access control.
- **Sustainability:** Test coverage, CI/CD, tech debt, documentation freshness.

### Section 7: Active Workstreams
**Job:** What's being worked on right now, by whom.

A table: `Workstream | Owner | Status | Focus`

### Section 8: Blockers & Dependencies
**Job:** What's preventing progress. Be ruthlessly honest.

If nothing is blocking, say "None." That's valuable information too.

### Section 9: Next Steps
**Job:** Concrete, actionable items. Numbered list with emoji status.

Not aspirational goals — specific things that should happen next.

### Section 10: Aggregate Statistics
**Job:** Summary numbers table for quick reference.

LOC, file counts, package counts, test coverage, migration counts, CI/CD workflow counts, versions shipped.

---

## IV. How to Fill the Template

Read `references/complete-template.md` for the full, copy-paste-ready template with all sections, placeholders, and HTML comments for guidance.

**The filling process:**

1. **Start with Section 1.** If you can't write the vision in one sentence, you don't understand the project well enough yet. Go back and read more.
2. **Fill Section 2 next.** The current state table forces you to identify the major areas. This becomes your map for the rest.
3. **Sections 3-4 require exploration.** Walk the filesystem, read key files, count LOC. This is the most time-consuming part.
4. **Section 5 requires judgment.** Which 10 files, if deleted, would break the system? Which 10 more would break major features?
5. **Section 6 requires honesty.** The health assessment is useless if it's all green. Every project has concerns. Find them.
6. **Sections 7-9 require current knowledge.** Talk to the team, read recent PRs, check the issue tracker.
7. **Section 10 is verification.** Run `find` and `wc -l` to get real numbers. Don't guess.

---

## V. Best Practices

- **Dot-prefix the file.** Save as `.status.md` to keep it discoverable but out of the way. It complements (doesn't replace) a lightweight `STATUS.md`.
- **Update incrementally.** After the first write, subsequent updates should diff against the existing document, not regenerate from scratch.
- **Emoji consistency.** Use the same status key everywhere. Mixing conventions creates noise.
- **Be concise in tables.** Each table cell should be scannable in under 3 seconds. Full sentences belong in prose sections, not tables.
- **The directory tree should be 2-3 levels deep** for most directories, deeper only for architecturally significant subtrees.
- **Statistics will drift.** Within 10% is accurate enough. Note the date so readers know how fresh the numbers are.
- **Section 4 is optional for simple projects.** If the project has fewer than 50 files, skip semantic clusters and use a flat feature list instead.
- **Section 5 is optional for small projects.** Below 100 files, importance ranking adds little value.

---

## VI. Adapting the Template

Not every project needs all 10 sections. Here's a sizing guide:

| Project Size | Sections to Include |
|-------------|-------------------|
| **Small** (< 50 files) | 1, 2, 3, 6, 7, 9 |
| **Medium** (50-300 files) | 1, 2, 3, 4 (simplified), 6, 7, 8, 9, 10 |
| **Large** (300+ files) | All 10 sections |
| **Monorepo** | All 10, with sub-sections per package/service |

For non-software projects (documentation sites, design systems, data pipelines), adapt the vocabulary but keep the structure. Section 4 clusters might become "capabilities" or "data flows" instead of "action verbs."

---

## VII. Quality Checklist

Before delivering the `.status.md`, confirm:

- [ ] Vision statement is present, accurate, and one sentence
- [ ] Current state table covers every major subsystem
- [ ] Directory tree is annotated with status emojis
- [ ] Health assessment is honest (not all green)
- [ ] Active workstreams reflect actual current work (not aspirations)
- [ ] Blockers section is truthful (empty or populated, not missing)
- [ ] Next steps are concrete and actionable (not "improve things")
- [ ] Statistics are programmatically verified (not guessed)
- [ ] The document reads coherently top-to-bottom
- [ ] The document is dated with `Last Updated`

## Output
- `.status.md` at the project root with all applicable sections from the 10-section schema. Section count adapts to project size (small projects use 6 sections; large projects use all 10).
- The file complements but does not replace a lightweight `STATUS.md` — `.status.md` is the comprehensive agent-facing audit; `STATUS.md` is the human-facing dashboard.

## Examples
**Scenario 1:** "Create a project overview before a strategy session" → `.status.md` produced with all 10 sections. Vision and Purpose anchors everything. Current State table reveals 2 blocked areas. Health assessment honestly flags no CI/CD and 8% test coverage as sustainability concerns.
**Scenario 2:** "Prepare a handoff doc for the incoming agent" → `.status.md` covers a 120-file project. Sections 4 (Semantic Clusters) and 5 (File Importance Ranking) included. Incoming agent reads the document instead of spending 30 minutes on discovery.

## Edge Cases
- For projects under 50 files, skip Sections 4 and 5 (clusters and importance ranking) — they add little value at small scale, and the overhead of generating them exceeds the benefit.
- For monorepos, use all 10 sections with sub-sections per package or service under each primary section.

## Anti-Patterns
- Treating the Current State table as aspirational — the emoji indicators must reflect actual verified state, not what the team wishes were true.
- Generating statistics from memory or rough estimates — run `find` and `wc -l` to get real numbers; within 10% is accurate enough, but guessing is not.
