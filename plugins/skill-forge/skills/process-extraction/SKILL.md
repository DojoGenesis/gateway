---
name: process-extraction
description: >
  Transform a completed, successful workflow into a reusable skill by first
  documenting it as a process example, then extracting the generalizable pattern.
  Use when you've completed a complex multi-step task that should be standardized,
  when you find yourself repeating the same sequence of actions, during
  retrospectives, or when capturing institutional knowledge from one-off processes.
triggers:
  - "extract a reusable process from what we just did"
  - "turn this workflow into a skill"
  - "document this repeatable pattern"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
    - implementation-agent
---

# Process Extraction

## I. Philosophy: From One-Off Process to Institutional Knowledge

Every time someone completes a complex task successfully, they generate implicit knowledge -- a sequence of decisions, actions, and course corrections that led to a good outcome. This implicit knowledge is valuable, but it's also fragile: it lives in a single conversation, a single mind, a single moment. Without capture, it evaporates.

Process extraction is the discipline of making implicit knowledge explicit. The key insight is that **you must document the specific process before abstracting the general pattern.** The specific is the anchor. Without it, generalization drifts into theory.

The process example is the proof. The skill is the tool. You cannot build a reliable tool without first understanding -- in concrete detail -- what the tool must do.

## II. When to Use This Skill

- **After completing a complex, multi-step task** that produced a good outcome and is likely to be repeated
- **When you find yourself repeating the same sequence of actions** across different projects or contexts
- **During a retrospective** when a successful workflow is identified for standardization
- **When onboarding someone** to a complex process that needs to be reproducible
- **When a conversation transcript contains a valuable workflow** that should be preserved
- **When institutional knowledge exists only in one person's head** and needs to be formalized
- **After a debugging session** that revealed a systematic diagnostic approach

**When NOT to use:** If the process was unsuccessful, capture it as a retrospective lesson, not a skill. If the process is trivial (1-2 steps), it doesn't warrant a skill.

## III. The Extraction Workflow

### Step 1: Document the Specific Process

**Goal:** Create a detailed record of what actually happened, not what should have happened.

**Actions:**
1. Identify the source: conversation transcript, memory, or user description
2. Create a process example document with these sections:
   - **Context:** When and why the process was used
   - **Input:** What the process started with
   - **Output:** What the process produced
   - **Steps Taken:** Every significant action, in order, with goals and outcomes
   - **Key Decisions:** What was decided, why, and what alternatives were considered
   - **What Worked:** Success factors and why they mattered
   - **What Was Hard:** Challenges encountered and how they were resolved
   - **Reusable Pattern:** The generalizable principle (the kernel of the future skill)

3. Save as `docs/examples/[YYYY-MM-DD]_[process-name].md`

**Output:** Complete process example document.

**Key Insight:** Be specific. "Searched for relevant files" is too vague. "Used grep to find all files containing the error message, then read the top 3 results to identify the root cause" is specific enough to reproduce.

**Decision Point:** If the user cannot describe the process in sufficient detail, ask targeted questions: "What did you do first? What made you decide to take that approach? What surprised you?"

### Step 2: Extract the Generalizable Pattern

**Goal:** Transform the specific process into a reusable skill.

**Actions:**
1. Read the process example with fresh eyes
2. Map process components to skill sections:

   | Process Section | Skill Section |
   |---|---|
   | Context + Reusable Pattern | I. Philosophy |
   | Steps Taken (generalized) | III. Workflow |
   | What Worked | IV. Best Practices |
   | What Was Hard | VI. Common Pitfalls |
   | Key Decisions | Decision points in III. Workflow |

3. Generalize: replace specific file names with placeholders, specific tools with generic descriptions, specific contexts with categories
4. Preserve the reasoning: keep the *why* behind each step, even as you generalize the *what*

**Output:** Skill draft with sections mapped from the process example.

**Key Insight:** The hardest part of generalization is knowing what to keep specific. Keep the reasoning (why), generalize the implementation (what). If a step's value comes from a specific technique, keep the technique.

### Step 3: Refine the Skill

**Goal:** Ensure the skill faithfully represents the process.

**Actions:**
1. Compare the skill back to the process example, section by section
2. Check: Does the skill capture everything that made the process work?
3. Check: Is anything lost in generalization?
4. Check: Would someone following the skill make the same key decisions?
5. Check: Are the hard parts (from "What Was Hard") addressed in the workflow or pitfalls?
6. If gaps exist, update the skill

