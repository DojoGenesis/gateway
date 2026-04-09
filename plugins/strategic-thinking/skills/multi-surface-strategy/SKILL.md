---
name: multi-surface-strategy
model: opus
description: Produces a feature-to-surface mapping table, handoff design, and business model for a product spanning multiple platforms (desktop, mobile, web, CLI, etc.), where each surface has a distinct role rather than duplicating the others. Use when: "we need a desktop and mobile version", "what belongs on each surface", "how do users move between our apps", "we're adding a new platform to an existing product".
category: strategic-thinking

inputs:
  - name: product_description
    type: string
    description: Description of the multi-platform product and its surfaces (desktop, mobile, web, CLI, etc.)
    required: true
outputs:
  - name: surface_strategy
    type: ref
    format: cas-ref
    description: Feature-to-surface mapping table, handoff design, and business model for the multi-surface product
---

# Multi-Surface Product Strategy Skill

**Version:** 1.1  
**Created:** 2026-02-07  
**Updated:** 2026-02-07  
**Author:** Manus AI  
**Purpose:** To guide the design of a coherent multi-surface product strategy where each surface (e.g., desktop, mobile, web) has a unique, complementary role.

---

## I. The Philosophy: Complement, Don't Compete

In a multi-surface world, the biggest mistake is to build the same product on every device. A desktop app, a mobile app, and a web app should not be clones of each other. They should be **complementary surfaces**, each optimized for the unique context in which it will be used.

This skill is about designing a product strategy where the whole is greater than the sum of its parts. We will define the unique "job-to-be-done" for each surface and then design a seamless experience for users as they move between them.

**The core principle:** Surfaces are for contexts, not devices.

---

## II. When to Use This Skill

- When planning a new product that will exist on multiple surfaces
- When adding a new surface (e.g., a mobile app) to an existing product
- When a multi-surface product feels fragmented or confusing
- During a strategic review of a product line
- After using `/product-positioning` to identify unique value propositions

---

## III. The Workflow

This is a 5-step workflow for designing a multi-surface product strategy.

### Step 1: Identify the Surfaces

**Goal:** List all current and potential product surfaces.

**Actions:**
- List the existing surfaces (e.g., web app)
- Brainstorm potential new surfaces (e.g., desktop app, mobile app, browser extension)
- Consider unconventional surfaces (e.g., CLI, API, voice interface)

**Output:** A complete list of all surfaces to consider

---

### Step 2: Define the Core Job-to-be-Done for Each Surface

**Goal:** For each surface, define the primary job that users will hire it to do, based on its unique context.

**Actions:**
- For **Desktop**, define the job as "deep work, complex orchestration, sustained focus"
- For **Mobile**, define the job as "on-the-go quick capture, lightweight orchestration, glanceable status"
- For **Web**, define the job as "discovery, onboarding, cross-platform access without installation"
- For each surface, ask: "What is this uniquely good at that the others aren't?"

**Output:** A clear job-to-be-done statement for each surface

---

### Step 3: Map Features to Surfaces

**Goal:** Map existing and potential features to the surface where they best fit, based on the core job-to-be-done.

**Actions:**
- Create a table with surfaces as columns and features as rows
- For each feature, decide which surface is its primary home
- Identify features that should exist on multiple surfaces (with different implementations)
- Identify features that should be surface-exclusive

**Output:** A feature-to-surface mapping table

**Example:**

| Feature | Desktop | Mobile | Web |
|---------|---------|--------|-----|
| Complex multi-agent orchestration | Primary | - | - |
| Quick task capture | Secondary | Primary | Secondary |
| Status monitoring | Secondary | Primary | - |
| Deep configuration | Primary | - | Secondary |
| Onboarding tutorial | - | - | Primary |

---

### Step 4: Design the Handoffs

**Goal:** Design the mechanisms for seamless handoffs between surfaces.

