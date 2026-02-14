# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: process-extraction
description: Extract repeatable workflows from conversation transcripts and codify them as skills
triggers:
  - "extract workflow from conversation"
  - "codify process as skill"
  - "turn transcript into skill"
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

# Process-to-Skill Workflow

**Version:** 1.0
**Created:** 2026-02-07
**Author:** Manus AI
**Purpose:** To provide a structured, repeatable process for identifying a valuable workflow, documenting it with concrete examples, and transforming it into a reusable skill.

---

## I. The Philosophy: From Process to Practice

We often develop valuable, multi-step processes for complex tasks. These processes are a form of implicit knowledge, a sequence of actions that leads to a high-quality outcome. This skill is about making that implicit knowledge explicit, transforming a one-off process into a repeatable practice.

By documenting a valuable workflow and then using the `/seed-to-skill-converter` to formalize it, we build a library of institutional knowledge that can be shared, improved, and reliably executed by any agent.

---

## II. When to Use This Skill

-   **After completing a complex, multi-step task** that you believe will be repeated in the future.
-   **When you find yourself manually repeating the same sequence of actions** across different projects.
-   **During a retrospective**, when you identify a successful workflow that should be standardized.
-   **When you want to onboard a new agent** to a complex process.

---

## III. The Workflow

This is a 4-step workflow for transforming a process into a skill.

### Step 1: Identify and Document the Process

**Goal:** Create a detailed record of the valuable workflow with concrete examples.

1.  **Identify a Candidate Process:** Select a recently completed workflow that was successful and is likely to be repeated.
2.  **Use the Process Example Template:** Create a new markdown file and use the template at `references/process_example_template.md` to document the process.
3.  **Be Detailed:** For each step, document the goal, actions taken, tools used, inputs, and outputs.
4.  **Extract Key Insights:** Document the key learnings, non-obvious steps, and reusable patterns from the process.

### Step 2: Convert the Process to a Skill

**Goal:** Use the `/seed-to-skill-converter` to transform the documented process into a SKILL.md file.

1.  **Invoke the Converter:** Use the command `/seed-to-skill-converter` on the process example document you just created.
2.  **Deconstruct the Process:** The converter will guide you through deconstructing the process into its core components (insight, trigger, process, outcome).
3.  **Draft the Skill:** The converter will then help you draft a `SKILL.md` file using the standard template.

### Step 3: Refine and Enhance the Skill

**Goal:** Improve the generated `SKILL.md` file to be a production-ready skill.

1.  **Add a Quality Checklist:** Include a checklist of yes/no questions to ensure the skill is used correctly.
2.  **Add Best Practices:** Document any non-obvious best practices or pitfalls to avoid.
3.  **Create Bundled Resources:** If the skill requires any scripts, templates, or reference files, create them in the skill's directory.

### Step 4: Validate and Deliver the Skill

**Goal:** Ensure the skill is complete, correct, and ready for use.

1.  **Validate Structure:** Check that the SKILL.md has all required sections: Philosophy, When to Use, Workflow, Best Practices, Quality Checklist. Grade against the skill-audit rubric.
2.  **Test the Skill:** Use the skill in a real scenario to verify the workflow is complete and the instructions are clear.
3.  **Place the Skill:** Add the skill directory to the appropriate plugin in the repository.

---

## IV. Best Practices

-   **Focus on the "Why":** When documenting the process, don't just list the steps. Explain *why* each step is important.
-   **Generalize the Pattern:** The goal is to create a reusable skill, so abstract the specific details of your example into a general pattern.
-   **The Template is Your Friend:** The `process_example_template.md` is designed to capture all the information needed for a successful conversion.
-   **Iterate:** The first version of a skill is rarely perfect. Be prepared to iterate on it after using it in a real-world scenario.

---

## V. Quality Checklist

Before delivering the skill, ensure you can answer "yes" to all of the following questions:

-   [ ] Have you documented the process with concrete examples using the provided template?
-   [ ] Have you used the `/seed-to-skill-converter` to generate the initial `SKILL.md`?
-   [ ] Have you added a quality checklist and best practices to the skill?
-   [ ] Have you created any necessary bundled resources (scripts, templates, references)?
-   [ ] Have you validated the skill structure against the skill-audit rubric?
-   [ ] Have you placed the skill in the appropriate plugin directory?

---

## VI. Common Pitfalls

### Documenting Too Specifically

**Problem:** Recording every detail of the specific instance instead of the generalizable pattern. The resulting skill only works for one exact scenario.

**Solution:** After documenting the concrete example, do a second pass where you replace specific names, paths, and values with placeholders and describe the *category* of thing, not the instance.

### Skipping the "Why"

**Problem:** A process document that says "do X, then do Y, then do Z" without explaining why each step matters. The resulting skill is fragile — any deviation breaks it because the user can't adapt.

**Solution:** For every step, add one sentence: "This matters because..." If you can't explain why, the step may be unnecessary.

### Premature Formalization

**Problem:** Turning a process into a skill after doing it once. One instance isn't enough to know what's essential vs. accidental. The skill encodes noise alongside signal.

**Solution:** Wait until you've done the process at least twice — ideally three times — before extracting it. The second and third times reveal which steps are truly reusable.

---

## VII. Example

**Context:** After the third time running a "merge and reconcile plugin directories" workflow, the team recognized a repeatable pattern: inventory both sides, map naming differences, do renames, copy additive content, update cross-references, verify.

**Step 1 output:** A detailed process example documenting the specific merge (which directories, which files, which renames), with key insights like "always rename longest strings first to avoid partial matches" and "verify zero stale references after replacement."

**Step 2 output:** A generalized "directory-reconciliation" SKILL.md with a 5-step workflow: Inventory → Map → Transform → Update References → Verify.

**Step 3 refinements:** Added a pitfall about filesystem permission restrictions preventing deletions, and a quality checklist item about checking JSON manifests alongside markdown files.

---

## VIII. Related Skills

- **skill-creation** — For building new skills from scratch (this skill starts from a completed process)
- **seed-extraction** — For capturing smaller patterns as seeds before they're ready to be full skills
- **seed-to-skill-converter** — The tool used in Step 2 to transform documented processes into SKILL.md format
- **retrospective** — Often the trigger for identifying processes worth extracting
