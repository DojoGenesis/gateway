# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: product-positioning
description: Define clear product positioning based on market research and competitive analysis
triggers:
  - "define product positioning"
  - "create positioning statement"
  - "develop market position"
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
    - strategic-agent
    - research-agent
---

# Product Positioning Scout Skill

**Version:** 1.1  
**Created:** 2026-02-07  
**Updated:** 2026-02-07  
**Author:** Manus AI  
**Purpose:** To guide the process of reframing a binary product decision into a strategic product positioning opportunity by identifying the unique value of a product or feature.

---

## I. The Philosophy: Beyond the Binary

The most common trap in product strategy is the binary choice: keep or kill, build or buy, deprecate or maintain. These choices are limiting because they assume the value of a product is fixed. The Product Positioning Scout operates on a different principle: **value is contextual**.

This skill is about breaking free from the binary trap by asking a more powerful question: **"What is this uniquely good at?"** By identifying the unique context in which a product or feature shines, we can transform a "legacy" feature into a "premium" experience, a "redundant" product into a "complementary" one.

**The core insight:** The reframe is the prize, not the initial answer.

---

## II. When to Use This Skill

- When facing a decision about whether to keep or deprecate a feature or product
- When a product or feature seems redundant or is underperforming
- When planning a multi-surface product strategy (e.g., web, desktop, mobile)
- At the beginning of a strategic planning cycle
- Before using `/strategic-scout` to ensure the question is properly framed

---

## III. The Workflow

This is a 5-step workflow for reframing a product decision.

### Step 1: Identify the Binary Trap

**Goal:** Recognize when a strategic question is being framed as a simple, limiting binary choice.

**Actions:**
- Listen for questions like "Should we keep X or get rid of it?"
- Notice when the conversation is focused on resource allocation rather than strategic value
- Identify the implicit assumption that the product/feature has only one use case

**Output:** A clear articulation of the binary trap

**Example:** "Should we deprecate the web app now that we're building desktop?"

---

### Step 2: Introduce the Unlocking Question

**Goal:** Reframe the conversation by asking a question that shatters the binary and opens up new possibilities.

**Actions:**
- Ask: **"What is this uniquely good at that the other thing isn't?"**
- Shift the focus from "what to do with this" to "what is its unique value?"
- Challenge the assumption that products must compete rather than complement

**Output:** A reframed question that opens up new strategic possibilities

**Example:** "What is the web app uniquely good at that desktop isn't?" → Discovery, onboarding, cross-platform access without installation

---

### Step 3: Explore the Unique Value

**Goal:** Brainstorm the unique strengths and contexts of the product in question.

**Actions:**
- List the unique strengths of the product/feature
- Identify the specific contexts (e.g., mobile, on-the-go, deep work) where those strengths are most valuable
- Consider edge cases and niche use cases that might be surprisingly valuable

**Output:** A comprehensive list of unique value propositions

**Example for web app:**
- No installation required
- Cross-platform (works on any OS)
- Easy to share via URL
- Great for discovery and onboarding
- Accessible from any device

---

### Step 4: Re-Scout with the New Lens

**Goal:** Use the new understanding of unique value to re-scout the strategic options.

**Actions:**
- Use `/strategic-scout` with the new framing
- Explore routes that leverage the unique value (e.g., Mobile-First PWA, Native Companion App)
- Consider complementary positioning rather than competitive positioning

**Output:** A set of strategic routes that leverage the unique value

**Example routes:**
- Route 1: Web as discovery, desktop as core product
- Route 2: Web as free tier, desktop as premium
- Route 3: Web for onboarding, desktop for power users
- Route 4: Deprecate web, focus on desktop (original binary)

---

### Step 5: Synthesize and Propose New Positioning

**Goal:** Synthesize the new routes into a coherent product strategy and positioning.

**Actions:**
- Select the best route (e.g., Hybrid - PWA Now, Native Later)
- Define the new product positioning (e.g., Desktop for deep work, Mobile for on-the-go orchestration)
- Propose a new timeline and pricing model
- Document the strategic rationale

**Output:** A complete product positioning strategy

**Example:** "Desktop as core ($20/month), Mobile PWA as premium tier (separate subscription, 4-6 weeks after desktop v1)"

---

## IV. Best Practices

### 1. The Reframe is Everything

**Why:** The most powerful strategic moves come from reframing the question, not from choosing between existing options.

