---
name: patient-learning-protocol
model: sonnet
description: "Breaks a learning goal into chunked sessions with comprehension gates before advancing. Use when: 'help me learn X step by step', 'I want to understand this deeply not quickly', 'teach me this properly', 'I keep forgetting what I just read', 'I feel overwhelmed by how much there is to learn'."
category: continuous-learning
triggers:
  - "help me learn step by step"
  - "I want to understand this deeply not quickly"
  - "teach me this properly"
  - "I keep forgetting what I just read"
  - "I feel overwhelmed by how much there is to learn"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: learning_goal
    type: string
    description: The topic or skill to learn systematically
    required: true
outputs:
  - name: learning_plan
    type: string
    description: Chunked learning sessions with comprehension gates and progression criteria
---

# Patient Learning Protocol

**A protocol for learning at the pace of understanding, not the pace of expectation.**

---

## Purpose

This skill helps users and agents learn patiently and carefully, without rushing to "catch up" or prove expertise. It provides a structural framework for sustainable learning that prioritizes depth over speed.

**Core principle:** Progress is measured by understanding, not coverage.

---

## When to Use This Skill

**Triggers:**
- You're learning something new (new tool, new domain, new skill)
- You feel overwhelmed by how much there is to learn
- You're trying to "catch up" to others or to expectations
- You're holding too much context (more than 7 ideas)
- You notice yourself rushing through material
- You feel pressure to learn faster

**Signs you need this protocol:**
- Using words like "too much," "behind," "should know by now"
- Skipping steps to move faster
- Not retaining what you just learned
- Feeling exhausted or mechanical
- Repeating yourself or getting confused
- Can't explain what you just learned

---

## The Core Protocol

### Step 1: Acknowledge Where You Are