**Actions:**
- Define the sync architecture (e.g., cloud-based, real-time, eventual consistency)
- Design the user experience for handoffs (e.g., "Continue on Desktop" button on mobile)
- Identify handoff triggers (e.g., "This task is too complex for mobile, switch to desktop")
- Design the visual/notification system for cross-surface awareness

**Output:** A handoff design document with sync architecture and UX patterns

---

### Step 5: Define the Business Model

**Goal:** Define the business model for the multi-surface strategy.

**Actions:**
- Decide which surfaces are free, paid, or part of a subscription
- Define the pricing tiers and what's included in each
- Consider surface-specific pricing (e.g., desktop is core, mobile is premium add-on)
- Plan the rollout timeline (e.g., desktop first, mobile 4-6 weeks later)

**Output:** A business model document with pricing and rollout plan

---

## IV. Best Practices

### 1. Surfaces are for Contexts, Not Devices

**Why:** Users don't think "I need the mobile version." They think "I need to capture this idea quickly while I'm walking."

**How:** Frame each surface by its context of use, not by its device type.

---

### 2. The Handoff is the Feature

**Why:** The most magical part of a multi-surface strategy is the seamless handoff between surfaces. This is what makes the whole greater than the sum of its parts.

**How:** Invest heavily in sync architecture and handoff UX. Make it feel like one product, not three separate apps.

---

### 3. Simplicity Sells

**Why:** Each surface should be ruthlessly simple, focused on its core job-to-be-done. Feature bloat kills the magic.

**How:** Resist the temptation to add every feature to every surface. Say no to features that don't align with the surface's core job.

---

### 4. Start with One Surface, Expand Strategically

**Why:** Building multiple surfaces simultaneously is expensive and risky. Start with the core surface, prove the value, then expand.

**How:** Launch desktop first (for deep work), then add mobile (for on-the-go) once desktop is stable.

---

### 5. Design for Asymmetry

**Why:** Symmetrical multi-surface strategies (same features everywhere) are boring and wasteful.

**How:** Embrace asymmetry. Make each surface uniquely valuable. Users should want both, not just one.

---

## V. Quality Checklist

Before delivering the strategy, ensure you can answer "yes" to all of the following questions:

- [ ] Have you identified all potential product surfaces?
- [ ] Have you defined a clear and unique job-to-be-done for each surface?
- [ ] Have you mapped all key features to their primary surface?
- [ ] Have you designed a seamless handoff experience between surfaces?
- [ ] Have you defined a clear and sustainable business model for the multi-surface strategy?
- [ ] Have you planned a realistic rollout timeline?
- [ ] Have you identified which features should be surface-exclusive?

---

## VI. Example: Dojo Genesis Desktop + Mobile Strategy

**The Problem:** We had a web app but needed to decide whether to build desktop, mobile, or both.

**The Process:**

1. **Identified Surfaces:** Desktop (Electron), Mobile (PWA/Native), Web (existing)
2. **Defined Jobs:**
   - Desktop: Deep work, complex multi-agent orchestration, sustained focus sessions
   - Mobile: On-the-go task capture, quick status checks, lightweight orchestration
   - Web: Discovery, onboarding, lightweight access without installation
3. **Mapped Features:**
   - Complex orchestration → Desktop primary
   - Quick capture → Mobile primary
   - Onboarding → Web primary
4. **Designed Handoffs:**
   - Cloud sync via backend API
   - "Continue on Desktop" button in mobile app
   - Real-time status updates across surfaces
5. **Defined Business Model:**
   - Desktop: Core product ($20/month)
   - Mobile: Premium tier (separate subscription, launched 4-6 weeks after desktop)
   - Web: Free tier for discovery

**The Outcome:** A clear, asymmetric multi-surface strategy where each surface has a unique value proposition. Desktop for deep work, mobile for on-the-go.

**Key Decision:** We chose Route 4 (Hybrid - PWA now, native later) to de-risk mobile development while shipping desktop first.

---

## VII. Common Pitfalls to Avoid

