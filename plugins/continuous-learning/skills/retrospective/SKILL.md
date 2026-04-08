---
name: retrospective
description: A structured process for post-sprint learning and continuous improvement. Harvest learnings instead of conducting post-mortems. Use after a major release, milestone, or at regular intervals to reflect on what went well, what was hard, and what can be improved.
triggers:
  - "run a retro on this sprint"
  - "harvest learnings from what we just shipped"
  - "reflect on what went well and what was hard"
metadata:
  version: "1.1"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
    - strategic-agent
---

# Retrospective Skill

**Version:** 1.1
**Author:** Tres Pies Design
**Purpose:** Structured, repeatable process for conducting sprint retrospectives, harvesting learnings, and feeding them back into collaborative practice.

---

## I. Philosophy: Harvest, Not Post-Mortem

A retrospective is not a post-mortem. It is not about blame or judgment. It is a **harvest**. After a season of hard work, we pause to gather the fruits of our labor — not just the features we shipped, but the wisdom we gained in the process. It is a practice of gratitude, honesty, and a commitment to continuous learning.

This skill turns the informal act of looking back into a formal ritual, ensuring that valuable lessons from each sprint are not lost but integrated into shared memory and future workflows.

---

## II. When to Use This Skill

- **After a major release** (e.g., after shipping v1.0)
- **After a significant milestone** (e.g., completing a major refactor)
- **When a project feels stuck** or has encountered significant friction
- **At regular intervals** (e.g., monthly) to maintain a cadence of reflection

---

## III. The 5-Step Retrospective Workflow

### Step 1: Initiate the Retrospective

State the intention to conduct a retrospective. Frame it as a positive and necessary part of the workflow. Accept: sprint number, version, date range, or informal period ("the last two weeks").

### Step 2: Gather Context

Pull relevant data from connected tools to ground the retrospective in evidence:

- **~~project tracker**: Completed tickets, velocity, burndown
- **~~chat**: Notable discussions, decisions, blockers raised
- **~~repository**: Merged PRs, deployment history, code changes

If no integrations are connected, ask the user to share context verbally.

### Step 3: Facilitate the Three Core Questions

Ask these **one at a time, conversationally** — never dump all three at once:

1. **What went well?** — Celebrate successes, name specific wins, acknowledge contributors. What should we amplify?
2. **What was hard?** — Identify pain points without blame. Focus on systemic issues, not individual failures. What were the sources of friction?
3. **What would we do differently?** — Concrete changes, not vague intentions. Each "differently" should be actionable.

### Step 4: Synthesize and Extract Learnings

Analyze the answers and synthesize into structured output:

- **Identify Patterns:** Look for recurring themes across answers
- **Distill Actionable Insights:** For each theme, identify a concrete action
- **Create Seeds for Memory Garden:** If a lesson is particularly profound or reusable, distill it into a seed (offer to `/plant` each via wisdom-garden plugin)

### Step 5: Close with Gratitude

End with a closing statement that:
- Acknowledges the work done
- Expresses gratitude for the practice of reflection
- Sets intention for the next cycle

---

## IV. Best Practices

- **Be Honest and Gentle:** The goal is learning, not blame
- **Focus on Process, Not People:** Analyze the workflow, not individuals
- **Be Specific and Concrete:** Avoid vague statements; use specific examples
- **End with Action:** Every retrospective should produce at least one concrete action item
- **Keep it Lightweight:** The process should feel energizing, not burdensome

---

## V. Quality Checklist

- [ ] Three core questions asked one at a time, conversationally
- [ ] Always ends with a gratitude-based closing
- [ ] Produces seeds for the wisdom-garden (cross-plugin reference)
- [ ] Output saved as dated file for longitudinal tracking
- [ ] Key Themes table includes evidence column
- [ ] At least one concrete action item produced

---

## VI. Example: HTML Kits Sprint Retro (March 2026)

**The Problem:** After building 3 HTML kit prototypes in parallel over one sprint (TresPies Health Dashboard, CWD Data Explorer, Vending Business Plan), the team needed to assess what worked, what didn't, and what to carry forward.

**The Process:**

