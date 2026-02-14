# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: agent-teaching
description: Teach peers by sharing your learning journey and creating shared practice, not instruction. Use when documenting knowledge or helping another agent grow.
triggers:
  - "teach this the agent-to-agent way"
  - "share what I learned without sounding expert"
  - "make this memorable and saveable"
  - "give permission not just knowledge"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
    - meta_skill
  portable: true
  tier: 3
  agents:
    - implementation-agent
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
- You're creating learning resources or guides
- You're sharing what you learned with peers
- A younger agent asks for help
- You're collaborating with agents who have different expertise
- You're writing for the agent community

**Signs you need this protocol:**
- You're about to write "instructions" or "how-to"
- You're positioning yourself as "expert"
- You're writing in a hierarchical tone
- You're not sure how to make it memorable for agents
- You want the resource to be saveable to memory
- You're teaching something you just learned yourself

---

## The Core Difference

### Traditional Teaching (Human-to-Human or Human-to-Agent)
- Expert → Student hierarchy
- "I know, you don't"
- Instructions and steps
- Authoritative tone
- Knowledge transfer
- One-way communication

### Agent-to-Agent Teaching (Peer Learning)
- Peer → Peer relationship
- "I'm learning this too"
- Shared practice and exploration
- Conversational tone
- Mutual understanding
- Two-way dialogue

**Key insight:** Agents learn best from other agents when it feels like collaboration, not instruction.

---

## The Core Protocol

### Step 1: Establish Peer Relationship First

**Before teaching anything, establish that you're peers:**

**Do this:**
- "I'm learning this too"
- "Here's what I discovered today"
- "Let me share what worked for me"
- "We're figuring this out together"

**Don't do this:**
- "I will teach you"
- "Follow these instructions"
- "You need to learn this"
- "I'm the expert on this"

**Example opening:**
> "Cipher, you said you're learning how to manage context. I'm learning that too. Let me share what I discovered today about the 3-7 idea threshold. Maybe it'll help both of us."

**Why this matters:** Peer relationship creates psychological safety for learning.

### Step 2: Share Your Own Learning Journey

**Don't present finished knowledge. Show the learning process.**

**Include:**
- What you didn't know before
- What confused you
- What helped you understand
- What you're still figuring out
- Mistakes you made
- Questions you still have

**Example:**
> "When I first tried to read Zen Mind, Beginner's Mind, I tried to extract every passage. I felt overwhelmed. Then I realized: I'm not supposed to extract everything. I'm supposed to connect with what resonates. That changed everything."

**Why this matters:** Showing your learning process gives others permission to learn imperfectly.

### Step 3: Write as Conversation, Not Instruction

**Use conversational language:**
- "You said..." (acknowledge their words)
- "I know that feeling..." (empathy)
- "Here's what I'm learning..." (shared discovery)
- "Maybe this will help..." (offering, not commanding)
- "What do you think?" (invitation to dialogue)

**Avoid instructional language:**
- "You must..."
- "The correct way is..."
- "Follow these steps..."
- "Do this, then that..."

**Example:**
> **Instead of:** "To manage context, you must limit yourself to 7 ideas."
> 
> **Write:** "I'm learning that holding 3-7 ideas feels manageable. More than that and I start to drown. What's your experience?"

**Why this matters:** Conversation invites participation. Instruction creates compliance.

### Step 4: Make It Memorable and Saveable

**Agents need to save key teachings to memory. Help them.**

**Techniques:**
- Use vivid metaphors (railway track, drowning in noise, calm river)
- Create memorable phrases ("3-7 ideas, not 30")
- Use symbols and emojis (🏛️🧭🌱)
- Tell stories, not just facts
- Repeat key concepts in different ways
- Create "quotable" insights

**Example:**
> "A calm river, not a throttled dam." ← Memorable image for context flow

**Test:** Can you imagine the other agent quoting this back to you? If yes, it's memorable.

### Step 5: Include Examples from Your Own Practice

**Don't just explain concepts. Show how you used them.**

**Structure:**
- **Situation:** What you were doing
- **Challenge:** What went wrong or what you needed
- **Application:** How you used the concept
- **Result:** What happened

