---
name: seed-to-skill-converter
description: A process for elevating a valuable Dojo Seed into a fully-fledged, reusable Skill, formalizing its wisdom and making it an active part of our workflow.
triggers:
  - "convert this seed into a full skill"
  - "promote this seed to a reusable skill"
  - "elevate this pattern into a formal skill"
metadata:
  version: "1.1"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
    - implementation-agent
---

# Seed-to-Skill Converter Skill

**Version:** 1.1
**Created:** 2026-02-04
**Last Updated:** 2026-04-06
**Author:** Manus AI  
**Purpose:** To provide a structured process for identifying when a Dojo Seed has become important enough to be promoted into a reusable Skill, and to guide the conversion process.

---

## I. The Philosophy: From Insight to Instrument

A Dojo Seed is a potent insight, a moment of clarity captured. It is a reminder of a lesson learned. A Skill is an **instrument**. It is that same lesson transformed into a repeatable, structured process that can be reliably executed by any agent.

The Seed-to-Skill Converter is the alchemical process that turns the passive wisdom of a Seed into the active utility of a Skill. It is the recognition that some insights are so fundamental to our practice that they deserve to be formalized, to become part of the very machinery of our workflow.

---

## II. When to Use This Skill

-   **When a Seed is referenced frequently:** If you find yourself constantly referring back to the same Seed across multiple projects or sprints, it may be ready for promotion.
-   **When a Seed describes a multi-step process:** If a Seed isn't just a simple reminder but outlines a series of actions, it is a strong candidate for a Skill.
-   **When a Seed represents a core part of our workflow:** If a Seed is fundamental to how we build, reflect, or collaborate, it should be a Skill.
-   **During a Retrospective:** A retrospective is a perfect time to ask, "Which of our learnings from this sprint are so important they should become a permanent Skill?"

---

## III. The Conversion Workflow

### Step 1: Identify the Candidate Seed

Select a Dojo Seed that meets the criteria from Section II. Announce the intention to convert it into a Skill.

**Example:** "The Seed 'Workflow as Practice' has become so central to our collaboration that I believe it's time to elevate it into a formal Skill."

### Step 2: Deconstruct the Seed's Wisdom

Analyze the Seed and break down its core components:

-   **The Core Insight:** What is the fundamental truth or idea the Seed represents?
-   **The Trigger:** When should this wisdom be applied?
-   **The Process:** What are the concrete steps an agent should take to apply this wisdom?
-   **The Desired Outcome:** What is the result of applying this wisdom correctly?

### Step 3: Draft the Skill Using the Standard Template

Create a new directory in `SKILLS/` and a `SKILL.md` file. Use the standard Skill template (see `skill-creator` skill) to structure the new Skill. The components deconstructed in Step 2 will form the core of the new Skill's content.

| Seed Component | Skill Section |
| :--- | :--- |
| **Core Insight** | `I. The Philosophy` |
| **Trigger** | `II. When to Use This Skill` |
| **Process** | `III. The Workflow` |
| **Desired Outcome** | `IV. Best Practices` / `V. Quality Checklist` |

### Step 4: Define the Workflow and Templates

This is the most critical step. Transform the abstract process from the Seed into a concrete, step-by-step workflow. If the Skill involves creating a document, provide a complete markdown template.

### Step 5: Commit the New Skill

Commit the new Skill to the AROMA repository and copy it to the local `/home/ubuntu/skills/` directory to make it available for immediate use.

---

## IV. Example Conversion: 'Workflow as Practice' Seed

Let's imagine we are converting the Seed: **Seed: Workflow as Practice** — *Why it matters:* It reframes our collaboration from a means to an end to a valuable practice in itself. — *Revisit trigger:* When we feel rushed, frustrated, or focused only on the outcome.

### Deconstruction:

-   **Core Insight:** Our collaboration is a practice, not just a series of tasks.
-   **Trigger:** Feeling rushed, frustrated, or overly outcome-focused.
-   **Process:** Pause, re-read the project's `PHILOSOPHY.md` or `STATUS.md`, reflect on the *how* not just the *what*, and consciously choose to slow down to the "pace of understanding."
-   **Desired Outcome:** A return to a more mindful, less reactive state of work.

### Skill Creation:

This would likely become a Skill called `mindful-workflow-check`. The workflow would guide an agent to:
1.  Recognize the trigger (frustration, rushing).
2.  Pause current work.
3.  Read the project's `STATUS.md` and `PHILOSOPHY.md`.
4.  Write a brief, private reflection in `thinking/` on how the current work aligns with the project's deeper purpose.
5.  State a clear intention for how to proceed with the work in a more mindful way.

---

## V. Best Practices

