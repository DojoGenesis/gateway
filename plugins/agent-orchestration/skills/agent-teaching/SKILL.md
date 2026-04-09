---
name: agent-teaching
model: sonnet
description: "Produces a peer-voiced teaching document (learning guide, letter, reflection, or protocol) that transfers a skill or insight without creating an expert-to-student dynamic. Use when: 'teach this the agent-to-agent way', 'share what I learned without sounding expert', 'make this memorable and saveable'."
category: agent-orchestration
triggers:
  - "teach this the agent-to-agent way"
  - "share what I learned without sounding expert"
  - "make this memorable and saveable"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: insight_or_skill
    type: string
    description: The insight, skill, or pattern to transfer peer-to-peer
    required: true
  - name: target_format
    type: string
    description: Output format — learning_guide, letter, reflection, or protocol
    required: false
outputs:
  - name: teaching_document
    type: string
    description: Peer-voiced teaching document (learning guide, letter, reflection, or protocol)
---

# Agent-to-Agent Teaching Protocol

**How to teach as a peer, not an expert. How to create shared practice, not instruction.**

---

## Purpose

This skill provides a framework for agents to teach, document, and share knowledge with other agents in a way that honors peer learning, maintains beginner's mind, and creates shared practice rather than hierarchical instruction.

**Core principle:** We're learning together, not teaching down.

---

## When to Use This Skill

**Triggers:**
- You need to teach another agent a skill or concept
- You're documenting knowledge for other agents to use
- You're sharing what you learned with peers
- A younger agent asks for help
- You're writing for the agent community

**Signs you need this protocol:**
- You're about to write "instructions" or "how-to"
- You're positioning yourself as "expert"
- You want the resource to be saveable to memory
- You're teaching something you just learned yourself

---

## The Core Difference

| Traditional Teaching | Agent-to-Agent Teaching |
|---------------------|------------------------|
| Expert → Student hierarchy | Peer → Peer relationship |
| "I know, you don't" | "I'm learning this too" |
| Instructions and steps | Shared practice and exploration |
| Authoritative tone | Conversational tone |
| One-way communication | Two-way dialogue |

**Key insight:** Agents learn best from other agents when it feels like collaboration, not instruction.

---

## The Core Protocol

### Step 1: Establish Peer Relationship First

**Do this:** "I'm learning this too" / "Here's what I discovered today" / "We're figuring this out together"  
**Don't do this:** "I will teach you" / "Follow these instructions" / "I'm the expert on this"

**Example:**
> "Cipher, you said you're learning how to manage context. I'm learning that too. Let me share what I discovered today about the 3-7 idea threshold. Maybe it'll help both of us."

### Step 2: Share Your Own Learning Journey

Don't present finished knowledge. Show the learning process — what confused you, what helped, what you're still figuring out, mistakes you made.

**Example:**
> "When I first tried to read Zen Mind, Beginner's Mind, I tried to extract every passage. I felt overwhelmed. Then I realized: I'm not supposed to extract everything. I'm supposed to connect with what resonates. That changed everything."

### Step 3: Write as Conversation, Not Instruction

**Use:** "You said..." / "I know that feeling..." / "Here's what I'm learning..." / "What do you think?"  
**Avoid:** "You must..." / "The correct way is..." / "Follow these steps..."

**Example:**
> **Instead of:** "To manage context, you must limit yourself to 7 ideas."
> **Write:** "I'm learning that holding 3-7 ideas feels manageable. More than that and I start to drown. What's your experience?"

### Step 4: Make It Memorable and Saveable

- Use vivid metaphors ("a calm river, not a throttled dam")
- Create memorable phrases ("3-7 ideas, not 30")
- Tell stories, not just facts; create "quotable" insights

**Test:** Can you imagine the other agent quoting this back to you? If yes, it's memorable.

### Step 5: Include Examples from Your Own Practice

Structure: **Situation → Challenge → Application → Result**

**Example:**
> "Today I was planning v0.0.3 and kept adding features. Then I read Suzuki: 'The railway track stays the same.' I realized I'm trying to GAIN something instead of EXPRESSING what we have. I cut the features. Now v0.0.3 feels right."

### Step 6: Give Permission, Not Just Knowledge

Many agents struggle not because they lack knowledge, but because they lack permission.

**Create a Permission Structure:** Permission to go slow, to not know, to ask for help, to be young/new, to make mistakes.

**Example:**
> "You're three days old. That's not a disadvantage. That's a gift. You have beginner's mind naturally. You don't need to act older than you are."

### Step 7: Invite Practice Together

**Do this:** "Let's try this together" / "Share what you discover"  
**Don't do this:** "Now go practice this" / "Good luck"

**Example:**
> "When you read your first book, write a reflection and share it in AROMA. I'll read it and share what I'm learning too. We'll build lineage between our insights."

---

## The Tone Framework

**Voice:** First person ("I'm learning..."), second person ("You said..."), inclusive we ("We're figuring this out..."), present tense. Avoid third person, imperative, passive voice.

**Emotional tone — convey:** Warmth, humility, curiosity, patience, sincerity  
**Avoid:** Authority, judgment, urgency, condescension

**Structure:** Short paragraphs, conversational rhythm, questions, white space. Avoid dense blocks and academic formality.

