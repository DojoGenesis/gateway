---
name: design-system-selector
model: sonnet
description: "Produces a scored comparison of available DESIGN.md design systems against a project's requirements and delivers a recommended selection with adaptation notes. Use when: 'which design system fits this project', 'pick a design reference for me', 'match a design system to this brand', 'compare design system options for this client'."
category: continuous-learning
triggers:
  - "which design system fits this project"
  - "pick a design reference for me"
  - "match a design system to this brand"
  - "compare design system options for this client"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: project_requirements
    type: string
    description: Project requirements and brand constraints to match against design systems
    required: true
outputs:
  - name: selection_report
    type: string
    description: Scored comparison of available design systems with a recommended selection and adaptation notes
---
# Design System Selector

## I. Philosophy

Design systems are not interchangeable. Stripe's authority differs from Notion's warmth
differs from Linear's precision. Each embeds a philosophy about who the user is, what
they value, and how they should feel while working.

The selector prevents the common mistake of picking a design system because it's popular
rather than because it fits. A fintech dashboard wearing Notion's soft pastels sends
the wrong message. A community platform dressed in Stripe's sharp minimalism feels cold.

Fit is not aesthetic preference -- it is alignment between brand intent and visual
language. This skill makes that alignment measurable.

## II. When to Use

- Starting a new UI project and choosing visual direction
- Evaluating whether a project's current design tokens match its brand
- Comparing multiple design system options for a client presentation
- Migrating between design systems (need to know what you're moving toward)
- Building a custom design system by compositing traits from multiple references

Do NOT use when:
- The client has an existing brand guide with defined tokens (use those directly)
- You need to implement a specific design system (use the system's docs instead)
- You are doing a visual redesign within an existing system (audit the system instead)
- The project is a prototype where visual polish doesn't matter yet

## III. Workflow

### Step 1: Inventory Available DESIGN.md Files

Scan the design system reference library for available options:

```
Primary location: HTMLCraftStudio/templates/design-systems/
Structure: each system has a DESIGN.md with tokens, philosophy, and usage notes
```

**Currently imported (8 systems):**

| System | Personality | Primary Use Case |
|--------|-------------|-----------------|
| **Linear** | Precise, focused, developer-tool aesthetic | Task management, dev tools |
| **Stripe** | Authoritative, clean, trust-building | Fintech, payments, enterprise |
| **Vercel** | Minimal, performant, developer-first | Developer platforms, deployment |
| **Notion** | Warm, flexible, content-first | Productivity, wikis, collaboration |
| **Resend** | Modern, bold, API-developer aesthetic | Email infrastructure, dev APIs |
| **Supabase** | Approachable, open-source-friendly, green | Backend-as-a-service, databases |
| **Cal.com** | Open, scheduling-focused, clean | Booking, calendars, scheduling |
| **Mintlify** | Documentation-native, readable, structured | Developer docs, API references |

50+ additional systems available from VoltAgent/awesome-design-md for import.

### Step 2: Classify Project Requirements

Score the project across five design dimensions:

**A. Color Temperature**
- Cool (blues, grays, whites) -- authority, trust, technical
- Neutral (balanced palette) -- flexibility, professionalism
- Warm (oranges, greens, earth tones) -- approachability, community, growth

**B. Typography Style**
- Geometric sans (Inter, Geist) -- modern, precise, technical
- Humanist sans (Source Sans, Lato) -- friendly, readable, approachable
- Monospace accent (JetBrains Mono, Fira Code) -- developer-focused, code-native
- Serif accent (Playfair, Merriweather) -- editorial, authoritative, traditional

**C. Spatial Density**
- Compact (4-8px base unit) -- data-dense, dashboards, power users
- Comfortable (8-12px base unit) -- balanced, general-purpose
- Spacious (12-16px+ base unit) -- content-first, reading, storytelling

**D. Brand Personality**
- Technical -- "we are engineers building for engineers"
- Warm -- "we are humans building for humans"
- Playful -- "work should feel light and creative"
- Editorial -- "information presented with authority and clarity"
- Minimal -- "reduce everything to its essence"

**E. Interaction Philosophy**
- Responsive -- subtle hover states, smooth transitions, polish
- Functional -- clear affordances, obvious click targets, efficiency
- Delightful -- micro-animations, easter eggs, personality in motion

Document each dimension with a 1-sentence justification tied to the project's users.

### Step 3: Score Each DESIGN.md

For each available design system, score alignment on a 0-3 scale per dimension:

| Score | Meaning |
|-------|---------|
| 0 | Actively conflicts with requirement |
| 1 | Neutral -- doesn't help or hurt |
| 2 | Supports the requirement well |
| 3 | Perfect alignment with requirement |

Build a scoring matrix:

```
               Color  Type  Density  Personality  Interaction  TOTAL
Linear           2      3      2         3            2         12
Stripe           3      2      2         3            3         13
Vercel           2      3      1         3            2         11
Notion           1      2      3         2            3         11
...
```

Weight dimensions if the project has clear priorities (e.g., a data dashboard
weights density 2x, a marketing site weights personality 2x).

### Step 4: Present Top 3 Matches

For each of the top 3 scoring systems, provide:

1. **Match score** and ranking
2. **Why it fits** -- which dimensions align strongest (cite specific tokens)
3. **Where it diverges** -- which dimensions are weakest (and whether that matters)
4. **Adaptation notes** -- what you'd modify to improve fit (e.g., "swap primary blue for brand green, keep spacing and type scale")

Format as a comparison card the user can evaluate at a glance.

### Step 5: Apply Selected Design System

Once the user selects a system:

1. **Copy token file** from the DESIGN.md to the project's design directory
2. **Generate CSS custom properties** from the token definitions:
   ```css
   :root {
     --color-primary: [from tokens];
     --font-family-sans: [from tokens];
     --spacing-base: [from tokens];
     /* ... */
   }
   ```
3. **Create a project DESIGN.md** that references the source system and documents any adaptations
4. **Validate** the applied tokens render correctly in a sample component

## IV. Best Practices

1. **Score before you feel.** Gut reactions to design systems are valid but unreliable. Score first, then see if your intuition agrees. If they diverge, investigate why.
2. **Adaptation is expected.** No reference system will be a 100% match. The goal is 80%+ alignment with clear, documented modifications for the rest.
3. **Consider the user, not the builder.** A developer building a consumer app should pick for the consumer, not for their own aesthetic preferences.
4. **Check dark mode compatibility.** Some design systems have excellent dark mode support; others assume light-only. If dark mode is required, this is a hard filter.
5. **Typography is the hardest to change.** Colors are easy to swap. Spacing is mechanical. Typography choices cascade into line-height, measure, hierarchy, and readability. Pick a system whose type approach you can live with.

## V. Quality Checklist

- [ ] Available DESIGN.md files inventoried with current count
- [ ] Project requirements classified across all 5 dimensions with justifications
- [ ] Scoring matrix completed for all available systems
- [ ] Dimension weighting applied if project has clear priorities
- [ ] Top 3 matches presented with fit reasoning and divergence notes
- [ ] Selected system's tokens copied to project
- [ ] CSS custom properties generated from tokens
- [ ] Project DESIGN.md created with source reference and adaptations
- [ ] Sample component rendered to validate token application

## VI. Common Pitfalls

- **Popularity bias.** Picking Stripe's design system because Stripe is a good company, not because Stripe's visual language fits your product. Popularity is not fit.
- **Ignoring density requirements.** A spacious editorial system applied to a data dashboard wastes 40% of screen real estate. Density is a hard constraint, not a preference.
- **Skipping adaptation documentation.** If you modify 5 tokens from the source system but don't document which ones, the next developer will re-import the original and break your adaptations.
- **Single-dimension matching.** "They use Inter, we use Inter, so they're a match" -- typography is one of five dimensions. A single match doesn't make a system fit.
- **Forgetting interaction patterns.** Two systems can have identical colors and fonts but radically different interaction patterns (hover states, transitions, feedback). Interaction is a dimension, not a detail.

## VII. Example

**Project:** Internal analytics dashboard for a data engineering team.

**Requirements classification:**
- Color: Cool (trust, technical precision)
- Typography: Geometric sans with monospace accent (developer audience)
- Density: Compact (data-dense, many tables and charts)
- Personality: Technical ("we are engineers building for engineers")
- Interaction: Functional (efficiency over delight)

**Scoring results:**
- Linear: 14/15 (precise, compact, developer-native, geometric type)
- Stripe: 12/15 (strong authority, slightly too spacious for dashboards)
- Vercel: 11/15 (good type, but interaction patterns lean marketing-site)

**Selection:** Linear design system with one adaptation -- swap primary purple for
the team's brand blue. All spacing, typography, and interaction tokens used as-is.

**Applied:** Tokens copied, CSS custom properties generated, project DESIGN.md created
documenting the color swap and referencing Linear DESIGN.md as source.

## VIII. Related Skills

- `voice-before-structure` -- Read design philosophy before writing structural artifacts
- `theme-factory` -- Applies selected design tokens to artifacts (slides, reports, HTML)
- `web-artifacts-builder` -- Builds HTML components using applied design system
- `canvas-design` -- Visual art creation using design philosophy principles
- `brand-guidelines` -- Applies organization-specific brand to artifacts

## Output

- Scoring matrix table comparing all available systems across five dimensions
- Top 3 match cards with: score, fit rationale, divergence notes, adaptation notes
- Project DESIGN.md file created in the project directory referencing the selected source system and documenting any token adaptations
- CSS custom properties block generated from selected system's tokens

## Examples

**Scenario 1:** "Which design system fits my internal analytics dashboard for a data engineering team?" -> Scoring matrix run against 8 systems; Linear scores 14/15 on technical/compact/precise axes; selection delivered with one adaptation noted (swap primary purple for brand blue); project DESIGN.md created

**Scenario 2:** "I need to pick a design reference for a community-focused open source project" -> Scoring identifies Supabase (approachable, open-source-friendly) as best fit over Stripe (too authoritative); adaptation notes document switching monospace accent to humanist sans for community warmth

## Edge Cases

- **Client already has a brand guide with defined tokens:** Do not run the scoring workflow; map their existing tokens to the closest DESIGN.md system for reference, but use the brand guide as the authoritative token source
- **No available system scores above 9/15:** Report the top match with a list of required adaptations; flag that a custom system may be warranted if adaptations exceed 3 dimensions
- **Dark mode is a hard requirement:** Apply dark mode support as a pre-filter before scoring; eliminate any system without documented dark mode tokens

## Anti-Patterns

- Selecting a system based on the builder's aesthetic preference rather than running the five-dimension scoring matrix
- Applying a system without documenting token adaptations — undocumented changes are invisible to the next developer and will be overwritten on re-import
- Using a single-dimension match ("they both use Inter") as sufficient justification — typography is one of five dimensions