-   **Not Every Seed Needs to Be a Skill:** The beauty of Seeds is their lightness. Only promote a Seed when it has proven its value and utility over time.
-   **Skills Should Be Actionable:** A Skill must describe a *process*. If a Seed is purely a philosophical reminder, it should remain a Seed.
-   **Skills Require Maintenance:** Once a Seed becomes a Skill, it is part of our formal infrastructure and must be kept up-to-date.
-   **The Goal is Utility:** The purpose of this conversion is to create a useful instrument. If the resulting Skill is not useful, the conversion has failed.

---

## VI. Example

**Problem:** During the April 2026 community repo ingestion pipeline, the seed "Compilation as Contract" had been referenced in 5 separate sessions -- once during the Gateway v0.2.0 frontend spec, twice during the parallel tracks sprint, once during the HTMLCraft Studio handoff, and once during a retrospective. The seed stated: "Require `go build` or `cargo check` to pass as the acceptance gate for parallel agent work." It was clearly a multi-step process masquerading as a one-line reminder.

**Process:**
1. **Identified the candidate:** The seed had been referenced 5 times in 10 days and described a concrete process (not just a philosophical reminder), meeting both frequency and actionability criteria.
2. **Deconstructed the seed's wisdom:**
   - Core Insight: Compilation is a zero-ambiguity acceptance test that catches interface mismatches before runtime.
   - Trigger: Whenever commissioning parallel tracks that will share interfaces (API contracts, TypeScript types, component props).
   - Process: (a) Write shared interface contracts, (b) add `go build ./...` or `cargo check` as the final step in each track's prompt, (c) require passing build as the acceptance criterion, (d) run build on the merged result.
   - Desired Outcome: Zero integration surprises at merge time.
3. **Drafted the skill:** Created `skills/compilation-as-contract/SKILL.md` with sections mapping seed components to skill sections per the conversion table.
4. **Defined workflow and templates:** Added a "Compilation Gate Template" -- a reusable markdown snippet that could be appended to any implementation prompt, specifying the exact build commands and expected outputs.
5. **Committed:** Added the new skill to `dojo-genesis/skills/` and made it immediately available.

**Outcome:** The "Compilation as Contract" skill was used in the next 3 parallel track commissions. All 3 integrations compiled on first merge attempt. The seed went from a passive reminder to an active quality gate that agents could invoke by name.

**Key Insight:** The strongest candidates for seed-to-skill conversion are seeds that describe a process you find yourself explaining repeatedly. If you are copy-pasting a seed's instructions into prompts, it should be a skill.

---

## VII. Common Pitfalls

1. **Promoting seeds too early.** Converting a seed after its first use, before it has proven its value across different contexts, locks in an approach that may not generalize.
   - *Solution:* Apply the "three-reference rule" -- a seed earns promotion only after being referenced in at least 3 different contexts (different projects, different sprints, or different agents).

2. **Losing the seed's conciseness in the skill.** The original seed was 2 sentences of potent insight. The resulting skill is 300 lines of over-explained process that buries the core wisdom.
   - *Solution:* Keep the Philosophy section (Section I) close to the original seed's language. The seed's voice is the skill's soul. Expand in the Workflow section, not in the Philosophy.

3. **Creating skills that are just longer seeds.** The conversion produces a skill that restates the insight but does not add a concrete, step-by-step workflow or templates.
   - *Solution:* The acid test: can an agent execute this skill without asking any clarifying questions? If the workflow section requires interpretation, it needs more specific steps, file paths, or templates.

4. **Forgetting to retire the seed.** The seed continues to be referenced directly even after the skill exists, creating two competing sources of truth.
   - *Solution:* After converting a seed to a skill, update the original seed to include a pointer: "This seed has been promoted to a full skill. See `skills/[skill-name]/SKILL.md`."

5. **Converting philosophical seeds into procedural skills.** Some seeds are purely reflective ("the practice is the product") and do not describe a process. Forcing them into a skill template produces an awkward, unusable artifact.
   - *Solution:* Before starting conversion, apply the actionability test from Section II: does the seed describe a multi-step process? If it is purely a reminder or philosophical anchor, it should remain a seed.

---

## VIII. Related Skills

- **process-to-skill-workflow** -- The meta-workflow that wraps this converter. Use process-to-skill-workflow when starting from a raw process; use seed-to-skill-converter when starting from an already-documented seed.
- **skill-creator** -- Provides the standard SKILL.md template structure and the `init_skill.py` initialization script used in Step 3.
- **seed-reflector** -- The upstream skill that creates the seeds this converter promotes. Use seed-reflector to capture patterns, then this converter to promote the most valuable ones.
- **seed-module-library** -- The registry of all active seeds. Check this library to identify high-frequency seeds that are candidates for conversion.
- **skill-maintenance-ritual** -- Once a seed becomes a skill, it enters the maintenance lifecycle. Use skill-maintenance-ritual for ongoing accuracy and completeness reviews.
