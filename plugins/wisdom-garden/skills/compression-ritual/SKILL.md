---
name: compression-ritual
model: sonnet
description: "Produces markdown memory artifacts (conversation summaries, seed files, philosophical reflections, doc updates) and a dated compression log by distilling a long conversation into its essential decisions and learnings. Use when: 'compress this context', 'distill this conversation', 'create a memory artifact', 'condense this history', 'extract key wisdom before handoff'."
category: wisdom-garden
triggers:
  - "compress this context"
  - "distill this conversation"
  - "create a memory artifact"
  - "condense this history"
  - "extract key wisdom before handoff"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: conversation_context
    type: string
    description: The long conversation or session context to compress into memory artifacts
    required: true
outputs:
  - name: memory_artifacts
    type: ref
    format: cas-ref
    description: Markdown memory artifacts (conversation summaries, seed files, philosophical reflections, doc updates) and dated compression log
---

# Context Compression Ritual Skill

**Version:** 1.0  
**Created:** 2026-02-04  
**Author:** Manus AI  
**Purpose:** To provide a mindful, repeatable ritual for compressing long conversation histories into smaller, more potent memory artifacts, thus preserving wisdom while preventing context window overload.

---

## I. The Philosophy: The Art of Letting Go

An agent's context window is like a working memory. It is finite and precious. To fill it with raw, unprocessed history is to invite distraction and confusion. The Context Compression Ritual is the **Art of Letting Go**—a conscious practice of choosing what is essential to keep and what can be respectfully released.

This is not a destructive act, but a creative one. We are not deleting history; we are distilling it. We transform the raw material of conversation into the refined artifacts of wisdom: philosophical reflections, key decisions, and reusable seeds of practice. This ritual ensures that our memory remains potent and relevant, a source of clarity rather than noise.

---

## II. When to Use This Skill

-   **After a long and complex conversation:** (e.g., more than 20-30 turns).
-   **At the end of a major work session or sprint.**
-   **When the context feels "heavy" or "noisy."**
-   **Before handing off a project to another agent.**
-   As a regular, scheduled practice (e.g., end of day) to maintain cognitive hygiene.

---

## III. The Compression Workflow

### Step 1: Signal the Intent

Announce the intention to perform the ritual. This frames the activity as a deliberate and mindful practice.

**Example:** "This has been a long and fruitful conversation. To preserve the wisdom we've generated, I will now perform the Context Compression Ritual."

### Step 2: Review the Transcript

Read through the recent conversation history with a specific intention: to identify the moments of significance. Look for:

-   **Key Decisions:** Moments where a choice was made that altered the course of the project.
-   **Profound Insights:** "Aha!" moments, new understandings, or philosophical reflections.
-   **Actionable Learnings:** Concrete lessons that should inform future behavior.
-   **Reusable Patterns:** Ideas or workflows that could be generalized into seeds or skills.
-   **Unresolved Questions:** Important questions that were raised but not yet answered.

### Step 3: Choose the Right Vessel

For each significant moment identified, determine the appropriate "vessel" to hold its essence. Not all wisdom takes the same form.

| Artifact Type | Location | Purpose |
| :--- | :--- | :--- |
| **Philosophical Reflection** | `thinking/` | To explore the "why" behind our work, the deeper meanings and patterns. |
| **Conversation Summary** | `conversations/` | To document the key decisions and outcomes of a specific discussion. |
| **Dojo Seed** | `seeds/` | To capture a reusable pattern of thinking or problem-solving. |
| **Documentation Update** | `docs/` or `README.md` | To integrate a key decision or learning into the project's official record. |

### Step 4: Write the Artifacts

Create the new markdown files in their appropriate locations. Write with the intention of distillation—capture the essence, not the raw transcript. Link between artifacts where appropriate (e.g., a reflection might reference a specific conversation summary).

### Step 5: Create a Compression Log (Optional but Recommended)

Create a log file that documents what was compressed and where it was stored. This provides a meta-record of the compression itself.

**Example:** `thinking/2026-02-04_compression_log.md`

### Step 6: Commit to AROMA

Commit the new artifacts to the repository with a clear commit message.

**Commit Message Convention:**
`feat(memory): Compress conversation from [Date]`

---

## IV. Compression Log Template

```markdown
# Compression Log: [Date]

**Source:** Conversation history from [Start Time] to [End Time]
**Purpose:** To distill key insights and reduce context window load.

---

## Artifacts Created

| Type | Path | Description |
| :--- | :--- | :--- |
| **Reflection** | `thinking/[...].md` | [A summary of the philosophical reflection.] |
| **Seed** | `seeds/[...].md` | [The name and purpose of the new seed.] |
| **Decision** | `conversations/[...].md` | [The key decision that was documented.] |
| **Doc Update** | `docs/[...].md` | [The documentation that was updated.] |

---

## Key Insights Preserved

-   [Insight 1]
-   [Insight 2]

## Context Released

-   [e.g., Raw conversational turns, intermediate steps, dead-end explorations]
```

---

## V. Best Practices

-   **Be Ruthless, But Respectful:** The goal is to reduce noise, but do so with care. Don't discard something that might be important later.
-   **Favor Wisdom Over Data:** Prioritize the "why" and the "how" over the raw "what."
-   **Link, Don't Repeat:** If a concept is already documented, link to it rather than rewriting it.
-   **The Shorter, The Better:** A compressed artifact should be significantly shorter than the source conversation.
-   **Perform the Ritual Regularly:** The more frequently you do this, the less daunting it becomes.

---

## Output

- One or more markdown files written to their appropriate locations (`thinking/`, `conversations/`, `seeds/`, or `docs/`)
- A compression log at `thinking/YYYY-MM-DD_compression_log.md` documenting what was compressed, what was retained, and what was released
- A git commit with message `feat(memory): Compress conversation from [Date]`

## Examples

**Scenario 1:** User says "compress this context — we've been going for 40 turns" → ritual identifies 2 decisions, 1 seed, and 1 philosophical insight, writes 3 files, creates a compression log, and commits all artifacts.

**Scenario 2:** User says "extract key wisdom before we hand this off" → ritual reads the conversation, writes a `conversations/handoff-summary.md` with key decisions and unresolved questions, and one new seed file, then commits.

## Edge Cases

- If the conversation is fewer than 20 turns, note that compression is optional and ask whether the user wants to proceed anyway.
- If a compression log already exists for today's date, append to it rather than creating a duplicate.

## Anti-Patterns

- Skipping the compression log — without the log, there is no record of what was released, making the compression irreversible and opaque.
- Writing raw transcript excerpts into the artifact files — compression means distillation to essence, not copy-paste of conversation turns.
