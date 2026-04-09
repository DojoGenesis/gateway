---
name: seed-to-skill-converter
model: sonnet
description: Produces a new SKILL.md file by converting a proven Dojo Seed into a fully-structured skill with philosophy, workflow, templates, and quality checklist. Use when: "promote this seed to a skill", "convert this seed into a skill", "make this seed into a skill", "formalize this pattern", "turn this seed active".
category: wisdom-garden

inputs:
  - name: seed_path
    type: string
    description: Path to the proven Dojo Seed file to promote into a skill
    required: true
outputs:
  - name: skill_file
    type: ref
    format: cas-ref
    description: New SKILL.md with philosophy, workflow, templates, and quality checklist converted from the seed
---

# Seed-to-Skill Converter Skill

**Version:** 1.0  
**Created:** 2026-02-04  
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

Create a new directory in `SKILLS/` and a `SKILL.md` file. Use the standard Skill template (see `skill-creation` skill) to structure the new Skill. The components deconstructed in Step 2 will form the core of the new Skill's content.

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

## Output

- A new directory at `SKILLS/[skill-name]/` containing `SKILL.md` with all 5 standard sections (Philosophy, When to Use, Workflow, Best Practices, Quality Checklist)
- The skill committed to the AROMA repository and copied to `/home/ubuntu/skills/` for immediate availability
- The original seed file updated with a note pointing to the promoted skill

## Examples

**Scenario 1:** User says "promote the 'Dry-Run as Founding Gate' seed to a skill" → deconstructs the seed's trigger (new scripts), process (run with --dry-run first), and outcome (caught bash crashes before file corruption); writes a `dry-run-gate/SKILL.md` with a 4-step workflow and checklist.

**Scenario 2:** User says "formalize this pattern — we always read before editing, no exceptions" → identifies the pattern as a candidate, checks that it describes a multi-step process (check file exists → read → verify understanding → edit), and produces `read-before-edit/SKILL.md`.

## Edge Cases

- If the seed is purely a philosophical reminder with no concrete steps, explain that it should remain a seed — skills must describe an actionable process, not just a mindset.
- If a skill already exists in `SKILLS/` that covers the same ground as the seed, compare them and propose merging rather than creating a duplicate skill.

## Anti-Patterns

- Promoting every seed to a skill — most seeds are appropriately lightweight as seeds. Only promote when the pattern has a clear multi-step workflow and is referenced frequently.
- Writing the skill without verifiable workflow steps — a skill that says "apply good judgment" is not actionable. Every step must be concrete enough to execute without ambiguity.
