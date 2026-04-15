---
name: figma-to-code
model: sonnet
description: "Extracts design tokens, component structure, and layout from a Figma file and produces framework-specific code (HTML, Svelte, React, or Vue) that preserves design intent rather than pixel-copying. Use when: 'implement this Figma design', 'convert this Figma file to code', 'extract tokens from Figma', 'build this component from the design'."
license: Adapted from nafiurrahmanniloy/figma-skill (MIT)
category: continuous-learning
triggers:
  - "implement this Figma design"
  - "convert this Figma file to code"
  - "extract tokens from Figma"
  - "build this component from the design"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: figma_url
    type: string
    description: URL or file reference to the Figma design file
    required: true
  - name: target_framework
    type: string
    description: Target framework — html, svelte, react, or vue
    required: false
outputs:
  - name: component_code
    type: string
    description: Framework-specific component code preserving design tokens, structure, and layout from the Figma file
---
# Figma to Code

---

## I. Philosophy: Design Intent Over Pixel Copying

The goal is not to reproduce every pixel from Figma — it is to capture the *design intent* (spacing rhythm, color relationships, typographic hierarchy) and express it in idiomatic code for the target framework. A good conversion reads like a developer wrote it with the design open, not like a screenshot was OCR'd into divs.

---

## II. When to Use This Skill

Use when:
- User shares a Figma URL and wants code generated
- Building HTML kits that should match an existing Figma design
- Extracting design tokens (colors, spacing, typography) from a Figma file
- Need to verify implementation fidelity against a Figma source

Do NOT use when:
- User wants to CREATE a Figma design from code (use `use_figma` MCP tool directly)
- Design is a screenshot, not a Figma file (use manual CSS instead)

---

## III. Workflow

### Step 1: Resolve Figma Context

Extract `fileKey` and `nodeId` from the Figma URL.
Call `get_design_context` MCP tool with these parameters.
This returns: reference code, screenshot, component metadata, and Code Connect mappings.

### Step 2: Extract Design Tokens

From the design context, extract:
- **Colors**: hex values, CSS custom properties, semantic names
- **Typography**: font families, sizes, weights, line heights
- **Spacing**: padding, margin, gap values
- **Border radius**: corner radius values
- **Shadows**: box-shadow definitions
- **Layout**: flex/grid structure, auto-layout direction and gap

If DESIGN.md files are available in `HTMLCraftStudio/templates/design-systems/`, cross-reference tokens against the closest matching design system.

### Step 3: Generate Framework Code

Based on user's target framework:

| Framework | Output |
|-----------|--------|
| HTML/CSS | Semantic HTML + CSS custom properties + utility classes |
| Svelte | `.svelte` component with scoped styles |
| React | TSX component with Tailwind or CSS modules |
| Vue | SFC with `<script setup>` and scoped styles |

Rules:
- Use semantic HTML elements (`nav`, `section`, `article`, `button`) not generic divs
- Extract repeated patterns as sub-components
- Use CSS custom properties for design tokens, not hardcoded values
- Preserve auto-layout as flexbox/grid, not absolute positioning

### Step 4: Verify Fidelity

Compare generated code against the Figma screenshot:
- Layout structure matches
- Typography hierarchy preserved
- Color relationships correct
- Spacing rhythm consistent
- Interactive states accounted for (hover, focus, active)

---

## IV. Best Practices

1. **Tokens first, components second.** Extract the design token set before generating any component code.
2. **Respect the component boundary.** If Figma shows a component with variants, generate one component with props — not multiple separate components.
3. **Use existing design system tokens when available.** Check `HTMLCraftStudio/templates/design-systems/` for matching DESIGN.md files before inventing new token names.
4. **Preserve accessibility.** Add `aria-label`, `role`, and proper heading hierarchy even if not explicit in the Figma design.
5. **Generate mobile-first.** Start with the smallest breakpoint and add `@media` queries for larger sizes.

---

## V. Quality Checklist

- [ ] Design tokens extracted and documented as CSS custom properties
- [ ] Semantic HTML used throughout (no div soup)
- [ ] Layout uses flexbox/grid matching Figma auto-layout
- [ ] Typography hierarchy preserved (h1 > h2 > h3 > p)
- [ ] Colors reference token variables, not hardcoded hex
- [ ] Component is responsive (mobile-first)
- [ ] Accessibility attributes present

---

## VI. Related Skills

- `canvas-design` — Visual artifact generation
- `theme-factory` — Styling toolkit for HTML pages
- `artifacts-builder` — HTML artifact scaffolding

## Output

- CSS custom properties block documenting all extracted design tokens (colors, typography, spacing, radius, shadows)
- Framework-specific component file: `.html` + `.css`, `.svelte`, `.tsx` (React), or `.vue` based on user's target
- Fidelity verification notes comparing generated output against the Figma screenshot

## Examples

**Scenario 1:** "Implement this Figma card component in HTML" -> Token set extracted (primary blue #1a73e8, Inter 16/24, 8px radius, 16px padding), semantic HTML `<article>` with CSS custom properties generated, auto-layout mapped to flexbox, accessibility attributes (`aria-label`, `role`) added

**Scenario 2:** "Build this navigation sidebar from the Figma design in Svelte" -> Svelte component with scoped styles generated, Figma auto-layout (vertical stack, 8px gap) mapped to `display: flex; flex-direction: column; gap: 8px`, active state variant captured as CSS class, mobile-first responsive breakpoint added

## Edge Cases

- **Figma file has no design tokens defined (all values are hardcoded in nodes):** Extract values directly from node properties and name them semantically based on their usage (e.g., `--color-nav-active` not `--color-1a73e8`); note in output that token naming is inferred
- **Design contains overlapping absolute-positioned layers (not auto-layout):** Preserve relative positioning using CSS `position: relative/absolute`; flag in output that this area did not use auto-layout and may need manual responsive handling
- **User provides a Figma URL but the MCP tool returns an error:** Ask user to verify the file is shared with link access enabled; do not guess at the design from the URL alone

## Anti-Patterns

- Generating `div` soup instead of semantic HTML — use `nav`, `section`, `article`, `button`, `header` based on the element's role in the design
- Hardcoding hex values directly in component styles instead of extracting them as CSS custom properties — hardcoded values cannot be themed or updated centrally
- Treating the Figma screenshot as the acceptance gate without also verifying that the code works at mobile breakpoints