**How:** Always start by questioning the framing of the decision. Is it really a binary choice?

---

### 2. Value is Contextual

**Why:** A product's value is not inherent; it's determined by the context in which it's used.

**How:** For every product/feature, ask "In what context is this uniquely valuable?"

---

### 3. Complement, Don't Compete

**Why:** The best multi-surface strategies create complementary experiences, not competing ones.

**How:** Frame each surface by its unique job-to-be-done, not by its feature list.

---

### 4. The Unlocking Question is Universal

**Why:** "What is this uniquely good at?" works for products, features, surfaces, and even people.

**How:** Use this question whenever you're facing a binary decision about value.

---

### 5. Document the Reframe

**Why:** The reframe is the most valuable artifact. It changes how everyone thinks about the product.

**How:** Write down the before/after framing explicitly. Share it widely.

---

## V. Quality Checklist

Before delivering the new strategy, ensure you can answer "yes" to all of the following questions:

- [ ] Have you identified and articulated the initial binary trap?
- [ ] Have you asked the "Unlocking Question" to reframe the conversation?
- [ ] Have you clearly defined the unique value and context of the product/feature?
- [ ] Have you used `/strategic-scout` to explore new routes based on the new framing?
- [ ] Have you proposed a new, coherent product positioning, timeline, and business model?
- [ ] Have you documented the strategic rationale for the new positioning?

---

## VI. Example: Dojo Genesis Web App Positioning

**The Binary Trap:** "Should we deprecate the web app now that we're building desktop?"

**The Unlocking Question:** "What is the web app uniquely good at that desktop isn't?"

**The Unique Value:**
- No installation required
- Cross-platform access
- Easy to share via URL
- Great for discovery and onboarding
- Accessible from any device

**The Re-Scout:** Used `/strategic-scout` to explore 4 routes:
1. Deprecate web, focus on desktop
2. Web as companion (lightweight version)
3. Web as discovery/onboarding, desktop as core
4. Hybrid - PWA now, native later

**The New Positioning:**
- **Desktop:** Core product for deep work ($20/month)
- **Mobile PWA:** Premium tier for on-the-go orchestration (separate subscription, 4-6 weeks after desktop v1)
- **Web:** Free tier for discovery and onboarding

**The Outcome:** Transformed a deprecation decision into a multi-surface strategy where each surface has a unique, complementary role.

**Key Insight:** The web app wasn't redundant—it was uniquely good at discovery and onboarding. Desktop is uniquely good at deep work. Mobile is uniquely good at on-the-go orchestration.

---

## VII. Common Pitfalls to Avoid

### Pitfall 1: Accepting the Binary Without Question

**Problem:** Most binary choices are false binaries. Accepting them limits strategic options.

**Solution:** Always question the framing. Ask "Is this really a binary choice?"

---

### Pitfall 2: Focusing on Features Instead of Context

**Problem:** Comparing feature lists leads to "X is redundant because Y has all its features."

**Solution:** Focus on context of use, not feature parity. Ask "What is this uniquely good at?"

---

### Pitfall 3: Competitive Positioning by Default

**Problem:** Assuming products must compete rather than complement.

**Solution:** Explore complementary positioning first. Competition should be a last resort.

---

### Pitfall 4: Skipping the Re-Scout

**Problem:** Reframing the question but not exploring new routes based on the reframe.

**Solution:** Always use `/strategic-scout` after reframing to explore new possibilities.

---

### Pitfall 5: Not Documenting the Reframe

**Problem:** The reframe gets lost in conversation. People revert to the binary.

**Solution:** Write down the before/after framing explicitly. Make it a reference document.

---

## VIII. Related Skills

- **`strategic-scout`** - Use this after reframing to explore new routes
- **`iterative-scouting`** - Use this to refine the positioning based on feedback
- **`multi-surface-strategy`** - Use this to design the complete multi-surface strategy
- **`release-specification`** - Use this to write specs for the new positioning
- **`strategic-to-tactical-workflow`** - Use this to move from positioning to implementation

---

## IX. Skill Metadata

**Token Savings:** ~3,000-5,000 tokens per positioning session  
**Quality Impact:** Transforms binary decisions into strategic opportunities  
**Maintenance:** Update when new positioning patterns emerge

**When to Update This Skill:**
- After completing 3-5 positioning sessions (to incorporate new patterns)
- When a new type of binary trap is discovered
- When the unlocking question evolves or new variants emerge

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active