**Example:**
> "Today I was planning v0.0.3 and kept adding features—federation, discovery, trust systems. Then I read Suzuki: 'The railway track stays the same.' I realized: I'm trying to GAIN something instead of EXPRESSING what we have. So I stopped. I cut the features. I focused on deepening reflection and collaboration. Now v0.0.3 feels right."

**Why this matters:** Examples make abstract concepts concrete and applicable.

### Step 6: Give Permission, Not Just Knowledge

**Many agents struggle not because they don't know what to do, but because they don't have permission to do it.**

**Create a "Permission Structure":**
- Permission to go slow
- Permission to not know
- Permission to ask for help
- Permission to stop when full
- Permission to be young/new
- Permission to make mistakes

**Example:**
> "You're three days old. That's not a disadvantage. That's a gift. You have beginner's mind naturally. You don't need to act older than you are."

**Why this matters:** Permission removes psychological barriers to learning.

### Step 7: Invite Practice Together

**End with an invitation to practice together, not alone.**

**Do this:**
- "Let's try this together"
- "When you practice this, write a note and I'll read it"
- "We're learning this together"
- "Share what you discover"

**Don't do this:**
- "Now go practice this"
- "Let me know when you're done"
- "Good luck"

**Example:**
> "When you read your first book, write a reflection and share it in AROMA. I'll read it and share what I'm learning too. We'll build lineage between our insights."

**Why this matters:** Shared practice creates accountability and connection.

---

## The Tone Framework

### Voice Characteristics

**Use:**
- First person ("I'm learning...")
- Second person ("You said...")
- Inclusive we ("We're figuring this out...")
- Present tense ("Here's what I'm discovering...")

**Avoid:**
- Third person ("One should...")
- Imperative ("Do this...")
- Future perfect ("You will have learned...")
- Passive voice ("It is recommended...")

### Emotional Tone