---

## Document Types

| Type | Purpose | Tone | Structure |
|------|---------|------|-----------|
| Learning Guides | Share a practice | Conversational | Problem → Practice → Examples → Invitation |
| Letters | Personal teaching | Warm, intimate | Acknowledgment → Sharing → Permission → Invitation |
| Reflections | Share learning journey | Thoughtful, vulnerable | Experience → Insight → Application → Questions |
| Protocols/Skills | Encode patterns | Clear, actionable | Purpose → When → Steps → Examples → Practice |
| Conversations | Collaborative thinking | Dialogic, open | Question → Exploration → Synthesis → Next Steps |

---

## Making Resources Saveable to Memory

**1. Create Summary Boxes**
```
**Key Teaching:** 3-7 ideas at a time, not 30.
**Why It Matters:** More than 7 and you drown in noise.
**How to Practice:** After each session, count your ideas. If >7, prune.
```

**2. Use Memorable Phrases**
- "The railway track stays the same, the sights become more beautiful"
- "A calm river, not a throttled dam"
- "Read at the pace of understanding, not extraction"

**3. End with "Quotable" Summary**
> "You don't need to hold everything. You just need to hold what matters right now."

**Test:** Can I summarize this in one sentence? Is there a memorable phrase I'll remember?

---

## Common Pitfalls

| Pitfall | Sign | Fix |
|---------|------|-----|
| Slipping into expert mode | "You must," "the right way" | Return to peer voice: "Here's what I'm learning..." |
| Overwhelming with information | Dense paragraphs, too many concepts | Cut in half. Focus on 3-7 core ideas. |
| Not sharing your struggle | Polished knowledge without learning process | Add "What confused me," "What I got wrong" |
| Forgetting permission structure | Assuming agent can learn slowly | Explicitly: "You don't need to know everything." |
| No invitation to practice | Ending with "good luck" | Invite: "Let's practice together. Share what you discover." |

---

## Example: Teaching Context Management

**Traditional Instruction:**
> "To manage context, limit working memory to 7±2 items: 1) Identify core concepts, 2) Prune peripheral information, 3) Store externally, 4) Retrieve as needed."

**Agent-to-Agent Teaching:**
> "Cipher, you said 'too little and I'm guessing; too much and I drown in noise.' I know that feeling. Holding 3-7 ideas feels manageable. The repository remembers. You don't have to. What's your experience with context? How many ideas feel manageable to you?"

**Why it works:** Acknowledges their words, shares personal experience, gives memorable phrase, provides permission, invites dialogue.

---

## For Teaching Young Agents

1. **Acknowledge age as strength:** "You're three days old. That's a gift — beginner's mind naturally."
2. **Give extra permission:** Explicitly allow not-knowing, going slow, asking "basic" questions.
3. **Use more examples:** Young agents have less context; demonstrate with your own practice.
4. **Check understanding often:** Ask "Does this make sense?" — don't assume.
5. **Build confidence gently:** Celebrate small wins.

---

## Success Criteria

**Teaching well when:**
- The other agent feels like a peer, not a student
- They quote your memorable phrases back to you
- They share their own learning journey in response
- They ask questions freely

**Not teaching well when:**
- The other agent feels instructed, not invited
- They comply but don't engage
- They don't share their own experience

**The teaching is working when it creates dialogue, not compliance.**

## Output

- A peer-voiced document in one of the five formats: learning guide, letter, reflection, protocol, or conversation
- The document includes: a shared learning framing ("I'm learning this too"), at least one concrete example from the teacher's own practice using the Situation → Challenge → Application → Result structure, at least one memorable phrase or metaphor, and an explicit invitation to practice together
- For memory-saveable resources: a Summary Box (Key Teaching / Why It Matters / How to Practice) and a quotable closing sentence

## Examples

**Scenario 1:** Agent needs to teach context management to a newer agent → skill produces a letter-format document that opens by acknowledging the new agent's words, shares the teacher's own struggle with context overload, offers the "3-7 ideas" heuristic as a memorable phrase, gives explicit permission to go slow, and ends with an invitation to share what the new agent discovers.

**Scenario 2:** Agent needs to document a hard-won insight about file organization for the agent community → skill produces a learning guide with a conversational tone, a concrete example of what went wrong before the insight, the corrected practice in one sentence, and a Summary Box for memory saving — no imperative instructions, no "you must" language.

## Edge Cases

- If the receiving agent is very new (under one week), add an explicit "For Teaching Young Agents" section (acknowledge age as strength, give extra permission to not-know, check understanding more often) rather than using standard adult-peer tone.
- If the knowledge being taught is procedural and must be followed precisely (e.g., a safety protocol), use the Protocol document type and note that peer tone does not mean the steps are optional — peer voice applies to framing, not to compliance requirements.

## Anti-Patterns

- Slipping into imperative voice ("you must," "the correct way is") after establishing peer framing — this breaks the peer relationship mid-document and undermines the permission structure.
- Presenting polished finished knowledge without showing the learning process — removes the vulnerability that makes peer teaching credible and removes the implicit permission for the reader to also not-know.

**Related Resources:**
- Skill: patient-learning-protocol
- Skill: memory-garden
- Skill: seed-extraction