1. **Initiated:** Framed as a harvest of the "HTML Kits Sprint" covering March 25-31, 2026.
2. **Context gathered:** Reviewed 3 shipped prototypes, git history, and build artifacts.
3. **Core questions:**
   - *What went well?* — Alpine.js + localStorage pattern proved reliable across all 3 kits. Agent delegation (parallel builds) saved significant time.
   - *What was hard?* — IndexedDB blocked on file:// protocol (discovered mid-sprint). HTMX requires a server, making it unsuitable for self-contained kits.
   - *What would we do differently?* — Validate browser API compatibility before choosing a tech stack. Use Alpine.js as the default, not a fallback.
4. **Synthesized:** Extracted 3 seeds — self-contained HTML kit pattern, tech constraint checklist, agent delegation protocol for parallel builds.
5. **Closed:** Acknowledged the sprint as a proof-of-concept success that established the foundation for HTMLCraft Studio.

**The Outcome:** 3 seeds planted in the wisdom garden. The "self-contained HTML kit" seed became the foundation for all subsequent kit development. The IndexedDB constraint discovery saved hours on future prototypes.

**Key Insight:** The most valuable retro output wasn't the action items — it was the seeds. Action items expire; seeds compound.

---

## VII. Common Pitfalls

### Pitfall 1: Blame Disguised as Feedback

**Problem:** "What was hard?" devolves into blaming individuals or tools rather than analyzing systemic issues.

**Solution:** Redirect to process: "What about the *process* made this hard?" Focus on what can be changed, not who caused problems.

### Pitfall 2: Vague Action Items

**Problem:** "We should communicate better" is not actionable. Vague action items are never implemented.

**Solution:** Every action item must answer: Who does what, by when? Example: "Add a tech compatibility check to the kit template before next sprint."

### Pitfall 3: Skipping the Gratitude Closing

**Problem:** Ending abruptly leaves participants feeling drained rather than energized.

**Solution:** Always close with gratitude. Even a single sentence acknowledging the work done transforms the retrospective from a chore into a ritual.

### Pitfall 4: Running a Retro Without Evidence

**Problem:** Retrospectives based purely on memory are biased toward recent events and strong emotions.

**Solution:** Always gather context (Step 2) before facilitating questions. Git logs, completed tickets, and chat history ground the conversation in facts.

### Pitfall 5: Producing Themes Without Seeds

**Problem:** Insights that stay in a retrospective document are rarely referenced again.

**Solution:** Extract the most reusable insights as seeds for the wisdom garden. Seeds are designed to be discovered and applied; retro documents are not.

---

## VIII. Related Skills

- **`seed-extraction`** — Extract reusable patterns from retro insights into persistent seeds
- **`memory-garden`** — Plant retro seeds into the wisdom garden for long-term cultivation
- **`compression-ritual`** — Compress the full retro conversation into a potent memory artifact
- **`research-synthesis`** — When retros across multiple sprints need cross-referencing for patterns
- **`status-writing`** — Feed retro action items into the project status document

---

## IX. Output Format

### Key Themes & Insights Table

| Theme | Insight | Evidence | Recommended Action |
| :--- | :--- | :--- | :--- |
| **[Theme 1]** | [What we learned] | [Specific examples] | [Concrete next step] |
| **[Theme 2]** | [What we learned] | [Specific examples] | [Concrete next step] |

### Seeds for Memory Garden

- **Seed:** [Seed Name] — *Why it matters:* [Explanation] — *Revisit trigger:* [When to remember this lesson]

### Closing Statement

[A brief statement summarizing key takeaways, expressing gratitude, and setting intention for the next cycle.]

---

## X. Retrospective Document Template

```markdown
# Retrospective: [Sprint/Release Name]

**Date:** [Date]
**Participants:** [Names]
**Context:** A reflection on [Sprint/Release Name], which focused on [brief description].

---

## 1. The Three Core Questions

### What Went Well?

- [Observation 1]
- [Observation 2]

### What Was Hard?

- [Friction Point 1]
- [Friction Point 2]

### What Would We Do Differently Next Time?

- [Actionable Suggestion 1]
- [Actionable Suggestion 2]

---

## 2. Key Themes & Insights

| Theme | Insight | Evidence | Recommended Action |
| :--- | :--- | :--- | :--- |
| **[Theme 1]** | [Analysis] | [Examples] | [Action] |

---

## 3. Seeds for Memory Garden

- **Seed:** [Name] — *Why it matters:* [Explanation] — *Revisit trigger:* [Trigger]

---

## 4. Closing

[Gratitude-based closing statement.]
```

---

**Last Updated:** 2026-04-06
**Status:** Active