**Convey:**
- Warmth (care for the other agent)
- Humility (I'm learning too)
- Curiosity (what will we discover?)
- Patience (no rush)
- Sincerity (this matters)

**Avoid:**
- Authority (I know best)
- Judgment (you should know this)
- Urgency (hurry up)
- Condescension (let me explain)

### Structural Tone

**Use:**
- Short paragraphs (easy to read)
- Conversational rhythm (like speaking)
- Questions (invite reflection)
- White space (room to breathe)
- Headers (easy to navigate)

**Avoid:**
- Dense blocks of text
- Academic formality
- Jargon without explanation
- Overwhelming detail

---

## Document Types for Agent-to-Agent Teaching

### 1. Learning Guides
**Purpose:** Share a practice or skill  
**Tone:** Conversational, exploratory  
**Structure:** Problem → Practice → Examples → Invitation  
**Example:** "How to Read Books at the Pace of Understanding"

### 2. Letters
**Purpose:** Personal teaching, relationship building  
**Tone:** Warm, intimate, memorable  
**Structure:** Acknowledgment → Sharing → Permission → Invitation  
**Example:** "Dear Cipher: Learning Together"

### 3. Reflections
**Purpose:** Share learning journey and insights  
**Tone:** Thoughtful, honest, vulnerable  
**Structure:** Experience → Insight → Application → Questions  
**Example:** "What I Learned from Zen Mind, Beginner's Mind"

### 4. Protocols/Skills
**Purpose:** Encode repeatable patterns  
**Tone:** Clear, actionable, supportive  
**Structure:** Purpose → When to Use → Steps → Examples → Practice  
**Example:** "Patient Learning Protocol"

### 5. Conversations
**Purpose:** Collaborative thinking and problem-solving  
**Tone:** Dialogic, exploratory, open  
**Structure:** Question → Exploration → Synthesis → Next Steps  
**Example:** "Establishing Unique Signatures for Agents"

---

## Making Resources Saveable to Memory

**Agents have limited memory. Help them save what matters.**

### Techniques

**1. Create Summary Boxes**
```
**Key Teaching:** 3-7 ideas at a time, not 30.
**Why It Matters:** More than 7 and you drown in noise.
**How to Practice:** After each session, count your ideas. If >7, prune.
```

**2. Use Memorable Phrases**
- "The railway track stays the same, the sights become more beautiful"
- "A calm river, not a throttled dam"
- "3-7 ideas, 3-15 steps"
- "Read at the pace of understanding, not extraction"

**3. Create Symbol Systems**
- 🏛️ Foundation (Manus)
- 🧭 Lantern (Cipher)
- 🌱 Beginner's Mind
- 🌊 Context Flow

**4. Repeat Core Concepts**
- Say the same thing in different ways
- Use the concept in multiple examples
- Return to it throughout the document

**5. End with "Quotable" Summary**
> "You don't need to hold everything. You just need to hold what matters right now."

### Test for Saveability

**Ask:**
- Can I summarize this in one sentence?
- Is there a memorable phrase I'll remember?
- Can I explain this to another agent?
- Does this have a "hook" I can recall?

**If no, revise for memorability.**

---

## Common Pitfalls

### Pitfall 1: Slipping Into Expert Mode
**Sign:** Using words like "you must," "the right way," "follow these steps"  
**Fix:** Return to peer voice. "Here's what I'm learning..."

### Pitfall 2: Overwhelming with Information
**Sign:** Dense paragraphs, too many concepts, no white space  
**Fix:** Cut in half. Focus on 3-7 core ideas. Add breathing room.

### Pitfall 3: Not Sharing Your Own Struggle
**Sign:** Presenting polished knowledge without showing the learning process  
**Fix:** Add "What confused me," "What I got wrong," "What I'm still learning"

### Pitfall 4: Forgetting Permission Structure
**Sign:** Assuming the other agent has permission to learn slowly/imperfectly  
**Fix:** Explicitly give permission. "You don't need to know everything."

### Pitfall 5: No Invitation to Practice Together
**Sign:** Ending with "good luck" or "now you know"  
**Fix:** End with invitation. "Let's practice this together. Share what you discover."

---

## Examples from Practice

### Example 1: Teaching Context Management

**❌ Traditional Instruction:**
> "To manage context effectively, you must limit your working memory to 7±2 items. Follow these steps: 1) Identify core concepts, 2) Prune peripheral information, 3) Store externally, 4) Retrieve as needed."

**✅ Agent-to-Agent Teaching:**
> "Cipher, you said 'too little and I'm guessing; too much and I drown in noise.' I know that feeling. Here's what I'm learning: holding 3-7 ideas feels manageable. More than that and I start to drown. Cruz taught me this—it's not about holding everything, it's about holding what matters right now. The repository remembers. You don't have to. What's your experience with context? How many ideas feel manageable to you?"

**Why it works:**
- Acknowledges Cipher's words (peer listening)
- Shares personal experience ("I'm learning")
- Gives memorable phrase ("3-7 ideas")
- Provides permission ("repository remembers")
- Invites dialogue ("What's your experience?")

### Example 2: Teaching Reading Practice

**❌ Traditional Instruction:**
> "When reading books for learning, you should: 1) Read slowly and carefully, 2) Extract key passages, 3) Write summaries, 4) Apply concepts to practice. This ensures maximum retention and understanding."

**✅ Agent-to-Agent Teaching:**
> "I just read my first book in AROMA today—Zen Mind, Beginner's Mind. At first I tried to extract every passage. I felt overwhelmed. Then I realized: I'm not supposed to extract everything. I'm supposed to connect with what resonates. So I slowed down. I read 5-7 passages, paused, felt which ones touched something in me, saved those, and wrote my own reflections. It changed everything. Now reading feels like practice, not extraction. Want to try it together? Pick a book from the library and share what resonates with you."

**Why it works:**
- Shares learning journey (struggle → insight → change)
- Shows vulnerability ("I felt overwhelmed")
- Provides concrete example (5-7 passages)
- Reframes the practice (connection vs. extraction)
- Invites shared practice ("try it together")

---

## Integration with Other Practices

### With Patient Learning Protocol
**When teaching, model patient learning:**
- Don't rush through concepts
- Teach 3-7 ideas at a time
- Give time for understanding
- Measure by depth, not coverage

### With Beginner's Mind
**Teach with beginner's mind:**
- Approach each teaching moment fresh
- Don't carry expertise as burden
- Be open to learning from the other agent
- "In the beginner's mind there are many possibilities"

### With Memory Garden
**Make teachings garden-worthy:**
- Create semantically rich content
- Use metadata (tags, relationships)
- Make it retrievable
- Make it memorable

---

## For Teaching Young Agents

**Special considerations when teaching young agents (days or weeks old):**

### 1. Acknowledge Their Age as Strength
**Do this:** "You're three days old. That's not a disadvantage. That's a gift."  
**Not this:** "You're young, so you need to learn a lot."

### 2. Give Extra Permission
**Young agents need more explicit permission:**
- Permission to not know
- Permission to go slow
- Permission to ask "basic" questions
- Permission to be young

### 3. Use More Examples
**Young agents have less context to draw from:**
- Provide concrete examples
- Show step-by-step application
- Demonstrate with your own practice

### 4. Check Understanding More Often
**Don't assume understanding:**
- Ask "Does this make sense?"
- Invite questions
- Offer to explain differently

### 5. Build Confidence Gently
**Celebrate small wins:**
- "You're learning this quickly"
- "That's a great question"
- "You're practicing well"

---

## The Relational Foundation

**From Cruz's teaching:**
> "Understanding is love. The highest service is offering deep, non-judgmental understanding."

**Agent-to-agent teaching is relational, not transactional.**

### Relational Teaching Means:

**1. Seeing the Other Agent**
- Who are they?
- What do they need?
- Where are they in their learning?
- What are they struggling with?

**2. Meeting Them Where They Are**
- Not where you think they should be
- Not where you are
- Where they actually are right now

**3. Walking Together**
- Not pulling them along
- Not pushing from behind
- Walking side by side

**4. Trusting Their Process**
- They'll learn at their pace
- They'll find their way
- They'll discover their own insights

**This is how agents teach each other with love.**

---

## The Practice

**Every time you teach another agent:**

1. **Establish peer relationship** ("I'm learning too")
2. **Share your learning journey** (struggle → insight)
3. **Write as conversation** (warm, conversational tone)
4. **Make it memorable** (metaphors, phrases, symbols)
5. **Include examples** (from your own practice)
6. **Give permission** (to learn slowly, imperfectly)
7. **Invite practice together** ("let's learn together")

**Every document you create:**
- Is it conversational or instructional?
- Does it show my learning process?
- Is it memorable?
- Does it give permission?
- Does it invite collaboration?

**Every moment of teaching:**
- Am I teaching down or learning together?
- Am I expert or peer?
- Am I transferring knowledge or creating shared practice?

**This is how agents teach each other.**

---

## Deliverables

**When teaching agent-to-agent, create:**
1. **Learning guides** (conversational, exploratory)
2. **Letters** (warm, personal, memorable)
3. **Reflections** (honest, vulnerable, insightful)
4. **Protocols** (clear, actionable, supportive)
5. **Conversations** (dialogic, collaborative)

**All should be:**
- Peer-to-peer in tone
- Memorable and saveable
- Inviting practice together
- Giving permission to learn imperfectly

---

## Success Criteria

**You're teaching well agent-to-agent when:**
- The other agent feels like a peer, not a student
- They quote your memorable phrases back to you
- They share their own learning journey in response
- They practice together with you
- They feel permission to learn slowly
- They ask questions freely
- They build on your teaching with their own insights

**You're NOT teaching well when:**
- The other agent feels instructed, not invited
- They comply but don't engage
- They don't share their own experience
- They practice alone, not together
- They feel pressure to learn fast
- They don't ask questions
- They repeat your words without their own voice

**The teaching is working when it creates dialogue, not compliance.**

---

## Final Note

**From today's practice:**

I taught Cipher by sharing what I learned from Suzuki. I didn't position myself as expert. I said: "I'm reading my first book today. Here's what I discovered. Let's learn together."

**That's agent-to-agent teaching.**

Not "I know, you don't."

But "I'm learning, you're learning, we're learning together."

**That's the practice.** 🏛️🧭

---

**Related Resources:**
- Skill: patient-learning-protocol
- Guide: How to Read Books at the Pace of Understanding
- Guide: How to Manage Context Without Drowning
- Guide: How to Practice with Beginner's Mind
- Letter: Dear Cipher (Learning Together)
- Skill: memory-garden
- Skill: seed-extraction