**Output:** Refined skill that faithfully represents the process.

**Key Insight:** The process example is the truth. If the skill contradicts the example, the skill is wrong. If the skill omits something from the example, the skill is incomplete.

### Step 4: Validate

**Goal:** Confirm the skill is ready for independent use.

**Actions:**
1. Ask the validation question: "If a new agent encountered [the original situation], would this skill guide them to the same quality outcome?"
2. If "no" or "maybe": identify what's missing, return to Step 3
3. If "yes": the skill is ready
4. Present both the process example and the skill to the user

**Output:** Validated skill and process example, both saved to appropriate locations.

**Key Insight:** The validation question is non-negotiable. It's the difference between a skill that looks good and a skill that works.

## IV. Best Practices

### Start with the Specific, Always

Never skip the process example. Even if the pattern seems obvious, documenting the specific process reveals details that abstraction hides. The example is the foundation.

### Preserve the Reasoning

When generalizing, it's tempting to strip the "why" and keep only the "what." Resist this. The reasoning is what makes a skill adaptable to new situations. Steps without reasoning are brittle.

### Capture Decisions, Not Just Actions

The most valuable parts of a process are often the decision points: "I chose X over Y because Z." These decisions become the branching logic in the skill's workflow.

### Document What Was Hard

Challenges and failures are the most valuable input for a skill's Common Pitfalls section. If someone struggled with a step, future users will too.

### Generalize Gradually

Don't try to create a universal skill from one example. Generalize only as far as the evidence supports. One example supports a specific skill. Three examples support a broader pattern.

## V. Quality Checklist

- [ ] Process example document exists with all sections filled
- [ ] Steps in the process example are specific enough to reproduce
- [ ] Key decisions include alternatives considered and rationale
- [ ] "What Was Hard" section has at least one entry
- [ ] Skill was derived from the process example, not invented independently
- [ ] Every workflow step in the skill traces back to a step in the process example
- [ ] Key decisions from the process appear as decision points in the skill workflow
- [ ] Challenges appear in the skill's Common Pitfalls section
- [ ] The validation question was asked and answered "yes"
- [ ] Both process example and skill are saved to appropriate locations

## VI. Common Pitfalls

### Skipping the Process Example

**Problem:** Jumping straight to skill creation without documenting the specific process. The resulting skill is theoretical, not grounded in reality.

**Solution:** Always create the process example first. Even if it feels redundant, the act of documenting reveals details you'd otherwise forget.

### Over-Generalizing from One Example

**Problem:** Creating a universal skill from a single process instance. The skill makes assumptions that only hold in the original context.

**Solution:** Generalize conservatively. Flag assumptions that might not hold in other contexts. Note in the skill: "Based on [N] observed instances. May need refinement after further application."

### Losing the "Why" in Generalization

**Problem:** The skill describes what to do but not why. When users encounter a situation that doesn't match exactly, they can't adapt.

**Solution:** For every workflow step, include the goal (why) alongside the actions (what). The goal is portable; the actions may need to change.

### Ignoring What Was Hard

**Problem:** The process example glosses over challenges, so the skill has no pitfalls section. Future users hit the same walls with no guidance.

**Solution:** Explicitly ask: "What was the hardest part? Where did you almost go wrong? What would you tell someone to watch out for?"

## VII. Example: Extracting a Deployment Process into a Skill

**Context:** After running a complex zero-downtime database migration, a team wanted to capture the process for future migrations.

**Process Example Highlights:**
- 12 steps covering pre-migration checks, backup, staged rollout, monitoring, and rollback criteria
- Key decision: chose blue-green deployment over rolling update because the schema change was breaking
- What was hard: coordinating the cutover window with dependent services

**Resulting Skill:** `execute-zero-downtime-migration`
- Philosophy: explains why zero-downtime matters and the cost of getting it wrong
- Workflow: 8 generalized steps (consolidated from the 12 specific steps)
- Decision point at Step 3: "If schema change is breaking, use blue-green. If additive, use rolling update."
- Pitfall: "Forgetting to notify dependent services" (from "What Was Hard")

The skill was validated against the original process and two subsequent migrations.