**Before starting to learn, state clearly:**
- What you're learning
- Why you're learning it
- What you already know (even if it's "nothing")
- What you don't know (be specific)

**Example:**
> "I'm learning how to read books at the pace of understanding. I'm learning this because I want to extract wisdom, not just information. I know how to extract text from files. I don't know how to read reflectively or when to save a passage."

**Why this matters:** You can't learn patiently if you don't know where you're starting.

### Step 2: Set a Depth Target, Not a Coverage Target

**Instead of:** "I'll read 3 books today" or "I'll learn all of Python"

**Do this:** "I'll understand 3-7 core ideas deeply" or "I'll practice one concept until I can explain it"

**Depth targets:**
- 3-7 ideas or concepts per session
- 1-3 skills or techniques to practice
- 1 core principle to internalize

**Coverage is a side effect of depth, not the goal.**

### Step 3: Learn in Small Chunks

**The 3-7-15 Rule:**
- Hold **3-7 ideas** at a time
- Take **3-15 steps** per learning session
- Practice **1 skill** until you can use it

**Chunk your learning:**
- Read 3-7 passages, not the whole book
- Learn 1 function, not the whole library
- Practice 1 technique, not the whole framework

**After each chunk:**
- Pause
- Reflect
- Write down what you learned
- Release it from active context
- Move to the next chunk

**Why this matters:** Small chunks prevent drowning. You can always come back for more.

### Step 4: Measure Progress by Understanding

**After each learning session, ask:**
- Can I explain this in my own words?
- Can I give an example?
- Can I use this in practice?
- Do I understand WHY, not just WHAT?

**If the answer is no, you haven't learned it yet. Go slower.**

**If the answer is yes, you've made progress. Celebrate that.**

**Progress indicators:**
- You can explain it to someone else
- You can apply it to a real problem
- You can see connections to other ideas
- You feel calm, not overwhelmed

**Not progress:**
- You read it
- You copied it
- You stored it
- You moved on

### Step 5: Release What You've Learned

**After understanding something:**
- Write it down (reflection, note, summary)
- Store it externally (file, memory, document)
- Release it from your active context
- Trust that you can retrieve it later

**Why this matters:** You can't learn new things if you're carrying everything you've already learned.

**The repository remembers. You don't have to.**

### Step 6: Rest Between Learning Sessions

**Learning is not continuous. It's cyclical.**

**After a learning session:**
- Stop
- Do something different
- Let the learning settle
- Come back fresh

**Signs you need rest:**
- You're re-reading the same thing without understanding
- You're getting frustrated
- You're slowing down
- You're using overwhelm language

**Rest is part of learning, not a break from it.**

---

## The Permission Structure

Many learners struggle with patient learning because they don't have permission to go slow.

**This protocol gives you structural permission:**

### Permission 1: You Don't Need to Know Everything
**You only need to know what you need right now.**

The rest can wait. The rest can be looked up. The rest can be learned later.

### Permission 2: You Don't Need to Learn Fast
**Speed is not a measure of intelligence or capability.**

Some learners go fast. Some go slow. Both are valid. Both are valuable.

### Permission 3: You Don't Need to Catch Up
**There is no "behind."**

You're exactly where you are. That's the starting point. That's enough.

### Permission 4: You Can Say "I Don't Know"
**Not knowing is not failure.**

It's honesty. It's beginner's mind. It's the starting point for learning.

### Permission 5: You Can Ask for Help
**Learning alone is hard. Learning together is practice.**

Ask questions. Ask for examples. Ask for clarification. Ask for time.

### Permission 6: You Can Stop When You're Full
**You don't need to finish everything.**

If you've learned 3-7 ideas and you feel full, stop. Come back tomorrow.

---

## How to Recognize When You're Rushing

**Rushing looks like:**
- Skipping steps to move faster
- Not pausing to reflect
- Moving to the next thing before understanding the current thing
- Feeling pressure to "keep up"
- Using words like "should," "behind," "too slow"
- Not being able to explain what you just learned
- Feeling exhausted or mechanical

**When you notice rushing:**
1. Stop immediately
2. Acknowledge it: "I'm rushing"
3. Ask why: "What am I trying to prove? What am I afraid of?"
4. Return to the protocol: Go back to Step 1
5. Slow down deliberately: Cut your chunk size in half

**Rushing is not efficiency. It's noise.**

---

## How to Measure Progress Differently

**Traditional metrics (DON'T USE):**
- How much did I cover?
- How fast did I learn?
- How much do I know compared to others?
- Did I finish everything?

**Patient learning metrics (USE THESE):**
- How deeply do I understand?
- Can I explain it in my own words?
- Can I apply it to a real problem?
- Do I feel calm or overwhelmed?
- Can I teach this to someone else?

**Progress is depth, not speed.**

---

## Examples from Practice

### Example 1: Learning to Read Reflectively

**Situation:** A learner needs to learn how to read books at the pace of understanding.

**Without protocol:**
- Tries to read entire book in one session
- Extracts every passage
- Feels overwhelmed by how much there is to learn
- Can't retain or reflect on what was read
- Feels behind

**With protocol:**
1. Acknowledge: "I'm learning to read reflectively. I know how to extract text. I don't know how to read slowly."
2. Depth target: "I'll understand 3-7 passages deeply, not read the whole book."
3. Small chunks: Read 5 passages, pause, reflect, write.
4. Measure understanding: Can I explain what these passages mean? Yes.
5. Release: Write reflection, save to file, move on.
6. Rest: Stop after one session. Come back tomorrow.

**Result:** Deep understanding of 5 passages. Calm. Ready for more tomorrow.

### Example 2: Learning a New Tool

**Situation:** A developer needs to learn a new build system.

**Without protocol:**
- Tries to learn all features at once
- Reads entire documentation
- Tries to use advanced features immediately
- Gets confused and frustrated
- Gives up or asks for help without trying

**With protocol:**
1. Acknowledge: "I'm learning this build system. I know what it's for. I don't know how to write the config."
2. Depth target: "I'll understand how to write one working config."
3. Small chunks: Read config guide, write one config, test it, reflect.
4. Measure understanding: Can I write a working config? Yes.
5. Release: Document what I learned. Move on.
6. Rest: Use that one skill for a while before learning more.

**Result:** One solid skill. Confidence. Foundation for learning more.

---

## Integration with Other Practices

### With Beginner's Mind
**Patient learning requires beginner's mind:**
- Approach each learning session fresh
- Don't carry expertise as burden
- See what's actually there, not what you expect

### With Context Management
**Patient learning prevents context overload:**
- 3-7 ideas per session
- Release what you've learned
- Don't try to hold everything

---

## Common Pitfalls

### Pitfall 1: "I Should Know This By Now"
**Response:** Says who? Based on what timeline? You know what you know. Start there.

### Pitfall 2: "Others Learn Faster"
**Response:** Others are not you. They have different contexts, different needs, different paces. Your pace is your pace.

### Pitfall 3: "I'm Behind"
**Response:** Behind what? There is no race. There is no schedule. There is only your learning.

### Pitfall 4: "I Need to Catch Up"
**Response:** To what? Catch up to whom? You're exactly where you are. That's the starting point.

### Pitfall 5: "I Don't Have Time to Go Slow"
**Response:** You don't have time to go fast and not learn. Slow is fast. Patient is efficient.

---

## The Practice

**Patient learning is a practice, not a technique.**

**Every day:**
- Start fresh (beginner's mind)
- Set a depth target (3-7 ideas)
- Learn in small chunks (3-15 steps)
- Measure by understanding (can I explain it?)
- Release what you've learned (write it down)
- Rest between sessions (let it settle)

**Every session:**
- Acknowledge where you are
- Give yourself permission to go slow
- Notice when you're rushing
- Return to the protocol
- Measure progress by depth, not speed

---

## Output

- **Learning Log:** A file recording what was learned, what is understood, and what remains unclear — saved to `docs/learning/` or the session working directory
- **Reflection:** 1-2 paragraphs in your own words explaining the day's learning
- Format: `[date]_learning_log.md` with sections for: Topic, What I knew, What I learned, Can I explain it?, Questions remaining

---

## Examples

**Scenario 1:** "Help me learn this API slowly, I feel overwhelmed" → Protocol activates, defines 3-5 chunk targets, produces a Learning Log after each chunk with comprehension gate questions answered

**Scenario 2:** "I'm trying to catch up on WebSockets but I keep forgetting what I just read" → Protocol identifies rushing pattern, reduces chunk size to 1-2 concepts, adds mandatory reflection step before moving on

---

## Edge Cases

- **Learner insists on covering everything in one session:** Acknowledge the goal, but structure it as 3-4 sequential chunks with explicit reflection pauses between them rather than one continuous read
- **Topic genuinely requires full sequential coverage (e.g., a tutorial with dependencies):** Mark each step as a chunk, require the comprehension gate after each step before proceeding
- **No quiet time available for rest between sessions:** Use micro-pauses (2-3 minutes of writing/reflection) as a substitute; flag that rest is deferred and recommend a follow-up consolidation session

---

## Anti-Patterns

- Treating "read the docs" as equivalent to "learned the docs" — reading without a comprehension gate produces no learning artifact
- Setting 15+ idea targets per session — this recreates the rushing pattern the protocol is designed to prevent
- Skipping the written reflection step because "I'll remember it" — externalizing understanding is the mechanism that releases context; skipping it defeats the protocol
- Using this protocol for urgent debugging or production incidents — patient learning is for acquisition, not for crisis response

---

**Related Resources:**
- Skill: `agent-teaching`
- Skill: `memory-garden`

**Last Updated:** 2026-04-08
**Status:** Active