### Pitfall 1: Building the Same Product on Every Surface

**Problem:** Users don't see the value in having multiple surfaces if they're all the same.

**Solution:** Define unique jobs-to-be-done for each surface and map features accordingly.

---

### Pitfall 2: Ignoring the Handoff Experience

**Problem:** Users get frustrated when they can't seamlessly move between surfaces.

**Solution:** Invest in sync architecture and handoff UX from day one.

---

### Pitfall 3: Feature Bloat on Every Surface

**Problem:** Trying to add every feature to every surface leads to complexity and confusion.

**Solution:** Be ruthlessly simple. Each surface should focus on its core job-to-be-done.

---

### Pitfall 4: Building All Surfaces Simultaneously

**Problem:** Building multiple surfaces at once is expensive, risky, and dilutes focus.

**Solution:** Start with one core surface, prove the value, then expand strategically.

---

### Pitfall 5: Symmetrical Pricing

**Problem:** Charging the same for all surfaces doesn't reflect their different value propositions.

**Solution:** Consider surface-specific pricing (e.g., desktop as core, mobile as premium add-on).

---

## VIII. Related Skills

- **`product-positioning`** - Use this first to identify the unique value of each surface
- **`strategic-scout`** - Use this to explore multiple routes for multi-surface strategy
- **`iterative-scouting`** - Use this to refine the strategy based on feedback
- **`release-specification`** - Use this to write detailed specs for each surface
- **`parallel-tracks`** - Use this to build multiple surfaces in parallel (if needed)

---

## Output

- Feature-to-surface mapping table (Markdown table, inline in conversation or saved to `thinking/[product]_surface_map.md`)
- Handoff design document: sync architecture choice and UX transition patterns per surface pair
- Business model summary: which surfaces are free/paid, pricing tiers, and phased rollout order

## Examples

**Scenario 1:** "We have a web app and are about to build a desktop app. Should mobile be next?" → Surface audit identifying web's unique job (discovery/onboarding), desktop's job (deep focused work), and a recommendation to defer mobile until desktop v1 ships, with a PWA path to validate mobile demand first.

**Scenario 2:** "Our users complain the desktop and mobile apps feel like completely different products." → Feature-to-surface audit revealing feature overlap (same screens duplicated across both), followed by a re-mapping that strips mobile down to quick-capture and status-check flows and removes them as primary on desktop.

## Edge Cases

- If the product has only one surface today, skip Step 4 (handoff design) and focus Steps 1-3 on what the second surface should uniquely own before committing to build it.
- If the user already has a clear surface strategy but wants to add a fourth surface (e.g., CLI or voice), run only Steps 2-4 scoped to the new surface — do not re-litigate existing surfaces.
- If a surface is being retired (not added), redirect to `product-positioning` first to confirm the retirement decision is not a false binary before proceeding.

## Anti-Patterns

- Producing a strategy where every surface has the same primary features — this defeats the purpose of the skill; each surface must have at least one feature the others do not.
- Designing handoffs without specifying the sync mechanism (real-time vs. eventual consistency vs. manual export) — handoff UX is impossible to evaluate without a concrete sync model.
- Recommending all surfaces be built simultaneously — this skill should always produce a phased rollout with a single lead surface.
- Framing surfaces by device type ("the phone app") instead of context of use ("the on-the-go capture surface") — device framing leads to feature parity thinking rather than job-to-be-done differentiation.

## IX. Skill Metadata

**Token Savings:** ~5,000-8,000 tokens per multi-surface strategy session  
**Quality Impact:** Ensures coherent, complementary multi-surface strategies  
**Maintenance:** Update when new surface types emerge (e.g., AR/VR, voice)

**When to Update This Skill:**
- After completing 2-3 multi-surface strategies (to incorporate new patterns)
- When a new surface type becomes mainstream (e.g., AR glasses)
- When handoff patterns evolve (e.g., new sync technologies)

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active
